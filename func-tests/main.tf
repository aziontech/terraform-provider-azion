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

# ---------------------- LOCALS ----------------------
locals {
  timestamp   = formatdate("YYYY-MM-DD-hhmm", timestamp())
  name_suffix = "test-func-${local.timestamp}"
}

# ---------------------- RESOURCES ----------------------

# Temporarily commented out due to API/SDK cache field mismatch
# resource "azion_edge_application_main_setting" "testfunc" {
#   edge_application = {
#     name   = "Terraform Main Settings ${local.name_suffix}"
#     active = true
#     debug  = true
#     modules = {
#       edge_cache = {
#         enabled = true
#       }
#       functions = {
#         enabled = var.edge_functions_module
#       }
#       application_accelerator = {
#         enabled = false
#       }
#       image_processor = {
#         enabled = false
#       }
#     }
#   }
# }

# resource "azion_edge_application_origin" "testfunc" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   origin = {
#     name        = "Terraform Edge App Origin ${local.name_suffix}"
#     origin_type = "single_origin"
#     addresses : [
#       {
#         "address" : "httpExample.org"
#       }
#     ],
#     origin_protocol_policy : "https",
#     host_header : "$${host}",
#   }
#   depends_on = [
#     azion_edge_application_main_setting.testfunc
#   ]
# }

# resource "azion_application_cache_setting" "testfunc" {
#   application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   cache_setting = {
#     name = "Terraform Cache Setting ${local.name_suffix}"
#     browser_cache = {
#       behavior = "override"
#       max_age  = 20
#     }
#     modules = {
#       cache = {
#         behavior = "override"
#         max_age  = 60
#         stale_cache = {
#           enabled = true
#         }
#       }
#     }
#   }
#   depends_on = [
#     azion_edge_application_main_setting.testfunc,
#     azion_edge_application_origin.testfunc
#   ]
# }

# resource "azion_application_rule_engine" "testfunc" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     name        = "Default Rule"
#     phase       = "request"
#     description = ""
#     behaviors = [
#       {
#         type = "deliver"
#       }
#     ]
#     criteria = [
#       {
#         entries = [
#           {
#             variable    = "$${uri}"
#             operator    = "starts_with"
#             conditional = "if"
#             argument    = "/"
#           },
#         ]
#       }
#     ]
#   }
#   depends_on = [
#     azion_edge_application_main_setting.testfunc,
#     azion_edge_application_origin.testfunc,
#     azion_application_cache_setting.testfunc
#   ]
# }
# 
# resource "azion_application_rule_engine" "testfunc2" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     name        = "Terraform Rule Engine test-func"
#     phase       = "request"
#     description = "My rule engine"
#     behaviors = [
#       {
#         type = "add_request_header"
#         attributes = {
#           value = "X-Cache: 100"
#         }
#       },
#       {
#         type = "filter_request_header"
#         attributes = {
#           value = "X-Cache"
#         }
#       }
#     ]
#     criteria = [
#       {
#         entries = [
#           {
#             variable    = "$${uri}"
#             operator    = "is_equal"
#             conditional = "if"
#             argument    = "/"
#           }
#         ]
#       }
#     ]
#   }
#   depends_on = [
#     azion_edge_application_main_setting.testfunc
#   ]
# }

resource "azion_function" "testfunc" {
  function = {
    name                  = "Terraform Function ${local.name_suffix}"
    code                  = trimspace(file("${path.module}/mock_files/dummy_script.txt"))
    active                = true
    default_args          = jsonencode({ "key" = "Value" })
    execution_environment = "default"
    runtime               = "nodejs20.x"
  }
}

resource "azion_function" "testfunc2firewall" {
  function = {
    name                  = "Terraform Function 2 Firewall ${local.name_suffix}"
    code                  = trimspace(file("${path.module}/mock_files/dummy_script2firewall.txt"))
    active                = true
    default_args          = jsonencode({ "key" = "Value" })
    execution_environment = "firewall"
    runtime               = "nodejs20.x"
  }
}

# resource "null_resource" "update_edge_functions" {
#   depends_on = [azion_edge_application_main_setting.testfunc]
# 
#   provisioner "local-exec" {
#     command = "sleep 30 && terraform apply -auto-approve -target='azion_edge_application_main_setting.testfunc' -var 'edge_functions_module=true'"
#   }
# }

# resource "azion_edge_application_edge_functions_instance" "testfunc" {
#   depends_on = [null_resource.update_edge_functions]
# 
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     name = "Terraform Edge Functions Instance test-func"
#     "edge_function_id" : azion_edge_function.testfunc.edge_function.function_id,
#     "args" : jsonencode(
#       { "key"     = "Value",
#         "Example" = "example"
#     })
#   }
# }

# resource "azion_domain" "testfunc" {
#   domain = {
#     cnames : [
#       "www.terraformtest-func-${local.timestamp}.qa"
#     ]
#     name                   = "Terraform domain ${local.name_suffix}"
#     digital_certificate_id = null
#     cname_access_only      = false
#     edge_application_id    = azion_edge_application_main_setting.testfunc.edge_application.application_id
#     is_active              = true
#   }
#   depends_on = [
#     azion_edge_application_main_setting.testfunc
#   ]
# }

# resource "azion_edge_firewall_main_setting" "testfunc" {
#   data = {
#     name   = "EdgeFirewall ${local.name_suffix}"
#     active = true
#     debug  = false
#     
#     modules = {
#       functions = {
#         enabled = true
#       }
#       
#       network_protection = {
#         enabled = true
#       }
#       
#       waf = {
#         enabled = true
#       }
#     }
#   }
# }

# resource "azion_edge_firewall_edge_functions_instance" "testfunc" {
#   edge_firewall_id = azion_edge_firewall_main_setting.testfunc.data.id
#   data = {
#     name     = "Terraform Test 1"
#     function = azion_edge_function.testfunc2firewall.edge_function.id
#     args = jsonencode({
#       a = "b"
#     })
#   }
#   depends_on = [
#     azion_edge_firewall_main_setting.testfunc,
#     azion_edge_function.testfunc2firewall
#   ]
# }

# Removed due to 401 Unauthorized error
# resource "azion_digital_certificate" "testfunc" {
#   certificate_result = {
#     name                = "Terraform Digital Certificate ${local.name_suffix}"
#     certificate_content = file("${path.module}/mock_files/dummy_certificate.pem")
#     private_key         = file("${path.module}/mock_files/dummy_private_key.pem")
#   }
# }

resource "azion_intelligent_dns_zone" "testfunc" {
  zone = {
    domain : "terraformtest-func-${local.timestamp}.qa",
    active : true,
    name : "example"
  }
}

# # resource "azion_intelligent_dns_dnssec" "testfunc" {
# #   zone_id = azion_intelligent_dns_zone.testfunc.zone.id
# #   dns_sec = {
# #     is_enabled = true
# #   }
# #   depends_on = [azion_intelligent_dns_zone.testfunc]
# # }

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
    name = "NetworkList Terraform ${local.name_suffix} Countries"
    type = "countries"
    items = [
      "BR",
      "US",
      "AG"
    ]
  }
}

resource "azion_network_list" "exampleTwo" {
  results = {
    name = "NetworkList Terraform ${local.name_suffix} ip_cidr"
    type = "ip_cidr"
    items = [
      "192.168.0.1",
      "192.168.0.2",
      "192.168.0.3"
    ]
  }
}

# Removed due to 401 Unauthorized error
# resource "azion_waf_rule_set" "testfunc" {
#   result = {
#     name                              = "Terraform WAF ${local.name_suffix}",
#     mode                              = "counting",
#     active                            = true,
#     sql_injection                     = true,
#     sql_injection_sensitivity         = "medium",
#     remote_file_inclusion             = true,
#     remote_file_inclusion_sensitivity = "medium",
#     directory_traversal               = true,
#     directory_traversal_sensitivity   = "medium",
#     cross_site_scripting              = true,
#     cross_site_scripting_sensitivity  = "highest",
#     evading_tricks                    = true,
#     evading_tricks_sensitivity        = "medium",
#     file_upload                       = true,
#     file_upload_sensitivity           = "medium",
#     unwanted_access                   = true,
#     unwanted_access_sensitivity       = "high",
#     identified_attack                 = false,
#     identified_attack_sensitivity     = "medium",
#     bypass_addresses                  = ["192.168.1.67", "192.168.1.64", "192.168.1.65", "192.168.1.63", "192.168.1.66"]
#   }
# }

# resource "azion_environment_variable" "testfunc" {
#   result = {
#     key    = "key-test Terraform test-func"
#     value  = "key-test Terraform test-func"
#     secret = false
#   }
# }

# ---------------------- DATA SOURCES ----------------------

# Temporarily commented out due to API/SDK mismatch (cache vs edge_cache field)
# data "azion_edge_applications_main_settings" "example" {
#   page      = 1
#   page_size = 2
# }

# data "azion_edge_application_main_settings" "example" {
#   id = azion_edge_application_main_setting.testfunc.edge_application.application_id
# }

# data "azion_edge_applications_origins" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
# }

# data "azion_edge_application_origin" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   origin = {
#     origin_key = azion_edge_application_origin.testfunc.origin.origin_key
#   }
# }

# data "azion_application_cache_settings" "example" {
#   application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
# }

# data "azion_application_cache_setting" "example" {
#   application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     id = azion_application_cache_setting.testfunc.cache_setting.id
#   }
# }

# data "azion_application_rules_engine" "example" {
#   application_id = azion_application_main_setting.testfunc.application.application_id
#   results = [{
#     phase = "request"
#   }]
# }

# # data "azion_application_rule_engine" "example" {
# #   application_id = azion_application_main_setting.testfunc.application.application_id
# #   results = {
# #     phase = "request"
# #     id    = azion_application_rule_engine.testfunc.results.id
# #   }
# # }
# 
data "azion_functions" "example" {
}

# data "azion_function" "example" {
#   id = azion_function.testfunc2firewall.function.id
# }

# data "azion_edge_application_edge_functions_instance" "example" {
#   depends_on          = [azion_edge_application_edge_functions_instance.testfunc]
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
# }

# data "azion_edge_application_edge_function_instance" "example" {
#   edge_application_id = azion_edge_application_main_setting.testfunc.edge_application.application_id
#   results = {
#     id = azion_edge_application_edge_functions_instance.testfunc.results.id
#   }
# }

# data "azion_edge_firewall_main_settings" "example" {
#   page      = 1
#   page_size = 2
# }

# data "azion_edge_firewall_main_setting" "example" {
#   edge_firewall_id = azion_edge_firewall_main_setting.testfunc.data.id
# }

data "azion_digital_certificates" "example" {
}

# Removed due to dependency on azion_digital_certificate.testfunc which has 401 Unauthorized error
# data "azion_digital_certificate" "example" {
#   certificate_id = azion_digital_certificate.testfunc.certificate_result.certificate_id
# }

data "azion_domains" "example" {
}

# data "azion_domain" "example" {
#   id = azion_domain.testfunc.domain.id
# }

# Removed due to 401 Unauthorized error
# data "azion_intelligent_dns_zones" "examples" {}

data "azion_intelligent_dns_zone" "examples" {
  id = azion_intelligent_dns_zone.testfunc.zone.id
}

# Removed due to 500 Internal Server Error
# data "azion_intelligent_dns_dnssec" "examples" {
#   zone_id = azion_intelligent_dns_zone.testfunc.zone.id
# }

data "azion_intelligent_dns_records" "examples" {
  zone_id = azion_intelligent_dns_zone.testfunc.zone.id
}

data "azion_network_lists" "example" {
  page = 1
}

data "azion_network_list" "exampleOne" {
  id = azion_network_list.exampleOne.results.id
}

data "azion_network_list" "exampleTwo" {
  id = azion_network_list.exampleTwo.results.id
}

# Removed due to 401 Unauthorized error
# data "azion_environment_variables" "example" {
# }

# data "azion_environment_variable" "example" {
#   result = {
#     uuid = azion_environment_variable.testfunc.result.uuid
#   }
# }

# Removed due to dependency on azion_waf_rule_set.testfunc which has 401 Unauthorized error
# data "azion_waf_rule_sets" "example" {
#   page      = 1
#   page_size = 10
# }

# data "azion_waf_rule_set" "example" {
#   result = {
#     waf_id = azion_waf_rule_set.testfunc.result.waf_id
#   }
# }

# data "azion_waf_domains" "example" {
#   page      = 1
#   page_size = 10
#   waf_id    = azion_waf_rule_set.testfunc.result.waf_id
# }

# data "azion_edge_firewall_edge_functions_instance" "example" {
#   edge_firewall_id = azion_edge_firewall_main_setting.testfunc.data.id
#   page             = 1
#   page_size        = 10
# }

# data "azion_edge_firewall_edge_function_instance" "example" {
#   edge_firewall_id = azion_edge_firewall_main_setting.testfunc.data.id
# }

