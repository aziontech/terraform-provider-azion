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

resource "azion_order" "dev" {
  zone = {
      domain: "ex3.com",
      is_active: true,
      name: "Hosted Zone criado pela terraform"
    }
}

output "dev_order" {
  value = azion_order.dev
}
