# Configure the Azion provider using the required_providers stanza
# required with Terraform 0.13 and beyond. You may optionally use version
# directive to prevent breaking changes occurring unannounced.
terraform {
  required_providers {
    cloudflare = {
      source = "aziontech/azion"
      version = "~≳ 0.2.0"
    }
  }
}

provider "azion" {
  api_token = "<token>"
}

# Create a zone
resource "azion_zone" "www" {
  # ...
}

# Create a record
resource "azion_record" "www" {
  # ...
}