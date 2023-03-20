terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "ee63d648794b8262a27ebf2eebd5b9535bd091a3"
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
