terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = ""
}

resource "azion_zone" "dev" {
  zone = {
      domain: "5-test.com",
      is_active: true,
      name: "test Alterado terraform"
    }
}

output "dev_order" {
  value = azion_zone.dev
}
