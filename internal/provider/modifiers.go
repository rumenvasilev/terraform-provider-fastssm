package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SyncAttributePlanModifier(target string) planmodifier.String {
	return &syncAttributePlanModifier{target}
}

type syncAttributePlanModifier struct {
	target string
}

func (d *syncAttributePlanModifier) Description(ctx context.Context) string {
	return "Ensures two attributes are kept synchronised."
}

func (d *syncAttributePlanModifier) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (d *syncAttributePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// var attributeOne types.String
	attributeOne := req.PlanValue
	// diags := req.Plan.GetAttribute(ctx, path.Root(d.first), &attributeOne)
	// resp.Diagnostics.Append(diags...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }

	var attributeTwo types.String
	diags := req.Plan.GetAttribute(ctx, path.Root(d.target), &attributeTwo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default to using value for attribute_two if attribute_one is null
	if attributeOne.IsNull() && !attributeTwo.IsNull() {
		resp.PlanValue = attributeTwo
		return
	}

	// Default to using value for attribute_one if attribute_two is null
	if !attributeOne.IsNull() && attributeTwo.IsNull() {
		resp.PlanValue = attributeOne
		return
	}
}
