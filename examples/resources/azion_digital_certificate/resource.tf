# Example using the TLS provider to generate a self-signed certificate
# This is useful for testing purposes
resource "tls_private_key" "example" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "example" {
  private_key_pem = tls_private_key.example.private_key_pem

  subject {
    common_name  = "example.com"
    organization = "Example Organization"
  }

  validity_period_hours = 8760 # 1 year

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]

  dns_names = [
    "example.com",
    "*.example.com",
  ]
}

# Create a digital certificate in Azion
resource "azion_digital_certificate" "example" {
  results = {
    name                = "My Certificate"
    certificate_content = tls_self_signed_cert.example.cert_pem
    private_key         = tls_private_key.example.private_key_pem
  }
}

# Example using local files for certificate and private key
# This requires you to have the certificate.pem and private_key.pem files in the same directory
resource "azion_digital_certificate" "from_file" {
  results = {
    name                = "My Certificate from File"
    certificate_content = file("${path.module}/certificate.pem")
    private_key         = file("${path.module}/private_key.pem")
  }
}

# Output the certificate ID
output "certificate_id" {
  description = "The ID of the created certificate"
  value       = azion_digital_certificate.example.results.id
}

output "certificate_status" {
  description = "The status of the created certificate"
  value       = azion_digital_certificate.example.results.status
}
