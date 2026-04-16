data "azion_application_function_instances" "example" {
  application_id = 1234567890
  page           = 1
  page_size      = 10
}