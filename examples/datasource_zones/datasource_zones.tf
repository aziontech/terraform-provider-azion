terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "b79f65853ff3386e4b3678397a808fe0aaf6fd5c"
}
data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}