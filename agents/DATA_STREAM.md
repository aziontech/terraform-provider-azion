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
| `inputs` | exactly 1, required | No | Only `type = "raw_logs"` exists; `attributes.data_source` selects the log source. The API keeps only one input out of a longer list, silently — the schema caps it at 1 |
| `transform` | 1..n, required | **Yes** (`oneOf`) | Must contain a `render_template` entry plus either `sampling` or `filter_workloads` (see below). The API normalizes the order, so the configured order is cosmetic |
| `outputs` | exactly 1, required | **Yes** (`oneOf`) | The API stores one output per stream — see below. The schema caps it at 1 |

### SDK type map

| Terraform concept | Response type | Request type |
|-------------------|---------------|--------------|
| Stream | `DataStream` | `DataStreamRequest` |
| Input entry | `InputInputDataSourceAttributes` | `InputInputDataSourceAttributesRequest` |
| Input attributes | `InputDataSource` | `InputDataSourceRequest` |
| Transform entry | `Transform` (oneOf wrapper) | `TransformRequest` (oneOf wrapper) |
| Output entry | `Output` (oneOf wrapper) | `OutputRequest` (oneOf wrapper) |
| Output variant | `<Endpoint>Attributes` (`Type` + `Attributes <Endpoint>`) | `<Endpoint>AttributesRequest` (`Type` + `Attributes <Endpoint>Request`) |

Transform and output entries share the same shape: the entry **is** the `oneOf` value, and each
variant carries its own `type` discriminator alongside the endpoint payload in `Attributes`.

> SDK `v0.266.0` reshaped outputs. Before it, an entry was an `OutputRequestBase`/`OutputBase`
> wrapper whose `Attributes` field held the `oneOf`, and the endpoint struct itself repeated the
> `type` field. The wrapper structs still exist in the SDK but are no longer referenced by
> `DataStreamRequest`/`DataStream` — do not use them.

### Data sources (`inputs[].attributes.data_source`)

`workloads`, `waf`, `functions_console`, `activity_history` — the four slugs served by
`GET /data_stream/data_sources`. `http`, `cells_console` and `rtm_activity` were accepted by
earlier API versions and are gone. The API does **not** reject an unrecognized slug; it silently
stores a different one, which surfaces in Terraform as `Provider produced inconsistent result
after apply`. That is why the schema validates the value with `stringvalidator.OneOf` instead of
deferring to the API. Re-check the endpoint before trusting this list.

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

Output entries follow the identical pattern — the endpoint struct holds only its own payload,
and the discriminator goes on the `<Endpoint>AttributesRequest` variant:

```go
endpoint := azionapi.NewKafkaEndpointRequest(servers, topic, useTLS)
item := azionapi.NewKafkaEndpointAttributesRequest(outputType, *endpoint)
out = append(out, azionapi.KafkaEndpointAttributesRequestAsOutputRequest(item))
```

### Reading responses (SDK → Terraform)

```go
switch t := transforms[i].GetActualInstance().(type) {
case *azionapi.TransformTransformSamplingAttributes:
case *azionapi.TransformTransformFilterWorkloadsAttributes:
case *azionapi.TransformTransformRenderTemplateAttributes:
}

switch e := outputs[i].GetActualInstance().(type) {
case *azionapi.HttpPostEndpointAttributes:
case *azionapi.KafkaEndpointAttributes:
// ...
}
```

Each output variant nests its payload one level down, so the endpoint fields are read from
`e.Attributes` and the discriminator from `e.GetType()`. `dataStreamOutputEndpoint` in
`internal/resource_data_stream.go` unwraps both in one place so the rest of the population
code can switch on the endpoint structs (`*azionapi.HttpPostEndpoint`, ...) directly.

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

### Closing response bodies

Every API call in these files uses:

```go
if response != nil {
    defer func(r *http.Response) { _ = r.Body.Close() }(response)
}
```

not the bare `defer response.Body.Close()` seen in older resources. `errcheck` rejects the bare form on new lines (CI gates with `--new-from-patch`), and the response must be passed as an argument rather than captured, so that retry sites which reassign `response` close each body exactly once. See [AGENTS.md](../AGENTS.md#response-body-closure-pattern).

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

### The API normalizes `transform` order

`outputs` order is preserved by the API, but `transform` is not: the pipeline comes back in the
API's own canonical order regardless of what was submitted (sending `[render_template, sampling]`
stores `[sampling, render_template]`). `populateDataStreamTransforms` therefore re-orders the
response to match the prior list's sequence of types via `orderTransformsLikePrior`, falling back
to the API's order when the types don't match so genuine drift still shows. Without that,
any config not already written in the API's order fails with `Provider produced inconsistent
result after apply`.

Both attributes stay `ListNestedAttribute`. Do not switch either to a `SetNestedAttribute`: it
would break the positional prior-state matching that keeps secrets stable.

### `transform` is required and carries cross-entry rules

Observed from the live API — all four are enforced:

| Constraint | Error |
|---|---|
| The `transform` key must be present | `400 - 10059 "Required Field"` |
| It must hold at least one entry | `400 - 10049 "Min Length List Field"` |
| It must include a `render_template` entry | `400 - 32008 "Template Must Be Provided"` |
| It must include a `sampling` or `filter_workloads` entry | `400 - 32002 "Workloads Must Be Provided"` |

All four point at `/data/transform`. The API treats the three transform types as slots with
cross-field rules rather than a free-form list, even though the schema models them as an ordered
array.

The schema enforces the first two (`Required` plus `listvalidator.SizeAtLeast(1)`), which
Terraform can express. The last two are cross-entry rules that schema validation cannot express
and that are inferred from error messages rather than documented in the OpenAPI spec — do not add
a client-side check that guesses at the full rule; let the API be the authority and surface its
error. Workaround for a template-only pipeline: pair `render_template` with a no-op `sampling`
entry at `rate = 100`.

### Only one `outputs` entry is stored

Fan-out from a single stream does not work, in two different ways:

- outputs all of the same `type`: the API keeps **only the last entry**, silently
- outputs of differing types: the API answers `500 - 10067 "Internal Server Error"`

Verified against the live API by posting three `kafka` outputs and reading the stream back — one
survived, the third. The schema therefore caps `outputs` at one entry
(`listvalidator.SizeBetween(1, 1)`), the same treatment `inputs` gets, so the loss surfaces as a
plan-time error instead of `Provider produced inconsistent result after apply`. One stream per
destination endpoint is the working pattern. Revisit if the API starts honouring longer lists.

### Only one stream per account can be active

Account-wide, not per `data_source`: activating a stream silently deactivates whichever was
active before. Verified by creating two active streams on different data sources — the older one
came back `active: false`. The create/update response still echoes `active: true`, so the
provider writes `true` into state while the API holds `false`, and the next plan shows
`active = false -> true` indefinitely. Nothing to validate at plan time (it depends on
account-wide state), so this is documentation-only.

### `elasticsearch`, `splunk` and `datadog` outputs cannot be decoded

All three are `{url, api_key}` and `oneOf(Output)` has no `discriminator`, so the generated
`UnmarshalJSON` tries every variant and counts matches — three match, and it fails with
`data matches more than one schema in oneOf(Output)`. 8 of the 11 endpoint types decode fine;
these three do not, in either the old or the new SDK.

The blast radius includes the plural data source: `azion_data_streams` decodes every stream in
the account, so one such stream anywhere makes the whole list read fail. There is no provider-side
workaround — by the time `Execute()` returns the error the response body is consumed, so the raw
JSON is not available to decode by hand. The fix is `discriminator: {propertyName: type}` on the
`Output`/`OutputRequest` schemas in the OpenAPI spec, or giving the three schemas distinguishing
required fields.

### `log_line_separator` is trimmed

The API runs the equivalent of `strings.TrimSpace` on it, so a real newline, tab, CR or run of
spaces is all stored as `""`. The API's own default is the **two-character** text `\n`
(backslash, n), which round-trips unchanged — as does any value with no surrounding whitespace.
The schema validates against leading/trailing whitespace with `stringvalidator.RegexMatches`
rather than silently normalizing, since normalizing would mean editing the user's configured
value behind their back.

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
