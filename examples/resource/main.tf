terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "25b58e5da327dfe81e77adf99a6cee05c27e7c3d"
}

resource "azion_order" "dev" {
  zone = {
      domain: "ex11.com",
      is_active: true,
      name: "Hosted Zone criado pela terraform"
    }
}

output "dev_order" {
  value = azion_order.dev
}
