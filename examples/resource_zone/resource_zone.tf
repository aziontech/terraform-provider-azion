terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "61d1047ec2ef6e4dbf59a4cf14650809ee9c42df"
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
