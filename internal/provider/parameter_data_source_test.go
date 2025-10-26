package provider

import (
	"fmt"
	"terraform-provider-fastssm/internal/names"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccParameterDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create resource and read with data source
			{
				Config: testAccParameterDataSourceConfigWithResource("test_param_data_source", "hello world"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check the resource was created
					resource.TestCheckResourceAttr("fastssm_parameter.test", names.AttrName, "test_param_data_source"),
					resource.TestCheckResourceAttr("fastssm_parameter.test", names.AttrValue, "hello world"),
					// Check the data source reads it correctly
					resource.TestCheckResourceAttr("data.fastssm_parameter.test", names.AttrName, "test_param_data_source"),
					resource.TestCheckResourceAttr("data.fastssm_parameter.test", names.AttrValue, "hello world"),
					resource.TestCheckResourceAttr("data.fastssm_parameter.test", names.AttrType, "String"),
					resource.TestCheckResourceAttr("data.fastssm_parameter.test", names.AttrVersion, "1"),
					resource.TestCheckResourceAttr("data.fastssm_parameter.test", names.AttrARN, "arn:aws:ssm:us-east-1:000000000000:parameter/test_param_data_source"),
				),
			},
		},
	})
}

func testAccParameterDataSourceConfigWithResource(name, value string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "fastssm_parameter" "test" {
  name  = %[1]q
  value = %[2]q
  type  = "String"
}

data "fastssm_parameter" "test" {
  depends_on = [fastssm_parameter.test]
  name       = %[1]q
}
`, name, value)
}
