resource "azion_edge_application_cache_setting" "example" {
  edge_application_id = <edge_application_id>
  cache_settings = {
    name = "Terraform New Cache Setting Example"
    browser_cache_settings = "override"
    browser_cache_settings_maximum_ttl = 0
    cdn_cache_settings = "override"
    cdn_cache_settings_maximum_ttl = 13660
    cache_by_query_string = "ignore"
    cache_by_cookies = "ignore"
  }
}