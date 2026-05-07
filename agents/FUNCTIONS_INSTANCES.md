# Application Function Instances - Code Generation Guide

This document provides specific guidance for implementing Application Function Instance resources and data sources in the Terraform provider. It serves as a comprehensive reference for generating and maintaining these files based on the latest Azion API SDK.

## Table of Contents

1. [Overview](#overview)
2. [SDK Selection](#sdk-selection)
3. [Naming Convention](#naming-convention)
4. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
5. [Resource Implementation](#resource-implementation)
   - [Resource Model Structs](#resource-model-structs)
   - [Create Method](#create-method)
   - [Read Method](#read-method)
   - [Update Method](#update-method)
   - [Delete Method](#delete-method)
   - [ImportState Method](#importstate-method)
6. [Schema Definition Patterns](#schema-definition-patterns)
7. [Error Handling](#error-handling)
8. [Documentation and Examples](#documentation-and-examples)
9. [Common Issues](#common-issues)
10. [Files Reference](#files-reference)

---

## Overview

Application Function Instances allow you to associate Edge Functions with Applications, enabling serverless compute at the edge. This resource manages the instantiation of functions within an application context.

### Key Concepts

- **Application ID**: The parent application that will use the function instance
- **Function ID**: Reference to the Edge Function to be instantiated
- **Args**: JSON-formatted arguments passed to the function
- **Active**: Whether the function instance is enabled

---

## SDK Selection

Application Function Instance uses the **V4 SDK (`azion-api`)**:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Application Function Instance (Singular Data Source) | `azion-api` (v4) | `api.ApplicationsFunctionAPI` | `https://api.azion.com/v4` |
| Application Function Instance (Plural Data Source) | `azion-api` (v4) | `api.ApplicationsFunctionAPI` | `https://api.azion.com/v4` |
| Application Function Instance (Resource) | `azion-api` (v4) | `api.ApplicationsFunctionAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` for most operations |
| Create Method | `.CreateApplicationFunctionInstance(ctx, applicationId).FunctionInstanceRequest(req).Execute()` |
| Retrieve Method | `.RetrieveApplicationFunctionInstance(ctx, applicationId, functionId).Execute()` |
| Update Method | `.PartialUpdateApplicationFunctionInstance(ctx, applicationId, functionId).PatchedFunctionInstanceRequest(req).Execute()` |
| Delete Method | `.DeleteApplicationFunctionInstance(ctx, applicationId, functionId).Execute()` |
| Response Type | `FunctionInstanceResponse` for single, `PaginatedFunctionInstanceList` for list |
| List Method | `.ListApplicationFunctionInstances(ctx, applicationId).Page(page).PageSize(pageSize).Execute()` |

### SDK Models

The SDK provides these model types:

```go
// Request models
azionapi.FunctionInstanceRequest       // For create operations
azionapi.PatchedFunctionInstanceRequest // For partial update operations

// Response models
azionapi.FunctionInstanceResponse       // Single instance response
azionapi.PaginatedFunctionInstanceList  // List response
azionapi.FunctionInstance               // Instance data structure
azionapi.DeleteResponse                 // Delete operation response
```

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

> **Important:** Do NOT use the legacy `edge-api` import path. The correct import is `azion-api`.

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - for Application Function Instance API
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
}
```

---

## Naming Convention

### No "Edge" Prefix

The "edge" prefix has been **completely removed** from all Go code naming - structs, variables, function names, and Terraform-facing type names. This aligns with the V4 SDK naming convention.

### Go Struct Names

Since both singular and plural data sources exist in the same package, unique struct names are required:

**Singular Data Source (single instance):**
- `ApplicationFunctionInstanceDataSource` - the datasource struct
- `FunctionInstanceDataSourceModel` - the state model

**Plural Data Source (list of instances):**
- `ApplicationFunctionInstancesDataSource` - the datasource struct
- `FunctionInstancesDataSourceModel` - the state model
- `FunctionInstanceResponse` - the results struct for each item

**Resource:**
- `functionInstanceResource` - the resource struct (private)
- `FunctionInstanceResourceModel` - the state model
- `FunctionInstanceResourceResults` - the nested data struct

### Terraform Type Names

| Resource Type | Terraform Name |
|---------------|----------------|
| Singular Data Source | `azion_application_function_instance` |
| Plural Data Source | `azion_application_function_instances` |
| Resource | `azion_application_function_instance` |

### Go Function Names

| Purpose | Function Name |
|---------|---------------|
| Singular Data Source Factory | `dataSourceAzionApplicationFunctionInstance()` |
| Plural Data Source Factory | `dataSourceAzionApplicationFunctionInstances()` |
| Resource Factory | `NewApplicationFunctionInstanceResource()` |

### File Names

The file names retain descriptive identifiers for clarity:

| Resource Type | File Name |
|---------------|-----------|
| Singular Data Source | `internal/data_source_application_function_instance.go` |
| Plural Data Source | `internal/data_source_application_functions_instance.go` |
| Resource | `internal/resource_application_functions_instance.go` |

> **Note:** File names retain "edge" for historical consistency with existing codebase structure, but the content uses the new naming convention without "edge" prefix.

---

## Data Source Implementation

### Singular Data Source (Read by ID)

File: `internal/data_source_application_function_instance.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "time"

    "github.com/hashicorp/terraform-plugin-framework/path"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &ApplicationFunctionInstanceDataSource{}
    _ datasource.DataSourceWithConfigure = &ApplicationFunctionInstanceDataSource{}
)

func dataSourceAzionApplicationFunctionInstance() datasource.DataSource {
    return &ApplicationFunctionInstanceDataSource{}
}

type ApplicationFunctionInstanceDataSource struct {
    client *apiClient
}

type FunctionInstanceDataSourceModel struct {
    ID            types.Int64  `tfsdk:"id"`
    ApplicationID types.Int64  `tfsdk:"application_id"`
    FunctionID    types.Int64  `tfsdk:"function_id"`
    FunctionName  types.String `tfsdk:"name"`
    Args          types.String `tfsdk:"args"`
    Active        types.Bool   `tfsdk:"active"`
    LastEditor    types.String `tfsdk:"last_editor"`
    LastModified  types.String `tfsdk:"last_modified"`
    CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *ApplicationFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_function_instance"
}

func (d *ApplicationFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Description: "Numeric identifier of the function instance.",
                Required:    true,
            },
            "application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Application.",
                Required:    true,
            },
            "function_id": schema.Int64Attribute{
                Description: "The function identifier.",
                Computed:    true,
            },
            "name": schema.StringAttribute{
                Description: "Name of the function instance.",
                Computed:    true,
            },
            "args": schema.StringAttribute{
                Description: "Arguments of the function instance in JSON format.",
                Computed:    true,
            },
            "active": schema.BoolAttribute{
                Description: "Active status of the function instance.",
                Computed:    true,
            },
            "last_editor": schema.StringAttribute{
                Description: "Last editor of the function instance.",
                Computed:    true,
            },
            "last_modified": schema.StringAttribute{
                Description: "Last modified timestamp of the function instance.",
                Computed:    true,
            },
            "created_at": schema.StringAttribute{
                Description: "The creation timestamp of the function instance.",
                Computed:    true,
            },
        },
    }
}

func (d *ApplicationFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *ApplicationFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    var functionInstanceID types.Int64
    diagsFunctionInstanceID := req.Config.GetAttribute(ctx, path.Root("id"), &functionInstanceID)
    resp.Diagnostics.Append(diagsFunctionInstanceID...)
    if resp.Diagnostics.HasError() {
        return
    }

    functionInstanceResponse, response, err := d.client.api.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, applicationID.ValueInt64(), functionInstanceID.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            functionInstanceResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
                return d.client.api.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, applicationID.ValueInt64(), functionInstanceID.ValueInt64()).Execute()
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

    jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "error converting args to string")
    }

    state := FunctionInstanceDataSourceModel{
        ID:            types.Int64Value(functionInstanceResponse.Data.GetId()),
        ApplicationID: applicationID,
        FunctionID:    types.Int64Value(functionInstanceResponse.Data.GetFunction()),
        FunctionName:  types.StringValue(functionInstanceResponse.Data.GetName()),
        Args:          types.StringValue(jsonArgsStr),
        Active:        types.BoolValue(functionInstanceResponse.Data.GetActive()),
        LastEditor:    types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
        LastModified:  types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC3339)),
        CreatedAt:     types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Plural Data Source (List Multiple Resources)

File: `internal/data_source_application_functions_instance.go`

The plural data source lists all function instances for an application.

Key differences from singular:
1. Uses `ListApplicationFunctionInstances` API method
2. Returns `results` as a list attribute
3. Has `total_count` attribute
4. Response type is `PaginatedFunctionInstanceList`
5. Supports pagination with `page` and `page_size` parameters

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "time"

    "github.com/hashicorp/terraform-plugin-framework/path"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &ApplicationFunctionInstancesDataSource{}
    _ datasource.DataSourceWithConfigure = &ApplicationFunctionInstancesDataSource{}
)

func dataSourceAzionApplicationFunctionInstances() datasource.DataSource {
    return &ApplicationFunctionInstancesDataSource{}
}

type ApplicationFunctionInstancesDataSource struct {
    client *apiClient
}

type FunctionInstancesDataSourceModel struct {
    ID            types.Int64                `tfsdk:"id"`
    ApplicationID types.Int64                `tfsdk:"application_id"`
    Page          types.Int64                `tfsdk:"page"`
    PageSize      types.Int64                `tfsdk:"page_size"`
    TotalCount    types.Int64                `tfsdk:"total_count"`
    Results       []FunctionInstanceResponse `tfsdk:"results"`
}

type FunctionInstanceResponse struct {
    ID           types.Int64  `tfsdk:"id"`
    FunctionID   types.Int64  `tfsdk:"function_id"`
    Name         types.String `tfsdk:"name"`
    Args         types.String `tfsdk:"args"`
    Active       types.Bool   `tfsdk:"active"`
    LastEditor   types.String `tfsdk:"last_editor"`
    LastModified types.String `tfsdk:"last_modified"`
    CreatedAt    types.String `tfsdk:"created_at"`
}

func (d *ApplicationFunctionInstancesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_function_instances"
}

func (d *ApplicationFunctionInstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "application_id": schema.Int64Attribute{
                Description: "Numeric identifier of the Application.",
                Required:    true,
            },
            "page": schema.Int64Attribute{
                Description: "Page number for pagination.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "Number of items per page.",
                Optional:    true,
            },
            "total_count": schema.Int64Attribute{
                Description: "The total number of function instances.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "The function instance identifier.",
                            Computed:    true,
                        },
                        "function_id": schema.Int64Attribute{
                            Description: "The function identifier.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "Name of the function instance.",
                            Computed:    true,
                        },
                        "args": schema.StringAttribute{
                            Description: "Arguments of the function instance.",
                            Computed:    true,
                        },
                        "active": schema.BoolAttribute{
                            Description: "Active status of the function instance.",
                            Computed:    true,
                        },
                        "last_editor": schema.StringAttribute{
                            Description: "Last editor of the function instance.",
                            Computed:    true,
                        },
                        "last_modified": schema.StringAttribute{
                            Description: "Last modified timestamp of the function instance.",
                            Computed:    true,
                        },
                        "created_at": schema.StringAttribute{
                            Description: "The creation timestamp of the function instance.",
                            Computed:    true,
                        },
                    },
                },
            },
        },
    }
}

func (d *ApplicationFunctionInstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *ApplicationFunctionInstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var page types.Int64
    var pageSize types.Int64
    var applicationID types.Int64

    diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
    resp.Diagnostics.Append(diagsPage...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
    resp.Diagnostics.Append(diagsPageSize...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diagsApplicationID...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Default pagination values
    if page.ValueInt64() == 0 {
        page = types.Int64Value(1)
    }
    if pageSize.ValueInt64() == 0 {
        pageSize = types.Int64Value(10)
    }

    functionInstancesResponse, response, err := d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            functionInstancesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedFunctionInstanceList, *http.Response, error) {
                return d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
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

    state := FunctionInstancesDataSourceModel{
        ApplicationID: applicationID,
        Page:          page,
        PageSize:      pageSize,
        TotalCount:    types.Int64Value(functionInstancesResponse.GetCount()),
    }

    for _, result := range functionInstancesResponse.GetResults() {
        jsonArgsStr, err := utils.ConvertInterfaceToString(result.GetArgs())
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "error converting args to string")
        }
        state.Results = append(state.Results, FunctionInstanceResponse{
            ID:           types.Int64Value(result.GetId()),
            FunctionID:   types.Int64Value(result.GetFunction()),
            Name:         types.StringValue(result.GetName()),
            Args:         types.StringValue(jsonArgsStr),
            Active:       types.BoolValue(result.GetActive()),
            LastEditor:   types.StringValue(result.GetLastEditor()),
            LastModified: types.StringValue(result.GetLastModified().Format(time.RFC3339)),
            CreatedAt:    types.StringValue(result.GetCreatedAt().Format(time.RFC3339)),
        })
    }

    state.ID = types.Int64Value(0)
    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular | Plural |
|--------|----------|--------|
| API Method | `RetrieveApplicationFunctionInstance` | `ListApplicationFunctionInstances` |
| Response Type | `FunctionInstanceResponse` | `PaginatedFunctionInstanceList` |
| ID Attribute | Required (specific instance) | Computed (data source ID) |
| Results | Direct attributes on model | `results` list attribute |
| Total Count | Not available | `total_count` attribute |
| Pagination | Not applicable | `page`, `page_size` parameters |

---

## Resource Implementation

File: `internal/resource_application_functions_instance.go`

### Resource Model Structs

```go
type functionInstanceResource struct {
    client *apiClient
}

type FunctionInstanceResourceModel struct {
    Function      *FunctionInstanceResourceResults `tfsdk:"data"`
    ID            types.Int64                      `tfsdk:"id"`
    ApplicationID types.Int64                      `tfsdk:"application_id"`
    LastUpdated   types.String                     `tfsdk:"last_updated"`
}

type FunctionInstanceResourceResults struct {
    FunctionID types.Int64  `tfsdk:"function_id"`
    Name       types.String `tfsdk:"name"`
    Args       types.String `tfsdk:"args"`
    ID         types.Int64  `tfsdk:"id"`
    Active     types.Bool   `tfsdk:"active"`
}
```

### Create Method

The Create method builds a `FunctionInstanceRequest` and calls the API:

```go
func (r *functionInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan FunctionInstanceResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse JSON args
    var argsStr string
    if plan.Function.Args.IsUnknown() {
        argsStr = "{}"
    } else {
        if plan.Function.Args.ValueString() == "" || plan.Function.Args.IsNull() {
            resp.Diagnostics.AddError("Args", "Is not null")
            return
        }
        argsStr = plan.Function.Args.ValueString()
    }

    planJsonArgs, err := utils.UnmarshallJsonArgs(argsStr)
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "err")
        return
    }

    // Build request
    functionInstanceRequest := azionapi.FunctionInstanceRequest{
        Name:     plan.Function.Name.ValueString(),
        Function: plan.Function.FunctionID.ValueInt64(),
        Args:     planJsonArgs,
        Active:   plan.Function.Active.ValueBoolPointer(),
    }

    // Call API
    functionInstanceResponse, response, err := r.client.api.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64()).FunctionInstanceRequest(functionInstanceRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            functionInstanceResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
                return r.client.api.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64()).FunctionInstanceRequest(functionInstanceRequest).Execute()
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

    // Update state with response
    jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "err")
        return
    }

    plan.Function = &FunctionInstanceResourceResults{
        FunctionID: types.Int64Value(functionInstanceResponse.Data.GetFunction()),
        Name:       types.StringValue(functionInstanceResponse.Data.GetName()),
        Args:       types.StringValue(jsonArgsStr),
        ID:         types.Int64Value(functionInstanceResponse.Data.GetId()),
        Active:     types.BoolValue(functionInstanceResponse.Data.GetActive()),
    }
    plan.ID = types.Int64Value(functionInstanceResponse.Data.GetId())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Method

The Read method handles both normal reads and import scenarios:

```go
func (r *functionInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state FunctionInstanceResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var applicationID int64
    var functionInstanceID int64

    // Handle import format "applicationID/instanceID"
    idStr := strconv.FormatInt(state.ID.ValueInt64(), 10)
    valueFromCmd := strings.Split(idStr, "/")
    if len(valueFromCmd) > 1 {
        appID, err := strconv.ParseInt(valueFromCmd[0], 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Invalid application ID format", err.Error())
            return
        }
        instanceID, err := strconv.ParseInt(valueFromCmd[1], 10, 64)
        if err != nil {
            resp.Diagnostics.AddError("Invalid instance ID format", err.Error())
            return
        }
        applicationID = appID
        functionInstanceID = instanceID
    } else {
        applicationID = state.ApplicationID.ValueInt64()
        functionInstanceID = state.ID.ValueInt64()
    }

    if functionInstanceID == 0 {
        resp.Diagnostics.AddError("Function Instance id error", "is not null")
        return
    }

    // Call API
    functionInstanceResponse, response, err := r.client.api.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, applicationID, functionInstanceID).Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        if response.StatusCode == 429 {
            functionInstanceResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
                return r.client.api.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, applicationID, functionInstanceID).Execute()
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

    jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "err")
        return
    }

    functionInstanceState := FunctionInstanceResourceModel{
        ApplicationID: types.Int64Value(applicationID),
        ID:            types.Int64Value(functionInstanceResponse.Data.GetId()),
        Function: &FunctionInstanceResourceResults{
            ID:         types.Int64Value(functionInstanceResponse.Data.GetId()),
            FunctionID: types.Int64Value(functionInstanceResponse.Data.GetFunction()),
            Name:       types.StringValue(functionInstanceResponse.Data.GetName()),
            Args:       types.StringValue(jsonArgsStr),
            Active:     types.BoolValue(functionInstanceResponse.Data.GetActive()),
        },
    }

    diags = resp.State.Set(ctx, &functionInstanceState)
    resp.Diagnostics.Append(diags...)
}
```

### Update Method

Uses PATCH for partial update with `PatchedFunctionInstanceRequest`:

```go
func (r *functionInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan FunctionInstanceResourceModel
    var functionInstanceID types.Int64
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state FunctionInstanceResourceModel
    diagsState := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsState...)
    if resp.Diagnostics.HasError() {
        return
    }

    if plan.Function.ID.IsNull() || plan.Function.ID.ValueInt64() == 0 {
        functionInstanceID = state.Function.ID
    } else {
        functionInstanceID = plan.Function.ID
    }

    // Parse JSON args
    var argsStr string
    if plan.Function.Args.IsUnknown() {
        argsStr = "{}"
    } else {
        if plan.Function.Args.ValueString() == "" || plan.Function.Args.IsNull() {
            resp.Diagnostics.AddError("Args", "Is not null")
            return
        }
        argsStr = plan.Function.Args.ValueString()
    }

    requestJsonArgsStr, err := utils.UnmarshallJsonArgs(argsStr)
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "error while unmarshalling json args")
        return
    }

    // Build patch request
    patchRequest := azionapi.PatchedFunctionInstanceRequest{
        Name:     plan.Function.Name.ValueStringPointer(),
        Function: plan.Function.FunctionID.ValueInt64Pointer(),
        Args:     requestJsonArgsStr,
        Active:   plan.Function.Active.ValueBoolPointer(),
    }

    // Call API
    functionInstanceUpdateResponse, response, err := r.client.api.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64(), functionInstanceID.ValueInt64()).PatchedFunctionInstanceRequest(patchRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            functionInstanceUpdateResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
                return r.client.api.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64(), functionInstanceID.ValueInt64()).PatchedFunctionInstanceRequest(patchRequest).Execute()
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

    jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceUpdateResponse.Data.GetArgs())
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "error while reading json args from response")
        return
    }

    plan.Function = &FunctionInstanceResourceResults{
        FunctionID: types.Int64Value(functionInstanceUpdateResponse.Data.GetFunction()),
        Name:       types.StringValue(functionInstanceUpdateResponse.Data.GetName()),
        Args:       types.StringValue(jsonArgsStr),
        ID:         types.Int64Value(functionInstanceUpdateResponse.Data.GetId()),
        Active:     types.BoolValue(functionInstanceUpdateResponse.Data.GetActive()),
    }

    plan.ID = types.Int64Value(functionInstanceUpdateResponse.Data.GetId())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Delete Method

```go
func (r *functionInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state FunctionInstanceResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    if state.Function.ID.IsNull() {
        resp.Diagnostics.AddError("Function Instance id error", "is not null")
        return
    }

    if state.ApplicationID.IsNull() {
        resp.Diagnostics.AddError("Application ID error", "is not null")
        return
    }

    _, response, err := r.client.api.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueInt64(), state.Function.ID.ValueInt64()).Execute()
    if err != nil {
        if response != nil && response.StatusCode == http.StatusNotFound {
            // Resource already deleted, consider this a success
            return
        }
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueInt64(), state.Function.ID.ValueInt64()).Execute()
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
}
```

### ImportState Method

```go
func (r *functionInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Import format: "applicationID/instanceID"
    parts := strings.Split(req.ID, "/")
    if len(parts) != 2 {
        resp.Diagnostics.AddError("Invalid import format", "Expected format: applicationID/instanceID")
        return
    }

    applicationID, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid application ID", err.Error())
        return
    }

    instanceID, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Invalid instance ID", err.Error())
        return
    }

    state := FunctionInstanceResourceModel{
        ApplicationID: types.Int64Value(applicationID),
        ID:            types.Int64Value(instanceID),
    }

    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

## Schema Definition Patterns

### Resource Schema

```go
func (r *functionInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_function_instance"
}

func (r *functionInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.Int64Attribute{
                Computed: true,
            },
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "data": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The function instance identifier.",
                        Computed:    true,
                    },
                    "function_id": schema.Int64Attribute{
                        Description: "The function identifier.",
                        Required:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the function.",
                        Required:    true,
                    },
                    "args": schema.StringAttribute{
                        Description: "JSON arguments of the function.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Whether the function instance is active.",
                        Optional:    true,
                        Computed:    true,
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
        result, response, err = utils.RetryOn429(func() (*ResponseType, *http.Response, error) {
            return client.API.Method(ctx, params).Execute()
        }, 5)  // Max 5 retries

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
```

### Special Error Codes

```go
// For Read operations - handle 404 specially
if response.StatusCode == http.StatusNotFound {
    resp.State.RemoveResource(ctx)  // Mark resource as deleted
    return
}
```

---

## Documentation and Examples

### Documentation Files

Documentation is auto-generated by `terraform-plugin-docs` and located in:

| Type | Location |
|------|----------|
| Singular Data Source Doc | `docs/data-sources/application_function_instance.md` |
| Plural Data Source Doc | `docs/data-sources/application_functions_instance.md` |
| Resource Doc | `docs/resources/application_functions_instance.md` |

### Example Files

Example Terraform configurations are located in:

| Type | Location |
|------|----------|
| Singular Data Source Example | `examples/data-sources/azion_application_function_instance/data-source.tf` |
| Plural Data Source Example | `examples/data-sources/azion_application_functions_instance/data-source.tf` |
| Resource Example | `examples/resources/azion_application_functions_instance/resource.tf` |

### Example: Resource Usage

```terraform
resource "azion_application_function_instance" "example" {
  application_id = 1234567890
  data = {
    name        = "Terraform Example"
    function_id = 12345
    active      = true
    args = jsonencode({
      key     = "Value"
      Example = "example"
    })
  }
}
```

### Example: Singular Data Source

```terraform
data "azion_application_function_instance" "example" {
  application_id = 1234567890
  id             = 123456
}
```

### Example: Plural Data Source

```terraform
data "azion_application_function_instances" "example" {
  application_id = 1234567890
}
```

### Example: Import

```shell
terraform import azion_application_function_instance.example 12345/67890
```

---

## Common Issues

### 1. Using Wrong SDK

**Problem:** Using `edge-api` instead of `azion-api`.

**Solution:** Always use `azion-api` for Application Function Instance:

```go
// WRONG
import edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"

// CORRECT
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### 2. Wrong Response Type for List

**Problem:** Using `FunctionInstanceResponse` for list operations.

**Solution:** Use `PaginatedFunctionInstanceList` for list operations:

```go
// WRONG - using wrong type
var response *azionapi.FunctionInstanceResponse
response, _, _ = d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, appID).Execute()

// CORRECT
funcInstancesResponse, _, _ := d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, appID).Execute()
// Use funcInstancesResponse.GetResults() to iterate
```

### 3. Not Handling 429 Status Code

**Problem:** Not retrying on rate limit errors.

**Solution:** Always wrap API calls with 429 handling:

```go
if response.StatusCode == 429 {
    response, err = utils.RetryOn429(func() (*ResponseType, *http.Response, error) {
        return client.API.Method(ctx, params).Execute()
    }, 5)
}
```

### 4. Not Closing Response Body

**Problem:** Not closing HTTP response body after successful retries.

**Solution:** Always close the response body after 429 retry:

```go
if response != nil {
    defer response.Body.Close()
}
```

### 5. Missing Args JSON Conversion

**Problem:** Not converting args between interface{} and string.

**Solution:** Use utility functions:

```go
// For reading from API response
jsonArgsStr, err := utils.ConvertInterfaceToString(response.Data.GetArgs())

// For writing to API request
planJsonArgs, err := utils.UnmarshallJsonArgs(argsStr)
```

### 6. Import State Format

**Problem:** Using wrong import format.

**Solution:** Use `applicationID/instanceID` format:

```shell
terraform import azion_application_function_instance.example 12345/67890
```

### 7. Using "edge" Prefix in Go Code

**Problem:** Using "edge" prefix in variable names, struct names, or comments.

**Solution:** Remove all "edge" prefixes from Go code:

```go
// WRONG
type EdgeFunctionInstanceResource struct { ... }
edgeApplicationID := ...

// CORRECT
type FunctionInstanceResource struct { ... }
applicationID := ...
```

### 8. Setting active = false During Creation

**Problem:** The API does not allow setting `active = false` when creating a function instance. The error message is: `{"errors":[{"code":"29002","title":"Cant Deactivate Function Instance","detail":"You can't deactivate a function instance."}]}`

**Solution:** Do not set `active = false` during creation. The `active` field is computed by the API and is always `true` for new instances. If you need to deactivate a function instance, use the update operation after creation.

```terraform
# WRONG - will fail with 400 Bad Request
resource "azion_application_function_instance" "example" {
  application_id = 1234567890
  data = {
    name        = "example"
    function_id = 12345
    active      = false  # This is not allowed during creation
  }
}

# CORRECT - omit active field, let API compute it
resource "azion_application_function_instance" "example" {
  application_id = 1234567890
  data = {
    name        = "example"
    function_id = 12345
    # active is computed by API (always true for new instances)
  }
}
```

### 9. Missing page and page_size Attributes in Plural Data Source

**Problem:** The plural data source schema is missing `page` and `page_size` attributes, causing errors when reading the configuration.

**Solution:** Always include `page` and `page_size` as optional attributes in the plural data source schema:

```go
func (d *ApplicationFunctionInstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            // ... other attributes
            "page": schema.Int64Attribute{
                Description: "Page number for pagination.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "Number of items per page.",
                Optional:    true,
            },
            // ... other attributes
        },
    }
}
```

---

## Files Reference

### Files to Update When Making Schema Changes

When changing the schema of Application Function Instance resources, update these files:

| File Type | Location | Purpose |
|-----------|----------|---------|
| Singular Data Source | `internal/data_source_application_function_instance.go` | Read single instance |
| Plural Data Source | `internal/data_source_application_functions_instance.go` | List instances |
| Resource | `internal/resource_application_functions_instance.go` | CRUD operations |
| Singular Data Source Doc | `docs/data-sources/application_function_instance.md` | Documentation |
| Plural Data Source Doc | `docs/data-sources/application_functions_instance.md` | Documentation |
| Resource Doc | `docs/resources/application_functions_instance.md` | Documentation |
| Singular Example | `examples/data-sources/azion_application_function_instance/data-source.tf` | Example usage |
| Plural Example | `examples/data-sources/azion_application_functions_instance/data-source.tf` | Example usage |
| Resource Example | `examples/resources/azion_application_functions_instance/resource.tf` | Example usage |

---

## Provider Registration

The resources and data sources are registered in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ... other data sources
        dataSourceAzionApplicationFunctionInstances,  // Plural
        dataSourceAzionApplicationFunctionInstance,   // Singular
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... other resources
        NewApplicationFunctionInstanceResource,
        // ... other resources
    }
}
```

---

## Running Linters

After making changes, run the linters:

```bash
golangci-lint run --config .golintci.yml ./internal/...
```

Or build the project to check for compilation errors:

```bash
go build ./...
```

---

## Summary Checklist

When implementing or updating Application Function Instance resources:

- [ ] Use `azion-api` SDK, not `edge-api`
- [ ] Remove "edge" prefix from all Go code (structs, variables, comments, function names)
- [ ] Use `int64` for IDs
- [ ] Handle 429 status codes with `utils.RetryOn429`
- [ ] Close response body after successful retries
- [ ] Convert args using `utils.ConvertInterfaceToString` and `utils.UnmarshallJsonArgs`
- [ ] Support import format "applicationID/instanceID"
- [ ] Use PATCH for partial updates with `PatchedFunctionInstanceRequest`
- [ ] Update documentation and examples after schema changes
- [ ] Run linters: `golangci-lint run --config .golintci.yml ./internal/...`
- [ ] Build project: `go build ./...`
