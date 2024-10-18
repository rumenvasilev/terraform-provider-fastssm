package provider

import (
	"context"
	"errors"
	"fmt"
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
var _ resource.Resource = &ParameterResource{}
var _ resource.ResourceWithImportState = &ParameterResource{}

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
	// TagsAll        types.Map    `tfsdk:"tags_all"`
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
		MarkdownDescription: "SSM Parameter resource",

		Attributes: map[string]schema.Attribute{
			"allowed_pattern": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(0, 1024)},
			},
			names.AttrARN: schema.StringAttribute{
				Optional: true,
				Computed: true,
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
			},
			names.AttrDescription: schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(0, 1024)},
			},
			"insecure_value": schema.StringAttribute{
				Optional: true,
				// Computed: true,
				// Sensitive: true,
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
				Validators: []validator.String{stringvalidator.LengthBetween(1, 2048)},
			},
			"overwrite": schema.BoolAttribute{
				Optional:           true,
				DeprecationMessage: "this attribute has been deprecated",
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
						// dependentParameterValidator{dependentParamName: "type", requiredValue: []string{"SecureString"}},
					)},
			},
			names.AttrVersion: schema.Int64Attribute{
				Computed: true,
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
	if (!data.InsecureValue.IsUnknown() && !data.InsecureValue.IsNull()) && data.Type.ValueString() != string(ssm_types.ParameterTypeSecureString) {
		val = data.InsecureValue.ValueString()
	}

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
			if isRetryableError(erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %v, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %v", erri))
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
			if isRetryableError(erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %v, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %v", erri))
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
	// Only call DescribeParameters if nothing but version has changed!
	if res.Version != data.Version.ValueInt64() {
		// if data.Arn.ValueString() == *res.ARN &&
		if data.Name.ValueString() == *res.Name &&
			data.Type.ValueString() == string(res.Type) &&
			data.DataType.ValueString() == *res.DataType &&
			data.Value.ValueString() == *res.Value {

			resp.Diagnostics.AddWarning("Running DescribeParameter call", "We will now do a describe call because we don't know what changed. This is an expensive operation!")
			name := "Name"
			equals := "Equals"
			oper := &ssm.DescribeParametersInput{ParameterFilters: []ssm_types.ParameterStringFilter{
				{
					Key:    &name,
					Option: &equals,
					Values: []string{data.Name.ValueString()},
				},
			}}
			// TODO NOT RETRIED!!!
			descParams, err := r.client.DescribeParameters(ctx, oper)
			if err != nil {
				resp.Diagnostics.AddError("Something went wrong while getting parameter metadata", err.Error())
				return
			}
			if len(descParams.Parameters) == 0 || len(descParams.Parameters) > 1 {
				resp.Diagnostics.AddError("Incorrect response for parameter metadata", "None or too many results found.")
				return
			}
			data.Description = basetypes.NewStringValue(*descParams.Parameters[0].Description)

			// Metadata contains these extra fields, but we must only use Description:
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

	if !data.InsecureValue.IsNull() && res.Type != ssm_types.ParameterTypeSecureString {
		data.InsecureValue = basetypes.NewStringValue(*res.Value)
	} else {
		data.Value = basetypes.NewStringValue(*res.Value)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ParameterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Prepare PutParameter request
	typ := ssm_types.ParameterType(data.Type.ValueString())
	val := data.Value.ValueString()
	if (!data.InsecureValue.IsUnknown() && !data.InsecureValue.IsNull()) && data.Type.ValueString() != string(ssm_types.ParameterTypeSecureString) {
		val = data.InsecureValue.ValueString()
	}

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
			if isRetryableError(erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %v, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %v", erri))
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
			if isRetryableError(erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %v, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %v", erri))
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
			if isRetryableError(erri) {
				// Return with retryable error, specifying how long to wait before the next retry
				return retry.RetryableError(fmt.Errorf("temporary failure: %v, retrying...", erri))
			}

			// If it's a permanent error, stop retrying
			return retry.NonRetryableError(fmt.Errorf("permanent failure: %v", erri))
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

func isRetryableError(err error) bool {
	if err == nil {
		return false // If err is nil, it's not a retryable error
	}
	// Type assertion for Smithy (used by AWS SDK v2)
	var apiErr smithy.APIError
	if ok := errors.As(err, &apiErr); ok {
		tflog.Info(context.TODO(), apiErr.ErrorCode())
		tflog.Info(context.TODO(), apiErr.ErrorMessage())
		tflog.Info(context.TODO(), apiErr.ErrorFault().String())

		if apiErr.ErrorCode() == "ThrottlingException" {
			tflog.Info(context.TODO(), "Rate limit exceeded, retrying...")
			// Implement backoff before retrying
			time.Sleep(time.Duration(5) * time.Second)
			return true // Retry on throttling error
		}
	}

	var ratelimited ratelimit.QuotaExceededError
	if ok := errors.As(err, &ratelimited); ok {
		tflog.Error(context.TODO(), "we are being rate limited dude")
		tflog.Info(context.TODO(), "Rate limit exceeded, retrying...")
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
