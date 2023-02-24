terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "22692e0c6f5bb53800e40ae139d32e5a5f58ae82"
}

data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}
