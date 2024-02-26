terraform {
  required_providers {
    azion = {
      source  = "github.com/aziontech/azion"
      version = "0.1.0"
    }
  }
}

# ---------------------- VARIABLES ----------------------
variable "edge_functions_module" {
  type    = bool
  default = false
}

# ---------------------- RESOURCES ----------------------

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
    edge_functions : var.edge_functions_module
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


resource "azion_edge_application_rule_engine" "testfunc" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  results = {
    name         = "Default Rule"
    phase        = "request"
    description  = ""
    behaviors    = [
      {
        name = "set_origin"
        target_object : {
            target = azion_edge_application_origin.testfunc.id
        }
      },
      {
        name = "capture_match_groups",
        target_object = { 
                "captured_array": "Terraform",
                "subject": "$${uri}",
                "regex": "1101"
        }
      },
      {
        name = "set_cache_policy"
        target_object : {
            target = azion_edge_application_cache_setting.testfunc.id
        }
      },
    ]
    criteria     = [
      {
        entries = [
          {
            variable    = "$${uri}"
            operator    = "starts_with"
            conditional = "if"
            input_value = "/"
          },
        ]
      }
    ]
  }
    depends_on = [
    azion_edge_application_main_setting.testfunc,
    azion_edge_application_origin.testfunc,
    azion_edge_application_cache_setting.testfunc
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
#         name = "add_request_header",
#         target_object = {
#           target = "X-Cache: 100"
#         }
#       },
#       { 
#         name   = "filter_request_header",
#         target_object = {
#           target = "X-Cache"
#         }
#       }
#     ]
#     criteria = [
#       {
#         entries = [
#           {
#             variable= "$${uri}"
#             operator= "is_equal"
#             conditional= "if"
#             input_value= "/"
#           }
#         ]
#       }
#     ]
#   }
#   depends_on = [
#     azion_edge_application_main_setting.testfunc
#   ]
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

resource "azion_edge_function" "testfunc2firewall" {
  edge_function = {
    name           = "Terraform Edge Function 2 Firewall test-func"
    code           = trimspace(file("${path.module}/mock_files/dummy_script2firewall.txt"))
    language       = "javascript"
    initiator_type = "edge_firewall"
    json_args = jsonencode(
      { "key" = "Value",
        "key" = "example"
    })
    active = true
  }
}

resource "null_resource" "update_edge_functions" {
  depends_on = [azion_edge_application_main_setting.testfunc]

  provisioner "local-exec" {
    command = "sleep 20 && terraform apply -auto-approve -target='azion_edge_application_main_setting.testfunc' -var 'edge_functions_module=true'"
  }
}

resource "azion_edge_application_edge_functions_instance" "testfunc" {
  depends_on = [null_resource.update_edge_functions]

  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  results = {
    name = "Terraform Edge Functions Instance test-func"
    "edge_function_id" : azion_edge_function.testfunc.edge_function.function_id,
    "args" : jsonencode(
      { "key"     = "Value",
        "Example" = "example"
    })
  }
}

resource "azion_domain" "testfunc" {
  domain = {
    cnames : [
      "www.terraformtest-func.qa"
    ]
    name                   = "Terraform domain test-func"
    digital_certificate_id = null
    cname_access_only      = false
    edge_application_id    = azion_edge_application_main_setting.testfunc.edge_application.application_id
    is_active              = true
  }
  depends_on = [
    azion_edge_application_main_setting.testfunc
  ]
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

resource "azion_edge_firewall_edge_functions_instance" "testfunc" {
  edge_firewall_id = azion_edge_firewall_main_setting.testfunc.results.id
  results = {
    name = "Terraform Test 1"
    "edge_function_id" : azion_edge_function.testfunc2firewall.edge_function.function_id
    "args" : jsonencode(
    { a = "b" })
  }
  depends_on = [
    azion_edge_firewall_main_setting.testfunc,
    azion_edge_function.testfunc2firewall
  ]
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
    domain : "terraformtest-func.qa",
    is_active : true,
    name : "example"
  }
}

# resource "azion_intelligent_dns_dnssec" "testfunc" {
#   zone_id = azion_intelligent_dns_zone.testfunc.zone.id
#   dns_sec = {
#     is_enabled = true
#   }
#   depends_on = [azion_intelligent_dns_zone.testfunc]
# }

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
  depends_on = [azion_intelligent_dns_zone.testfunc]
}

resource "azion_network_list" "exampleOne" {
  results = {
    name      = "NetworkList Terraform test-func Countries"
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
    name      = "NetworkList Terraform test-func ip_cidr"
    list_type = "ip_cidr"
    items_values_str = [
      "192.168.0.1",
      "192.168.0.2",
      "192.168.0.3"
    ]
  }
}

resource "azion_waf_rule_set" "testfunc" {
  result = {
    name                              = "Terraform WAF test-func",
    mode                              = "counting",
    active                            = true,
    sql_injection                     = true,
    sql_injection_sensitivity         = "medium",
    remote_file_inclusion             = true,
    remote_file_inclusion_sensitivity = "medium",
    directory_traversal               = true,
    directory_traversal_sensitivity   = "medium",
    cross_site_scripting              = true,
    cross_site_scripting_sensitivity  = "highest",
    evading_tricks                    = true,
    evading_tricks_sensitivity        = "medium",
    file_upload                       = true,
    file_upload_sensitivity           = "medium",
    unwanted_access                   = true,
    unwanted_access_sensitivity       = "high",
    identified_attack                 = false,
    identified_attack_sensitivity     = "medium",
    bypass_addresses                  = ["192.168.1.67", "192.168.1.64", "192.168.1.65", "192.168.1.63", "192.168.1.66"]
  }
}

resource "azion_environment_variable" "testfunc" {
  result = {
    key    = "key-test Terraform test-func"
    value  = "key-test Terraform test-func"
    secret = false
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
#     id    = azion_edge_application_rule_engine.testfunc.results.id
#   }
# }

data "azion_edge_functions" "example" {
}

data "azion_edge_function" "example" {
  id = azion_edge_function.testfunc.edge_function.function_id
}

data "azion_edge_application_edge_functions_instance" "example" {
  depends_on = [ azion_edge_application_edge_functions_instance.testfunc ]
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
}

data "azion_edge_application_edge_function_instance" "example" {
  edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
  results = {
    id = azion_edge_application_edge_functions_instance.testfunc.results.id
  }
}

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

data "azion_network_lists" "example" {
  page = 1
}

data "azion_network_list" "exampleOne" {
  network_list_id = azion_network_list.exampleOne.id
}

data "azion_network_list" "exampleTwo" {
  network_list_id = azion_network_list.exampleTwo.id
}

data "azion_environment_variables" "example" {
}

data "azion_environment_variable" "example" {
  result = {
    uuid = azion_environment_variable.testfunc.result.uuid
  }
}

data "azion_waf_rule_sets" "example" {
  page      = 1
  page_size = 10
}

data "azion_waf_rule_set" "example" {
  result = {
    waf_id = azion_waf_rule_set.testfunc.result.waf_id
  }
}

data "azion_waf_domains" "example" {
  page      = 1
  page_size = 10
  waf_id    = azion_waf_rule_set.testfunc.result.waf_id
}

data "azion_edge_firewall_edge_functions_instance" "example" {
  edge_firewall_id = azion_edge_firewall_main_setting.testfunc.results.id
  page             = 1
  page_size        = 10
}

data "azion_edge_firewall_edge_function_instance" "example" {
  edge_firewall_id = azion_edge_firewall_main_setting.testfunc.results.id
  results = {
    edge_function_instance_id = azion_edge_firewall_edge_functions_instance.testfunc.results.id
  }
}