# Example usage of the azion_crl resource
# This resource creates a Certificate Revocation List (CRL) for verifying the revocation status of X.509 digital certificates.

resource "azion_crl" "example" {
  crl = {
    name   = "My Certificate Revocation List"
    issuer = "CN=My Certificate Authority,O=My Organization,C=US"
    crl    = <<-EOT
      -----BEGIN X509 CRL-----
      MIIBpzCCAVMCAQEwDQYJKoZIhvcNAQELBQAwGjEYMBYGA1UEAwwPVGhpcyBpcyBh
      IHRlc3QgQ0EXDTI0MDEwMTAwMDAwMFoXDTI1MDEwMTAwMDAwMFqgDjAMMAoGA1Ud
      FAQDAgEBMAoGA1UdFAQDAgEAMA0GCSqGSIb3DQEBCwUAA4IBAQC0test
      -----END X509 CRL-----
    EOT
  }
}

# Example with active field explicitly set
resource "azion_crl" "active_example" {
  crl = {
    name   = "Active CRL Example"
    active = true
    issuer = "CN=My Certificate Authority,O=My Organization,C=US"
    crl    = <<-EOT
      -----BEGIN X509 CRL-----
      MIIBpzCCAVMCAQEwDQYJKoZIhvcNAQELBQAwGjEYMBYGA1UEAwwPVGhpcyBpcyBh
      IHRlc3QgQ0EXDTI0MDEwMTAwMDAwMFoXDTI1MDEwMTAwMDAwMFqgDjAMMAoGA1Ud
      FAQDAgEBMAoGA1UdFAQDAgEAMA0GCSqGSIb3DQEBCwUAA4IBAQC0test
      -----END X509 CRL-----
    EOT
  }
}
