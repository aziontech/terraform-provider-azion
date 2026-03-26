# Zones - Code Generation Guide

This document provides specific guidance for implementing Zone resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
3. [Resource Implementation](#resource-implementation)
   - [Create Operation](#create-operation)
   - [Read Operation](#read-operation)
   - [Update Operation](#update-operation)
   - [Delete Operation](#delete-operation)
   - [Import State](#import-state)
4. [Schema Definition Patterns](#schema-definition-patterns)
5. [Error Handling](#error-handling)
6. [Type Conversions](#type-conversions)
7. [Common Issues](#common-issues)

---

## SDK Selection

Zones use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Zone (Singular Data Source) | `azion-api` (v4) | `api.DNSZonesAPI` | `https://api.azion.com/v4` |
| Zones (Plural Data Source) | `azion-api` (v4) | `api.DNSZonesAPI` | `https://api.azion.com/v4` |
| Zone (Resource) | `azion-api` (v4) | `api.DNSZonesAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `ZoneRequest` |
| Update Request Type | `UpdateZoneRequest` |
| Response Type | `ZoneResponse` with `Data` field |
| List Response Type | `PaginatedZoneList` |
| Create Pattern | `.CreateDnsZone(ctx).ZoneRequest(req).Execute()` |
| Update Pattern | `.UpdateDnsZone(ctx, zoneId).UpdateZoneRequest(req).Execute()` |
| Retrieve Pattern | `.RetrieveDnsZone(ctx, zoneId).Execute()` |
| List Method | `.ListDnsZones(ctx).Page(page).PageSize(pageSize).Execute()` |
| Delete Method | `.DeleteDnsZone(ctx, zoneId).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Zone by its identifier:

**File:** `internal/data_source_zone.go`

```go
package provider

import (
    "context"
    "fmt"
    "net/http"
    "strconv"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &ZoneDataSource{}
    _ datasource.DataSourceWithConfigure = &ZoneDataSource{}
)

func dataSourceAzionZone() datasource.DataSource {
    return &ZoneDataSource{}
}

type ZoneDataSource struct {
    client *apiClient
}

type ZoneDataSourceModel struct {
    Data ZoneModel  `tfsdk:"data"`
    ID   types.String `tfsdk:"id"`
}

type ZoneModel struct {
    ZoneID         types.Int64  `tfsdk:"zone_id"`
    Name           types.String `tfsdk:"name"`
    Domain         types.String `tfsdk:"domain"`
    Active         types.Bool   `tfsdk:"active"`
    Nameservers    types.List   `tfsdk:"nameservers"`
    ProductVersion types.String `tfsdk:"product_version"`
}

func (d *ZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_intelligent_dns_zone"
}

func (d *ZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Required:    true,
            },
            "data": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "zone_id": schema.Int64Attribute{
                        Description: "The zone identifier to target for the resource.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the zone.",
                        Computed:    true,
                    },
                    "domain": schema.StringAttribute{
                        Computed:    true,
                        Description: "Domain name attributed by Azion to this configuration.",
                    },
                    "active": schema.BoolAttribute{
                        Computed:    true,
                        Description: "Status of the zone.",
                    },
                    "nameservers": schema.ListAttribute{
                        Computed:    true,
                        ElementType: types.StringType,
                        Description: "List of nameservers for the zone.",
                    },
                    "product_version": schema.StringAttribute{
                        Computed:    true,
                        Description: "Product version of the zone.",
                    },
                },
            },
        },
    }
}

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getZoneId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("id"), &getZoneId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(getZoneId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    zoneResponse, response, err := d.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
                return d.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
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
            usrMsg, errMsg := errPrintZone(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    zoneData := zoneResponse.GetData()

    // Convert nameservers to Terraform List
    var nameserversList types.List
    if zoneData.GetNameservers() != nil {
        nsSlice := make([]string, len(zoneData.GetNameservers()))
        for i, ns := range zoneData.GetNameservers() {
            nsSlice[i] = ns
        }
        nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
    } else {
        nameserversList = types.ListNull(types.StringType)
    }

    zoneState := ZoneDataSourceModel{
        Data: ZoneModel{
            ZoneID:         types.Int64Value(zoneData.GetId()),
            Name:           types.StringValue(zoneData.GetName()),
            Domain:         types.StringValue(zoneData.GetDomain()),
            Active:         types.BoolValue(zoneData.GetActive()),
            Nameservers:    nameserversList,
            ProductVersion: types.StringValue(zoneData.GetProductVersion()),
        },
    }

    zoneState.ID = types.StringValue("Get By ID Zone")
    diags = resp.State.Set(ctx, &zoneState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}

func errPrintZone(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "Zone not found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

### Plural Data Source (List Multiple Resources)

For listing multiple Zones with pagination:

**File:** `internal/data_source_zones.go`

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
    "github.com/hashicorp/terraform-plugin-framework/diag"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &ZonesDataSource{}
    _ datasource.DataSourceWithConfigure = &ZonesDataSource{}
)

func dataSourceAzionZones() datasource.DataSource {
    return &ZonesDataSource{}
}

type ZonesDataSource struct {
    client *apiClient
}

type ZonesDataSourceModel struct {
    TotalCount types.Int64         `tfsdk:"total_count"`
    TotalPages types.Int64         `tfsdk:"total_pages"`
    Page       types.Int64         `tfsdk:"page"`
    PageSize   types.Int64         `tfsdk:"page_size"`
    Links      *ZonesResponseLinks `tfsdk:"links"`
    Results    []ZonesModel        `tfsdk:"results"`
    ID         types.String        `tfsdk:"id"`
}

type ZonesResponseLinks struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

type ZonesModel struct {
    ZoneID         types.Int64  `tfsdk:"zone_id"`
    Name           types.String `tfsdk:"name"`
    Domain         types.String `tfsdk:"domain"`
    Active         types.Bool   `tfsdk:"active"`
    Nameservers    types.List   `tfsdk:"nameservers"`
    ProductVersion types.String `tfsdk:"product_version"`
}

func (d *ZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *ZonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_intelligent_dns_zones"
}

func (d *ZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The page number of Zones.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The page size number of Zones.",
                Optional:    true,
            },
            "total_count": schema.Int64Attribute{
                Description: "The total number of zones.",
                Computed:    true,
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
                        "zone_id": schema.Int64Attribute{
                            Description: "The zone identifier to target for the resource.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "The name of the zone.",
                            Computed:    true,
                        },
                        "domain": schema.StringAttribute{
                            Description: "Domain name attributed by Azion to this configuration.",
                            Computed:    true,
                        },
                        "active": schema.BoolAttribute{
                            Computed:    true,
                            Description: "Status of the zone.",
                        },
                        "nameservers": schema.ListAttribute{
                            Computed:    true,
                            ElementType: types.StringType,
                            Description: "List of nameservers for the zone.",
                        },
                        "product_version": schema.StringAttribute{
                            Computed:    true,
                            Description: "Product version of the zone.",
                        },
                    },
                },
            },
        },
    }
}

func (d *ZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var Page types.Int64
    var PageSize types.Int64
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

    if Page.IsNull() || Page.IsUnknown() {
        Page = types.Int64Value(1)
    }

    if PageSize.IsNull() || PageSize.IsUnknown() {
        PageSize = types.Int64Value(10)
    }

    zoneResponse, response, err := d.client.api.DNSZonesAPI.ListDnsZones(ctx).
        Page(Page.ValueInt64()).
        PageSize(PageSize.ValueInt64()).
        Execute()
    if err != nil {
        if response.StatusCode == 429 {
            zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedZoneList, *http.Response, error) {
                return d.client.api.DNSZonesAPI.ListDnsZones(ctx).
                    Page(Page.ValueInt64()).
                    PageSize(PageSize.ValueInt64()).
                    Execute()
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
            usrMsg, errMsg := errPrintZones(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    zoneState := ZonesDataSourceModel{
        Page:     types.Int64Value(Page.ValueInt64()),
        PageSize: types.Int64Value(PageSize.ValueInt64()),
    }

    // Set optional pagination fields
    if zoneResponse.HasCount() {
        zoneState.TotalCount = types.Int64Value(zoneResponse.GetCount())
    } else {
        zoneState.TotalCount = types.Int64Value(0)
    }

    if zoneResponse.HasTotalPages() {
        zoneState.TotalPages = types.Int64Value(zoneResponse.GetTotalPages())
    } else {
        zoneState.TotalPages = types.Int64Value(0)
    }

    // Set links
    zoneState.Links = &ZonesResponseLinks{}
    if zoneResponse.HasPrevious() {
        zoneState.Links.Previous = types.StringValue(zoneResponse.GetPrevious())
    } else {
        zoneState.Links.Previous = types.StringValue("")
    }

    if zoneResponse.HasNext() {
        zoneState.Links.Next = types.StringValue(zoneResponse.GetNext())
    } else {
        zoneState.Links.Next = types.StringValue("")
    }

    // Process results
    if zoneResponse.HasResults() {
        for _, resultZone := range zoneResponse.GetResults() {
            var nameserversList types.List
            if resultZone.GetNameservers() != nil {
                nsSlice := make([]string, len(resultZone.GetNameservers()))
                for i, ns := range resultZone.GetNameservers() {
                    nsSlice[i] = ns
                }
                var diagsList diag.Diagnostics
                nameserversList, diagsList = types.ListValueFrom(ctx, types.StringType, nsSlice)
                resp.Diagnostics.Append(diagsList...)
                if resp.Diagnostics.HasError() {
                    return
                }
            } else {
                nameserversList = types.ListNull(types.StringType)
            }

            zoneState.Results = append(zoneState.Results, ZonesModel{
                ZoneID:         types.Int64Value(resultZone.GetId()),
                Name:           types.StringValue(resultZone.GetName()),
                Domain:         types.StringValue(resultZone.GetDomain()),
                Active:         types.BoolValue(resultZone.GetActive()),
                Nameservers:    nameserversList,
                ProductVersion: types.StringValue(resultZone.GetProductVersion()),
            })
        }
    }

    zoneState.ID = types.StringValue("Get All Zones")
    diags := resp.State.Set(ctx, &zoneState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}

func errPrintZones(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Zones found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`data_source_zone.go`) | Plural (`data_source_zones.go`) |
|--------|----------------------------------|--------------------------------|
| ID Attribute | Required (user provides zone ID) | Computed (generated string) |
| Results Structure | `SingleNestedAttribute` (single object) | `ListNestedAttribute` (array) |
| Model Name | `ZoneModel` | `ZonesModel` |
| Response Type | `ZoneResponse` | `PaginatedZoneList` |
| API Method | `RetrieveDnsZone(ctx, zoneId)` | `ListDnsZones(ctx).Page().PageSize()` |
| Pagination | No pagination fields | Includes `count`, `total_pages`, `links` |
| Default Values | N/A | Page defaults to 1, PageSize to 10 |

---

## Resource Implementation

### Resource Structure

**File:** `internal/resource_zones.go`

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
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
    _ resource.Resource                = &zoneResource{}
    _ resource.ResourceWithConfigure   = &zoneResource{}
    _ resource.ResourceWithImportState = &zoneResource{}
)

func NewZoneResource() resource.Resource {
    return &zoneResource{}
}

type zoneResource struct {
    client *apiClient
}

type zoneResourceModel struct {
    ID          types.String `tfsdk:"id"`
    LastUpdated types.String `tfsdk:"last_updated"`
    Zone        *zoneModel   `tfsdk:"zone"`
}

type zoneModel struct {
    ID             types.Int64  `tfsdk:"id"`
    Name           types.String `tfsdk:"name"`
    Domain         types.String `tfsdk:"domain"`
    Active         types.Bool   `tfsdk:"active"`
    Nameservers    types.List   `tfsdk:"nameservers"`
    ProductVersion types.String `tfsdk:"product_version"`
}
```

### Create Operation

```go
func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan zoneResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneRequest := azionapi.NewZoneRequest(
        plan.Zone.Name.ValueString(),
        plan.Zone.Domain.ValueString(),
        plan.Zone.Active.ValueBool(),
    )

    zoneResponse, response, err := r.client.api.DNSZonesAPI.CreateDnsZone(ctx).
        ZoneRequest(*zoneRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
                return r.client.api.DNSZonesAPI.CreateDnsZone(ctx).
                    ZoneRequest(*zoneRequest).Execute()
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
            usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    zoneData := zoneResponse.GetData()

    // Convert nameservers to Terraform List
    var nameserversList types.List
    if zoneData.GetNameservers() != nil {
        nsSlice := make([]string, len(zoneData.GetNameservers()))
        for i, ns := range zoneData.GetNameservers() {
            nsSlice[i] = ns
        }
        nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
    } else {
        nameserversList = types.ListNull(types.StringType)
    }

    plan.ID = types.StringValue(strconv.FormatInt(zoneData.GetId(), 10))
    plan.Zone = &zoneModel{
        ID:             types.Int64Value(zoneData.GetId()),
        Name:           types.StringValue(zoneData.GetName()),
        Domain:         types.StringValue(zoneData.GetDomain()),
        Active:         types.BoolValue(zoneData.GetActive()),
        Nameservers:    nameserversList,
        ProductVersion: types.StringValue(zoneData.GetProductVersion()),
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
func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state zoneResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    zoneResponse, response, err := r.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        if response.StatusCode == 429 {
            zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
                return r.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
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
            usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    zoneData := zoneResponse.GetData()

    // Convert nameservers to Terraform List
    var nameserversList types.List
    if zoneData.GetNameservers() != nil {
        nsSlice := make([]string, len(zoneData.GetNameservers()))
        for i, ns := range zoneData.GetNameservers() {
            nsSlice[i] = ns
        }
        nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
    } else {
        nameserversList = types.ListNull(types.StringType)
    }

    state.Zone = &zoneModel{
        ID:             types.Int64Value(zoneData.GetId()),
        Name:           types.StringValue(zoneData.GetName()),
        Domain:         types.StringValue(zoneData.GetDomain()),
        Active:         types.BoolValue(zoneData.GetActive()),
        Nameservers:    nameserversList,
        ProductVersion: types.StringValue(zoneData.GetProductVersion()),
    }

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Update Operation

```go
func (r *zoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan zoneResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    updateRequest := azionapi.NewUpdateZoneRequest(
        plan.Zone.Name.ValueString(),
        plan.Zone.Active.ValueBool(),
    )

    zoneResponse, response, err := r.client.api.DNSZonesAPI.UpdateDnsZone(ctx, zoneId).
        UpdateZoneRequest(*updateRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
                return r.client.api.DNSZonesAPI.UpdateDnsZone(ctx, zoneId).
                    UpdateZoneRequest(*updateRequest).Execute()
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
            usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    zoneData := zoneResponse.GetData()

    // Convert nameservers to Terraform List
    var nameserversList types.List
    if zoneData.GetNameservers() != nil {
        nsSlice := make([]string, len(zoneData.GetNameservers()))
        for i, ns := range zoneData.GetNameservers() {
            nsSlice[i] = ns
        }
        nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
    } else {
        nameserversList = types.ListNull(types.StringType)
    }

    plan.ID = types.StringValue(strconv.FormatInt(zoneData.GetId(), 10))
    plan.Zone = &zoneModel{
        ID:             types.Int64Value(zoneData.GetId()),
        Name:           types.StringValue(zoneData.GetName()),
        Domain:         types.StringValue(zoneData.GetDomain()),
        Active:         types.BoolValue(zoneData.GetActive()),
        Nameservers:    nameserversList,
        ProductVersion: types.StringValue(zoneData.GetProductVersion()),
    }
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Delete Operation

```go
func (r *zoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state zoneResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    _, response, err := r.client.api.DNSZonesAPI.DeleteDnsZone(ctx, zoneId).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.DNSZonesAPI.DeleteDnsZone(ctx, zoneId).Execute()
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
            usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }
}
```

### Import State

```go
func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

---

## Schema Definition Patterns

### Zone Model Fields

| Field | Type | SDK Field | Description |
|-------|------|-----------|-------------|
| `zone_id` / `id` | `types.Int64` | `Id` | Zone identifier |
| `name` | `types.String` | `Name` | Zone name |
| `domain` | `types.String` | `Domain` | Domain name |
| `active` | `types.Bool` | `Active` | Zone status |
| `nameservers` | `types.List` | `Nameservers` | List of nameservers |
| `product_version` | `types.String` | `ProductVersion` | Product version |

### Request Types

```go
// Create request - requires name, domain, and active
type ZoneRequest struct {
    Name   string `json:"name"`
    Domain string `json:"domain"`
    Active bool   `json:"active"`
}

// Update request - requires name and active (domain is not updatable)
type UpdateZoneRequest struct {
    Name   string `json:"name"`
    Active bool   `json:"active"`
}
```

### Response Structure

The V4 SDK returns responses in a structured format:

```go
// Single Zone response
type ZoneResponse struct {
    State *string `json:"state,omitempty"`
    Data  Zone    `json:"data"`
}

// Zone list response
type PaginatedZoneList struct {
    Count      *int64         `json:"count,omitempty"`
    TotalPages *int64         `json:"total_pages,omitempty"`
    Page       *int64         `json:"page,omitempty"`
    PageSize   *int64         `json:"page_size,omitempty"`
    Next       NullableString `json:"next,omitempty"`
    Previous   NullableString `json:"previous,omitempty"`
    Results    []Zone         `json:"results,omitempty"`
}

// Zone struct
type Zone struct {
    Id             int64    `json:"id"`
    Name           string   `json:"name"`
    Domain         string   `json:"domain"`
    Active         bool     `json:"active"`
    Nameservers    []string `json:"nameservers"`
    ProductVersion string   `json:"product_version"`
}
```

---

## Error Handling

### Standard Error Pattern

```go
func errPrintZoneResource(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "Zone not found"
    case 409:
        usrMsg = "Conflict - Zone already exists"
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
    zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
        return r.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
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
}
```

### 404 Not Found Handling (Read Operation)

```go
if response.StatusCode == http.StatusNotFound {
    resp.State.RemoveResource(ctx)
    return
}
```

---

## Type Conversions

### SDK to Terraform Types

```go
// Direct conversion
zoneId := types.Int64Value(zoneData.GetId())
name := types.StringValue(zoneData.GetName())
domain := types.StringValue(zoneData.GetDomain())
active := types.BoolValue(zoneData.GetActive())

// Slice to List conversion
var nameserversList types.List
if zoneData.GetNameservers() != nil {
    nsSlice := make([]string, len(zoneData.GetNameservers()))
    for i, ns := range zoneData.GetNameservers() {
        nsSlice[i] = ns
    }
    nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
    resp.Diagnostics.Append(diags...)
} else {
    nameserversList = types.ListNull(types.StringType)
}
```

### String ID to Int64

```go
zoneId, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
if err != nil {
    resp.Diagnostics.AddError(
        "Value Conversion error ",
        "Could not convert ID",
    )
    return
}
```

### Int64 to String ID

```go
plan.ID = types.StringValue(strconv.FormatInt(zoneData.GetId(), 10))
```

### Creating SDK Request Objects

```go
// Create request
zoneRequest := azionapi.NewZoneRequest(
    plan.Zone.Name.ValueString(),
    plan.Zone.Domain.ValueString(),
    plan.Zone.Active.ValueBool(),
)

// Update request (no domain field)
updateRequest := azionapi.NewUpdateZoneRequest(
    plan.Zone.Name.ValueString(),
    plan.Zone.Active.ValueBool(),
)
```

---

## Common Issues

### 1. Missing Response Body Closure

**Issue:** Not closing HTTP response body after successful API calls.

**Solution:** Always add `defer response.Body.Close()` after successful responses:

```go
if response != nil {
    defer response.Body.Close()
}
```

### 2. Incorrect Type Conversion for Lists

**Issue:** Using wrong type conversion pattern for list values.

**Solution:** Use `types.ListValueFrom()` with proper error handling:

```go
nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
resp.Diagnostics.Append(diags...)
if resp.Diagnostics.HasError() {
    return
}
```

### 3. Nullable Fields Handling

**Issue:** Not checking if optional fields exist before accessing.

**Solution:** Use `Has` methods for optional fields in list responses:

```go
if zoneResponse.HasCount() {
    zoneState.TotalCount = types.Int64Value(zoneResponse.GetCount())
} else {
    zoneState.TotalCount = types.Int64Value(0)
}
```

### 4. Context Propagation

**Issue:** Not passing context through function calls.

**Solution:** Always include `ctx context.Context` as first parameter and pass it through:

```go
func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // ...
    nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
}
```

### 5. Domain Field Not Updatable

**Issue:** Attempting to update the `domain` field during Update operation.

**Solution:** The `UpdateZoneRequest` only accepts `name` and `active`. The `domain` field cannot be updated after creation:

```go
// Update request - domain is NOT included
updateRequest := azionapi.NewUpdateZoneRequest(
    plan.Zone.Name.ValueString(),
    plan.Zone.Active.ValueBool(),
)
```

### 6. Using Wrong Field Names

**Issue:** Using legacy field names like `is_active` instead of V4 field names.

**Solution:** V4 SDK uses `active` (not `is_active`), and `zone_id` for data sources:

| Legacy SDK | V4 SDK |
|------------|--------|
| `is_active` | `active` |
| `IsActive` | `Active` |

---

## Summary Checklist

When generating Zone resources or data sources from OpenAPI:

- [x] Use V4 SDK (`azion-api`) import
- [x] Use `api.DNSZonesAPI` client field (not `idnsApi`)
- [x] ID type is `int64`
- [x] Response uses `ZoneResponse` with `Data` field
- [x] List response uses `PaginatedZoneList`
- [x] Create uses `ZoneRequest` with `name`, `domain`, `active`
- [x] Update uses `UpdateZoneRequest` with `name`, `active` only
- [x] Handle 429 errors with `utils.RetryOn429`
- [x] Close response body with `defer response.Body.Close()`
- [x] Use `types.ListValueFrom()` for list conversions
- [x] Use `Has` methods for optional pagination fields
- [x] Pass `ctx` to all `types.ListValueFrom()` calls
- [x] Avoid using "edge" prefix in naming (use `Zone` not `EdgeZone`)
- [x] Handle 404 in Read operation with `resp.State.RemoveResource(ctx)`
- [x] Implement `ImportState` for resource import support
