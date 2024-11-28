package provider

import (
	"terraform-provider-fastssm/internal/names"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccParameterEphemeral(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccParameterEphemeralConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ephemeral.fastssm_parameter.test", names.AttrName, "test"),
				),
			},
		},
	})
}

const testAccParameterEphemeralConfig = `
ephemeral "fastssm_parameter" "test" {
  name = "test"
}
`
