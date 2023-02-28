terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "<security token>"
}

data "azion_records" "dev" {
  zone_id = <zone_id>
}

output "dev_records" {
  value = data.azion_records.dev
}
