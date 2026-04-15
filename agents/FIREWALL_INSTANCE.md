# Firewall Function Instance - Code Generation Guide

This document provides specific guidance for implementing Firewall Function Instance resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Naming Convention](#naming-convention)
3. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
4. [Resource Implementation](#resource-implementation)
   - [Resource Model Structs](#resource-model-structs)
   - [Create Method](#create-method)
   - [Read Method](#read-method)
   - [Update Method](#update-method)
   - [Delete Method](#delete-method)
   - [ImportState Method](#importstate-method)
5. [Schema Definition Patterns](#schema-definition-patterns)
6. [Common Issues](#common-issues)

---

## SDK Selection

Firewall Function Instance uses the **V4 SDK (`azion-api`)**:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Firewall Function Instance (Singular Data Source) | `azion-api` (v4) | `api.FirewallsFunctionAPI` | `https://api.azion.com/v4` |
| Firewall Function Instance (Plural Data Source) | `azion-api` (v4) | `api.FirewallsFunctionAPI` | `https://api.azion.com/v4` |
| Firewall Function Instance (Resource) | `azion-api` (v4) | `api.FirewallsFunctionAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` for most operations |
| Create Method | `.CreateFirewallFunction(ctx, firewallId).FirewallFunctionInstanceRequest(req).Execute()` |
| Retrieve Method | `.RetrieveFirewallFunction(ctx, firewallId, functionId).Execute()` |
| Update Method | `.PartialUpdateFirewallFunction(ctx, firewallId, functionId).PatchedFirewallFunctionInstanceRequest(req).Execute()` |
| Delete Method | `.DeleteFirewallFunction(ctx, firewallId, functionId).Execute()` |
| Response Type | `Response.Data.GetId()` |
| List Method | `.ListFirewallFunction(ctx, firewallId).Page(page).PageSize(pageSize).Execute()` |

### Import Statement

```go
import sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

> **Important:** Do NOT use the legacy `edge-api` import path. The correct import is `azion-api`.

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - for Firewall Function Instance API
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // Legacy V4 SDK (azionapi-v4-go-sdk-dev/edge-api) - deprecated, do not use
    edgeConfig *edgeapi.Configuration
    edgeApi    *edgeapi.APIClient
}
```

---

## Naming Convention

### No "edge" Prefix

The "edge" prefix has been **completely removed** from all naming - both internal Go code and Terraform-facing identifiers.

| Legacy Naming (with `edge`) | New Naming (no prefix) |
|------------------------------|------------------------|
| `EdgeFirewallFunctionInstanceDataSource` | `FirewallFunctionInstanceDataSource` |
| `EdgeFirewallFunctionsInstanceDataSource` | `FirewallFunctionsInstanceDataSource` |
| `EdgeFirewallFunctionInstanceDataSourceModel` | `FirewallFunctionInstanceDataSourceModel` |
| `EdgeFirewallEdgeFunctionInstanceResults` | `FirewallFunctionInstanceData` |
| `EdgeFirewallEdgeFunctionsInstanceResults` | `FirewallFunctionInstanceResults` |
| `edge_firewall_id` (Terraform attribute) | `firewall_id` |
| `azion_edge_firewall_edge_function_instance` (type name) | `azion_firewall_function_instance` |

> **Note:** The actual implementation uses `firewall_id` as the attribute name in both schema and struct tags. The "edge" prefix has been completely removed from all naming.

### Naming Convention for Structs

Since both singular and plural data sources exist in the same package, unique struct names are required:

**Singular Data Source (single instance):**
- `FirewallFunctionInstanceDataSource` - the datasource struct
- `FirewallFunctionInstanceDataSourceModel` - the state model
- `FirewallFunctionInstanceData` - the results struct

**Plural Data Source (list of instances):**
- `FirewallFunctionsInstanceDataSource` - the datasource struct
- `FirewallFunctionsInstanceDataSourceModel` - the state model
- `FirewallFunctionInstanceResults` - the results struct for each item

**Resource:**
- `FirewallFunctionsInstanceResource` - the resource struct (plural to avoid collision)
- `FirewallFunctionInstanceResourceModel` - the state model
- `FirewallFunctionInstanceResourceData` - the data nested block

### Example: Naming Pattern

```go
// CORRECT - Naming without "edge" prefix anywhere
func dataSourceAzionFirewallFunctionInstance() datasource.DataSource {
    return &FirewallFunctionInstanceDataSource{}  // No "Edge" prefix
}

type FirewallFunctionInstanceDataSource struct {  // No "Edge" prefix
    client *apiClient
}

type FirewallFunctionInstanceDataSourceModel struct {  // No "Edge" prefix
    ID         types.Int64                  `tfsdk:"id"`
    FirewallID types.Int64                  `tfsdk:"firewall_id"`  // No "edge_" prefix
    Data       FirewallFunctionInstanceData `tfsdk:"data"`
}

func (f *FirewallFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    // No "edge_" in TypeName
    resp.TypeName = req.ProviderTypeName + "_firewall_function_instance"
}

func (f *FirewallFunctionInstanceDataSource) Read(ctx context.Context, ...) {
    var firewallID types.Int64  // No "edge" prefix in variable names
    response, resp, err := f.client.api.FirewallsFunctionAPI.RetrieveFirewallFunction(ctx, firewallID.ValueInt64(), 0).Execute()
    // ...
}

// WRONG - Legacy naming with "edge" prefix
func dataSourceAzionEdgeFirewallEdgeFunctionInstance() datasource.DataSource {
    return &EdgeFirewallEdgeFunctionInstanceDataSource{}  // Don't use "Edge" prefix
}

type EdgeFirewallEdgeFunctionInstanceDataSource struct {  // Don't use "Edge" prefix
    client *apiClient
}
```

### Terraform Resource and Data Source Names

| Type | Name |
|------|------|
| Singular Data Source | `azion_firewall_function_instance` |
| Plural Data Source | `azion_firewall_functions_instance` |
| Resource | `azion_firewall_functions_instance` |

> **Note:** The Terraform resource names do not include the `edge_` prefix. The internal Go code naming also follows this convention - no "edge" prefix in struct names, variable names, or attribute names.

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single firewall function instance.

#### File: `internal/data_source_edge_firewall_edge_function_instance.go`

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

func dataSourceAzionFirewallFunctionInstance() datasource.DataSource {
	return &FirewallFunctionInstanceDataSource{}
}

type FirewallFunctionInstanceDataSource struct {
	client *apiClient
}

type FirewallFunctionInstanceDataSourceModel struct {
	ID         types.Int64                  `tfsdk:"id"`
	FirewallID types.Int64                  `tfsdk:"firewall_id"`
	Data       FirewallFunctionInstanceData `tfsdk:"data"`
}

type FirewallFunctionInstanceData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (f *FirewallFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	f.client = req.ProviderData.(*apiClient)
}

func (f *FirewallFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_function_instance"
}

func (f *FirewallFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "ID of the firewall function instance to retrieve.",
				Required:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Identifier of the Firewall",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the firewall function instance.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the firewall function instance.",
						Computed:    true,
					},
					"args": schema.StringAttribute{
						Description: "Arguments for the function instance.",
						Computed:    true,
					},
					"function": schema.Int64Attribute{
						Description: "ID of the Function for Firewall you wish to configure.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the function instance is active.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the firewall function instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the firewall function instance.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the firewall function instance.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (f *FirewallFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var firewallID types.Int64
	var functionInstanceID types.Int64

	diagsFirewallId := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallId...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsFunctionInstanceId := req.Config.GetAttribute(ctx, path.Root("id"), &functionInstanceID)
	resp.Diagnostics.Append(diagsFunctionInstanceId...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallFunctionInstanceResponse, response, err := f.client.api.FirewallsFunctionAPI.
		RetrieveFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			firewallFunctionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return f.client.api.FirewallsFunctionAPI.
					RetrieveFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).Execute()
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(firewallFunctionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	data := FirewallFunctionInstanceData{
		ID:           types.Int64Value(firewallFunctionInstanceResponse.Data.GetId()),
		Name:         types.StringValue(firewallFunctionInstanceResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(firewallFunctionInstanceResponse.Data.GetFunction()),
		Active:       types.BoolValue(firewallFunctionInstanceResponse.Data.GetActive()),
		LastEditor:   types.StringValue(firewallFunctionInstanceResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(firewallFunctionInstanceResponse.Data.GetLastModified().Format(time.RFC3339)),
		CreatedAt:    types.StringValue(firewallFunctionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
	}

	state := FirewallFunctionInstanceDataSourceModel{
		ID:         functionInstanceID,
		FirewallID: firewallID,
		Data:       data,
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
```

---

### Plural Data Source (List Multiple Resources)

For listing multiple firewall function instances with pagination.

#### File: `internal/data_source_edge_firewall_edge_functions_instance.go`

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

func dataSourceAzionFirewallFunctionsInstance() datasource.DataSource {
	return &FirewallFunctionsInstanceDataSource{}
}

type FirewallFunctionsInstanceDataSource struct {
	client *apiClient
}

type FirewallFunctionsInstanceDataSourceModel struct {
	ID         types.Int64                       `tfsdk:"id"`
	FirewallID types.Int64                       `tfsdk:"firewall_id"`
	Counter    types.Int64                       `tfsdk:"counter"`
	Page       types.Int64                       `tfsdk:"page"`
	PageSize   types.Int64                       `tfsdk:"page_size"`
	TotalPages types.Int64                       `tfsdk:"total_pages"`
	Results    []FirewallFunctionInstanceResults `tfsdk:"results"`
}

type FirewallFunctionInstanceResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (f *FirewallFunctionsInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	f.client = req.ProviderData.(*apiClient)
}

func (f *FirewallFunctionsInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_functions_instance"
}

func (f *FirewallFunctionsInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Firewall",
				Required:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of firewall function instances.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of firewall function instances.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of firewall function instances.",
				Optional:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "ID of the firewall function instance.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the firewall function instance.",
							Computed:    true,
						},
						"args": schema.StringAttribute{
							Description: "Arguments for the function instance.",
							Computed:    true,
						},
						"function": schema.Int64Attribute{
							Description: "ID of the Function for Firewall you wish to configure.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the function instance is active.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the firewall function instance.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the firewall function instance.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the firewall function instance.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (f *FirewallFunctionsInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var firewallID types.Int64

	diagsFirewallId := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallId...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// Set default values
	if page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	firewallFunctionInstancesResponse, response, err := f.client.api.FirewallsFunctionAPI.
		ListFirewallFunction(ctx, firewallID.ValueInt64()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64()).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			firewallFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedFirewallFunctionInstanceList, *http.Response, error) {
				return f.client.api.FirewallsFunctionAPI.
					ListFirewallFunction(ctx, firewallID.ValueInt64()).Page(page.ValueInt64()).
					PageSize(pageSize.ValueInt64()).Execute()
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

	var functionInstancesResults []FirewallFunctionInstanceResults
	for _, result := range firewallFunctionInstancesResponse.GetResults() {
		jsonArgsStr, err := utils.ConvertInterfaceToString(result.GetArgs())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
		}

		functionInstance := FirewallFunctionInstanceResults{
			ID:           types.Int64Value(result.GetId()),
			Name:         types.StringValue(result.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(result.GetFunction()),
			Active:       types.BoolValue(result.GetActive()),
			LastEditor:   types.StringValue(result.GetLastEditor()),
			LastModified: types.StringValue(result.GetLastModified().Format(time.RFC3339)),
			CreatedAt:    types.StringValue(result.GetCreatedAt().Format(time.RFC3339)),
		}
		functionInstancesResults = append(functionInstancesResults, functionInstance)
	}

	state := FirewallFunctionsInstanceDataSourceModel{
		ID:         firewallID,
		FirewallID: firewallID,
		Counter:    types.Int64Value(firewallFunctionInstancesResponse.GetCount()),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: types.Int64Value(firewallFunctionInstancesResponse.GetTotalPages()),
		Results:    functionInstancesResults,
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular | Plural |
|--------|----------|--------|
| **Purpose** | Read a single resource by ID | List multiple resources with pagination |
| **API Method** | `RetrieveFirewallFunction(ctx, firewallId, functionId)` | `ListFirewallFunction(ctx, firewallId).Page().PageSize()` |
| **Response Type** | `*sdk.FirewallFunctionInstanceResponse` | `*sdk.PaginatedFirewallFunctionInstanceList` |
| **Data Attribute** | `schema.SingleNestedAttribute` | `schema.ListNestedAttribute` |
| **ID Field** | `types.Int64` (Required - function instance ID) | `types.Int64` (Computed - firewall ID) |
| **Pagination** | No | Yes (`page`, `page_size`, `counter`, `total_pages`) |
| **Struct Naming** | `FirewallFunctionInstanceDataSource` | `FirewallFunctionsInstanceDataSource` (plural) |
| **Results Struct** | `FirewallFunctionInstanceData` | `FirewallFunctionInstanceResults` |
| **Created At Field** | Included in data struct | Included in results struct |

---

## Resource Implementation

### Resource Structs

The resource uses plural naming to differentiate from data sources:

```go
// FirewallFunctionsInstanceResource - the resource struct (plural to avoid collision with data sources)
type FirewallFunctionsInstanceResource struct {
	client *apiClient
}

// FirewallFunctionInstanceResourceModel - the state model
type FirewallFunctionInstanceResourceModel struct {
	State       types.String                         `tfsdk:"state"`
	Data        FirewallFunctionInstanceResourceData `tfsdk:"data"`
	ID          types.String                         `tfsdk:"id"`
	FirewallID  types.Int64                          `tfsdk:"firewall_id"`
	LastUpdated types.String                         `tfsdk:"last_updated"`
}

// FirewallFunctionInstanceResourceData - the data nested block
type FirewallFunctionInstanceResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
}
```

### Create Method

```go
func (r *FirewallFunctionsInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallFunctionInstanceResourceModel
	var firewallID types.Int64
	
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle JSON args
	var argsStr string
	if plan.Data.Args.IsNull() || plan.Data.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		argsStr = plan.Data.Args.ValueString()
		if argsStr == "" {
			argsStr = "{}"
		}
	}

	planJsonArgs, err := utils.UnmarshallJsonArgsFirewall(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "failed to unmarshal json args from plan")
		return
	}

	// Create the request object using FirewallFunctionInstanceRequest
	functionInstanceRequest := sdk.FirewallFunctionInstanceRequest{
		Name:     plan.Data.Name.ValueString(),
		Function: plan.Data.Function.ValueInt64(),
		Args:     &planJsonArgs,
		Active:   plan.Data.Active.ValueBoolPointer(),
	}

	// Call API
	functionInstanceResponse, response, err := r.client.api.FirewallsFunctionAPI.
		CreateFirewallFunction(ctx, firewallID.ValueInt64()).
		FirewallFunctionInstanceRequest(functionInstanceRequest).
		Execute()
	
	if err != nil {
		if response.StatusCode == 429 {
			// Handle retry with utils.RetryOn429...
		} else {
			// Handle other errors...
		}
		return
	}

	// Convert args back to string for state
	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "err")
		return
	}

	// Update plan with response data
	plan.Data = FirewallFunctionInstanceResourceData{
		Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
		ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
		Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
		LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
		CreatedAt:    types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
	}

	plan.State = types.StringValue(functionInstanceResponse.GetState())
	plan.ID = types.StringValue(strconv.FormatInt(functionInstanceResponse.Data.GetId(), 10))
	plan.FirewallID = firewallID
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
```

### Read Method

```go
func (r *FirewallFunctionsInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallFunctionInstanceResourceModel
	
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var firewallID int64
	var functionInstanceID int64
	
	// Handle import format: "firewallId/functionInstanceId"
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		firewallID, _ = strconv.ParseInt(valueFromCmd[0], 10, 64)
		functionInstanceID, _ = strconv.ParseInt(valueFromCmd[1], 10, 64)
	} else {
		firewallID = state.FirewallID.ValueInt64()
		functionInstanceID = state.Data.ID.ValueInt64()
	}

	if functionInstanceID == 0 {
		resp.Diagnostics.AddError(
			"Function Instance id error ",
			"should not be null or empty",
		)
		return
	}

	functionInstanceResponse, response, err := r.client.api.FirewallsFunctionAPI.
		RetrieveFirewallFunction(ctx, firewallID, functionInstanceID).
		Execute()
	
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			// Handle retry with utils.RetryOn429...
		} else {
			// Handle other errors...
		}
		return
	}

	// Convert args back to string
	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "err")
	}

	// Build read state
	readState := FirewallFunctionInstanceResourceModel{
		FirewallID: types.Int64Value(firewallID),
		State:      types.StringValue("executed"),
		ID:         types.StringValue(strconv.FormatInt(functionInstanceResponse.Data.GetId(), 10)),
		Data: FirewallFunctionInstanceResourceData{
			ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
			LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
			LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
			Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
			Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
			CreatedAt:    types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		},
	}

	diags = resp.State.Set(ctx, &readState)
	resp.Diagnostics.Append(diags...)
}
```

### Update Method

Uses PATCH for partial updates:

```go
func (r *FirewallFunctionsInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallFunctionInstanceResourceModel
	var firewallID types.Int64
	var functionInstanceID types.Int64
	
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallFunctionInstanceResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always use the function instance ID from state (it's a computed field)
	functionInstanceID = state.Data.ID

	// Always use the firewall ID from state (it's required and shouldn't change)
	firewallID = state.FirewallID

	// Handle JSON args
	var argsStr string
	if plan.Data.Args.IsNull() || plan.Data.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		argsStr = plan.Data.Args.ValueString()
		if argsStr == "" {
			argsStr = "{}"
		}
	}

	requestJsonArgsStr, err := utils.UnmarshallJsonArgsFirewall(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "failed to unmarshal json args from plan")
		return
	}

	// Build patch request using PatchedFirewallFunctionInstanceRequest
	patchRequest := sdk.PatchedFirewallFunctionInstanceRequest{
		Name:     plan.Data.Name.ValueStringPointer(),
		Function: plan.Data.Function.ValueInt64Pointer(),
		Args:     &requestJsonArgsStr,
		Active:   plan.Data.Active.ValueBoolPointer(),
	}

	updateResponse, response, err := r.client.api.FirewallsFunctionAPI.
		PartialUpdateFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).
		PatchedFirewallFunctionInstanceRequest(patchRequest).
		Execute()
	
	if err != nil {
		if response.StatusCode == 429 {
			// Handle retry with utils.RetryOn429...
		} else {
			// Handle other errors...
		}
		return
	}

	// Convert args back to string
	jsonArgsStr, err := utils.ConvertInterfaceToString(updateResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "err")
		return
	}

	// Update plan with response data
	plan.Data = FirewallFunctionInstanceResourceData{
		Function:     types.Int64Value(updateResponse.Data.GetFunction()),
		Name:         types.StringValue(updateResponse.Data.GetName()),
		LastEditor:   types.StringValue(updateResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(updateResponse.Data.GetLastModified().Format(time.RFC850)),
		Args:         types.StringValue(jsonArgsStr),
		ID:           types.Int64Value(updateResponse.Data.GetId()),
		Active:       types.BoolValue(updateResponse.Data.GetActive()),
		CreatedAt:    types.StringValue(updateResponse.Data.GetCreatedAt().Format(time.RFC3339)),
	}

	plan.State = types.StringValue(updateResponse.GetState())
	plan.ID = types.StringValue(strconv.FormatInt(updateResponse.Data.GetId(), 10))
	plan.FirewallID = firewallID
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
```

### Delete Method

```go
func (r *FirewallFunctionsInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallFunctionInstanceResourceModel
	
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Data.ID.IsNull() {
		resp.Diagnostics.AddError("Function Instance id error ", "is not null")
		return
	}

	if state.FirewallID.IsNull() {
		resp.Diagnostics.AddError("Firewall ID error ", "is not null")
		return
	}

	_, response, err := r.client.api.FirewallsFunctionAPI.
		DeleteFirewallFunction(ctx, state.FirewallID.ValueInt64(), state.Data.ID.ValueInt64()).
		Execute()
	
	if err != nil {
		if response.StatusCode == 429 {
			// Handle retry with utils.RetryOn429...
		} else {
			// Handle other errors...
		}
	}
}
```

### ImportState Method

The current implementation uses passthrough ID:

```go
func (r *FirewallFunctionsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

The Read method handles parsing the imported ID in the format `firewallId/functionInstanceId`.

---

## Schema Definition Patterns

### Required Attributes

```go
"firewall_id": schema.Int64Attribute{
    Description: "Identifier of the Firewall",
    Required:    true,
},
"name": schema.StringAttribute{
    Description: "Name of the firewall function instance",
    Required:    true,
},
"function": schema.Int64Attribute{
    Description: "ID of the Function for Firewall",
    Required:    true,
},
```

### Optional Attributes

```go
"active": schema.BoolAttribute{
    Description: "Whether the function instance is active",
    Optional:    true,
    Computed:    true,
    Default:     booldefault.StaticBool(true),
},
"args": schema.StringAttribute{
    Description: "JSON arguments for the function instance",
    Optional:    true,
    Computed:    true,
    Default:     stringdefault.StaticString("{}"),
},
```

### Computed Attributes

```go
"id": schema.Int64Attribute{
    Description: "ID of the firewall function instance",
    Computed:    true,
},
"last_editor": schema.StringAttribute{
    Description: "Last editor of the firewall function instance",
    Computed:    true,
},
"last_modified": schema.StringAttribute{
    Description: "Last modified timestamp",
    Computed:    true,
},
"created_at": schema.StringAttribute{
    Description: "The creation timestamp of the firewall function instance",
    Computed:    true,
},
```

---

## Common Issues

### 1. Using Wrong SDK Import

**Wrong:**
```go
import edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"

// Using edgeApi client
e.client.edgeApi.FirewallsFunctionAPI...
```

**Correct:**
```go
import sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"

// Using api client
f.client.api.FirewallsFunctionAPI...
```

### 2. Using "edge" Prefix in Internal Naming

**Wrong:**
```go
type EdgeFirewallFunctionInstanceDataSource struct { ... }
type EdgeFirewallFunctionInstanceResults struct { ... }
```

**Correct:**
```go
type FirewallFunctionInstanceDataSource struct { ... }
type FirewallFunctionInstanceData struct { ... }
```

### 3. Handling JSON Args

The `args` field is a JSON string that needs conversion:

```go
// From API to Terraform (Read)
jsonArgsStr, err := utils.ConvertInterfaceToString(response.Data.GetArgs())

// From Terraform to API (Create/Update)
argsInterface, err := utils.UnmarshallJsonArgsFirewall(plan.Data.Args.ValueString())
```

### 4. Active Field Defaults to true

The `active` field has a default value of `true` (not `false`):

```go
"active": schema.BoolAttribute{
    Description: "Whether the function instance is active.",
    Optional:    true,
    Computed:    true,
    Default:     booldefault.StaticBool(true),
},
```

### 4. Handling Rate Limiting (429)

Always implement retry logic for rate limiting:

```go
if response.StatusCode == 429 {
    response, httpResponse, err = utils.RetryOn429(func() (*sdk.ResponseType, *http.Response, error) {
        return client.API.Method(ctx, params).Execute()
    }, 5)  // Max 5 retries
    
    if response != nil {
        defer response.Body.Close()
    }
}
```

### 5. Struct Name Collisions

Since singular and plural data sources are in the same package, use unique struct names:

- Singular: `FirewallFunctionInstanceDataSource`, `FirewallFunctionInstanceData`
- Plural: `FirewallFunctionsInstanceDataSource`, `FirewallFunctionInstanceResults`

### 6. Singular Data Source ID Field

The singular data source's `id` attribute is **Required** (not Computed). Both `id` (function instance ID) and `firewall_id` must be provided to read a specific function instance.

```go
"id": schema.Int64Attribute{
    Description: "ID of the firewall function instance to retrieve.",
    Required:    true,
},
```

---

## Summary Checklist

When implementing firewall function instance data sources:

1. **Use correct SDK**: `azion-api` (not `edge-api`)
2. **Client access**: `f.client.api.FirewallsFunctionAPI`
3. **Naming**: No "edge" prefix in Terraform resource names or internal struct/function names
4. **Unique struct names**: Differentiate singular and plural with `Instance` vs `Instances` or `Data` vs `Results`
5. **Handle 429 errors**: Use `utils.RetryOn429`
6. **Handle JSON args**: Use `utils.ConvertInterfaceToString` and `utils.UnmarshallJsonArgsFirewall`
7. **Time formatting**: Use `time.RFC3339` for created_at, `time.RFC850` for last_modified and last_updated
8. **Register in provider.go**: Add to `DataSources()` and `Resources()` functions
9. **Include created_at**: All data structs should include the `created_at` field
