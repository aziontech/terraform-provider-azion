terraform {
  required_providers {
    azion = {
      source  = "github.com/aziontech/azion"
      version = "0.1.0"
    }
  }
}

# # ---------------------- RESOURCES ----------------------

resource "azion_edge_application_main_setting" "testfunc" {
  edge_application = {
    name : "Terraform Main Settings test-func"
    supported_ciphers : "all"
    delivery_protocol : "http"
    http_port : [80]
    https_port : [443]
    minimum_tls_version : ""
    debug_rules : false
    edge_firewall : false
    edge_functions : false
    image_optimization : false
    http3 : false
    application_acceleration : false
    l2_caching : false
    load_balancer : false
    raw_logs : true
    device_detection : false
    web_application_firewall : false
    raw_logs : false
  }
}

resource "azion_edge_application_origin" "testfunc" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  origin = {
    name        = "Terraform Edge App Origin test-func"
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
    azion_edge_application_main_setting.testfunc
  ]
}

resource "azion_edge_application_cache_setting" "testfunc" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  cache_settings = {
    name                               = "Terraform Cache Setting test-func"
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
    azion_edge_application_main_setting.testfunc,
    azion_edge_application_origin.testfunc
  ]
}

# resource "azion_edge_application_rule_engine" "testfunc" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     name        = "Terraform Rule Engine test-func"
#     phase       = "request"
#     description = "My rule engine"
#     behaviors = [
#       {
#         name   = "deliver"
#         target = ""
#       }
#     ]
#     criteria = [
#       {
#         entries = [
#           {
#             variable    = "$${uri}"
#             operator    = "is_equal"
#             conditional = "if"
#             input_value = "/"
#           }
#         ]
#       }
#     ]
#   }
# }

resource "azion_edge_function" "testfunc" {
  edge_function = {
    name           = "Terraform Edge Function test-func"
    code           = file("${path.module}/mock_files/dummy_script.txt")
    language       = "javascript"
    initiator_type = "edge_application"
    json_args = jsonencode(
      { "key" = "Value",
        "key" = "example"
    })
    active = true
  }
}

# resource "azion_edge_application_edge_functions_instance" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     name = "Terraform Edge Functions Instance test-func"
#     "edge_function_id" : 12345,
#     "args" : jsonencode(
#       { "key"     = "Value",
#         "Example" = "example"
#     })
#   }
# }

resource "azion_domain" "testfunc" {
  domain = {
    cnames : [
      "www.testterraform3x4mpl3.com"
    ]
    name                   = "Terraform domain test-func"
    digital_certificate_id = null
    cname_access_only      = false
    edge_application_id    = azion_edge_application_main_setting.testfunc.edge_application.application_id
    is_active              = true
  }
}

resource "azion_edge_firewall_main_setting" "testfunc" {
  results = {
    name                       = "EdgeFirewall test-func"
    is_active                  = true
    edge_functions_enabled     = true
    network_protection_enabled = true
    waf_enabled                = true
    domains                    = []
  }
}

resource "azion_digital_certificate" "testfunc" {
  certificate_result = {
    name                = "Terraform Digital Certificate test-func"
    certificate_content = file("${path.module}/mock_files/dummy_certificate.pem")
    private_key         = file("${path.module}/mock_files/dummy_private_key.pem")
  }
}

resource "azion_intelligent_dns_zone" "testfunc" {
  zone = {
    domain : "terraformtestfunc.qa",
    is_active : true,
    name : "example"
  }
}

resource "azion_intelligent_dns_dnssec" "testfunc" {
  zone_id = azion_intelligent_dns_zone.testfunc.zone.id
  dns_sec = {
    is_enabled = true
  }
  depends_on = [ azion_intelligent_dns_zone.testfunc ]
}

resource "azion_intelligent_dns_record" "testfunc" {
  zone_id = azion_intelligent_dns_zone.testfunc.zone.id
  record = {
    record_type = "A"
    entry       = "site"
    answers_list = [
      "8.8.8.8"
    ]
    policy      = "weighted"
    weight      = 50
    description = "This is a description"
    ttl         = 20
  }
}

# ---------------------- DATA SOURCES ----------------------

data "azion_edge_applications_main_settings" "example" {
  page      = 1
  page_size = 2
}

data "azion_edge_application_main_settings" "example" {
  id = azion_edge_application_main_setting.testfunc.edge_application.application_id
}

data "azion_edge_applications_origins" "example" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
}

data "azion_edge_application_origin" "example" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  origin = {
    origin_key = azion_edge_application_origin.testfunc.origin.origin_key
  }
}

data "azion_edge_application_cache_settings" "example" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
}

data "azion_edge_application_cache_setting" "example" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  results = {
    cache_setting_id = azion_edge_application_cache_setting.testfunc.cache_settings.cache_setting_id
  }
}

data "azion_edge_application_rules_engine" "example" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  results = [{
    phase = "request"
  }]
}

# data "azion_edge_application_rule_engine" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     phase = "request"
#     id    = 123456
#   }
# }

data "azion_edge_functions" "example" {
}

data "azion_edge_function" "example" {
  id = azion_edge_function.testfunc.edge_function.function_id
}

# data "azion_edge_application_edge_functions_instance" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
# }

# data "azion_edge_application_edge_function_instance" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     id = 123456
#   }
# }

data "azion_edge_firewall_main_settings" "example" {
  page      = 1
  page_size = 2
}

data "azion_edge_firewall_main_setting" "example" {
  edge_firewall_id = azion_edge_firewall_main_setting.testfunc.results.id
}

data "azion_digital_certificates" "example" {
}

data "azion_digital_certificate" "example" {
  certificate_id = azion_digital_certificate.testfunc.certificate_result.certificate_id
}

data "azion_domains" "example" {
}

data "azion_domain" "example" {
  id = azion_domain.testfunc.domain.id
}

data "azion_intelligent_dns_zones" "examples" {}

data "azion_intelligent_dns_zone" "examples" {
  id = azion_intelligent_dns_zone.testfunc.zone.id
}

data "azion_intelligent_dns_dnssec" "examples" {
  zone_id = azion_intelligent_dns_zone.testfunc.zone.id
}

data "azion_intelligent_dns_records" "examples" {
  zone_id = azion_intelligent_dns_zone.testfunc.zone.id
}
