package provider

import (
	"context"
	"net/http"
	"strconv"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	State          types.String             `tfsdk:"state"`
	VersionID      types.String             `tfsdk:"version_id"`
}

// Azion API defaults for firewall fields.
//
// The configuration is the desired state: a field left out is reset to the value
// below on every apply, so a module toggled in Azion Console is undone rather
// than silently adopted.
//
// Module blocks cannot carry a default, so buildFirewallModulesRequest sends the
// value below for every module whose block (or flag) is left out.
const (
	defaultFirewallActive            = true
	defaultFirewallDebug             = false
	defaultFirewallFunctions         = true
	defaultFirewallNetworkProtection = true
	defaultFirewallWAF               = false

	// DDoS protection is always on and cannot be edited, and it is absent from
	// FirewallModulesRequest, so the API accepts no value for it. The attribute
	// stays Optional anyway, so configurations that already declare it keep
	// working; the default keeps plan and state agreeing on the only value it
	// ever has.
	defaultFirewallDdosProtection = true
)

// firewallModuleBlock builds the schema for one configurable module, all of
// which are a single `enabled` flag. Modules are blocks rather than nested
// attributes so Terraform rejects arguments that do not exist instead of
// silently dropping them.
func firewallModuleBlock(description, flagDescription string, enabled bool) schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: flagDescription,
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(enabled),
			},
		},
	}
}

// transformFirewallResponseToModel mirrors the API response into the resource
// model without filtering. State has to match the remote firewall for Terraform
// to plan a revert when a module was toggled outside Terraform; unconfigured
// fields do not drift perpetually because every optional attribute is Computed
// with a schema default, and keepDeclaredFirewallModules drops the module blocks
// the configuration leaves out.
func transformFirewallResponseToModel(data *sdk.Firewall) *FirewallResourceResults {
	if data == nil {
		return nil
	}

	results := &FirewallResourceResults{
		ID:             types.Int64Value(data.GetId()),
		Name:           types.StringValue(data.GetName()),
		Debug:          types.BoolValue(data.GetDebug()),
		Active:         types.BoolValue(data.GetActive()),
		LastEditor:     types.StringValue(data.GetLastEditor()),
		LastModified:   types.StringValue(data.GetLastModified().Format(time.RFC3339)),
		CreatedAt:      types.StringValue(data.GetCreatedAt().Format(time.RFC3339)),
		ProductVersion: types.StringValue(data.GetProductVersion()),
		State:          types.StringPointerValue(data.VersionState.Get()),
		VersionID:      types.StringPointerValue(data.VersionId.Get()),
	}

	if data.Modules == nil {
		return results
	}

	modules := data.GetModules()
	results.Modules = &FirewallResourceModules{}

	// ddos_protection is always present in the response, so it is not a pointer.
	ddosProtection := modules.GetDdosProtection()
	results.Modules.DdosProtection = &DdosProtectionModule{Enabled: types.BoolValue(ddosProtection.GetEnabled())}
	if modules.Functions != nil {
		results.Modules.Functions = &FunctionsModule{Enabled: types.BoolValue(modules.Functions.GetEnabled())}
	}
	if modules.NetworkProtection != nil {
		results.Modules.NetworkProtection = &NetworkProtectionModule{Enabled: types.BoolValue(modules.NetworkProtection.GetEnabled())}
	}
	if modules.Waf != nil {
		results.Modules.WAF = &WAFModule{Enabled: types.BoolValue(modules.Waf.GetEnabled())}
	}

	return results
}

// firewallModuleEnabled returns the configured flag, or the module's default
// when the block or the flag is left out, so an omitted module is still reset to
// its default on every apply.
func firewallModuleEnabled(enabled *types.Bool, fallback bool) *bool {
	if enabled != nil && isSet(*enabled) {
		return enabled.ValueBoolPointer()
	}
	return &fallback
}

// buildFirewallModulesRequest builds the settable modules from a plan. Every
// module the API accepts is always sent, with its default when its block is
// omitted, so one toggled out-of-band is reset. ddos_protection is deliberately
// absent: it is always on and has no request field.
func buildFirewallModulesRequest(mods *FirewallResourceModules) sdk.FirewallModulesRequest {
	if mods == nil {
		mods = &FirewallResourceModules{}
	}

	var functions, networkProtection, waf *types.Bool
	if mods.Functions != nil {
		functions = &mods.Functions.Enabled
	}
	if mods.NetworkProtection != nil {
		networkProtection = &mods.NetworkProtection.Enabled
	}
	if mods.WAF != nil {
		waf = &mods.WAF.Enabled
	}

	return sdk.FirewallModulesRequest{
		Functions:         &sdk.FirewallModuleRequest{Enabled: firewallModuleEnabled(functions, defaultFirewallFunctions)},
		NetworkProtection: &sdk.FirewallModuleRequest{Enabled: firewallModuleEnabled(networkProtection, defaultFirewallNetworkProtection)},
		Waf:               &sdk.FirewallModuleRequest{Enabled: firewallModuleEnabled(waf, defaultFirewallWAF)},
	}
}

// keepDeclaredFirewallModules drops from the API result every module block the
// prior plan or state does not declare. Terraform rejects a block that is
// planned as absent and comes back present, so an undeclared block must stay
// null. A nil prior (import) keeps everything.
func keepDeclaredFirewallModules(prior *FirewallResourceResults, results *FirewallResourceResults) {
	if prior == nil || results == nil || results.Modules == nil {
		return
	}
	if prior.Modules == nil {
		results.Modules = nil
		return
	}
	if prior.Modules.DdosProtection == nil {
		results.Modules.DdosProtection = nil
	}
	if prior.Modules.Functions == nil {
		results.Modules.Functions = nil
	}
	if prior.Modules.NetworkProtection == nil {
		results.Modules.NetworkProtection = nil
	}
	if prior.Modules.WAF == nil {
		results.Modules.WAF = nil
	}
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
		},
		Blocks: map[string]schema.Block{
			"data": schema.SingleNestedBlock{
				Description: "Required. The Firewall settings.",
				Validators: []validator.Object{
					objectvalidator.IsRequired(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the firewall rule set.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the firewall rule set.",
						Required:    true,
					},
					"debug": schema.BoolAttribute{
						Description: "Whether debug is enabled for the rule set.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(defaultFirewallDebug),
					},
					"active": schema.BoolAttribute{
						Description: "Whether the firewall rule set is active.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(defaultFirewallActive),
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
					"state": schema.StringAttribute{
						Description: "The state of the current firewall version.",
						Computed:    true,
					},
					"version_id": schema.StringAttribute{
						Description: "The identifier of the current firewall version.",
						Computed:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"modules": schema.SingleNestedBlock{
						Description: "Modules configuration for the firewall. A module left out is reset to its default on every apply, but its drift is only detected while its block is declared.",
						Blocks: map[string]schema.Block{
							// Stays Optional so existing configurations that declare
							// it keep working, even though the API has no
							// ddos_protection field and the module is always on.
							// Its `enabled` flag is Computed with a default of true, so
							// the value the API reports is the value the plan holds.
							"ddos_protection": firewallModuleBlock(
								"DDoS protection module configuration. Always enabled; the API accepts no value for it.",
								"Whether DDoS protection is enabled. Always true — it cannot be disabled.",
								defaultFirewallDdosProtection,
							),
							"functions": firewallModuleBlock(
								"Functions module configuration.",
								"Whether functions are enabled.",
								defaultFirewallFunctions,
							),
							"network_protection": firewallModuleBlock(
								"Network protection module configuration.",
								"Whether network protection is enabled.",
								defaultFirewallNetworkProtection,
							),
							"waf": firewallModuleBlock(
								"WAF module configuration.",
								"Whether WAF is enabled.",
								defaultFirewallWAF,
							),
						},
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

	modules := buildFirewallModulesRequest(plan.Firewall.Modules)

	firewallRequest := sdk.FirewallRequest{
		Name:    plan.Firewall.Name.ValueString(),
		Active:  plan.Firewall.Active.ValueBoolPointer(),
		Debug:   plan.Firewall.Debug.ValueBoolPointer(),
		Modules: &modules,
	}

	firewallResponse, response, err := r.client.api.FirewallsAPI.CreateFirewall(ctx).FirewallRequest(firewallRequest).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}

	results := transformFirewallResponseToModel(&firewallResponse.Data)
	keepDeclaredFirewallModules(plan.Firewall, results)
	plan.Firewall = results

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
		if state.Firewall == nil {
			resp.Diagnostics.AddError(
				"Firewall id error ",
				"should not be null or empty",
			)
			return
		}
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
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}

	results := transformFirewallResponseToModel(&firewallResponse.Data)
	keepDeclaredFirewallModules(state.Firewall, results)
	state.Firewall = results
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
		if state.Firewall == nil {
			resp.Diagnostics.AddError(
				"Firewall id error ",
				"should not be null or empty",
			)
			return
		}
		firewallID = state.Firewall.ID.ValueInt64()
	} else {
		var err error
		firewallID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse firewall ID", err.Error())
			return
		}
	}

	// Full replacement, not a partial update: the configuration is the desired
	// state, so every field is asserted on every apply. A PATCH would leave a
	// console change to an undeclared module in place forever.
	modules := buildFirewallModulesRequest(plan.Firewall.Modules)

	firewallRequest := sdk.FirewallRequest{
		Name:    plan.Firewall.Name.ValueString(),
		Active:  plan.Firewall.Active.ValueBoolPointer(),
		Debug:   plan.Firewall.Debug.ValueBoolPointer(),
		Modules: &modules,
	}

	firewallResponse, response, err := r.client.api.FirewallsAPI.UpdateFirewall(ctx, firewallID).FirewallRequest(firewallRequest).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return r.client.api.FirewallsAPI.UpdateFirewall(ctx, firewallID).FirewallRequest(firewallRequest).Execute() //nolint
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}

	results := transformFirewallResponseToModel(&firewallResponse.Data)
	keepDeclaredFirewallModules(plan.Firewall, results)
	plan.Firewall = results

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
		if state.Firewall == nil {
			resp.Diagnostics.AddError(
				"Firewall id error ",
				"should not be null or empty",
			)
			return
		}
		firewallID = state.Firewall.ID.ValueInt64()
	} else {
		var err error
		firewallID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse firewall ID", err.Error())
			return
		}
	}

	_, response, err := utils.RetryOn429Delete(func() (*sdk.DeleteResponse, *http.Response, error) {
		return r.client.api.FirewallsAPI.DeleteFirewall(ctx, firewallID).Execute() //nolint
	}, 5) // Maximum 5 retries
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
		return
	}
}

func (r *firewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
