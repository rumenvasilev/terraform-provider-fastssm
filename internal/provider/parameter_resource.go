package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"terraform-provider-fastssm/internal/names"
	"terraform-provider-fastssm/internal/tfresource"

	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssm_types "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// const (
// 	errCodeValidationException = "ValidationException"
// )

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.ResourceWithConfigure = &ParameterResource{}
var _ resource.ResourceWithImportState = &ParameterResource{}
var _ resource.ResourceWithMoveState = &ParameterResource{}

func NewParameterResource() resource.Resource {
	return &ParameterResource{}
}

// ParameterResource defines the resource implementation.
type ParameterResource struct {
	client *ssm.Client
}

// ParameterResourceModel describes the resource data model.
type ParameterResourceModel struct {
	AllowedPattern types.String `tfsdk:"allowed_pattern"`
	Arn            types.String `tfsdk:"arn"`
	DataType       types.String `tfsdk:"data_type"`
	Description    types.String `tfsdk:"description"`
	InsecureValue  types.String `tfsdk:"insecure_value"`
	// KeyId     types.String `tfsdk:"key_id"`
	Name      types.String `tfsdk:"name"`
	Overwrite types.Bool   `tfsdk:"overwrite"`
	Tags      types.Map    `tfsdk:"tags"`
	// TagsAll   types.Map    `tfsdk:"tags_all"`
	// Tier    types.String `tfsdk:"tier"`
	Type    types.String `tfsdk:"type"`
	Value   types.String `tfsdk:"value"`
	Version types.Int64  `tfsdk:"version"`
}

func (r *ParameterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parameter"
}

func (r *ParameterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Provides an SSM Parameter resource.",
		MarkdownDescription: "~> **Note:** this is a slimmer and faster version, but doesn't include all the metadata you would normally enjoy with the official terraform-provider-aws. If performance is an issue, this should be a drop-in replacement for ssm_parameter resource from the official aws provider.",

		Attributes: map[string]schema.Attribute{
			"allowed_pattern": schema.StringAttribute{
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(0, 1024)},
				Description: "Regular expression used to validate the parameter value.",
			},
			names.AttrARN: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "ARN of the parameter.",
			},
			"data_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("text"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						"aws:ec2:image",
						"aws:ssm:integration",
						"text",
					}...)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Data type of the parameter. Valid values: `text`, `aws:ssm:integration` and `aws:ec2:image` for AMI format, see the [Native parameter support for Amazon Machine Image IDs](https://docs.aws.amazon.com/systems-manager/latest/userguide/parameter-store-ec2-aliases.html)",
			},
			names.AttrDescription: schema.StringAttribute{
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(0, 1024)},
				Description: "Description of the parameter.",
			},
			"insecure_value": schema.StringAttribute{
				Optional: true,
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
				Description: "Value of the parameter. **Use caution:** This value is _never_ marked as sensitive in the Terraform plan output. This argument is not valid with a `type` of `SecureString`.",
			},
			// KMS KeyID - we use the default.
			// names.AttrKeyID: schema.StringAttribute{
			// 	Optional: true,
			// 	Computed: true,
			// },
			names.AttrName: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 2048)},
				Description: "Name of the parameter. If the name contains a path (e.g., any forward slashes (`/`)), it must be fully qualified with a leading forward slash (`/`). For additional requirements and constraints, see the [AWS SSM User Guide](https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-parameter-name-constraints.html).",
			},
			"overwrite": schema.BoolAttribute{
				Optional:           true,
				DeprecationMessage: "this attribute has been deprecated",
				Description:        "Overwrite an existing parameter. If not specified, defaults to `false` if the resource has not been created by Terraform to avoid overwrite of existing resource, and will default to `true` otherwise (Terraform lifecycle rules should then be used to manage the update behavior).",
			},
			names.AttrTags: schema.MapAttribute{
				Optional:           true,
				ElementType:        types.StringType,
				Description:        "UNSUPPORTED. This feature is intentionally unavailable for performance reasons. You can still pass input data to it for backwards compatibility, but it will not be reflected in the ssm_parameter resource in AWS.",
				DeprecationMessage: "UNSUPPORTED. This feature is intentionally unavailable for performance reasons. You can still pass input data to it for backwards compatibility, but it will not be reflected in the ssm_parameter resource in AWS.",
			},
			// names.AttrTagsAll: schema.MapAttribute{
			// 	Optional:    true,
			// 	Computed:    true,
			// 	ElementType: types.StringType,
			// },
			// "tier" is auto-upgraded by Amazon from standard to advanced if needed.
			// We don't use that in our SSM configurations.
			names.AttrType: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("String", "StringList", "SecureString"),
				}, // awstypes.ParameterType.Values()
				Description: "Type of the parameter. Valid types are `String`, `StringList` and `SecureString`.",
			},
			names.AttrValue: schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				// Computed:  true,
				// https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator#ExactlyOneOf
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.ConflictsWith(path.Expressions{
							path.MatchRoot("insecure_value"),
						}...),
						stringvalidator.AtLeastOneOf(path.Expressions{
							path.MatchRoot("insecure_value"),
							path.MatchRoot("value"),
						}...),
						// dependentParameterValidator{dependentParamName: "type", requiredValue: []string{"SecureString"}},
					)},
				Description: "Value of the parameter. This value is always marked as sensitive in the Terraform plan output, regardless of `type`. In Terraform CLI version 0.15 and later, this may require additional configuration handling for certain scenarios. For more information, see the [Terraform v0.15 Upgrade Guide](https://www.terraform.io/upgrade-guides/0-15.html#sensitive-output-values).",
			},
			names.AttrVersion: schema.Int64Attribute{
				Computed:    true,
				Description: "Version of the parameter.",
			},
		},
	}
}

func (r *ParameterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ssm.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ssm.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ParameterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ParameterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare PutParameter request
	typ := ssm_types.ParameterType(data.Type.ValueString())
	val := data.Value.ValueString()

	input := &ssm.PutParameterInput{
		Name:           data.Name.ValueStringPointer(),
		Value:          &val,
		AllowedPattern: data.AllowedPattern.ValueStringPointer(),
		Type:           typ,
		DataType:       data.DataType.ValueStringPointer(),
	}

	if !data.DataType.IsNull() {
		input.DataType = data.DataType.ValueStringPointer()
	}

	if !data.Description.IsNull() {
		input.Description = data.Description.ValueStringPointer()
	}

	// KeyID is unsupported

	// No Tags support

	// Send create parameter request
	// var err error
	var result = &ssm.PutParameterOutput{}
	var erri error
	// Define retry logic
	err := retry.RetryContext(ctx, 10*time.Minute, func() *retry.RetryError {
		result, erri = r.client.PutParameter(ctx, input)
		if erri != nil {
			// Check if the error is retryable (e.g., rate limiting, network issues)
			if isRetryableError(ctx, erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %w", erri))
		}

		// If success, return nil (no retry)
		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("SSM parameter create error", fmt.Sprintf("creating SSM Parameter (%s): %s", data.Name.String(), err))
		return
	}

	data.Version = basetypes.NewInt64Value(result.Version)

	// All values must be known after apply
	withDecryption := true
	get, err := r.client.GetParameter(ctx, &ssm.GetParameterInput{Name: data.Name.ValueStringPointer(), WithDecryption: &withDecryption})
	if err != nil {
		resp.Diagnostics.AddError("parameter get failed", "Couldn't get the SSM parameter data after creation")
		return
	}
	data.Arn = basetypes.NewStringValue(*get.Parameter.ARN)

	data.InsecureValue = basetypes.NewStringNull()
	// Populate insecure_value if it's not a secure string
	if get.Parameter.Type != ssm_types.ParameterTypeSecureString {
		data.InsecureValue = data.Value
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ParameterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	const (
		// Maximum amount of time to wait for asynchronous validation on SSM Parameter creation.
		timeout = 2 * time.Minute
	)

	var res = &ssm_types.Parameter{}
	var erri error
	// Define retry logic
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		res, erri = findParameterByName(ctx, r.client, data.Name.ValueString(), true)
		if erri != nil {
			// Check if the error is retryable (e.g., rate limiting, network issues)
			if isRetryableError(ctx, erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying...", erri))
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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
		return
	}

	// The following information is only available with DescribeParameter call, to get the additional metadata.
	// Only call DescribeParameters if nothing but the version has changed!
	if res.Version != data.Version.ValueInt64() {
		if data.Name.ValueString() == *res.Name &&
			data.Type.ValueString() == string(res.Type) &&
			data.DataType.ValueString() == *res.DataType &&
			data.Value.ValueString() == *res.Value {

			resp.Diagnostics.AddWarning("Running DescribeParameter call", "We will now do a describe call because we don't know what changed (most likely metadata). This is an expensive operation!")
			name := "Name"
			equals := "Equals"
			oper := &ssm.DescribeParametersInput{ParameterFilters: []ssm_types.ParameterStringFilter{
				{
					Key:    &name,
					Option: &equals,
					Values: []string{data.Name.ValueString()},
				},
			}}

			var md = &ssm.DescribeParametersOutput{}
			err := retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
				md, erri = r.client.DescribeParameters(ctx, oper)
				if erri != nil {
					// Check if the error is retryable (e.g., rate limiting, network issues)
					if isRetryableError(ctx, erri) {
						// Return with retryable error, specifying how long to wait before the next retry
						return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying...", erri))
					}

					// If it's a permanent error, stop retrying
					return retry.NonRetryableError(fmt.Errorf("permanent failure: %w", erri))
				}

				// If success, return nil (no retry)
				return nil
			})

			if err != nil {
				resp.Diagnostics.AddError("Something went wrong while getting parameter metadata", err.Error())
				return
			}
			if len(md.Parameters) == 0 || len(md.Parameters) > 1 {
				resp.Diagnostics.AddError("Incorrect response for parameter metadata", "None or too many results found.")
				return
			}

			data.Description = basetypes.NewStringValue("") // default to empty
			// If description is empty we don't want to panic
			if md.Parameters[0].Description != nil {
				data.Description = basetypes.NewStringValue(*md.Parameters[0].Description)
			}

			// Metadata contains these extra fields, but we only use & need Description:
			//
			// AllowedPattern
			// Description
			// KeyId
			// LastModifiedUser
			// Policies
			// Tier
		}
	}

	data.Arn = basetypes.NewStringValue(*res.ARN)
	data.Name = basetypes.NewStringValue(*res.Name)
	data.Type = basetypes.NewStringValue(string(res.Type))
	data.Version = basetypes.NewInt64Value(res.Version)
	data.DataType = basetypes.NewStringValue(*res.DataType)

	data.Value = basetypes.NewStringValue(*res.Value)
	// In case `value` is not provided, but `insecure_value`, copy it
	if data.Value.IsNull() || data.Value.IsUnknown() {
		data.Value = data.InsecureValue
	}

	// Populate insecure_value if it's not a secure string
	if res.Type != ssm_types.ParameterTypeSecureString {
		data.InsecureValue = basetypes.NewStringValue(*res.Value)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ParameterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// copy value to insecure_value if it's not a secure string
	data.InsecureValue = basetypes.NewStringNull()
	if data.Type.ValueString() != "SecureString" {
		data.InsecureValue = data.Value
	}

	// Prepare PutParameter request
	typ := ssm_types.ParameterType(data.Type.ValueString())
	val := data.Value.ValueString()
	// Update should always overwrite
	overwrite := true

	input := &ssm.PutParameterInput{
		Name:           data.Name.ValueStringPointer(),
		Value:          &val,
		AllowedPattern: data.AllowedPattern.ValueStringPointer(),
		Type:           typ,
		DataType:       data.DataType.ValueStringPointer(),
		Description:    data.Description.ValueStringPointer(),
		Overwrite:      &overwrite,
	}

	// No Tags support

	// Send create parameter request
	var result = &ssm.PutParameterOutput{}
	var erri error
	// Define retry logic
	err := retry.RetryContext(ctx, 10*time.Minute, func() *retry.RetryError {
		result, erri = r.client.PutParameter(ctx, input)
		if erri != nil {
			// Check if the error is retryable (e.g., rate limiting, network issues)
			if isRetryableError(ctx, erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %w", erri))
		}

		// If success, return nil (no retry)
		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("SSM parameter update error", fmt.Sprintf("updating SSM Parameter (%s): %s", data.Name.String(), err))
		return
	}

	data.Version = basetypes.NewInt64Value(result.Version)

	// All values must be known after apply!
	// We need to read once again before the end, to get the ARN,
	// because it's not included in the response of the PutParameter call.
	withDecryption := true
	var res = &ssm_types.Parameter{}
	// Define retry logic
	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		res, erri = findParameterByName(ctx, r.client, data.Name.ValueString(), withDecryption)
		if erri != nil {
			// Check if the error is retryable (e.g., rate limiting, network issues)
			if isRetryableError(ctx, erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %w", erri))
		}

		// If success, return nil (no retry)
		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("parameter get failed", "Couldn't get the SSM parameter data after creation")
		return
	}
	data.Arn = basetypes.NewStringValue(*res.ARN)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "updated a resource")

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ParameterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	input := &ssm.DeleteParameterInput{
		Name: data.Name.ValueStringPointer(),
	}

	var erri error
	err := retry.RetryContext(ctx, 10*time.Minute, func() *retry.RetryError {
		_, erri = r.client.DeleteParameter(ctx, input)
		if erri != nil {
			// Check if the error is retryable (e.g., rate limiting, network issues)
			if isRetryableError(ctx, erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %w, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %w", erri))
		}

		// If success, return nil (no retry)
		return nil
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ssm parameter, got error: %s", err))
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ParameterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// This currently only supports migrating from aws_ssm_parameter to fastssm_parameter
//
//	moved {
//	  from = aws_ssm_parameter.test
//	  to   = fastssm_parameter.test
//	}
//
// You cannot revert back, because that support needs to be present in aws_ssm_parameter
func (r *ParameterResource) MoveState(ctx context.Context) []resource.StateMover {
	sourceSchema := awsSSMParameterResourceSchema()
	return []resource.StateMover{
		{
			SourceSchema: &sourceSchema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				// Always verify the expected source before working with the data.
				if req.SourceTypeName != "aws_ssm_parameter" {
					resp.Diagnostics.AddError(
						"Source schema name type mismatch",
						fmt.Sprintf("Expected source schema to be aws_ssm_parameter, but was %q", req.SourceTypeName),
					)
					return
				}

				if req.SourceSchemaVersion != 0 {
					resp.Diagnostics.AddError(
						"Source schema version mismatch",
						fmt.Sprintf("Expected source schema version to be 0, but was %d", req.SourceSchemaVersion),
					)
					return
				}

				// This only checks the provider address namespace and type
				// since practitioners may use differing hostnames for the same
				// provider, such as a network mirror. If necessary though, the
				// hostname can be used for disambiguation.
				if !strings.HasSuffix(req.SourceProviderAddress, "hashicorp/aws") {
					resp.Diagnostics.AddError(
						"Source provider unsupported",
						fmt.Sprintf("Expected source provider was hashicorp/aws, but we got %q", req.SourceProviderAddress),
					)
					return
				}

				var sourceStateData awsSSMParameterResourceModel

				resp.Diagnostics.Append(req.SourceState.Get(ctx, &sourceStateData)...)

				if resp.Diagnostics.HasError() {
					return
				}

				targetStateData := ParameterResourceModel{
					AllowedPattern: sourceStateData.AllowedPattern,
					Arn:            sourceStateData.Arn,
					DataType:       sourceStateData.DataType,
					Description:    sourceStateData.Description,
					Value:          sourceStateData.Value,
					// InsecureValue:  sourceStateData.InsecureValue,
					Name:      sourceStateData.Name,
					Overwrite: sourceStateData.Overwrite,
					Tags:      sourceStateData.Tags,
					Type:      sourceStateData.Type,
					Version:   sourceStateData.Version,
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, targetStateData)...)
			},
		},
	}
}

func isRetryableError(ctx context.Context, err error) bool {
	if err == nil {
		return false // If err is nil, it's not a retryable error
	}
	// Type assertion for Smithy (used by AWS SDK v2)
	var apiErr smithy.APIError
	if ok := errors.As(err, &apiErr); ok {
		tflog.Info(ctx, apiErr.ErrorCode())
		tflog.Info(ctx, apiErr.ErrorMessage())
		tflog.Info(ctx, apiErr.ErrorFault().String())

		if apiErr.ErrorCode() == "ThrottlingException" {
			tflog.Info(ctx, "Rate limit exceeded, retrying...")
			// Implement backoff before retrying
			time.Sleep(time.Duration(5) * time.Second)
			return true // Retry on throttling error
		}
	}

	var ratelimited ratelimit.QuotaExceededError
	if ok := errors.As(err, &ratelimited); ok {
		tflog.Error(ctx, "we are being rate limited dude")
		tflog.Info(ctx, "Rate limit exceeded, retrying...")
		// Implement backoff before retrying
		time.Sleep(time.Duration(5) * time.Second)
		return true // Retry on throttling error
	}
	return false
}

func findParameterByName(ctx context.Context, conn *ssm.Client, name string, withDecryption bool) (*ssm_types.Parameter, error) {
	input := &ssm.GetParameterInput{
		Name:           &name,
		WithDecryption: &withDecryption,
	}

	output, err := conn.GetParameter(ctx, input)

	var notfound = new(ssm_types.ParameterNotFound)
	if errors.As(err, &notfound) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || output.Parameter == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output.Parameter, nil
}
