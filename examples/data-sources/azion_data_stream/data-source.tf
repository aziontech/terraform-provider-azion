# Read a single data stream by its identifier.
data "azion_data_stream" "example" {
  id = "1234"
}

output "azion_data_stream" {
  value = data.azion_data_stream.example
}

# The polymorphic lists come back as JSON strings, so they can be decoded
# when a specific field is needed.
output "azion_data_stream_first_output_type" {
  value = jsondecode(data.azion_data_stream.example.data.outputs)[0].type
}
