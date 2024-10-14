resource "fastssm_parameter" "example" {
  name           = "some-ssm-parameter"
  type           = "String"
  insecure_value = "some-insecure-value"
  description    = "An example description"
}