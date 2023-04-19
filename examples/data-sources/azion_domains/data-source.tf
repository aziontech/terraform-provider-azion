terraform {
  required_providers {
    azion = {
      source  = "github.com/actions/azion"
    }
  }
}

provider "azion" {
  api_token  = "azion9252873bb250dfac51625125cfd702e57af"
}

data "azion_domains" "dev" {
}

output "dev_domains" {
  value = data.azion_domains.dev
}