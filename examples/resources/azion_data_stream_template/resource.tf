# =====================================================
# DATA STREAM TEMPLATE RESOURCES
# =====================================================

# A template defines the payload shape applied by the `render_template`
# transform of a data stream. `data_set` is stored verbatim by the API.
resource "azion_data_stream_template" "example" {
  template = {
    name   = "My HTTP Events Template"
    active = true
    data_set = jsonencode({
      time        = "$time_iso8601"
      host        = "$host"
      remote_addr = "$remote_addr"
      request_uri = "$request_uri"
      status      = "$status"
      bytes_sent  = "$bytes_sent"
    })
  }
}

# A template referenced by a stream. The `render_template` transform takes the
# template's numeric ID, so it can be wired up without hardcoding it.
resource "azion_data_stream" "with_template" {
  data_stream = {
    name   = "Rendered logs to HTTP endpoint"
    active = true

    inputs = [
      {
        type = "raw_logs"
        attributes = {
          data_source = "http"
        }
      }
    ]

    transform = [
      # A no-op sampling entry: `render_template` cannot be the only transform,
      # the API requires either sampling or filter_workloads alongside it.
      {
        type = "sampling"
        sampling_attributes = {
          rate = 100
        }
      },
      {
        type = "render_template"
        render_template_attributes = {
          template = azion_data_stream_template.example.template.id
        }
      }
    ]

    outputs = [
      {
        type = "standard"
        standard_attributes = {
          url = "https://logs.example.com/ingest"
          headers = {
            "Authorization" = "Bearer my-token"
          }
        }
      }
    ]
  }
}

# Templates do not have to hold JSON. A flat, space-separated layout works too.
resource "azion_data_stream_template" "plain_text" {
  template = {
    name     = "My Plain Text Template"
    active   = true
    data_set = "$time_iso8601 $remote_addr $request_method $request_uri $status"
  }
}
