resource "azion_network_list" "exampleOne" {
  results = {
    name  = "New NetworkList for terraform Countries"
    list_type = "countries"
    items_values_str = [
      "BR",
      "US",
      "AG"
    ]
  }
}

resource "azion_network_list" "exampleTwo" {
  results = {
    name  = "New NetworkList for terraform ip_cidr"
    list_type = "ip_cidr"
    items_values_str = [
      "192.168.0.1",
      "192.168.0.2",
      "192.168.0.3"
    ]
  }
}