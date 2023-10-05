resource "azion_waf_rule_set" "example" {
  result = {
    name                              = "Terraform WAF",
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