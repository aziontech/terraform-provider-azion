# Firewall - Code Generation Guide

This document provides specific guidance for implementing Firewall resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Naming Convention](#naming-convention)
3. [Firewall Main Settings](#firewall-main-settings)
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
7. [Common Issues](#common-issues)

---

## SDK Selection

Firewall uses the **V4 SDK (`azion-api`)** for Main Settings:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Firewall Main Setting (Data Source) | `azion-api` (v4) | `api.FirewallsAPI` | `https://api.azion.com/v4` |
| Firewall Main Settings (Plural Data Source) | `azion-api` (v4) | `api.FirewallsAPI` | `https://api.azion.com/v4` |
| Firewall Main Setting (Resource) | `azion-api` (v4) | `api.FirewallsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` for most operations |
| Create Method | `.CreateFirewall(ctx).FirewallRequest(req).Execute()` |
| Retrieve Method | `.RetrieveFirewall(ctx, id).Execute()` |
| Update Method | `.PartialUpdateFirewall(ctx, id).PatchedFirewallRequest(req).Execute()` |
| Delete Method | `.DeleteFirewall(ctx, id).Execute()` |
| Response Type | `Response.Data.GetId()` |
| List Method | `.ListFirewalls(ctx).Page(page).PageSize(pageSize).Execute()` |

### Import Statement

```go
import sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

> **Important:** Do NOT use the legacy `edge-api` import path. The correct import is `azion-api`.

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev/azion-api) - for Firewall API
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // Legacy V4 SDK (azionapi-v4-go-sdk-dev/edge-api) - deprecated, do not use
    edgeConfig *edgeapi.Configuration
    edgeApi    *edgeapi.APIClient
    
    // Legacy SDKs (azionapi-go-sdk) - deprecated
    edgeFirewallApi *edgefirewall.APIClient
    // ... more SDK clients
}
```

---

## Naming Convention

### No "edge" Prefix in V4 SDK

When implementing resources using the V4 API, the "edge" prefix is **removed** from all naming:

| Legacy Naming (with `edge`) | V4 SDK Naming (no prefix) |
|------------------------------|---------------------------|
| `EdgeFirewallDataSource` | `FirewallDataSource` |
| `EdgeFirewallsDataSource` | `FirewallsDataSource` |
| `EdgeFirewallResource` | `FirewallResource` |
| `EdgeFirewallResourceModel` | `FirewallResourceModel` |
| `EdgeFirewallModules` | `FirewallModules` |
| `EdgeFirewallResults` | `FirewallResults` |
| `edge_firewall_id` (Terraform attribute) | `firewall_id` |
| `azion_edge_firewall_main_setting` (type name) | `azion_firewall_main_setting` |
| `e.client.edgeApi` | `f.client.api` |
| `dataSourceAzionEdgeFirewall` | `dataSourceAzionFirewall` |
| `EdgeFirewallResource` | `FirewallMainSettingResource` |

### Example: Naming Pattern

```go
// CORRECT - V4 SDK naming without "edge" prefix
func dataSourceAzionFirewall() datasource.DataSource {
    return &FirewallDataSource{}
}

type FirewallDataSource struct {
    client *apiClient
}

type FirewallDataSourceModel struct {
    ID         types.String    `tfsdk:"id"`
    FirewallID types.Int64     `tfsdk:"firewall_id"`  // No "edge_" prefix
    Data       FirewallResults `tfsdk:"data"`
}

func (f *FirewallDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_firewall_main_setting"  // No "edge_" prefix
}

func (f *FirewallDataSource) Read(ctx context.Context, ...) {
    var getFirewallID types.Int64
    firewallResponse, response, err := f.client.api.FirewallsAPI.RetrieveFirewall(ctx, getFirewallID.ValueInt64()).Execute()
    // ...
}

// WRONG - Legacy naming with "edge" prefix
func dataSourceAzionEdgeFirewall() datasource.DataSource {  // Don't use "Edge" prefix
    return &EdgeFirewallDataSource{}  // Don't use "Edge" prefix
}
```

---

## Firewall Main Settings

### V4 SDK Pattern

The main settings use the V4 SDK:

```go
// Singular Data Source - Read by ID
f.client.api.FirewallsAPI.RetrieveFirewall(ctx, idInt64).Execute()

// Plural Data Source - List with pagination
f.client.api.FirewallsAPI.ListFirewalls(ctx).Page(page).PageSize(pageSize).Execute()

// Resource - Create
r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(firewall).Execute()

// Resource - Read
r.client.api.FirewallsAPI.RetrieveFirewall(ctx, idInt64).Execute()

// Resource - Update (PATCH)
r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, idInt64).PatchedFirewallRequest(firewall).Execute()

// Resource - Delete
r.client.api.FirewallsAPI.DeleteFirewall(ctx, idInt64).Execute()
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single firewall by its identifier:

#### File: `internal/data_source_edge_firewall_main_setting.go`

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

func dataSourceAzionFirewall() datasource.DataSource {
	return &FirewallDataSource{}
}

type FirewallDataSource struct {
	client *apiClient
}

type FirewallDataSourceModel struct {
	ID         types.String    `tfsdk:"id"`
	FirewallID types.Int64     `tfsdk:"firewall_id"`
	Data       FirewallResults `tfsdk:"data"`
}

type FirewallModules struct {
	DdosProtection    *DdosProtectionModule    `tfsdk:"ddos_protection"`
	Functions         *FunctionsModule         `tfsdk:"functions"`
	NetworkProtection *NetworkProtectionModule `tfsdk:"network_protection"`
	WAF               *WAFModule               `tfsdk:"waf"`
}

type DdosProtectionModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type FunctionsModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type NetworkProtectionModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type WAFModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type FirewallResults struct {
	ID             types.Int64     `tfsdk:"id"`
	Name           types.String    `tfsdk:"name"`
	Modules        FirewallModules `tfsdk:"modules"`
	Debug          types.Bool      `tfsdk:"debug"`
	Active         types.Bool      `tfsdk:"active"`
	LastEditor     types.String    `tfsdk:"last_editor"`
	LastModified   types.String    `tfsdk:"last_modified"`
	ProductVersion types.String    `tfsdk:"product_version"`
	CreatedAt      types.String    `tfsdk:"created_at"`
}

func (f *FirewallDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	f.client = req.ProviderData.(*apiClient)
}

func (f *FirewallDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_main_setting"
}

func (f *FirewallDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall identifier.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					// ... schema attributes
				},
			},
		},
	}
}

func (f *FirewallDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getFirewallID types.Int64
	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &getFirewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallResponse, response, err := f.client.api.FirewallsAPI.RetrieveFirewall(ctx, getFirewallID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return f.client.api.FirewallsAPI.RetrieveFirewall(ctx, getFirewallID.ValueInt64()).Execute() //nolint
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

	mods := firewallResponse.Data.GetModules()
	ddosProtection := mods.GetDdosProtection()
	functions := mods.GetFunctions()
	networkProtection := mods.GetNetworkProtection()
	waf := mods.GetWaf()

	modules := FirewallModules{
		DdosProtection: &DdosProtectionModule{
			Enabled: types.BoolValue(ddosProtection.GetEnabled()),
		},
		Functions: &FunctionsModule{
			Enabled: types.BoolValue(functions.GetEnabled()),
		},
		NetworkProtection: &NetworkProtectionModule{
			Enabled: types.BoolValue(networkProtection.GetEnabled()),
		},
		WAF: &WAFModule{
			Enabled: types.BoolValue(waf.GetEnabled()),
		},
	}

	firewallResults := FirewallResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		Modules:        modules,
		Debug:          types.BoolValue(firewallResponse.Data.GetDebug()),
		Active:         types.BoolValue(firewallResponse.Data.GetActive()),
		LastEditor:     types.StringValue(firewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(firewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		ProductVersion: types.StringValue(firewallResponse.Data.GetProductVersion()),
		CreatedAt:      types.StringValue(firewallResponse.Data.GetCreatedAt().Format(time.RFC3339)),
	}

	firewallState := FirewallDataSourceModel{
		FirewallID: getFirewallID,
		Data:       firewallResults,
	}

	firewallState.ID = types.StringValue("Get Firewall by ID")
	diags := resp.State.Set(ctx, &firewallState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
```

---

### Plural Data Source (List Multiple Resources)

For listing multiple firewalls with pagination support:

#### File: `internal/data_source_edge_firewall_main_settings.go`

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

func dataSourceAzionFirewalls() datasource.DataSource {
	return &FirewallsDataSource{}
}

type FirewallsDataSource struct {
	client *apiClient
}

type FirewallsDataSourceModel struct {
	Page     types.Int64        `tfsdk:"page"`
	PageSize types.Int64        `tfsdk:"page_size"`
	Counter  types.Int64        `tfsdk:"counter"`
	Results  []FirewallsResults `tfsdk:"results"`
}

type FirewallsResults struct {
	ID             types.Int64     `tfsdk:"id"`
	Name           types.String    `tfsdk:"name"`
	Modules        FirewallModules `tfsdk:"modules"`
	Debug          types.Bool      `tfsdk:"debug"`
	Active         types.Bool      `tfsdk:"active"`
	LastEditor     types.String    `tfsdk:"last_editor"`
	LastModified   types.String    `tfsdk:"last_modified"`
	ProductVersion types.String    `tfsdk:"product_version"`
}

func (f *FirewallsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_main_settings"
}

func (f *FirewallsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var Page types.Int64
	var PageSize types.Int64

	// ... pagination handling

	firewallsResponse, response, err := f.client.api.FirewallsAPI.ListFirewalls(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallsResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedFirewallList, *http.Response, error) {
				return f.client.api.FirewallsAPI.ListFirewalls(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
			}, 5) // Maximum 5 retries
			// ... error handling
		}
	}

	var firewallsResults []FirewallsResults
	for _, results := range firewallsResponse.Results {
		// ... transform results
		firewallResult := FirewallsResults{
			ID:             types.Int64Value(results.GetId()),
			Name:           types.StringValue(results.GetName()),
			// ... other fields
		}
		firewallsResults = append(firewallsResults, firewallResult)
	}

	firewallsState := FirewallsDataSourceModel{
		Page:     Page,
		PageSize: PageSize,
		Counter:  types.Int64Value(firewallsResponse.GetCount()),
		Results:  firewallsResults,
	}

	diags := resp.State.Set(ctx, &firewallsState)
	resp.Diagnostics.Append(diags...)
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`_main_setting`) | Plural (`_main_settings`) |
|--------|---------------------------|---------------------------|
| **Purpose** | Read a single firewall by ID | List multiple firewalls |
| **ID Parameter** | `firewall_id` (Required, Int64) | None (uses pagination) |
| **Pagination** | Not applicable | `page`, `page_size`, `counter` |
| **Results Type** | `SingleNestedAttribute` | `ListNestedAttribute` |
| **API Method** | `RetrieveFirewall(ctx, id)` | `ListFirewalls(ctx).Page().PageSize()` |
| **Response Type** | `*sdk.FirewallResponse` | `*sdk.PaginatedFirewallList` |
| **Data Access** | `response.Data` | `response.Results` (array) |
| **Struct Naming** | `FirewallDataSource`, `FirewallResults` | `FirewallsDataSource`, `FirewallsResults` |
| **Terraform Type** | `azion_firewall_main_setting` | `azion_firewall_main_settings` |

---

## Resource Implementation

### Resource Model Structs

```go
// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &firewallResource{}
	_ resource.ResourceWithConfigure   = &firewallResource{}
	_ resource.ResourceWithImportState = &firewallResource{}
)

func FirewallMainSettingResource() resource.Resource {
	return &firewallResource{}
}

type firewallResource struct {
	client *apiClient
}

type FirewallResourceModel struct {
	Firewall    *FirewallResourceResults `tfsdk:"data"`
	ID          types.String             `tfsdk:"id"`
	LastUpdated types.String             `tfsdk:"last_updated"`
}

type FirewallResourceModules struct {
	DdosProtection    *DdosProtectionModule    `tfsdk:"ddos_protection"`
	Functions         *FunctionsModule         `tfsdk:"functions"`
	NetworkProtection *NetworkProtectionModule `tfsdk:"network_protection"`
	WAF               *WAFModule               `tfsdk:"waf"`
}

type FirewallResourceResults struct {
	ID             types.Int64              `tfsdk:"id"`
	Name           types.String             `tfsdk:"name"`
	Modules        *FirewallResourceModules `tfsdk:"modules"`
	Debug          types.Bool               `tfsdk:"debug"`
	Active         types.Bool               `tfsdk:"active"`
	LastEditor     types.String             `tfsdk:"last_editor"`
	LastModified   types.String             `tfsdk:"last_modified"`
	CreatedAt      types.String             `tfsdk:"created_at"`
	ProductVersion types.String             `tfsdk:"product_version"`
}
```

### Create Method

```go
func (r *firewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build modules request
	modules := sdk.FirewallModulesRequest{}
	if plan.Firewall.Modules != nil {
		if plan.Firewall.Modules.Functions != nil && !plan.Firewall.Modules.Functions.Enabled.IsNull() {
			modules.Functions = &sdk.FirewallModuleRequest{
				Enabled: plan.Firewall.Modules.Functions.Enabled.ValueBoolPointer(),
			}
		}
		if plan.Firewall.Modules.NetworkProtection != nil && !plan.Firewall.Modules.NetworkProtection.Enabled.IsNull() {
			modules.NetworkProtection = &sdk.FirewallModuleRequest{
				Enabled: plan.Firewall.Modules.NetworkProtection.Enabled.ValueBoolPointer(),
			}
		}
		if plan.Firewall.Modules.WAF != nil && !plan.Firewall.Modules.WAF.Enabled.IsNull() {
			modules.Waf = &sdk.FirewallModuleRequest{
				Enabled: plan.Firewall.Modules.WAF.Enabled.ValueBoolPointer(),
			}
		}
	}

	// Build firewall request
	firewallRequest := sdk.FirewallRequest{
		Name:    plan.Firewall.Name.ValueString(),
		Active:  plan.Firewall.Active.ValueBoolPointer(),
		Debug:   plan.Firewall.Debug.ValueBoolPointer(),
		Modules: &modules,
	}

	// Execute API call
	firewallResponse, response, err := r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(firewallRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(firewallRequest).Execute()
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

	// Transform response to state model
	plan.Firewall = &FirewallResourceResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		// ... other fields
	}
	plan.ID = types.StringValue(strconv.FormatInt(firewallResponse.Data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
```

### Read Method

```go
func (r *firewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get firewall ID
	var firewallID int64
	if state.ID.IsNull() {
		firewallID = state.Firewall.ID.ValueInt64()
	} else {
		var err error
		firewallID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse firewall ID", err.Error())
			return
		}
	}

	// Execute API call
	firewallResponse, response, err := r.client.api.FirewallsAPI.RetrieveFirewall(ctx, firewallID).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		// ... handle 429 and other errors
	}

	// Update state with response data
	state.Firewall = &FirewallResourceResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		// ... other fields
	}
	state.ID = types.StringValue(strconv.FormatInt(firewallID, 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
```

### Update Method

Uses PATCH for partial updates:

```go
func (r *firewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get firewall ID
	var firewallID int64
	if state.ID.IsNull() {
		firewallID = state.Firewall.ID.ValueInt64()
	} else {
		var err error
		firewallID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse firewall ID", err.Error())
			return
		}
	}

	// Build patched request (use pointers for partial update)
	firewallRequest := sdk.PatchedFirewallRequest{
		Name:   plan.Firewall.Name.ValueStringPointer(),
		Active: plan.Firewall.Active.ValueBoolPointer(),
		Debug:  plan.Firewall.Debug.ValueBoolPointer(),
	}

	// Build modules request
	modules := sdk.FirewallModulesRequest{}
	if plan.Firewall.Modules != nil {
		if plan.Firewall.Modules.Functions != nil && !plan.Firewall.Modules.Functions.Enabled.IsNull() {
			modules.Functions = &sdk.FirewallModuleRequest{
				Enabled: plan.Firewall.Modules.Functions.Enabled.ValueBoolPointer(),
			}
		}
		// ... other modules
	}
	firewallRequest.Modules = &modules

	// Execute API call with PartialUpdateFirewall (PATCH)
	firewallResponse, response, err := r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, firewallID).PatchedFirewallRequest(firewallRequest).Execute()
	if err != nil {
		// ... handle errors with RetryOn429
	}

	// Update plan with response
	plan.Firewall = &FirewallResourceResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		// ... other fields
	}
	plan.ID = types.StringValue(strconv.FormatInt(firewallResponse.Data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}
```

### Delete Method

```go
func (r *firewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var firewallID int64
	if state.ID.IsNull() {
		firewallID = state.Firewall.ID.ValueInt64()
	} else {
		var err error
		firewallID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse firewall ID", err.Error())
			return
		}
	}

	_, response, err := r.client.api.FirewallsAPI.DeleteFirewall(ctx, firewallID).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.DeleteResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.DeleteFirewall(ctx, firewallID).Execute()
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
func (r *firewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

---

## Schema Definition Patterns

### Modules Schema Pattern

Firewall modules follow a consistent nested structure:

```go
"modules": schema.SingleNestedAttribute{
    Description: "Modules configuration for the firewall.",
    Computed:    true,
    Attributes: map[string]schema.Attribute{
        "ddos_protection": schema.SingleNestedAttribute{
            Description: "DDoS protection module configuration.",
            Computed:    true,
            Attributes: map[string]schema.Attribute{
                "enabled": schema.BoolAttribute{
                    Description: "Whether DDoS protection is enabled.",
                    Computed:    true,
                },
            },
        },
        "functions": schema.SingleNestedAttribute{
            Description: "Functions module configuration.",
            Computed:    true,
            Attributes: map[string]schema.Attribute{
                "enabled": schema.BoolAttribute{
                    Description: "Whether functions are enabled.",
                    Computed:    true,
                },
            },
        },
        "network_protection": schema.SingleNestedAttribute{
            Description: "Network protection module configuration.",
            Computed:    true,
            Attributes: map[string]schema.Attribute{
                "enabled": schema.BoolAttribute{
                    Description: "Whether network protection is enabled.",
                    Computed:    true,
                },
            },
        },
        "waf": schema.SingleNestedAttribute{
            Description: "WAF module configuration.",
            Computed:    true,
            Attributes: map[string]schema.Attribute{
                "enabled": schema.BoolAttribute{
                    Description: "Whether WAF is enabled.",
                    Computed:    true,
                },
            },
        },
    },
},
```

### Module Struct Pattern

```go
type FirewallModules struct {
    DdosProtection    *DdosProtectionModule    `tfsdk:"ddos_protection"`
    Functions         *FunctionsModule         `tfsdk:"functions"`
    NetworkProtection *NetworkProtectionModule `tfsdk:"network_protection"`
    WAF               *WAFModule               `tfsdk:"waf"`
}

type DdosProtectionModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type FunctionsModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type NetworkProtectionModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}

type WAFModule struct {
    Enabled types.Bool `tfsdk:"enabled"`
}
```

---

## Common Issues

### 1. Wrong SDK Import Path

**Problem:** Using the legacy `edge-api` import path instead of `azion-api`.

```go
// WRONG - Legacy import
import sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"

// CORRECT - V4 SDK import
import sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### 2. Wrong Client Field

**Problem:** Using `edgeApi` instead of `api`.

```go
// WRONG - Uses deprecated client field
f.client.edgeApi.FirewallsAPI.ListFirewalls(ctx).Execute()

// CORRECT - Uses V4 SDK client field
f.client.api.FirewallsAPI.ListFirewalls(ctx).Execute()
```

### 3. Using "edge" Prefix in Naming

**Problem:** Using `EdgeFirewallDataSource` or `edge_firewall_id` instead of `FirewallDataSource` or `firewall_id`.

```go
// WRONG - Uses legacy "edge" naming
func dataSourceAzionEdgeFirewall() datasource.DataSource { ... }
type EdgeFirewallDataSource struct { ... }
func (e *EdgeFirewallDataSource) Read(...) { ... }
resp.TypeName = req.ProviderTypeName + "_edge_firewall_main_setting"

// CORRECT - No "edge" prefix
func dataSourceAzionFirewall() datasource.DataSource { ... }
type FirewallDataSource struct { ... }
func (f *FirewallDataSource) Read(...) { ... }
resp.TypeName = req.ProviderTypeName + "_firewall_main_setting"
```

### 4. Wrong Response Type in RetryOn429

**Problem:** Mismatched response types when using `RetryOn429`.

```go
// WRONG - Type mismatch
firewallsResponse, response, err = utils.RetryOn429(func() (*sdk.SomeOtherType, *http.Response, error) {

// CORRECT - Match the return type from the API call
firewallsResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedFirewallList, *http.Response, error) {
    return f.client.api.FirewallsAPI.ListFirewalls(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute()
}, 5)
```

### 5. Missing Response Body Close

Always close the response body after retry:

```go
if response != nil {
    defer response.Body.Close()
}
```

### 6. Time Formatting

Use `time.RFC3339` for the `last_modified` field:

```go
LastModified: types.StringValue(results.GetLastModified().Format(time.RFC3339)),
```

### 7. Resource Update Uses PATCH

Use `PartialUpdateFirewall` with `PatchedFirewallRequest` for updates (PATCH, not PUT):

```go
// CORRECT - Uses PATCH for partial updates
r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, firewallID).PatchedFirewallRequest(firewallRequest).Execute()
```

---

## Summary Checklist

When implementing Firewall data sources and resources:

1. **Use correct import**: `github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api`
2. **Use correct client field**: `f.client.api.FirewallsAPI`
3. **Remove "edge" prefix**: Use `FirewallDataSource`, not `EdgeFirewallDataSource`
4. **Remove "edge_" from attributes**: Use `firewall_id`, not `edge_firewall_id`
5. **Remove "edge_" from type names**: Use `azion_firewall_main_setting`, not `azion_edge_firewall_main_setting`
6. **Match response types**: `*sdk.FirewallResponse` for singular, `*sdk.PaginatedFirewallList` for plural
7. **Handle 429 errors**: Use `utils.RetryOn429` with correct type signature
8. **Close response body**: `defer response.Body.Close()` after retry
9. **Format time correctly**: Use `time.RFC3339`
10. **Use PATCH for updates**: `PartialUpdateFirewall` with `PatchedFirewallRequest`
11. **Register in provider.go**: Add to `DataSources()` and `Resources()` functions
