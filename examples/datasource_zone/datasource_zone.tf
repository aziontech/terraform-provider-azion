terraform {
  required_providers {
    azion = {
      source = "aziontech/azion"
      version = "0.2.0"
    }
  }
}
provider "azion" {
  api_token  = "token"
}

data "azion_zones" "getAll" {}

data "azion_zone" "byID" {
  id = 2580
}

output "dev_zone" {
  value = data.azion_zone.byID
}

output "dev_zones" {
  value = data.azion_zones.getAll.id
}
