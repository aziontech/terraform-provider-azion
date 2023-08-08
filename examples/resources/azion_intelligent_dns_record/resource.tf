resource "azion_intelligent_dns_record" "examples" {
  zone_id = "12345"
  record = {
    record_type= "A"
    entry = "site"
    answers_list = [
      "8.8.8.8"
    ]
    policy = "weighted"
    weight = 50
    description = "This is a description"
    ttl = 20
  }
}
