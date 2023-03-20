terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "azion9252873bb250dfac51625125cfd702e57af"
}
data "azion_zone" "dev" {
  id = 2580
}

output "dev_zone" {
  value = data.azion_zone.dev
}
