package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
)

var accountIDRegexp = regexache.MustCompile(`^(aws|aws-managed|third-party|\d{12}|cw.{10})$`)
var partitionRegexp = regexache.MustCompile(`^aws(-[a-z]+)*$`)
var regionRegexp = regexache.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d$`)

// validates all listed in https://gist.github.com/shortjared/4c1e3fe52bdfa47522cfe5b41e5d6f22
var servicePrincipalRegexp = regexache.MustCompile(`^([0-9a-z-]+\.){1,4}(amazonaws|amazon)\.com$`)

type durationValidator struct{}

func (v durationValidator) Description(ctx context.Context) string {
	return "Validates that the duration is between 15 minutes and 12 hours with valid time units (ns, us, µs, ms, s, m, h)."
}

func (v durationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v durationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		// If the value is null, no need to validate (optional field)
		return
	}

	// Convert to string
	val := req.ConfigValue.ValueString()

	duration, err := time.ParseDuration(val)
	if err != nil {
		resp.Diagnostics.AddError(
			"error parsing duration",
			fmt.Sprintf("%q cannot be parsed as a duration: %v", val, err),
		)
		return
	}

	if duration.Minutes() < 15 || duration.Hours() > 12 {
		resp.Diagnostics.AddError(
			"invalid duration",
			fmt.Sprintf("duration %q must be between 15 minutes (15m) and 12 hours (12h), inclusive", val),
		)
	}
}

type jsonValidator struct{}

func (v jsonValidator) Description(ctx context.Context) string {
	return "Validates that the supplied string is a valid JSON"
}

func (v jsonValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		// If the value is null, no need to validate (optional field)
		return
	}

	// Convert to string
	val := req.ConfigValue.ValueString()
	if val == "" {
		resp.Diagnostics.AddError(
			"input is empty",
			"expected type of input to be string",
		)
		return
	}

	if _, err := structure.NormalizeJsonString(v); err != nil {
		resp.Diagnostics.AddError(
			"invalid json input",
			fmt.Sprintf("%q contains an invalid JSON: %s", val, err),
		)
	}
}

type arnValidator struct {
	kind string
}

func (v arnValidator) Description(ctx context.Context) string {
	return "Validates ARN"
}

func (v arnValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateSet validates that a string value matches an ARN format with additional validation on the parsed ARN value
// It must:
// * Be parseable as an ARN
// * Have a valid partition
// * Have a valid region
// * Have either an empty or valid account ID
// * Have a non-empty resource part
// * Pass the supplied checks
func (v arnValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() {
		// If the value is null, no need to validate (optional field)
		return
	}

	// Convert to string
	for _, value := range req.ConfigValue.Elements() {
		resp.Diagnostics.Append(validateArn(value)...)
	}
}

func (v arnValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		// If the value is null, no need to validate (optional field)
		return
	}

	// Convert to string
	resp.Diagnostics.Append(validateArn(req.ConfigValue)...)
}

func validateArn(value attr.Value) (diag diag.Diagnostics) {
	if value.IsNull() {
		diag.AddError(
			"input is empty",
			"expected type of input to be a string",
		)
		return
	}

	parsedARN, err := arn.Parse(value.String())

	if err != nil {
		diag.AddError(
			"invalid arn",
			fmt.Sprintf("%s is an invalid ARN: %s", value, err),
		)
		return
	}

	if parsedARN.Partition == "" {
		diag.AddError(
			"missing partition value",
			fmt.Sprintf("%s is an invalid ARN: missing partition value", value),
		)
	} else if !partitionRegexp.MatchString(parsedARN.Partition) {
		diag.AddError(
			"invalid partition value",
			fmt.Sprintf("%s is an invalid ARN: invalid partition value (expecting to match regular expression: %s)", value, partitionRegexp),
		)
	}

	if parsedARN.Region != "" && !regionRegexp.MatchString(parsedARN.Region) {
		diag.AddError(
			"invalid region value",
			fmt.Sprintf("%s is an invalid ARN: invalid region value (expecting to match regular expression: %s)", value, regionRegexp),
		)
	}

	if parsedARN.AccountID != "" && !accountIDRegexp.MatchString(parsedARN.AccountID) {
		diag.AddError(
			"invalid account ID value",
			fmt.Sprintf("%s is an invalid ARN: invalid account ID value (expecting to match regular expression: %s)", value, accountIDRegexp),
		)
	}

	if parsedARN.Resource == "" {
		diag.AddError(
			"missing resource value",
			fmt.Sprintf("%s is an invalid ARN: missing resource value", value),
		)
	}

	return
}

// Custom validator to ensure param_b is set only if param_a has a specific value
type dependentParameterValidator struct {
	dependentParamName string
	requiredValue      []string
}

func (d dependentParameterValidator) Description(ctx context.Context) string {
	return "Validates against dependent parameter's value"
}

func (d dependentParameterValidator) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

// Validate ensures that one parameter can only be set if another parameter has a specific value
func (v dependentParameterValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if the current parameter is not set
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Retrieve the value of the dependent parameter (param_a)
	var dependentParamValue types.String
	diags := req.Config.GetAttribute(ctx, path.Root(v.dependentParamName), &dependentParamValue)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Check if param_a has the required value, e.g., "enabled"
	for _, val := range v.requiredValue {
		if dependentParamValue.ValueString() == val {
			return
		}
	}

	resp.Diagnostics.AddError(
		"Invalid Configuration",
		fmt.Sprintf("'%s' can only be set if '%s' has the value '%s'.", req.Path.String(), v.dependentParamName, v.requiredValue),
	)
}

// // Helper function to create the custom validator
// func dependentParameter(dependentParamName string, requiredValue string) validators.String {
// 	return dependentParameterValidator{
// 		dependentParamName: dependentParamName,
// 		requiredValue:      requiredValue,
// 	}
// }
