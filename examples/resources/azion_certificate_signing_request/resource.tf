terraform {
  required_providers {
    azion = {
      source  = "azion/azion"
      version = ">= 1.0.0"
    }
  }
}

provider "azion" {
  api_token = var.api_token
}

# Create a Certificate Signing Request (CSR)
# This will generate a new certificate with a CSR that can be sent to a Certificate Authority
# The resource supports Create, Read, Delete, and Import operations.
# Update is not supported - any changes will require resource recreation.
resource "azion_certificate_signing_request" "example" {
  results = {
    name               = "My Certificate CSR"
    common_name        = "example.com"
    country            = "US"
    state              = "California"
    locality           = "San Francisco"
    organization       = "Example Organization"
    organization_unity = "IT Department"
    email              = "admin@example.com"
    
    # Optional fields
    alternative_names  = ["www.example.com", "api.example.com"]
    key_algorithm      = "rsa_2048"
    active             = true
  }
}

# The CSR will be available in the results.csr attribute
# You can extract it and send it to your Certificate Authority for signing
output "csr_content" {
  value     = azion_certificate_signing_request.example.results.csr
  sensitive = true
}

# The generated certificate ID
output "certificate_id" {
  value = azion_certificate_signing_request.example.results.id
}

# Certificate status
output "certificate_status" {
  value = azion_certificate_signing_request.example.results.status
}

# Import an existing certificate:
# terraform import azion_certificate_signing_request.example 12345
