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

resource "azion_dnssec" "dev" {
  zone_id = 2580
  dns_sec = {
      is_enabled = true
    }
}

output "dev_order" {
  value = azion_dnssec.dev
}
