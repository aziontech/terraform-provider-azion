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

resource "azion_order" "dev" {
  zone = {
      domain: "1-test.com",
      is_active: true,
      name: "test create terraform"
    }
}

output "dev_order" {
  value = azion_order.dev
}
