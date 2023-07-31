resource "azion_edge_application_main_setting" "example" {
  edge_application = {
    name = "Terraform Example Main Settings"
  }
}

resource "azion_edge_application_origin" "example" {
  edge_application_id = azion_edge_application_main_setting.example.edge_application.application_id
  origin = {
    name = "Terraform Main Settings Example"
    origin_type = "single_origin"
    addresses: [
      {
        "address": "httpExample.org"
      }
    ],
    origin_protocol_policy: "https",
    host_header: "$${host}",
  }
  depends_on = [
    azion_edge_application_main_setting.example
  ]
}

resource "azion_edge_application_cache_setting" "example" {
  edge_application_id = azion_edge_application_main_setting.example.edge_application.application_id
  cache_settings = {
    name = "Terraform Main Settings Example"
    browser_cache_settings = "override"
    browser_cache_settings_maximum_ttl = 20
    cdn_cache_settings = "override"
    cdn_cache_settings_maximum_ttl = 60
    adaptive_delivery_action = "ignore"
    cache_by_query_string = "ignore"
    cache_by_cookies= "ignore"
    enable_stale_cache = true
  }
  depends_on = [
    azion_edge_application_main_setting.example,
    azion_edge_application_origin.example
  ]
}
