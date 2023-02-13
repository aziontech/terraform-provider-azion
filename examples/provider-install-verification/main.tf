terraform {
  required_providers {
    hashicups = {
      source = "hashicorp.com/azion/azion-pf"
    }
  }
}

provider "azion" {}

provider "hashicups" {
  APIToken  = "***"
}

data "azion" "example" {}