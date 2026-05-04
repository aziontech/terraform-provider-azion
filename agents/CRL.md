# Certificate Revocation Lists (CRL) - Code Generation Guide

This document provides specific guidance for implementing Certificate Revocation List (CRL) resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
3. [Resource Implementation](#resource-implementation)
4. [Schema Definition Patterns](#schema-definition-patterns)
5. [Error Handling](#error-handling)
6. [Type Conversions](#type-conversions)
7. [Common Issues](#common-issues)

---

## SDK Selection

Certificate Revocation Lists use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| CRL (Singular Data Source) | `azion-api` (v4) | `api.DigitalCertificatesCertificateRevocationListsAPI` | `https://api.azion.com/v4` |
| CRLs (Plural Data Source) | `azion-api` (v4) | `api.DigitalCertificatesCertificateRevocationListsAPI` | `https://api.azion.com/v4` |
| CRL (Resource) | `azion-api` (v4) | `api.DigitalCertificatesCertificateRevocationListsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `CertificateRevocationList` |
| Update Request Type | `CertificateRevocationList` (PUT) or `PatchedCertificateRevocationList` (PATCH) |
| Response Type | `CertificateRevocationListResponse` with `Data` field |
| List Response Type | `PaginatedCertificateRevocationList` |
| Create Pattern | `.CreateCertificateRevocationList(ctx).CertificateRevocationList(crl).Execute()` |
| Update Pattern | `.UpdateCertificateRevocationList(ctx, id).CertificateRevocationList(crl).Execute()` |
| Retrieve Pattern | `.RetrieveCertificateRevocationList(ctx, crlId).Execute()` |
| List Method | `.ListCertificateRevocationLists(ctx).Execute()` |
| Delete Method | `.DeleteCertificateRevocationList(ctx, id).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azion-api) - preferred for all implementations
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
}
```

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Certificate Revocation List by its identifier:

**File:** `internal/data_source_crl.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
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
    _ datasource.DataSource              = &CrlDataSource{}
    _ datasource.DataSourceWithConfigure = &CrlDataSource{}
)

// Constructor function
func dataSourceAzionCrl() datasource.DataSource {
    return &CrlDataSource{}
}

// DataSource struct
type CrlDataSource struct {
    client *apiClient
}

// State model
type CrlDataSourceModel struct {
    ID            types.String      `tfsdk:"id"`
    SchemaVersion types.Int64       `tfsdk:"schema_version"`
    Results       *CrlResultsModel `tfsdk:"results"`
    CrlID         types.Int64       `tfsdk:"crl_id"`
}

// Results model
type CrlResultsModel struct {
    ID             types.Int64  `tfsdk:"id"`
    Name           types.String `tfsdk:"name"`
    Active         types.Bool   `tfsdk:"active"`
    LastEditor     types.String `tfsdk:"last_editor"`
    CreatedAt      types.String `tfsdk:"created_at"`
    LastModified   types.String `tfsdk:"last_modified"`
    ProductVersion types.String `tfsdk:"product_version"`
    Issuer         types.String `tfsdk:"issuer"`
    LastUpdate     types.String `tfsdk:"last_update"`
    NextUpdate     types.String `tfsdk:"next_update"`
    Crl            types.String `tfsdk:"crl"`
}
```

### Plural Data Source (List Multiple Resources)

For listing multiple Certificate Revocation Lists:

**File:** `internal/data_source_crls.go`

```go
package provider

import (
    "context"
    "io"
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
    _ datasource.DataSource              = &CrlsDataSource{}
    _ datasource.DataSourceWithConfigure = &CrlsDataSource{}
)

// Constructor function
func dataSourceAzionCrls() datasource.DataSource {
    return &CrlsDataSource{}
}

// DataSource struct
type CrlsDataSource struct {
    client *apiClient
}

// State model with pagination
type CrlsDataSourceModel struct {
    ID            types.String       `tfsdk:"id"`
    Counter       types.Int64        `tfsdk:"counter"`
    TotalPages    types.Int64        `tfsdk:"total_pages"`
    Page          types.Int64        `tfsdk:"page"`
    PageSize      types.Int64        `tfsdk:"page_size"`
    Links         *CrlLinksModel     `tfsdk:"links"`
    SchemaVersion types.Int64        `tfsdk:"schema_version"`
    Results       []CrlsResultModel  `tfsdk:"results"`
}

// Links model for pagination
type CrlLinksModel struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

// Results model for each CRL in the list
type CrlsResultModel struct {
    ID             types.Int64  `tfsdk:"id"`
    Name           types.String `tfsdk:"name"`
    Active         types.Bool   `tfsdk:"active"`
    LastEditor     types.String `tfsdk:"last_editor"`
    CreatedAt      types.String `tfsdk:"created_at"`
    LastModified   types.String `tfsdk:"last_modified"`
    ProductVersion types.String `tfsdk:"product_version"`
    Issuer         types.String `tfsdk:"issuer"`
    LastUpdate     types.String `tfsdk:"last_update"`
    NextUpdate     types.String `tfsdk:"next_update"`
    Crl            types.String `tfsdk:"crl"`
}
```

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`crl`) | Plural (`crls`) |
|--------|------------------|-----------------|
| **Purpose** | Read a specific CRL by ID | List all CRLs with pagination |
| **Input** | `crl_id` (int64, required) | No required input |
| **Output** | Single `results` object | Array of `results` objects |
| **Pagination** | No pagination fields | Includes `counter`, `total_pages`, `page`, `page_size`, `links` |
| **API Method** | `RetrieveCertificateRevocationList` | `ListCertificateRevocationLists` |
| **Response Type** | `CertificateRevocationListResponse` | `PaginatedCertificateRevocationList` |
| **Results Type** | `*CrlResultsModel` (pointer to single object) | `[]CrlsResultModel` (slice of objects) |

---

## Resource Implementation

For implementing a full resource with CRUD operations:

**File:** `internal/resource_crl.go`

### Model Struct

The resource uses a nested `crl` attribute to group all CRL-specific fields, following the provider's standard pattern:

```go
type crlResourceModel struct {
    Crl         *crlResourceResults `tfsdk:"crl"`
    ID          types.String        `tfsdk:"id"`
    LastUpdated types.String        `tfsdk:"last_updated"`
}

type crlResourceResults struct {
    ID             types.Int64  `tfsdk:"id"`
    Name           types.String `tfsdk:"name"`
    Active         types.Bool   `tfsdk:"active"`
    LastEditor     types.String `tfsdk:"last_editor"`
    CreatedAt      types.String `tfsdk:"created_at"`
    LastModified   types.String `tfsdk:"last_modified"`
    ProductVersion types.String `tfsdk:"product_version"`
    Issuer         types.String `tfsdk:"issuer"`
    LastUpdate     types.String `tfsdk:"last_update"`
    NextUpdate     types.String `tfsdk:"next_update"`
    Crl            types.String `tfsdk:"crl"`
}
```

### Resource Schema

```go
func (r *crlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Creates a Certificate Revocation List (CRL) resource.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the certificate revocation list.",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "crl": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "Identifier of the certificate revocation list.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the certificate revocation list.",
                        Required:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Indicates if the certificate revocation list is active. This field cannot be set to false.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor of the certificate revocation list.",
                        Computed:    true,
                    },
                    "created_at": schema.StringAttribute{
                        Description: "Timestamp of the certificate revocation list creation on the platform.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Timestamp of the last modification made to the certificate content on the platform.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version of the certificate revocation list.",
                        Computed:    true,
                    },
                    "issuer": schema.StringAttribute{
                        Description: "Issuer of the certificate revocation list.",
                        Required:    true,
                    },
                    "last_update": schema.StringAttribute{
                        Description: "Timestamp of the last update issued by the certification revocation list issuer.",
                        Computed:    true,
                    },
                    "next_update": schema.StringAttribute{
                        Description: "Timestamp of the next scheduled update from the certification revocation list issuer.",
                        Computed:    true,
                    },
                    "crl": schema.StringAttribute{
                        Description: "The certificate revocation list content.",
                        Required:    true,
                    },
                },
            },
        },
    }
}
```

### Create Operation

```go
func (r *crlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan crlResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the request using NewCertificateRevocationListWithDefaults
    // This is preferred over NewCertificateRevocationList to avoid passing all parameters
    crlRequest := azionapi.NewCertificateRevocationListWithDefaults()
    crlRequest.SetName(plan.Crl.Name.ValueString())
    crlRequest.SetIssuer(plan.Crl.Issuer.ValueString())
    crlRequest.SetCrl(plan.Crl.Crl.ValueString())
    crlRequest.SetLastModified(time.Now())
    crlRequest.SetLastUpdate(time.Now())
    crlRequest.SetNextUpdate(time.Now())

    // Set optional active field
    if !plan.Crl.Active.IsNull() && !plan.Crl.Active.IsUnknown() {
        crlRequest.SetActive(plan.Crl.Active.ValueBool())
    }

    // Execute the request
    createCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
        CreateCertificateRevocationList(ctx).
        CertificateRevocationList(*crlRequest).
        Execute()

    if err != nil {
        if response.StatusCode == 429 {
            createCrl, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
                return r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
                    CreateCertificateRevocationList(ctx).
                    CertificateRevocationList(*crlRequest).
                    Execute()
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
    } else {
        if response != nil {
            defer response.Body.Close()
        }
    }

    // Populate the state from the response
    crlData := createCrl.GetData()
    plan.Crl = populateCrlResourceResults(ctx, crlData)
    plan.ID = types.StringValue(strconv.FormatInt(crlData.GetId(), 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Operation

```go
func (r *crlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state crlResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the CRL ID from state
    var crlID int64
    var err error
    if state.Crl != nil {
        crlID = state.Crl.ID.ValueInt64()
    } else {
        crlID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert CRL ID")
            return
        }
    }

    // Retrieve from API
    getCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
        RetrieveCertificateRevocationList(ctx, crlID).
        Execute()

    if err != nil {
        // Handle 404 (resource deleted outside Terraform)
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle 429 with retry...
        // Handle other errors...
    } else {
        if response != nil {
            defer response.Body.Close()
        }
    }

    // Update state from response
    crlData := getCrl.GetData()
    state.Crl = populateCrlResourceResults(ctx, crlData)
    state.ID = types.StringValue(strconv.FormatInt(crlData.GetId(), 10))

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Update Operation

The CRL resource uses PATCH for partial updates, which allows updating only the fields that have changed:

```go
func (r *crlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan crlResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state crlResourceModel
    diagsState := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsState...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the CRL ID from state
    var crlID int64
    var err error
    if state.ID.IsNull() {
        crlID = state.Crl.ID.ValueInt64()
    } else {
        crlID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert CRL ID")
            return
        }
    }

    // Build the PATCH request
    patchedCrl := azionapi.NewPatchedCertificateRevocationList()
    patchedCrl.SetName(plan.Crl.Name.ValueString())
    patchedCrl.SetIssuer(plan.Crl.Issuer.ValueString())
    patchedCrl.SetCrl(plan.Crl.Crl.ValueString())

    // Set optional active field
    if !plan.Crl.Active.IsNull() && !plan.Crl.Active.IsUnknown() {
        patchedCrl.SetActive(plan.Crl.Active.ValueBool())
    }

    // Execute the PATCH request
    updateCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
        PartialUpdateCertificateRevocationList(ctx, crlID).
        PatchedCertificateRevocationList(*patchedCrl).
        Execute()

    // Handle errors (429 retry, etc.)...

    // Update state from response
    crlData := updateCrl.GetData()
    plan.Crl = populateCrlResourceResults(ctx, crlData)
    plan.ID = types.StringValue(strconv.FormatInt(crlData.GetId(), 10))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Delete Operation

```go
func (r *crlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state crlResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the CRL ID from state
    var crlID int64
    var err error
    if state.Crl != nil {
        crlID = state.Crl.ID.ValueInt64()
    } else {
        crlID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error", "Could not convert CRL ID")
            return
        }
    }

    // Execute the delete request
    _, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
        DeleteCertificateRevocationList(ctx, crlID).
        Execute()

    if err != nil {
        // Handle 429 with retry...
        // Handle other errors...
    } else {
        if response != nil {
            defer response.Body.Close()
        }
    }
}
```

### Import State Operation

```go
func (r *crlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    crlID, err := strconv.ParseInt(req.ID, 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid ID format", "The ID must be a valid integer")
        return
    }

    // Retrieve the CRL to populate the state
    getCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
        RetrieveCertificateRevocationList(ctx, crlID).
        Execute()

    // Handle errors (429 retry, etc.)...

    state := crlResourceModel{
        Crl: populateCrlResourceResults(ctx, getCrl.GetData()),
        ID:  types.StringValue(strconv.FormatInt(getCrl.GetData().GetId(), 10)),
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Helper Function for Populating Results

```go
func populateCrlResourceResults(ctx context.Context, crl azionapi.CertificateRevocationList) *crlResourceResults {
    var createdAt string
    if crl.CreatedAt.IsSet() && crl.CreatedAt.Get() != nil {
        createdAt = (*crl.CreatedAt.Get()).Format(time.RFC3339)
    }

    result := &crlResourceResults{
        ID:             types.Int64Value(crl.GetId()),
        Name:           types.StringValue(crl.GetName()),
        LastEditor:     types.StringValue(crl.GetLastEditor()),
        CreatedAt:      types.StringValue(createdAt),
        LastModified:   types.StringValue(crl.GetLastModified().Format(time.RFC3339)),
        ProductVersion: types.StringValue(crl.GetProductVersion()),
        Issuer:         types.StringValue(crl.GetIssuer()),
        LastUpdate:     types.StringValue(crl.GetLastUpdate().Format(time.RFC3339)),
        NextUpdate:     types.StringValue(crl.GetNextUpdate().Format(time.RFC3339)),
        Crl:            types.StringValue(crl.GetCrl()),
    }

    // Handle optional fields
    if crl.Active != nil {
        result.Active = types.BoolValue(*crl.Active)
    }

    return result
}
```

### Registration in Provider

The resource must be registered in `internal/provider.go`:

```go
func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... other resources
        NewCrlResource,
    }
}
```

---

## Schema Definition Patterns

### Singular Data Source Schema

```go
func (c *CrlDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "crl_id": schema.Int64Attribute{
                Description: "Identifier of the certificate revocation list.",
                Required:    true,
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "results": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "Identifier of the certificate revocation list.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the certificate revocation list.",
                        Computed:    true,
                    },
                    // ... more fields
                },
            },
        },
    }
}
```

### Plural Data Source Schema

```go
func (d *CrlsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total number of certificate revocation lists.",
                Computed:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The current page number.",
                Computed:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The number of items per page.",
                Computed:    true,
            },
            "links": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "previous": schema.StringAttribute{Computed: true},
                    "next":     schema.StringAttribute{Computed: true},
                },
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "Identifier of the certificate revocation list.",
                            Computed:    true,
                        },
                        // ... more fields
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
crlResponse, response, err := c.client.api.DigitalCertificatesCertificateRevocationListsAPI.
    RetrieveCertificateRevocationList(ctx, crlID).
    Execute()

if err != nil {
    // Handle 429 (rate limiting) with retry
    if response.StatusCode == 429 {
        crlResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
            return c.client.api.DigitalCertificatesCertificateRevocationListsAPI.
                RetrieveCertificateRevocationList(ctx, crlID).
                Execute()
        }, 5)

        if response != nil {
            defer response.Body.Close()
        }

        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // Read error body for details
        bodyBytes, errReadAll := io.ReadAll(response.Body)
        if errReadAll != nil {
            resp.Diagnostics.AddError(errReadAll.Error(), "err")
        }
        bodyString := string(bodyBytes)
        resp.Diagnostics.AddError(err.Error(), bodyString)
        return
    }
} else {
    if response != nil {
        defer response.Body.Close()
    }
}
```

### HTTP Status Codes

| Code | Description | Action |
|------|-------------|--------|
| 200 | Success | Continue processing |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Check request body format |
| 401 | Unauthorized | Check API token |
| 403 | Forbidden | Check permissions |
| 404 | Not Found | Resource doesn't exist (remove from state for Read) |
| 405 | Method Not Allowed | Check HTTP method |
| 406 | Not Acceptable | Check Accept header |
| 429 | Too Many Requests | Retry with exponential backoff |
| 500 | Internal Server Error | Retry or report to support |

---

## Type Conversions

### Time Fields

The CRL API uses `time.Time` for timestamp fields. Convert to RFC3339 string format for Terraform:

```go
// Nullable time (e.g., created_at)
var createdAt string
if crl.CreatedAt.IsSet() && crl.CreatedAt.Get() != nil {
    createdAt = (*crl.CreatedAt.Get()).Format(time.RFC3339)
}

// Non-nullable time (e.g., last_modified)
lastModified := crl.GetLastModified().Format(time.RFC3339)
```

### Optional Bool Field

```go
// Handle optional fields
if crl.Active != nil {
    result.Results.Active = types.BoolValue(*crl.Active)
}
```

---

## Common Issues

### Issue: Missing `defer response.Body.Close()`

**Symptom:** Resource leaks, connection pool exhaustion.

**Solution:** Always close response body after successful API calls:

```go
if response != nil {
    defer response.Body.Close()
}
```

### Issue: Context Not Passed Through

**Symptom:** Linter error `contextcheck`.

**Solution:** Pass context as first parameter to all functions that need it:

```go
func populateCrlsListResults(ctx context.Context, list *azionapi.PaginatedCertificateRevocationList) CrlsDataSourceModel {
    // Use ctx if needed for logging or cancellation
}
```

### Issue: Nullable Time Fields

**Symptom:** Panic when accessing `CreatedAt` or other nullable time fields.

**Solution:** Always check if nullable fields are set before accessing:

```go
if crl.CreatedAt.IsSet() && crl.CreatedAt.Get() != nil {
    createdAt = (*crl.CreatedAt.Get()).Format(time.RFC3339)
}
```

---

## API Reference

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspace/tls/crls` | List all CRLs |
| POST | `/workspace/tls/crls` | Create a new CRL |
| GET | `/workspace/tls/crls/{crl_id}` | Retrieve a specific CRL |
| PUT | `/workspace/tls/crls/{crl_id}` | Update a CRL (full replacement) |
| PATCH | `/workspace/tls/crls/{crl_id}` | Partially update a CRL |
| DELETE | `/workspace/tls/crls/{crl_id}` | Delete a CRL |

### Query Parameters for List

| Parameter | Type | Description |
|-----------|------|-------------|
| `fields` | string | Comma-separated list of field names to include |
| `id` | int64 | Filter by CRL ID |
| `issuer` | string | Filter by issuer (case-insensitive, partial match) |
| `last_modified` | time | Filter by exact last modified date and time |
| `last_modified__gte` | time | Filter by last modified date (greater than or equal) |
| `last_modified__lte` | time | Filter by last modified date (less than or equal) |
| `last_update` | time | Filter by exact last update date and time |
| `last_update__gte` | time | Filter by last update date (greater than or equal) |
| `last_update__lte` | time | Filter by last update date (less than or equal) |
| `name` | string | Filter by CRL name (case-insensitive, partial match) |
| `next_update` | time | Filter by exact next update date and time |
| `next_update__gte` | time | Filter by next update date (greater than or equal) |
| `next_update__lte` | time | Filter by next update date (less than or equal) |
| `ordering` | string | Which field to use when ordering the results |
| `page` | int64 | A page number within the paginated result set |
| `page_size` | int64 | Number of items per page |
| `search` | string | A search term |

---

## Summary Checklist

When implementing CRL data sources or resources:

### Data Sources

1. ✅ Use the V4 SDK (`azion-api` package)
2. ✅ Import: `import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"`
3. ✅ Use `api.DigitalCertificatesCertificateRevocationListsAPI` client
4. ✅ ID type is `int64`
5. ✅ Handle 429 errors with `utils.RetryOn429`
6. ✅ Close response body with `defer response.Body.Close()`
7. ✅ Handle nullable time fields with `IsSet()` and `Get()` checks
8. ✅ Register data sources in `internal/provider.go`
9. ✅ Create documentation in `docs/data-sources/`
10. ✅ Create examples in `examples/data-sources/`

### Resource

1. ✅ Use the V4 SDK (`azion-api` package)
2. ✅ Import: `import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"`
3. ✅ Use `api.DigitalCertificatesCertificateRevocationListsAPI` client
4. ✅ ID type is `int64` (stored as `types.String` in resource model)
5. ✅ Use `NewCertificateRevocationListWithDefaults()` for create
6. ✅ Use `NewPatchedCertificateRevocationList()` for partial updates (PATCH)
7. ✅ Handle 429 errors with `utils.RetryOn429`
8. ✅ Close response body with `defer response.Body.Close()`
9. ✅ Handle nullable time fields with `IsSet()` and `Get()` checks
10. ✅ Implement ImportState for resource import support
11. ✅ Register resource in `internal/provider.go` via `NewCrlResource`
12. ✅ Create documentation in `docs/resources/`
13. ✅ Create examples in `examples/resources/`

### Key Patterns for Resource

- **Nested Attribute Pattern**: The resource uses a nested `crl` attribute to group all CRL-specific fields, following the provider's standard pattern used by other resources like `custom_page`
- **Response Handling**: Always store `GetData()` result in a variable before calling methods on it to avoid "cannot call pointer method" errors
- **Create vs Update**: Use `CertificateRevocationList` for create and `PatchedCertificateRevocationList` for partial updates
- **ID Storage**: Store the ID as `types.String` in the resource model, but use `int64` when calling API methods
