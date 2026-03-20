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

### IMPORTANT: Nested Attributes in Resources vs Data Sources

**For Resources:** Nested `SingleNestedAttribute` attributes must use `Optional: true` **without** `Computed: true`. This prevents Terraform from sending unknown values that concrete struct pointers cannot handle.

**For Data Sources:** Nested attributes use `Computed: true` since they are read-only.

### Resource Schema - TLS Configuration (Optional only, NO Computed)

```go
"tls": schema.SingleNestedAttribute{
    Description: "TLS configuration for the workload.",
    Optional:    true,  // NO Computed: true!
    Attributes: map[string]schema.Attribute{
        "certificate": schema.Int64Attribute{
            Description: "Certificate ID for TLS.",
            Optional:    true,
        },
        "ciphers": schema.Int64Attribute{
            Description: "Cipher suite configuration.",
            Optional:    true,
        },
        "minimum_version": schema.StringAttribute{
            Description: "Minimum TLS version.",
            Optional:    true,
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

### Protocols Configuration Schema (Resource - Optional only)

```go
"protocols": schema.SingleNestedAttribute{
    Description: "Protocol configurations for the workload.",
    Optional:    true,  // NO Computed: true!
    Attributes: map[string]schema.Attribute{
        "http": schema.SingleNestedAttribute{
            Description: "HTTP protocol configuration.",
            Optional:    true,
            Attributes: map[string]schema.Attribute{
                "versions": schema.ListAttribute{
                    ElementType: types.StringType,
                    Description: "HTTP versions supported.",
                    Optional:    true,
                },
                "http_ports": schema.ListAttribute{
                    ElementType: types.Int64Type,
                    Description: "HTTP ports.",
                    Optional:    true,
                },
                "https_ports": schema.ListAttribute{
                    ElementType: types.Int64Type,
                    Description: "HTTPS ports.",
                    Optional:    true,
                },
                "quic_ports": schema.ListAttribute{
                    ElementType: types.Int64Type,
                    Description: "QUIC ports.",
                    Optional:    true,
                },
            },
        },
    },
},
```

### MTLS Configuration Schema (Resource - Optional only)

```go
"mtls": schema.SingleNestedAttribute{
    Description: "Mutual TLS configuration for the workload.",
    Optional:    true,  // NO Computed: true!
    Attributes: map[string]schema.Attribute{
        "enabled": schema.BoolAttribute{
            Description: "Whether MTLS is enabled.",
            Optional:    true,
        },
        "config": schema.SingleNestedAttribute{
            Description: "MTLS configuration.",
            Optional:    true,
            Attributes: map[string]schema.Attribute{
                "certificate": schema.Int64Attribute{
                    Description: "MTLS certificate ID.",
                    Optional:    true,
                },
                "crl": schema.ListAttribute{
                    ElementType: types.Int64Type,
                    Description: "Certificate Revocation List.",
                    Optional:    true,
                },
                "verification": schema.StringAttribute{
                    Description: "MTLS verification type: enforce or permissive.",
                    Optional:    true,
                },
            },
        },
    },
},
```

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

### 1. Unknown Value Errors with Nested Attributes

**Problem:** Using `Computed: true` on `SingleNestedAttribute` with struct pointer types causes errors:
```
Received unknown value, however the target type cannot handle unknown values.
Use the corresponding `types` package type or a custom type that handles unknown values.
```

**Solution:** Remove `Computed: true` from nested attributes in resources. Use `Optional: true` only:
```go
// WRONG - causes unknown value errors
"tls": schema.SingleNestedAttribute{
    Optional: true,
    Computed: true,  // <-- REMOVE THIS
    Attributes: map[string]schema.Attribute{...},
}

// CORRECT - use Optional only for nested attributes in resources
"tls": schema.SingleNestedAttribute{
    Optional: true,
    Attributes: map[string]schema.Attribute{...},
}
```

### 2. Inconsistent Result After Apply

**Problem:** When the API returns default values for nested fields that were `null` in the plan:
```
Provider produced inconsistent result after apply
.workload.tls: was null, but now cty.ObjectVal(...)
```

**Solution:** Only populate nested fields from API response if they were specified in the plan. Modify `populateWorkloadResults` to accept the plan:

```go
func populateWorkloadResults(response *azionapi.WorkloadResponse, plan *workloadResourceResults) *workloadResourceResults {
    result := &workloadResourceResults{
        // ... basic fields ...
    }

    // Handle TLS - only populate from API if it was specified in the plan
    if plan.Tls != nil && response.Data.Tls != nil {
        tlsModel := &TLSWorkloadResourceModel{}
        // ... populate tlsModel ...
        result.Tls = tlsModel
    }
    // If plan.Tls was nil, result.Tls stays nil (not populated from API defaults)

    // Same pattern for Protocols and MTLS
    if plan.Protocols != nil && response.Data.Protocols != nil {
        // ... populate protocols ...
    }

    if plan.Mtls != nil && response.Data.Mtls != nil {
        // ... populate mtls ...
    }

    return result
}
```

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
    LastUpdated types.String             `tfsdk:"last_updated"`
}

type workloadResourceResults struct {
    ID                        types.Int64               `tfsdk:"id"`
    Name                      types.String              `tfsdk:"name"`
    Active                    types.Bool                `tfsdk:"active"`
    LastEditor                types.String              `tfsdk:"last_editor"`
    LastModified              types.String              `tfsdk:"last_modified"`
    Infrastructure            types.Int64               `tfsdk:"infrastructure"`
    Tls                       *TLSWorkloadResourceModel `tfsdk:"tls"`
    Protocols                 *ProtocolsResourceModel   `tfsdk:"protocols"`
    Mtls                      *MTLSResourceModel        `tfsdk:"mtls"`
    Domains                   types.List                `tfsdk:"domains"`
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
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "workload": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The workload identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the workload.",
                        Required:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Status of the workload.",
                        Optional:    true,
                        Computed:    true,
                    },
                    // tls, protocols, mtls - Optional only (NO Computed)
                    // ... other attributes
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
    plan.Workload = populateWorkloadResults(createWorkload, plan.Workload)
    plan.ID = types.StringValue(strconv.FormatInt(createWorkload.Data.Id, 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Update Operation Pattern (PATCH)

Workload uses **partial update (PATCH)** via `PartialUpdateWorkload`:

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
    plan.Workload = populateWorkloadResults(updateWorkload, plan.Workload)
    plan.ID = types.StringValue(strconv.FormatInt(updateWorkload.Data.Id, 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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
    state.Workload = populateWorkloadResults(getWorkload, state.Workload)
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

**CRITICAL:** This function must accept the plan and only populate nested structures if they were specified in the plan. This prevents "inconsistent result after apply" errors when the API returns default values.

```go
// populateWorkloadResults populates workload results from API response.
// plan is used to preserve optional nested field values - if a nested field was null in the plan,
// it stays null in the result to avoid "Provider produced inconsistent result after apply" errors.
func populateWorkloadResults(response *azionapi.WorkloadResponse, plan *workloadResourceResults) *workloadResourceResults {
    result := &workloadResourceResults{
        ID:             types.Int64Value(response.Data.Id),
        Name:           types.StringValue(response.Data.Name),
        LastEditor:     types.StringValue(response.Data.LastEditor),
        LastModified:   types.StringValue(response.Data.LastModified.Format(time.RFC850)),
        ProductVersion: types.StringValue(response.Data.ProductVersion),
        WorkloadDomain: types.StringValue(response.Data.WorkloadDomain),
    }

    if response.Data.Active != nil {
        result.Active = types.BoolValue(*response.Data.Active)
    }

    if response.Data.Infrastructure != nil {
        result.Infrastructure = types.Int64Value(*response.Data.Infrastructure)
    }

    if response.Data.WorkloadDomainAllowAccess != nil {
        result.WorkloadDomainAllowAccess = types.BoolValue(*response.Data.WorkloadDomainAllowAccess)
    }

    // Handle TLS - only populate from API if it was specified in the plan
    if plan.Tls != nil && response.Data.Tls != nil {
        tlsModel := &TLSWorkloadResourceModel{}
        if response.Data.Tls.Certificate.IsSet() {
            cert := response.Data.Tls.Certificate.Get()
            if cert != nil {
                tlsModel.Certificate = types.Int64Value(*cert)
            }
        }
        if response.Data.Tls.Ciphers != nil {
            tlsModel.Ciphers = types.Int64Value(*response.Data.Tls.Ciphers)
        }
        if response.Data.Tls.MinimumVersion.IsSet() {
            minVer := response.Data.Tls.MinimumVersion.Get()
            if minVer != nil {
                tlsModel.MinimumVersion = types.StringValue(*minVer)
            }
        }
        result.Tls = tlsModel
    }
    // If plan.Tls was nil, result.Tls stays nil (not populated from API defaults)

    // Handle Protocols - only populate from API if it was specified in the plan
    if plan.Protocols != nil && response.Data.Protocols != nil {
        protocolsModel := &ProtocolsResourceModel{}
        if response.Data.Protocols.Http != nil {
            httpModel := &HttpProtocolResourceModel{}
            if response.Data.Protocols.Http.Versions != nil {
                versionsList, _ := types.ListValueFrom(context.Background(), types.StringType, response.Data.Protocols.Http.Versions)
                httpModel.Versions = versionsList
            } else {
                httpModel.Versions = types.ListNull(types.StringType)
            }
            if response.Data.Protocols.Http.HttpPorts != nil {
                httpPortsList, _ := types.ListValueFrom(context.Background(), types.Int64Type, response.Data.Protocols.Http.HttpPorts)
                httpModel.HttpPorts = httpPortsList
            } else {
                httpModel.HttpPorts = types.ListNull(types.Int64Type)
            }
            if response.Data.Protocols.Http.HttpsPorts != nil {
                httpsPortsList, _ := types.ListValueFrom(context.Background(), types.Int64Type, response.Data.Protocols.Http.HttpsPorts)
                httpModel.HttpsPorts = httpsPortsList
            } else {
                httpModel.HttpsPorts = types.ListNull(types.Int64Type)
            }
            if response.Data.Protocols.Http.QuicPorts != nil {
                quicPortsList, _ := types.ListValueFrom(context.Background(), types.Int64Type, response.Data.Protocols.Http.QuicPorts)
                httpModel.QuicPorts = quicPortsList
            } else {
                httpModel.QuicPorts = types.ListNull(types.Int64Type)
            }
            protocolsModel.Http = httpModel
        }
        result.Protocols = protocolsModel
    }

    // Handle MTLS - only populate from API if it was specified in the plan
    if plan.Mtls != nil && response.Data.Mtls != nil {
        mtlsModel := &MTLSResourceModel{}
        if response.Data.Mtls.Enabled.IsSet() {
            enabled := response.Data.Mtls.Enabled.Get()
            if enabled != nil {
                mtlsModel.Enabled = types.BoolValue(*enabled)
            }
        }
        if response.Data.Mtls.Config.IsSet() {
            config := response.Data.Mtls.Config.Get()
            if config != nil {
                configModel := &MTLSConfigResourceModel{}
                if config.Certificate.IsSet() {
                    cert := config.Certificate.Get()
                    if cert != nil {
                        configModel.Certificate = types.Int64Value(*cert)
                    }
                }
                if config.Crl != nil {
                    crlList, _ := types.ListValueFrom(context.Background(), types.Int64Type, config.Crl)
                    configModel.Crl = crlList
                } else {
                    configModel.Crl = types.ListNull(types.Int64Type)
                }
                if config.Verification.IsSet() {
                    verif := config.Verification.Get()
                    if verif != nil {
                        configModel.Verification = types.StringValue(*verif)
                    }
                }
                mtlsModel.Config = configModel
            }
        }
        result.Mtls = mtlsModel
    }

    // Handle Domains
    if response.Data.Domains != nil {
        domainsList, _ := types.ListValueFrom(context.Background(), types.StringType, response.Data.Domains)
        result.Domains = domainsList
    } else {
        result.Domains = types.ListNull(types.StringType)
    }

    return result
}
```

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

### 5. Preserving Optional Nested Field Values

**CRITICAL:** When populating results from API responses, always check if the nested field was specified in the plan. If it was `nil` in the plan, keep it `nil` in the result to avoid "Provider produced inconsistent result after apply" errors.

This pattern follows the same approach used in `edge_application_main_setting.go` for the `modules` attribute.
