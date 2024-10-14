# apply # ~/Downloads/terraform_1.8.5_darwin_arm64/terraform apply -auto-approve  45.48s user 5.46s system 36% cpu 2:18.35 total
# plan # ~/Downloads/terraform_1.8.5_darwin_arm64/terraform plan  19.60s user 2.55s system 140% cpu 15.771 total
# destroy # ~/Downloads/terraform_1.8.5_darwin_arm64/terraform apply -destroy   119.41s user 6.69s system 47% cpu 4:22.89 total

terraform {
  required_providers {
    fastssm = {
      source = "rumenvasilev/fastssm"
      # version = "0.1.0"
    }
  }
}


provider "fastssm" {
  # provider-specific configurations
  region = "eu-west-1"
}

locals {
  ssm_tags = merge(
    {
      just-testing-bro = "some-nice-tag-ssm"
    },
  )
}

resource "fastssm_parameter" "test" {
    count = 1234
    name = "/pencho/vladigerov/${count.index}"
    type = "String"
    insecure_value = "lsakdfasdr4jwe"
    description = "Just renaming to something else. 3"
    # overwrite = true
    tags = local.ssm_tags
}

output "this" {
  value = fastssm_parameter.test
  sensitive = true
}
