terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "ae21724627cc5f3fac461060caf86ac0dacfa00f"
}

resource "azion_order" "dev" {
  zone = {
      domain: "3ex.com",
      is_active: true,
      name: "Hosted Zone criado pela terraform"
    }
}

output "dev_order" {
  value = azion_order.dev
}
