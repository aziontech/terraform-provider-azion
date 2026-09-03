---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_data_stream"
description: |-
  Provides a Data Stream resource for shipping logs to external endpoints.
---

# azion_data_stream

Creates a Data Stream. A stream pipes logs from one or more **inputs** through a **transform** pipeline into one or more **outputs** (endpoints).

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
          data_source = "workloads"
        }
      }
    ]

    # transform is required. rate = 100 keeps every record, so this pipeline only
    # renders the template.
    transform = [
      {
        type = "sampling"
        sampling_attributes = {
          rate = 100
        }
      },
      {
        type = "render_template"
        render_template_attributes = {
          template = 2
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
      },
      {
        type = "render_template"
        render_template_attributes = {
          template = 4
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

### Workload filtering and template rendering

One stream carries one endpoint, so fanning logs out to Kafka *and* Datadog means two streams sharing the same inputs and transforms — see the notes on `outputs` below.

```terraform
resource "azion_data_stream" "filtered_to_kafka" {
  data_stream = {
    name   = "Filtered logs to Kafka"
    active = true

    inputs = [
      {
        type = "raw_logs"
        attributes = {
          data_source = "workloads"
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
  * `inputs` - (Required) Input feeding the stream. Must hold exactly one entry — see the notes below.
    * `type` - (Required) Type of the input. Supported value: `raw_logs`.
    * `attributes` - (Required) Attributes of the input.
      * `data_source` - (Required) Source of the logs. One of `workloads`, `waf`, `functions_console`, `activity_history` — the slugs served by the API's `GET /data_stream/data_sources` endpoint.
  * `transform` - (Required) Transforms applied to the records. Each entry requires the `*_attributes` block matching its `type`. Must hold at least one entry, must include a `render_template` entry, and must include either a `sampling` or a `filter_workloads` entry — see the notes below. The API normalizes the pipeline order, so the order the entries are listed in does not change how they are applied.
    * `type` - (Required) One of `sampling`, `filter_workloads`, `render_template`.
    * `sampling_attributes` - (Optional) Required when `type` is `sampling`.
      * `rate` - (Required) Percentage of records to keep.
    * `filter_workloads_attributes` - (Optional) Required when `type` is `filter_workloads`.
      * `workloads` - (Required) List of workload identifiers whose logs are kept.
    * `render_template_attributes` - (Optional) Required when `type` is `render_template`.
      * `template` - (Required) Identifier of the Data Stream template used to render the payload. Create one with [`azion_data_stream_template`](data_stream_template.md) and reference `azion_data_stream_template.example.template.id`, or look up a built-in template's ID with the [`azion_data_stream_templates`](../data-sources/data_stream_templates.md) data source.
  * `outputs` - (Required) Endpoint the records are delivered to. Requires the `*_attributes` block matching its `type`. Must hold exactly one entry — see the notes below.
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

* **`transform` is required and has three constraints.** The API enforces all of them, and rejects a stream that misses any one:

  | Constraint | Error when violated |
  |---|---|
  | The list must be present | `400 - code 10059 "Required Field"` |
  | It must hold at least one entry | `400 - code 10049 "Min Length List Field"` |
  | It must include a `render_template` entry | `400 - code 32008 "Template Must Be Provided"` |
  | It must include a `sampling` or `filter_workloads` entry | `400 - code 32002 "Workloads Must Be Provided"` |

  So the smallest valid pipeline is a `render_template` entry paired with something that satisfies the last rule. To render a template without actually sampling or filtering, use a no-op `sampling` entry at `rate = 100`:

  ```terraform
  transform = [
    {
      type = "sampling"
      sampling_attributes = {
        rate = 100
      }
    },
    {
      type = "render_template"
      render_template_attributes = {
        template = 2
      }
    }
  ]
  ```

* **The API normalizes the transform order.** It stores the pipeline in its own canonical order no matter which order the entries were submitted in. The provider re-orders the response back onto the configured order so this doesn't surface as `Provider produced inconsistent result after apply`, which also means the order written in the configuration has no effect on the pipeline.

* **Only one `inputs` entry is stored.** Given a longer list the API keeps a single input and discards the rest, without reporting an error. The provider rejects more than one entry at plan time rather than letting the discarded inputs surface as `Provider produced inconsistent result after apply`.

* **`data_source` is validated against a fixed list.** The API silently rewrites an unrecognized slug to a different one instead of returning an error, which would also read as an inconsistent-result failure. The four accepted slugs (`workloads`, `waf`, `functions_console`, `activity_history`) are what `GET /data_stream/data_sources` currently serves; the values `http`, `cells_console` and `rtm_activity` accepted by earlier API versions are gone.

* **Only one `outputs` entry is stored.** Fan-out to several endpoints from a single stream does not currently work. Given a longer list the API either keeps just the last entry, when every entry shares the same `type`, or answers `500 - code 10067 "Internal Server Error"` when the types differ. Both lose the configuration silently or unhelpfully, so the provider rejects more than one entry at plan time. Use one stream per destination endpoint.

* **`log_line_separator` is trimmed by the API.** Surrounding whitespace is stripped, so a literal newline is stored as an empty string. Write a newline separator as the two-character escape `"\\n"` instead — that round-trips unchanged, and is also the API's own default. The provider rejects a value with leading or trailing whitespace at plan time rather than letting the trimmed echo fail as an inconsistent result.

* **An account can hold only one active stream.** Not one per `data_source` — one in total. Activating a stream silently deactivates whichever was active before, and the create/update response still echoes `active: true`, so the provider records `true` while the API stores `false`. The next plan then shows `active = false -> true` forever. Keep `active = true` on a single stream and set the rest to `false`; this cannot be validated at plan time because it depends on account-wide state.

* **`elasticsearch`, `splunk` and `datadog` outputs cannot be read back.** All three serialize as `{url, api_key}`, and the SDK's `oneOf(Output)` decoder matches variants by shape rather than by the `type` discriminator, so any of the three matches all three:

  ```
  data matches more than one schema in oneOf(Output)
  ```

  This bites harder than it looks: a single stream of one of those types anywhere in the account also breaks the [`azion_data_streams`](../data-sources/data_streams.md) data source, since listing decodes every stream. The fix belongs in the OpenAPI spec — the `Output` and `OutputRequest` schemas need `discriminator: {propertyName: type}` so the generator emits a `type`-based decoder — and cannot be worked around in the provider, which never sees the raw body once the SDK fails to decode it.

* Updates use `PUT`, not `PATCH`: the partial-update payload cannot carry `outputs`, so a `PATCH` could never change the endpoints. Every apply therefore sends the full stream body.
* Credentials (`access_key`, `secret_key`, `api_key`, `shared_key`, `blob_sas_token`, `service_account_key`, `headers`) are returned masked by the API. The provider keeps the configured value in state so masked echoes don't show up as perpetual drift. On `terraform import` the masked value from the API is stored instead, so re-adding the real credential to the configuration will produce one diff.
