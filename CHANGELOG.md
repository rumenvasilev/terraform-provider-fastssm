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
