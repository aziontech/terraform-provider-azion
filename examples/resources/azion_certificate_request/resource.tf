resource "azion_certificate_request" "example" {
  results = {
    name        = "my-letsencrypt-certificate"
    common_name = "example.com"
    challenge   = "dns"
    authority   = "lets_encrypt"

    alternative_names = [
      "www.example.com",
      "api.example.com"
    ]
  }
}