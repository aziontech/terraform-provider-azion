package provider

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &firewallRuleEngineOrderResource{}
	_ resource.ResourceWithConfigure   = &firewallRuleEngineOrderResource{}
	_ resource.ResourceWithImportState = &firewallRuleEngineOrderResource{}
)

func NewFirewallRuleEngineOrderResource() resource.Resource {
	return &firewallRuleEngineOrderResource{}
}

type firewallRuleEngineOrderResource struct {
	client *apiClient
}

type firewallRuleEngineOrderModel struct {
	ID          types.String  `tfsdk:"id"`
	FirewallID  types.Int64   `tfsdk:"firewall_id"`
	Order       []types.Int64 `tfsdk:"order"`
	LastUpdated types.String  `tfsdk:"last_updated"`
}

func (r *firewallRuleEngineOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule_engine_order"
}

func (r *firewallRuleEngineOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall identifier whose rules are being ordered.",
				Required:    true,
			},
			"order": schema.ListAttribute{
				Description: "The ordered list of rule IDs. The first ID will be evaluated first.",
				Required:    true,
				ElementType: types.Int64Type,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
		},
	}
}

func (r *firewallRuleEngineOrderResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *firewallRuleEngineOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallRuleEngineOrderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyOrder(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.ID = types.StringValue(strconv.FormatInt(plan.FirewallID.ValueInt64(), 10))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallRuleEngineOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallRuleEngineOrderModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallID, ok := parseFirewallOrderID(state.ID.ValueString(), state.FirewallID, resp)
	if !ok {
		return
	}

	currentOrder, removed, err := r.listOrderedRuleIDs(ctx, firewallID)
	if err != nil {
		if removed {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(err.Error(), "failed to list firewall rules for drift detection")
		return
	}

	state.FirewallID = types.Int64Value(firewallID)
	state.Order = intSliceToInt64TypeSlice(currentOrder)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallRuleEngineOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallRuleEngineOrderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state firewallRuleEngineOrderModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsNull() {
		plan.FirewallID = state.FirewallID
	}

	r.applyOrder(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.ID = types.StringValue(strconv.FormatInt(plan.FirewallID.ValueInt64(), 10))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *firewallRuleEngineOrderResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The API has no inverse operation for ordering. Rules retain their last set order.
	resp.Diagnostics.AddWarning(
		"Rule order not reset on destroy",
		"The Azion API does not provide an operation to clear or reset firewall rule order. Rules will remain in the order last applied. The resource has been removed from Terraform state only.",
	)
}

func (r *firewallRuleEngineOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	firewallID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Expected import ID to be the firewall_id (numeric).",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(firewallID, 10))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("firewall_id"), firewallID)...)
}

func (r *firewallRuleEngineOrderResource) applyOrder(ctx context.Context, plan *firewallRuleEngineOrderModel, diags diagAccumulator) {
	firewallID := plan.FirewallID.ValueInt64()

	orderIDs := make([]int64, 0, len(plan.Order))
	for _, v := range plan.Order {
		if v.IsNull() || v.IsUnknown() {
			diags.AddError("Invalid order", "Order list cannot contain null or unknown values.")
			return
		}
		orderIDs = append(orderIDs, v.ValueInt64())
	}
	if len(orderIDs) == 0 {
		diags.AddError("Empty order", "The order list must contain at least one rule ID.")
		return
	}

	body := azionapi.NewFirewallRuleEngineOrderRequest(orderIDs)
	_, response, err := r.client.api.FirewallsRulesEngineAPI.
		OrderFirewallRules(ctx, firewallID).
		FirewallRuleEngineOrderRequest(*body).
		Execute()
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.PaginatedFirewallRuleList, *http.Response, error) {
				return r.client.api.FirewallsRulesEngineAPI.
					OrderFirewallRules(ctx, firewallID).
					FirewallRuleEngineOrderRequest(*body).
					Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				diags.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			appendBodyError(diags, response, err)
			return
		}
	}
}

// listOrderedRuleIDs returns the rule IDs currently stored for the given firewall, sorted by their `order` field.
// The boolean second return value indicates the firewall is gone (404), so the resource should be dropped from state.
func (r *firewallRuleEngineOrderResource) listOrderedRuleIDs(ctx context.Context, firewallID int64) ([]int64, bool, error) {
	type ruleEntry struct {
		id    int64
		order int64
	}
	var rules []ruleEntry

	var page int64 = 1
	const pageSize int64 = 100

	for {
		entries, totalPages, removed, err := r.fetchRulePage(ctx, firewallID, page, pageSize)
		if err != nil {
			return nil, removed, err
		}
		for _, e := range entries {
			rules = append(rules, ruleEntry{id: e.id, order: e.order})
		}
		if totalPages == 0 || page >= totalPages {
			break
		}
		page++
	}

	sort.SliceStable(rules, func(i, j int) bool { return rules[i].order < rules[j].order })

	ids := make([]int64, 0, len(rules))
	for _, e := range rules {
		ids = append(ids, e.id)
	}
	return ids, false, nil
}

func (r *firewallRuleEngineOrderResource) fetchRulePage(ctx context.Context, firewallID, page, pageSize int64) ([]ruleIDOrder, int64, bool, error) {
	listResp, response, err := r.client.api.FirewallsRulesEngineAPI.
		ListFirewallRules(ctx, firewallID).
		Page(page).PageSize(pageSize).Ordering("order").Execute()
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil, 0, true, err
		}
		return nil, 0, false, err
	}
	entries := make([]ruleIDOrder, 0, len(listResp.Results))
	for _, rule := range listResp.Results {
		entries = append(entries, ruleIDOrder{id: rule.GetId(), order: rule.GetOrder()})
	}
	var totalPages int64
	if listResp.TotalPages != nil {
		totalPages = *listResp.TotalPages
	}
	return entries, totalPages, false, nil
}

func parseFirewallOrderID(rawID string, fallbackFirewallID types.Int64, resp *resource.ReadResponse) (int64, bool) {
	if rawID != "" {
		firewallID, err := strconv.ParseInt(rawID, 10, 64)
		if err == nil {
			return firewallID, true
		}
	}
	if !fallbackFirewallID.IsNull() {
		return fallbackFirewallID.ValueInt64(), true
	}
	resp.Diagnostics.AddError("Invalid state ID", fmt.Sprintf("State ID must be the numeric firewall_id, got %q", rawID))
	return 0, false
}
