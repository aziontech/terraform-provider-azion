# Workloads - Code Generation Guide

This document provides specific guidance for implementing Workload resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
3. [Schema Definition Patterns](#schema-definition-patterns)
4. [Error Handling](#error-handling)
5. [Type Conversions](#type-conversions)
6. [Common Issues](#common-issues)

---

## SDK Selection

Workloads use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Workload (Singular Data Source) | `azion-api` (v4) | `api.WorkloadsAPI` | `https://api.azion.com/v4` |
| Workloads (Plural Data Source) | `azion-api` (v4) | `api.WorkloadsAPI` | `https://api.azion.com/v4` |
| Workload (Resource) | `azion-api` (v4) | `api.WorkloadsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `WorkloadRequest` |
| Update Request Type | `PatchedWorkloadRequest` |
| Response Type | `WorkloadResponse` with `Data` field |
| List Response Type | `PaginatedWorkloadList` |
| Create Pattern | `.CreateWorkload(ctx).WorkloadRequest(req).Execute()` |
| Update Pattern | `.PartialUpdateWorkload(ctx, id).PatchedWorkloadRequest(req).Execute()` |
| Retrieve Pattern | `.RetrieveWorkload(ctx, workloadId).Execute()` |
| List Method | `.ListWorkloads(ctx).Execute()` |
| Delete Method | `.DeleteWorkload(ctx, workloadId).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Workload by its identifier:

**File:** `internal/data_source_workload.go`

```go
package provider

import (
    "context"
    "fmt"
    "net/http"
    "strconv"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &WorkloadDataSource{}
    _ datasource.DataSourceWithConfigure = &WorkloadDataSource{}
)

func dataSourceAzionWorkload() datasource.DataSource {
    return &WorkloadDataSource{}
}

type WorkloadDataSource struct {
    client *apiClient
}

type WorkloadDataSourceModel struct {
    Data WorkloadResults `tfsdk:"data"`
    ID   types.String    `tfsdk:"id"`
}

type WorkloadResults struct {
    ID                        types.Int64        `tfsdk:"id"`
    Name                      types.String       `tfsdk:"name"`
    Active                    types.Bool         `tfsdk:"active"`
    LastEditor                types.String       `tfsdk:"last_editor"`
    LastModified              types.String       `tfsdk:"last_modified"`
    CreatedAt                 types.String       `tfsdk:"created_at"`
    Infrastructure            types.Int64        `tfsdk:"infrastructure"`
    Tls                       *TLSWorkloadModel  `tfsdk:"tls"`
    Protocols                 *ProtocolsModel    `tfsdk:"protocols"`
    Mtls                      *MTLSModel         `tfsdk:"mtls"`
    Domains                   types.List         `tfsdk:"domains"`
    WorkloadDomainAllowAccess types.Bool         `tfsdk:"workload_domain_allow_access"`
    WorkloadDomain            types.String       `tfsdk:"workload_domain"`
    ProductVersion            types.String       `tfsdk:"product_version"`
}
```

### Plural Data Source (List Multiple Resources)

For listing all Workloads:

**File:** `internal/data_source_workloads.go`

```go
package provider

import (
    "context"
    "fmt"
    "net/http"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &WorkloadsDataSource{}
    _ datasource.DataSourceWithConfigure = &WorkloadsDataSource{}
)

func dataSourceAzionWorkloads() datasource.DataSource {
    return &WorkloadsDataSource{}
}

type WorkloadsDataSource struct {
    client *apiClient
}

type WorkloadsDataSourceModel struct {
    Counter types.Int64        `tfsdk:"counter"`
    Results []WorkloadsResults `tfsdk:"results"`
    ID      types.String       `tfsdk:"id"`
}

type WorkloadsResults struct {
    ID                        types.Int64        `tfsdk:"id"`
    Name                      types.String       `tfsdk:"name"`
    Active                    types.Bool         `tfsdk:"active"`
    LastEditor                types.String       `tfsdk:"last_editor"`
    LastModified              types.String       `tfsdk:"last_modified"`
    CreatedAt                 types.String       `tfsdk:"created_at"`
    Infrastructure            types.Int64        `tfsdk:"infrastructure"`
    Tls                       *TLSWorkloadsModel `tfsdk:"tls"`
    Protocols                 *ProtocolsModel    `tfsdk:"protocols"`
    Mtls                      *MTLSModel         `tfsdk:"mtls"`
    Domains                   types.List         `tfsdk:"domains"`
    WorkloadDomainAllowAccess types.Bool         `tfsdk:"workload_domain_allow_access"`
    WorkloadDomain            types.String       `tfsdk:"workload_domain"`
    ProductVersion            types.String       `tfsdk:"product_version"`
}

type TLSWorkloadsModel struct {
    Certificate    types.Int64  `tfsdk:"certificate"`
    Ciphers        types.Int64  `tfsdk:"ciphers"`
    MinimumVersion types.String `tfsdk:"minimum_version"`
}
```

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`data_source_workload.go`) | Plural (`data_source_workloads.go`) |
|--------|--------------------------------------|-------------------------------------|
| **Metadata TypeName** | `req.ProviderTypeName + "_workload"` | `req.ProviderTypeName + "_workloads"` |
| **ID Attribute** | Required (user provides it) | Computed (generated after read) |
| **Primary Model Field** | `Data WorkloadResults` | `Results []WorkloadsResults` |
| **Additional Fields** | None | `Counter types.Int64` |
| **API Method** | `RetrieveWorkload(ctx, id).Execute()` | `ListWorkloads(ctx).Execute()` |
| **Response Type** | `*WorkloadResponse` | `*PaginatedWorkloadList` |
| **Data Extraction** | `workloadResponse.Data` | Loop through `workloadsResponse.Results` |

---

## Schema Definition Patterns

### IMPORTANT: Blocks in Resources, Nested Attributes in Data Sources

**For Resources:** every writable nested object is a **block** (`SingleNestedBlock`), never a nested attribute, so Terraform rejects unknown or misplaced arguments with "Unsupported argument" (see "Nested Objects Must Be Blocks" in `AGENTS.md`). `workload` is a required block (`objectvalidator.IsRequired()`, description starting with `Required.`); `tls`, `protocols` → `http` and `mtls` → `config` are optional blocks. Blocks cannot be `Computed` or carry a `Default`, so:

- **Request:** `workloadTLSRequest`, `workloadProtocolsRequest` and `workloadMTLSRequest` substitute the defaults (`defaultWorkloadTLS()`, `defaultWorkloadHTTP()`, `defaultWorkloadMTLS()`) for an omitted block, so the payload is the same as when these were Computed objects with object defaults.
- **State:** `keepDeclaredWorkloadBlocks(prior, results)` drops every block the prior plan/state does not declare (including `protocols.http` / `mtls.config` inside a declared parent). Nil prior (import) keeps everything; the first plan after import then removes undeclared blocks in place.
- Leaves inside a declared block keep `Optional + Computed (+ Default)` and keep mirroring the API, so drift is detected while the block is declared.

**For Data Sources:** Nested attributes use `Computed: true` since they are read-only.

### Resource Schema - TLS Block

```go
"tls": schema.SingleNestedBlock{
    Description: "TLS configuration for the workload. ...",
    Attributes: map[string]schema.Attribute{
        "certificate": schema.Int64Attribute{
            Description: "Certificate ID for TLS. Supplied by the API when not declared.",
            Optional:    true,
            Computed:    true,
        },
        "ciphers": schema.Int64Attribute{
            Description: "Cipher suite configuration. Supplied by the API when not declared.",
            Optional:    true,
            Computed:    true,
        },
        "minimum_version": schema.StringAttribute{
            Description: "Minimum TLS version: tls_1_0, tls_1_1, tls_1_2 or tls_1_3.",
            Optional:    true,
            Computed:    true,
            Default:     stringdefault.StaticString(defaultWorkloadTLSMinimumVersion),
        },
    },
},
```

### Data Source Schema - TLS Configuration (Computed for read-only)

```go
"tls": schema.SingleNestedAttribute{
    Description: "TLS configuration for the workload.",
    Computed:    true,  // Data sources use Computed
    Attributes: map[string]schema.Attribute{
        "certificate": schema.Int64Attribute{
            Description: "Certificate ID for TLS.",
            Computed:    true,
        },
        "ciphers": schema.Int64Attribute{
            Description: "Cipher suite configuration.",
            Computed:    true,
        },
        "minimum_version": schema.StringAttribute{
            Description: "Minimum TLS version.",
            Computed:    true,
        },
    },
},
```

### Protocols and MTLS Blocks (Resource)

`protocols` holds a nested `http` block; `mtls` holds `enabled` plus a nested `config` block. Same rules as `tls`: blocks without `Computed`/`Default`, leaves `Optional + Computed`, with `Default` only on enforced leaves (`http_ports` `[80]`, `https_ports` `[443]`, `mtls.enabled` `false`). `versions`, `quic_ports` and the `mtls.config` leaves have no default: the API supplies them. See `Schema()` in `internal/resource_workload.go`.

---

## Error Handling

### Standard Error Handling Pattern

```go
func (d *WorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // ... get ID and convert ...
    
    workloadResponse, response, err := d.client.api.WorkloadsAPI.
        RetrieveWorkload(ctx, workloadID).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            workloadResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadResponse, *http.Response, error) {
                return d.client.api.WorkloadsAPI.RetrieveWorkload(ctx, workloadID).Execute()
            }, 5) // Maximum 5 retries

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            usrMsg, errMsg := errPrintWorkload(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }
    // ... process response ...
}

func errPrintWorkload(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Workload found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

## Type Conversions

### Handling Nullable Fields

Workload SDK models use nullable types for optional fields. Here's how to handle them:

```go
// TLS Certificate (NullableInt64)
if workloadResponse.Data.Tls != nil && workloadResponse.Data.Tls.Certificate.IsSet() {
    cert := workloadResponse.Data.Tls.Certificate.Get()
    if cert != nil {
        tlsModel.Certificate = types.Int64Value(*cert)
    }
}

// MTLS Enabled (NullableBool)
if workloadResponse.Data.Mtls != nil && workloadResponse.Data.Mtls.Enabled.IsSet() {
    enabled := workloadResponse.Data.Mtls.Enabled.Get()
    if enabled != nil {
        mtlsModel.Enabled = types.BoolValue(*enabled)
    }
}

// MTLS Config Verification (NullableString)
if config.Verification.IsSet() {
    verif := config.Verification.Get()
    if verif != nil {
        configModel.Verification = types.StringValue(*verif)
    }
}
```

### Converting Lists with Null Handling

**CRITICAL:** Always provide a null fallback for list fields to avoid "MISSING TYPE" errors:

```go
// String list (Domains) - with null handling
if workloadResponse.Data.Domains != nil {
    domainsList, diags := types.ListValueFrom(ctx, types.StringType, workloadResponse.Data.Domains)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    workloadState.Data.Domains = domainsList
} else {
    workloadState.Data.Domains = types.ListNull(types.StringType)
}

// Int64 list (CRL) - with null handling
if config.Crl != nil {
    crlList, diags := types.ListValueFrom(ctx, types.Int64Type, config.Crl)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    configModel.Crl = crlList
} else {
    configModel.Crl = types.ListNull(types.Int64Type)
}

// HTTP Protocol lists - all need null handling
if httpProto.Versions != nil {
    versionsList, diags := types.ListValueFrom(ctx, types.StringType, httpProto.Versions)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    httpModel.Versions = versionsList
} else {
    httpModel.Versions = types.ListNull(types.StringType)
}

if httpProto.HttpPorts != nil {
    httpPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.HttpPorts)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    httpModel.HttpPorts = httpPortsList
} else {
    httpModel.HttpPorts = types.ListNull(types.Int64Type)
}

if httpProto.HttpsPorts != nil {
    httpsPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.HttpsPorts)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    httpModel.HttpsPorts = httpsPortsList
} else {
    httpModel.HttpsPorts = types.ListNull(types.Int64Type)
}

if httpProto.QuicPorts != nil {
    quicPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.QuicPorts)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
    httpModel.QuicPorts = quicPortsList
} else {
    httpModel.QuicPorts = types.ListNull(types.Int64Type)
}
```

### Time Formatting

```go
LastModified: types.StringValue(workloadResponse.Data.LastModified.Format(time.RFC850)),
```

---

## Common Issues

### 1. Unknown Value Errors with Nested Objects

`workloadResourceResults` reads nested objects into `*Struct` fields, which cannot hold unknown values ("Received unknown value, however the target type cannot handle unknown values"). With blocks this cannot happen: an omitted block is null, never unknown. Do not turn the blocks back into `Optional + Computed` nested attributes.

### 2. Inconsistent Result After Apply

```
Provider produced inconsistent result after apply
.workload.tls: block count changed from 0 to 1
```

An omitted block is planned as null, so the API echo must not put it back in state. Create, Read and Update pass the result through `keepDeclaredWorkloadBlocks(prior, results)` (prior = plan in Create/Update, state in Read). Apply it to every nested block, not only the top-level ones (`protocols.http`, `mtls.config`).

#### 2a. Empty `mtls.config` echo

When `mtls.config` is declared, the API may still echo an all-null config (see "The API echoes an empty `mtls.config`" below); `populateWorkloadResults` treats it as absent.

#### 2b. Inconsistent Result From Missing Setter in Create/Update

**Problem:** The plan has a concrete value for a nested field but the API returns a different value:
```
.workload.tls.minimum_version: was cty.StringVal("tls_1_2"), but now cty.StringVal("tls_1_3")
```

This happens when Create/Update builds the SDK request struct but forgets to call the setter for one of the fields. The API receives the request without that field, applies its default, and returns the default — which then disagrees with the plan.

**Solution:** Every optional field declared in the schema must have a matching `IsNull()/IsUnknown()`-gated `SetX(...)` call in **both** Create and Update. Missing setters are silent — the build passes, the plan looks fine, the bug only appears at apply time.

```go
// WRONG - schema exposes minimum_version but Create never sends it
if plan.Workload.Tls != nil {
    tls := azionapi.NewTLSWorkloadRequest()
    if !plan.Workload.Tls.Certificate.IsNull() && !plan.Workload.Tls.Certificate.IsUnknown() {
        tls.SetCertificate(plan.Workload.Tls.Certificate.ValueInt64())
    }
    if !plan.Workload.Tls.Ciphers.IsNull() && !plan.Workload.Tls.Ciphers.IsUnknown() {
        tls.SetCiphers(plan.Workload.Tls.Ciphers.ValueInt64())
    }
    // BUG: MinimumVersion is in the schema and model but never set on the request
    workload.SetTls(*tls)
}

// CORRECT - every schema field has a corresponding setter
if plan.Workload.Tls != nil {
    tls := azionapi.NewTLSWorkloadRequest()
    if !plan.Workload.Tls.Certificate.IsNull() && !plan.Workload.Tls.Certificate.IsUnknown() {
        tls.SetCertificate(plan.Workload.Tls.Certificate.ValueInt64())
    }
    if !plan.Workload.Tls.Ciphers.IsNull() && !plan.Workload.Tls.Ciphers.IsUnknown() {
        tls.SetCiphers(plan.Workload.Tls.Ciphers.ValueInt64())
    }
    if !plan.Workload.Tls.MinimumVersion.IsNull() && !plan.Workload.Tls.MinimumVersion.IsUnknown() {
        tls.SetMinimumVersion(plan.Workload.Tls.MinimumVersion.ValueString())
    }
    workload.SetTls(*tls)
}
```

When adding a field to a nested struct's schema, audit Create, Update, and `populateWorkloadResults` together — all three must handle it.

### 3. Nullable Types Not Checked Properly

**Problem:** Accessing nullable types without checking `IsSet()` and `Get()`.

**Wrong:**
```go
if workloadResponse.Data.Tls.Certificate != nil {
    tlsModel.Certificate = types.Int64Value(*workloadResponse.Data.Tls.Certificate)
}
```

**Correct:**
```go
if workloadResponse.Data.Tls.Certificate.IsSet() {
    cert := workloadResponse.Data.Tls.Certificate.Get()
    if cert != nil {
        tlsModel.Certificate = types.Int64Value(*cert)
    }
}
```

### 4. Missing Nested Model Types

**Problem:** Forgetting to define nested model types for complex objects.

**Solution:** Define separate model types for each nested structure:
- `TLSWorkloadResourceModel` / `TLSWorkloadModel`
- `ProtocolsResourceModel` / `ProtocolsModel`
- `HttpProtocolResourceModel` / `HttpProtocolModel`
- `MTLSResourceModel` / `MTLSModel`
- `MTLSConfigResourceModel` / `MTLSConfigModel`

### 5. List Type Mismatches

**Problem:** Using wrong element types for lists or missing null handling.

**Correct Mapping:**
| SDK Type | Terraform Type | Null Type |
|----------|----------------|-----------|
| `[]string` | `types.StringType` | `types.ListNull(types.StringType)` |
| `[]int64` | `types.Int64Type` | `types.ListNull(types.Int64Type)` |

### 5a. Choosing `types.List` vs `types.Set`

The resource's `domains` field uses `types.Set` (not `types.List`). Use a set whenever the collection's order has no meaning to the API and duplicates would be invalid — hostnames, tags, IDs returned in arbitrary order. Use a list when index position is part of the contract (HTTP versions returned in priority order, ports where order matters).

| Aspect | `types.List` | `types.Set` |
|--------|-------------|------------|
| Order significant | yes | no |
| Duplicates allowed | yes | no |
| Reorder triggers plan diff | yes | no |
| Schema attribute | `schema.ListAttribute` | `schema.SetAttribute` |
| Constructor | `types.ListValueFrom`, `types.ListNull` | `types.SetValueFrom`, `types.SetNull` |

`ElementsAs(ctx, &dst, false)` works identically for both, so Create/Update bodies don't change when switching types.

**Note on schema asymmetry:** Workload data sources still expose `domains` as `types.List` because they reflect API ordering directly. The resource uses `types.Set` to avoid spurious plan diffs when the API echoes domains in a different order than the user wrote them. Don't mechanically align these — they serve different purposes.

### 6. Provider Registration

**Problem:** Forgetting to register the data source in `provider.go`.

**Solution:** Add to the `DataSources()` function:
```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ... existing data sources ...
        dataSourceAzionWorkload,
        dataSourceAzionWorkloads,
    }
}
```

---

## SDK Model Reference

### Workload Model

```go
type Workload struct {
    Id int64 `json:"id"`
    Name string `json:"name"`
    Active *bool `json:"active,omitempty"`
    LastEditor string `json:"last_editor"`
    LastModified time.Time `json:"last_modified"`
    Infrastructure *int64 `json:"infrastructure,omitempty"`
    Tls *TLSWorkload `json:"tls,omitempty"`
    Protocols *Protocols `json:"protocols,omitempty"`
    Mtls *MTLS `json:"mtls,omitempty"`
    Domains []string `json:"domains,omitempty"`
    WorkloadDomainAllowAccess *bool `json:"workload_domain_allow_access,omitempty"`
    WorkloadDomain string `json:"workload_domain"`
    ProductVersion string `json:"product_version"`
}
```

### TLSWorkload Model

```go
type TLSWorkload struct {
    Certificate NullableInt64 `json:"certificate,omitempty"`
    Ciphers *int64 `json:"ciphers,omitempty"`
    MinimumVersion NullableTLSWorkloadMinimumVersion `json:"minimum_version,omitempty"`
}
```

### Protocols Model

```go
type Protocols struct {
    Http *HttpProtocol `json:"http,omitempty"`
}

type HttpProtocol struct {
    Versions []string `json:"versions,omitempty"`
    HttpPorts []int64 `json:"http_ports,omitempty"`
    HttpsPorts []int64 `json:"https_ports,omitempty"`
    QuicPorts []int64 `json:"quic_ports,omitempty"`
}
```

### MTLS Model

```go
type MTLS struct {
    Enabled NullableBool `json:"enabled,omitempty"`
    Config NullableMTLSConfig `json:"config,omitempty"`
}

type MTLSConfig struct {
    Certificate NullableInt64 `json:"certificate,omitempty"`
    Crl []int64 `json:"crl,omitempty"`
    Verification NullableString `json:"verification,omitempty"`
}
```

---

## Resource Implementation

For managing Workload resources (Create, Read, Update, Delete, Import):

**File:** `internal/resource_workload.go`

### Resource Model Structure

```go
type workloadResourceModel struct {
    Workload    *workloadResourceResults `tfsdk:"workload"`
    ID          types.String             `tfsdk:"id"`
}

type workloadResourceResults struct {
    ID                        types.Int64               `tfsdk:"id"`
    Name                      types.String              `tfsdk:"name"`
    Active                    types.Bool                `tfsdk:"active"`
    CreatedAt                 types.String              `tfsdk:"created_at"`
    Infrastructure            types.Int64               `tfsdk:"infrastructure"`
    Tls                       *TLSWorkloadResourceModel `tfsdk:"tls"`
    Protocols                 *ProtocolsResourceModel   `tfsdk:"protocols"`
    Mtls                      *MTLSResourceModel        `tfsdk:"mtls"`
    Domains                   types.Set                 `tfsdk:"domains"`
    WorkloadDomainAllowAccess types.Bool                `tfsdk:"workload_domain_allow_access"`
    WorkloadDomain            types.String              `tfsdk:"workload_domain"`
    ProductVersion            types.String              `tfsdk:"product_version"`
}

// Nested model types
type TLSWorkloadResourceModel struct {
    Certificate    types.Int64  `tfsdk:"certificate"`
    Ciphers        types.Int64  `tfsdk:"ciphers"`
    MinimumVersion types.String `tfsdk:"minimum_version"`
}

type ProtocolsResourceModel struct {
    Http *HttpProtocolResourceModel `tfsdk:"http"`
}

type HttpProtocolResourceModel struct {
    Versions   types.List `tfsdk:"versions"`
    HttpPorts  types.List `tfsdk:"http_ports"`
    HttpsPorts types.List `tfsdk:"https_ports"`
    QuicPorts  types.List `tfsdk:"quic_ports"`
}

type MTLSResourceModel struct {
    Enabled types.Bool               `tfsdk:"enabled"`
    Config  *MTLSConfigResourceModel `tfsdk:"config"`
}

type MTLSConfigResourceModel struct {
    Certificate  types.Int64  `tfsdk:"certificate"`
    Crl          types.List   `tfsdk:"crl"`
    Verification types.String `tfsdk:"verification"`
}
```

### Resource Schema Definition

The resource schema includes both configurable and computed attributes:

```go
func (r *workloadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Resource for managing Azion Workloads.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
        },
        Blocks: map[string]schema.Block{
            "workload": schema.SingleNestedBlock{
                Description: "Required. The workload settings.",
                Validators: []validator.Object{
                    objectvalidator.IsRequired(),
                },
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The workload identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the workload.",
                        Required:    true,
                    },
                    // ... other leaf attributes
                },
                Blocks: map[string]schema.Block{
                    // tls, protocols (-> http), mtls (-> config): SingleNestedBlock
                },
            },
        },
    }
}
```

### Create Operation Pattern

```go
func (r *workloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan workloadResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Create the workload request - name is REQUIRED
    workload := azionapi.NewWorkloadRequest(plan.Workload.Name.ValueString())

    // Set optional fields
    if !plan.Workload.Active.IsNull() && !plan.Workload.Active.IsUnknown() {
        workload.SetActive(plan.Workload.Active.ValueBool())
    }

    if !plan.Workload.Infrastructure.IsNull() && !plan.Workload.Infrastructure.IsUnknown() {
        workload.SetInfrastructure(plan.Workload.Infrastructure.ValueInt64())
    }

    // Handle TLS configuration - only if specified in plan
    if plan.Workload.Tls != nil {
        tls := azionapi.NewTLSWorkloadRequest()
        if !plan.Workload.Tls.Certificate.IsNull() && !plan.Workload.Tls.Certificate.IsUnknown() {
            tls.SetCertificate(plan.Workload.Tls.Certificate.ValueInt64())
        }
        if !plan.Workload.Tls.Ciphers.IsNull() && !plan.Workload.Tls.Ciphers.IsUnknown() {
            tls.SetCiphers(plan.Workload.Tls.Ciphers.ValueInt64())
        }
        if !plan.Workload.Tls.MinimumVersion.IsNull() && !plan.Workload.Tls.MinimumVersion.IsUnknown() {
            tls.SetMinimumVersion(plan.Workload.Tls.MinimumVersion.ValueString())
        }
        workload.SetTls(*tls)
    }

    // Handle Protocols configuration - only if specified in plan
    if plan.Workload.Protocols != nil && plan.Workload.Protocols.Http != nil {
        protocols := azionapi.NewProtocolsRequest()
        http := azionapi.NewHttpProtocolRequest()

        if !plan.Workload.Protocols.Http.Versions.IsNull() && !plan.Workload.Protocols.Http.Versions.IsUnknown() {
            var versions []string
            diags := plan.Workload.Protocols.Http.Versions.ElementsAs(ctx, &versions, false)
            resp.Diagnostics.Append(diags...)
            if resp.Diagnostics.HasError() {
                return
            }
            http.SetVersions(versions)
        }

        protocols.SetHttp(*http)
        workload.SetProtocols(*protocols)
    }

    // Handle MTLS configuration - only if specified in plan
    if plan.Workload.Mtls != nil {
        mtls := azionapi.NewMTLSRequest()
        if !plan.Workload.Mtls.Enabled.IsNull() && !plan.Workload.Mtls.Enabled.IsUnknown() {
            mtls.SetEnabled(plan.Workload.Mtls.Enabled.ValueBool())
        }

        if plan.Workload.Mtls.Config != nil {
            config := azionapi.NewMTLSConfigRequest()
            if !plan.Workload.Mtls.Config.Certificate.IsNull() && !plan.Workload.Mtls.Config.Certificate.IsUnknown() {
                config.SetCertificate(plan.Workload.Mtls.Config.Certificate.ValueInt64())
            }
            mtls.SetConfig(*config)
        }
        workload.SetMtls(*mtls)
    }

    // Handle Domains
    if !plan.Workload.Domains.IsNull() && !plan.Workload.Domains.IsUnknown() {
        var domains []string
        diags := plan.Workload.Domains.ElementsAs(ctx, &domains, false)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
        workload.SetDomains(domains)
    }

    // Execute create request with retry on 429
    createWorkload, response, err := r.client.api.WorkloadsAPI.CreateWorkload(ctx).WorkloadRequest(*workload).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            createWorkload, response, err = utils.RetryOn429(func() (*azionapi.WorkloadResponse, *http.Response, error) {
                return r.client.api.WorkloadsAPI.CreateWorkload(ctx).WorkloadRequest(*workload).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            return
        }
    }

    // Populate state from response, passing plan to preserve optional nested field values
    results := populateWorkloadResults(ctx, createWorkload)
    keepDeclaredWorkloadBlocks(plan.Workload, results)
    plan.Workload = results
    plan.ID = types.StringValue(strconv.FormatInt(createWorkload.Data.Id, 10))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Drift and enforcement

Read mirrors the API response through `populateWorkloadResults`, so a change made in Azion Console is reverted on the next apply instead of being silently kept. The only filter is `keepDeclaredWorkloadBlocks`, which drops whole blocks the configuration does not declare; it never filters values inside a declared block. **Do not gate leaf values on the plan**: that made Read a state-preserving merge, so out-of-band changes were invisible on every plan.

Perpetual drift is prevented in the schema instead — optional leaf attributes are `Optional + Computed`, either with a `Default` (enforced) or without one (the API supplies the value).

An omitted `tls`/`protocols`/`mtls` block still sends its defaults on every apply (request builders), but it is null in state, so drift in its fields is only shown while the block is declared — the same trade-off as `modules` in `azion_application_main_setting`.

### The API echoes an empty `mtls.config`

For a workload with no mTLS configured, the API returns `mtls.config` as an object whose every field is null, rather than omitting it. `populateWorkloadResults` treats that as **absent** via `MTLSConfigResourceModel.isEmpty()`.

Do not remove that check. `mtls.config` defaults to a null object in the schema, so writing the echoed object into state contradicts the plan and fails the apply with:

```
Provider produced inconsistent result after apply
.workload.mtls.config: was null, but now cty.ObjectVal(map[string]cty.Value{"certificate":cty.NullVal(cty.Number), ...})
```

The same rule applies to any block defaulting to a null object: an all-null echo carries no configuration, so treat it as absent. `TestPopulateWorkloadResultsIgnoresEmptyMTLSConfig` pins this, with a companion test asserting a config with real values still reaches state.

### Update Operation Pattern (PATCH)

Workload uses **partial update (PATCH)** via `PartialUpdateWorkload` — unlike the other main-settings resources in this provider, which use PUT.

**Do not "upgrade" this to `UpdateWorkload` (PUT).** A full `WorkloadRequest` carries a `bindings` field that this resource does not model; bindings belong to `azion_workload_deployment`. A PUT sends none of them and wipes the workload's binding to its deployment.

Enforcement does not depend on the verb here. Every field this resource models carries a schema default, so the plan holds a concrete value and the request always sends it — a console change to any modeled field is still reverted. PATCH only spares the fields the provider does not model, which is exactly what is wanted.

```go
func (r *workloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan workloadResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state workloadResourceModel
    diagsState := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsState...)
    if resp.Diagnostics.HasError() {
        return
    }

    workloadId := state.Workload.ID.ValueInt64()
    updateWorkloadRequest := azionapi.NewPatchedWorkloadRequest()

    // Set fields to update
    if !plan.Workload.Name.IsNull() && !plan.Workload.Name.IsUnknown() {
        updateWorkloadRequest.SetName(plan.Workload.Name.ValueString())
    }

    if !plan.Workload.Active.IsNull() && !plan.Workload.Active.IsUnknown() {
        updateWorkloadRequest.SetActive(plan.Workload.Active.ValueBool())
    }

    // ... set other fields similar to Create

    // Execute partial update
    updateWorkload, response, err := r.client.api.WorkloadsAPI.
        PartialUpdateWorkload(ctx, workloadId).
        PatchedWorkloadRequest(*updateWorkloadRequest).
        Execute()
    // ... error handling similar to Create

    // Populate state, passing plan to preserve optional nested field values
    results := populateWorkloadResults(ctx, updateWorkload)
    keepDeclaredWorkloadBlocks(plan.Workload, results)
    plan.Workload = results
    plan.ID = types.StringValue(strconv.FormatInt(updateWorkload.Data.Id, 10))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Operation Pattern

```go
func (r *workloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state workloadResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var workloadId int64
    var err error
    if state.Workload != nil {
        workloadId = state.Workload.ID.ValueInt64()
    } else {
        workloadId, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert Workload ID")
            return
        }
    }

    getWorkload, response, err := r.client.api.WorkloadsAPI.RetrieveWorkload(ctx, workloadId).Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // ... handle other errors
    }

    // Pass existing state to preserve optional nested field values
    results := populateWorkloadResults(ctx, getWorkload)
    keepDeclaredWorkloadBlocks(state.Workload, results)
    state.Workload = results
    state.ID = types.StringValue(strconv.FormatInt(getWorkload.Data.Id, 10))

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Delete Operation Pattern

```go
func (r *workloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state workloadResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    workloadId := state.Workload.ID.ValueInt64()

    _, response, err := r.client.api.WorkloadsAPI.DeleteWorkload(ctx, workloadId).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.WorkloadsAPI.DeleteWorkload(ctx, workloadId).Execute()
            }, 5)
            // ... error handling
        }
    }
}
```

### Populate Results Helper Function

`populateWorkloadResults(ctx, response)` mirrors the whole API response (no plan parameter); every caller then applies `keepDeclaredWorkloadBlocks(prior, results)` so blocks the configuration omits stay null. Leaf values inside declared blocks are never filtered. An all-null `mtls.config` echo is treated as absent (`MTLSConfigResourceModel.isEmpty()`). See `internal/resource_workload.go` for the current code.

### Import State Pattern

```go
func (r *workloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

---

## File Checklist

When implementing Workload data sources and resources, ensure these files are created/updated:

| File | Purpose |
|------|---------|
| `internal/data_source_workload.go` | Singular data source implementation |
| `internal/data_source_workloads.go` | Plural data source implementation |
| `internal/resource_workload.go` | Resource implementation (CRUD) |
| `internal/provider.go` | Register data sources and resources |
| `docs/data-sources/workload.md` | Documentation for singular data source |
| `docs/data-sources/workloads.md` | Documentation for plural data source |
| `docs/resources/workload.md` | Documentation for resource |
| `examples/data-sources/azion_workload/data-source.tf` | Example for singular data source |
| `examples/data-sources/azion_workloads/data-source.tf` | Example for plural data source |
| `examples/resources/azion_workload/resource.tf` | Example for resource |
| `examples/resources/azion_workload/import.sh` | Import command example |

---

## Naming Convention Note

**IMPORTANT:** Avoid using the "edge" prefix in internal Go code. While the Terraform resource names may still use `edge_` prefix for backwards compatibility, the internal naming should not use "edge":

| Terraform Name | Internal Go Name |
|-----------------|------------------|
| `azion_workload` | `WorkloadDataSource`, `workloadResource` |
| `azion_workloads` | `WorkloadsDataSource` |

This follows the V4 SDK convention which has removed the `edge` prefix from API naming.

---

## Key Implementation Notes

### 1. SDK Request Types

| Operation | SDK Type | Constructor |
|-----------|----------|-------------|
| Create | `WorkloadRequest` | `azionapi.NewWorkloadRequest(name)` |
| Update (PATCH) | `PatchedWorkloadRequest` | `azionapi.NewPatchedWorkloadRequest()` |
| TLS Config | `TLSWorkloadRequest` | `azionapi.NewTLSWorkloadRequest()` |
| Protocols Config | `ProtocolsRequest` | `azionapi.NewProtocolsRequest()` |
| HTTP Protocol | `HttpProtocolRequest` | `azionapi.NewHttpProtocolRequest()` |
| MTLS Config | `MTLSRequest` | `azionapi.NewMTLSRequest()` |
| MTLS Config Details | `MTLSConfigRequest` | `azionapi.NewMTLSConfigRequest()` |

### 2. Nullable Type Handling

The SDK uses `Nullable*` types for optional fields. Always check with `IsSet()` before accessing:

```go
// For NullableInt64, NullableBool, NullableString
if field.IsSet() {
    value := field.Get()
    if value != nil {
        // use *value
    }
}
```

### 3. Partial Update vs Full Update

Workload uses **PATCH** (`PartialUpdateWorkload`) for updates, not PUT. This means:
- Only fields that are set in the request will be updated
- Use `PatchedWorkloadRequest` type for updates
- Omitted fields remain unchanged on the server

### 4. Required vs Optional Fields

| Field | Create | Update |
|-------|--------|--------|
| `name` | Required | Optional |
| `active` | Optional | Optional |
| `infrastructure` | Optional | Optional |
| `tls` | Optional | Optional |
| `protocols` | Optional | Optional |
| `mtls` | Optional | Optional |
| `domains` | Optional | Optional |
| `workload_domain_allow_access` | Optional | Optional |

### 5. Keeping Undeclared Blocks Null

An omitted optional block must stay null in state: `keepDeclaredWorkloadBlocks` drops it from the API result. Values inside declared blocks are never filtered. This follows `keepDeclaredModules` in `resource_application_main_setting.go`.
