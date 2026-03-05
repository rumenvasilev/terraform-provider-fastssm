terraform {
  required_providers {
    fastssm = {
      source = "rumenvasilev/fastssm"
    }
  }
}

provider "fastssm" {}

resource "fastssm_parameter" "test" {
  name  = "/e2e/test/no-config"
  type  = "String"
  value = "should-fail"
}
