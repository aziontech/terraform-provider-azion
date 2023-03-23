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

resource "azion_record" "dev" {
  zone_id = 2638
  record = {
    record_type= "A"
    entry = "site"
    answers_list = [
      "8.8.8.8"
    ]
    # policy = "simple"
    policy = "weighted"
    weight = 50
    description = "This is a description"
    ttl = 20
  }
}

output "dev_record" {
  value = azion_record.dev
}
