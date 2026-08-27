# Read a single data stream template by its identifier.
data "azion_data_stream_template" "example" {
  id = "1234"
}

output "azion_data_stream_template" {
  value = data.azion_data_stream_template.example
}

output "azion_data_stream_template_data_set" {
  value = data.azion_data_stream_template.example.data.data_set
}
