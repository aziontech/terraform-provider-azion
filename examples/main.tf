terraform {
  required_providers {
    azion = {
      source  = "hashicorp.com/dev/azion"
    }
  }
}
provider "azion" {
  api_token  = "5941152beb56dd1f27425a9b1744ff407d903d64"
}

data "azion" "example" {}