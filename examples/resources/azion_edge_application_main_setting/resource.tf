resource "azion_edge_application_main_setting" "example" {
  edge_application = {
    name                     = "Terraform Examples"
    supported_ciphers        = "all"
    delivery_protocol        = "http,https"
    http_port                = [80, 8080]
    https_port               = [443]
    minimum_tls_version      = ""
    debug_rules              = false
    caching                  = true
    edge_functions           = false
    image_optimization       = false
    http3                    = false
    application_acceleration = false
    l2_caching               = false
    load_balancer            = false
    raw_logs                 = true
    device_detection         = false
  }
}

resource "azion_edge_application_origin" "example" {
  edge_application_id = azion_edge_application_main_setting.example.edge_application.application_id
  origin = {
    name        = "Terraform Main Settings Example"
    origin_type = "single_origin"
    addresses : [
      {
        "address" : "httpExample.org"
      }
    ],
    origin_protocol_policy : "https",
    host_header : "$${host}",
  }
  depends_on = [
    azion_edge_application_main_setting.example
  ]
}

resource "azion_edge_application_cache_setting" "example" {
  edge_application_id = azion_edge_application_main_setting.example.edge_application.application_id
  cache_settings = {
    name                               = "Terraform Main Settings Example"
    browser_cache_settings             = "override"
    browser_cache_settings_maximum_ttl = 20
    cdn_cache_settings                 = "override"
    cdn_cache_settings_maximum_ttl     = 60
    adaptive_delivery_action           = "ignore"
    cache_by_query_string              = "ignore"
    cache_by_cookies                   = "ignore"
    enable_stale_cache                 = true
  }
  depends_on = [
    azion_edge_application_main_setting.example,
    azion_edge_application_origin.example
  ]
}
