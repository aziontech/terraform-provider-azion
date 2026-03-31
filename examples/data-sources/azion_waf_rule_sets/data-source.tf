data "azion_waf_rule_sets" "example" {
  waf_id    = 12345
  page      = 1
  page_size = 10
}