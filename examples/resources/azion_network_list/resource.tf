resource "azion_network_list" "exampleOne" {
  data = {
    name = "New NetworkList for terraform Countries"
    type = "countries"
    items = [
      "BR",
      "US",
      "AG"
    ]
  }
}

resource "azion_network_list" "exampleTwo" {
  data = {
    name = "New NetworkList for terraform ip_cidr"
    type = "ip_cidr"
    items = [
      "192.168.0.1",
      "192.168.0.2",
      "192.168.0.3"
    ]
  }
}