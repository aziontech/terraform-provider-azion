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

resource "azion_records" "dev" {
  zone_id = 2553
  record = {
    record_type= "A"
    entry = "site2"
    answers_list = [
      "8.8.8.8",
      "1.1.1.1"
    ]
    policy = "simple"
    ttl = 20
  }
}

output "dev_records" {
  value = azion_records.dev
}
