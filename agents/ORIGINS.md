# Origins - Code Generation Guide

This document provides specific guidance for implementing Origins resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Origin Data Structures](#origin-data-structures)
3. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by Key)](#singular-data-source-read-by-key)
   - [Plural Data Source (List Multiple Origins)](#plural-data-source-list-multiple-origins)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
4. [Resource Implementation](#resource-implementation)
5. [Schema Definition Patterns](#schema-definition-patterns)
6. [Transform Functions](#transform-functions)
7. [Common Issues](#common-issues)

---

## SDK Selection

Origins currently use **multiple SDKs** depending on the API version:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Origins (Legacy/V3) | `azionapi-go-sdk/edgeapplications` | `edgeApplicationsApi.EdgeApplicationsOriginsAPI` | Configurable |
| Origins (V4) | `azionapi-v4-go-sdk-dev/edge-api` | `edgeApi.ApplicationsCacheSettingsAPI` (embedded) | `https://api.azion.com/v4` |

### Important: V4 SDK Origin Support

**The V4 SDK (`edge-api`) does not have a dedicated OriginsAPIService.** Origins in V4 are managed through:

1. **Application-level configuration** - Origins are embedded within Application resources
2. **Cache Settings API** - `ApplicationsCacheSettingsAPIService` references origins by ID
3. **Rules Engine behaviors** - `BehaviorSetOrigin` sets the origin for requests

For V4 implementations, origins are referenced by ID within:
- `CacheSetting` → `Modules.CacheSettingsEdgeCacheModule` references origin settings
- `BehaviorSetOrigin` → Sets origin in rules engine

### Key Differences Between SDKs

| Feature | Legacy SDK (`edgeapplications`) | V4 SDK (`edge-api`) |
|---------|--------------------------------|---------------------|
| ID Type | `int64` for application, `string` for origin_key | `int64` for all IDs |
| Origin Identifier | `origin_key` (string) | `id` (int64) |
| API Pattern | `EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(ctx, appID, originKey)` | Embedded in Application/CacheSettings |
| Create Pattern | `.EdgeApplicationsEdgeApplicationIdOriginsPost(ctx, id).CreateOriginsRequest(req).Execute()` | Part of Application creation |
| Update Method | PUT (full update) | N/A - managed via Application |
| Response Type | `OriginsIdResponse.Results.GetOriginKey()` | Embedded in Application response |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev) - for Applications, Cache Settings
    edgeConfig *edgeapi.Configuration
    edgeApi    *edgeapi.APIClient
    
    // Legacy SDKs (azionapi-go-sdk) - for Origins (current implementation)
    edgeApplicationsApi *edgeapplications.APIClient
    // ... more SDK clients
}
```

---

## Origin Data Structures

### Legacy SDK (Current Implementation)

The current origin implementation uses the legacy SDK with these fields:

```go
// Origin model - Legacy SDK
type OriginResourceResults struct {
    OriginID                   types.Int64     `tfsdk:"origin_id"`
    OriginKey                  types.String    `tfsdk:"origin_key"`      // String identifier
    Name                       types.String    `tfsdk:"name"`
    OriginType                 types.String    `tfsdk:"origin_type"`     // single_origin, load_balancer, live_ingest
    Addresses                  []OriginAddress `tfsdk:"addresses"`
    OriginProtocolPolicy       types.String    `tfsdk:"origin_protocol_policy"` // preserve, http, https
    IsOriginRedirectionEnabled types.Bool      `tfsdk:"is_origin_redirection_enabled"`
    HostHeader                 types.String    `tfsdk:"host_header"`
    Method                     types.String    `tfsdk:"method"`
    OriginPath                 types.String    `tfsdk:"origin_path"`
    ConnectionTimeout          types.Int64     `tfsdk:"connection_timeout"`
    TimeoutBetweenBytes        types.Int64     `tfsdk:"timeout_between_bytes"`
    HMACAuthentication         types.Bool      `tfsdk:"hmac_authentication"`
    HMACRegionName             types.String    `tfsdk:"hmac_region_name"`
    HMACAccessKey              types.String    `tfsdk:"hmac_access_key"`
    HMACSecretKey              types.String    `tfsdk:"hmac_secret_key"`
}

type OriginAddress struct {
    Address    types.String `tfsdk:"address"`
    Weight     types.Int64  `tfsdk:"weight"`
    ServerRole types.String `tfsdk:"server_role"`  // primary, backup
    IsActive   types.Bool   `tfsdk:"is_active"`
}
```

### V4 SDK Structures

In V4, origins are represented through the `Address` struct:

```go
// V4 SDK - Address struct
type Address struct {
    Active    *bool              `json:"active,omitempty"`     // Indicates if the address is active
    Address   string             `json:"address"`              // IPv4/IPv6 address or CNAME
    HttpPort  *int64             `json:"http_port,omitempty"`  // Port for HTTP connections
    HttpsPort *int64             `json:"https_port,omitempty"` // Port for HTTPS connections
    Modules   NullableAddressModules `json:"modules,omitempty"`
}

// V4 SDK - AddressModules for load balancing
type AddressModules struct {
    LoadBalancer *AddressLoadBalancerModule `json:"load_balancer,omitempty"`
}

// V4 SDK - LoadBalancer module per address
type AddressLoadBalancerModule struct {
    ServerRole *string `json:"server_role,omitempty"` // primary, backup
    Weight     *int64  `json:"weight,omitempty"`      // Weight for load balancing
}
```

### V4 Origin References in Behaviors

```go
// V4 SDK - Behavior to set origin in Rules Engine
type BehaviorSetOrigin struct {
    Type       string                      `json:"type"`              // "set_origin"
    Attributes BehaviorSetOriginAttributes `json:"attributes"`
}

type BehaviorSetOriginAttributes struct {
    Value int64 `json:"value"` // Origin ID to target
}
```

---

## Data Source Implementation

### Singular Data Source (Read by Key)

For reading a single origin by its key (legacy SDK):

```go
package provider

import (
    "context"
    "io"
    "net/http"

    "github.com/aziontech/azionapi-go-sdk/edgeapplications"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &OriginDataSource{}
    _ datasource.DataSourceWithConfigure = &OriginDataSource{}
)

// Constructor function
func dataSourceAzionEdgeApplicationOrigin() datasource.DataSource {
    return &OriginDataSource{}
}

// DataSource struct - holds the client
type OriginDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state
type OriginDataSourceModel struct {
    SchemaVersion types.Int64   `tfsdk:"schema_version"`
    ID            types.String  `tfsdk:"id"`
    ApplicationID types.Int64   `tfsdk:"edge_application_id"`  // Parent ID
    Results       OriginResults `tfsdk:"origin"`
}

// Results struct - represents the API response data
type OriginResults struct {
    OriginId                   types.Int64            `tfsdk:"origin_id"`
    OriginKey                  types.String           `tfsdk:"origin_key"`
    Name                       types.String           `tfsdk:"name"`
    OriginType                 types.String           `tfsdk:"origin_type"`
    Addresses                  []OriginAddressResults `tfsdk:"addresses"`
    OriginProtocolPolicy       types.String           `tfsdk:"origin_protocol_policy"`
    IsOriginRedirectionEnabled types.Bool             `tfsdk:"is_origin_redirection_enabled"`
    HostHeader                 types.String           `tfsdk:"host_header"`
    Method                     types.String           `tfsdk:"method"`
    OriginPath                 types.String           `tfsdk:"origin_path"`
    ConnectionTimeout          types.Int64            `tfsdk:"connection_timeout"`
    TimeoutBetweenBytes        types.Int64            `tfsdk:"timeout_between_bytes"`
    HMACAuthentication         types.Bool             `tfsdk:"hmac_authentication"`
    HMACRegionName             types.String           `tfsdk:"hmac_region_name"`
    HMACAccessKey              types.String           `tfsdk:"hmac_access_key"`
    HMACSecretKey              types.String           `tfsdk:"hmac_secret_key"`
}

// OriginAddressResults - nested address structure
type OriginAddressResults struct {
    Address    types.String `tfsdk:"address"`
    Weight     types.Int64  `tfsdk:"weight"`
    ServerRole types.String `tfsdk:"server_role"`
    IsActive   types.Bool   `tfsdk:"is_active"`
}

// Metadata - sets the data source type name
func (o *OriginDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_origin"
}

// Schema - defines the Terraform schema
func (o *OriginDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "The edge application identifier.",
                Required:    true,  // Parent ID is required
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "origin": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "origin_id": schema.Int64Attribute{
                        Description: "The origin identifier to target for the resource.",
                        Computed:    true,
                    },
                    "origin_key": schema.StringAttribute{
                        Description: "Origin key.",
                        Required:    true,  // User must provide origin_key to look up
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the origin.",
                        Computed:    true,
                    },
                    "origin_type": schema.StringAttribute{
                        Description: "Type of the origin.",
                        Computed:    true,
                    },
                    "addresses": schema.ListNestedAttribute{
                        Computed: true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "address": schema.StringAttribute{
                                    Description: "Address of the origin.",
                                    Computed:    true,
                                },
                                "weight": schema.Int64Attribute{
                                    Description: "Weight of the origin.",
                                    Computed:    true,
                                },
                                "server_role": schema.StringAttribute{
                                    Description: "Server role of the origin.",
                                    Computed:    true,
                                },
                                "is_active": schema.BoolAttribute{
                                    Description: "Status of the origin.",
                                    Computed:    true,
                                },
                            },
                        },
                    },
                    // ... other computed attributes
                },
            },
        },
    }
}

// Configure - receives the API client from the provider
func (o *OriginDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    o.client = req.ProviderData.(*apiClient)
}

// Read - performs the API call and updates state
func (o *OriginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // 1. Get both parent ID and origin_key from config
    var edgeApplicationID types.Int64
    var getOriginsKey types.String
    
    diags := req.Config.GetAttribute(ctx, path.Root("origin").AtName("origin_key"), &getOriginsKey)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    if getOriginsKey.ValueString() == "" {
        resp.Diagnostics.AddError("Origin Key error ", "is not null")
        return
    }

    diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
    resp.Diagnostics.Append(diagsEdgeApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Make the API call - requires both application ID and origin key
    originResponse, response, err := o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(
            ctx, 
            edgeApplicationID.ValueInt64(), 
            getOriginsKey.ValueString(),
        ).Execute()
    
    // 3. Handle errors (including 429 rate limiting)
    if err != nil {
        if response.StatusCode == 429 {
            originResponse, response, err = utils.RetryOn429(func() (*edgeapplications.OriginsIdResponse, *http.Response, error) {
                return o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
                    EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(
                        ctx, 
                        edgeApplicationID.ValueInt64(), 
                        getOriginsKey.ValueString(),
                    ).Execute()
            }, 5) // Maximum 5 retries

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

    // 4. Transform response to state model
    var addresses []OriginAddressResults
    for _, addr := range originResponse.Results.Addresses {
        addresses = append(addresses, OriginAddressResults{
            Address:    types.StringValue(addr.GetAddress()),
            Weight:     types.Int64Value(addr.GetWeight()),
            ServerRole: types.StringValue(addr.GetServerRole()),
            IsActive:   types.BoolValue(addr.GetIsActive()),
        })
    }

    origin := OriginResults{
        OriginId:                   types.Int64Value(originResponse.Results.GetOriginId()),
        OriginKey:                  types.StringValue(originResponse.Results.GetOriginKey()),
        Name:                       types.StringValue(originResponse.Results.GetName()),
        OriginType:                 types.StringValue(originResponse.Results.GetOriginType()),
        Addresses:                  addresses,
        OriginProtocolPolicy:       types.StringValue(originResponse.Results.GetOriginProtocolPolicy()),
        IsOriginRedirectionEnabled: types.BoolValue(originResponse.Results.GetIsOriginRedirectionEnabled()),
        HostHeader:                 types.StringValue(originResponse.Results.GetHostHeader()),
        Method:                     types.StringValue(originResponse.Results.GetMethod()),
        OriginPath:                 types.StringValue(originResponse.Results.GetOriginPath()),
        ConnectionTimeout:          types.Int64Value(originResponse.Results.GetConnectionTimeout()),
        TimeoutBetweenBytes:        types.Int64Value(originResponse.Results.GetTimeoutBetweenBytes()),
        HMACAuthentication:         types.BoolValue(originResponse.Results.GetHmacAuthentication()),
        HMACRegionName:             types.StringValue(originResponse.Results.GetHmacRegionName()),
        HMACAccessKey:              types.StringValue(originResponse.Results.GetHmacAccessKey()),
        HMACSecretKey:              types.StringValue(originResponse.Results.GetHmacSecretKey()),
    }

    // 5. Set state
    state := OriginDataSourceModel{
        SchemaVersion: types.Int64Value(originResponse.SchemaVersion),
        Results:       origin,
    }
    state.ID = types.StringValue("Get By Key Edge Application Origins")
    
    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Plural Data Source (List Multiple Origins)

For listing all origins of an application with pagination support:

#### Complete Plural Data Source Structure

```go
package provider

import (
    "context"
    "io"
    "net/http"

    "github.com/aziontech/azionapi-go-sdk/edgeapplications"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &OriginsDataSource{}
    _ datasource.DataSourceWithConfigure = &OriginsDataSource{}
)

// Constructor function
func dataSourceAzionEdgeApplicationsOrigins() datasource.DataSource {
    return &OriginsDataSource{}
}

// DataSource struct - holds the client
type OriginsDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state for plural data source
type OriginsDataSourceModel struct {
    SchemaVersion types.Int64                              `tfsdk:"schema_version"`
    ID            types.String                             `tfsdk:"id"`
    ApplicationID types.Int64                              `tfsdk:"edge_application_id"`
    Counter       types.Int64                              `tfsdk:"counter"`
    TotalPages    types.Int64                              `tfsdk:"total_pages"`
    Page          types.Int64                              `tfsdk:"page"`
    PageSize      types.Int64                              `tfsdk:"page_size"`
    Links         *GetEdgeApplicationsOriginsResponseLinks `tfsdk:"links"`
    Results       []OriginsResults                         `tfsdk:"results"`
}

// Links struct for pagination
type GetEdgeApplicationsOriginsResponseLinks struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

// OriginsResults - represents each item in the results list
type OriginsResults struct {
    OriginId                   types.Int64             `tfsdk:"origin_id"`
    OriginKey                  types.String            `tfsdk:"origin_key"`
    Name                       types.String            `tfsdk:"name"`
    OriginType                 types.String            `tfsdk:"origin_type"`
    Addresses                  []OriginsAddressResults `tfsdk:"addresses"`
    OriginProtocolPolicy       types.String            `tfsdk:"origin_protocol_policy"`
    IsOriginRedirectionEnabled types.Bool              `tfsdk:"is_origin_redirection_enabled"`
    HostHeader                 types.String            `tfsdk:"host_header"`
    Method                     types.String            `tfsdk:"method"`
    OriginPath                 types.String            `tfsdk:"origin_path"`
    ConnectionTimeout          types.Int64             `tfsdk:"connection_timeout"`
    TimeoutBetweenBytes        types.Int64             `tfsdk:"timeout_between_bytes"`
    HMACAuthentication         types.Bool              `tfsdk:"hmac_authentication"`
    HMACRegionName             types.String            `tfsdk:"hmac_region_name"`
    HMACAccessKey              types.String            `tfsdk:"hmac_access_key"`
    HMACSecretKey              types.String            `tfsdk:"hmac_secret_key"`
}

// OriginsAddressResults - nested address structure
type OriginsAddressResults struct {
    Address    types.String `tfsdk:"address"`
    Weight     types.Int64  `tfsdk:"weight"`
    ServerRole types.String `tfsdk:"server_role"`
    IsActive   types.Bool   `tfsdk:"is_active"`
}
```

#### Metadata Method

```go
func (o *OriginsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    // Note: plural naming convention
    resp.TypeName = req.ProviderTypeName + "_edge_applications_origins"
}
```

#### Schema Method

The plural data source schema differs from singular in key ways:
- `edge_application_id` is **Required** (parent ID to list origins for)
- Includes pagination fields (`page`, `page_size`) as **Optional**
- Uses `ListNestedAttribute` for results instead of `SingleNestedAttribute`

```go
func (o *OriginsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,  // Computed, not Required
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "The edge application identifier.",
                Required:    true,  // Must specify which application's origins to list
            },
            "counter": schema.Int64Attribute{
                Description: "The total number of origins.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The page number of origins.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The Page Size number of origins.",
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
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "origin_id": schema.Int64Attribute{
                            Description: "The origin identifier.",
                            Computed:    true,
                        },
                        "origin_key": schema.StringAttribute{
                            Description: "Origin key.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "Name of the origin.",
                            Computed:    true,
                        },
                        "origin_type": schema.StringAttribute{
                            Description: "Type of the origin.",
                            Computed:    true,
                        },
                        "addresses": schema.ListNestedAttribute{
                            Computed: true,
                            NestedObject: schema.NestedAttributeObject{
                                Attributes: map[string]schema.Attribute{
                                    "address": schema.StringAttribute{Computed: true},
                                    "weight": schema.Int64Attribute{Computed: true},
                                    "server_role": schema.StringAttribute{Computed: true},
                                    "is_active": schema.BoolAttribute{Computed: true},
                                },
                            },
                        },
                        // ... other computed attributes
                    },
                },
            },
        },
    }
}
```

#### Read Method

The plural Read method handles pagination and builds a list of results:

```go
func (o *OriginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // 1. Get required parent ID and optional pagination parameters
    var edgeApplicationID types.Int64
    var Page types.Int64
    var PageSize types.Int64

    diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
    resp.Diagnostics.Append(diagsEdgeApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &Page)
    resp.Diagnostics.Append(diagsPage...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &PageSize)
    resp.Diagnostics.Append(diagsPageSize...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Set default values for pagination
    if Page.ValueInt64() == 0 {
        Page = types.Int64Value(1)
    }
    if PageSize.ValueInt64() == 0 {
        PageSize = types.Int64Value(10)
    }

    // 3. Make the API call
    originsResponse, response, err := o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsGet(ctx, edgeApplicationID.ValueInt64()).Execute()
    
    // 4. Handle errors (including 429 rate limiting)
    if err != nil {
        if response.StatusCode == 429 {
            originsResponse, response, err = utils.RetryOn429(func() (*edgeapplications.OriginsResponse, *http.Response, error) {
                return o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
                    EdgeApplicationsEdgeApplicationIdOriginsGet(ctx, edgeApplicationID.ValueInt64()).Execute()
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

    // 5. Extract pagination links
    var previous, next string
    if originsResponse.Links.Previous.Get() != nil {
        previous = *originsResponse.Links.Previous.Get()
    }
    if originsResponse.Links.Next.Get() != nil {
        next = *originsResponse.Links.Next.Get()
    }

    // 6. Iterate over results and transform each origin
    var origins []OriginsResults
    for _, origin := range originsResponse.Results {
        var addresses []OriginsAddressResults
        for _, addr := range origin.Addresses {
            addresses = append(addresses, OriginsAddressResults{
                Address:    types.StringValue(addr.GetAddress()),
                Weight:     types.Int64Value(addr.GetWeight()),
                ServerRole: types.StringValue(addr.GetServerRole()),
                IsActive:   types.BoolValue(addr.GetIsActive()),
            })
        }

        origins = append(origins, OriginsResults{
            OriginId:                   types.Int64Value(origin.GetOriginId()),
            OriginKey:                  types.StringValue(origin.GetOriginKey()),
            Name:                       types.StringValue(origin.GetName()),
            OriginType:                 types.StringValue(origin.GetOriginType()),
            Addresses:                  addresses,
            OriginProtocolPolicy:       types.StringValue(origin.GetOriginProtocolPolicy()),
            IsOriginRedirectionEnabled: types.BoolValue(origin.GetIsOriginRedirectionEnabled()),
            HostHeader:                 types.StringValue(origin.GetHostHeader()),
            Method:                     types.StringValue(origin.GetMethod()),
            OriginPath:                 types.StringValue(origin.GetOriginPath()),
            ConnectionTimeout:          types.Int64Value(origin.GetConnectionTimeout()),
            TimeoutBetweenBytes:        types.Int64Value(origin.GetTimeoutBetweenBytes()),
            HMACAuthentication:         types.BoolValue(origin.GetHmacAuthentication()),
            HMACRegionName:             types.StringValue(origin.GetHmacRegionName()),
            HMACAccessKey:              types.StringValue(origin.GetHmacAccessKey()),
            HMACSecretKey:              types.StringValue(origin.GetHmacSecretKey()),
        })
    }

    // 7. Build state
    state := OriginsDataSourceModel{
        SchemaVersion: types.Int64Value(originsResponse.SchemaVersion),
        Results:       origins,
        TotalPages:    types.Int64Value(originsResponse.TotalPages),
        Counter:       types.Int64Value(originsResponse.Count),
        Links: &GetEdgeApplicationsOriginsResponseLinks{
            Previous: types.StringValue(previous),
            Next:     types.StringValue(next),
        },
    }

    // 8. Set descriptive ID
    state.ID = types.StringValue("Get All Edge Application Origins")

    // 9. Set state
    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

#### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`azion_edge_application_origin`) | Plural (`azion_edge_applications_origins`) |
|--------|--------------------------------------------|-------------------------------------------|
| **Parent ID Field** | Required (`edge_application_id`) | Required (`edge_application_id`) |
| **Origin Key** | Required inside `origin` block | Not applicable (lists all) |
| **Schema Root** | `origin` (SingleNestedAttribute) | `results` (ListNestedAttribute) |
| **Pagination** | Not applicable | `page`, `page_size` (Optional) |
| **Counter Field** | Not applicable | `counter` (Computed) |
| **Links Field** | Not applicable | `links` (Computed) |
| **API Method** | `EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(ctx, appID, key)` | `EdgeApplicationsEdgeApplicationIdOriginsGet(ctx, appID)` |
| **Response Type** | `*edgeapplications.OriginsIdResponse` | `*edgeapplications.OriginsResponse` |
| **State ID Value** | `"Get By Key Edge Application Origins"` | `"Get All Edge Application Origins"` |

#### File Naming Convention

| Type | File Name | Data Source Name |
|------|-----------|------------------|
| Singular | `data_source_edge_application_origin.go` | `azion_edge_application_origin` |
| Plural | `data_source_edge_applications_origins.go` | `azion_edge_applications_origins` |

Note: The plural form adds an "s" after "application" in the data source name.

---

## Resource Implementation

### Complete Resource Structure

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/aziontech/azionapi-go-sdk/edgeapplications"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ resource.Resource                = &originResource{}
    _ resource.ResourceWithConfigure   = &originResource{}
    _ resource.ResourceWithImportState = &originResource{}
)

// Constructor
func NewEdgeApplicationOriginResource() resource.Resource {
    return &originResource{}
}

// Resource struct
type originResource struct {
    client *apiClient
}

// Model struct - note the wrapping pattern
type OriginResourceModel struct {
    SchemaVersion types.Int64            `tfsdk:"schema_version"`
    Origin        *OriginResourceResults `tfsdk:"origin"`
    ID            types.String           `tfsdk:"id"`
    ApplicationID types.Int64            `tfsdk:"edge_application_id"`
    LastUpdated   types.String           `tfsdk:"last_updated"`
}

// Results struct
type OriginResourceResults struct {
    OriginID                   types.Int64     `tfsdk:"origin_id"`
    OriginKey                  types.String    `tfsdk:"origin_key"`
    Name                       types.String    `tfsdk:"name"`
    OriginType                 types.String    `tfsdk:"origin_type"`
    Addresses                  []OriginAddress `tfsdk:"addresses"`
    OriginProtocolPolicy       types.String    `tfsdk:"origin_protocol_policy"`
    IsOriginRedirectionEnabled types.Bool      `tfsdk:"is_origin_redirection_enabled"`
    HostHeader                 types.String    `tfsdk:"host_header"`
    Method                     types.String    `tfsdk:"method"`
    OriginPath                 types.String    `tfsdk:"origin_path"`
    ConnectionTimeout          types.Int64     `tfsdk:"connection_timeout"`
    TimeoutBetweenBytes        types.Int64     `tfsdk:"timeout_between_bytes"`
    HMACAuthentication         types.Bool      `tfsdk:"hmac_authentication"`
    HMACRegionName             types.String    `tfsdk:"hmac_region_name"`
    HMACAccessKey              types.String    `tfsdk:"hmac_access_key"`
    HMACSecretKey              types.String    `tfsdk:"hmac_secret_key"`
}

// OriginAddress - nested address structure
type OriginAddress struct {
    Address    types.String `tfsdk:"address"`
    Weight     types.Int64  `tfsdk:"weight"`
    ServerRole types.String `tfsdk:"server_role"`
    IsActive   types.Bool   `tfsdk:"is_active"`
}

// Metadata
func (r *originResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_application_origin"
}

// Schema
func (r *originResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "edge_application_id": schema.Int64Attribute{
                Description: "The edge application identifier.",
                Required:    true,
            },
            "schema_version": schema.Int64Attribute{
                Computed: true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "origin": schema.SingleNestedAttribute{
                Description: "Origin configuration.",
                Required:    true,
                Attributes: map[string]schema.Attribute{
                    "origin_id": schema.Int64Attribute{
                        Description: "Origin identifier.",
                        Computed:    true,
                    },
                    "origin_key": schema.StringAttribute{
                        Description: "Origin key.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Origin name.",
                        Required:    true,
                    },
                    "origin_type": schema.StringAttribute{
                        Description: "Identifies the source of a record.\n" +
                            "~> **Note about Origin Type**\n" +
                            "Accepted values: `single_origin`(default), `load_balancer` and `live_ingest`\n\n",
                        Optional: true,
                        Computed: true,
                    },
                    "addresses": schema.ListNestedAttribute{
                        Required: true,
                        NestedObject: schema.NestedAttributeObject{
                            Attributes: map[string]schema.Attribute{
                                "address": schema.StringAttribute{
                                    Description: "Address of the origin.",
                                    Required:    true,
                                },
                                "weight": schema.Int64Attribute{
                                    Description: "Weight of the origin.",
                                    Optional:    true,
                                    Computed:    true,
                                },
                                "server_role": schema.StringAttribute{
                                    Description: "Server role of the origin.",
                                    Optional:    true,
                                    Computed:    true,
                                },
                                "is_active": schema.BoolAttribute{
                                    Description: "Status of the origin.",
                                    Optional:    true,
                                    Computed:    true,
                                },
                            },
                        },
                    },
                    "origin_protocol_policy": schema.StringAttribute{
                        Description: "Protocols for connection to the origin.\n" +
                            "~> **Note about Origin Protocol Policy**\n" +
                            "Accepted values: `preserve`(default), `http` and `https`\n\n",
                        Optional: true,
                        Computed: true,
                    },
                    "is_origin_redirection_enabled": schema.BoolAttribute{
                        Description: "Whether origin redirection is enabled.",
                        Computed:    true,
                    },
                    "host_header": schema.StringAttribute{
                        Description: "Host header value that will be delivered to the origin.\n" +
                            "~> **Note about Host Header**\n" +
                            "Accepted values: `${host}`(default) and must be specified with `$${host}`\n\n",
                        Required: true,
                    },
                    "method": schema.StringAttribute{
                        Description: "HTTP method used by the origin.",
                        Computed:    true,
                    },
                    "origin_path": schema.StringAttribute{
                        Description: "Path of the origin.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "connection_timeout": schema.Int64Attribute{
                        Description: "Connection timeout in seconds.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "timeout_between_bytes": schema.Int64Attribute{
                        Description: "Timeout between bytes in seconds.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "hmac_authentication": schema.BoolAttribute{
                        Description: "Whether HMAC authentication is enabled.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "hmac_region_name": schema.StringAttribute{
                        Description: "HMAC region name.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "hmac_access_key": schema.StringAttribute{
                        Description: "HMAC access key.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "hmac_secret_key": schema.StringAttribute{
                        Description: "HMAC secret key.",
                        Optional:    true,
                        Computed:    true,
                    },
                },
            },
        },
    }
}

// Configure
func (r *originResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}
```

### Create Method Pattern

```go
func (r *originResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // 1. Get the plan
    var plan OriginResourceModel
    var edgeApplicationID types.Int64
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
    resp.Diagnostics.Append(diagsEdgeApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Build addresses request
    var addressesRequest []edgeapplications.CreateOriginsRequestAddresses
    for _, addr := range plan.Origin.Addresses {
        var serverRole *string = addr.ServerRole.ValueStringPointer()
        if addr.ServerRole.ValueString() == "" {
            serverRole = nil
        }

        var weight *int64 = addr.Weight.ValueInt64Pointer()
        if addr.Weight.ValueInt64() == 0 {
            weight = nil
        }

        requestAddresses := edgeapplications.CreateOriginsRequestAddresses{
            Address:    addr.Address.ValueString(),
            IsActive:   addr.IsActive.ValueBoolPointer(),
            Weight:     weight,
            ServerRole: serverRole,
        }
        addressesRequest = append(addressesRequest, requestAddresses)
    }

    // 3. Handle optional/computed fields with defaults
    var originProtocolPolicy string
    if plan.Origin.OriginProtocolPolicy.IsUnknown() {
        originProtocolPolicy = "preserve"
    } else {
        if plan.Origin.OriginProtocolPolicy.ValueString() == "" || plan.Origin.OriginProtocolPolicy.IsNull() {
            resp.Diagnostics.AddError("Origin Protocol Policy",
                "Is not null, Possible choices are: [preserve(default), http, https]")
            return
        }
        originProtocolPolicy = plan.Origin.OriginProtocolPolicy.ValueString()
    }

    var OriginType string
    if plan.Origin.OriginType.IsUnknown() {
        OriginType = "single_origin"
    } else {
        if plan.Origin.OriginType.ValueString() == "" || plan.Origin.OriginType.IsNull() {
            resp.Diagnostics.AddError("Origin Type",
                "Is not null, Possible choices are: [single_origin]")
            return
        }
        OriginType = plan.Origin.OriginType.ValueString()
    }

    // 4. Build the SDK request object
    originRequest := edgeapplications.CreateOriginsRequest{
        Name:                 plan.Origin.Name.ValueString(),
        Addresses:            addressesRequest,
        OriginType:           edgeapplications.PtrString(OriginType),
        OriginProtocolPolicy: edgeapplications.PtrString(originProtocolPolicy),
        HostHeader:           plan.Origin.HostHeader.ValueStringPointer(),
        OriginPath:           edgeapplications.PtrString(plan.Origin.OriginPath.ValueString()),
        HmacAuthentication:   edgeapplications.PtrBool(plan.Origin.HMACAuthentication.ValueBool()),
        HmacRegionName:       edgeapplications.PtrString(plan.Origin.HMACRegionName.ValueString()),
        HmacAccessKey:        edgeapplications.PtrString(plan.Origin.HMACAccessKey.ValueString()),
        HmacSecretKey:        edgeapplications.PtrString(plan.Origin.HMACSecretKey.ValueString()),
    }

    // 5. Make the API call
    originResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsPost(ctx, edgeApplicationID.ValueInt64()).
        CreateOriginsRequest(originRequest).
        Execute()
    
    // 6. Handle errors (including 429)
    if err != nil {
        if response.StatusCode == 429 {
            originResponse, response, err = utils.RetryOn429(func() (*edgeapplications.OriginsIdResponse, *http.Response, error) {
                return r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
                    EdgeApplicationsEdgeApplicationIdOriginsPost(ctx, edgeApplicationID.ValueInt64()).
                    CreateOriginsRequest(originRequest).
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
    }

    // 7. Build the state from response
    var addresses []OriginAddress
    if len(originResponse.Results.Addresses) > 0 {
        for _, addr := range originResponse.Results.Addresses {
            addresses = append(addresses, OriginAddress{
                Address:    types.StringValue(addr.GetAddress()),
                Weight:     types.Int64Value(addr.GetWeight()),
                ServerRole: types.StringValue(addr.GetServerRole()),
                IsActive:   types.BoolValue(addr.GetIsActive()),
            })
        }
    }

    plan.Origin = &OriginResourceResults{
        OriginID:                   types.Int64Value(originResponse.Results.GetOriginId()),
        OriginKey:                  types.StringValue(originResponse.Results.GetOriginKey()),
        Name:                       types.StringValue(originResponse.Results.GetName()),
        OriginType:                 types.StringValue(originResponse.Results.GetOriginType()),
        Addresses:                  addresses,
        OriginProtocolPolicy:       types.StringValue(originResponse.Results.GetOriginProtocolPolicy()),
        IsOriginRedirectionEnabled: types.BoolValue(originResponse.Results.GetIsOriginRedirectionEnabled()),
        HostHeader:                 types.StringValue(originResponse.Results.GetHostHeader()),
        Method:                     types.StringValue(originResponse.Results.GetMethod()),
        OriginPath:                 types.StringValue(originResponse.Results.GetOriginPath()),
        ConnectionTimeout:          types.Int64Value(originResponse.Results.GetConnectionTimeout()),
        TimeoutBetweenBytes:        types.Int64Value(originResponse.Results.GetTimeoutBetweenBytes()),
        HMACAuthentication:         types.BoolValue(originResponse.Results.GetHmacAuthentication()),
        HMACRegionName:             types.StringValue(originResponse.Results.GetHmacRegionName()),
        HMACAccessKey:              types.StringValue(originResponse.Results.GetHmacAccessKey()),
        HMACSecretKey:              types.StringValue(originResponse.Results.GetHmacSecretKey()),
    }

    // 8. Set ID and timestamp
    plan.SchemaVersion = types.Int64Value(originResponse.SchemaVersion)
    plan.ID = types.StringValue(strconv.FormatInt(*originResponse.Results.OriginId, 10))
    plan.ApplicationID = edgeApplicationID
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    // 9. Set the state
    diags = resp.State.Set(ctx, &plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Method Pattern

```go
func (r *originResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // 1. Get current state
    var state OriginResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Determine IDs from state - handle both regular and import cases
    var ApplicationID int64
    var OriginKey string
    valueFromCmd := strings.Split(state.ID.ValueString(), "/")
    if len(valueFromCmd) > 1 {
        // Import case: ID format is "applicationID/originKey"
        ApplicationID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
        OriginKey = valueFromCmd[1]
    } else {
        // Normal case: get from state
        ApplicationID = state.ApplicationID.ValueInt64()
        OriginKey = state.Origin.OriginKey.ValueString()
    }

    if OriginKey == "" {
        resp.Diagnostics.AddError("Origin Key error ", "is not null")
        return
    }

    // 3. Call retrieve API
    originResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(ctx, ApplicationID, OriginKey).
        Execute()
    
    // 4. Handle 404 - resource was deleted outside Terraform
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle 429 and other errors...
    }

    // 5. Update state from response
    var addresses []OriginAddress
    for _, addr := range originResponse.Results.Addresses {
        addresses = append(addresses, OriginAddress{
            Address:    types.StringValue(addr.GetAddress()),
            Weight:     types.Int64Value(addr.GetWeight()),
            ServerRole: types.StringValue(addr.GetServerRole()),
            IsActive:   types.BoolValue(addr.GetIsActive()),
        })
    }

    state.Origin = &OriginResourceResults{
        OriginID:                   types.Int64Value(originResponse.Results.GetOriginId()),
        OriginKey:                  types.StringValue(originResponse.Results.GetOriginKey()),
        Name:                       types.StringValue(originResponse.Results.GetName()),
        OriginType:                 types.StringValue(originResponse.Results.GetOriginType()),
        Addresses:                  addresses,
        OriginProtocolPolicy:       types.StringValue(originResponse.Results.GetOriginProtocolPolicy()),
        IsOriginRedirectionEnabled: types.BoolValue(originResponse.Results.GetIsOriginRedirectionEnabled()),
        HostHeader:                 types.StringValue(originResponse.Results.GetHostHeader()),
        Method:                     types.StringValue(originResponse.Results.GetMethod()),
        OriginPath:                 types.StringValue(originResponse.Results.GetOriginPath()),
        ConnectionTimeout:          types.Int64Value(originResponse.Results.GetConnectionTimeout()),
        TimeoutBetweenBytes:        types.Int64Value(originResponse.Results.GetTimeoutBetweenBytes()),
        HMACAuthentication:         types.BoolValue(originResponse.Results.GetHmacAuthentication()),
        HMACRegionName:             types.StringValue(originResponse.Results.GetHmacRegionName()),
        HMACAccessKey:              types.StringValue(originResponse.Results.GetHmacAccessKey()),
        HMACSecretKey:              types.StringValue(originResponse.Results.GetHmacSecretKey()),
    }
    state.ID = types.StringValue(strconv.FormatInt(*originResponse.Results.OriginId, 10))
    state.ApplicationID = types.Int64Value(ApplicationID)
    state.SchemaVersion = types.Int64Value(originResponse.SchemaVersion)

    // 6. Set state
    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Update Method (PUT - Full Update)

Origins use PUT for full updates:

```go
func (r *originResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan OriginResourceModel
    var edgeApplicationID types.Int64
    var originKey types.String
    
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get previous state for IDs
    var state OriginResourceModel
    diagsOrigin := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsOrigin...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Determine origin key and application ID
    if plan.Origin.OriginKey.ValueString() == "" {
        originKey = state.Origin.OriginKey
    } else {
        originKey = plan.Origin.OriginKey
    }

    if plan.ApplicationID.IsNull() {
        edgeApplicationID = state.ApplicationID
    } else {
        edgeApplicationID = plan.ApplicationID
    }

    // Build addresses request
    var addressesRequest []edgeapplications.CreateOriginsRequestAddresses
    for _, addr := range plan.Origin.Addresses {
        var serverRole *string = addr.ServerRole.ValueStringPointer()
        if addr.ServerRole.ValueString() == "" {
            serverRole = nil
        }

        var weight *int64 = addr.Weight.ValueInt64Pointer()
        if addr.Weight.ValueInt64() == 0 {
            weight = nil
        }

        addressesRequest = append(addressesRequest, edgeapplications.CreateOriginsRequestAddresses{
            Address:    addr.Address.ValueString(),
            IsActive:   addr.IsActive.ValueBoolPointer(),
            Weight:     weight,
            ServerRole: serverRole,
        })
    }

    // Build full update request
    originRequest := edgeapplications.UpdateOriginsRequest{
        Name:                 plan.Origin.Name.ValueString(),
        Addresses:            addressesRequest,
        OriginType:           edgeapplications.PtrString(plan.Origin.OriginType.ValueString()),
        OriginProtocolPolicy: edgeapplications.PtrString(plan.Origin.OriginProtocolPolicy.ValueString()),
        HostHeader:           edgeapplications.PtrString(plan.Origin.HostHeader.ValueString()),
        OriginPath:           edgeapplications.PtrString(plan.Origin.OriginPath.ValueString()),
        HmacAuthentication:   edgeapplications.PtrBool(plan.Origin.HMACAuthentication.ValueBool()),
        HmacRegionName:       edgeapplications.PtrString(plan.Origin.HMACRegionName.ValueString()),
        HmacAccessKey:        edgeapplications.PtrString(plan.Origin.HMACAccessKey.ValueString()),
        HmacSecretKey:        edgeapplications.PtrString(plan.Origin.HMACSecretKey.ValueString()),
    }

    // PUT request
    originResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsOriginKeyPut(
            ctx, 
            edgeApplicationID.ValueInt64(), 
            originKey.ValueString(),
        ).
        UpdateOriginsRequest(originRequest).
        Execute()
    
    // Handle errors and update state...
}
```

### Delete Method Pattern

```go
func (r *originResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // 1. Get current state
    var state OriginResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Get IDs
    edgeApplicationID := state.ApplicationID.ValueInt64()

    if state.Origin.OriginKey.ValueString() == "" {
        resp.Diagnostics.AddError("Origin Key error ", "is not null")
        return
    }

    if state.ApplicationID.IsNull() {
        resp.Diagnostics.AddError("Edge Application ID error ", "is not null")
        return
    }

    // 3. Call delete API
    response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsOriginKeyDelete(
            ctx, 
            edgeApplicationID, 
            state.Origin.OriginKey.ValueString(),
        ).
        Execute()
    
    // 4. Handle errors
    if err != nil {
        if response.StatusCode == 429 {
            response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
                return r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
                    EdgeApplicationsEdgeApplicationIdOriginsOriginKeyDelete(
                        ctx, 
                        edgeApplicationID, 
                        state.Origin.OriginKey.ValueString(),
                    ).
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
    }
    
    // 5. No need to set state - resource is deleted
}
```

### ImportState Method Pattern

```go
func (r *originResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Parse composite ID: "applicationID/originKey" or just "applicationID,originKey"
    idParts := strings.Split(req.ID, "/")
    if len(idParts) != 2 {
        // Try comma separator
        idParts = strings.Split(req.ID, ",")
        if len(idParts) != 2 {
            resp.Diagnostics.AddError("Invalid import ID", "Expected format: applicationID/originKey or applicationID,originKey")
            return
        }
    }
    
    appID, err := strconv.ParseInt(idParts[0], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid application ID", "Could not parse application ID")
        return
    }
    
    resp.Diagnostics.Append(resp.State.Set(ctx, &OriginResourceModel{
        ApplicationID: types.Int64Value(appID),
        ID:            types.StringValue(req.ID),
        Origin: &OriginResourceResults{
            OriginKey: types.StringValue(idParts[1]),
        },
    })...)
}
```

---

## Schema Definition Patterns

### Attribute Types

```go
// String attribute
"name": schema.StringAttribute{
    Description: "Name of the origin.",
    Required:    true,
},

// Integer attribute
"connection_timeout": schema.Int64Attribute{
    Description: "Connection timeout in seconds.",
    Optional:    true,
    Computed:    true,
},

// Boolean attribute
"is_origin_redirection_enabled": schema.BoolAttribute{
    Description: "Whether origin redirection is enabled.",
    Computed:    true,
},

// List of nested objects - Addresses
"addresses": schema.ListNestedAttribute{
    Required: true,
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "address": schema.StringAttribute{
                Description: "Address of the origin.",
                Required:    true,
            },
            "weight": schema.Int64Attribute{
                Description: "Weight for load balancing.",
                Optional:    true,
                Computed:    true,
            },
            "server_role": schema.StringAttribute{
                Description: "Server role (primary/backup).",
                Optional:    true,
                Computed:    true,
            },
            "is_active": schema.BoolAttribute{
                Description: "Whether the address is active.",
                Optional:    true,
                Computed:    true,
            },
        },
    },
},
```

### Plan Modifiers

```go
"id": schema.StringAttribute{
    Computed: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),  // Use existing ID when planning update
    },
},
```

### Schema-Level Description

```go
func (r *originResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "" +
            "~> **Note about Origin Type**\n" +
            "Accepted values: `single_origin`(default), `load_balancer` and `live_ingest`\n\n" +
            "~> **Note about Origin Protocol Policy**\n" +
            "Accepted values: `preserve`(default), `http` and `https`\n\n" +
            "~> **Note about Host Header**\n" +
            "Accepted values: `${host}`(default) and must be specified with `$${host}`\n\n",
        Attributes: map[string]schema.Attribute{
            // ...
        },
    }
}
```

---

## Transform Functions

For building address request objects:

```go
func transformAddressesToRequest(addresses []OriginAddress) []edgeapplications.CreateOriginsRequestAddresses {
    var addressesRequest []edgeapplications.CreateOriginsRequestAddresses
    for _, addr := range addresses {
        var serverRole *string = addr.ServerRole.ValueStringPointer()
        if addr.ServerRole.ValueString() == "" {
            serverRole = nil
        }

        var weight *int64 = addr.Weight.ValueInt64Pointer()
        if addr.Weight.ValueInt64() == 0 {
            weight = nil
        }

        addressesRequest = append(addressesRequest, edgeapplications.CreateOriginsRequestAddresses{
            Address:    addr.Address.ValueString(),
            IsActive:   addr.IsActive.ValueBoolPointer(),
            Weight:     weight,
            ServerRole: serverRole,
        })
    }
    return addressesRequest
}
```

For transforming response addresses to state:

```go
func transformAddressesToState(addresses []edgeapplications.OriginsResultsAddresses) []OriginAddress {
    var result []OriginAddress
    for _, addr := range addresses {
        result = append(result, OriginAddress{
            Address:    types.StringValue(addr.GetAddress()),
            Weight:     types.Int64Value(addr.GetWeight()),
            ServerRole: types.StringValue(addr.GetServerRole()),
            IsActive:   types.BoolValue(addr.GetIsActive()),
        })
    }
    return result
}
```

---

## Common Issues

### Origin Type Validation

The `origin_type` field accepts specific values. Handle unknown/computed defaults:

```go
var OriginType string
if plan.Origin.OriginType.IsUnknown() {
    OriginType = "single_origin"  // Default value
} else {
    if plan.Origin.OriginType.ValueString() == "" || plan.Origin.OriginType.IsNull() {
        resp.Diagnostics.AddError("Origin Type",
            "Is not null, Possible choices are: [single_origin, load_balancer, live_ingest]")
        return
    }
    OriginType = plan.Origin.OriginType.ValueString()
}
```

### Origin Protocol Policy Validation

Similar handling for protocol policy:

```go
var originProtocolPolicy string
if plan.Origin.OriginProtocolPolicy.IsUnknown() {
    originProtocolPolicy = "preserve"  // Default value
} else {
    if plan.Origin.OriginProtocolPolicy.ValueString() == "" || plan.Origin.OriginProtocolPolicy.IsNull() {
        resp.Diagnostics.AddError("Origin Protocol Policy",
            "Is not null, Possible choices are: [preserve(default), http, https]")
        return
    }
    originProtocolPolicy = plan.Origin.OriginProtocolPolicy.ValueString()
}
```

### Address Weight and Server Role

Weight and server_role are optional but should handle empty/zero values:

```go
var serverRole *string = addr.ServerRole.ValueStringPointer()
if addr.ServerRole.ValueString() == "" {
    serverRole = nil  // Don't send empty string
}

var weight *int64 = addr.Weight.ValueInt64Pointer()
if addr.Weight.ValueInt64() == 0 {
    weight = nil  // Don't send zero
}
```

### Parent-Child Relationship

Origins are children of Edge Applications. Always require `edge_application_id`:

```go
"edge_application_id": schema.Int64Attribute{
    Description: "The edge application identifier.",
    Required:    true,  // Always required - origins belong to applications
},
```

### Import ID Format

The import ID should support both formats:
- `applicationID/originKey`
- `applicationID,originKey`

### 404 Handling in Read

Always handle 404 in Read to detect resources deleted outside Terraform:

```go
if response.StatusCode == http.StatusNotFound {
    resp.State.RemoveResource(ctx)
    return
}
```

---

## Summary Checklist

When implementing Origins resources or data sources:

1. **Identify the correct SDK**: Currently using legacy SDK (`edgeapplications`), V4 SDK embeds origins in Applications
2. **Parent-child relationship**: Origins require `edge_application_id` (parent application ID)
3. **Origin identifier**: Uses `origin_key` (string) in legacy SDK, `id` (int64) in V4 SDK
4. **Update method**: PUT (full update) for legacy SDK
5. **Create model structs**: With appropriate `tfsdk` tags
6. **Implement schema**: With correct Required/Optional/Computed for each field
7. **Implement all methods**: Create, Read, Update, Delete, ImportState (for resources)
8. **Handle 429 errors**: Use `utils.RetryOn429`
9. **Handle optional fields**: Check `IsNull()` and `IsUnknown()` for optional/computed
10. **Transform addresses**: Create helper functions for address transformations
11. **Handle defaults**: Set default values for `origin_type` and `origin_protocol_policy`
12. **Register in provider.go**: Add to DataSources() or Resources()
13. **Generate documentation**: Create docs and examples
