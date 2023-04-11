resource "azion_record" "examples" {
  zone_id = "<zone_id>"
  record = {
    record_type= "A"
    entry = "site"
    answers_list = [
      "8.8.8.8"
    ]
    # policy = "simple"
    policy = "weighted"
    weight = 50
    description = "This is a description"
    ttl = 20
  }
}
