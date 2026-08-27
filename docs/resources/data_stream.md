---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_stream"
description: |-
  Provides a Data Stream resource for shipping logs to external endpoints.
---

# azion_data_stream

Creates a Data Stream. A stream pipes logs from one or more **inputs** through optional **transforms** into one or more **outputs** (endpoints).

Both `transform` and `outputs` entries are polymorphic: each entry carries a `type` and exactly one matching `*_attributes` block.

## Example Usage

### Application logs to an HTTP endpoint

```terraform
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
```

### Sampled WAF events to an S3-compatible bucket

```terraform
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
```

### Workload filtering, template rendering and multiple outputs

```terraform
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
          workloads = [1234, 5678]
        }
      },
      {
        type = "render_template"
        render_template_attributes = {
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
```

## Import

```sh
terraform import azion_data_stream.example 1234
```

## Argument Reference

* `data_stream` - (Required) The data stream configuration.
  * `name` - (Required) Name of the data stream.
  * `active` - (Optional) Status of the data stream. Computed when omitted.
  * `inputs` - (Required) List of inputs feeding the stream.
    * `type` - (Required) Type of the input. Supported value: `raw_logs`.
    * `attributes` - (Required) Attributes of the input.
      * `data_source` - (Required) Source of the logs. One of `http`, `waf`, `workloads`, `functions_console`, `cells_console`, `activity_history`, `rtm_activity`.
  * `transform` - (Optional) Ordered list of transforms applied to the records. Each entry requires the `*_attributes` block matching its `type`. If the list is present, it must contain either a `sampling` entry or a `filter_workloads` entry — see the notes below.
    * `type` - (Required) One of `sampling`, `filter_workloads`, `render_template`.
    * `sampling_attributes` - (Optional) Required when `type` is `sampling`.
      * `rate` - (Required) Percentage of records to keep.
    * `filter_workloads_attributes` - (Optional) Required when `type` is `filter_workloads`.
      * `workloads` - (Required) List of workload identifiers whose logs are kept.
    * `render_template_attributes` - (Optional) Required when `type` is `render_template`.
      * `template` - (Required) Identifier of the Data Stream template used to render the payload. Create one with [`azion_data_stream_template`](data_stream_template.md) and reference `azion_data_stream_template.example.template.id`, or look up a built-in template's ID with the [`azion_data_stream_templates`](../data-sources/data_stream_templates.md) data source.
  * `outputs` - (Required) List of endpoints the records are delivered to. Each entry requires the `*_attributes` block matching its `type`.
    * `type` - (Required) One of `standard`, `kafka`, `s3`, `big_query`, `elasticsearch`, `splunk`, `aws_kinesis_firehose`, `datadog`, `qradar`, `azure_monitor`, `azure_blob_storage`.
    * `standard_attributes` - (Optional) Required when `type` is `standard`.
      * `url` - (Required) Destination URL.
      * `headers` - (Required, Sensitive) Map of headers sent with each request.
      * `log_line_separator` - (Optional) Separator inserted between records in the payload.
      * `payload_format` - (Optional) Format of the payload, for example `$dataset`.
      * `max_size` - (Optional) Maximum payload size, in bytes.
    * `kafka_attributes` - (Optional) Required when `type` is `kafka`.
      * `bootstrap_servers` - (Required) Comma-separated list of Kafka bootstrap servers.
      * `kafka_topic` - (Required) Kafka topic the records are published to.
      * `use_tls` - (Required) Whether TLS is used to connect to the brokers.
    * `s3_attributes` - (Optional) Required when `type` is `s3`.
      * `access_key` - (Required, Sensitive) Access key of the S3-compatible service.
      * `secret_key` - (Required, Sensitive) Secret key of the S3-compatible service.
      * `region` - (Required) Region of the bucket.
      * `object_key_prefix` - (Optional) Prefix prepended to the object keys.
      * `bucket_name` - (Required) Name of the bucket.
      * `content_type` - (Required) One of `plain/text`, `application/gzip`.
      * `host_url` - (Required) Host URL of the S3-compatible service.
    * `big_query_attributes` - (Optional) Required when `type` is `big_query`.
      * `dataset_id` - (Required) BigQuery dataset identifier.
      * `project_id` - (Required) Google Cloud project identifier.
      * `table_id` - (Required) BigQuery table identifier.
      * `service_account_key` - (Required, Sensitive) Service account key, as a JSON string.
    * `elasticsearch_attributes` - (Optional) Required when `type` is `elasticsearch`.
      * `url` - (Required) Elasticsearch endpoint URL.
      * `api_key` - (Required, Sensitive) Elasticsearch API key.
    * `splunk_attributes` - (Optional) Required when `type` is `splunk`.
      * `url` - (Required) Splunk endpoint URL.
      * `api_key` - (Required, Sensitive) Splunk API key.
    * `aws_kinesis_firehose_attributes` - (Optional) Required when `type` is `aws_kinesis_firehose`.
      * `access_key` - (Required, Sensitive) AWS access key.
      * `stream_name` - (Required) Name of the Kinesis Data Firehose delivery stream.
      * `region` - (Required) AWS region of the delivery stream.
      * `secret_key` - (Required, Sensitive) AWS secret key.
    * `datadog_attributes` - (Optional) Required when `type` is `datadog`.
      * `url` - (Required) Datadog endpoint URL.
      * `api_key` - (Required, Sensitive) Datadog API key.
    * `qradar_attributes` - (Optional) Required when `type` is `qradar`.
      * `url` - (Required) IBM QRadar HTTP receiver URL.
    * `azure_monitor_attributes` - (Optional) Required when `type` is `azure_monitor`.
      * `log_type` - (Required) Record type of the data being submitted.
      * `shared_key` - (Required, Sensitive) Primary or secondary key of the workspace.
      * `time_generated_field` - (Optional) Name of the field used as the event timestamp.
      * `workspace_id` - (Required) Log Analytics workspace identifier.
    * `azure_blob_storage_attributes` - (Optional) Required when `type` is `azure_blob_storage`.
      * `storage_account` - (Required) Name of the storage account.
      * `container_name` - (Required) Name of the blob container.
      * `blob_sas_token` - (Required, Sensitive) Shared access signature token of the container.

## Attribute Reference

* `id` - The data stream identifier, as a string.
* `last_updated` - Timestamp of the last Terraform update of the resource.
* `schema_version` - Schema version of the resource.
* `data_stream`
  * `id` - The data stream identifier.
  * `last_editor` - The last editor of the data stream.
  * `created_at` - The creation timestamp of the data stream.
  * `last_modified` - Last modified timestamp of the data stream.
  * `product_version` - Product version of the data stream.

## Notes

* **A non-empty `transform` list must include either `sampling` or `filter_workloads`.** The API rejects a `transform` list holding only `render_template` with:

  ```
  400 - code 32002 "Workloads Must Be Provided"
  "If sampling is disabled, workloads must be provided."
  ```

  Omitting `transform` entirely is fine — the constraint only applies once the list is present. To render a template without actually sampling or filtering, pair it with a no-op `sampling` entry at `rate = 100`.

* Updates use `PUT`, not `PATCH`: the partial-update payload cannot carry `outputs`, so a `PATCH` could never change the endpoints. Every apply therefore sends the full stream body.
* Credentials (`access_key`, `secret_key`, `api_key`, `shared_key`, `blob_sas_token`, `service_account_key`, `headers`) are returned masked by the API. The provider keeps the configured value in state so masked echoes don't show up as perpetual drift. On `terraform import` the masked value from the API is stored instead, so re-adding the real credential to the configuration will produce one diff.
