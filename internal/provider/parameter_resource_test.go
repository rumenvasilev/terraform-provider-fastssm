package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccParameterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccParameterResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("fastssm_parameter.test", "name", "one"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", "value", "one"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", "overwrite", "false"),
					// resource.TestCheckResourceAttr("fastssm_parameter.test", "defaulted", "Parameter value when not configured"),
					// resource.TestCheckResourceAttr("fastssm_parameter.test", "id", "Parameter-id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "fastssm_parameter.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// Parameter code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{"name", "one"},
			},
			// Update and Read testing
			{
				Config: testAccParameterResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("fastssm_parameter.test", "name", "two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccParameterResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "fastssm_parameter" "test" {
  name = %[1]q
  insecure_value = %[1]q
  type = "String"
}
`, configurableAttribute)
}
