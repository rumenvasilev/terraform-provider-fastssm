╷
│ Error: SSM parameter create error
│ 
│   with fastssm_parameter.test[146],
│   on main.tf line 32, in resource "fastssm_parameter" "test":
│   32: resource "fastssm_parameter" "test" {
│ 
│ creating SSM Parameter ("/pencho/vladigerov/146"): operation error SSM: PutParameter, failed to get rate limit token, retry quota exceeded, 3 available, 5 requested
╵



RATE LIMIT
╷
│ Error: Client Error
│ 
│   with fastssm_parameter.test[1045],
│   on main.tf line 32, in resource "fastssm_parameter" "test":
│   32: resource "fastssm_parameter" "test" {
│ 
│ Unable to read example, got error: operation error SSM: GetParameter, exceeded maximum number of attempts, 3, https response error StatusCode: 400, RequestID: 95b7b2da-bb17-4e52-9fac-9223024a1721, api error ThrottlingException: Rate exceeded
╵


AWS_DEFAULT_REGION fixes this, passing region within provider config produces the following
╷
│ Error: SSM parameter create error
│ 
│   with fastssm_parameter.test[524],
│   on main.tf line 29, in resource "fastssm_parameter" "test":
│   29: resource "fastssm_parameter" "test" {
│ 
│ creating SSM Parameter ("/pencho/vladigerov/524"): permanent failure: operation error SSM: PutParameter, https response error StatusCode: 0, RequestID: , request send failed,
│ Post "https://ssm.\"eu-west-1\".amazonaws.com/": dial tcp: lookup ssm."eu-west-1".amazonaws.com: no such host


│ Error: creating SSM Parameter (/pencho/vladigerov/852): operation error SSM: PutParameter, https response error StatusCode: 400, RequestID: 1ba6559f-3b09-4201-93cd-8856519dbebd, ParameterAlreadyExists: The parameter already exists. To overwrite this value, set the overwrite option in the request to true.
