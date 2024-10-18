### Basic example

resource "fastssm_parameter" "example" {
  name           = "some-ssm-parameter"
  type           = "String"
  insecure_value = "some-insecure-value"
  description    = "An example description"
}

### Encrypted string with default KMS key

resource "fastssm_parameter" "secure" {
  name        = "some-encrypted-ssm-parameter"
  type        = "SecureString"
  value       = "some-secure-value"
  description = "An example description"
}