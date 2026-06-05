package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
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
	IsVersioned    types.Bool               `tfsdk:"is_versioned"`
	Version        types.Int64              `tfsdk:"version"`
	VersionState   types.String             `tfsdk:"version_state"`
	VersionID      types.String             `tfsdk:"version_id"`
}

func (r *firewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_main_setting"
}

func (r *firewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
					"created_at": schema.StringAttribute{
						Description: "Creation timestamp of the firewall rule set.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the firewall rule set.",
						Computed:    true,
					},
					"is_versioned": schema.BoolAttribute{
						Description: "Whether the firewall is versioned.",
						Computed:    true,
					},
					"version": schema.Int64Attribute{
						Description: "The current version of the firewall.",
						Computed:    true,
					},
					"version_state": schema.StringAttribute{
						Description: "The state of the current firewall version.",
						Computed:    true,
					},
					"version_id": schema.StringAttribute{
						Description: "The identifier of the current firewall version.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *firewallResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *firewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	firewallRequest := sdk.FirewallRequest{
		Name:    plan.Firewall.Name.ValueString(),
		Active:  plan.Firewall.Active.ValueBoolPointer(),
		Debug:   plan.Firewall.Debug.ValueBoolPointer(),
		Modules: &modules,
	}

	firewallResponse, response, err := r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(firewallRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(firewallRequest).Execute() //nolint
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

	mods := firewallResponse.Data.GetModules()
	ddosProtection := mods.GetDdosProtection()
	functions := mods.GetFunctions()
	networkProtection := mods.GetNetworkProtection()
	waf := mods.GetWaf()

	var responseModulesPtr *FirewallResourceModules
	if plan.Firewall.Modules != nil {
		responseModules := FirewallResourceModules{}

		if plan.Firewall.Modules.DdosProtection != nil {
			responseModules.DdosProtection = &DdosProtectionModule{
				Enabled: types.BoolValue(ddosProtection.GetEnabled()),
			}
		}
		if plan.Firewall.Modules.Functions != nil {
			responseModules.Functions = &FunctionsModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			}
		}
		if plan.Firewall.Modules.NetworkProtection != nil {
			responseModules.NetworkProtection = &NetworkProtectionModule{
				Enabled: types.BoolValue(networkProtection.GetEnabled()),
			}
		}
		if plan.Firewall.Modules.WAF != nil {
			responseModules.WAF = &WAFModule{
				Enabled: types.BoolValue(waf.GetEnabled()),
			}
		}

		responseModulesPtr = &responseModules
	}

	plan.Firewall = &FirewallResourceResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		Modules:        responseModulesPtr,
		Debug:          types.BoolValue(firewallResponse.Data.GetDebug()),
		Active:         types.BoolValue(firewallResponse.Data.GetActive()),
		LastEditor:     types.StringValue(firewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(firewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		CreatedAt:      types.StringValue(firewallResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		ProductVersion: types.StringValue(firewallResponse.Data.GetProductVersion()),
		IsVersioned:    types.BoolValue(firewallResponse.Data.IsVersioned),
		Version:        types.Int64PointerValue(firewallResponse.Data.Version.Get()),
		VersionState:   types.StringPointerValue(firewallResponse.Data.VersionState.Get()),
		VersionID:      types.StringPointerValue(firewallResponse.Data.VersionId.Get()),
	}

	plan.ID = types.StringValue(strconv.FormatInt(firewallResponse.Data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	firewallResponse, response, err := r.client.api.FirewallsAPI.
		RetrieveFirewall(ctx, firewallID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.RetrieveFirewall(ctx, firewallID).Execute() //nolint
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

	// Preserve the prior state's Modules shape so unconfigured submodules
	// aren't introduced into state by the API response, which would cause
	// perpetual drift on subsequent plans (state {} vs plan null). When prior
	// state is nil (import), populate every submodule the API returned so the
	// imported state mirrors reality.
	var priorModules *FirewallResourceModules
	if state.Firewall != nil {
		priorModules = state.Firewall.Modules
	}
	var modulesResponsePtr *FirewallResourceModules
	if firewallResponse.Data.Modules != nil {
		modules := firewallResponse.Data.GetModules()
		modulesResponse := FirewallResourceModules{}
		if priorModules == nil || priorModules.DdosProtection != nil {
			ddosProtection := modules.GetDdosProtection()
			modulesResponse.DdosProtection = &DdosProtectionModule{
				Enabled: types.BoolValue(ddosProtection.GetEnabled()),
			}
		}
		if priorModules == nil || priorModules.Functions != nil {
			functions := modules.GetFunctions()
			modulesResponse.Functions = &FunctionsModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			}
		}
		if priorModules == nil || priorModules.NetworkProtection != nil {
			networkProtection := modules.GetNetworkProtection()
			modulesResponse.NetworkProtection = &NetworkProtectionModule{
				Enabled: types.BoolValue(networkProtection.GetEnabled()),
			}
		}
		if priorModules == nil || priorModules.WAF != nil {
			waf := modules.GetWaf()
			modulesResponse.WAF = &WAFModule{
				Enabled: types.BoolValue(waf.GetEnabled()),
			}
		}
		modulesResponsePtr = &modulesResponse
	}

	state.Firewall = &FirewallResourceResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		LastEditor:     types.StringValue(firewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(firewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		CreatedAt:      types.StringValue(firewallResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		Active:         types.BoolValue(firewallResponse.Data.GetActive()),
		Debug:          types.BoolValue(firewallResponse.Data.GetDebug()),
		Modules:        modulesResponsePtr,
		ProductVersion: types.StringValue(firewallResponse.Data.GetProductVersion()),
		IsVersioned:    types.BoolValue(firewallResponse.Data.IsVersioned),
		Version:        types.Int64PointerValue(firewallResponse.Data.Version.Get()),
		VersionState:   types.StringPointerValue(firewallResponse.Data.VersionState.Get()),
		VersionID:      types.StringPointerValue(firewallResponse.Data.VersionId.Get()),
	}
	state.ID = types.StringValue(strconv.FormatInt(firewallID, 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallResourceModel
	diagsFirewall := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsFirewall...)
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

	firewallRequest := sdk.PatchedFirewallRequest{
		Name:   plan.Firewall.Name.ValueStringPointer(),
		Active: plan.Firewall.Active.ValueBoolPointer(),
		Debug:  plan.Firewall.Debug.ValueBoolPointer(),
	}

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
	firewallRequest.Modules = &modules

	firewallResponse, response, err := r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, firewallID).PatchedFirewallRequest(firewallRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.PartialUpdateFirewall(ctx, firewallID).PatchedFirewallRequest(firewallRequest).Execute() //nolint
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

	mods := firewallResponse.Data.GetModules()
	ddosProtection := mods.GetDdosProtection()
	functions := mods.GetFunctions()
	networkProtection := mods.GetNetworkProtection()
	waf := mods.GetWaf()

	var responseModulesPtr *FirewallResourceModules

	if plan.Firewall.Modules != nil {
		responseModules := FirewallResourceModules{}

		if plan.Firewall.Modules.DdosProtection != nil {
			responseModules.DdosProtection = &DdosProtectionModule{
				Enabled: types.BoolValue(ddosProtection.GetEnabled()),
			}
		}
		if plan.Firewall.Modules.Functions != nil {
			responseModules.Functions = &FunctionsModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			}
		}
		if plan.Firewall.Modules.NetworkProtection != nil {
			responseModules.NetworkProtection = &NetworkProtectionModule{
				Enabled: types.BoolValue(networkProtection.GetEnabled()),
			}
		}
		if plan.Firewall.Modules.WAF != nil {
			responseModules.WAF = &WAFModule{
				Enabled: types.BoolValue(waf.GetEnabled()),
			}
		}

		responseModulesPtr = &responseModules
	}

	plan.Firewall = &FirewallResourceResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		LastEditor:     types.StringValue(firewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(firewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		CreatedAt:      types.StringValue(firewallResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		Active:         types.BoolValue(firewallResponse.Data.GetActive()),
		Debug:          types.BoolValue(firewallResponse.Data.GetDebug()),
		ProductVersion: types.StringValue(firewallResponse.Data.GetProductVersion()),
		Modules:        responseModulesPtr,
		IsVersioned:    types.BoolValue(firewallResponse.Data.IsVersioned),
		Version:        types.Int64PointerValue(firewallResponse.Data.Version.Get()),
		VersionState:   types.StringPointerValue(firewallResponse.Data.VersionState.Get()),
		VersionID:      types.StringPointerValue(firewallResponse.Data.VersionId.Get()),
	}

	plan.ID = types.StringValue(strconv.FormatInt(firewallResponse.Data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

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

	_, response, err := r.client.api.FirewallsAPI.DeleteFirewall(ctx, firewallID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.DeleteResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.DeleteFirewall(ctx, firewallID).Execute() //nolint
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

func (r *firewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
