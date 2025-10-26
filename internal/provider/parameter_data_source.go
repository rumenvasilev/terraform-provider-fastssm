package provider

import (
	"context"
	"fmt"
	"terraform-provider-fastssm/internal/names"
	"terraform-provider-fastssm/internal/tfresource"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssm_types "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &ParameterDataSource{}

func NewParameterDataSource() datasource.DataSource {
	return &ParameterDataSource{}
}

// ParameterDataSource defines the data source implementation.
type ParameterDataSource struct {
	client *ssm.Client
}

// ParameterDataSourceModel describes the data source data model.
type ParameterDataSourceModel struct {
	Arn            types.String `tfsdk:"arn"`
	InsecureValue  types.String `tfsdk:"insecure_value"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Value          types.String `tfsdk:"value"`
	Version        types.Int64  `tfsdk:"version"`
	WithDecryption types.Bool   `tfsdk:"with_decryption"`
}

func (d *ParameterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parameter"
}

func (d *ParameterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "SSM Parameter data source",

		Attributes: map[string]schema.Attribute{
			names.AttrARN: schema.StringAttribute{
				// Optional: true,
				Computed:    true,
				Description: "ARN of the parameter.",
			},
			"insecure_value": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.ConflictsWith(path.Expressions{
							path.MatchRoot(names.AttrValue),
						}...),
						dependentParameterValidator{dependentParamName: "type", requiredValue: []string{"String", "StringList"}},
					)},
				// PlanModifiers: []planmodifier.String{
				// 	SyncAttributePlanModifier("value"),
				// },
				Description: "Value of the parameter. **Use caution:** This value is never marked as sensitive.",
			},
			names.AttrName: schema.StringAttribute{
				Required: true,
				// PlanModifiers: []planmodifier.String{
				// 	stringplanmodifier.RequiresReplace(),
				// },
				Description: "Name of the parameter.",
			},
			names.AttrType: schema.StringAttribute{
				// Required: true,
				Computed: true,
				// Validators: []validator.String{
				// 	stringvalidator.OneOf("String", "StringList", "SecureString"),
				// }, // awstypes.ParameterType.Values()
				Description: "Type of the parameter. Valid types are `String`, `StringList` and `SecureString`.",
			},
			names.AttrValue: schema.StringAttribute{
				Sensitive: true,
				Computed:  true,
				// Computed:  true,
				// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator#ExactlyOneOf
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.ConflictsWith(path.Expressions{
							path.MatchRoot("insecure_value"),
						}...),
						// dependentParameterValidator{dependentParamName: "type", requiredValue: []string{"SecureString"}},
					)},
				Description: "Value of the parameter. This value is always marked as sensitive in the Terraform plan output, regardless of `type`. In Terraform CLI version 0.15 and later, this may require additional configuration handling for certain scenarios. For more information, see the [Terraform v0.15 Upgrade Guide](https://www.terraform.io/upgrade-guides/0-15.html#sensitive-output-values).",
			},
			names.AttrVersion: schema.Int64Attribute{
				Computed:    true,
				Description: "Version of the parameter.",
			},
			"with_decryption": schema.BoolAttribute{
				Optional: true,
				// TODO need to set this to default = true
				// Default:  true,
				Description: "Whether to return decrypted `SecureString` value. Defaults to `true`.",
			},
		},
	}
}

func (d *ParameterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ssm.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ssm.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ParameterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ParameterDataSourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	const (
		// Maximum amount of time to wait for asynchronous validation on SSM Parameter creation.
		timeout = 2 * time.Minute
	)

	decryption := true
	if !data.WithDecryption.IsNull() {
		decryption = data.WithDecryption.ValueBool()
	}

	var res = &ssm_types.Parameter{}
	var erri error
	// Define retry logic
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		res, erri = findParameterByName(ctx, d.client, data.Name.ValueString(), decryption)
		if erri != nil {
			// Check if the error is retryable (e.g., rate limiting, network issues)
			if isRetryableError(ctx, erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %w", erri))
		}

		// If success, return nil (no retry)
		return nil
	})

	if tfresource.NotFound(err) {
		resp.Diagnostics.AddError("parameter not found", fmt.Sprintf("SSM Parameter %s not found, removing from state", data.Name.String()))
		data.Name = basetypes.NewStringNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read parameter, got error: %v", err))
		return
	}

	data.Arn = basetypes.NewStringValue(*res.ARN)
	data.Name = basetypes.NewStringValue(*res.Name)
	data.Type = basetypes.NewStringValue(string(res.Type))
	data.Version = basetypes.NewInt64Value(res.Version)

	data.Value = basetypes.NewStringValue(*res.Value)
	if !data.InsecureValue.IsNull() && res.Type != ssm_types.ParameterTypeSecureString {
		data.InsecureValue = basetypes.NewStringValue(*res.Value)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
