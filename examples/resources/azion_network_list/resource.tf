resource "azion_network_list" "example_countries" {
  results = {
    name = "NetworkList Countries Example"
    type = "countries"
    items = [
      "BR",
      "US",
      "AG"
    ]
  }
}

resource "azion_network_list" "example_ip_cidr" {
  results = {
    name = "NetworkList IP CIDR Example"
    type = "ip_cidr"
    items = [
      "192.168.0.1",
      "192.168.0.2",
      "192.168.0.3"
    ]
  }
}

resource "azion_network_list" "example_asn" {
  results = {
    name = "NetworkList ASN Example"
    type = "asn"
    items = [
      "1234",
      "5678",
      "13335"
    ]
  }
}
