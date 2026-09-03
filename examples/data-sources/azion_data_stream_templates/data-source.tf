# List every data stream template available to the account, both Azion's
# built-in templates and the ones created in this account.
data "azion_data_stream_templates" "example" {
}

output "azion_data_stream_templates" {
  value = data.azion_data_stream_templates.example
}

# `custom = false` marks Azion's built-in templates. This is the practical way
# to discover the template ID needed by a stream's `render_template` transform.
output "azion_data_stream_builtin_templates" {
  value = {
    for template in data.azion_data_stream_templates.example.results :
    template.name => template.id
    if template.custom == false
  }
}
