terraform {
  required_providers {
    fastssm = {
      source = "rumenvasilev/fastssm"
      # No version constraint when using dev_overrides
    }
  }
}

provider "fastssm" {
  region                      = "us-east-1"
  access_key                  = "test"
  secret_key                  = "test"
  skip_credentials_validation = true

  endpoints {
    ssm = "http://localhost:4566"
    sts = "http://localhost:4566"
  }
}

# Test 1: Create a basic string parameter
resource "fastssm_parameter" "test_string" {
  name  = "/e2e/test/string"
  type  = "String"
  value = "test-value-123"
}

# Test 2: Create a StringList parameter
resource "fastssm_parameter" "test_stringlist" {
  name  = "/e2e/test/stringlist"
  type  = "StringList"
  value = "item1,item2,item3"
}

# Test 3: Create a SecureString parameter
resource "fastssm_parameter" "test_securestring" {
  name  = "/e2e/test/securestring"
  type  = "SecureString"
  value = "secret-password-456"
}

# Test 4: Create a parameter with description
resource "fastssm_parameter" "test_with_description" {
  name        = "/e2e/test/with-description"
  type        = "String"
  value       = "test-value-with-desc"
  description = "This is a test parameter with description"
}

# Test 5: Create a parameter with allowed pattern
resource "fastssm_parameter" "test_with_pattern" {
  name            = "/e2e/test/with-pattern"
  type            = "String"
  value           = "test123"
  allowed_pattern = "^test[0-9]+$"
}

# Test 6: Create a non-sensitive string parameter (using value, not insecure_value)
resource "fastssm_parameter" "test_insecure" {
  name  = "/e2e/test/insecure"
  type  = "String"
  value = "visible-value"
}

# Test 7: Create a parameter with data_type
resource "fastssm_parameter" "test_datatype" {
  name      = "/e2e/test/datatype"
  type      = "String"
  value     = "text-data"
  data_type = "text"
}

# Data source tests
data "fastssm_parameter" "test_string" {
  name       = fastssm_parameter.test_string.name
  depends_on = [fastssm_parameter.test_string]
}

data "fastssm_parameter" "test_securestring" {
  name       = fastssm_parameter.test_securestring.name
  depends_on = [fastssm_parameter.test_securestring]
}

# Outputs for verification
output "test_string_value" {
  value     = data.fastssm_parameter.test_string.value
  sensitive = true
}

output "test_string_arn" {
  value = fastssm_parameter.test_string.arn
}

output "test_string_version" {
  value = fastssm_parameter.test_string.version
}

output "test_securestring_value" {
  value     = data.fastssm_parameter.test_securestring.value
  sensitive = true
}

output "all_parameter_names" {
  value = [
    fastssm_parameter.test_string.name,
    fastssm_parameter.test_stringlist.name,
    fastssm_parameter.test_securestring.name,
    fastssm_parameter.test_with_description.name,
    fastssm_parameter.test_with_pattern.name,
    fastssm_parameter.test_insecure.name,
    fastssm_parameter.test_datatype.name,
  ]
}

