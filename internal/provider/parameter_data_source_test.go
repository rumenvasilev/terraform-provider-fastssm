package provider

import (
	"terraform-provider-fastssm/internal/names"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccParameterDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccParameterDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.fastssm_parameter.test", names.AttrName, "test"),
				),
			},
		},
	})
}

const testAccParameterDataSourceConfig = `
data "fastssm_parameter" "test" {
  name = "test"
}
`
