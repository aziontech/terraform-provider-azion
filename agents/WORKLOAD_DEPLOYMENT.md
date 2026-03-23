# Workload Deployments - Code Generation Guide

This document provides specific guidance for implementing Workload Deployment data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
3. [Schema Definition Patterns](#schema-definition-patterns)
4. [Error Handling](#error-handling)
5. [Type Conversions](#type-conversions)
6. [Common Issues](#common-issues)

---

## SDK Selection

Workload Deployments use the **V4 SDK (`azion-api`)** for all data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Workload Deployment (Singular Data Source) | `azion-api` (v4) | `api.WorkloadDeploymentsAPI` | `https://api.azion.com/v4` |
| Workload Deployments (Plural Data Source) | `azion-api` (v4) | `api.WorkloadDeploymentsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Response Type | `WorkloadDeploymentResponse` with `Data` field |
| List Response Type | `PaginatedWorkloadDeploymentList` |
| Retrieve Pattern | `.RetrieveWorkloadDeployment(ctx, deploymentId, workloadId).Execute()` |
| List Pattern | `.ListWorkloadDeployments(ctx, workloadId).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### Important: Naming Convention

**The "edge" prefix is NOT used in V4 SDK.** Use naming without the `edge` prefix for variables, structs, and function parameters:

| Avoid (Legacy) | Use (V4) |
|----------------|----------|
| `edgeWorkloadId` | `workloadID` |
| `edgeDeploymentId` | `deploymentID` |

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Workload Deployment by its identifier:

**File:** `internal/data_source_workload_deployment.go`

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
    _ datasource.DataSource              = &WorkloadDeploymentDataSource{}
    _ datasource.DataSourceWithConfigure = &WorkloadDeploymentDataSource{}
)

func dataSourceAzionWorkloadDeployment() datasource.DataSource {
    return &WorkloadDeploymentDataSource{}
}

type WorkloadDeploymentDataSource struct {
    client *apiClient
}

type WorkloadDeploymentDataSourceModel struct {
    WorkloadID   types.String                   `tfsdk:"workload_id"`
    DeploymentID types.String                   `tfsdk:"deployment_id"`
    Data         WorkloadDeploymentResultsModel `tfsdk:"data"`
    ID           types.String                   `tfsdk:"id"`
}

type WorkloadDeploymentResultsModel struct {
    ID           types.Int64                  `tfsdk:"id"`
    Name         types.String                 `tfsdk:"name"`
    Current      types.Bool                   `tfsdk:"current"`
    Active       types.Bool                   `tfsdk:"active"`
    Strategy     *DeploymentStrategyModel     `tfsdk:"strategy"`
    LastEditor   types.String                 `tfsdk:"last_editor"`
    LastModified types.String                 `tfsdk:"last_modified"`
}

type DeploymentStrategyModel struct {
    Type       types.String                   `tfsdk:"type"`
    Attributes *DeploymentStrategyAttrsModel  `tfsdk:"attributes"`
}

type DeploymentStrategyAttrsModel struct {
    Application types.Int64 `tfsdk:"application"`
    Firewall    types.Int64 `tfsdk:"firewall"`
    CustomPage  types.Int64 `tfsdk:"custom_page"`
}
```

### Key Implementation Points for Singular Data Source

1. **ID Parameters**: Requires both `workload_id` and `deployment_id` as string attributes
2. **API Call Pattern**: Uses `RetrieveWorkloadDeployment(ctx, deploymentId, workloadId).Execute()`
3. **Parameter Order**: Note that deploymentId comes BEFORE workloadId in the API call
4. **ID Conversion**: Convert string IDs to int64 using `strconv.ParseInt()`

### Read Method for Singular Data Source

```go
func (d *WorkloadDeploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getWorkloadId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("workload_id"), &getWorkloadId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var getDeploymentId types.String
    diags = req.Config.GetAttribute(ctx, path.Root("deployment_id"), &getDeploymentId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    workloadID, err := strconv.ParseInt(getWorkloadId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", "Could not convert workload_id")
        return
    }

    deploymentID, err := strconv.ParseInt(getDeploymentId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", "Could not convert deployment_id")
        return
    }

    deploymentResponse, response, err := d.client.api.WorkloadDeploymentsAPI.
        RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            deploymentResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
                return d.client.api.WorkloadDeploymentsAPI.RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute() //nolint
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            usrMsg, errMsg := errPrintWorkloadDeployment(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    // Populate state with response data...
}
```

---

### Plural Data Source (List Multiple Resources)

For listing all Workload Deployments for a specific Workload:

**File:** `internal/data_source_workload_deployments.go`

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
    _ datasource.DataSource              = &WorkloadDeploymentsDataSource{}
    _ datasource.DataSourceWithConfigure = &WorkloadDeploymentsDataSource{}
)

func dataSourceAzionWorkloadDeployments() datasource.DataSource {
    return &WorkloadDeploymentsDataSource{}
}

type WorkloadDeploymentsDataSource struct {
    client *apiClient
}

type WorkloadDeploymentsDataSourceModel struct {
    WorkloadID types.String                      `tfsdk:"workload_id"`
    Count      types.Int64                       `tfsdk:"count"`
    Results    []WorkloadDeploymentsResultsModel `tfsdk:"results"`
    ID         types.String                      `tfsdk:"id"`
}
```

### Read Method for Plural Data Source

```go
func (d *WorkloadDeploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getWorkloadId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("workload_id"), &getWorkloadId)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    workloadID, err := strconv.ParseInt(getWorkloadId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", "Could not convert workload_id")
        return
    }

    deploymentsResponse, response, err := d.client.api.WorkloadDeploymentsAPI.
        ListWorkloadDeployments(ctx, workloadID).Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            deploymentsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedWorkloadDeploymentList, *http.Response, error) {
                return d.client.api.WorkloadDeploymentsAPI.ListWorkloadDeployments(ctx, workloadID).Execute() //nolint
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            usrMsg, errMsg := errPrintWorkloadDeployments(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    // Iterate over results and build state...
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`workload_deployment`) | Plural (`workload_deployments`) |
|--------|----------------------------------|--------------------------------|
| ID Parameter | Both `workload_id` AND `deployment_id` | Only `workload_id` |
| API Method | `RetrieveWorkloadDeployment` | `ListWorkloadDeployments` |
| Response Type | `WorkloadDeploymentResponse` | `PaginatedWorkloadDeploymentList` |
| Results Field | `data` (single object) | `results` (list of objects) |
| Count Field | No | Yes (`deployments_count`) |
| Terraform Name | `azion_workload_deployment` | `azion_workload_deployments` |

---

## Schema Definition Patterns

### Singular Data Source Schema

```go
func (d *WorkloadDeploymentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "workload_id": schema.StringAttribute{
                Description: "Numeric identifier of the Workload.",
                Required:    true,
            },
            "deployment_id": schema.StringAttribute{
                Description: "Numeric identifier of the Deployment.",
                Required:    true,
            },
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "data": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    // ... nested attributes
                },
            },
        },
    }
}
```

### Plural Data Source Schema

```go
func (d *WorkloadDeploymentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "workload_id": schema.StringAttribute{
                Description: "Numeric identifier of the Workload.",
                Required:    true,
            },
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "deployments_count": schema.Int64Attribute{
                Description: "The total number of deployments.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        // ... nested attributes
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
func errPrintWorkloadDeployment(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Workload Deployment found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

## Type Conversions

### Handling Nullable Fields in Strategy Attributes

The `DefaultDeploymentStrategyAttrs` has nullable fields for `Firewall` and `CustomPage`:

```go
attrs := strategy.GetAttributes()
strategyAttrsModel := &DeploymentStrategyAttrsModel{
    Application: types.Int64Value(attrs.GetApplication()),
}

// Handle optional firewall field
if attrs.Firewall.IsSet() {
    firewall := attrs.Firewall.Get()
    if firewall != nil {
        strategyAttrsModel.Firewall = types.Int64Value(*firewall)
    }
}

// Handle optional custom_page field
if attrs.CustomPage.IsSet() {
    customPage := attrs.CustomPage.Get()
    if customPage != nil {
        strategyAttrsModel.CustomPage = types.Int64Value(*customPage)
    }
}
```

### Time Formatting

```go
LastModified: types.StringValue(deploymentResponse.Data.LastModified.Format(time.RFC850)),
```

---

## Common Issues

### 1. Wrong Parameter Order in RetrieveWorkloadDeployment

**Issue**: The API expects `deploymentId` BEFORE `workloadId`.

**Wrong**:
```go
RetrieveWorkloadDeployment(ctx, workloadID, deploymentID)
```

**Correct**:
```go
RetrieveWorkloadDeployment(ctx, deploymentID, workloadID)
```

### 2. Not Using Nullable Fields Properly

**Issue**: `Firewall` and `CustomPage` are nullable fields using `NullableInt64`.

**Correct Pattern**:
```go
if attrs.Firewall.IsSet() {
    firewall := attrs.Firewall.Get()
    if firewall != nil {
        strategyAttrsModel.Firewall = types.Int64Value(*firewall)
    }
}
```

### 3. Forgetting to Handle 429 Rate Limiting

Always implement retry logic for rate-limited responses:

```go
if response.StatusCode == 429 {
    deploymentResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
        return d.client.api.WorkloadDeploymentsAPI.RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
    }, 5)

    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
        return
    }
}
```

### 4. Not Closing Response Body

**IMPORTANT**: The response body must be closed after successful API calls, not just inside retry blocks.

**Correct Pattern for Data Sources**:
```go
deploymentResponse, response, err := d.client.api.WorkloadDeploymentsAPI.
    RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
if err != nil {
    if response.StatusCode == 429 {
        deploymentResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
            return d.client.api.WorkloadDeploymentsAPI.RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
        }, 5)

        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // Handle other errors
        resp.Diagnostics.AddError(usrMsg, errMsg)
        return
    }
}
// Close response body after successful API call
if response != nil {
    defer response.Body.Close()
}
```

**Correct Pattern for Resources**:
```go
createDeployment, response, err := r.client.api.WorkloadDeploymentsAPI.
    CreateWorkloadDeployment(ctx, plan.WorkloadID.ValueInt64()).
    WorkloadDeploymentRequest(*deploymentRequest).Execute()
if err != nil {
    if response.StatusCode == 429 {
        createDeployment, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
            return r.client.api.WorkloadDeploymentsAPI.
                CreateWorkloadDeployment(ctx, plan.WorkloadID.ValueInt64()).
                WorkloadDeploymentRequest(*deploymentRequest).Execute()
        }, 5)

        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // Handle other errors
        bodyBytes, _ := io.ReadAll(response.Body)
        resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
        return
    }
}
// Close response body after successful API call
if response != nil {
    defer response.Body.Close()
}
```

The `defer response.Body.Close()` must be placed **after** the error handling block, so it only runs for successful responses.

---

## Registration in Provider

Both data sources must be registered in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // ... other data sources
        dataSourceAzionWorkloadDeployment,
        dataSourceAzionWorkloadDeployments,
    }
}
```

---

## Resource Implementation

For managing Workload Deployments (Create, Read, Update, Delete, Import):

**File:** `internal/resource_workload_deployment.go`

### Resource Model

```go
type WorkloadDeploymentResourceModel struct {
    Deployment   *WorkloadDeploymentResourceResults `tfsdk:"deployment"`
    ID           types.String                        `tfsdk:"id"`
    WorkloadID   types.Int64                         `tfsdk:"workload_id"`
    LastUpdated  types.String                        `tfsdk:"last_updated"`
}

type WorkloadDeploymentResourceResults struct {
    ID           types.Int64                           `tfsdk:"id"`
    Name         types.String                          `tfsdk:"name"`
    Current      types.Bool                            `tfsdk:"current"`
    Active       types.Bool                            `tfsdk:"active"`
    Strategy     *DeploymentStrategyResourceModel      `tfsdk:"strategy"`
    LastEditor   types.String                          `tfsdk:"last_editor"`
    LastModified types.String                          `tfsdk:"last_modified"`
}

type DeploymentStrategyResourceModel struct {
    Type       types.String                         `tfsdk:"type"`
    Attributes *DeploymentStrategyAttrsResourceModel `tfsdk:"attributes"`
}

type DeploymentStrategyAttrsResourceModel struct {
    Application types.Int64 `tfsdk:"application"`
    Firewall    types.Int64 `tfsdk:"firewall"`
    CustomPage  types.Int64 `tfsdk:"custom_page"`
}
```

### Key Implementation Points for Resource

1. **ID Format**: The resource uses a composite ID in the format `workloadID/deploymentID` for import support
2. **Parent Reference**: Requires `workload_id` as a required attribute to identify the parent workload
3. **API Call Pattern for Create**: Uses `CreateWorkloadDeployment(ctx, workloadId).WorkloadDeploymentRequest(request).Execute()`
4. **API Call Pattern for Update**: Uses `PartialUpdateWorkloadDeployment(ctx, deploymentId, workloadId).PatchedWorkloadDeploymentRequest(request).Execute()`
5. **API Call Pattern for Delete**: Uses `DeleteWorkloadDeployment(ctx, deploymentId, workloadId).Execute()`

### Create Method Pattern

```go
func (r *workloadDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan WorkloadDeploymentResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the strategy request
    strategyAttrs := azionapi.NewDefaultDeploymentStrategyAttrsRequest(
        plan.Deployment.Strategy.Attributes.Application.ValueInt64(),
    )

    // Handle optional firewall field
    if !plan.Deployment.Strategy.Attributes.Firewall.IsNull() && !plan.Deployment.Strategy.Attributes.Firewall.IsUnknown() {
        strategyAttrs.SetFirewall(plan.Deployment.Strategy.Attributes.Firewall.ValueInt64())
    }

    // Handle optional custom_page field
    if !plan.Deployment.Strategy.Attributes.CustomPage.IsNull() && !plan.Deployment.Strategy.Attributes.CustomPage.IsUnknown() {
        strategyAttrs.SetCustomPage(plan.Deployment.Strategy.Attributes.CustomPage.ValueInt64())
    }

    strategy := azionapi.NewDeploymentStrategyDefaultDeploymentStrategyRequest(
        plan.Deployment.Strategy.Type.ValueString(),
        *strategyAttrs,
    )

    // Build the deployment request
    deploymentRequest := azionapi.NewWorkloadDeploymentRequest(
        plan.Deployment.Name.ValueString(),
        *strategy,
    )

    // Set optional fields
    if !plan.Deployment.Current.IsNull() && !plan.Deployment.Current.IsUnknown() {
        deploymentRequest.SetCurrent(plan.Deployment.Current.ValueBool())
    }

    if !plan.Deployment.Active.IsNull() && !plan.Deployment.Active.IsUnknown() {
        deploymentRequest.SetActive(plan.Deployment.Active.ValueBool())
    }

    // Create the deployment
    createDeployment, response, err := r.client.api.WorkloadDeploymentsAPI.
        CreateWorkloadDeployment(ctx, plan.WorkloadID.ValueInt64()).
        WorkloadDeploymentRequest(*deploymentRequest).Execute()
    // ... error handling ...
}
```

### Update Method Pattern

The resource uses `PartialUpdateWorkloadDeployment` (PATCH) for updates:

```go
func (r *workloadDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // ... get plan and state ...

    // Build the patched request
    patchedRequest := azionapi.NewPatchedWorkloadDeploymentRequest()

    if !plan.Deployment.Name.IsNull() && !plan.Deployment.Name.IsUnknown() {
        patchedRequest.SetName(plan.Deployment.Name.ValueString())
    }

    if !plan.Deployment.Current.IsNull() && !plan.Deployment.Current.IsUnknown() {
        patchedRequest.SetCurrent(plan.Deployment.Current.ValueBool())
    }

    if !plan.Deployment.Active.IsNull() && !plan.Deployment.Active.IsUnknown() {
        patchedRequest.SetActive(plan.Deployment.Active.ValueBool())
    }

    // Build strategy if provided
    if plan.Deployment.Strategy != nil {
        strategyAttrs := azionapi.NewDefaultDeploymentStrategyAttrsRequest(
            plan.Deployment.Strategy.Attributes.Application.ValueInt64(),
        )
        // ... set optional fields ...
        strategy := azionapi.NewDeploymentStrategyDefaultDeploymentStrategyRequest(
            plan.Deployment.Strategy.Type.ValueString(),
            *strategyAttrs,
        )
        patchedRequest.SetStrategy(*strategy)
    }

    updateResponse, response, err := r.client.api.WorkloadDeploymentsAPI.
        PartialUpdateWorkloadDeployment(ctx, deploymentID, plan.WorkloadID.ValueInt64()).
        PatchedWorkloadDeploymentRequest(*patchedRequest).Execute()
    // ... error handling ...
}
```

### Import State Method Pattern

```go
func (r *workloadDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Import format: "workloadID/deploymentID"
    parts := strings.Split(req.ID, "/")
    if len(parts) != 2 {
        resp.Diagnostics.AddError(
            "Invalid import format",
            "Expected format: workloadID/deploymentID",
        )
        return
    }

    workloadID, err := strconv.ParseInt(parts[0], 10, 64)
    // ... error handling ...

    deploymentID, err := strconv.ParseInt(parts[1], 10, 64)
    // ... error handling ...

    // Read the deployment to populate state
    deploymentResponse, response, err := r.client.api.WorkloadDeploymentsAPI.
        RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
    // ... populate state ...
}
```

### Resource Schema Definition

```go
func (r *workloadDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Resource for managing Azion Workload Deployments.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed:    true,
                Description: "Identifier of the resource (workloadID/deploymentID format).",
            },
            "workload_id": schema.Int64Attribute{
                Description: "The workload identifier.",
                Required:    true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "deployment": schema.SingleNestedAttribute{
                Required:    true,
                Description: "The deployment configuration.",
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The deployment identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the deployment.",
                        Required:    true,
                    },
                    "current": schema.BoolAttribute{
                        Description: "Whether this is the current deployment.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Status of the deployment.",
                        Optional:    true,
                        Computed:    true,
                    },
                    "strategy": schema.SingleNestedAttribute{
                        Description: "Deployment strategy configuration.",
                        Required:    true,
                        Attributes: map[string]schema.Attribute{
                            "type": schema.StringAttribute{
                                Description: "Type of deployment strategy.",
                                Required:    true,
                            },
                            "attributes": schema.SingleNestedAttribute{
                                Description: "Strategy attributes.",
                                Required:    true,
                                Attributes: map[string]schema.Attribute{
                                    "application": schema.Int64Attribute{
                                        Description: "Application ID for the deployment.",
                                        Required:    true,
                                    },
                                    "firewall": schema.Int64Attribute{
                                        Description: "Firewall ID for the deployment.",
                                        Optional:    true,
                                    },
                                    "custom_page": schema.Int64Attribute{
                                        Description: "Custom page ID for the deployment.",
                                        Optional:    true,
                                    },
                                },
                            },
                        },
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "The last editor of the deployment.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the deployment.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}
```

---

## Resource Registration

The resource must be registered in `internal/provider.go`:

```go
func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... other resources
        NewWorkloadDeploymentResource,
    }
}
```

---

## Documentation Files

| File | Purpose |
|------|---------|
| `docs/data-sources/workload_deployment.md` | Singular data source documentation |
| `docs/data-sources/workload_deployments.md` | Plural data source documentation |
| `docs/resources/workload_deployment.md` | Resource documentation |
| `examples/data-sources/azion_workload_deployment/data-source.tf` | Singular data source example |
| `examples/data-sources/azion_workload_deployments/data-source.tf` | Plural data source example |
| `examples/resources/azion_workload_deployment/resource.tf` | Resource example |
| `examples/resources/azion_workload_deployment/import.sh` | Import example |

---

## Summary Checklist

When implementing Workload Deployment data sources and resources:

1. **Use V4 SDK**: Import `azion-api` (not `edge-api`)
2. **Avoid "edge" prefix**: Use `workloadID`, `deploymentID` naming
3. **Correct parameter order**: `deploymentID` before `workloadID` in RetrieveWorkloadDeployment
4. **Handle nullable fields**: Check `IsSet()` then `Get()` for optional strategy attributes
5. **Handle 429 errors**: Use `utils.RetryOn429`
6. **Close response body**: Use `defer response.Body.Close()` after retry
7. **Register in provider.go**: Add data sources to `DataSources()` and resources to `Resources()`
8. **Create documentation**: Create docs and examples files
9. **Run linters**: Execute `golangci-lint run --config .golintci.yml ./internal/...`
10. **Resource ID format**: Use `workloadID/deploymentID` composite ID for import support
