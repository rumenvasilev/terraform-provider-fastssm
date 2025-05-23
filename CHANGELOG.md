## 0.2.0 (unreleased)
FEATURES:
* implement ephemeral resource (terraform 1.10+)
* upgraded framework libs

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
