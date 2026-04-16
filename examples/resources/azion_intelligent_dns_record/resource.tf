resource "azion_intelligent_dns_record" "example" {
  zone_id = "12345"
  record = {
    type = "A"
    name = "site"
    rdata = [
      "8.8.8.8"
    ]
    policy      = "weighted"
    weight      = 50
    description = "This is a description"
    ttl         = 20
  }
}
