# Network Lists - Code Generation Guide

This document provides specific guidance for implementing Network Lists resources and data sources in the Terraform provider using the V4 SDK (`azion-api`).

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
7. [Examples and Documentation](#examples-and-documentation)

---

## SDK Selection

Network Lists use the **V4 SDK (`azion-api`)** from the `azionapi-v4-go-sdk-dev` module. **Important:** Do not use "edge" prefix in variable names or field names when working with the V4 SDK.

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Network List (Singular Data Source) | `azion-api` (v4) | `api.NetworkListsAPI` | `https://api.azion.com/v4` |
| Network Lists (Plural Data Source) | `azion-api` (v4) | `api.NetworkListsAPI` | `https://api.azion.com/v4` |
| Network List (Resource) | `azion-api` (v4) | `api.NetworkListsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `NetworkListRequest` |
| Update Request Type | `NetworkListRequest` (full update) |
| Response Type | `NetworkListResponse` with `Data` field |
| List Response Type | `PaginatedNetworkListSummaryList` |
| Create Pattern | `.CreateNetworkList(ctx).NetworkListRequest(req).Execute()` |
| Update Pattern | `.UpdateNetworkList(ctx, id).NetworkListRequest(req).Execute()` |
| List Method | `.ListNetworkLists(ctx).Page(page).Execute()` |
| Retrieve Method | `.RetrieveNetworkList(ctx, networkListId).Execute()` |
| Delete Method | `.DeleteNetworkList(ctx, networkListId).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - preferred for all implementations
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // ... other SDK clients
}
```

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Network List by its identifier:

**File:** `internal/data_source_network_list.go`

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
    _ datasource.DataSource              = &NetworkListDataSource{}
    _ datasource.DataSourceWithConfigure = &NetworkListDataSource{}
)

func dataSourceAzionNetworkList() datasource.DataSource {
    return &NetworkListDataSource{}
}

type NetworkListDataSource struct {
    client *apiClient
}

// Model definition - results wrapped in a nested attribute
type NetworkListDataSourceModel struct {
    ID      types.Int64        `tfsdk:"id"`
    Results *NetworkListResult `tfsdk:"results"`
}

type NetworkListResult struct {
    ID           types.Int64  `tfsdk:"id"`
    LastEditor   types.String `tfsdk:"last_editor"`
    LastModified types.String `tfsdk:"last_modified"`
    Type         types.String `tfsdk:"type"`
    Name         types.String `tfsdk:"name"`
    Items        types.List   `tfsdk:"items"`
}

// Configure - stores the API client from the provider
func (n *NetworkListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    n.client = req.ProviderData.(*apiClient)
}

// Metadata - defines the data source name (azion_network_list)
func (n *NetworkListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_network_list"
}

// Schema - defines the data source schema
func (n *NetworkListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Description: "Identifier of the network list.",
                Required:    true,
            },
            "results": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "ID of the network list.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor of the network list.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the network list.",
                        Computed:    true,
                    },
                    "type": schema.StringAttribute{
                        Description: "Type of the network list. Can be: asn, countries, or ip_cidr.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the network list.",
                        Computed:    true,
                    },
                    "items": schema.ListAttribute{
                        Computed:    true,
                        ElementType: types.StringType,
                        Description: "List of items in the network list.",
                    },
                },
            },
        },
    }
}

// Read - fetches the network list by ID
func (n *NetworkListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var networkListID types.Int64
    diagsID := req.Config.GetAttribute(ctx, path.Root("id"), &networkListID)
    resp.Diagnostics.Append(diagsID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Call the API
    networkListResponse, response, err := n.client.api.NetworkListsAPI.RetrieveNetworkList(ctx, networkListID.ValueInt64()).Execute()
    if err != nil {
        // Handle rate limiting (429)
        if response != nil && response.StatusCode == 429 {
            networkListResponse, response, err = utils.RetryOn429(func() (*azionapi.NetworkListResponse, *http.Response, error) {
                return n.client.api.NetworkListsAPI.RetrieveNetworkList(ctx, networkListID.ValueInt64()).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            // Handle other errors
            if response != nil && response.Body != nil {
                bodyBytes, _ := io.ReadAll(response.Body)
                resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            } else {
                resp.Diagnostics.AddError(err.Error(), "API request failed")
            }
            return
        }
    }

    // Populate the state
    networkListState := populateNetworkListResult(ctx, networkListResponse.GetData())
    diags := resp.State.Set(ctx, &networkListState)
    resp.Diagnostics.Append(diags...)
}

// Helper function to populate the result
func populateNetworkListResult(ctx context.Context, data azionapi.NetworkList) NetworkListDataSourceModel {
    var itemsSlice []types.String
    for _, item := range data.GetItems() {
        itemsSlice = append(itemsSlice, types.StringValue(item))
    }

    return NetworkListDataSourceModel{
        ID: types.Int64Value(data.GetId()),
        Results: &NetworkListResult{
            ID:           types.Int64Value(data.GetId()),
            LastEditor:   types.StringValue(data.GetLastEditor()),
            LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
            Type:         types.StringValue(data.GetType()),
            Name:         types.StringValue(data.GetName()),
            Items:        utils.SliceStringTypeToList(itemsSlice),
        },
    }
}
```

---

### Plural Data Source (List Multiple Resources)

For listing multiple Network Lists with pagination:

**File:** `internal/data_source_network_lists.go`

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

var (
    _ datasource.DataSource              = &NetworkListsDataSource{}
    _ datasource.DataSourceWithConfigure = &NetworkListsDataSource{}
)

func dataSourceAzionNetworkLists() datasource.DataSource {
    return &NetworkListsDataSource{}
}

type NetworkListsDataSource struct {
    client *apiClient
}

// Model with pagination fields
type NetworkListsDataSourceModel struct {
    Counter    types.Int64                `tfsdk:"counter"`
    Page       types.Int64                `tfsdk:"page"`
    TotalPages types.Int64                `tfsdk:"total_pages"`
    Links      *NetworkListsResponseLinks `tfsdk:"links"`
    Results    []NetworkListsResults      `tfsdk:"results"`
    ID         types.String               `tfsdk:"id"`
}

type NetworkListsResponseLinks struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

type NetworkListsResults struct {
    ID           types.Int64  `tfsdk:"id"`
    LastEditor   types.String `tfsdk:"last_editor"`
    LastModified types.String `tfsdk:"last_modified"`
    Type         types.String `tfsdk:"type"`
    Name         types.String `tfsdk:"name"`
}

func (n *NetworkListsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_network_lists"
}

func (n *NetworkListsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Optional:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total number of network lists.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The page number of network lists.",
                Optional:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
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
                        "id":            schema.Int64Attribute{Computed: true},
                        "last_editor":   schema.StringAttribute{Computed: true},
                        "last_modified": schema.StringAttribute{Computed: true},
                        "type":          schema.StringAttribute{Computed: true},
                        "name":          schema.StringAttribute{Computed: true},
                    },
                },
            },
        },
    }
}

func (n *NetworkListsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var page types.Int64
    diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
    resp.Diagnostics.Append(diagsPage...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Default to page 1 if not specified
    if page.IsNull() || page.IsUnknown() {
        page = types.Int64Value(1)
    }

    // Security check for int64 to int32 conversion
    page32, err := utils.CheckInt64toInt32Security(page.ValueInt64())
    if err != nil {
        utils.ExceedsValidRange(resp, page.ValueInt64())
        return
    }

    // Call the API
    networkListsResponse, response, err := n.client.api.NetworkListsAPI.ListNetworkLists(ctx).Page(int64(page32)).Execute()
    if err != nil {
        if response != nil && response.StatusCode == 429 {
            networkListsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedNetworkListSummaryList, *http.Response, error) {
                return n.client.api.NetworkListsAPI.ListNetworkLists(ctx).Page(int64(page32)).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            if response != nil && response.Body != nil {
                bodyBytes, _ := io.ReadAll(response.Body)
                resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            } else {
                resp.Diagnostics.AddError(err.Error(), "API request failed")
            }
            return
        }
    }

    // Build the results slice
    var networkLists []NetworkListsResults
    for _, nl := range networkListsResponse.GetResults() {
        networkList := NetworkListsResults{
            ID:           types.Int64Value(nl.GetId()),
            LastEditor:   types.StringValue(nl.GetLastEditor()),
            LastModified: types.StringValue(nl.GetLastModified().Format(time.RFC3339)),
            Type:         types.StringValue(nl.GetType()),
            Name:         types.StringValue(nl.GetName()),
        }
        networkLists = append(networkLists, networkList)
    }

    // Set the state
    networkListsState := NetworkListsDataSourceModel{
        Counter:    types.Int64Value(networkListsResponse.GetCount()),
        Page:       types.Int64Value(networkListsResponse.GetPage()),
        TotalPages: types.Int64Value(networkListsResponse.GetTotalPages()),
        Links: &NetworkListsResponseLinks{
            Previous: types.StringValue(networkListsResponse.GetPrevious()),
            Next:     types.StringValue(networkListsResponse.GetNext()),
        },
        Results: networkLists,
        ID:      types.StringValue("Get All Network Lists"),
    }

    diags := resp.State.Set(ctx, &networkListsState)
    resp.Diagnostics.Append(diags...)
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`network_list`) | Plural (`network_lists`) |
|--------|--------------------------|--------------------------|
| API Method | `RetrieveNetworkList(ctx, id)` | `ListNetworkLists(ctx).Page(n)` |
| Response Type | `NetworkListResponse` | `PaginatedNetworkListSummaryList` |
| Data Field | `GetData()` → `NetworkList` | `GetResults()` → `[]NetworkListSummary` |
| Items Field | **Yes** - Full list of items | **No** - Summary only |
| ID Parameter | **Required** (`int64`) | **Not applicable** |
| Page Parameter | **Not applicable** | **Optional** (defaults to 1) |
| Schema Structure | `SingleNestedAttribute` for results | `ListNestedAttribute` for results |

**Important:** The plural data source returns a summary (`NetworkListSummary`) that does NOT include the `items` field. You must use the singular data source to get the full list of items in a network list.

---

## Resource Implementation

**File:** `internal/resource_network_list.go`

The resource implementation follows these patterns:

### Create

```go
func (r *networkListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan NetworkListResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Extract items from the Set
    var items []string
    diagsItems := plan.NetworkList.Items.ElementsAs(ctx, &items, false)
    resp.Diagnostics.Append(diagsItems...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the request
    networkListRequest := azionapi.NetworkListRequest{
        Name:  plan.NetworkList.Name.ValueString(),
        Type:  plan.NetworkList.Type.ValueString(),
        Items: items,
    }

    // Call the API
    createNetworkListResponse, response, err := r.client.api.NetworkListsAPI.CreateNetworkList(ctx).
        NetworkListRequest(networkListRequest).Execute()
    if err != nil {
        // Handle errors...
    }

    // Populate state from response
    data := createNetworkListResponse.GetData()
    plan.NetworkList = &NetworkListResourceResults{
        ID:           types.Int64Value(data.GetId()),
        LastEditor:   types.StringValue(data.GetLastEditor()),
        LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
        Type:         types.StringValue(data.GetType()),
        Name:         types.StringValue(data.GetName()),
        Items:        utils.SliceStringTypeToSet(sliceString),
    }
    plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
}
```

### Update (Full Update with PUT)

```go
func (r *networkListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // Build the request (same as Create)
    networkListRequest := azionapi.NetworkListRequest{
        Name:  plan.NetworkList.Name.ValueString(),
        Type:  plan.NetworkList.Type.ValueString(),
        Items: items,
    }

    // Call the API - uses PUT for full update
    updateNetworkList, response, err := r.client.api.NetworkListsAPI.
        UpdateNetworkList(ctx, networkListId).
        NetworkListRequest(networkListRequest).Execute()
}
```

### Delete

```go
func (r *networkListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    _, response, err := r.client.api.NetworkListsAPI.DeleteNetworkList(ctx, networkListId).Execute()
    // Handle errors...
}
```

### Key Points

1. **Update Method**: Uses `UpdateNetworkList` with PUT (full update), not partial update
2. **Items Field**: Uses `types.Set` for items in the schema
3. **Time Format**: Use `time.RFC3339` for `LastModified` timestamp

---

## Schema Definition Patterns

### Network List Types

Network Lists support three types:

| Type | Description | Items Format |
|------|-------------|--------------|
| `asn` | Autonomous System Numbers | Numeric strings (e.g., "1234", "13335") |
| `countries` | Country Codes | ISO 3166-1 alpha-2 codes (e.g., "BR", "US") |
| `ip_cidr` | IP Addresses/CIDR | IPv4/IPv6 with optional CIDR (e.g., "192.168.0.1", "192.168.0.0/24") |

### Items Format

For `ip_cidr` type, items can include:
- Simple IPv4: `192.168.0.1`
- IPv4 with CIDR: `192.168.0.1/24`
- Simple IPv6: `2001:db8:3333:4444:5555:6666:7777:8888`
- IPv6 with CIDR: `2001:db8::/32`
- IP with expiration: `192.168.0.1 --LT2025-05-29T12:25:23Z`

### Schema Attribute Names

| SDK Field | Terraform Attribute | Description |
|-----------|---------------------|-------------|
| `id` | `id` | Resource identifier |
| `name` | `name` | Network list name |
| `type` | `type` | Network list type (asn/countries/ip_cidr) |
| `items` | `items` | List of items (use `types.Set`) |
| `last_editor` | `last_editor` | Last editor (computed) |
| `last_modified` | `last_modified` | Last modified timestamp (computed) |

---

## Error Handling

### Standard Error Handling Pattern

```go
if err != nil {
    // Check for rate limiting (429)
    if response != nil && response.StatusCode == 429 {
        // Retry with exponential backoff
        result, response, err = utils.RetryOn429(func() (*ResponseType, *http.Response, error) {
            return client.API.Method(ctx, params).Execute()
        }, 5) // Max 5 retries

        if response != nil {
            defer response.Body.Close()
        }

        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // Handle other errors
        if response != nil && response.Body != nil {
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
        } else {
            resp.Diagnostics.AddError(err.Error(), "API request failed")
        }
        return
    }
}
```

### Special HTTP Status Codes

- **404 (Not Found)**: In Read operations, use `resp.State.RemoveResource(ctx)` to mark the resource as deleted
- **429 (Rate Limited)**: Always retry with `utils.RetryOn429`
- **Always close response body**: `defer response.Body.Close()` after successful retry

---

## Type Conversions

### Items as Set

```go
// Converting items from plan to []string
var items []string
diags := plan.NetworkList.Items.ElementsAs(ctx, &items, false)

// Converting items from API to types.Set
var sliceString []types.String
for _, item := range data.GetItems() {
    sliceString = append(sliceString, types.StringValue(item))
}
plan.NetworkList.Items = utils.SliceStringTypeToSet(sliceString)
```

### Time Formatting

```go
// Format time.Time from API to string for Terraform
LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339))
```

### ID Conversion

```go
// Convert int64 ID to string for Terraform resource ID
plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))

// Convert string ID from state to int64 for API calls
id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
```

---

## Examples and Documentation

### Resource Example

**File:** `examples/resources/azion_network_list/resource.tf`

```terraform
# Network List with country codes
resource "azion_network_list" "countries_example" {
  results = {
    name = "Blocked Countries"
    type = "countries"
    items = [
      "BR",
      "US",
      "AG"
    ]
  }
}

# Network List with IP CIDR addresses
resource "azion_network_list" "ip_cidr_example" {
  results = {
    name = "Allowed IPs"
    type = "ip_cidr"
    items = [
      "192.168.0.1",
      "192.168.0.0/24",
      "2001:db8::/32"
    ]
  }
}

# Network List with ASN numbers
resource "azion_network_list" "asn_example" {
  results = {
    name = "Allowed ASNs"
    type = "asn"
    items = [
      "1234",
      "5678",
      "13335"
    ]
  }
}
```

### Data Source Example

**File:** `examples/data-sources/azion_network_list/data-source.tf`

```terraform
data "azion_network_list" "example" {
  id = 12345
}
```

### Documentation

**File:** `docs/resources/network_list.md`

Include:
- Description of the resource
- Example usage for all three types (asn, countries, ip_cidr)
- Schema documentation with all attributes
- Import instructions
- Item format documentation by type

---

## Summary Checklist

When implementing Network Lists:

1. ✅ Use V4 SDK (`azion-api`) import: `import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"`
2. ✅ Use `api.NetworkListsAPI` client (no "edge" prefix)
3. ✅ Use `types.Set` for items field
4. ✅ Use `time.RFC3339` format for timestamps
5. ✅ Handle 429 errors with `utils.RetryOn429`
6. ✅ Close response body after successful API calls
7. ✅ Use `UpdateNetworkList` (PUT) for full updates
8. ✅ Update examples and documentation after schema changes
9. ✅ Run linters: `golangci-lint run --config .golintci.yml ./internal/...`
