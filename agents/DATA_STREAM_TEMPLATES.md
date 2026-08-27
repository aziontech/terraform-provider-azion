# Data Stream - Templates - Code Generation Guide

This document provides specific guidance for implementing the Data Stream **Templates** resource and data sources in the Terraform provider.

See **[agents/DATA_STREAM.md](DATA_STREAM.md)** for the sibling Streams implementation, which consumes templates through its `render_template` transform.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Domain Model](#domain-model)
3. [Data Source Implementation](#data-source-implementation)
4. [Resource Implementation](#resource-implementation)
5. [Schema Definition Patterns](#schema-definition-patterns)
6. [Error Handling](#error-handling)
7. [Type Conversions](#type-conversions)
8. [Common Issues](#common-issues)
9. [Files](#files)

---

## SDK Selection

Data Stream Templates use the **V4 SDK (`azion-api`)** for the resource and both data sources:

| Implementation | SDK Package | Client Field | Base URL |
|----------------|-------------|--------------|----------|
| Data Stream Template (Singular Data Source) | `azion-api` (v4) | `api.DataStreamTemplatesAPI` | `https://api.azion.com/v4` |
| Data Stream Templates (Plural Data Source) | `azion-api` (v4) | `api.DataStreamTemplatesAPI` | `https://api.azion.com/v4` |
| Data Stream Template (Resource) | `azion-api` (v4) | `api.DataStreamTemplatesAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `TemplateRequest` |
| Update Request Type | `PatchedTemplateRequest` (PATCH) — see note below |
| Response Type | `TemplateResponse` with `Data Template` |
| List Response Type | `PaginatedTemplateList` |
| Create Pattern | `.CreateTemplate(ctx).TemplateRequest(req).Execute()` |
| Update Pattern | `.PartialUpdateTemplate(ctx, templateId).PatchedTemplateRequest(req).Execute()` |
| Retrieve Pattern | `.RetrieveTemplate(ctx, templateId).Execute()` |
| List Method | `.ListTemplates(ctx).Execute()` |
| Delete Method | `.DeleteTemplate(ctx, templateId).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### Update method: PATCH, not PUT

`PatchedTemplateRequest` exposes all three mutable fields (`name`, `active`, `data_set`), so PATCH is complete here and matches the provider's V4 convention.

This is the **opposite** of the Streams resource, which is forced onto `UpdateDataStream` (PUT) because `PatchedDataStreamRequest` has no `outputs` field. Do not "unify" the two — the asymmetry is a property of the API, not an oversight. `UpdateTemplate` (PUT) also exists and would work; PATCH is preferred only for convention.

---

## Domain Model

Templates are a **flat scalar model** — no polymorphism, no nested lists. This makes them far simpler than Streams; none of the `oneOf` machinery in `DATA_STREAM.md` applies.

| API field | Type | Mutable? | Notes |
|-----------|------|----------|-------|
| `id` | `int64` | No | |
| `name` | `string` | Yes | Required on create |
| `active` | `*bool` | Yes | Optional; API supplies a default |
| `data_set` | `string` | Yes | Required on create; the payload template |
| `custom` | `bool` | **No** | Not in `TemplateRequest`; computed only |
| `last_editor` | `string` | No | |
| `created_at` | `time.Time` | No | |
| `last_modified` | `time.Time` | No | |

### `custom` distinguishes built-in from user-defined templates

Azion ships built-in templates (`custom = false`) that every account can reference but nobody can modify. Anything this resource creates is `custom = true`.

`custom` must stay `Computed`-only. Never make it Optional — a user setting `custom = false` would silently do nothing, since the field is absent from both request types.

### `data_set` is an opaque string, but the API trims it

It holds the record layout with `$variable` placeholders. It is commonly JSON, but the API accepts any string (a flat space-separated layout works too), and the provider does **not** parse or validate it. Do not add JSON handling via `utils.ConvertStringToInterface` — unlike the `json_args` fields elsewhere in this provider, `data_set` is a `string` on both the request and response side, not an object.

**The API does strip leading and trailing whitespace.** Confirmed against the live API: a heredoc `data_set` sent as `"{...}\n"` comes back as `"{...}"`. See [`data_set` whitespace](#data_set-whitespace-is-stripped-by-the-api) for why this needs handling rather than being passed straight through.

---

## Data Source Implementation

### Singular Data Source (Read by ID)

`internal/data_source_data_stream_template.go` → `azion_data_stream_template`.

* `id` is a **required string** at the root; parse it with `strconv.ParseInt` before calling the API.
* State ID marker: `types.StringValue("Get By Id Data Stream Template")`.
* Reads built-in and custom templates alike.

### Plural Data Source (List Multiple Resources)

`internal/data_source_data_stream_templates.go` → `azion_data_stream_templates`.

* No arguments; `counter` from `PaginatedTemplateList.Count`, `results` from `.GetResults()`.
* State ID marker: `types.StringValue("Get All Data Stream Templates")`.

This data source carries real product weight: built-in templates have no resource, so listing is the only way for a user to discover the numeric ID that a stream's `render_template` transform needs. Filter on `custom == false` to isolate the built-ins.

### Shared attribute map

Both data sources call `dataStreamTemplateDataSourceAttributes()` (defined in `data_source_data_stream_template.go`) and both reuse `DataStreamTemplateResults` and `populateDataStreamTemplateDataSourceResults`. When adding a field, edit the shared helper — not one file.

Unlike the Streams data sources, no field needs to be flattened to a JSON string; the model is already flat.

### Key Differences: Singular vs Plural

| Aspect | Singular | Plural |
|--------|----------|--------|
| Root `id` | Required input | Computed marker |
| Payload attribute | `data` (single nested) | `results` (list nested) + `counter` |
| API call | `RetrieveTemplate(ctx, id)` | `ListTemplates(ctx)` |

---

## Resource Implementation

`internal/resource_data_stream_template.go` → `azion_data_stream_template`.

Standard V4 resource shape: a `template` single-nested attribute holding the body, plus root-level `id` (string), `last_updated`, and `schema_version`.

| Method | API call | Notes |
|--------|----------|-------|
| Create | `CreateTemplate` | Populates state directly from the create response |
| Read | `RetrieveTemplate` | 404 → `resp.State.RemoveResource(ctx)` |
| Update | `PartialUpdateTemplate` (PATCH) | Sends all three mutable fields every apply |
| Delete | `DeleteTemplate` via `utils.RetryOn429Delete` | 404 is treated as success |
| ImportState | `RetrieveTemplate` | Builds a fresh model |

Create does not re-read the resource afterwards — the create response already carries the full `Template`.

---

## Schema Definition Patterns

Flat `SingleNestedAttribute`, no per-type blocks:

```go
"template": schema.SingleNestedAttribute{
    Required: true,
    Attributes: map[string]schema.Attribute{
        "name":     schema.StringAttribute{Required: true},
        "data_set": schema.StringAttribute{Required: true},
        "active":   schema.BoolAttribute{Optional: true, Computed: true},
        "custom":   schema.BoolAttribute{Computed: true},
        // id, last_editor, created_at, last_modified: Computed
    },
},
```

### No sensitive fields

Nothing in a template is a credential. Do not mark `data_set` `Sensitive` — it is the field users most need to read in plan output, and marking it would hide every meaningful diff behind `(sensitive value)`.

---

## Error Handling

Standard V4 pattern. `addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)` in the resource; `errPrintDataStreamTemplate` / `errPrintDataStreamTemplates` in the data sources.

```go
if err != nil {
    if response != nil && response.StatusCode == http.StatusTooManyRequests {
        result, response, err = utils.RetryOn429(func() (*azionapi.TemplateResponse, *http.Response, error) {
            return r.client.api.DataStreamTemplatesAPI.CreateTemplate(ctx).TemplateRequest(req).Execute()
        }, 5)
        // ...
    } else {
        addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)
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
| `custom` | `bool` (non-pointer) | `types.BoolValue(template.GetCustom())` |
| `data_set` | `string` | `types.StringValue` — verbatim, no normalization |
| `created_at`, `last_modified` | `time.Time` | `types.StringValue(t.Format(time.RFC850))` |

Note `created_at` is already the API field name here, unlike Streams where the API field is `created` and the Terraform attribute is renamed to `created_at`.

---

## Common Issues

### `data_set` whitespace is stripped by the API

The API strips leading and trailing whitespace from `data_set`. Because heredocs (`<<-EOT`) always end in a newline, and heredocs are the natural way to write a multi-line template, this hits real configurations immediately:

```
Error: Provider produced inconsistent result after apply

.template.data_set: was cty.StringVal("{...\n}\n"), but now cty.StringVal("{...\n}")
```

`data_set` is `Required`, and Terraform requires a Required attribute to survive apply byte-for-byte. Storing the API's trimmed echo fails that check outright — and had it not, the value would then differ from config on every subsequent plan.

`preferConfiguredDataSet` handles it: keep the prior/configured value when it matches the API's echo **after `strings.TrimSpace` on both sides**, otherwise store the API value.

* Create / Update — prior is the plan value from config, so config wins over the trimmed echo.
* Read — prior is the state value, so a whitespace-only difference is not reported as drift.
* ImportState — the model is empty, prior is null, so the API value is used as-is.

Note this is **not** provider-side normalization: normalizing the value we send or store cannot fix the problem, because Terraform compares stored state against the *config* value, not against anything the provider computed. Preserving the configured value is the only correct move.

Comparison is `TrimSpace` (both ends) rather than `TrimRight`, even though only trailing stripping has been observed. If the server strips leading whitespace too, `TrimRight` would still fail; the cost of the wider comparison is that a leading-whitespace-only edit is not reported as a change, which is an acceptable trade.

A genuine server-side rewrite (reordered JSON keys, changed content) still differs after trimming, so it is stored and surfaces as real drift. That is intended: this is the one field users actively edit, so silently absorbing server changes would hide real problems. Advise `jsonencode(...)` in HCL to keep the local value canonical.

Other than `data_set`, no prior-state seeding is used — unlike Streams, which needs `preferPriorSecret` and `shouldPopulate` because its endpoints return masked credentials and server-side defaults. Templates have neither, so plain round-tripping is correct for every other field. Do not copy that machinery over.

### Deleting a template still referenced by a stream

Confirmed against the live API — deleting a template a stream still references fails with:

```
400 - code 32005 "Cannot Delete Template"
"Cannot delete a template with associated data streams."   source.pointer: /data
```

Terraform's own ordering is already correct: a stream whose transform references `azion_data_stream_template.x.template.id` carries an implicit dependency and is destroyed first. No `depends_on` is needed, and adding one would not help.

`utils.RetryOn429Delete` already retries `400 + isReferencedByAnotherResource`, but the detail string above matched **none** of the phrases in `utils.stillInUseMsgs` — not even `"in use"` — so the retry silently never fired. `"associated data stream"` was added to that list. When touching a delete path, check that the API's actual error phrasing matches an entry in `stillInUseMsgs`; the retry is only as good as that substring list.

Note the retry only helps with a genuine lag in releasing the association. If a stream exists in the API but not in Terraform state, the reference is permanent and the retry just burns ~50s before failing with the same error. Diagnose with:

```sh
curl -s -H "Authorization: token $AZION_TOKEN" \
  "https://api.azion.com/v4/workspace/stream/streams?page_size=100"
```

and inspect the `template` value inside each entry's `transform` array.

### Built-in templates are not importable as this resource

`terraform import` on a built-in template ID will succeed at the API level and land `custom = false` in state, but any subsequent apply that tries to modify it will fail. Only import templates you actually own.

---

## Files

| Path | Purpose |
|------|---------|
| `internal/resource_data_stream_template.go` | `azion_data_stream_template` resource |
| `internal/data_source_data_stream_template.go` | `azion_data_stream_template` data source + shared attribute map, `DataStreamTemplateResults`, `populateDataStreamTemplateDataSourceResults` |
| `internal/data_source_data_stream_templates.go` | `azion_data_stream_templates` data source |
| `internal/provider.go` | Registers `dataSourceAzionDataStreamTemplate`, `dataSourceAzionDataStreamTemplates`, `NewDataStreamTemplateResource` |
| `docs/resources/data_stream_template.md` | Resource documentation |
| `docs/data-sources/data_stream_template.md` | Singular data source documentation |
| `docs/data-sources/data_stream_templates.md` | Plural data source documentation |
| `examples/resources/azion_data_stream_template/resource.tf` | Resource examples |
| `examples/data-sources/azion_data_stream_template/data-source.tf` | Singular data source example |
| `examples/data-sources/azion_data_stream_templates/data-source.tf` | Plural data source example |

Data Stream Templates are a **top-level resource** — they have no parent resource, so the child-resource documentation rule in `AGENTS.md` does not apply. Streams reference templates by ID, but that is a sibling reference, not a parent-child containment.
