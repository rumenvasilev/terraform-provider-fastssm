## Unreleased

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
