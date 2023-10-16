data "azion_waf_domains" "example" {
  page      = 1
  page_size = 10
  waf_id    = 5793
}