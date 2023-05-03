resource "azion_domain" "example" {
  domain = {
    cnames: [
      "www.example.com",
      "www.example2.com"
    ]
    name = "Terraform-domain-example"
    digital_certificate_id = null
    cname_access_only = true/false
    edge_application_id = <edge_application_id>
    is_active = true/false
  }
}