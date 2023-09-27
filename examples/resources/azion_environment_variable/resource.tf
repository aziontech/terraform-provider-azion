resource "azion_environment_variable" "example" {
  result = {
    key    = "key-test Terraform"
    value  = "key-test Terraform"
    secret = false
  }
}