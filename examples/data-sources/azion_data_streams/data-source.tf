# List every data stream in the account.
data "azion_data_streams" "example" {
}

output "azion_data_streams" {
  value = data.azion_data_streams.example
}

output "azion_data_streams_names" {
  value = [for stream in data.azion_data_streams.example.results : stream.name]
}
