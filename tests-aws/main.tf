# 1234 params # high throughtput ssm #
# apply # ~/Downloads/terraform_1.8.5_darwin_arm64/terraform apply -auto-approve  54.74s user 5.34s system 41% cpu 2:26.49 total
# plan # ~/Downloads/terraform_1.8.5_darwin_arm64/terraform plan  31.87s user 4.93s system 23% cpu 2:35.18 total
# destroy # ~/Downloads/terraform_1.8.5_darwin_arm64/terraform apply -destroy   155.90s user 10.12s system 36% cpu 7:29.39 total

terraform {
  required_providers {
    aws = {
      source = "aws"
      # version = "0.1.0"
    }
  }
}


provider "aws" {
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

resource "aws_ssm_parameter" "test" {
    count = 1234
    name = "/pencho/vladigerov/${count.index}"
    type = "String"
    insecure_value = "lsakdfasdr4jwe"
    description = "Just renaming to something else. 3"
    # overwrite = true
    tags = local.ssm_tags
}

output "this" {
  value = aws_ssm_parameter.test
  sensitive = true
}
