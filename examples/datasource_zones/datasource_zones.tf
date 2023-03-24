terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "<token>"
}
data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}
