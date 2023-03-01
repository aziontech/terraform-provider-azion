terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "4ebf949f99d68d2092dc50284ce7269ef6413193"
}
data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}
