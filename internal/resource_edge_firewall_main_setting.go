package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
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
	_ resource.Resource                = &edgeFirewallResource{}
	_ resource.ResourceWithConfigure   = &edgeFirewallResource{}
	_ resource.ResourceWithImportState = &edgeFirewallResource{}
)

func EdgeFirewallResource() resource.Resource {
	return &edgeFirewallResource{}
}

type edgeFirewallResource struct {
	client *apiClient
}

type EdgeFirewallResourceModel struct {
	EdgeFirewall *EdgeFirewallResourceResults `tfsdk:"data"`
	ID           types.Int64                  `tfsdk:"id"`
	LastUpdated  types.String                 `tfsdk:"last_updated"`
}

type EdgeFirewallResourceModules struct {
	DdosProtection    *DdosProtectionModule    `tfsdk:"ddos_protection"`
	Functions         *FunctionsModule         `tfsdk:"functions"`
	NetworkProtection *NetworkProtectionModule `tfsdk:"network_protection"`
	WAF               *WAFModule               `tfsdk:"waf"`
}

type EdgeFirewallResourceResults struct {
	ID             types.Int64                  `tfsdk:"id"`
	Name           types.String                 `tfsdk:"name"`
	Modules        *EdgeFirewallResourceModules `tfsdk:"modules"`
	Debug          types.Bool                   `tfsdk:"debug"`
	Active         types.Bool                   `tfsdk:"active"`
	LastEditor     types.String                 `tfsdk:"last_editor"`
	LastModified   types.String                 `tfsdk:"last_modified"`
	ProductVersion types.String                 `tfsdk:"product_version"`
}

func (r *edgeFirewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_main_setting"
}

func (r *edgeFirewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the firewall rule set.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the firewall rule set.",
						Required:    true,
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Modules configuration for the firewall.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"ddos_protection": schema.SingleNestedAttribute{
								Description: "DDoS protection module configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether DDoS protection is enabled.",
										Optional:    true,
									},
								},
							},
							"functions": schema.SingleNestedAttribute{
								Description: "Functions module configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether functions are enabled.",
										Optional:    true,
									},
								},
							},
							"network_protection": schema.SingleNestedAttribute{
								Description: "Network protection module configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether network protection is enabled.",
										Optional:    true,
									},
								},
							},
							"waf": schema.SingleNestedAttribute{
								Description: "WAF module configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether WAF is enabled.",
										Optional:    true,
									},
								},
							},
						},
					},
					"debug": schema.BoolAttribute{
						Description: "Whether debug is enabled for the rule set.",
						Optional:    true,
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the firewall rule set is active.",
						Optional:    true,
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the firewall rule set.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the firewall rule set.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the firewall rule set.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *edgeFirewallResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *edgeFirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeFirewallResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	modules := sdk.FirewallModulesRequest{}
	if plan.EdgeFirewall.Modules != nil {
		if plan.EdgeFirewall.Modules.Functions != nil && !plan.EdgeFirewall.Modules.Functions.Enabled.IsNull() {
			modules.Functions = &sdk.FirewallModuleRequest{
				Enabled: plan.EdgeFirewall.Modules.Functions.Enabled.ValueBoolPointer(),
			}
		}
		if plan.EdgeFirewall.Modules.NetworkProtection != nil && !plan.EdgeFirewall.Modules.NetworkProtection.Enabled.IsNull() {
			modules.NetworkProtection = &sdk.FirewallModuleRequest{
				Enabled: plan.EdgeFirewall.Modules.NetworkProtection.Enabled.ValueBoolPointer(),
			}
		}
		if plan.EdgeFirewall.Modules.WAF != nil && !plan.EdgeFirewall.Modules.WAF.Enabled.IsNull() {
			modules.Waf = &sdk.FirewallModuleRequest{
				Enabled: plan.EdgeFirewall.Modules.WAF.Enabled.ValueBoolPointer(),
			}
		}
	}

	edgeFirewallRequest := sdk.FirewallRequest{
		Name:    plan.EdgeFirewall.Name.ValueString(),
		Active:  plan.EdgeFirewall.Active.ValueBoolPointer(),
		Debug:   plan.EdgeFirewall.Debug.ValueBoolPointer(),
		Modules: &modules,
	}

	edgeFirewallResponse, response, err := r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(edgeFirewallRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFirewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(edgeFirewallRequest).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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

	mods := edgeFirewallResponse.Data.GetModules()
	ddosProtection := mods.GetDdosProtection()
	functions := mods.GetFunctions()
	networkProtection := mods.GetNetworkProtection()
	waf := mods.GetWaf()

	var responseModulesPtr *EdgeFirewallResourceModules
	if plan.EdgeFirewall.Modules != nil {
		responseModules := EdgeFirewallResourceModules{}

		if plan.EdgeFirewall.Modules.DdosProtection != nil {
			responseModules.DdosProtection = &DdosProtectionModule{
				Enabled: types.BoolValue(ddosProtection.GetEnabled()),
			}
		}
		if plan.EdgeFirewall.Modules.Functions != nil {
			responseModules.Functions = &FunctionsModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			}
		}
		if plan.EdgeFirewall.Modules.NetworkProtection != nil {
			responseModules.NetworkProtection = &NetworkProtectionModule{
				Enabled: types.BoolValue(networkProtection.GetEnabled()),
			}
		}
		if plan.EdgeFirewall.Modules.WAF != nil {
			responseModules.WAF = &WAFModule{
				Enabled: types.BoolValue(waf.GetEnabled()),
			}
		}

		responseModulesPtr = &responseModules
	}

	plan.EdgeFirewall = &EdgeFirewallResourceResults{
		ID:             types.Int64Value(edgeFirewallResponse.Data.GetId()),
		Name:           types.StringValue(edgeFirewallResponse.Data.GetName()),
		Modules:        responseModulesPtr,
		Debug:          types.BoolValue(edgeFirewallResponse.Data.GetDebug()),
		Active:         types.BoolValue(edgeFirewallResponse.Data.GetActive()),
		LastEditor:     types.StringValue(edgeFirewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(edgeFirewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		ProductVersion: types.StringValue(edgeFirewallResponse.Data.GetProductVersion()),
	}

	plan.ID = types.Int64Value(edgeFirewallResponse.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeFirewallResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var edgeFirewallID int64
	if state.ID.IsNull() {
		edgeFirewallID = state.EdgeFirewall.ID.ValueInt64()
	} else {
		edgeFirewallID = state.ID.ValueInt64()
	}

	edgeFirewallResponse, response, err := r.client.api.FirewallsAPI.
		RetrieveFirewall(ctx, edgeFirewallID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			edgeFirewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.RetrieveFirewall(ctx, edgeFirewallID).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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

	modules := edgeFirewallResponse.Data.GetModules()
	ddosProtection := modules.GetDdosProtection()
	functions := modules.GetFunctions()
	networkProtection := modules.GetNetworkProtection()
	waf := modules.GetWaf()

	modulesResponse := EdgeFirewallResourceModules{}
	if state.EdgeFirewall != nil && state.EdgeFirewall.Modules != nil {
		if state.EdgeFirewall.Modules.DdosProtection != nil {
			modulesResponse.DdosProtection = &DdosProtectionModule{
				Enabled: types.BoolValue(ddosProtection.GetEnabled()),
			}
		}
		if state.EdgeFirewall.Modules.Functions != nil {
			modulesResponse.Functions = &FunctionsModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			}
		}
		if state.EdgeFirewall.Modules.NetworkProtection != nil {
			modulesResponse.NetworkProtection = &NetworkProtectionModule{
				Enabled: types.BoolValue(networkProtection.GetEnabled()),
			}
		}
		if state.EdgeFirewall.Modules.WAF != nil {
			modulesResponse.WAF = &WAFModule{
				Enabled: types.BoolValue(waf.GetEnabled()),
			}
		}
	}

	state.EdgeFirewall = &EdgeFirewallResourceResults{
		ID:             types.Int64Value(edgeFirewallResponse.Data.GetId()),
		LastEditor:     types.StringValue(edgeFirewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(edgeFirewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		Name:           types.StringValue(edgeFirewallResponse.Data.GetName()),
		Active:         types.BoolValue(edgeFirewallResponse.Data.GetActive()),
		Debug:          types.BoolValue(edgeFirewallResponse.Data.GetDebug()),
		Modules:        &modulesResponse,
		ProductVersion: types.StringValue(edgeFirewallResponse.Data.GetProductVersion()),
	}
	state.ID = types.Int64Value(edgeFirewallID)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeFirewallResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state EdgeFirewallResourceModel
	diagsEdgeFirewall := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsEdgeFirewall...)
	if resp.Diagnostics.HasError() {
		return
	}

	var edgeFirewallID int64
	if state.ID.IsNull() {
		edgeFirewallID = state.EdgeFirewall.ID.ValueInt64()
	} else {
		edgeFirewallID = state.ID.ValueInt64()
	}

	edgeFirewallRequest := sdk.PatchedFirewallRequest{
		Name:   plan.EdgeFirewall.Name.ValueStringPointer(),
		Active: plan.EdgeFirewall.Active.ValueBoolPointer(),
		Debug:  plan.EdgeFirewall.Debug.ValueBoolPointer(),
	}

	modules := sdk.FirewallModulesRequest{}
	if plan.EdgeFirewall.Modules != nil {
		if plan.EdgeFirewall.Modules.Functions != nil && !plan.EdgeFirewall.Modules.Functions.Enabled.IsNull() {
			modules.Functions = &sdk.FirewallModuleRequest{
				Enabled: plan.EdgeFirewall.Modules.Functions.Enabled.ValueBoolPointer(),
			}
		}
		if plan.EdgeFirewall.Modules.NetworkProtection != nil && !plan.EdgeFirewall.Modules.NetworkProtection.Enabled.IsNull() {
			modules.NetworkProtection = &sdk.FirewallModuleRequest{
				Enabled: plan.EdgeFirewall.Modules.NetworkProtection.Enabled.ValueBoolPointer(),
			}
		}
		if plan.EdgeFirewall.Modules.WAF != nil && !plan.EdgeFirewall.Modules.WAF.Enabled.IsNull() {
			modules.Waf = &sdk.FirewallModuleRequest{
				Enabled: plan.EdgeFirewall.Modules.WAF.Enabled.ValueBoolPointer(),
			}
		}
	}
	edgeFirewallRequest.Modules = &modules

	edgeFirewallResponse, response, err := r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, edgeFirewallID).PatchedFirewallRequest(edgeFirewallRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFirewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, edgeFirewallID).PatchedFirewallRequest(edgeFirewallRequest).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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

	mods := edgeFirewallResponse.Data.GetModules()
	ddosProtection := mods.GetDdosProtection()
	functions := mods.GetFunctions()
	networkProtection := mods.GetNetworkProtection()
	waf := mods.GetWaf()

	var responseModulesPtr *EdgeFirewallResourceModules

	if plan.EdgeFirewall.Modules != nil {
		responseModules := EdgeFirewallResourceModules{}

		if plan.EdgeFirewall.Modules.DdosProtection != nil {
			responseModules.DdosProtection = &DdosProtectionModule{
				Enabled: types.BoolValue(ddosProtection.GetEnabled()),
			}
		}
		if plan.EdgeFirewall.Modules.Functions != nil {
			responseModules.Functions = &FunctionsModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			}
		}
		if plan.EdgeFirewall.Modules.NetworkProtection != nil {
			responseModules.NetworkProtection = &NetworkProtectionModule{
				Enabled: types.BoolValue(networkProtection.GetEnabled()),
			}
		}
		if plan.EdgeFirewall.Modules.WAF != nil {
			responseModules.WAF = &WAFModule{
				Enabled: types.BoolValue(waf.GetEnabled()),
			}
		}

		responseModulesPtr = &responseModules
	}

	plan.EdgeFirewall = &EdgeFirewallResourceResults{
		ID:             types.Int64Value(edgeFirewallResponse.Data.GetId()),
		LastEditor:     types.StringValue(edgeFirewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(edgeFirewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		Name:           types.StringValue(edgeFirewallResponse.Data.GetName()),
		Active:         types.BoolValue(edgeFirewallResponse.Data.GetActive()),
		Debug:          types.BoolValue(edgeFirewallResponse.Data.GetDebug()),
		ProductVersion: types.StringValue(edgeFirewallResponse.Data.GetProductVersion()),
		Modules:        responseModulesPtr,
	}

	plan.ID = types.Int64Value(edgeFirewallResponse.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeFirewallResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var edgeFirewallID int64
	if state.ID.IsNull() {
		edgeFirewallID = state.EdgeFirewall.ID.ValueInt64()
	} else {
		edgeFirewallID = state.ID.ValueInt64()
	}

	_, response, err := r.client.api.FirewallsAPI.DeleteFirewall(ctx, edgeFirewallID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.DeleteResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.DeleteFirewall(ctx, edgeFirewallID).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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
}

func (r *edgeFirewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
