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

resource "azion_zone" "dev" {
  zone = {
      domain: "test-test.com",
      is_active: true,
      name: "test3 Alterar terraform"
    }
}
resource "azion_zone" "dev2" {
  zone = {
    domain: "test2-test2.com",
    is_active: true,
    name: "test2 Alterar terraform"
  }
}
resource "azion_zone" "dev3" {
  zone = {
    domain: "test5-test5.com",
    is_active: true,
    name: "test5 show terraform"
  }
}


output "dev_order" {
  value = azion_zone.dev
}


