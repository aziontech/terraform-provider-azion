# Configure the Azion provider using the required_providers
# required with Terraform 0.13 and beyond. You may optionally use version
# directive to prevent breaking changes occurring unannounced.
terraform {
  required_providers {
    azion = {
      source = "registry.terraform.io/aziontech/azion"
      version = "~≳ <version>"
    }
  }
}

provider "azion" {
  api_token = "<token>"
}

# Create a zone
resource "azion_zone" "example" {
  # ...
}

# Create a record
resource "azion_record" "example" {
  # ...
}