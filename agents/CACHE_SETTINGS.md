# Cache Settings - Code Generation Guide

This document provides specific guidance for implementing Cache Settings data sources and resources in the Terraform provider using the V4 SDK.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [V4 API Structure](#v4-api-structure)
   - [Response Types](#response-types)
   - [Module Structure](#module-structure)
3. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple)](#plural-data-source-list-multiple)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
4. [Resource Implementation](#resource-implementation)
   - [Resource Schema Definition](#resource-schema-definition)
   - [Create Method](#create-method)
   - [Read Method](#read-method)
   - [Update Method (PATCH)](#update-method-patch)
   - [Delete Method](#delete-method)
   - [ImportState Method](#importstate-method)
5. [Transform Functions](#transform-functions)
6. [Provider Registration](#provider-registration)

---

## SDK Selection

Cache Settings use the **V4 SDK (`azion-api`)**:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Cache Settings (Singular Data Source) | `azion-api` (v4) | `api.ApplicationsCacheSettingsAPI` | `https://api.azion.com/v4` |
| Cache Settings (Plural Data Source) | `azion-api` (v4) | `api.ApplicationsCacheSettingsAPI` | `https://api.azion.com/v4` |
| Cache Settings (Resource) | `azion-api` (v4) | `api.ApplicationsCacheSettingsAPI` | `https://api.azion.com/v4` |

### API Endpoint Path Pattern

**CRITICAL:** The V4 SDK uses the following URL path pattern for cache settings:

```
/v4/edge_applications/{application_id}/cache_settings/{cache_setting_id}
```

The full URL is: `https://api.azion.com/v4/edge_applications/{application_id}/cache_settings/{cache_setting_id}`

### API Methods

```go
// V4 SDK Pattern
r.client.api.ApplicationsCacheSettingsAPI.RetrieveCacheSetting(ctx, applicationId, cacheSettingId).Execute()
r.client.api.ApplicationsCacheSettingsAPI.ListCacheSettings(ctx, applicationId).Page(page).PageSize(pageSize).Execute()
r.client.api.ApplicationsCacheSettingsAPI.CreateCacheSetting(ctx, applicationId).CacheSettingRequest(request).Execute()
r.client.api.ApplicationsCacheSettingsAPI.PartialUpdateCacheSetting(ctx, applicationId, cacheSettingId).PatchedCacheSettingRequest(request).Execute()
r.client.api.ApplicationsCacheSettingsAPI.DeleteCacheSetting(ctx, applicationId, cacheSettingId).Execute()
```

### SDK Validation Issue - Read Method Workaround

**IMPORTANT:** The SDK's `RetrieveCacheSetting` method has a validation issue where it expects the response body to be a `CacheSetting` directly, but the actual API returns a wrapper structure: `{"data": {...}}`.

This causes validation errors like: `"no value given for required property id"` when using the SDK directly.

**Workaround:** Use a raw HTTP request that manually parses the response:

```go
// retrieveCacheSettingRawDS makes a raw HTTP request and manually parses the response
// to work around SDK validation issues with the data wrapper.
func retrieveCacheSettingRawDS(ctx context.Context, client *apiClient, applicationId, cacheSettingId int64) (*azionapi.CacheSetting, error) {
    // Build the request URL - note the actual endpoint path
    url := fmt.Sprintf("https://api.azion.com/v4/edge_applications/%d/cache_settings/%d", applicationId, cacheSettingId)

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers from the SDK config
    for k, v := range client.apiConfig.DefaultHeader {
        httpReq.Header.Set(k, v)
    }
    httpReq.Header.Set("User-Agent", client.apiConfig.UserAgent)

    // Get HTTP client from config, or use default
    httpClient := client.apiConfig.HTTPClient
    if httpClient == nil {
        httpClient = http.DefaultClient
    }

    // Execute the request
    httpResp, err := httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer httpResp.Body.Close()

    // Read the response body
    bodyBytes, err := io.ReadAll(httpResp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    // Check for error status codes
    if httpResp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("404")
    }
    if httpResp.StatusCode >= 400 {
        return nil, fmt.Errorf("API error: %s - %s", httpResp.Status, string(bodyBytes))
    }

    // Parse the response - the API returns {"data": {...}} wrapper
    var wrapper struct {
        Data azionapi.CacheSetting `json:"data"`
    }
    if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
        return nil, fmt.Errorf("failed to parse response '%s': %w", string(bodyBytes), err)
    }

    return &wrapper.Data, nil
}
```

Use this function in the Read method instead of calling the SDK's `RetrieveCacheSetting` directly.

---

## V4 API Structure

### Response Types

```go
// Single cache setting response
CacheSettingResponse {
    State *string
    Data  CacheSetting
}

// Cache setting model
CacheSetting {
    Id           int64
    Name         string
    BrowserCache *BrowserCacheModule
    Modules      *CacheSettingsModules
    CreatedAt    NullableTime
}

// List response
PaginatedCacheSettingList {
    Count      *int64
    TotalPages *int64
    Page       *int64
    PageSize   *int64
    Next       NullableString
    Previous   NullableString
    Results    []CacheSetting
}
```

### Module Structure

```go
// Browser Cache Module
BrowserCacheModule {
    Behavior *string  // "override", "honor", "no-cache"
    MaxAge   *int64
}

// Cache Settings Modules (container)
CacheSettingsModules {
    Cache                  *CacheSettingsEdgeCacheModule
    ApplicationAccelerator *CacheSettingsApplicationAcceleratorModule
}

// Edge Cache Module
CacheSettingsEdgeCacheModule {
    Behavior       *string
    MaxAge         *int64
    StaleCache     *StateCacheModule
    LargeFileCache *LargeFileCacheModule
    TieredCache    NullableCacheSettingsTieredCacheModule
}

// Application Accelerator Module
CacheSettingsApplicationAcceleratorModule {
    CacheVaryByMethod       []string
    CacheVaryByQuerystring  *CacheVaryByQuerystringModule
    CacheVaryByCookies      *CacheVaryByCookiesModule
    CacheVaryByDevices      *CacheVaryByDevicesModule
}

// Query String Module
CacheVaryByQuerystringModule {
    Behavior     *string     // "ignore", "all", "allowlist", "denylist"
    Fields       []string
    SortEnabled  *bool
}

// Cookies Module
CacheVaryByCookiesModule {
    Behavior    *string     // "ignore", "all", "allowlist", "denylist"
    CookieNames []string
}

// Devices Module
CacheVaryByDevicesModule {
    Behavior    *string     // "ignore", "allowlist"
    DeviceGroup []int64
}

// Stale Cache Module
StateCacheModule {
    Enabled *bool
}

// Tiered Cache Module
CacheSettingsTieredCacheModule {
    Topology *string  // "nearest-region", "br-east-1", "us-east-1"
    Enabled  *bool
}

// Large File Cache Module
LargeFileCacheModule {
    Enabled *bool
    Offset  *int64
}
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single cache setting by its identifier:

File: `internal/data_source_application_cache_setting.go`

```go
package provider

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &CacheSettingDataSource{}
    _ datasource.DataSourceWithConfigure = &CacheSettingDataSource{}
)

// Constructor function
func dataSourceAzionApplicationCacheSetting() datasource.DataSource {
    return &CacheSettingDataSource{}
}

// DataSource struct - holds the client
type CacheSettingDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state for singular data source
type CacheSettingDataSourceModel struct {
    ApplicationID types.Int64        `tfsdk:"application_id"`
    Results       *CacheSettingModel `tfsdk:"results"`
    ID            types.Int64        `tfsdk:"id"`
}

// CacheSettingModel struct - represents the API response data
type CacheSettingModel struct {
    ID           types.Int64                `tfsdk:"id"`
    Name         types.String               `tfsdk:"name"`
    BrowserCache *BrowserCacheModuleModel   `tfsdk:"browser_cache"`
    Modules      *CacheSettingsModulesModel `tfsdk:"modules"`
    CreatedAt    types.String               `tfsdk:"created_at"`
}

type BrowserCacheModuleModel struct {
    Behavior types.String `tfsdk:"behavior"`
    MaxAge   types.Int64  `tfsdk:"max_age"`
}

type CacheSettingsModulesModel struct {
    Cache                  *CacheSettingsEdgeCacheModuleModel        `tfsdk:"cache"`
    ApplicationAccelerator *CacheSettingsApplicationAcceleratorModel `tfsdk:"application_accelerator"`
}

type CacheSettingsEdgeCacheModuleModel struct {
    Behavior       types.String                   `tfsdk:"behavior"`
    MaxAge         types.Int64                    `tfsdk:"max_age"`
    StaleCache     *StateCacheModuleModel         `tfsdk:"stale_cache"`
    LargeFileCache *LargeFileCacheModuleModel     `tfsdk:"large_file_cache"`
    TieredCache    *CacheSettingsTieredCacheModel `tfsdk:"tiered_cache"`
}

type CacheSettingsApplicationAcceleratorModel struct {
    CacheVaryByMethod      []types.String                     `tfsdk:"cache_vary_by_method"`
    CacheVaryByQuerystring *CacheVaryByQuerystringModuleModel  `tfsdk:"cache_vary_by_querystring"`
    CacheVaryByCookies     *CacheVaryByCookiesModuleModel      `tfsdk:"cache_vary_by_cookies"`
    CacheVaryByDevices     *CacheVaryByDevicesModuleModel      `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringModuleModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    Fields      []types.String `tfsdk:"fields"`
    SortEnabled types.Bool     `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesModuleModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    CookieNames []types.String `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesModuleModel struct {
    Behavior    types.String  `tfsdk:"behavior"`
    DeviceGroup []types.Int64 `tfsdk:"device_group"`
}

type StateCacheModuleModel struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type CacheSettingsTieredCacheModel struct {
    Topology types.String `tfsdk:"topology"`
    Enabled  types.Bool   `tfsdk:"enabled"`
}

type LargeFileCacheModuleModel struct {
    Enabled types.Bool  `tfsdk:"enabled"`
    Offset  types.Int64 `tfsdk:"offset"`
}

// Metadata - sets the data source type name
func (d *CacheSettingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_cache_setting"
}

// Schema - defines the Terraform schema
func (d *CacheSettingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Application.",
                Required:    true,
            },
            "results": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The cache setting identifier.",
                        Required:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the cache setting.",
                        Computed:    true,
                    },
                    "browser_cache": schema.SingleNestedAttribute{
                        Description: "Browser cache settings.",
                        Computed:    true,
                        Attributes: map[string]schema.Attribute{
                            "behavior": schema.StringAttribute{
                                Description: "Browser cache behavior: override, honor, no-cache.",
                                Computed:    true,
                            },
                            "max_age": schema.Int64Attribute{
                                Description: "Maximum TTL for browser cache.",
                                Computed:    true,
                            },
                        },
                    },
                    "modules": schema.SingleNestedAttribute{
                        Description: "Cache settings modules.",
                        Computed:    true,
                        Attributes: map[string]schema.Attribute{
                            "cache": schema.SingleNestedAttribute{
                                Description: "Edge cache module settings.",
                                Computed:    true,
                                Attributes: map[string]schema.Attribute{
                                    "behavior": schema.StringAttribute{
                                        Description: "Cache behavior: honor, override.",
                                        Computed:    true,
                                    },
                                    "max_age": schema.Int64Attribute{
                                        Description: "Maximum TTL for edge cache.",
                                        Computed:    true,
                                    },
                                    "stale_cache": schema.SingleNestedAttribute{
                                        Description: "Stale cache settings.",
                                        Computed:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Computed: true,
                                            },
                                        },
                                    },
                                    "large_file_cache": schema.SingleNestedAttribute{
                                        Description: "Large file cache settings.",
                                        Computed:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Computed: true,
                                            },
                                            "offset": schema.Int64Attribute{
                                                Computed: true,
                                            },
                                        },
                                    },
                                    "tiered_cache": schema.SingleNestedAttribute{
                                        Description: "Tiered cache settings.",
                                        Computed:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "topology": schema.StringAttribute{
                                                Description: "Tiered cache topology.",
                                                Computed:    true,
                                            },
                                            "enabled": schema.BoolAttribute{
                                                Computed: true,
                                            },
                                        },
                                    },
                                },
                            },
                            "application_accelerator": schema.SingleNestedAttribute{
                                Description: "Application accelerator module settings.",
                                Computed:    true,
                                Attributes: map[string]schema.Attribute{
                                    "cache_vary_by_method": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Computed:    true,
                                    },
                                    "cache_vary_by_querystring": schema.SingleNestedAttribute{
                                        Computed: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Computed: true},
                                            "fields": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Computed:    true,
                                            },
                                            "sort_enabled": schema.BoolAttribute{Computed: true},
                                        },
                                    },
                                    "cache_vary_by_cookies": schema.SingleNestedAttribute{
                                        Computed: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Computed: true},
                                            "cookie_names": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Computed:    true,
                                            },
                                        },
                                    },
                                    "cache_vary_by_devices": schema.SingleNestedAttribute{
                                        Computed: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{Computed: true},
                                            "device_group": schema.ListAttribute{
                                                ElementType: types.Int64Type,
                                                Computed:    true,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                    "created_at": schema.StringAttribute{
                        Description: "The creation timestamp of the cache setting.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}

// Configure - receives the API client from the provider
func (d *CacheSettingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

// Read - performs the API call and updates state
func (d *CacheSettingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var cacheSettingID types.Int64

    diags := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    diags = req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &cacheSettingID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Use raw HTTP request to work around SDK validation issue with data wrapper
    cacheSetting, err := retrieveCacheSettingRawDS(ctx, d.client, applicationID.ValueInt64(), cacheSettingID.ValueInt64())
    if err != nil {
        if err.Error() == "404" {
            resp.Diagnostics.AddError("Cache setting not found", "")
            return
        }
        resp.Diagnostics.AddError("Failed to retrieve cache setting", err.Error())
        return
    }

    // Transform API response to state model
    result := transformCacheSettingToModel(cacheSetting)

    state := CacheSettingDataSourceModel{
        ApplicationID: applicationID,
        Results:       result,
        ID:            types.Int64Value(cacheSettingID.ValueInt64()),
    }

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

### Plural Data Source (List Multiple)

For listing multiple cache settings with pagination support:

File: `internal/data_source_application_cache_settings.go`

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
    _ datasource.DataSource              = &CacheSettingsDataSource{}
    _ datasource.DataSourceWithConfigure = &CacheSettingsDataSource{}
)

// Constructor function
func dataSourceAzionApplicationCacheSettings() datasource.DataSource {
    return &CacheSettingsDataSource{}
}

// DataSource struct - holds the client
type CacheSettingsDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state for plural data source
type CacheSettingsDataSourceModel struct {
    ApplicationID types.Int64         `tfsdk:"application_id"`
    Counter       types.Int64         `tfsdk:"counter"`
    Page          types.Int64         `tfsdk:"page"`
    PageSize      types.Int64         `tfsdk:"page_size"`
    TotalPages    types.Int64         `tfsdk:"total_pages"`
    Links         *LinksModel         `tfsdk:"links"`
    Results       []CacheSettingModel `tfsdk:"results"`
    ID            types.Int64         `tfsdk:"id"`
}

type LinksModel struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

// Metadata - sets the data source type name (note plural naming)
func (d *CacheSettingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_cache_settings"
}

// Schema - defines the Terraform schema for plural data source
func (d *CacheSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Application.",
                Required:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total number of Cache Settings.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The page number of Cache Settings.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The Page Size number of Cache Settings.",
                Optional:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
                Computed:    true,
            },
            "links": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "previous": schema.StringAttribute{
                        Computed: true,
                    },
                    "next": schema.StringAttribute{
                        Computed: true,
                    },
                },
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        // Same nested attributes as singular data source
                        // (id, name, browser_cache, modules, created_at)
                    },
                },
            },
        },
    }
}

// Configure - receives the API client from the provider
func (d *CacheSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

// Read - performs the API call and updates state
func (d *CacheSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var page types.Int64
    var pageSize types.Int64

    diags := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Set defaults for pagination
    if page.IsNull() || page.IsUnknown() {
        page = types.Int64Value(1)
    }
    if pageSize.IsNull() || pageSize.IsUnknown() {
        pageSize = types.Int64Value(10)
    }

    // Make the API call
    listResponse, response, err := d.client.api.ApplicationsCacheSettingsAPI.
        ListCacheSettings(ctx, applicationID.ValueInt64()).
        Page(page.ValueInt64()).
        PageSize(pageSize.ValueInt64()).
        Execute()

    // Handle errors
    if err != nil {
        if response.StatusCode == 429 {
            listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCacheSettingList, *http.Response, error) {
                return d.client.api.ApplicationsCacheSettingsAPI.
                    ListCacheSettings(ctx, applicationID.ValueInt64()).
                    Page(page.ValueInt64()).
                    PageSize(pageSize.ValueInt64()).
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
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    // Build state
    state := CacheSettingsDataSourceModel{
        ApplicationID: applicationID,
        Page:          page,
        PageSize:      pageSize,
    }

    if listResponse.HasCount() {
        state.Counter = types.Int64Value(listResponse.GetCount())
    }
    if listResponse.HasTotalPages() {
        state.TotalPages = types.Int64Value(listResponse.GetTotalPages())
    }

    // Links
    var previous, next string
    if listResponse.HasPrevious() {
        prev := listResponse.GetPrevious()
        if prev != "" {
            previous = prev
        }
    }
    if listResponse.HasNext() {
        n := listResponse.GetNext()
        if n != "" {
            next = n
        }
    }
    state.Links = &LinksModel{
        Previous: types.StringValue(previous),
        Next:     types.StringValue(next),
    }

    // Results
    if listResponse.HasResults() {
        for _, cs := range listResponse.GetResults() {
            state.Results = append(state.Results, *transformCacheSettingToModel(&cs))
        }
    }

    state.ID = types.Int64Value(1) // Placeholder ID
    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular Data Source | Plural Data Source |
|--------|---------------------|-------------------|
| **File Name** | `data_source_application_cache_setting.go` | `data_source_application_cache_settings.go` |
| **Type Name** | `azion_application_cache_setting` | `azion_application_cache_settings` |
| **ID Field** | `Computed` (set after read) | `Computed` (set after read) |
| **Results** | `SingleNestedAttribute` (single object) | `ListNestedAttribute` (array of objects) |
| **Pagination** | No pagination fields | Has `page`, `page_size`, `counter`, `total_pages`, `links` |
| **API Method** | `RetrieveCacheSetting` (via raw HTTP) | `ListCacheSettings` |
| **Required Input** | `application_id` + `results.id` | `application_id` only |

---

## Resource Implementation

File: `internal/resource_application_cache_setting.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ resource.Resource                = &applicationCacheSettingsResource{}
    _ resource.ResourceWithConfigure   = &applicationCacheSettingsResource{}
    _ resource.ResourceWithImportState = &applicationCacheSettingsResource{}
)

// Constructor function
func NewApplicationCacheSettingsResource() resource.Resource {
    return &applicationCacheSettingsResource{}
}

// Resource struct - holds the client
type applicationCacheSettingsResource struct {
    client *apiClient
}

// Resource Model - represents Terraform state
type ApplicationCacheSettingsResourceModel struct {
    ApplicationID types.Int64                `tfsdk:"application_id"`
    CacheSetting  *CacheSettingResourceModel `tfsdk:"cache_setting"`
    ID            types.Int64                `tfsdk:"id"`
    LastUpdated   types.String               `tfsdk:"last_updated"`
}

// CacheSettingResourceModel - V4 API structure
type CacheSettingResourceModel struct {
    ID           types.Int64                        `tfsdk:"id"`
    Name         types.String                       `tfsdk:"name"`
    BrowserCache *BrowserCacheResourceModel         `tfsdk:"browser_cache"`
    Modules      *CacheSettingsModulesResourceModel `tfsdk:"modules"`
    CreatedAt    types.String                       `tfsdk:"created_at"`
}

type BrowserCacheResourceModel struct {
    Behavior types.String `tfsdk:"behavior"`
    MaxAge   types.Int64  `tfsdk:"max_age"`
}

type CacheSettingsModulesResourceModel struct {
    Cache                  *CacheSettingsCacheResourceModel          `tfsdk:"cache"`
    ApplicationAccelerator *CacheSettingsAppAcceleratorResourceModel `tfsdk:"application_accelerator"`
}

type CacheSettingsCacheResourceModel struct {
    Behavior       types.String                           `tfsdk:"behavior"`
    MaxAge         types.Int64                            `tfsdk:"max_age"`
    StaleCache     *StateCacheResourceModel               `tfsdk:"stale_cache"`
    LargeFileCache *LargeFileCacheResourceModel           `tfsdk:"large_file_cache"`
    TieredCache    *CacheSettingsTieredCacheResourceModel `tfsdk:"tiered_cache"`
}

type CacheSettingsAppAcceleratorResourceModel struct {
    CacheVaryByMethod      []types.String                       `tfsdk:"cache_vary_by_method"`
    CacheVaryByQuerystring *CacheVaryByQuerystringResourceModel `tfsdk:"cache_vary_by_querystring"`
    CacheVaryByCookies     *CacheVaryByCookiesResourceModel     `tfsdk:"cache_vary_by_cookies"`
    CacheVaryByDevices     *CacheVaryByDevicesResourceModel     `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringResourceModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    Fields      []types.String `tfsdk:"fields"`
    SortEnabled types.Bool     `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesResourceModel struct {
    Behavior    types.String   `tfsdk:"behavior"`
    CookieNames []types.String `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesResourceModel struct {
    Behavior    types.String  `tfsdk:"behavior"`
    DeviceGroup []types.Int64 `tfsdk:"device_group"`
}

type StateCacheResourceModel struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type CacheSettingsTieredCacheResourceModel struct {
    Topology types.String `tfsdk:"topology"`
    Enabled  types.Bool   `tfsdk:"enabled"`
}

type LargeFileCacheResourceModel struct {
    Enabled types.Bool  `tfsdk:"enabled"`
    Offset  types.Int64 `tfsdk:"offset"`
}

// Metadata - sets the resource type name
func (r *applicationCacheSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_cache_setting"
}

// Configure - receives the API client from the provider
func (r *applicationCacheSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}
```

---

## Resource Schema Definition

```go
func (r *applicationCacheSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Description: "Resource identifier.",
                Computed:    true,
                PlanModifiers: []planmodifier.Int64{
                    int64planmodifier.UseStateForUnknown(),
                },
            },
            "application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Application.",
                Required:    true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "cache_setting": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "Cache setting identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the cache setting.",
                        Required:    true,
                    },
                    "browser_cache": schema.SingleNestedAttribute{
                        Description: "Browser cache settings.",
                        Optional:    true,
                        Attributes: map[string]schema.Attribute{
                            "behavior": schema.StringAttribute{
                                Description: "Browser cache behavior: override, honor, no-cache.",
                                Optional:    true,
                            },
                            "max_age": schema.Int64Attribute{
                                Description: "Maximum TTL for browser cache.",
                                Optional:    true,
                            },
                        },
                    },
                    "modules": schema.SingleNestedAttribute{
                        Description: "Cache settings modules.",
                        Optional:    true,
                        Attributes: map[string]schema.Attribute{
                            "cache": schema.SingleNestedAttribute{
                                Description: "Edge cache module settings.",
                                Optional:    true,
                                Attributes: map[string]schema.Attribute{
                                    "behavior": schema.StringAttribute{
                                        Description: "Cache behavior: honor, override.",
                                        Optional:    true,
                                    },
                                    "max_age": schema.Int64Attribute{
                                        Description: "Maximum TTL for edge cache.",
                                        Optional:    true,
                                    },
                                    "stale_cache": schema.SingleNestedAttribute{
                                        Description: "Stale cache settings.",
                                        Optional:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                    "large_file_cache": schema.SingleNestedAttribute{
                                        Description: "Large file cache settings.",
                                        Optional:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                            "offset": schema.Int64Attribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                    "tiered_cache": schema.SingleNestedAttribute{
                                        Description: "Tiered cache settings.",
                                        Optional:    true,
                                        Attributes: map[string]schema.Attribute{
                                            "topology": schema.StringAttribute{
                                                Description: "Tiered cache topology: nearest-region, br-east-1, us-east-1.",
                                                Optional:    true,
                                            },
                                            "enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                },
                            },
                            "application_accelerator": schema.SingleNestedAttribute{
                                Description: "Application accelerator module settings.",
                                Optional:    true,
                                Attributes: map[string]schema.Attribute{
                                    "cache_vary_by_method": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Optional:    true,
                                    },
                                    "cache_vary_by_querystring": schema.SingleNestedAttribute{
                                        Optional: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{
                                                Description: "Query string behavior: ignore, all, allowlist, denylist.",
                                                Optional:    true,
                                            },
                                            "fields": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Optional:    true,
                                            },
                                            "sort_enabled": schema.BoolAttribute{
                                                Optional: true,
                                            },
                                        },
                                    },
                                    "cache_vary_by_cookies": schema.SingleNestedAttribute{
                                        Optional: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{
                                                Description: "Cookies behavior: ignore, all, allowlist, denylist.",
                                                Optional:    true,
                                            },
                                            "cookie_names": schema.ListAttribute{
                                                ElementType: types.StringType,
                                                Optional:    true,
                                            },
                                        },
                                    },
                                    "cache_vary_by_devices": schema.SingleNestedAttribute{
                                        Optional: true,
                                        Attributes: map[string]schema.Attribute{
                                            "behavior": schema.StringAttribute{
                                                Description: "Devices behavior: ignore, allowlist.",
                                                Optional:    true,
                                            },
                                            "device_group": schema.ListAttribute{
                                                ElementType: types.Int64Type,
                                                Optional:    true,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                    "created_at": schema.StringAttribute{
                        Description: "The creation timestamp.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}
```

---

## Create Method

```go
func (r *applicationCacheSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan ApplicationCacheSettingsResourceModel
    var applicationID types.Int64

    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the V4 API request
    cacheSettingRequest := azionapi.NewCacheSettingRequest(plan.CacheSetting.Name.ValueString())

    // Browser Cache
    if plan.CacheSetting.BrowserCache != nil {
        browserCache := azionapi.NewBrowserCacheModuleRequest()
        if !plan.CacheSetting.BrowserCache.Behavior.IsNull() {
            browserCache.SetBehavior(plan.CacheSetting.BrowserCache.Behavior.ValueString())
        }
        if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() {
            browserCache.SetMaxAge(plan.CacheSetting.BrowserCache.MaxAge.ValueInt64())
        }
        cacheSettingRequest.SetBrowserCache(*browserCache)
    }

    // Modules - build nested request objects
    if plan.CacheSetting.Modules != nil {
        modulesRequest := buildModulesRequest(plan.CacheSetting.Modules)
        cacheSettingRequest.SetModules(*modulesRequest)
    }

    // Call V4 API
    createdCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
        CreateCacheSetting(ctx, applicationID.ValueInt64()).
        CacheSettingRequest(*cacheSettingRequest).
        Execute()

    if err != nil {
        if response.StatusCode == 429 {
            createdCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
                return r.client.api.ApplicationsCacheSettingsAPI.
                    CreateCacheSetting(ctx, applicationID.ValueInt64()).
                    CacheSettingRequest(*cacheSettingRequest).
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
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    // Transform response to state
    plan.CacheSetting = transformCacheSettingResponseToModel(createdCacheSetting.GetData())
    plan.ID = types.Int64Value(createdCacheSetting.GetData().GetId())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}
```

---

## Read Method

**IMPORTANT:** The Read method must use a raw HTTP request function to work around the SDK validation issue.

```go
func (r *applicationCacheSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state ApplicationCacheSettingsResourceModel

    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationId := state.ApplicationID.ValueInt64()
    cacheSettingId := state.CacheSetting.ID.ValueInt64()

    // Use raw HTTP request to work around SDK validation issue
    cacheSetting, err := retrieveCacheSettingRawDS(ctx, r.client, applicationId, cacheSettingId)
    if err != nil {
        if err.Error() == "404" {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Failed to retrieve cache setting", err.Error())
        return
    }

    // Update state with response
    state.CacheSetting = transformCacheSettingResponseToResourceModel(cacheSetting)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Update Method (PATCH)

The Cache Settings resource uses PATCH for partial updates:

```go
func (r *applicationCacheSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan ApplicationCacheSettingsResourceModel

    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationId := plan.ApplicationID.ValueInt64()
    cacheSettingId := plan.CacheSetting.ID.ValueInt64()

    // Build patched request (only include fields that changed)
    patchedRequest := azionapi.NewPatchedCacheSettingRequest()

    if !plan.CacheSetting.Name.IsNull() {
        patchedRequest.SetName(plan.CacheSetting.Name.ValueString())
    }

    // Browser Cache
    if plan.CacheSetting.BrowserCache != nil {
        browserCache := azionapi.NewBrowserCacheModuleRequest()
        if !plan.CacheSetting.BrowserCache.Behavior.IsNull() {
            browserCache.SetBehavior(plan.CacheSetting.BrowserCache.Behavior.ValueString())
        }
        if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() {
            browserCache.SetMaxAge(plan.CacheSetting.BrowserCache.MaxAge.ValueInt64())
        }
        patchedRequest.SetBrowserCache(*browserCache)
    }

    // Modules
    if plan.CacheSetting.Modules != nil {
        modulesRequest := buildModulesRequest(plan.CacheSetting.Modules)
        patchedRequest.SetModules(*modulesRequest)
    }

    // Call V4 API PATCH
    updatedCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
        PartialUpdateCacheSetting(ctx, applicationId, cacheSettingId).
        PatchedCacheSettingRequest(*patchedRequest).
        Execute()

    if err != nil {
        if response.StatusCode == 429 {
            updatedCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
                return r.client.api.ApplicationsCacheSettingsAPI.
                    PartialUpdateCacheSetting(ctx, applicationId, cacheSettingId).
                    PatchedCacheSettingRequest(*patchedRequest).
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
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    // Update state
    plan.CacheSetting = transformCacheSettingResponseToModel(updatedCacheSetting.GetData())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}
```

---

## Delete Method

```go
func (r *applicationCacheSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state ApplicationCacheSettingsResourceModel

    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    applicationId := state.ApplicationID.ValueInt64()
    cacheSettingId := state.CacheSetting.ID.ValueInt64()

    _, response, err := r.client.api.ApplicationsCacheSettingsAPI.
        DeleteCacheSetting(ctx, applicationId, cacheSettingId).
        Execute()

    if err != nil {
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.ApplicationsCacheSettingsAPI.
                    DeleteCacheSetting(ctx, applicationId, cacheSettingId).
                    Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else if response.StatusCode != http.StatusNotFound {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }
}
```

---

## ImportState Method

**IMPORTANT:** The ImportState method must also use the raw HTTP request function for the same reason as the Read method.

```go
func (r *applicationCacheSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Expected format: {application_id}/{cache_setting_id}
    parts := strings.Split(req.ID, "/")
    if len(parts) != 2 {
        resp.Diagnostics.AddError(
            "Invalid import ID format",
            "Expected format: {application_id}/{cache_setting_id}",
        )
        return
    }

    applicationId, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid application ID", err.Error())
        return
    }

    cacheSettingId, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid cache setting ID", err.Error())
        return
    }

    // Set the application ID
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationId)...)

    // Read the cache setting using raw HTTP to work around SDK validation issue
    cacheSetting, err := retrieveCacheSettingRawDS(ctx, r.client, applicationId, cacheSettingId)
    if err != nil {
        if err.Error() == "404" {
            resp.Diagnostics.AddError("Cache setting not found", "")
            return
        }
        resp.Diagnostics.AddError("Failed to retrieve cache setting", err.Error())
        return
    }

    // Build state
    state := ApplicationCacheSettingsResourceModel{
        ApplicationID: types.Int64Value(applicationId),
        CacheSetting:  transformCacheSettingResponseToResourceModel(cacheSetting),
        ID:            types.Int64Value(cacheSettingId),
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Transform Functions

### Transform API Response to Data Source Model

```go
func transformCacheSettingToModel(cs *azionapi.CacheSetting) *CacheSettingModel {
    if cs == nil {
        return nil
    }

    model := &CacheSettingModel{
        ID:   types.Int64Value(cs.GetId()),
        Name: types.StringValue(cs.GetName()),
    }

    // CreatedAt - handle NullableTime
    if cs.CreatedAt.IsSet() && cs.CreatedAt.Get() != nil {
        model.CreatedAt = types.StringValue(cs.GetCreatedAt().Format(time.RFC3339))
    }

    // Browser Cache
    if cs.HasBrowserCache() {
        bc := cs.GetBrowserCache()
        model.BrowserCache = &BrowserCacheModuleModel{}
        if bc.HasBehavior() {
            model.BrowserCache.Behavior = types.StringValue(bc.GetBehavior())
        }
        if bc.HasMaxAge() {
            model.BrowserCache.MaxAge = types.Int64Value(bc.GetMaxAge())
        }
    }

    // Modules
    if cs.HasModules() {
        modules := cs.GetModules()
        model.Modules = &CacheSettingsModulesModel{}

        // Cache (Edge Cache)
        if modules.HasCache() {
            cache := modules.GetCache()
            model.Modules.Cache = &CacheSettingsEdgeCacheModuleModel{}

            if cache.HasBehavior() {
                model.Modules.Cache.Behavior = types.StringValue(cache.GetBehavior())
            }
            if cache.HasMaxAge() {
                model.Modules.Cache.MaxAge = types.Int64Value(cache.GetMaxAge())
            }

            // Stale Cache
            if cache.HasStaleCache() {
                sc := cache.GetStaleCache()
                model.Modules.Cache.StaleCache = &StateCacheModuleModel{}
                if sc.HasEnabled() {
                    model.Modules.Cache.StaleCache.Enabled = types.BoolValue(sc.GetEnabled())
                }
            }

            // Large File Cache
            if cache.HasLargeFileCache() {
                lfc := cache.GetLargeFileCache()
                model.Modules.Cache.LargeFileCache = &LargeFileCacheModuleModel{}
                if lfc.HasEnabled() {
                    model.Modules.Cache.LargeFileCache.Enabled = types.BoolValue(lfc.GetEnabled())
                }
                if lfc.HasOffset() {
                    model.Modules.Cache.LargeFileCache.Offset = types.Int64Value(lfc.GetOffset())
                }
            }

            // Tiered Cache
            if cache.HasTieredCache() {
                tc := cache.GetTieredCache()
                model.Modules.Cache.TieredCache = &CacheSettingsTieredCacheModel{}
                if tc.HasTopology() {
                    model.Modules.Cache.TieredCache.Topology = types.StringValue(tc.GetTopology())
                }
                if tc.HasEnabled() {
                    model.Modules.Cache.TieredCache.Enabled = types.BoolValue(tc.GetEnabled())
                }
            }
        }

        // Application Accelerator
        if modules.HasApplicationAccelerator() {
            aa := modules.GetApplicationAccelerator()
            model.Modules.ApplicationAccelerator = &CacheSettingsApplicationAcceleratorModel{}

            // Cache Vary By Method
            if aa.HasCacheVaryByMethod() {
                for _, method := range aa.GetCacheVaryByMethod() {
                    model.Modules.ApplicationAccelerator.CacheVaryByMethod = append(
                        model.Modules.ApplicationAccelerator.CacheVaryByMethod,
                        types.StringValue(method),
                    )
                }
            }

            // Cache Vary By Querystring
            if aa.HasCacheVaryByQuerystring() {
                qs := aa.GetCacheVaryByQuerystring()
                model.Modules.ApplicationAccelerator.CacheVaryByQuerystring = &CacheVaryByQuerystringModuleModel{}

                if qs.HasBehavior() {
                    model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Behavior = types.StringValue(qs.GetBehavior())
                }
                if qs.HasFields() {
                    for _, f := range qs.GetFields() {
                        model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields = append(
                            model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields,
                            types.StringValue(f),
                        )
                    }
                }
                if qs.HasSortEnabled() {
                    model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.SortEnabled = types.BoolValue(qs.GetSortEnabled())
                }
            }

            // Cache Vary By Cookies
            if aa.HasCacheVaryByCookies() {
                cookies := aa.GetCacheVaryByCookies()
                model.Modules.ApplicationAccelerator.CacheVaryByCookies = &CacheVaryByCookiesModuleModel{}

                if cookies.HasBehavior() {
                    model.Modules.ApplicationAccelerator.CacheVaryByCookies.Behavior = types.StringValue(cookies.GetBehavior())
                }
                if cookies.HasCookieNames() {
                    for _, cn := range cookies.GetCookieNames() {
                        model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames = append(
                            model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames,
                            types.StringValue(cn),
                        )
                    }
                }
            }

            // Cache Vary By Devices
            if aa.HasCacheVaryByDevices() {
                devices := aa.GetCacheVaryByDevices()
                model.Modules.ApplicationAccelerator.CacheVaryByDevices = &CacheVaryByDevicesModuleModel{}

                if devices.HasBehavior() {
                    model.Modules.ApplicationAccelerator.CacheVaryByDevices.Behavior = types.StringValue(devices.GetBehavior())
                }
                if devices.HasDeviceGroup() {
                    for _, dg := range devices.GetDeviceGroup() {
                        model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup = append(
                            model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup,
                            types.Int64Value(dg),
                        )
                    }
                }
            }
        }
    }

    return model
}
```

### Helper: Build Modules Request

```go
func buildModulesRequest(modules *CacheSettingsModulesResourceModel) *azionapi.CacheSettingsModulesRequest {
    modulesRequest := azionapi.NewCacheSettingsModulesRequest()

    if modules.Cache != nil {
        cacheRequest := azionapi.NewCacheSettingsEdgeCacheModuleRequest()

        if !modules.Cache.Behavior.IsNull() {
            cacheRequest.SetBehavior(modules.Cache.Behavior.ValueString())
        }
        if !modules.Cache.MaxAge.IsNull() {
            cacheRequest.SetMaxAge(modules.Cache.MaxAge.ValueInt64())
        }

        if modules.Cache.StaleCache != nil {
            staleCache := azionapi.NewStateCacheModuleRequest()
            if !modules.Cache.StaleCache.Enabled.IsNull() {
                staleCache.SetEnabled(modules.Cache.StaleCache.Enabled.ValueBool())
            }
            cacheRequest.SetStaleCache(*staleCache)
        }

        if modules.Cache.TieredCache != nil {
            tieredCache := azionapi.NewCacheSettingsTieredCacheModuleRequest()
            if !modules.Cache.TieredCache.Topology.IsNull() {
                tieredCache.SetTopology(modules.Cache.TieredCache.Topology.ValueString())
            }
            if !modules.Cache.TieredCache.Enabled.IsNull() {
                tieredCache.SetEnabled(modules.Cache.TieredCache.Enabled.ValueBool())
            }
            cacheRequest.SetTieredCache(*tieredCache)
        }

        if modules.Cache.LargeFileCache != nil {
            largeFileCache := azionapi.NewLargeFileCacheModuleRequest()
            if !modules.Cache.LargeFileCache.Enabled.IsNull() {
                largeFileCache.SetEnabled(modules.Cache.LargeFileCache.Enabled.ValueBool())
            }
            if !modules.Cache.LargeFileCache.Offset.IsNull() {
                largeFileCache.SetOffset(modules.Cache.LargeFileCache.Offset.ValueInt64())
            }
            cacheRequest.SetLargeFileCache(*largeFileCache)
        }

        modulesRequest.SetCache(*cacheRequest)
    }

    if modules.ApplicationAccelerator != nil {
        aa := modules.ApplicationAccelerator
        aaRequest := azionapi.NewCacheSettingsApplicationAcceleratorModuleRequest()

        if len(aa.CacheVaryByMethod) > 0 {
            var methods []string
            for _, m := range aa.CacheVaryByMethod {
                methods = append(methods, m.ValueString())
            }
            aaRequest.SetCacheVaryByMethod(methods)
        }

        if aa.CacheVaryByQuerystring != nil {
            qs := buildQuerystringRequest(aa.CacheVaryByQuerystring)
            aaRequest.SetCacheVaryByQuerystring(*qs)
        }

        if aa.CacheVaryByCookies != nil {
            cookies := buildCookiesRequest(aa.CacheVaryByCookies)
            aaRequest.SetCacheVaryByCookies(*cookies)
        }

        if aa.CacheVaryByDevices != nil {
            devices := buildDevicesRequest(aa.CacheVaryByDevices)
            aaRequest.SetCacheVaryByDevices(*devices)
        }

        modulesRequest.SetApplicationAccelerator(*aaRequest)
    }

    return modulesRequest
}

func buildQuerystringRequest(qs *CacheVaryByQuerystringResourceModel) *azionapi.CacheVaryByQuerystringModuleRequest {
    request := azionapi.NewCacheVaryByQuerystringModuleRequest()

    if !qs.Behavior.IsNull() {
        request.SetBehavior(qs.Behavior.ValueString())
    }
    if len(qs.Fields) > 0 {
        var fields []string
        for _, f := range qs.Fields {
            fields = append(fields, f.ValueString())
        }
        request.SetFields(fields)
    }
    if !qs.SortEnabled.IsNull() {
        request.SetSortEnabled(qs.SortEnabled.ValueBool())
    }

    return request
}

func buildCookiesRequest(cookies *CacheVaryByCookiesResourceModel) *azionapi.CacheVaryByCookiesModuleRequest {
    request := azionapi.NewCacheVaryByCookiesModuleRequest()

    if !cookies.Behavior.IsNull() {
        request.SetBehavior(cookies.Behavior.ValueString())
    }
    if len(cookies.CookieNames) > 0 {
        var names []string
        for _, n := range cookies.CookieNames {
            names = append(names, n.ValueString())
        }
        request.SetCookieNames(names)
    }

    return request
}

func buildDevicesRequest(devices *CacheVaryByDevicesResourceModel) *azionapi.CacheVaryByDevicesModuleRequest {
    request := azionapi.NewCacheVaryByDevicesModuleRequest()

    if !devices.Behavior.IsNull() {
        request.SetBehavior(devices.Behavior.ValueString())
    }
    if len(devices.DeviceGroup) > 0 {
        var groups []int64
        for _, g := range devices.DeviceGroup {
            groups = append(groups, g.ValueInt64())
        }
        request.SetDeviceGroup(groups)
    }

    return request
}
```

---

## Documentation and Examples

### MANDATORY: Parent Resource Documentation

**IMPORTANT**: Cache Settings is a child resource of `azion_application_main_setting`. Documentation and examples MUST include the parent resource creation to show complete context.

When updating documentation, always include:

1. **Parent Application Example** - Show creation of the parent application first
2. **Reference Using Terraform Interpolation** - Use `azion_application_main_setting.example.application.application_id` to reference the parent ID

### Documentation Files

Documentation is auto-generated by `terraform-plugin-docs` and located in:

| Type | Location |
|------|----------|
| Singular Data Source Doc | `docs/data-sources/application_cache_setting.md` |
| Plural Data Source Doc | `docs/data-sources/application_cache_settings.md` |
| Resource Doc | `docs/resources/application_cache_setting.md` |

### Example Files

Example Terraform configurations are located in:

| Type | Location |
|------|----------|
| Singular Data Source Example | `examples/data-sources/azion_application_cache_setting/data-source.tf` |
| Plural Data Source Example | `examples/data-sources/azion_application_cache_settings/data-source.tf` |
| Resource Example | `examples/resources/azion_application_cache_setting/resource.tf` |

### Example: Complete Resource Usage with Parent Application

```terraform
# First, create the parent application with edge cache enabled
resource "azion_application_main_setting" "example" {
  application = {
    name   = "My Application"
    active = true
    modules = {
      edge_cache = {
        enabled = true
      }
    }
  }
}

# Then create the cache setting for that application
resource "azion_application_cache_setting" "example" {
  application_id = azion_application_main_setting.example.application.application_id
  cache_setting = {
    name = "My Cache Setting"
    browser_cache = {
      behavior = "override"
      max_age  = 3600
    }
  }
}
```

---

## Provider Registration

Register in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionApplicationCacheSetting,
        dataSourceAzionApplicationCacheSettings,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewApplicationCacheSettingsResource,
        // ... other resources
    }
}
```

---

## Summary Checklist

When implementing or updating Cache Settings resources and data sources:

1. **Use the correct SDK**: V4 (`azion-api` package)
2. **Use raw HTTP for Read**: Work around SDK validation issue with `retrieveCacheSettingRawDS`
3. **Use PATCH for updates**: `PartialUpdateCacheSetting` with `PatchedCacheSettingRequest`
4. **ID type is `int64`**: Not `string`
5. **Import format**: `{application_id}/{cache_setting_id}`
6. **Handle 429 errors**: Use `utils.RetryOn429`
7. **Handle nullable fields**: Check `IsNull()` and `IsUnknown()` before accessing values
8. **Transform nested objects**: Use helper functions for complex module structures
9. **Register in provider.go**: Add to both DataSources() and Resources()
10. **Run linters**: `golangci-lint run --config .golintci.yml ./internal/...`
