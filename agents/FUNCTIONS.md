# Functions - Code Generation Guide

This document provides specific guidance for implementing Edge Functions resources and data sources in the Terraform provider.

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

Edge Functions use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Edge Function (Singular Data Source) | `azion-api` (v4) | `api.FunctionsAPI` | `https://api.azion.com/v4` |
| Edge Functions (Plural Data Source) | `azion-api` (v4) | `api.FunctionsAPI` | `https://api.azion.com/v4` |
| Edge Function (Resource) | `azion-api` (v4) | `api.FunctionsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `FunctionsRequest` |
| Update Request Type | `PatchedFunctionsRequest` |
| Response Type | `FunctionResponse` with `Data` field |
| List Response Type | `PaginatedFunctionsList` |
| Create Pattern | `.CreateFunction(ctx).FunctionsRequest(req).Execute()` |
| Update Pattern | `.PartialUpdateFunction(ctx, id).PatchedFunctionsRequest(req).Execute()` |
| List Method | `.ListFunctions(ctx).Execute()` |
| Delete Method | `.DeleteFunction(ctx, functionId).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - preferred for all implementations
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // Legacy V4 SDK (azionapi-v4-go-sdk-dev/edge-api) - kept for backward compatibility
    edgeConfig *edgeapi.Configuration
    edgeApi    *edgeapi.APIClient
    
    // Legacy SDKs (azionapi-go-sdk) - deprecated
    edgefunctionsApi *edgefunctions.APIClient
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

For reading a single Edge Function by its identifier:

**File:** `internal/data_source_edgeFunction.go`

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
    _ datasource.DataSource              = &EdgeFunctionDataSource{}
    _ datasource.DataSourceWithConfigure = &EdgeFunctionDataSource{}
)

// Constructor function
func dataSourceAzionEdgeFunction() datasource.DataSource {
    return &EdgeFunctionDataSource{}
}

// DataSource struct - holds the client
type EdgeFunctionDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state
type EdgeFunctionDataSourceModel struct {
    Data EdgeFunctionResults `tfsdk:"data"`
    ID   types.String        `tfsdk:"id"`
}

// Results struct - represents the API response data
type EdgeFunctionResults struct {
    ID                   types.Int64  `tfsdk:"id"`
    Name                 types.String `tfsdk:"name"`
    LastEditor           types.String `tfsdk:"last_editor"`
    LastModified         types.String `tfsdk:"last_modified"`
    ProductVersion       types.String `tfsdk:"product_version"`
    Active               types.Bool   `tfsdk:"active"`
    Runtime              types.String `tfsdk:"runtime"`
    ExecutionEnvironment types.String `tfsdk:"execution_environment"`
    Code                 types.String `tfsdk:"code"`
    DefaultArgs          types.String `tfsdk:"default_args"`
    ReferenceCount       types.Int64  `tfsdk:"reference_count"`
    Version              types.String `tfsdk:"version"`
    Vendor               types.String `tfsdk:"vendor"`
}

// Configure - receives the API client
func (d *EdgeFunctionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

// Metadata - sets the data source type name
func (d *EdgeFunctionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_function"
}

// Schema - defines the Terraform schema
func (d *EdgeFunctionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
                        Description: "The function identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the function.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "The last editor of the function.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the function.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version of the function.",
                        Computed:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Status of the function.",
                        Computed:    true,
                    },
                    "runtime": schema.StringAttribute{
                        Description: "Runtime of the function.",
                        Computed:    true,
                    },
                    "execution_environment": schema.StringAttribute{
                        Description: "Execution environment of the function.",
                        Computed:    true,
                    },
                    "code": schema.StringAttribute{
                        Description: "Code of the function.",
                        Computed:    true,
                    },
                    "default_args": schema.StringAttribute{
                        Description: "Default arguments of the function as JSON.",
                        Computed:    true,
                    },
                    "reference_count": schema.Int64Attribute{
                        Description: "The reference count of the function.",
                        Computed:    true,
                    },
                    "version": schema.StringAttribute{
                        Description: "Version of the function.",
                        Computed:    true,
                    },
                    "vendor": schema.StringAttribute{
                        Description: "Vendor of the function.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}

// Read - performs the API call to retrieve a single function
func (d *EdgeFunctionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getEdgeFunctionId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("id"), &getEdgeFunctionId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Convert string ID to int64
    edgeFunctionID, err := strconv.ParseInt(getEdgeFunctionId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert ID",
        )
        return
    }

    // API call using V4 SDK
    functionsResponse, response, err := d.client.api.FunctionsAPI.
        RetrieveFunction(ctx, edgeFunctionID).Execute() //nolint
    if err != nil {
        // Handle rate limiting (429)
        if response.StatusCode == 429 {
            functionsResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionResponse, *http.Response, error) {
                return d.client.api.FunctionsAPI.RetrieveFunction(ctx, edgeFunctionID).Execute() //nolint
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
            usrMsg, errMsg := errPrint(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    // Convert default_args from interface{} to JSON string
    defaultArgsStr := ""
    if functionsResponse.Data.DefaultArgs != nil {
        var err error
        defaultArgsStr, err = utils.ConvertInterfaceToString(functionsResponse.Data.DefaultArgs)
        if err != nil {
            resp.Diagnostics.AddError(
                err.Error(),
                "Failed to convert default_args to string",
            )
            return
        }
    }

    // Build state from response
    EdgeFunctionState := EdgeFunctionDataSourceModel{
        Data: EdgeFunctionResults{
            ID:                   types.Int64Value(functionsResponse.Data.Id),
            Name:                 types.StringValue(functionsResponse.Data.Name),
            Code:                 types.StringValue(functionsResponse.Data.Code),
            DefaultArgs:          types.StringValue(defaultArgsStr),
            ExecutionEnvironment: types.StringValue(*functionsResponse.Data.ExecutionEnvironment),
            Active:               types.BoolValue(*functionsResponse.Data.Active),
            LastEditor:           types.StringValue(functionsResponse.Data.LastEditor),
            LastModified:         types.StringValue(functionsResponse.Data.LastModified.Format(time.RFC850)),
            ProductVersion:       types.StringValue(functionsResponse.Data.ProductVersion),
            Version:              types.StringValue(functionsResponse.Data.Version),
            Vendor:               types.StringValue(functionsResponse.Data.Vendor),
            ReferenceCount:       types.Int64Value(functionsResponse.Data.ReferenceCount),
        },
    }

    // Handle optional fields
    if functionsResponse.Data.Runtime != nil {
        EdgeFunctionState.Data.Runtime = types.StringValue(*functionsResponse.Data.Runtime)
    }

    EdgeFunctionState.ID = types.StringValue("Get By Id Edge Function")
    diags = resp.State.Set(ctx, &EdgeFunctionState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}

// Error helper function
func errPrint(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Edge Function found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

### Plural Data Source (List Multiple Resources)

For listing all Edge Functions:

**File:** `internal/data_source_edgeFunctions.go`

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

// Interface assertions
var (
    _ datasource.DataSource              = &EdgeFunctionsDataSource{}
    _ datasource.DataSourceWithConfigure = &EdgeFunctionsDataSource{}
)

// Constructor function
func dataSourceAzionEdgeFunctions() datasource.DataSource {
    return &EdgeFunctionsDataSource{}
}

// DataSource struct - holds the client
type EdgeFunctionsDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state
type EdgeFunctionsDataSourceModel struct {
    Counter types.Int64            `tfsdk:"counter"`
    Results []EdgeFunctionsResults `tfsdk:"results"`
    ID      types.String           `tfsdk:"id"`
}

// Results struct - represents each item in the list
type EdgeFunctionsResults struct {
    ID                   types.Int64  `tfsdk:"id"`
    Name                 types.String `tfsdk:"name"`
    LastEditor           types.String `tfsdk:"last_editor"`
    LastModified         types.String `tfsdk:"last_modified"`
    ProductVersion       types.String `tfsdk:"product_version"`
    Active               types.Bool   `tfsdk:"active"`
    Runtime              types.String `tfsdk:"runtime"`
    ExecutionEnvironment types.String `tfsdk:"execution_environment"`
    Code                 types.String `tfsdk:"code"`
    DefaultArgs          types.String `tfsdk:"default_args"`
    ReferenceCount       types.Int64  `tfsdk:"reference_count"`
    Version              types.String `tfsdk:"version"`
    Vendor               types.String `tfsdk:"vendor"`
}

// Configure - receives the API client
func (d *EdgeFunctionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

// Metadata - sets the data source type name
func (d *EdgeFunctionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_edge_functions"
}

// Schema - defines the Terraform schema
func (d *EdgeFunctionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total count of edge functions.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "The function identifier.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "Name of the function.",
                            Computed:    true,
                        },
                        "last_editor": schema.StringAttribute{
                            Description: "The last editor of the function.",
                            Computed:    true,
                        },
                        "last_modified": schema.StringAttribute{
                            Description: "Last modified timestamp of the function.",
                            Computed:    true,
                        },
                        "product_version": schema.StringAttribute{
                            Description: "Product version of the function.",
                            Computed:    true,
                        },
                        "active": schema.BoolAttribute{
                            Description: "Status of the function.",
                            Computed:    true,
                        },
                        "runtime": schema.StringAttribute{
                            Description: "Runtime of the function.",
                            Computed:    true,
                        },
                        "execution_environment": schema.StringAttribute{
                            Description: "Execution environment of the function.",
                            Computed:    true,
                        },
                        "code": schema.StringAttribute{
                            Description: "Code of the function.",
                            Computed:    true,
                        },
                        "default_args": schema.StringAttribute{
                            Description: "Default arguments of the function as JSON.",
                            Computed:    true,
                        },
                        "reference_count": schema.Int64Attribute{
                            Description: "The reference count of the function.",
                            Computed:    true,
                        },
                        "version": schema.StringAttribute{
                            Description: "Version of the function.",
                            Computed:    true,
                        },
                        "vendor": schema.StringAttribute{
                            Description: "Vendor of the function.",
                            Computed:    true,
                        },
                    },
                },
            },
        },
    }
}

// Read - performs the API call to list all functions
func (d *EdgeFunctionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // API call using V4 SDK
    functionsResponse, response, err := d.client.api.FunctionsAPI.ListFunctions(ctx).Execute() //nolint
    if err != nil {
        // Handle rate limiting (429)
        if response.StatusCode == 429 {
            functionsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedFunctionsList, *http.Response, error) {
                return d.client.api.FunctionsAPI.ListFunctions(ctx).Execute() //nolint
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
            usrMsg, errMsg := errPrintFunctions(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    // Build state from response
    edgeFunctionsState := EdgeFunctionsDataSourceModel{
        Counter: types.Int64Value(*functionsResponse.Count),
    }

    // Iterate over results
    for _, resultEdgeFunctions := range functionsResponse.GetResults() {
        defaultArgsStr := ""
        if resultEdgeFunctions.DefaultArgs != nil {
            var err error
            defaultArgsStr, err = utils.ConvertInterfaceToString(resultEdgeFunctions.DefaultArgs)
            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "Failed to convert default_args to string",
                )
                return
            }
        }

        result := EdgeFunctionsResults{
            ID:             types.Int64Value(resultEdgeFunctions.Id),
            Name:           types.StringValue(resultEdgeFunctions.Name),
            Code:           types.StringValue(resultEdgeFunctions.Code),
            DefaultArgs:    types.StringValue(defaultArgsStr),
            Active:         types.BoolValue(*resultEdgeFunctions.Active),
            LastEditor:     types.StringValue(resultEdgeFunctions.LastEditor),
            ProductVersion: types.StringValue(resultEdgeFunctions.ProductVersion),
            Version:        types.StringValue(resultEdgeFunctions.Version),
            Vendor:         types.StringValue(resultEdgeFunctions.Vendor),
            ReferenceCount: types.Int64Value(resultEdgeFunctions.ReferenceCount),
        }

        // Handle optional fields
        if resultEdgeFunctions.Runtime != nil {
            result.Runtime = types.StringValue(*resultEdgeFunctions.Runtime)
        }

        edgeFunctionsState.Results = append(edgeFunctionsState.Results, result)
    }
    
    edgeFunctionsState.ID = types.StringValue("Get All Edge Functions")
    diags := resp.State.Set(ctx, &edgeFunctionsState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}

// Error helper function
func errPrintFunctions(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Edge Functions found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`edge_function`) | Plural (`edge_functions`) |
|--------|---------------------------|--------------------------|
| **ID Attribute** | Required (user provides) | Computed (generated) |
| **Schema Structure** | `SingleNestedAttribute` for data | `ListNestedAttribute` for results |
| **API Method** | `RetrieveFunction(ctx, id)` | `ListFunctions(ctx)` |
| **Response Type** | `FunctionResponse` | `PaginatedFunctionsList` |
| **Data Access** | `response.Data` (single object) | `response.GetResults()` (slice) |
| **Counter Field** | Not present | Present (`types.Int64`) |
| **Error on 404** | "No Edge Function found" | "No Edge Functions found" |

---

## Resource Implementation

The resource implementation (`resource_edgeFunction.go`) uses the `azion-api` SDK for all CRUD operations.

### Resource SDK Methods (V4 SDK)

```go
// Create
createEdgeFunction, response, err := r.client.api.FunctionsAPI.
    CreateFunction(ctx).FunctionsRequest(edgeFunction).Execute()

// Read
getEdgeFunction, response, err := r.client.api.FunctionsAPI.
    RetrieveFunction(ctx, edgeFunctionId).Execute()

// Update (PATCH - partial update)
updateEdgeFunction, response, err := r.client.api.FunctionsAPI.
    PartialUpdateFunction(ctx, edgeFunctionId).
    PatchedFunctionsRequest(updateEdgeFunctionRequest).Execute()

// Delete
_, response, err := r.client.api.FunctionsAPI.
    DeleteFunction(ctx, edgeFunctionId).Execute()
```

### Create Operation Pattern

```go
func (r *edgeFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan edgeFunctionResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the request struct
    edgeFunction := azionapi.FunctionsRequest{
        Name: plan.EdgeFunction.Name.ValueString(),
        Code: plan.EdgeFunction.Code.ValueString(),
    }

    // Use setter methods for optional fields
    if !plan.EdgeFunction.Active.IsNull() && !plan.EdgeFunction.Active.IsUnknown() {
        edgeFunction.SetActive(plan.EdgeFunction.Active.ValueBool())
    }

    if !plan.EdgeFunction.ExecutionEnvironment.IsNull() && !plan.EdgeFunction.ExecutionEnvironment.IsUnknown() {
        edgeFunction.SetExecutionEnvironment(plan.EdgeFunction.ExecutionEnvironment.ValueString())
    }

    if !plan.EdgeFunction.Runtime.IsNull() && !plan.EdgeFunction.Runtime.IsUnknown() {
        edgeFunction.SetRuntime(plan.EdgeFunction.Runtime.ValueString())
    }

    if !plan.EdgeFunction.DefaultArgs.IsNull() && !plan.EdgeFunction.DefaultArgs.IsUnknown() {
        planJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.DefaultArgs.ValueString())
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "err")
            return
        }
        edgeFunction.SetDefaultArgs(planJsonArgs)
    }

    // Execute the API call
    createEdgeFunction, response, err := r.client.api.FunctionsAPI.
        CreateFunction(ctx).FunctionsRequest(edgeFunction).Execute() //nolint
    if err != nil {
        // Handle errors (see Error Handling section)
    }

    // Build state from response
    // ...
}
```

### Update Operation Pattern (PATCH)

```go
func (r *edgeFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan edgeFunctionResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state edgeFunctionResourceModel
    diagsEdgeFunction := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsEdgeFunction...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the patch request - all fields are optional pointers
    updateEdgeFunctionRequest := azionapi.PatchedFunctionsRequest{}

    // Use setter methods for all fields
    if !plan.EdgeFunction.Name.IsNull() && !plan.EdgeFunction.Name.IsUnknown() {
        updateEdgeFunctionRequest.SetName(plan.EdgeFunction.Name.ValueString())
    }

    if !plan.EdgeFunction.Code.IsNull() && !plan.EdgeFunction.Code.IsUnknown() {
        updateEdgeFunctionRequest.SetCode(plan.EdgeFunction.Code.ValueString())
    }

    if !plan.EdgeFunction.Active.IsNull() && !plan.EdgeFunction.Active.IsUnknown() {
        updateEdgeFunctionRequest.SetActive(plan.EdgeFunction.Active.ValueBool())
    }

    if !plan.EdgeFunction.ExecutionEnvironment.IsNull() && !plan.EdgeFunction.ExecutionEnvironment.IsUnknown() {
        updateEdgeFunctionRequest.SetExecutionEnvironment(plan.EdgeFunction.ExecutionEnvironment.ValueString())
    }

    if !plan.EdgeFunction.Runtime.IsNull() && !plan.EdgeFunction.Runtime.IsUnknown() {
        updateEdgeFunctionRequest.SetRuntime(plan.EdgeFunction.Runtime.ValueString())
    }

    if !plan.EdgeFunction.DefaultArgs.IsNull() && !plan.EdgeFunction.DefaultArgs.IsUnknown() {
        requestJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.DefaultArgs.ValueString())
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "err")
            return
        }
        updateEdgeFunctionRequest.SetDefaultArgs(requestJsonArgs)
    }

    // Get function ID from state
    var edgeFunctionId int64
    var err error
    if state.ID.IsNull() {
        edgeFunctionId = state.EdgeFunction.ID.ValueInt64()
    } else {
        edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error ", "Could not convert edgeFunctionId to int")
            return
        }
    }

    // Execute the API call
    updateEdgeFunction, response, err := r.client.api.FunctionsAPI.
        PartialUpdateFunction(ctx, edgeFunctionId).
        PatchedFunctionsRequest(updateEdgeFunctionRequest).Execute() //nolint
    if err != nil {
        // Handle errors (see Error Handling section)
    }

    // Build state from response
    // ...
}
```

### Delete Operation Pattern

```go
func (r *edgeFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state edgeFunctionResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var edgeFunctionId int64
    var err error
    if state.EdgeFunction != nil {
        edgeFunctionId = state.EdgeFunction.ID.ValueInt64()
    } else {
        edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
        if err != nil {
            resp.Diagnostics.AddError("Value Conversion error ", "Could not convert Edge Function ID")
            return
        }
    }

    _, response, err := r.client.api.FunctionsAPI.DeleteFunction(ctx, edgeFunctionId).Execute() //nolint
    if err != nil {
        // Handle errors (see Error Handling section)
    }
}
```

---

## Schema Definition Patterns

### Field Mappings (API to Terraform)

| API Field | Terraform Attribute | Type | Notes |
|-----------|---------------------|------|-------|
| `id` | `id` | `Int64` | Function identifier |
| `name` | `name` | `String` | Function name |
| `last_editor` | `last_editor` | `String` | Last user to edit |
| `last_modified` | `last_modified` | `String` | `time.Time` formatted to RFC850 |
| `product_version` | `product_version` | `String` | Product version |
| `active` | `active` | `Bool` | Pointer in API |
| `runtime` | `runtime` | `String` | Optional (pointer in API) |
| `execution_environment` | `execution_environment` | `String` | Pointer in API |
| `code` | `code` | `String` | Function source code |
| `default_args` | `default_args` | `String` | `interface{}` converted to JSON string |
| `reference_count` | `reference_count` | `Int64` | Number of references |
| `version` | `version` | `String` | Installed version |
| `vendor` | `vendor` | `String` | Function vendor |

### Handling Optional/Pointer Fields

```go
// API returns pointers for optional fields
if functionsResponse.Data.Runtime != nil {
    EdgeFunctionState.Data.Runtime = types.StringValue(*functionsResponse.Data.Runtime)
}

// API returns pointers for Active
Active: types.BoolValue(*functionsResponse.Data.Active)
```

### Handling Default Args (interface{})

```go
// Convert from API (interface{}) to Terraform (string)
defaultArgsStr := ""
if functionsResponse.Data.DefaultArgs != nil {
    var err error
    defaultArgsStr, err = utils.ConvertInterfaceToString(functionsResponse.Data.DefaultArgs)
    if err != nil {
        resp.Diagnostics.AddError(
            err.Error(),
            "Failed to convert default_args to string",
        )
        return
    }
}
```

---

## Error Handling

### Standard Error Pattern

```go
if err != nil {
    // 1. Handle rate limiting (429)
    if response.StatusCode == 429 {
        functionsResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionResponse, *http.Response, error) {
            return d.client.api.FunctionsAPI.RetrieveFunction(ctx, edgeFunctionID).Execute()
        }, 5)  // Max 5 retries
        
        if response != nil {
            defer response.Body.Close()
        }
        
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // 2. Use error helper for common status codes
        usrMsg, errMsg := errPrint(response.StatusCode, err)
        resp.Diagnostics.AddError(usrMsg, errMsg)
        return
    }
}
```

### Error Helper Function

```go
func errPrint(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Edge Function found"  // or "No Edge Functions found" for plural
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

## Type Conversions

### Time Formatting

```go
// From API response (time.Time) to Terraform (string)
LastModified: types.StringValue(functionsResponse.Data.LastModified.Format(time.RFC850))
```

### String ID to Int64

```go
// For singular data source where ID comes from user input
edgeFunctionID, err := strconv.ParseInt(getEdgeFunctionId.ValueString(), 10, 64)
if err != nil {
    resp.Diagnostics.AddError(
        "Value Conversion error ",
        "Could not convert ID",
    )
    return
}
```

### Default Args (interface{} to String)

```go
// Convert from API (interface{}) to Terraform (string)
defaultArgsStr := ""
if functionsResponse.Data.DefaultArgs != nil {
    var err error
    defaultArgsStr, err = utils.ConvertInterfaceToString(functionsResponse.Data.DefaultArgs)
    if err != nil {
        resp.Diagnostics.AddError(
            err.Error(),
            "Failed to convert default_args to string",
        )
        return
    }
}
```

---

## Common Issues

### Issue: Using Wrong SDK Client

**Problem:** Using `edgeApi` instead of `api` for V4 SDK operations.

**Solution:** Always use `r.client.api.FunctionsAPI` (azion-api SDK) for all Edge Functions operations.

```go
// WRONG - uses legacy edge-api SDK
functionsResponse, response, err := r.client.edgeApi.FunctionsAPI.ListFunctions(ctx).Execute()

// CORRECT - uses azion-api SDK
functionsResponse, response, err := r.client.api.FunctionsAPI.ListFunctions(ctx).Execute()
```

### Issue: Using Wrong Request Type Names

**Problem:** Using `EdgeFunctionsRequest` or `PatchedEdgeFunctionsRequest` instead of the correct types.

**Solution:** The correct types are `FunctionsRequest` and `PatchedFunctionsRequest` (without "Edge" prefix).

```go
// WRONG
edgeFunction := azionapi.EdgeFunctionsRequest{...}
updateRequest := azionapi.PatchedEdgeFunctionsRequest{...}

// CORRECT
edgeFunction := azionapi.FunctionsRequest{...}
updateRequest := azionapi.PatchedFunctionsRequest{...}
```

### Issue: Using Wrong Method Names

**Problem:** Using `.EdgeFunctionsRequest()` or `.PatchedEdgeFunctionsRequest()` method names.

**Solution:** The correct method names are `.FunctionsRequest()` and `.PatchedFunctionsRequest()`.

```go
// WRONG
r.client.api.FunctionsAPI.CreateFunction(ctx).EdgeFunctionsRequest(edgeFunction).Execute()
r.client.api.FunctionsAPI.PartialUpdateFunction(ctx, id).PatchedEdgeFunctionsRequest(req).Execute()

// CORRECT
r.client.api.FunctionsAPI.CreateFunction(ctx).FunctionsRequest(edgeFunction).Execute()
r.client.api.FunctionsAPI.PartialUpdateFunction(ctx, id).PatchedFunctionsRequest(req).Execute()
```

### Issue: Not Using Setter Methods

**Problem:** Trying to assign pointer fields directly on `FunctionsRequest` or `PatchedFunctionsRequest`.

**Solution:** Use the setter methods provided by the SDK instead of direct assignment.

```go
// WRONG - direct assignment (may not work correctly)
edgeFunction.Active = azionapi.PtrBool(true)
edgeFunction.Runtime = azionapi.PtrString("azion_js")

// CORRECT - use setter methods
edgeFunction.SetActive(true)
edgeFunction.SetRuntime("azion_js")
```

### Issue: Missing Error Helper Function

**Problem:** Error messages are not user-friendly.

**Solution:** Use the `errPrint` helper function for consistent error messages.

```go
usrMsg, errMsg := errPrint(response.StatusCode, err)
resp.Diagnostics.AddError(usrMsg, errMsg)
```

### Issue: Not Handling Optional Fields

**Problem:** Runtime field causes panic when nil.

**Solution:** Always check for nil before dereferencing pointers.

```go
// Handle optional/pointer fields
if functionsResponse.Data.Runtime != nil {
    EdgeFunctionState.Data.Runtime = types.StringValue(*functionsResponse.Data.Runtime)
}
```

### Issue: Not Converting Default Args

**Problem:** `default_args` is returned as `interface{}` from API but needs to be a JSON string in Terraform.

**Solution:** Use `utils.ConvertInterfaceToString` helper.

```go
defaultArgsStr := ""
if functionsResponse.Data.DefaultArgs != nil {
    var err error
    defaultArgsStr, err = utils.ConvertInterfaceToString(functionsResponse.Data.DefaultArgs)
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "Failed to convert default_args to string")
        return
    }
}
```

---

## Summary Checklist

When implementing or updating Edge Functions resources and data sources:

1. **Use correct SDK**: `azion-api` (`r.client.api.FunctionsAPI` or `d.client.api.FunctionsAPI`)
2. **Import statement**: `import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"`
3. **Request types**: Use `FunctionsRequest` (create) and `PatchedFunctionsRequest` (update)
4. **Method names**: Use `.FunctionsRequest()` and `.PatchedFunctionsRequest()` methods
5. **Use setter methods**: Use `SetName()`, `SetCode()`, `SetActive()`, etc. instead of direct assignment
6. **Handle 429 errors**: Use `utils.RetryOn429` with correct response type
7. **Convert default_args**: Use `utils.ConvertInterfaceToString`
8. **Handle optional fields**: Check for nil before dereferencing pointers
9. **Format time**: Use `time.RFC850` for `last_modified`
10. **Use error helper**: Use `errPrint` for consistent error messages
11. **Singular vs Plural**: Remember key differences in ID handling and schema structure
