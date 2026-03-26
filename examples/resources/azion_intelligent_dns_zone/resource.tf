resource "azion_intelligent_dns_zone" "example" {
  zone = {
    domain : "example.com",
    active : true,
    name : "example"
  }
}
