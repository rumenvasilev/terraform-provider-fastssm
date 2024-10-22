package provider

import (
	"terraform-provider-fastssm/internal/names"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ParameterResourceModel describes the resource data model.
type awsSSMParameterResourceModel struct {
	AllowedPattern types.String `tfsdk:"allowed_pattern"`
	Arn            types.String `tfsdk:"arn"`
	DataType       types.String `tfsdk:"data_type"`
	Description    types.String `tfsdk:"description"`
	Id             types.String `tfsdk:"id"`
	InsecureValue  types.String `tfsdk:"insecure_value"`
	KeyId          types.String `tfsdk:"key_id"`
	Name           types.String `tfsdk:"name"`
	Overwrite      types.Bool   `tfsdk:"overwrite"`
	Tags           types.Map    `tfsdk:"tags"`
	TagsAll        types.Map    `tfsdk:"tags_all"`
	Tier           types.String `tfsdk:"tier"`
	Type           types.String `tfsdk:"type"`
	Value          types.String `tfsdk:"value"`
	Version        types.Int64  `tfsdk:"version"`
}

func awsSSMParameterResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Provides an SSM Parameter resource.",
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
			"id": schema.StringAttribute{},
			"insecure_value": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.ConflictsWith(path.Expressions{
							path.MatchRoot(names.AttrValue),
						}...),
						dependentParameterValidator{dependentParamName: "type", requiredValue: []string{"String", "StringList"}},
					)},
			},
			names.AttrKeyID: schema.StringAttribute{},
			names.AttrName: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{stringvalidator.LengthBetween(1, 2048)},
			},
			"overwrite": schema.BoolAttribute{
				Optional: true,
			},
			names.AttrTags: schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			names.AttrTagsAll: schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"tier": schema.StringAttribute{},
			names.AttrType: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("String", "StringList", "SecureString"),
				},
			},
			names.AttrValue: schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.ConflictsWith(path.Expressions{
							path.MatchRoot("insecure_value"),
						}...),
					)},
			},
			names.AttrVersion: schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}
