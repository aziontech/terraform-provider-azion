terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "5865e6915209e1c58ed111a513a03b8ad1d44fed"
}

data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}
