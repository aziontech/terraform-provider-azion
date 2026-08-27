# Data Stream - Streams - Code Generation Guide

This document provides specific guidance for implementing the Data Stream **Streams** resource and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Domain Model](#domain-model)
3. [Data Source Implementation](#data-source-implementation)
4. [Resource Implementation](#resource-implementation)
5. [Schema Definition Patterns](#schema-definition-patterns)
6. [Polymorphic Type Handling](#polymorphic-type-handling)
7. [Error Handling](#error-handling)
8. [Type Conversions](#type-conversions)
9. [Common Issues](#common-issues)
10. [Files](#files)

---

## SDK Selection

Data Stream Streams use the **V4 SDK (`azion-api`)** for the resource and both data sources:

| Implementation | SDK Package | Client Field | Base URL |
|----------------|-------------|--------------|----------|
| Data Stream (Singular Data Source) | `azion-api` (v4) | `api.DataStreamStreamsAPI` | `https://api.azion.com/v4` |
| Data Streams (Plural Data Source) | `azion-api` (v4) | `api.DataStreamStreamsAPI` | `https://api.azion.com/v4` |
| Data Stream (Resource) | `azion-api` (v4) | `api.DataStreamStreamsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `DataStreamRequest` |
| Update Request Type | `DataStreamRequest` (PUT) — see note below |
| Patch Request Type | `PatchedDataStreamRequest` (**cannot carry `outputs`**) |
| Response Type | `DataStreamResponse` with `Data DataStream` |
| List Response Type | `PaginatedDataStreamList` |
| Create Pattern | `.CreateDataStream(ctx).DataStreamRequest(req).Execute()` |
| Update Pattern | `.UpdateDataStream(ctx, streamId).DataStreamRequest(req).Execute()` |
| Retrieve Pattern | `.RetrieveDataStream(ctx, streamId).Execute()` |
| List Method | `.ListDataStreams(ctx).Execute()` |
| Delete Method | `.DeleteDataStream(ctx, streamId).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### Update method: PUT, not PATCH

`PatchedDataStreamRequest` only exposes `name`, `active`, `inputs` and `transform` — there is **no `outputs` field**. A `PATCH`-based update could therefore never change the stream's endpoints. Use `UpdateDataStream` (PUT) with a full `DataStreamRequest`.

This is the opposite of the choice made for Connectors and most other V4 resources, which use `PartialUpdate*`. Do not "fix" it to PATCH.

### Sibling APIs

The SDK also exposes `DataStreamTemplatesAPI` and `DataStreamDataSourcesAPI`.

* **Templates** are implemented — see **[agents/DATA_STREAM_TEMPLATES.md](DATA_STREAM_TEMPLATES.md)** for `azion_data_stream_template` and `azion_data_stream_templates`. Note their update method differs from this one: Templates use PATCH, Streams must use PUT.
* **Data Sources** (read-only list of available log sources) is **not** wired into the provider. The valid `inputs[].attributes.data_source` slugs are hardcoded in this document instead.

---

## Domain Model

A stream is a pipeline of three lists:

```
inputs  ──▶  transform (ordered, optional)  ──▶  outputs
```

| Field | Cardinality | Polymorphic? | Notes |
|-------|-------------|--------------|-------|
| `inputs` | 1..n, required | No | Only `type = "raw_logs"` exists; `attributes.data_source` selects the log source |
| `transform` | 0..n, optional | **Yes** (`oneOf`) | Order matters — the API applies them in sequence. If present, must contain `sampling` or `filter_workloads` (see below) |
| `outputs` | 1..n, required | **Yes** (`oneOf`) | One entry per destination endpoint |

### SDK type map

| Terraform concept | Response type | Request type |
|-------------------|---------------|--------------|
| Stream | `DataStream` | `DataStreamRequest` |
| Input entry | `InputInputDataSourceAttributes` | `InputInputDataSourceAttributesRequest` |
| Input attributes | `InputDataSource` | `InputDataSourceRequest` |
| Transform entry | `Transform` (oneOf wrapper) | `TransformRequest` (oneOf wrapper) |
| Output entry | `OutputBase` (`Type` + `Attributes Output`) | `OutputRequestBase` (`Type` + `Attributes OutputRequest`) |

Note the asymmetry: transform entries **are** the `oneOf` value, while output entries are a base struct whose `Attributes` field holds the `oneOf`. Getting this backwards is the easiest mistake to make here.

### Data sources (`inputs[].attributes.data_source`)

`workloads`, `waf`, `functions_console`, `activity_history`, `http`, `cells_console`, `rtm_activity`.

### Transform types

| `type` | oneOf member | Attributes |
|--------|--------------|------------|
| `sampling` | `TransformTransformSamplingAttributes` | `rate` (int64) |
| `filter_workloads` | `TransformTransformFilterWorkloadsAttributes` | `workloads` (`[]int64`) |
| `render_template` | `TransformTransformRenderTemplateAttributes` | `template` (int64) |

### Output (endpoint) types

| `type` | oneOf member | Required attributes | Optional attributes |
|--------|--------------|---------------------|---------------------|
| `standard` | `HttpPostEndpoint` | `url`, `headers` | `log_line_separator`, `payload_format`, `max_size` |
| `kafka` | `KafkaEndpoint` | `bootstrap_servers`, `kafka_topic`, `use_tls` | — |
| `s3` | `S3Endpoint` | `access_key`, `secret_key`, `region`, `bucket_name`, `content_type`, `host_url` | `object_key_prefix` |
| `big_query` | `BigQueryEndpoint` | `dataset_id`, `project_id`, `table_id`, `service_account_key` | — |
| `elasticsearch` | `ElasticsearchEndpoint` | `url`, `api_key` | — |
| `splunk` | `SplunkEndpoint` | `url`, `api_key` | — |
| `aws_kinesis_firehose` | `AWSKinesisFirehoseEndpoint` | `access_key`, `stream_name`, `region`, `secret_key` | — |
| `datadog` | `DatadogEndpoint` | `url`, `api_key` | — |
| `qradar` | `QRadarEndpoint` | `url` | — |
| `azure_monitor` | `AzureMonitorEndpoint` | `log_type`, `shared_key`, `workspace_id` | `time_generated_field` |
| `azure_blob_storage` | `AzureBlobStorageEndpoint` | `storage_account`, `container_name`, `blob_sas_token` | — |

`content_type` accepts `plain/text` or `application/gzip`.

---

## Data Source Implementation

### Singular Data Source (Read by ID)

`internal/data_source_data_stream.go` → `azion_data_stream`.

* `id` is a **required string** at the root; parse it with `strconv.ParseInt` before calling the API.
* State ID marker: `types.StringValue("Get By Id Data Stream")`.
* The three polymorphic lists are exposed as **JSON strings** (`json.Marshal` of the SDK slices), following the precedent set by `azion_connector`'s `attributes`. This avoids a data-source schema with eleven optional endpoint blocks, and users decode with `jsondecode`.

### Plural Data Source (List Multiple Resources)

`internal/data_source_data_streams.go` → `azion_data_streams`.

* No arguments; `counter` from `PaginatedDataStreamList.Count`, `results` from `.GetResults()`.
* State ID marker: `types.StringValue("Get All Data Streams")`.

### Shared attribute map

Both data sources call `dataStreamDataSourceAttributes()` (defined in `data_source_data_stream.go`) and both reuse `DataStreamResults` and `populateDataStreamDataSourceResults`. When adding a field, edit the shared helper — not one file.

### Key Differences: Singular vs Plural

| Aspect | Singular | Plural |
|--------|----------|--------|
| Root `id` | Required input | Computed marker |
| Payload attribute | `data` (single nested) | `results` (list nested) + `counter` |
| API call | `RetrieveDataStream(ctx, id)` | `ListDataStreams(ctx)` |

---

## Resource Implementation

`internal/resource_data_stream.go` → `azion_data_stream`.

Standard V4 resource shape: a `data_stream` single-nested attribute holding the body, plus root-level `id` (string), `last_updated`, and `schema_version`.

| Method | API call | Notes |
|--------|----------|-------|
| Create | `CreateDataStream` | Populates state directly from the create response |
| Read | `RetrieveDataStream` | 404 → `resp.State.RemoveResource(ctx)` |
| Update | `UpdateDataStream` (PUT) | Full body; see [PUT, not PATCH](#update-method-put-not-patch) |
| Delete | `DeleteDataStream` via `utils.RetryOn429Delete` | 404 is treated as success |
| ImportState | `RetrieveDataStream` | Builds a fresh model, so no prior state to seed from |

Unlike Connectors, Create does **not** re-read the resource afterwards — the create response already carries the full `DataStream`.

---

## Schema Definition Patterns

### Polymorphic entries: one optional block per type

Each `transform` / `outputs` entry has a required `type` plus one optional `*_attributes` block per supported type, named `<type>_attributes`. This mirrors `azion_connector`'s `storage_attributes` / `http_attributes`.

```go
"outputs": schema.ListNestedAttribute{
    Required: true,
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "type":                schema.StringAttribute{Required: true},
            "standard_attributes": schema.SingleNestedAttribute{Optional: true, Attributes: ...},
            "kafka_attributes":    schema.SingleNestedAttribute{Optional: true, Attributes: ...},
            // ... one per endpoint type
        },
    },
},
```

The "exactly one block, matching `type`" invariant is **not** expressible in the schema; it is enforced at build time and surfaces as a plan-time error from `buildDataStreamOutputAttributes`.

### Shared attribute helpers

`elasticsearch`, `splunk` and `datadog` all take only `url` + `api_key`. They share `dataStreamURLAPIKeyAttributes(vendor string)` for the schema and `DataStreamURLAPIKeyOutputModel` / `populateURLAPIKeyOutput` for the model. If any of the three gains a distinct field, split it out rather than adding an unused optional to all three.

### Sensitive fields

Mark `Sensitive: true` on: `access_key`, `secret_key`, `api_key`, `shared_key`, `blob_sas_token`, `service_account_key`, and the `standard` endpoint's `headers` map (it usually carries an auth token).

---

## Polymorphic Type Handling

### Building requests (Terraform → SDK)

Transform entries wrap the typed struct directly:

```go
attrs := azionapi.NewTransformSamplingRequest(transform.SamplingAttributes.Rate.ValueInt64())
item := azionapi.NewTransformTransformSamplingAttributesRequest(transformType, *attrs)
out = append(out, azionapi.TransformTransformSamplingAttributesRequestAsTransformRequest(item))
```

Output entries wrap the endpoint inside `OutputRequestBase`, and the endpoint carries its own `type` field too — **set it in both places**:

```go
endpoint := azionapi.NewKafkaEndpointRequest(servers, topic, useTLS, outputType)
attrs := azionapi.KafkaEndpointRequestAsOutputRequest(endpoint)
out = append(out, *azionapi.NewOutputRequestBase(outputType, attrs))
```

### Reading responses (SDK → Terraform)

```go
switch t := transforms[i].GetActualInstance().(type) {
case *azionapi.TransformTransformSamplingAttributes:
case *azionapi.TransformTransformFilterWorkloadsAttributes:
case *azionapi.TransformTransformRenderTemplateAttributes:
}

switch e := outputs[i].Attributes.GetActualInstance().(type) {
case *azionapi.HttpPostEndpoint:
case *azionapi.KafkaEndpoint:
// ...
}
```

`GetActualInstance()` returns `nil` when the discriminator didn't match any member — always guard for it.

### Empty required lists must marshal as `[]`, not `null`

`DataStreamRequest.Inputs`, `.Transform` and `.Outputs` are required non-pointer slices, so a `nil` slice serializes as JSON `null` and the API rejects it. All three `build*Requests` helpers start from `[]T{}`:

```go
out := []azionapi.TransformRequest{} // not: var out []azionapi.TransformRequest
```

This matters most for `transform`, which is optional in the Terraform schema.

---

## Error Handling

Standard V4 pattern. `addDataStreamAPIError(&resp.Diagnostics, err, response)` in the resource; `errPrintDataStream` / `errPrintDataStreams` in the data sources.

```go
if err != nil {
    if response != nil && response.StatusCode == http.StatusTooManyRequests {
        result, response, err = utils.RetryOn429(func() (*azionapi.DataStreamResponse, *http.Response, error) {
            return r.client.api.DataStreamStreamsAPI.CreateDataStream(ctx).DataStreamRequest(req).Execute()
        }, 5)
        // ...
    } else {
        addDataStreamAPIError(&resp.Diagnostics, err, response)
        return
    }
}
```

Delete uses `utils.RetryOn429Delete`. Read treats 404 as "resource gone"; Delete treats 404 as success.

---

## Type Conversions

| API field | Go type | Terraform |
|-----------|---------|-----------|
| `id` | `int64` | `types.Int64Value` (nested) / `strconv.FormatInt` for root `id` |
| `active` | `*bool` | `types.BoolPointerValue` |
| `created`, `last_modified` | `time.Time` | `types.StringValue(t.Format(time.RFC850))` |
| `max_size` | `NullableInt64` | `types.Int64PointerValue(e.MaxSize.Get())` |
| `object_key_prefix`, `time_generated_field` | `NullableString` | `types.StringPointerValue(e.X.Get())` |
| `log_line_separator`, `payload_format` | `*string` | `types.StringPointerValue` |
| `headers` | `map[string]string` | `map[string]types.String` + `schema.MapAttribute{ElementType: types.StringType}` |
| `workloads` | `[]int64` | `[]types.Int64` + `schema.ListAttribute{ElementType: types.Int64Type}` |

The API field is `created`, but the Terraform attribute is **`created_at`** to match the rest of the provider (Connectors, CRLs, Custom Pages).

---

## Common Issues

### Credentials come back masked → perpetual drift

The API redacts secrets in responses. `populateDataStreamOutputs` therefore seeds each output from the prior state entry and keeps the configured value for write-only fields, via `preferPriorSecret`. Prior entries are matched **positionally**, and only when `prior[i].Type == outputs[i].Type` — a type change at the same index is treated as a fresh entry so a stale secret is never carried across endpoint types.

Consequence: after `terraform import` there is no prior state, so the masked value lands in state and re-adding the real credential produces one diff. Document this rather than trying to work around it.

### API defaults on optional fields → perpetual drift

`log_line_separator`, `payload_format`, `max_size`, `object_key_prefix` and `time_generated_field` get server-side defaults. They use the package-level generic `shouldPopulate` helper (defined in `resource_connector.go`) so a field the user never configured stays null instead of absorbing the echo:

```go
if shouldPopulate(priorAttrs, func(p *DataStreamStandardOutputModel) bool { return !p.MaxSize.IsNull() }) {
    attrs.MaxSize = types.Int64PointerValue(e.MaxSize.Get())
} else {
    attrs.MaxSize = priorAttrs.MaxSize
}
```

`shouldPopulate` returns `true` when `prior == nil`, so the `else` branch only runs when `priorAttrs` is non-nil — the dereference is safe.

### Reordering `transform` or `outputs` is a real change

Both are `ListNestedAttribute`, so order is significant, and for `transform` it genuinely is (the API applies steps in sequence). Do not switch either to a `SetNestedAttribute`: it would break the positional prior-state matching that keeps secrets stable.

### A non-empty `transform` list must include `sampling` or `filter_workloads`

Observed from the live API: a `transform` list holding only `render_template` is rejected with

```
400 - code 32002 "Workloads Must Be Provided"
"If sampling is disabled, workloads must be provided."   source.pointer: /data/transform
```

Omitting `transform` entirely is accepted, so the constraint applies only once the list is present. The API appears to treat the three transform types as slots with a cross-field rule rather than a free-form list, even though the schema models them as an ordered array.

This is **not** validated in the provider schema — it is a cross-entry rule that Terraform's schema validation cannot express, and it is inferred from one API error message rather than documented in the OpenAPI spec. Do not add a client-side check that guesses at the full rule; let the API be the authority and surface its error. Workaround for a template-only pipeline: pair `render_template` with a no-op `sampling` entry at `rate = 100`.

### `render_template` template IDs

The `render_template` transform takes a numeric template ID. Custom templates can be created with `azion_data_stream_template` and referenced directly:

```hcl
render_template_attributes = {
  template = azion_data_stream_template.example.template.id
}
```

Azion's built-in templates have no resource — they are discovered through the `azion_data_stream_templates` data source, filtering on `custom == false`. Deleting a template a live stream still references may be rejected by the API; repoint or remove the transform first.

---

## Files

| Path | Purpose |
|------|---------|
| `internal/resource_data_stream.go` | `azion_data_stream` resource |
| `internal/data_source_data_stream.go` | `azion_data_stream` data source + shared attribute map, `DataStreamResults`, `populateDataStreamDataSourceResults` |
| `internal/data_source_data_streams.go` | `azion_data_streams` data source |
| `internal/provider.go` | Registers `dataSourceAzionDataStream`, `dataSourceAzionDataStreams`, `NewDataStreamResource` |
| `docs/resources/data_stream.md` | Resource documentation |
| `docs/data-sources/data_stream.md` | Singular data source documentation |
| `docs/data-sources/data_streams.md` | Plural data source documentation |
| `examples/resources/azion_data_stream/resource.tf` | Resource examples |
| `examples/data-sources/azion_data_stream/data-source.tf` | Singular data source example |
| `examples/data-sources/azion_data_streams/data-source.tf` | Plural data source example |

Data Stream Streams are a **top-level resource** — they have no parent resource, so the child-resource documentation rule in `AGENTS.md` does not apply.
