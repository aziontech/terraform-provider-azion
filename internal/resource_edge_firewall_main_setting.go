package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgefirewall"

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
	SchemaVersion types.Int64                  `tfsdk:"schema_version"`
	EdgeFirewall  *EdgeFirewallResourceResults `tfsdk:"results"`
	ID            types.String                 `tfsdk:"id"`
	LastUpdated   types.String                 `tfsdk:"last_updated"`
}

type EdgeFirewallResourceResults struct {
	ID                       types.Int64  `tfsdk:"id"`
	LastEditor               types.String `tfsdk:"last_editor"`
	LastModified             types.String `tfsdk:"last_modified"`
	Name                     types.String `tfsdk:"name"`
	IsActive                 types.Bool   `tfsdk:"is_active"`
	EdgeFunctionsEnabled     types.Bool   `tfsdk:"edge_functions_enabled"`
	NetworkProtectionEnabled types.Bool   `tfsdk:"network_protection_enabled"`
	WAFEnabled               types.Bool   `tfsdk:"waf_enabled"`
	DebugRules               types.Bool   `tfsdk:"debug_rules"`
	Domains                  types.List   `tfsdk:"domains"`
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
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the edge firewall rule set.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the edge firewall rule set.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the edge firewall rule set.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the edge firewall rule set.",
						Required:    true,
					},
					"is_active": schema.BoolAttribute{
						Description: "Whether the edge firewall rule set is active.",
						Computed:    true,
						Optional:    true,
					},
					"edge_functions_enabled": schema.BoolAttribute{
						Description: "Whether edge functions are enabled for the rule set.",
						Computed:    true,
						Optional:    true,
					},
					"network_protection_enabled": schema.BoolAttribute{
						Description: "Whether network protection is enabled for the rule set.",
						Computed:    true,
						Optional:    true,
					},
					"waf_enabled": schema.BoolAttribute{
						Description: "Whether Web Application Firewall (WAF) is enabled for the rule set.",
						Computed:    true,
						Optional:    true,
					},
					"debug_rules": schema.BoolAttribute{
						Description: "Whether debug rules are enabled for the rule set.",
						Computed:    true,
						Optional:    true,
					},
					"domains": schema.ListAttribute{
						Computed:    true,
						Optional:    true,
						ElementType: types.Int64Type,
						Description: "List of domains associated with the edge firewall rule set.",
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

	edgeFirewallRequest := edgefirewall.CreateEdgeFirewallRequest{
		Name:                     plan.EdgeFirewall.Name.ValueStringPointer(),
		IsActive:                 plan.EdgeFirewall.IsActive.ValueBoolPointer(),
		EdgeFunctionsEnabled:     plan.EdgeFirewall.EdgeFunctionsEnabled.ValueBoolPointer(),
		NetworkProtectionEnabled: plan.EdgeFirewall.NetworkProtectionEnabled.ValueBoolPointer(),
		WafEnabled:               plan.EdgeFirewall.WAFEnabled.ValueBoolPointer(),
	}

	if len(plan.EdgeFirewall.Domains.Elements()) > 0 {
		diags = plan.EdgeFirewall.Domains.ElementsAs(ctx, &edgeFirewallRequest.Domains, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	edgeFirewallResponse, response, err := r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallPost(ctx).CreateEdgeFirewallRequest(edgeFirewallRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFirewallResponse, response, err = utils.RetryOn429(func() (*edgefirewall.EdgeFirewallResponse, *http.Response, error) {
				return r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallPost(ctx).CreateEdgeFirewallRequest(edgeFirewallRequest).Execute() //nolint
			}, 15) // Maximum 15 retries

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

	plan.SchemaVersion = types.Int64Value(3)
	var sliceInt []types.Int64
	for _, itemsValuesInt := range edgeFirewallResponse.Results.GetDomains() {
		sliceInt = append(sliceInt, types.Int64Value(itemsValuesInt))
	}
	plan.EdgeFirewall = &EdgeFirewallResourceResults{
		ID:                       types.Int64Value(edgeFirewallResponse.Results.GetId()),
		LastEditor:               types.StringValue(edgeFirewallResponse.Results.GetLastEditor()),
		LastModified:             types.StringValue(edgeFirewallResponse.Results.GetLastModified()),
		Name:                     types.StringValue(edgeFirewallResponse.Results.GetName()),
		IsActive:                 types.BoolValue(edgeFirewallResponse.Results.GetIsActive()),
		EdgeFunctionsEnabled:     types.BoolValue(edgeFirewallResponse.Results.GetEdgeFunctionsEnabled()),
		NetworkProtectionEnabled: types.BoolValue(edgeFirewallResponse.Results.GetNetworkProtectionEnabled()),
		WAFEnabled:               types.BoolValue(edgeFirewallResponse.Results.GetWafEnabled()),
		DebugRules:               types.BoolValue(edgeFirewallResponse.Results.GetDebugRules()),
		Domains:                  utils.SliceIntTypeToList(sliceInt),
	}

	plan.ID = types.StringValue(strconv.FormatInt(edgeFirewallResponse.Results.GetId(), 10))
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
	var edgeFirewallID string
	if state.ID.IsNull() {
		edgeFirewallID = strconv.Itoa(int(state.EdgeFirewall.ID.ValueInt64()))
	} else {
		edgeFirewallID = state.ID.ValueString()
	}

	edgeFirewallResponse, response, err := r.client.edgeFirewallApi.DefaultAPI.
		EdgeFirewallUuidGet(ctx, edgeFirewallID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			edgeFirewallResponse, response, err = utils.RetryOn429(func() (*edgefirewall.EdgeFirewallResponse, *http.Response, error) {
				return r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidGet(ctx, edgeFirewallID).Execute() //nolint
			}, 15) // Maximum 15 retries

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

	var sliceInt []types.Int64
	for _, itemsValuesInt := range edgeFirewallResponse.Results.GetDomains() {
		sliceInt = append(sliceInt, types.Int64Value(int64(itemsValuesInt)))
	}

	state.EdgeFirewall = &EdgeFirewallResourceResults{
		ID:                       types.Int64Value(edgeFirewallResponse.Results.GetId()),
		LastEditor:               types.StringValue(edgeFirewallResponse.Results.GetLastEditor()),
		LastModified:             types.StringValue(edgeFirewallResponse.Results.GetLastModified()),
		Name:                     types.StringValue(edgeFirewallResponse.Results.GetName()),
		IsActive:                 types.BoolValue(edgeFirewallResponse.Results.GetIsActive()),
		EdgeFunctionsEnabled:     types.BoolValue(edgeFirewallResponse.Results.GetEdgeFunctionsEnabled()),
		NetworkProtectionEnabled: types.BoolValue(edgeFirewallResponse.Results.GetNetworkProtectionEnabled()),
		WAFEnabled:               types.BoolValue(edgeFirewallResponse.Results.GetWafEnabled()),
		DebugRules:               types.BoolValue(edgeFirewallResponse.Results.GetDebugRules()),
		Domains:                  utils.SliceIntTypeToList(sliceInt),
	}
	state.ID = types.StringValue(edgeFirewallID)
	state.SchemaVersion = types.Int64Value(int64(edgeFirewallResponse.GetSchemaVersion()))

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

	var edgeFirewallID string
	if state.ID.IsNull() {
		edgeFirewallID = strconv.Itoa(int(state.EdgeFirewall.ID.ValueInt64()))
	} else {
		edgeFirewallID = state.ID.ValueString()
	}

	edgeFirewallRequest := edgefirewall.UpdateEdgeFirewallRequest{
		Name:                     plan.EdgeFirewall.Name.ValueStringPointer(),
		IsActive:                 plan.EdgeFirewall.IsActive.ValueBoolPointer(),
		EdgeFunctionsEnabled:     plan.EdgeFirewall.EdgeFunctionsEnabled.ValueBoolPointer(),
		NetworkProtectionEnabled: plan.EdgeFirewall.NetworkProtectionEnabled.ValueBoolPointer(),
		WafEnabled:               plan.EdgeFirewall.WAFEnabled.ValueBoolPointer(),
	}

	requestItemsValue := plan.EdgeFirewall.Domains.ElementsAs(ctx, &edgeFirewallRequest.Domains, false)
	resp.Diagnostics.Append(requestItemsValue...)
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFirewallResponse, response, err := r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidPut(ctx, edgeFirewallID).UpdateEdgeFirewallRequest(edgeFirewallRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFirewallResponse, response, err = utils.RetryOn429(func() (*edgefirewall.EdgeFirewallResponse, *http.Response, error) {
				return r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidPut(ctx, edgeFirewallID).UpdateEdgeFirewallRequest(edgeFirewallRequest).Execute() //nolint
			}, 15) // Maximum 15 retries

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

	plan.SchemaVersion = types.Int64Value(3)
	var sliceInt []types.Int64
	for _, itemsValuesInt := range edgeFirewallResponse.Results.GetDomains() {
		sliceInt = append(sliceInt, types.Int64Value(int64(itemsValuesInt)))
	}
	plan.EdgeFirewall = &EdgeFirewallResourceResults{
		ID:                       types.Int64Value(edgeFirewallResponse.Results.GetId()),
		LastEditor:               types.StringValue(edgeFirewallResponse.Results.GetLastEditor()),
		LastModified:             types.StringValue(edgeFirewallResponse.Results.GetLastModified()),
		Name:                     types.StringValue(edgeFirewallResponse.Results.GetName()),
		IsActive:                 types.BoolValue(edgeFirewallResponse.Results.GetIsActive()),
		EdgeFunctionsEnabled:     types.BoolValue(edgeFirewallResponse.Results.GetEdgeFunctionsEnabled()),
		NetworkProtectionEnabled: types.BoolValue(edgeFirewallResponse.Results.GetNetworkProtectionEnabled()),
		WAFEnabled:               types.BoolValue(edgeFirewallResponse.Results.GetWafEnabled()),
		DebugRules:               types.BoolValue(edgeFirewallResponse.Results.GetDebugRules()),
		Domains:                  utils.SliceIntTypeToList(sliceInt),
	}

	plan.ID = types.StringValue(strconv.FormatInt(edgeFirewallResponse.Results.GetId(), 10))
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

	var edgeFirewallID string
	if state.ID.IsNull() {
		edgeFirewallID = strconv.Itoa(int(state.EdgeFirewall.ID.ValueInt64()))
	} else {
		edgeFirewallID = state.ID.ValueString()
	}

	response, err := r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidDelete(ctx, edgeFirewallID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
				return r.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidDelete(ctx, edgeFirewallID).Execute() //nolint
			}, 15) // Maximum 15 retries

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
