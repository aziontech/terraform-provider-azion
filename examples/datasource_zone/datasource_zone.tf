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
data "azion_zone" "dev" {
  id = 2580
}

output "dev_zone" {
  value = data.azion_zone.dev
}
