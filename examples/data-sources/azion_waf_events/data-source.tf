data "azion_waf_events" "example" {
  #    page = 1
  #    page_size = 30
  waf_id      = 5793
  domains_ids = [1683058761, 1697207627]
  hour_range  = 6 #hour_rage [1,3,6,9,12,15,18,21,24,27,30....72]
}