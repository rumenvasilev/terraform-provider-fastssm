provider "fastssm" {
  region = "eu-west-1"
  # Optional
  endpoints {
    ssm = "https://ssm.eu-west-1.amazonaws.com"
  }
}