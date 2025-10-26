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
				Config: testAccParameterResourceConfig("one", "fake value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("fastssm_parameter.test", "name", "one"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", "value", "fake value"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", "type", "String"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", "insecure_value", "fake value"),
				),
			},
			// ImportState testing
			// Requires ID in the schema, which we don't have currently
			// {
			// 	ResourceName:      "fastssm_parameter.test",
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// 	// This is not normally necessary, but is here because this
			// 	// Parameter code does not have an actual upstream service.
			// 	// Once the Read method is able to refresh information from
			// 	// the upstream service, this can be removed.
			// 	ImportStateVerifyIgnore: []string{"name", "one"},
			// },
			// Update and Read testing
			{
				Config: testAccParameterResourceConfig("two", "fake value bom bom"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("fastssm_parameter.test", "name", "two"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", "value", "fake value bom bom"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccParameterResourceConfig(name, value string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "fastssm_parameter" "test" {
  name = %[1]q
  value = %[2]q
  type = "String"
}
`, name, value)
}
