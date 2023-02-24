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

data "azion_records" "dev" {
  zoneid = 
}

output "dev_records" {
  value = data.azion_records.dev
}
