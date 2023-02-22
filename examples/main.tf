terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "5d8ece2bd706de3823577ef25c170748437568ed"
}

data "azion_zones" "dev" {}

output "dev_zones" {
  value = data.azion_zones.dev
}
