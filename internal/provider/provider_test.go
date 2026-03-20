package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"fastssm": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Skip acceptance tests if AWS credentials are not configured.
	// These tests require real AWS access or LocalStack and should only run
	// in environments where infrastructure is available (e.g., e2e test workflow).
	requiredVars := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			t.Skipf("Skipping acceptance test: environment variable %s not set", v)
		}
	}
}

// testAccProviderConfig generates provider configuration for acceptance tests.
// If LOCALSTACK_ENDPOINT is set, it configures the provider to use LocalStack.
func testAccProviderConfig() string {
	localstackEndpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	if localstackEndpoint != "" {
		return fmt.Sprintf(`
provider "fastssm" {
  region = %[2]q
  endpoints {
    ssm = %[1]q
    sts = %[1]q
  }
  skip_credentials_validation = true
}
`, localstackEndpoint, region)
	}

	return fmt.Sprintf(`
provider "fastssm" {
  region = %[1]q
}
`, region)
}

func TestProviderRequiresRegion(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "fastssm" {}

resource "fastssm_parameter" "test" {
  name  = "/test/missing-region"
  value = "test"
  type  = "String"
}
`,
				ExpectError: regexp.MustCompile(`(?i)region`),
			},
		},
	})
}
