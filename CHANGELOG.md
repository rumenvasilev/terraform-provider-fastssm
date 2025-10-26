## 0.2.0 (unreleased)
FEATURES:
* implement ephemeral resource (terraform 1.10+)

FIXES:
* implement `skip_credentials_validation` provider configuration (was defined but not functional)
* correct `endpoints` schema from SetNestedAttribute to SingleNestedBlock for proper HCL block syntax; provider configuration now uses proper block syntax instead of attribute list
* removed trailing punctuation per Go style guide in error messages

ENHANCEMENTS:
* upgraded framework libs
* upgraded go to 1.25.3
* upgrade github action versions
* comprehensive e2e test suite with LocalStack (docker-compose setup, GitHub Actions integration)
* added Make targets for e2e testing (`e2e-test`, `e2e-up`, `e2e-down`, `e2e-logs`, `e2e-clean`)

DOCUMENTATION:
* updated README to reflect production-ready status (1+ year, tens of thousands of parameters)
* clarified unsupported features and trade-offs (tags, tier, custom KMS keys)
* improved "Why FastSSM?" section with emphasis on rate limit failures
* corrected version requirements throughout documentation

## 0.1.6

FIXES:
* correctly wrap retry errors, so they are recognised as such, instead of generic client error
* region override must use WithRegion() function call so it superseeds any others, including WithDefaultRegion()

## 0.1.5

FIXES:
* region override in combination with profile works

## 0.1.4

FIXES:
* CI: remove build for terraform 1.7 - it's not supported by the provider
* make testing working locally
* fix logging of retries
* add retries to describe call

## 0.1.3

FEATURES:
* migration support with moved{} block from `aws_ssm_parameter` to `fastssm_parameter`

FIXES:
* properly handle `insecure_value` and `value` in the state

## 0.1.2

FEATURES:
* validate AWS credentials before sending any API calls
* update documentation

## 0.1.1

FEATURES:
* provision SSM parameters (resource and data_source)
* improve read process of any SSM parameter significantly, by reducing DescribeParameter calls significantly
