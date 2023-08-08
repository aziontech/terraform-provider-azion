resource "azion_edge_application_origin" "example" {
  edge_application_id = 1234567890
  origin = {
    name = "Terraform Example"
    origin_type = "single_origin"
    addresses: [
      {
        "address": "terraform.org"
      }
    ],
    origin_protocol_policy: "http",
    host_header: "$${host}",
    origin_path: "/requests",
    hmac_authentication: false,
    hmac_region_name: "",
    hmac_access_key: "",
    hmac_secret_key: ""
  }
}