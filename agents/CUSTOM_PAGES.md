# Custom Pages - Code Generation Guide

This document provides specific guidance for implementing Custom Pages resources and data sources in the Terraform provider.

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
7. [API Models](#api-models)
8. [Common Issues](#common-issues)

---

## SDK Selection

Custom Pages use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Custom Page (Singular Data Source) | `azion-api` (v4) | `api.CustomPagesAPI` | `https://api.azion.com/v4` |
| Custom Pages (Plural Data Source) | `azion-api` (v4) | `api.CustomPagesAPI` | `https://api.azion.com/v4` |
| Custom Page (Resource) | `azion-api` (v4) | `api.CustomPagesAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `CustomPageRequest` |
| Update Request Type | `CustomPageRequest` (PUT) or `PatchedCustomPageRequest` (PATCH) |
| Response Type | `CustomPageResponse` with `Data` field |
| List Response Type | `PaginatedCustomPageList` |
| Create Pattern | `.CreateCustomPage(ctx).CustomPageRequest(req).Execute()` |
| Update Pattern | `.UpdateCustomPage(ctx, id).CustomPageRequest(req).Execute()` |
| Retrieve Pattern | `.RetrieveCustomPage(ctx, customPageId).Execute()` |
| List Method | `.ListCustomPages(ctx).Execute()` |
| Delete Method | `.DestroyCustomPage(ctx, customPageId).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - preferred for all implementations
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    // ... more SDK clients
}
```

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Custom Page by its identifier:

**File:** `internal/data_source_custom_page.go`

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
    _ datasource.DataSource              = &CustomPageDataSource{}
    _ datasource.DataSourceWithConfigure = &CustomPageDataSource{}
)

func dataSourceAzionCustomPage() datasource.DataSource {
    return &CustomPageDataSource{}
}

type CustomPageDataSource struct {
    client *apiClient
}

// Model for the data source state
type CustomPageDataSourceModel struct {
    Data CustomPageResults `tfsdk:"data"`
    ID   types.String      `tfsdk:"id"`
}

// Results model matching API response structure
type CustomPageResults struct {
    ID             types.Int64             `tfsdk:"id"`
    Name           types.String            `tfsdk:"name"`
    LastEditor     types.String            `tfsdk:"last_editor"`
    LastModified   types.String            `tfsdk:"last_modified"`
    Active         types.Bool              `tfsdk:"active"`
    ProductVersion types.String            `tfsdk:"product_version"`
    Pages          []CustomPagePageResults `tfsdk:"pages"`
}

// Nested page structure models
type CustomPagePageResults struct {
    Code types.String                   `tfsdk:"code"`
    Page CustomPagePageConnectorResults `tfsdk:"page"`
}

type CustomPagePageConnectorResults struct {
    Type       types.String                    `tfsdk:"type"`
    Attributes CustomPagePageAttributesResults `tfsdk:"attributes"`
}

type CustomPagePageAttributesResults struct {
    Connector        types.Int64  `tfsdk:"connector"`
    TTL              types.Int64  `tfsdk:"ttl"`
    URI              types.String `tfsdk:"uri"`
    CustomStatusCode types.Int64  `tfsdk:"custom_status_code"`
}
```

### Schema Definition for Singular Data Source

```go
func (d *CustomPageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Required:    true,
            },
            "data": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The custom page identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the custom page.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "The last editor of the custom page.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the custom page.",
                        Computed:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Status of the custom page.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version of the custom page.",
                        Computed:    true,
                    },
                    "pages": schema.ListNestedAttribute{
                        Description: "List of pages associated with the custom page.",
                        Computed:    true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "code": schema.StringAttribute{
                                    Description: "HTTP status code for the page.",
                                    Computed:    true,
                                },
                                "page": schema.SingleNestedAttribute{
                                    Description: "Page connector configuration.",
                                    Computed:    true,
                                    Attributes: map[string]schema.Attribute{
                                        "type": schema.StringAttribute{
                                            Description: "Type of the page connector.",
                                            Computed:    true,
                                        },
                                        "attributes": schema.SingleNestedAttribute{
                                            Description: "Attributes of the page connector.",
                                            Computed:    true,
                                            Attributes: map[string]schema.Attribute{
                                                "connector": schema.Int64Attribute{
                                                    Description: "Connector ID.",
                                                    Computed:    true,
                                                },
                                                "ttl": schema.Int64Attribute{
                                                    Description: "Time to live for the page.",
                                                    Computed:    true,
                                                },
                                                "uri": schema.StringAttribute{
                                                    Description: "URI for the page.",
                                                    Computed:    true,
                                                },
                                                "custom_status_code": schema.Int64Attribute{
                                                    Description: "Custom status code for the page.",
                                                    Computed:    true,
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}
```

### Read Method for Singular Data Source

```go
func (d *CustomPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getCustomPageId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("id"), &getCustomPageId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    customPageID, err := strconv.ParseInt(getCustomPageId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    customPageResponse, response, err := d.client.api.CustomPagesAPI.
        RetrieveCustomPage(ctx, customPageID).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            customPageResponse, response, err = utils.RetryOn429(func() (*azionapi.CustomPageResponse, *http.Response, error) {
                return d.client.api.CustomPagesAPI.RetrieveCustomPage(ctx, customPageID).Execute()
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
            usrMsg, errMsg := errPrintCustomPage(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    customPageState := CustomPageDataSourceModel{
        Data: CustomPageResults{
            ID:             types.Int64Value(customPageResponse.Data.Id),
            Name:           types.StringValue(customPageResponse.Data.Name),
            LastEditor:     types.StringValue(customPageResponse.Data.LastEditor),
            LastModified:   types.StringValue(customPageResponse.Data.LastModified.Format(time.RFC3339)),
            ProductVersion: types.StringValue(customPageResponse.Data.ProductVersion),
        },
    }

    // Handle optional active field
    if customPageResponse.Data.Active != nil {
        customPageState.Data.Active = types.BoolValue(*customPageResponse.Data.Active)
    }

    // Convert pages
    for _, page := range customPageResponse.Data.Pages {
        pageResult := CustomPagePageResults{
            Code: types.StringValue(page.Code),
            Page: CustomPagePageConnectorResults{
                Type: types.StringValue(page.Page.Type),
                Attributes: CustomPagePageAttributesResults{
                    Connector: types.Int64Value(page.Page.Attributes.Connector),
                },
            },
        }

        // Handle optional TTL
        if page.Page.Attributes.Ttl != nil {
            pageResult.Page.Attributes.TTL = types.Int64Value(*page.Page.Attributes.Ttl)
        }

        // Handle optional URI (NullableString)
        if page.Page.Attributes.Uri.IsSet() && page.Page.Attributes.Uri.Get() != nil {
            pageResult.Page.Attributes.URI = types.StringValue(*page.Page.Attributes.Uri.Get())
        }

        // Handle optional CustomStatusCode (NullableInt64)
        if page.Page.Attributes.CustomStatusCode.IsSet() && page.Page.Attributes.CustomStatusCode.Get() != nil {
            pageResult.Page.Attributes.CustomStatusCode = types.Int64Value(*page.Page.Attributes.CustomStatusCode.Get())
        }

        customPageState.Data.Pages = append(customPageState.Data.Pages, pageResult)
    }

    customPageState.ID = types.StringValue("Get By Id Custom Page")
    diags = resp.State.Set(ctx, &customPageState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

### Plural Data Source (List Multiple Resources)

For listing all Custom Pages:

**File:** `internal/data_source_custom_pages.go`

```go
package provider

import (
    "context"
    "fmt"
    "net/http"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &CustomPagesDataSource{}
    _ datasource.DataSourceWithConfigure = &CustomPagesDataSource{}
)

func dataSourceAzionCustomPages() datasource.DataSource {
    return &CustomPagesDataSource{}
}

type CustomPagesDataSource struct {
    client *apiClient
}

// Model for plural data source
type CustomPagesDataSourceModel struct {
    Counter types.Int64         `tfsdk:"counter"`
    Results []CustomPagesResults `tfsdk:"results"`
    ID      types.String        `tfsdk:"id"`
}

// Results model (same structure as singular but with different type name for clarity)
type CustomPagesResults struct {
    ID             types.Int64              `tfsdk:"id"`
    Name           types.String             `tfsdk:"name"`
    LastEditor     types.String             `tfsdk:"last_editor"`
    LastModified   types.String             `tfsdk:"last_modified"`
    Active         types.Bool               `tfsdk:"active"`
    ProductVersion types.String             `tfsdk:"product_version"`
    Pages          []CustomPagesPageResults `tfsdk:"pages"`
}
```

### Read Method for Plural Data Source

```go
func (d *CustomPagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    customPagesResponse, response, err := d.client.api.CustomPagesAPI.ListCustomPages(ctx).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            customPagesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCustomPageList, *http.Response, error) {
                return d.client.api.CustomPagesAPI.ListCustomPages(ctx).Execute()
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
            usrMsg, errMsg := errPrintCustomPages(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    customPagesState := CustomPagesDataSourceModel{
        Counter: types.Int64Value(*customPagesResponse.Count),
    }

    for _, resultCustomPage := range customPagesResponse.GetResults() {
        result := CustomPagesResults{
            ID:             types.Int64Value(resultCustomPage.Id),
            Name:           types.StringValue(resultCustomPage.Name),
            LastEditor:     types.StringValue(resultCustomPage.LastEditor),
            LastModified:   types.StringValue(resultCustomPage.LastModified.String()),
            ProductVersion: types.StringValue(resultCustomPage.ProductVersion),
        }

        // Handle optional fields
        if resultCustomPage.Active != nil {
            result.Active = types.BoolValue(*resultCustomPage.Active)
        }

        // Convert pages...
        customPagesState.Results = append(customPagesState.Results, result)
    }

    customPagesState.ID = types.StringValue("Get All Custom Pages")
    diags := resp.State.Set(ctx, &customPagesState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`custom_page`) | Plural (`custom_pages`) |
|--------|--------------------------|-------------------------|
| **ID Parameter** | Required (user provides) | Computed (generated) |
| **ID Type** | `types.String` (input), converted to `int64` | `types.String` (static identifier) |
| **Schema Root Attribute** | `data` (SingleNestedAttribute) | `results` (ListNestedAttribute) |
| **Counter Field** | Not present | Present (`types.Int64`) |
| **API Method** | `RetrieveCustomPage(ctx, id)` | `ListCustomPages(ctx)` |
| **Response Type** | `*CustomPageResponse` | `*PaginatedCustomPageList` |
| **Data Access** | `response.Data` (single object) | `response.Results` (slice) |
| **Purpose** | Read specific resource by ID | List all resources |

---

## Resource Implementation

For creating, updating, and deleting Custom Pages, implement a resource:

**File:** `internal/resource_custom_page.go`

The resource should implement:
- `Create` - Create a new custom page
- `Read` - Read the current state from API
- `Update` - Update an existing custom page (PUT for full update)
- `Delete` - Delete a custom page
- `ImportState` - Allow importing existing custom pages

### Create Method Pattern

```go
func (r *CustomPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan CustomPageResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build request from plan
    createRequest := azionapi.CustomPageRequest{
        Name:           plan.Name.ValueString(),
        ProductVersion: plan.ProductVersion.ValueString(),
        // ... build pages
    }

    // Execute create
    createResponse, response, err := r.client.api.CustomPagesAPI.
        CreateCustomPage(ctx).
        CustomPageRequest(createRequest).
        Execute()
    
    // Handle errors and 429 retries...
    
    // Set state with response data
    plan.ID = types.Int64Value(createResponse.Data.Id)
    // ... set other fields from response
}
```

### Update Method Pattern

Custom Pages use PUT (full update) method:

```go
func (r *CustomPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan CustomPageResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build request
    updateRequest := azionapi.CustomPageRequest{
        Name:           plan.Name.ValueString(),
        ProductVersion: plan.ProductVersion.ValueString(),
        // ... build pages
    }

    // Execute update (PUT)
    updateResponse, response, err := r.client.api.CustomPagesAPI.
        UpdateCustomPage(ctx, plan.ID.ValueInt64()).
        CustomPageRequest(updateRequest).
        Execute()
    
    // Handle errors...
}
```

### Delete Method Pattern

```go
func (r *CustomPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state CustomPageResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    customPageId := state.CustomPage.ID.ValueInt64()

    _, response, err := r.client.api.CustomPagesAPI.
        DeleteCustomPage(ctx, customPageId).
        Execute()
    
    // Handle errors and 429 retries...
}
```

### ImportState Method Pattern

```go
func (r *CustomPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

### Building Pages for Request

When building pages for create/update requests, use the request-specific types:

```go
// Build pages from plan
var pages []azionapi.PageRequestBase
for _, page := range plan.CustomPage.Pages {
    pageRequest := azionapi.PageRequestBase{
        Code: page.Code.ValueString(),
        Page: azionapi.PageConnectorRequest{
            Type: page.Page.Type.ValueString(),
            Attributes: azionapi.PageConnectorAttributesRequest{
                Connector: page.Page.Attributes.Connector.ValueInt64(),
            },
        },
    }

    // Set optional TTL
    if !page.Page.Attributes.TTL.IsNull() && !page.Page.Attributes.TTL.IsUnknown() {
        pageRequest.Page.Attributes.SetTtl(page.Page.Attributes.TTL.ValueInt64())
    }

    // Set optional URI
    if !page.Page.Attributes.URI.IsNull() && !page.Page.Attributes.URI.IsUnknown() {
        pageRequest.Page.Attributes.SetUri(page.Page.Attributes.URI.ValueString())
    }

    // Set optional CustomStatusCode
    if !page.Page.Attributes.CustomStatusCode.IsNull() && !page.Page.Attributes.CustomStatusCode.IsUnknown() {
        pageRequest.Page.Attributes.SetCustomStatusCode(page.Page.Attributes.CustomStatusCode.ValueInt64())
    }

    pages = append(pages, pageRequest)
}
customPageRequest.SetPages(pages)
```

---

## Resource Schema Definition

```go
func (r *CustomPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Creates a Custom Page resource.",
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
            "custom_page": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The custom page identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the custom page.",
                        Required:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Status of the custom page.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "pages": schema.ListNestedAttribute{
                        Description: "List of pages associated with the custom page.",
                        Required:    true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "code": schema.StringAttribute{
                                    Description: "HTTP status code for the page.",
                                    Required:    true,
                                },
                                "page": schema.SingleNestedAttribute{
                                    Description: "Page connector configuration.",
                                    Required:    true,
                                    Attributes: map[string]schema.Attribute{
                                        "type": schema.StringAttribute{
                                            Description: "Type of the page connector.",
                                            Required:    true,
                                        },
                                        "attributes": schema.SingleNestedAttribute{
                                            Description: "Attributes of the page connector.",
                                            Required:    true,
                                            Attributes: map[string]schema.Attribute{
                                                "connector": schema.Int64Attribute{
                                                    Description: "Connector ID.",
                                                    Required:    true,
                                                },
                                                "ttl": schema.Int64Attribute{
                                                    Description: "Time to live for the page.",
                                                    Optional:    true,
                                                    Computed:    true,
                                                },
                                                "uri": schema.StringAttribute{
                                                    Description: "URI for the page.",
                                                    Optional:    true,
                                                    Computed:    true,
                                                },
                                                "custom_status_code": schema.Int64Attribute{
                                                    Description: "Custom status code for the page.",
                                                    Optional:    true,
                                                    Computed:    true,
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                    // Computed fields...
                    "last_editor": schema.StringAttribute{
                        Description: "The last editor of the custom page.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the custom page.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version of the custom page.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}

---

## Schema Definition Patterns

### Required vs Optional vs Computed

| Field | Singular DS | Plural DS | Resource Plan | Resource State |
|-------|-------------|-----------|---------------|----------------|
| `id` (input) | Required | - | - | Computed |
| `id` (data) | Computed | Computed | Computed | Computed |
| `name` | Computed | Computed | Required | Computed |
| `active` | Computed | Computed | Optional | Computed |
| `pages` | Computed | Computed | Required | Computed |
| `last_editor` | Computed | Computed | Computed | Computed |
| `last_modified` | Computed | Computed | Computed | Computed |
| `product_version` | Computed | Computed | Computed | Computed |

### Nested Object Patterns

For complex nested structures like `pages`, use `ListNestedAttribute`:

```go
"pages": schema.ListNestedAttribute{
    Description: "List of pages associated with the custom page.",
    Computed:    true,  // For data sources
    // For resources: Optional/Required
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            // Nested attributes...
        },
    },
},
```

---

## Error Handling

### Standard Error Handling Pattern

```go
func errPrintCustomPage(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Custom Page found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

### Rate Limiting (429) Handling

```go
if response.StatusCode == 429 {
    result, response, err = utils.RetryOn429(func() (*azionapi.CustomPageResponse, *http.Response, error) {
        return r.client.api.CustomPagesAPI.RetrieveCustomPage(ctx, id).Execute()
    }, 5) // Max 5 retries

    if response != nil {
        defer response.Body.Close()
    }

    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
        return
    }
}
```

---

## Type Conversions

### Time Formatting

```go
// From API response (time.Time)
lastModified := types.StringValue(response.Data.LastModified.Format(time.RFC3339))
```

### Nullable Types Handling

The SDK uses special nullable types for optional fields:

```go
// NullableString - check with IsSet() and Get()
if page.Page.Attributes.Uri.IsSet() && page.Page.Attributes.Uri.Get() != nil {
    uri := types.StringValue(*page.Page.Attributes.Uri.Get())
}

// NullableInt64 - similar pattern
if page.Page.Attributes.CustomStatusCode.IsSet() && page.Page.Attributes.CustomStatusCode.Get() != nil {
    statusCode := types.Int64Value(*page.Page.Attributes.CustomStatusCode.Get())
}
```

### Pointer Types

```go
// Optional bool pointer
if response.Data.Active != nil {
    active := types.BoolValue(*response.Data.Active)
}
```

---

## API Models

### CustomPage Model

```go
type CustomPage struct {
    Id             int64       `json:"id"`
    Name           string      `json:"name"`
    LastEditor     string      `json:"last_editor"`
    LastModified   time.Time   `json:"last_modified"`
    Active         *bool       `json:"active,omitempty"`
    ProductVersion string      `json:"product_version"`
    Pages          []PageBase  `json:"pages"`
}
```

### PageBase Model

```go
type PageBase struct {
    Code string        `json:"code"`  // HTTP status code: "default", "400", "404", etc.
    Page PageConnector `json:"page"`
}
```

### PageConnector Model

```go
type PageConnector struct {
    Type       string                  `json:"type"`
    Attributes PageConnectorAttributes `json:"attributes"`
}
```

### PageConnectorAttributes Model

```go
type PageConnectorAttributes struct {
    Connector        int64          `json:"connector"`         // Required
    Ttl              *int64         `json:"ttl,omitempty"`    // Optional
    Uri              NullableString `json:"uri,omitempty"`    // Optional
    CustomStatusCode NullableInt64  `json:"custom_status_code,omitempty"` // Optional
}
```

### HTTP Status Codes for Pages

The `code` field in PageBase accepts these values:
- `default` - Default page
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `405` - Method Not Allowed
- `406` - Not Acceptable
- `408` - Request Timeout
- `409` - Conflict
- `410` - Gone
- `411` - Length Required
- `414` - URI Too Long
- `415` - Unsupported Media Type
- `416` - Range Not Satisfiable
- `426` - Upgrade Required
- `429` - Too Many Requests
- `431` - Request Header Fields Too Large
- `500` - Internal Server Error
- `501` - Not Implemented
- `502` - Bad Gateway
- `503` - Service Unavailable
- `504` - Gateway Timeout
- `505` - HTTP Version Not Supported

---

## Common Issues

### Issue: Missing `defer response.Body.Close()`

**Problem:** Not closing the HTTP response body after 429 retry handling.

**Solution:** Always close the response body:

```go
if response != nil {
    defer response.Body.Close()
}
```

### Issue: Nullable Types Not Handled Correctly

**Problem:** SDK nullable types (`NullableString`, `NullableInt64`) require special handling.

**Solution:** Check both `IsSet()` and `Get() != nil`:

```go
if attr.Uri.IsSet() && attr.Uri.Get() != nil {
    value := *attr.Uri.Get()
}
```

### Issue: Missing Provider Registration

**Problem:** Data source is not registered in `provider.go`.

**Solution:** Add to the `DataSources()` function:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ... existing data sources
        dataSourceAzionCustomPage,
        dataSourceAzionCustomPages,
    }
}
```

### Issue: Schema Mismatch Between Data Source and Resource

**Problem:** Data sources and resources should have consistent schema structures.

**Solution:** Use consistent naming and structure. Create shared model types if needed.

---

## Files to Create/Update

When implementing Custom Pages, the following files need to be created or modified:

| File | Purpose |
|------|---------|
| `internal/data_source_custom_page.go` | Singular data source implementation |
| `internal/data_source_custom_pages.go` | Plural data source implementation |
| `internal/resource_custom_page.go` | Resource implementation (CRUD) |
| `internal/provider.go` | Register data sources and resources |
| `docs/data-sources/custom_page.md` | Documentation for singular data source |
| `docs/data-sources/custom_pages.md` | Documentation for plural data source |
| `docs/resources/custom_page.md` | Documentation for resource |
| `examples/data-sources/azion_custom_page/data-source.tf` | Example for singular data source |
| `examples/data-sources/azion_custom_pages/data-source.tf` | Example for plural data source |
| `examples/resources/azion_custom_page/resource.tf` | Example for resource |
| `examples/resources/azion_custom_page/import.sh` | Import example |

---

## Running Linters

After implementing, run the linters to ensure code quality:

```bash
golangci-lint run --config .golintci.yml ./internal/...
```

Key linter checks:
- `bodyclose` - Ensures HTTP response bodies are closed
- `contextcheck` - Ensures context is passed through function calls
- `godot` - Ensures comments end with a period
