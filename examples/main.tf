terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "55e6a1a3b0a16d30ba70a360bb19566724244226"
}

data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}
