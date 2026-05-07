# DNSSEC - Code Generation Guide

This document provides specific guidance for implementing DNSSEC resources and data sources in the Terraform provider.

## Important: Data Source vs Resource Differences

The DNSSEC data source and resource have **different zone_id types**:

| Component | Zone ID Type | Reason |
|-----------|-------------|--------|
| **Data Source** | `types.Int64` | Used for reading - accepts numeric input directly |
| **Resource** | `types.String` | Used for import - string ID passed through from import state |

**Data Source Schema:**
```go
"zone_id": schema.Int64Attribute{
    Description: "The zone identifier to target for the resource.",
    Required:    true,
}
```

**Resource Schema:**
```go
"zone_id": schema.StringAttribute{
    Required:    true,
    Description: "The zone identifier to target for the resource.",
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
}
```

The resource uses `types.String` for `zone_id` because `ImportStatePassthroughID` passes the import ID as a string.

## Important: SDK Naming Convention

When implementing DNSSEC resources, **do not use the "edge" prefix** in variable names or struct fields. The V4 SDK (`azion-api`) uses cleaner naming without this prefix:

| Incorrect (with "edge" prefix) | Correct (V4 SDK) |
|-------------------------------|------------------|
| `edgeDnssecApi` | `dnssecApi` or `DNSDNSSECAPI` |
| `edgeZoneId` | `zoneId` |
| `edgeConfig` | `apiConfig` |

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by Zone ID)](#singular-data-source-read-by-zone-id)
3. [Resource Implementation](#resource-implementation)
   - [Create/Update Operation](#createupdate-operation)
   - [Read Operation](#read-operation)
   - [Delete Operation](#delete-operation)
   - [Import State](#import-state)
4. [Schema Definition Patterns](#schema-definition-patterns)
5. [Error Handling](#error-handling)
6. [Type Conversions](#type-conversions)
7. [Common Issues](#common-issues)
8. [File Generation Format](#file-generation-format)

---

## SDK Selection

DNSSEC uses the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| DNSSEC (Data Source) | `azion-api` (v4) | `api.DNSDNSSECAPI` | `https://api.azion.com/v4` |
| DNSSEC (Resource) | `azion-api` (v4) | `api.DNSDNSSECAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| Zone ID Type | `int64` |
| Update Request Type | `DNSSECRequest` |
| Patch Request Type | `PatchedDNSSECRequest` |
| Response Type | `DNSSECResponse` with `Data` field |
| Retrieve Pattern | `.RetrieveDnssec(ctx, zoneId).Execute()` |
| Update Pattern | `.UpdateDnssec(ctx, zoneId).DNSSECRequest(req).Execute()` |
| Patch Pattern | `.PartialUpdateDnssec(ctx, zoneId).PatchedDNSSECRequest(req).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### SDK Types

The V4 SDK provides the following types for DNSSEC:

```go
// DNSSECRequest - for creating/updating DNSSEC
type DNSSECRequest struct {
    Enabled bool `json:"enabled"`
}

// DNSSECResponse - the response from the API
type DNSSECResponse struct {
    Data DNSSEC `json:"data"`
}

// DNSSEC - the main DNSSEC data structure
type DNSSEC struct {
    Enabled           bool                      `json:"enabled"`
    Status            string                    `json:"status"`
    DelegationSigner  NullableDelegationSigner  `json:"delegation_signer"`
}

// DelegationSigner - nested object in DNSSEC
type DelegationSigner struct {
    AlgorithmType AlgType `json:"algorithm_type"`
    Digest        string  `json:"digest"`
    DigestType    AlgType `json:"digest_type"`
    KeyTag        int64   `json:"key_tag"`
}

// AlgType - algorithm or digest type
type AlgType struct {
    Id   int64  `json:"id"`
    Slug string `json:"slug"`
}
```

### Important: API Response "state" Field

The API returns an additional `state` field in the response that's not defined in the SDK's `DNSSECResponse` model. Since the SDK uses `DisallowUnknownFields()` in its JSON unmarshaler, parsing fails with:

```
json: unknown field "state"
```

**Workaround:** Define custom response types and parse the response manually:

```go
// Custom response type to handle the additional "state" field
type dnssecResponse struct {
    State string     `json:"state"`
    Data  dnssecData `json:"data"`
}

type dnssecData struct {
    Enabled          bool                  `json:"enabled"`
    Status           string                `json:"status"`
    DelegationSigner *delegationSignerData `json:"delegation_signer,omitempty"`
}

// Parse response manually
bodyBytes, err := io.ReadAll(response.Body)
if err != nil {
    // handle error
}

var dnssecResp dnssecResponse
if err := json.Unmarshal(bodyBytes, &dnssecResp); err != nil {
    // handle error
}

// Use dnssecResp.Data.Enabled, dnssecResp.Data.Status, etc.
```

---

## Data Source Implementation

### Singular Data Source (Read by Zone ID)

For reading DNSSEC configuration for a specific zone:

**File:** `internal/data_source_dnssec.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &dnssecDataSource{}
    _ datasource.DataSourceWithConfigure = &dnssecDataSource{}
)

func dataSourceAzionDNSSec() datasource.DataSource {
    return &dnssecDataSource{}
}

type dnssecDataSource struct {
    client *apiClient
}

type dnssecDataSourceModel struct {
    ID               types.String             `tfsdk:"id"`
    ZoneId           types.Int64              `tfsdk:"zone_id"`
    SchemaVersion    types.Int64              `tfsdk:"schema_version"`
    Dnssec           *dnssecDSModel          `tfsdk:"dnssec"`
    DelegationSigner *DelegationSignerDSModel `tfsdk:"delegation_signer"`
}

type dnssecDSModel struct {
    IsEnabled types.Bool   `tfsdk:"is_enabled"`
    Status    types.String `tfsdk:"status"`
}

type DelegationSignerDSModel struct {
    AlgorithmType *AlgTypeDS   `tfsdk:"algorithm_type"`
    Digest        types.String `tfsdk:"digest"`
    DigestType    *AlgTypeDS   `tfsdk:"digest_type"`
    KeyTag        types.Int64  `tfsdk:"key_tag"`
}

type AlgTypeDS struct {
    Id   types.Int64  `tfsdk:"id"`
    Slug types.String `tfsdk:"slug"`
}

func (d *dnssecDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *dnssecDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_intelligent_dns_dnssec"
}

func (d *dnssecDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getZoneId types.Int64
    diags := req.Config.GetAttribute(ctx, path.Root("zone_id"), &getZoneId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId := getZoneId.ValueInt64()

    getDnssec, response, err := d.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            getDnssec, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
                return d.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
            }, 5)

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
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(
                    errReadAll.Error(),
                    "err",
                )
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(
                err.Error(),
                bodyString,
            )
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    dnssecData := getDnssec.GetData()
    dnssecState := &dnssecDataSourceModel{
        ZoneId: getZoneId,
        ID:     types.StringValue("Get DNSSEC"),
        Dnssec: &dnssecDSModel{
            IsEnabled: types.BoolValue(dnssecData.GetEnabled()),
            Status:    types.StringValue(dnssecData.GetStatus()),
        },
    }

    if delegationSigner, ok := dnssecData.GetDelegationSignerOk(); ok && delegationSigner != nil {
        dnssecState.DelegationSigner = &DelegationSignerDSModel{
            AlgorithmType: &AlgTypeDS{
                Id:   types.Int64Value(delegationSigner.AlgorithmType.GetId()),
                Slug: types.StringValue(delegationSigner.AlgorithmType.GetSlug()),
            },
            Digest: types.StringValue(delegationSigner.GetDigest()),
            DigestType: &AlgTypeDS{
                Id:   types.Int64Value(delegationSigner.DigestType.GetId()),
                Slug: types.StringValue(delegationSigner.DigestType.GetSlug()),
            },
            KeyTag: types.Int64Value(delegationSigner.GetKeyTag()),
        }
    }

    diags = resp.State.Set(ctx, &dnssecState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

## Resource Implementation

### Create/Update Operation

DNSSEC uses a PUT operation for creating/updating. The zone must exist before enabling DNSSEC.

**File:** `internal/resource_dnssec.go`

```go
func (r *dnssecResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan dnssecResourceModel
    diags := req.Config.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    dnssecReq := azionapi.NewDNSSECRequest(plan.Dnssec.IsEnabled.ValueBool())

    dnssecResp, response, err := r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            dnssecResp, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
                return r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
            }, 5)

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
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(
                    errReadAll.Error(),
                    "err",
                )
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(
                err.Error(),
                bodyString,
            )
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Update state from response
    dnssecData := dnssecResp.GetData()
    plan.Dnssec = &dnssecModel{
        IsEnabled: types.BoolValue(dnssecData.GetEnabled()),
    }
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Read Operation

```go
func (r *dnssecResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state dnssecResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    dnssecResp, response, err := r.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle other errors...
    }

    if response != nil {
        defer response.Body.Close()
    }

    dnssecData := dnssecResp.GetData()
    state.Dnssec = &dnssecModel{
        IsEnabled: types.BoolValue(dnssecData.GetEnabled()),
    }

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Delete Operation

DNSSEC doesn't have a traditional delete. Instead, disable DNSSEC by setting `enabled` to `false`.

```go
func (r *dnssecResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state dnssecResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    dnssecReq := azionapi.NewDNSSECRequest(false)

    _, response, err := r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
    if err != nil {
        // Handle errors...
    }

    if response != nil {
        defer response.Body.Close()
    }
}
```

### Import State

```go
func (r *dnssecResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
```

---

## Schema Definition Patterns

### Data Source Schema

```go
func (d *dnssecDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Optional: true,
            },
            "zone_id": schema.Int64Attribute{
                Description: "The zone identifier to target for the resource.",
                Required:    true,
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Optional:    true,
            },
            "dnssec": schema.SingleNestedAttribute{
                Optional: true,
                Attributes: map[string]schema.Attribute{
                    "is_enabled": schema.BoolAttribute{
                        Optional:    true,
                        Description: "Zone DNSSEC flags for enabled.",
                    },
                    "status": schema.StringAttribute{
                        Optional:    true,
                        Description: "The status of the Zone DNSSEC.",
                    },
                },
            },
            "delegation_signer": schema.SingleNestedAttribute{
                Description: "Zone DNSSEC delegation signer.",
                Optional:    true,
                Attributes: map[string]schema.Attribute{
                    "algorithm_type": schema.SingleNestedAttribute{
                        Description: "Algorithm type for Zone DNSSEC.",
                        Optional:    true,
                        Attributes: map[string]schema.Attribute{
                            "id": schema.Int64Attribute{
                                Description: "The ID of this algorithm type.",
                                Optional:    true,
                            },
                            "slug": schema.StringAttribute{
                                Description: "The slug of this algorithm type.",
                                Optional:    true,
                            },
                        },
                    },
                    "digest": schema.StringAttribute{
                        Optional:    true,
                        Description: "Zone DNSSEC digest.",
                    },
                    "digest_type": schema.SingleNestedAttribute{
                        Description: "Digest type for Zone DNSSEC.",
                        Optional:    true,
                        Attributes: map[string]schema.Attribute{
                            "id": schema.Int64Attribute{
                                Description: "The ID of this digest type.",
                                Optional:    true,
                            },
                            "slug": schema.StringAttribute{
                                Description: "The slug of this digest type.",
                                Optional:    true,
                            },
                        },
                    },
                    "key_tag": schema.Int64Attribute{
                        Optional:    true,
                        Description: "Key Tag for the Zone DNSSEC.",
                    },
                },
            },
        },
    }
}
```

### Resource Schema

```go
func (r *dnssecResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "zone_id": schema.StringAttribute{
                Required:    true,
                Description: "The zone identifier to target for the resource.",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the order.",
                Computed:    true,
            },
            "dnssec": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "is_enabled": schema.BoolAttribute{
                        Required:    true,
                        Description: "Zone DNSSEC flags for enabled.",
                    },
                },
            },
        },
    }
}
```

---

## Error Handling

### Standard Error Handling Pattern

```go
if err != nil {
    // 1. Check for 429 (rate limiting)
    if response.StatusCode == 429 {
        result, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
            return d.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
        }, 5)
        
        if response != nil {
            defer response.Body.Close()
        }
        
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // 2. Read error body for details
        bodyBytes, errReadAll := io.ReadAll(response.Body)
        if errReadAll != nil {
            resp.Diagnostics.AddError(errReadAll.Error(), "err")
        }
        bodyString := string(bodyBytes)
        resp.Diagnostics.AddError(err.Error(), bodyString)
        return
    }
}

// Always close response body after successful API calls
if response != nil {
    defer response.Body.Close()
}
```

### Special Error Codes

```go
// For Read operations - handle 404 specially
if response.StatusCode == http.StatusNotFound {
    resp.State.RemoveResource(ctx)
    return
}
```

---

## Type Conversions

### SDK Types to Terraform Types

```go
// DNSSECResponse to Terraform model
dnssecData := getDnssec.GetData()

// Simple fields
IsEnabled: types.BoolValue(dnssecData.GetEnabled())
Status:    types.StringValue(dnssecData.GetStatus())

// Nested object with nullable check
if delegationSigner, ok := dnssecData.GetDelegationSignerOk(); ok && delegationSigner != nil {
    state.DelegationSigner = &DelegationSignerDSModel{
        AlgorithmType: &AlgTypeDS{
            Id:   types.Int64Value(delegationSigner.AlgorithmType.GetId()),
            Slug: types.StringValue(delegationSigner.AlgorithmType.GetSlug()),
        },
        Digest: types.StringValue(delegationSigner.GetDigest()),
        DigestType: &AlgTypeDS{
            Id:   types.Int64Value(delegationSigner.DigestType.GetId()),
            Slug: types.StringValue(delegationSigner.DigestType.GetSlug()),
        },
        KeyTag: types.Int64Value(delegationSigner.GetKeyTag()),
    }
}
```

### Terraform Types to SDK Types

```go
// Create DNSSECRequest from plan
dnssecReq := azionapi.NewDNSSECRequest(plan.Dnssec.IsEnabled.ValueBool())
```

---

## Common Issues

### Issue 1: Using Legacy SDK Instead of V4

**Problem:** Using the legacy `idns` SDK instead of `azion-api` (V4).

**Solution:** Always use the V4 SDK:
```go
// CORRECT - V4 SDK
d.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()

// INCORRECT - Legacy SDK
d.client.idnsApi.DNSSECAPI.GetZoneDnsSec(ctx, zoneID32).Execute()
```

### Issue 2: ID Type Mismatch

**Problem:** V4 SDK uses `int64` for zone IDs, not `int32`.

**Solution:** Use `int64` directly:
```go
// CORRECT - V4 SDK uses int64
zoneId := getZoneId.ValueInt64()

// INCORRECT - Legacy SDK used int32
zoneID32, err := utils.CheckInt64toInt32Security(getZoneId.ValueInt64())
```

### Issue 3: Field Name Differences

**Problem:** V4 SDK uses different field names than legacy SDK.

**Solution:** Use the correct V4 field names:
```go
// CORRECT - V4 SDK field names
dnssecData.GetEnabled()           // NOT GetIsEnabled()
dnssecData.GetStatus()

// Nested object access
delegationSigner.AlgorithmType    // NOT Algorithmtype
delegationSigner.DigestType       // NOT Digesttype
delegationSigner.Digest
delegationSigner.KeyTag           // NOT Keytag
```

### Issue 4: Response Body Not Closed

**Problem:** Forgetting to close the HTTP response body.

**Solution:** Always close response body after successful API calls:
```go
if response != nil {
    defer response.Body.Close()
}
```

### Issue 5: Nullable DelegationSigner Handling

**Problem:** `DelegationSigner` can be `nil` when DNSSEC is not fully configured.

**Solution:** Use the `Ok` pattern for nullable fields:
```go
if delegationSigner, ok := dnssecData.GetDelegationSignerOk(); ok && delegationSigner != nil {
    // Process delegation signer
}
```

---

## File Generation Format

When generating DNSSEC resources and data sources, follow these file organization patterns:

### File Structure

```
terraform-provider-azion/
├── internal/
│   ├── resource_dnssec.go        # Resource implementation
│   └── data_source_dnssec.go     # Data source implementation
├── docs/
│   ├── resources/
│   │   └── intelligent_dns_dnssec.md    # Resource documentation
│   └── data-sources/
│       └── intelligent_dns_dnssec.md    # Data source documentation
└── examples/
    ├── resources/
    │   └── azion_intelligent_dns_dnssec/
    │       ├── resource.tf        # Example usage
    │       └── import.sh          # Import example
    └── data-sources/
        └── azion_intelligent_dns_dnssec/
            └── data-source.tf     # Data source example
```

### Naming Conventions

| Component | File Name | Terraform Name |
|-----------|-----------|----------------|
| Resource | `resource_dnssec.go` | `azion_intelligent_dns_dnssec` |
| Data Source | `data_source_dnssec.go` | `azion_intelligent_dns_dnssec` |
| Documentation | `intelligent_dns_dnssec.md` | N/A |
| Examples Dir | `azion_intelligent_dns_dnssec/` | N/A |

### Model Struct Naming

```go
// Resource model - note zone_id is types.String for import support
type dnssecResourceModel struct {
    ZoneId        types.String `tfsdk:"zone_id"`
    SchemaVersion types.Int64  `tfsdk:"schema_version"`
    Dnssec        *dnssecModel `tfsdk:"dnssec"`
    LastUpdated   types.String `tfsdk:"last_updated"`
}

type dnssecModel struct {
    IsEnabled types.Bool `tfsdk:"is_enabled"`
}

// Data source model - note zone_id is types.Int64 for numeric input
type dnssecDataSourceModel struct {
    ID               types.String             `tfsdk:"id"`
    ZoneId           types.Int64              `tfsdk:"zone_id"`  // Int64 for data source
    SchemaVersion    types.Int64              `tfsdk:"schema_version"`
    Dnssec           *dnssecDSModel           `tfsdk:"dnssec"`
    DelegationSigner *DelegationSignerDSModel `tfsdk:"delegation_signer"`
}

type dnssecDSModel struct {
    IsEnabled types.Bool   `tfsdk:"is_enabled"`
    Status    types.String `tfsdk:"status"`
}

type DelegationSignerDSModel struct {
    AlgorithmType *AlgTypeDS   `tfsdk:"algorithm_type"`
    Digest        types.String `tfsdk:"digest"`
    DigestType    *AlgTypeDS   `tfsdk:"digest_type"`
    KeyTag        types.Int64  `tfsdk:"key_tag"`
}

type AlgTypeDS struct {
    Id   types.Int64  `tfsdk:"id"`
    Slug types.String `tfsdk:"slug"`
}
```

### Required Imports

```go
import (
    "context"
    "io"
    "net/http"
    "strconv"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)
```

### Provider Registration

Register in `internal/provider.go`:

```go
func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewDnssecResource,
        // ... other resources
    }
}

func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionDNSSec,
        // ... other data sources
    }
}
```

---

## Documentation and Examples

### MANDATORY: Parent Resource Documentation

**IMPORTANT**: DNSSEC is a child resource of `azion_intelligent_dns_zone`. Documentation and examples MUST include the parent resource creation to show complete context.

When updating documentation, always include:

1. **Parent Zone Example** - Show creation of the parent DNS zone first
2. **Reference Using Terraform Interpolation** - Use `azion_intelligent_dns_zone.example.id` to reference the parent ID

### Documentation Files

Documentation is auto-generated by `terraform-plugin-docs` and located in:

| Type | Location |
|------|----------|
| Data Source Doc | `docs/data-sources/intelligent_dns_dnssec.md` |
| Resource Doc | `docs/resources/intelligent_dns_dnssec.md` |

### Example Files

Example Terraform configurations are located in:

| Type | Location |
|------|----------|
| Data Source Example | `examples/data-sources/azion_intelligent_dns_dnssec/data-source.tf` |
| Resource Example | `examples/resources/azion_intelligent_dns_dnssec/resource.tf` |

### Example: Complete Resource Usage with Parent Zone

```terraform
# First, create the parent DNS zone
resource "azion_intelligent_dns_zone" "example" {
  zone = {
    name    = "example.com"
    active  = true
    domain  = "example.com"
  }
}

# Then configure DNSSEC for that zone
resource "azion_intelligent_dns_dnssec" "example" {
  zone_id = azion_intelligent_dns_zone.example.id
  dnssec = {
    is_enabled = true
  }
}
```

---

## Summary Checklist

When implementing DNSSEC resources or data sources:

1. **Use V4 SDK**: `azion-api` package with `api.DNSDNSSECAPI`
2. **No "edge" prefix**: Use `dnssecApi`, `zoneId`, `apiConfig` naming (not `edgeDnssecApi`, `edgeZoneId`, `edgeConfig`)
3. **ID types**: Use `int64` for zone IDs (no conversion needed)
4. **Response handling**: Response has `Data` field containing `DNSSEC` struct
5. **Nested objects**: `DelegationSigner` contains `AlgorithmType` and `DigestType` (both `AlgType`)
6. **Handle 429 errors**: Use `utils.RetryOn429`
7. **Close response bodies**: Add `defer response.Body.Close()` after successful calls
8. **Nullable fields**: Use `GetFieldOk()` for nullable nested objects
9. **Field names**: `Enabled` (not `IsEnabled`), `algorithm_type`, `digest_type`, `key_tag`
10. **Register in provider.go**: Add to `DataSources()` or `Resources()`
11. **Generate documentation**: Create docs and examples
