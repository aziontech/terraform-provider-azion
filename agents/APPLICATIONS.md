# Applications - Code Generation Guide

This document provides specific guidance for implementing Applications resources and data sources in the Terraform provider.

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
   - [Update Method (PUT)](#update-method-put)
   - [Delete Method](#delete-method)
   - [ImportState Method](#importstate-method)
5. [Transform Functions](#transform-functions)
6. [Common Issues](#common-issues)
7. [Provider Registration](#provider-registration)

---

## SDK Selection

Applications use the **V4 SDK (`azion-api`)** for Main Settings:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Application Main Settings (Singular Data Source) | `azion-api` (v4) | `api.ApplicationsAPI` | `https://api.azion.com/v4` |
| Application Main Settings (Plural Data Source) | `azion-api` (v4) | `api.ApplicationsAPI` | `https://api.azion.com/v4` |
| Application Main Settings (Resource) | `azion-api` (v4) | `api.ApplicationsAPI` | `https://api.azion.com/v4` |

> **Note:** Origins, Cache Settings, and Rules Engine are documented separately in their respective agent files.

### API Methods

```go
// V4 SDK Pattern
r.client.api.ApplicationsAPI.RetrieveApplication(ctx, applicationId).Execute()
r.client.api.ApplicationsAPI.ListApplications(ctx).Page(page).PageSize(pageSize).Execute()
r.client.api.ApplicationsAPI.CreateApplication(ctx).ApplicationRequest(request).Execute()
r.client.api.ApplicationsAPI.UpdateApplication(ctx, applicationId).ApplicationRequest(request).Execute()
r.client.api.ApplicationsAPI.DeleteApplication(ctx, applicationId).Execute()
```

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` for most operations |
| Update Method | `UpdateApplication` (PUT) |
| Create Pattern | `.CreateApplication(ctx).ApplicationRequest(req).Execute()` |
| Response Type | `Response.Data.GetId()` |
| List Method | `.ListApplications(ctx).Page(page).PageSize(pageSize).Execute()` |
| Delete Method | `.DeleteApplication(ctx, id).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - for Applications API
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // Legacy SDKs (azionapi-go-sdk) - deprecated
    edgeApplicationsApi *edgeapplications.APIClient
    // ... more SDK clients
}
```

---

## V4 API Structure

### Response Types

```go
// Single application response
ApplicationResponse {
    State *string
    Data  Application
}

// Application model
Application {
    Id             int64
    Name           string
    LastEditor     string
    LastModified   time.Time
    ProductVersion string
    Active         bool
    Debug          bool
    Modules        *ApplicationModules
}

// List response
PaginatedApplicationList {
    Count      *int64
    TotalPages *int64
    Page       *int64
    PageSize   *int64
    Next       NullableString
    Previous   NullableString
    Results    []Application
}
```

### Module Structure

```go
// Application Modules (container)
ApplicationModules {
    Cache                  *CacheModule
    Functions              *FunctionModule
    ApplicationAccelerator *ApplicationAcceleratorModule
    ImageProcessor         *ImageProcessorModule
}

// Cache Module
CacheModule {
    Enabled *bool
}

// Function Module
FunctionModule {
    Enabled *bool
}

// Application Accelerator Module
ApplicationAcceleratorModule {
    Enabled *bool
}

// Image Processor Module
ImageProcessorModule {
    Enabled *bool
}
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single application by its identifier:

File: `internal/data_source_application_main_settings.go` (singular) and `internal/data_source_applications_main_settings.go` (plural)

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "strconv"
    "time"

    sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &ApplicationDataSource{}
    _ datasource.DataSourceWithConfigure = &ApplicationDataSource{}
)

// Constructor function
func dataSourceAzionApplication() datasource.DataSource {
    return &ApplicationDataSource{}
}

// DataSource struct - holds the client
type ApplicationDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state for singular data source
type ApplicationDataSourceModel struct {
    SchemaVersion types.Int64      `tfsdk:"schema_version"`
    Data          *ApplicationData `tfsdk:"data"`
    ID            types.String     `tfsdk:"id"`
}

// ApplicationData struct - represents the API response data
type ApplicationData struct {
    Id             types.Int64         `tfsdk:"id"`
    Name           types.String        `tfsdk:"name"`
    LastEditor     types.String        `tfsdk:"last_editor"`
    LastModified   types.String        `tfsdk:"last_modified"` // RFC3339 as string
    Modules        *ApplicationModules `tfsdk:"modules"`
    Active         types.Bool          `tfsdk:"active"`
    Debug          types.Bool          `tfsdk:"debug"`
    ProductVersion types.String        `tfsdk:"product_version"`
}

type ApplicationModules struct {
    Cache                  *CacheModule                  `tfsdk:"edge_cache"`
    Functions              *FunctionModule               `tfsdk:"functions"`
    ApplicationAccelerator *ApplicationAcceleratorModule `tfsdk:"application_accelerator"`
    ImageProcessor         *ImageProcessorModule         `tfsdk:"image_processor"`
}

type CacheModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type FunctionModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type ApplicationAcceleratorModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type ImageProcessorModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

// Metadata - sets the data source type name
func (e *ApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_main_settings"
}

// Schema - defines the Terraform schema
func (e *ApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datas.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Required:    true, // User must provide the ID to look up
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "data": schema.SingleNestedAttribute{
                Computed: true, // Filled by the Read operation
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The Application identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the Application.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor identifier.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version.",
                        Computed:    true,
                    },
                    "active": schema.BoolAttribute{
                        Computed:    true,
                        Description: "Whether the Application is active.",
                    },
                    "debug": schema.BoolAttribute{
                        Computed:    true,
                        Description: "Whether debug is enabled.",
                    },
                    "modules": schema.SingleNestedAttribute{
                        Computed: true,
                        Attributes: map[string]schema.Attribute{
                            "edge_cache": schema.SingleNestedAttribute{
                                Computed: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Computed: true},
                                },
                            },
                            "functions": schema.SingleNestedAttribute{
                                Computed: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Computed: true},
                                },
                            },
                            "application_accelerator": schema.SingleNestedAttribute{
                                Computed: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Computed: true},
                                },
                            },
                            "image_processor": schema.SingleNestedAttribute{
                                Computed: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Computed: true},
                                },
                            },
                        },
                    },
                },
            },
        },
    }
}

// Configure - receives the API client from the provider
func (e *ApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    e.client = req.ProviderData.(*apiClient)
}

// Read - performs the API call and updates state
func (e *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // 1. Get the ID from config
    var getApplicationId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("id"), &getApplicationId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    if getApplicationId.ValueString() == "" {
        resp.Diagnostics.AddError(
            "Application ID error ",
            "empty application ID",
        )
        return
    }

    // 2. Convert ID to required type
    applicationId, err := strconv.ParseInt(getApplicationId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Application ID error ",
            "not a valid application ID (integer)",
        )
        return
    }

    // 3. Make the API call
    applicationsResponse, response, err := e.client.api.ApplicationsAPI.
        RetrieveApplication(ctx, applicationId).Execute()

    // 4. Handle errors (see Error Handling section)
    if err != nil {
        if response.StatusCode == 429 {
            applicationsResponse, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
                return e.client.api.ApplicationsAPI.RetrieveApplication(ctx, applicationId).Execute()
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

    // 5. Transform response to state model
    mods := applicationsResponse.Data.GetModules()
    cache := mods.GetCache()
    functions := mods.GetFunctions()
    applicationAccelerator := mods.GetApplicationAccelerator()
    imageProcessor := mods.GetImageProcessor()

    modules := &ApplicationModules{
        Cache: &CacheModule{
            Enabled: types.BoolValue(cache.GetEnabled()),
        },
        Functions: &FunctionModule{
            Enabled: types.BoolValue(functions.GetEnabled()),
        },
        ApplicationAccelerator: &ApplicationAcceleratorModule{
            Enabled: types.BoolValue(applicationAccelerator.GetEnabled()),
        },
        ImageProcessor: &ImageProcessorModule{
            Enabled: types.BoolValue(imageProcessor.GetEnabled()),
        },
    }

    state := ApplicationDataSourceModel{
        SchemaVersion: types.Int64Null(),
        Data: &ApplicationData{
            Id:             types.Int64Value(applicationsResponse.Data.GetId()),
            Name:           types.StringValue(applicationsResponse.Data.GetName()),
            Active:         types.BoolValue(applicationsResponse.Data.GetActive()),
            Debug:          types.BoolValue(applicationsResponse.Data.GetDebug()),
            Modules:        modules,
            LastEditor:     types.StringValue(applicationsResponse.Data.GetLastEditor()),
            LastModified:   types.StringValue(applicationsResponse.Data.GetLastModified().Format(time.RFC3339)),
            ProductVersion: types.StringValue(applicationsResponse.Data.GetProductVersion()),
        },
    }

    // 6. Set the state
    state.ID = types.StringValue("Get Application By ID")
    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

### Plural Data Source (List Multiple)

For listing multiple applications with pagination support:

File: `internal/data_source_applications_main_settings.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "time"

    sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &ApplicationsDataSource{}
    _ datasource.DataSourceWithConfigure = &ApplicationsDataSource{}
)

// Constructor function
func dataSourceAzionApplications() datasource.DataSource {
    return &ApplicationsDataSource{}
}

// DataSource struct - holds the client
type ApplicationsDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state for plural data source
type ApplicationsDataSourceModel struct {
    TotalCount types.Int64       `tfsdk:"total_count"`
    Page       types.Int64       `tfsdk:"page"`
    PageSize   types.Int64       `tfsdk:"page_size"`
    Results    []ApplicationData `tfsdk:"results"`
    ID         types.String      `tfsdk:"id"`
}

// ApplicationData struct - represents each item in the results list
// (shared with singular data source, defined above)

// Metadata - sets the data source type name (note plural naming)
func (e *ApplicationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_applications_main_settings"
}

// Schema - defines the Terraform schema for plural data source
func (e *ApplicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true, // Computed, not Required
            },
            "total_count": schema.Int64Attribute{
                Description: "The total number of edge applications.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The page number of edge applications.",
                Optional:    true, // User can specify pagination
            },
            "page_size": schema.Int64Attribute{
                Description: "The Page Size number of edge applications.",
                Optional:    true, // User can specify page size
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "The Application identifier.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "The name of the Application.",
                            Computed:    true,
                        },
                        "last_editor": schema.StringAttribute{
                            Description: "Last editor identifier.",
                            Computed:    true,
                        },
                        "last_modified": schema.StringAttribute{
                            Description: "Last modified timestamp.",
                            Computed:    true,
                        },
                        "product_version": schema.StringAttribute{
                            Description: "Product version.",
                            Computed:    true,
                        },
                        "active": schema.BoolAttribute{
                            Description: "Whether the Application is active.",
                            Computed:    true,
                        },
                        "debug": schema.BoolAttribute{
                            Description: "Whether the Application is in debug mode.",
                            Computed:    true,
                        },
                        "modules": schema.SingleNestedAttribute{
                            Description: "Modules configuration.",
                            Computed:    true,
                            Attributes: map[string]schema.Attribute{
                                "edge_cache": schema.SingleNestedAttribute{
                                    Computed: true,
                                    Attributes: map[string]schema.Attribute{
                                        "enabled": schema.BoolAttribute{Computed: true},
                                    },
                                },
                                "functions": schema.SingleNestedAttribute{
                                    Computed: true,
                                    Attributes: map[string]schema.Attribute{
                                        "enabled": schema.BoolAttribute{Computed: true},
                                    },
                                },
                                "application_accelerator": schema.SingleNestedAttribute{
                                    Computed: true,
                                    Attributes: map[string]schema.Attribute{
                                        "enabled": schema.BoolAttribute{Computed: true},
                                    },
                                },
                                "image_processor": schema.SingleNestedAttribute{
                                    Computed: true,
                                    Attributes: map[string]schema.Attribute{
                                        "enabled": schema.BoolAttribute{Computed: true},
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

// Configure - receives the API client from the provider
func (e *ApplicationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    e.client = req.ProviderData.(*apiClient)
}

// Read - performs the API call and updates state
func (e *ApplicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // 1. Get optional pagination parameters from config
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

    // 2. Set default values for pagination
    if Page.ValueInt64() == 0 {
        Page = types.Int64Value(1)
    }
    if PageSize.ValueInt64() == 0 {
        PageSize = types.Int64Value(10)
    }

    // 3. Make the API call with pagination
    appResponse, response, err := e.client.api.ApplicationsAPI.
        ListApplications(ctx).
        Page(Page.ValueInt64()).
        PageSize(PageSize.ValueInt64()).
        Execute()
    
    // 4. Handle errors (including 429 rate limiting)
    if err != nil {
        if response.StatusCode == 429 {
            appResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedApplicationList, *http.Response, error) {
                return e.client.api.ApplicationsAPI.
                    ListApplications(ctx).
                    Page(Page.ValueInt64()).
                    PageSize(PageSize.ValueInt64()).
                    Execute()
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
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    // 5. Build state from response
    appState := ApplicationsDataSourceModel{
        Page:       Page,
        PageSize:   PageSize,
        TotalCount: types.Int64Value(*appResponse.Count),
    }

    // 6. Iterate over results and transform each item
    for _, resultApplication := range appResponse.GetResults() {
        // Extract nested modules
        mods := resultApplication.GetModules()
        cache := mods.GetCache()
        functions := mods.GetFunctions()
        applicationAccelerator := mods.GetApplicationAccelerator()
        imageProcessor := mods.GetImageProcessor()

        // Build modules structure
        modules := &ApplicationModules{
            Cache: &CacheModule{
                Enabled: types.BoolValue(cache.GetEnabled()),
            },
            Functions: &FunctionModule{
                Enabled: types.BoolValue(functions.GetEnabled()),
            },
            ApplicationAccelerator: &ApplicationAcceleratorModule{
                Enabled: types.BoolValue(applicationAccelerator.GetEnabled()),
            },
            ImageProcessor: &ImageProcessorModule{
                Enabled: types.BoolValue(imageProcessor.GetEnabled()),
            },
        }

        // Append to results slice
        appState.Results = append(appState.Results, ApplicationData{
            Id:             types.Int64Value(resultApplication.GetId()),
            Name:           types.StringValue(resultApplication.GetName()),
            LastEditor:     types.StringValue(resultApplication.GetLastEditor()),
            LastModified:   types.StringValue(resultApplication.GetLastModified().Format(time.RFC3339)),
            Modules:        modules,
            ProductVersion: types.StringValue(resultApplication.GetProductVersion()),
            Active:         types.BoolValue(resultApplication.GetActive()),
            Debug:          types.BoolValue(resultApplication.GetDebug()),
        })
    }

    // 7. Set a descriptive ID
    appState.ID = types.StringValue("Get All Application")

    // 8. Set the state
    diags := resp.State.Set(ctx, &appState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular Data Source | Plural Data Source |
|--------|---------------------|-------------------|
| **File Name** | `data_source_application_main_settings.go` | `data_source_applications_main_settings.go` |
| **Type Name** | `azion_application_main_settings` | `azion_applications_main_settings` |
| **ID Field** | `Required` (user provides ID to look up) | `Computed` (set after reading) |
| **Results** | `SingleNestedAttribute` (single object) | `ListNestedAttribute` (array of objects) |
| **Pagination** | No pagination fields | Has `page`, `page_size` (Optional) |
| **Count Field** | Not applicable | `total_count` (Computed) |
| **API Method** | `RetrieveApplication(ctx, id)` | `ListApplications(ctx).Page().PageSize()` |
| **Response Type** | `*sdk.ApplicationResponse` | `*sdk.PaginatedApplicationList` |
| **State ID Value** | `"Get Application By ID"` | `"Get All Application"` |

---

## Resource Implementation

File: `internal/resource_application_main_setting.go`

```go
package provider

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "sync"
    "time"

    sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ resource.Resource                = &applicationResource{}
    _ resource.ResourceWithConfigure   = &applicationResource{}
    _ resource.ResourceWithImportState = &applicationResource{}
)

// Constructor function
func NewApplicationMainSettingsResource() resource.Resource {
    return &applicationResource{}
}

// Resource struct - holds the client
type applicationResource struct {
    client *apiClient
}

// Resource Model - represents Terraform state
type ApplicationResourceModel struct {
    Application *ApplicationResults `tfsdk:"application"`
    ID          types.String         `tfsdk:"id"`
    LastUpdated types.String         `tfsdk:"last_updated"`
}

type ApplicationResults struct {
    ApplicationID  types.Int64         `tfsdk:"application_id"`
    Name           types.String        `tfsdk:"name"`
    Modules        *ApplicationModules `tfsdk:"modules"`
    Active         types.Bool          `tfsdk:"active"`
    Debug          types.Bool          `tfsdk:"debug"`
    ProductVersion types.String        `tfsdk:"product_version"`
}

// Metadata - sets the resource type name
func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_main_setting"
}

// Configure - receives the API client from the provider
func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    r.client = req.ProviderData.(*apiClient)
}
```

---

## Resource Schema Definition

```go
func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
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
            "application": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "application_id": schema.Int64Attribute{
                        Description: "The Application identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "The name of the Application.",
                        Required:    true,
                    },
                    "active": schema.BoolAttribute{
                        Optional:    true,
                        Computed:    true,
                        Default:     booldefault.StaticBool(true),
                        Description: "Indicates whether the Application is active.",
                    },
                    "debug": schema.BoolAttribute{
                        Optional:    true,
                        Computed:    true,
                        Default:     booldefault.StaticBool(false),
                        Description: "Indicates whether debug rules are enabled for the Application.",
                    },
                    "product_version": schema.StringAttribute{
                        Computed:    true,
                        Description: "The product version.",
                    },
                    "modules": schema.SingleNestedAttribute{
                        Optional: true,
                        Attributes: map[string]schema.Attribute{
                            "edge_cache": schema.SingleNestedAttribute{
                                Optional: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Optional: true},
                                },
                            },
                            "functions": schema.SingleNestedAttribute{
                                Optional: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Optional: true},
                                },
                            },
                            "application_accelerator": schema.SingleNestedAttribute{
                                Optional: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Optional: true},
                                },
                            },
                            "image_processor": schema.SingleNestedAttribute{
                                Optional: true,
                                Attributes: map[string]schema.Attribute{
                                    "enabled": schema.BoolAttribute{Optional: true},
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

---

## Create Method

```go
var mutex sync.Mutex

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan ApplicationResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)

    // Use mutex for thread safety
    mutex.Lock()
    defer mutex.Unlock()

    if resp.Diagnostics.HasError() {
        return
    }

    // Build the SDK request object using ValueBoolPointer for optional fields
    application := sdk.ApplicationRequest{
        Name:   plan.Application.Name.ValueString(),
        Active: plan.Application.Active.ValueBoolPointer(),
        Debug:  plan.Application.Debug.ValueBoolPointer(),
    }

    // Transform modules into request format
    modsPlan := plan.Application.Modules
    modsRequest := transformModuleIntoRequest(modsPlan)
    application.Modules = &modsRequest

    // Make the API call using r.client.api (V4 SDK)
    createApplication, response, err := r.client.api.
        ApplicationsAPI.CreateApplication(ctx).
        ApplicationRequest(application).Execute()

    // Handle errors with 429 retry logic
    if err != nil {
        if response != nil && response.StatusCode == 429 {
            createApplication, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
                return r.client.api.
                    ApplicationsAPI.CreateApplication(ctx).
                    ApplicationRequest(application).Execute()
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

    // Build the state from response using GetData() methods
    appResults := &ApplicationResults{
        ApplicationID:  types.Int64Value(createApplication.Data.GetId()),
        Name:           types.StringValue(createApplication.Data.GetName()),
        Active:         types.BoolValue(createApplication.Data.GetActive()),
        Debug:          types.BoolValue(createApplication.Data.GetDebug()),
        ProductVersion: types.StringValue(createApplication.Data.GetProductVersion()),
        Modules:        plan.Application.Modules,
    }

    // Only update modules from API response if the plan had modules specified
    // This prevents Terraform from seeing an inconsistency when modules was null in plan
    if plan.Application.Modules != nil && createApplication.Data.Modules != nil {
        modulesResp := createApplication.Data.GetModules()
        modules := ApplicationModules{}

        // Only populate modules that were specified in the plan
        if plan.Application.Modules.Cache != nil && modulesResp.Cache != nil {
            modules.Cache = &CacheModule{
                Enabled: types.BoolValue(modulesResp.Cache.GetEnabled()),
            }
        }
        if plan.Application.Modules.Functions != nil && modulesResp.Functions != nil {
            modules.Functions = &FunctionModule{
                Enabled: types.BoolValue(modulesResp.Functions.GetEnabled()),
            }
        }
        if plan.Application.Modules.ApplicationAccelerator != nil && modulesResp.ApplicationAccelerator != nil {
            modules.ApplicationAccelerator = &ApplicationAcceleratorModule{
                Enabled: types.BoolValue(modulesResp.ApplicationAccelerator.GetEnabled()),
            }
        }
        if plan.Application.Modules.ImageProcessor != nil && modulesResp.ImageProcessor != nil {
            modules.ImageProcessor = &ImageProcessorModule{
                Enabled: types.BoolValue(modulesResp.ImageProcessor.GetEnabled()),
            }
        }
        appResults.Modules = &modules
    }

    plan.Application = appResults
    plan.ID = types.StringValue(fmt.Sprintf("%d", createApplication.Data.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

## Read Method

```go
func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state ApplicationResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse ID from state
    idInt64, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)

    // Call retrieve API using r.client.api (V4 SDK)
    stateApplication, response, err := r.client.api.
        ApplicationsAPI.
        RetrieveApplication(ctx, idInt64).Execute()

    // Handle 404 - resource was deleted outside Terraform
    if err != nil {
        if response != nil && response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        if response != nil && response.StatusCode == 429 {
            stateApplication, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
                return r.client.api.ApplicationsAPI.RetrieveApplication(ctx, idInt64).Execute()
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

    // Update state from response
    state.Application = &ApplicationResults{
        ApplicationID:  types.Int64Value(stateApplication.Data.GetId()),
        Name:           types.StringValue(stateApplication.Data.GetName()),
        Active:         types.BoolValue(stateApplication.Data.GetActive()),
        Debug:          types.BoolValue(stateApplication.Data.GetDebug()),
        ProductVersion: types.StringValue(stateApplication.Data.GetProductVersion()),
    }
    state.ID = types.StringValue(fmt.Sprintf("%d", stateApplication.Data.GetId()))

    // Handle modules from response
    modelPlan := ApplicationModules{}
    if stateApplication.Data.Modules != nil {
        modelState := stateApplication.Data.GetModules()
        if modelState.Cache != nil {
            modelPlan.Cache = &CacheModule{
                Enabled: types.BoolValue(modelState.Cache.GetEnabled()),
            }
        }
        if modelState.Functions != nil {
            modelPlan.Functions = &FunctionModule{
                Enabled: types.BoolValue(modelState.Functions.GetEnabled()),
            }
        }
        if modelState.ApplicationAccelerator != nil {
            modelPlan.ApplicationAccelerator = &ApplicationAcceleratorModule{
                Enabled: types.BoolValue(modelState.ApplicationAccelerator.GetEnabled()),
            }
        }
        if modelState.ImageProcessor != nil {
            modelPlan.ImageProcessor = &ImageProcessorModule{
                Enabled: types.BoolValue(modelState.ImageProcessor.GetEnabled()),
            }
        }
    }
    state.Application.Modules = &modelPlan

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

## Update Method (PUT)

Applications Main Settings uses PUT for full updates:

```go
func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan ApplicationResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build full request object using ValueBoolPointer for optional fields
    application := sdk.ApplicationRequest{
        Name:   plan.Application.Name.ValueString(),
        Debug:  plan.Application.Debug.ValueBoolPointer(),
        Active: plan.Application.Active.ValueBoolPointer(),
    }

    // Transform modules into request format
    modsPlan := plan.Application.Modules
    modsRequest := transformModuleIntoRequest(modsPlan)
    application.Modules = &modsRequest

    // Parse ID from plan
    idInt64, _ := strconv.ParseInt(plan.ID.ValueString(), 10, 64)

    // PUT request using r.client.api (V4 SDK)
    updateApplication, response, err := r.client.api.
        ApplicationsAPI.
        UpdateApplication(ctx, idInt64).
        ApplicationRequest(application).Execute()

    // Handle errors with 429 retry logic
    if err != nil {
        if response != nil && response.StatusCode == 429 {
            updateApplication, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
                return r.client.api.
                    ApplicationsAPI.
                    UpdateApplication(ctx, idInt64).
                    ApplicationRequest(application).Execute()
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

    // Update state from response
    plan.Application = &ApplicationResults{
        ApplicationID:  types.Int64Value(updateApplication.Data.GetId()),
        Name:           types.StringValue(updateApplication.Data.GetName()),
        Active:         types.BoolValue(updateApplication.Data.GetActive()),
        Debug:          types.BoolValue(updateApplication.Data.GetDebug()),
        ProductVersion: types.StringValue(updateApplication.Data.GetProductVersion()),
        Modules:        modsPlan,
    }

    plan.ID = types.StringValue(fmt.Sprintf("%d", updateApplication.Data.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

## Delete Method

```go
func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state ApplicationResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse ID from state
    idInt64, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)

    // Call delete API using r.client.api (V4 SDK)
    _, response, err := r.client.api.ApplicationsAPI.
        DeleteApplication(ctx, idInt64).Execute()

    // Handle errors with 429 retry logic
    if err != nil {
        if response != nil && response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*sdk.DeleteResponse, *http.Response, error) {
                return r.client.api.ApplicationsAPI.DeleteApplication(ctx, idInt64).Execute()
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

    // No need to set state - resource is deleted
}
```

---

## ImportState Method

```go
func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

For resources with parent-child relationships, import may need special handling:

```go
func (r *applicationOriginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Parse composite ID: "applicationID,originKey"
    idParts := strings.Split(req.ID, ",")
    if len(idParts) != 2 {
        resp.Diagnostics.AddError("Invalid import ID", "Expected format: applicationID,originKey")
        return
    }
    
    appID, err := strconv.ParseInt(idParts[0], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid application ID", "Could not parse application ID")
        return
    }
    
    resp.Diagnostics.Append(resp.State.Set(ctx, &OriginResourceModel{
        ApplicationID: types.Int64Value(appID),
        ID:            types.StringValue(req.ID),
        Results: &OriginResults{
            OriginKey: types.StringValue(idParts[1]),
        },
    })...)
}
```

---

## Transform Functions

### transformModuleIntoRequest

This function transforms the Terraform plan modules into the SDK request format:

```go
func transformModuleIntoRequest(modsPlan *ApplicationModules) sdk.ApplicationModulesRequest {
    modsRequest := sdk.ApplicationModulesRequest{}
    if modsPlan != nil {
        cachePlan := modsPlan.Cache
        if cachePlan != nil && !cachePlan.Enabled.IsNull() {
            enabled := cachePlan.Enabled
            cacheReq := sdk.CacheModuleRequest{
                Enabled: enabled.ValueBoolPointer(),
            }
            modsRequest.SetCache(cacheReq)
        }

        functionsPlan := modsPlan.Functions
        if functionsPlan != nil && !functionsPlan.Enabled.IsNull() {
            enabled := functionsPlan.Enabled
            functionsReq := sdk.FunctionModuleRequest{
                Enabled: enabled.ValueBoolPointer(),
            }
            modsRequest.SetFunctions(functionsReq)
        }

        applicationAcceleratorPlan := modsPlan.ApplicationAccelerator
        if applicationAcceleratorPlan != nil && !applicationAcceleratorPlan.Enabled.IsNull() {
            enabled := applicationAcceleratorPlan.Enabled
            appAccReq := sdk.ApplicationAcceleratorModuleRequest{
                Enabled: enabled.ValueBoolPointer(),
            }
            modsRequest.SetApplicationAccelerator(appAccReq)
        }

        imageProcessorPlan := modsPlan.ImageProcessor
        if imageProcessorPlan != nil && !imageProcessorPlan.Enabled.IsNull() {
            enabled := imageProcessorPlan.Enabled
            imgProcReq := sdk.ImageProcessorModuleRequest{
                Enabled: enabled.ValueBoolPointer(),
            }
            modsRequest.SetImageProcessor(imgProcReq)
        }
    }

    return modsRequest
}
```

---

## Common Issues

### Application Schema Fields

**IMPORTANT:** The OpenAPI `Application` schema only contains these fields:
- `id`, `name`, `last_editor`, `last_modified`, `product_version` (required)
- `active`, `debug` (optional boolean)
- `modules` (nested object)

Do NOT include these fields in the Application schema (they exist in other API endpoints):
- `http_port` (list of int64) - **DOES NOT EXIST in Application API**
- `https_port` (list of int64) - **DOES NOT EXIST in Application API**
- `delivery_protocol` (string) - **DOES NOT EXIST in Application API**
- `minimum_tls_version` (string) - **DOES NOT EXIST in Application API**
- `supported_ciphers` (string) - **DOES NOT EXIST in Application API**
- `debug_rules` (boolean) - **DOES NOT EXIST in Application API**

These fields exist in other parts of the API (e.g., `HttpProtocol` schema has `http_ports` and `https_ports`), but they are NOT part of the `Application` model.

### Prevention Guidelines

1. **Always verify against the OpenAPI schema definition** - Check the exact schema name referenced by the API response (e.g., `Application` not `ApplicationRequest` or other similarly named schemas)

2. **Cross-reference API response handling** - Every field in the model struct must be populated from the API response in the Read method

3. **For lists and nested objects** - Always initialize them even if empty:
   ```go
   // For empty lists
   state.Results = []ApplicationData{}
   
   // For populated lists
   for _, item := range response.GetResults() {
       state.Results = append(state.Results, ApplicationData{...})
   }
   ```

4. **Remove unused imports** - If you remove fields that required special imports, clean up the imports

### Parent-Child Resource Pattern

For resources that belong to a parent (e.g., origin belongs to application):

```go
type OriginDataSourceModel struct {
    SchemaVersion types.Int64   `tfsdk:"schema_version"`
    ID            types.String  `tfsdk:"id"`
    ApplicationID types.Int64   `tfsdk:"application_id"`  // Parent ID
    Results       OriginResults `tfsdk:"origin"`
}

func (o *OriginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // Get both parent ID and resource key
    var applicationID types.Int64
    var getOriginsKey types.String
    
    diags := req.Config.GetAttribute(ctx, path.Root("origin").AtName("origin_key"), &getOriginsKey)
    resp.Diagnostics.Append(diags...)
    
    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    
    // API call requires both IDs
    originResponse, response, err := o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.
        EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(
            ctx, 
            applicationID.ValueInt64(), 
            getOriginsKey.ValueString(),
        ).Execute()
}
```

---

## Provider Registration

Register in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionApplication,
        dataSourceAzionApplications,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewApplicationMainSettingsResource,
        // ... other resources
    }
}
```

---

## Summary Checklist

When implementing or updating Applications resources and data sources:

1. **Use the correct SDK**: V4 (`azion-api` package)
2. **Use correct client access**: `r.client.api` (not `r.client.edgeApi`)
3. **Determine ID types**: `int64` for V4 SDK
4. **Determine update method**: PUT (full update) for Applications Main Settings
5. **Create model structs**: With appropriate `tfsdk` tags
6. **Implement schema**: Include default values for `active` and `debug`
7. **Implement all methods**: Create, Read, Update, Delete, ImportState (for resources)
8. **Handle 429 errors**: Use `utils.RetryOn429`
9. **Handle optional fields**: Use `ValueBoolPointer()` for boolean pointers
10. **Transform nested objects**: Use helper functions for modules
11. **Register in provider.go**: Add to DataSources() or Resources()
12. **Generate documentation**: Create docs and examples
13. **Update example/test files**: After any schema changes, update the corresponding files
14. **Run linters**: After any change, run `golangci-lint run --config .golintci.yml ./internal/...`
