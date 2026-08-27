# =====================================================
# DATA STREAM RESOURCES
# =====================================================

# Minimal stream: application logs delivered to an HTTP endpoint.
resource "azion_data_stream" "http_post" {
  data_stream = {
    name   = "Applications to HTTP endpoint"
    active = true

    inputs = [
      {
        type = "raw_logs"
        attributes = {
          data_source = "http"
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

# Sampled WAF events delivered to an S3-compatible bucket.
resource "azion_data_stream" "waf_to_s3" {
  data_stream = {
    name   = "WAF events to S3"
    active = true

    inputs = [
      {
        type = "raw_logs"
        attributes = {
          data_source = "waf"
        }
      }
    ]

    transform = [
      {
        type = "sampling"
        sampling_attributes = {
          rate = 50
        }
      }
    ]

    outputs = [
      {
        type = "s3"
        s3_attributes = {
          host_url          = "https://s3.us-east-1.amazonaws.com"
          bucket_name       = "my-log-bucket"
          region            = "us-east-1"
          object_key_prefix = "azion/waf"
          content_type      = "application/gzip"
          access_key        = "AKIAIOSFODNN7EXAMPLE"
          secret_key        = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
        }
      }
    ]
  }
}

# Workload-filtered logs rendered through a template and fanned out to Kafka
# and Datadog. Both transforms and multiple outputs are supported per stream.
resource "azion_data_stream" "multi_output" {
  data_stream = {
    name   = "Filtered logs to Kafka and Datadog"
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
      {
        type = "filter_workloads"
        filter_workloads_attributes = {
          # Replace with the identifiers of your own workloads.
          workloads = [1234, 5678]
        }
      },
      {
        type = "render_template"
        render_template_attributes = {
          # Replace with the identifier of an existing Data Stream template.
          template = 1
        }
      }
    ]

    outputs = [
      {
        type = "kafka"
        kafka_attributes = {
          bootstrap_servers = "kafka-1.example.com:9092,kafka-2.example.com:9092"
          kafka_topic       = "azion-logs"
          use_tls           = true
        }
      },
      {
        type = "datadog"
        datadog_attributes = {
          url     = "https://http-intake.logs.datadoghq.com/v1/input"
          api_key = "my-datadog-api-key"
        }
      }
    ]
  }
}
