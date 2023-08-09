resource "azion_intelligent_dns_zone" "example" {
  zone = {
    domain : "example.com",
    is_active : true,
    name : "example"
  }
}
