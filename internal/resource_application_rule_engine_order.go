package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &applicationRuleEngineOrderResource{}
	_ resource.ResourceWithConfigure   = &applicationRuleEngineOrderResource{}
	_ resource.ResourceWithImportState = &applicationRuleEngineOrderResource{}
)

func NewApplicationRuleEngineOrderResource() resource.Resource {
	return &applicationRuleEngineOrderResource{}
}

type applicationRuleEngineOrderResource struct {
	client *apiClient
}

type applicationRuleEngineOrderModel struct {
	ID            types.String  `tfsdk:"id"`
	ApplicationID types.Int64   `tfsdk:"application_id"`
	Phase         types.String  `tfsdk:"phase"`
	Order         []types.Int64 `tfsdk:"order"`
	LastUpdated   types.String  `tfsdk:"last_updated"`
}

func (r *applicationRuleEngineOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_rule_engine_order"
}

func (r *applicationRuleEngineOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.Int64Attribute{
				Description: "The application identifier whose rules are being ordered.",
				Required:    true,
			},
			"phase": schema.StringAttribute{
				Description: "The rule phase to order. Must be 'request' or 'response'.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("request", "response"),
				},
			},
			"order": schema.ListAttribute{
				Description: "The ordered list of rule IDs. The first ID will be evaluated first. All managed rules of the chosen phase must be present.",
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

func (r *applicationRuleEngineOrderResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *applicationRuleEngineOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationRuleEngineOrderModel
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
	plan.ID = types.StringValue(fmt.Sprintf("%d/%s", plan.ApplicationID.ValueInt64(), plan.Phase.ValueString()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationRuleEngineOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationRuleEngineOrderModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationID, phase, ok := parseOrderID(state.ID.ValueString(), state.ApplicationID, state.Phase, resp)
	if !ok {
		return
	}

	currentOrder, removed, err := r.listOrderedRuleIDs(ctx, applicationID, phase)
	if err != nil {
		if removed {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(err.Error(), "failed to list rules for drift detection")
		return
	}

	state.ApplicationID = types.Int64Value(applicationID)
	state.Phase = types.StringValue(phase)
	state.Order = intSliceToInt64TypeSlice(currentOrder)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationRuleEngineOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationRuleEngineOrderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state applicationRuleEngineOrderModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ApplicationID.IsNull() {
		plan.ApplicationID = state.ApplicationID
	}
	if plan.Phase.IsNull() {
		plan.Phase = state.Phase
	}

	r.applyOrder(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.ID = types.StringValue(fmt.Sprintf("%d/%s", plan.ApplicationID.ValueInt64(), plan.Phase.ValueString()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationRuleEngineOrderResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The API has no inverse operation for ordering. Rules retain their last set order.
	resp.Diagnostics.AddWarning(
		"Rule order not reset on destroy",
		"The Azion API does not provide an operation to clear or reset rule order. Rules will remain in the order last applied. The resource has been removed from Terraform state only.",
	)
}

func (r *applicationRuleEngineOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Expected format: {application_id}/{phase}",
		)
		return
	}
	applicationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application ID", "Could not parse application ID")
		return
	}
	phase := parts[1]
	if phase != "request" && phase != "response" {
		resp.Diagnostics.AddError("Invalid phase", "Phase must be 'request' or 'response'")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%d/%s", applicationID, phase))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("phase"), phase)...)
}

func (r *applicationRuleEngineOrderResource) applyOrder(ctx context.Context, plan *applicationRuleEngineOrderModel, diags diagAccumulator) {
	applicationID := plan.ApplicationID.ValueInt64()
	phase := plan.Phase.ValueString()

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

	switch phase {
	case "request":
		body := azionapi.NewApplicationRequestPhaseRuleEngineOrder(orderIDs)
		_, response, err := r.client.api.ApplicationsRequestRulesAPI.
			UpdateApplicationRequestRulesOrder(ctx, applicationID).
			ApplicationRequestPhaseRuleEngineOrder(*body).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response != nil && response.StatusCode == 429 {
				_, response, err = utils.RetryOn429(func() (*azionapi.PaginatedRequestPhaseRuleList, *http.Response, error) {
					return r.client.api.ApplicationsRequestRulesAPI.
						UpdateApplicationRequestRulesOrder(ctx, applicationID).
						ApplicationRequestPhaseRuleEngineOrder(*body).
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
	case "response":
		body := azionapi.NewApplicationResponsePhaseRuleEngineOrderRequest(orderIDs)
		_, response, err := r.client.api.ApplicationsResponseRulesAPI.
			UpdateApplicationResponseRulesOrder(ctx, applicationID).
			ApplicationResponsePhaseRuleEngineOrderRequest(*body).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response != nil && response.StatusCode == 429 {
				_, response, err = utils.RetryOn429(func() (*azionapi.PaginatedResponsePhaseRuleList, *http.Response, error) {
					return r.client.api.ApplicationsResponseRulesAPI.
						UpdateApplicationResponseRulesOrder(ctx, applicationID).
						ApplicationResponsePhaseRuleEngineOrderRequest(*body).
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
	default:
		diags.AddError("Invalid phase", fmt.Sprintf("Phase must be 'request' or 'response', got: %s", phase))
	}
}

// listOrderedRuleIDs returns the rule IDs currently stored for the given phase, sorted by their `order` field.
// The boolean second return value indicates the application/phase is gone (404), so the resource should be dropped from state.
func (r *applicationRuleEngineOrderResource) listOrderedRuleIDs(ctx context.Context, applicationID int64, phase string) ([]int64, bool, error) {
	type ruleEntry struct {
		id    int64
		order int64
	}
	var rules []ruleEntry

	var page int64 = 1
	const pageSize int64 = 100

	for {
		entries, totalPages, removed, err := r.fetchRulePage(ctx, applicationID, phase, page, pageSize)
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

type ruleIDOrder struct {
	id    int64
	order int64
}

func (r *applicationRuleEngineOrderResource) fetchRulePage(ctx context.Context, applicationID int64, phase string, page, pageSize int64) ([]ruleIDOrder, int64, bool, error) {
	switch phase {
	case "request":
		listResp, response, err := r.client.api.ApplicationsRequestRulesAPI.
			ListApplicationRequestRules(ctx, applicationID).
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
	case "response":
		listResp, response, err := r.client.api.ApplicationsResponseRulesAPI.
			ListApplicationResponseRules(ctx, applicationID).
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
	default:
		return nil, 0, false, fmt.Errorf("invalid phase: %s", phase)
	}
}

// diagAccumulator is the subset of diag.Diagnostics used by applyOrder, so it can be
// invoked from any of the CRUD methods without per-method duplication.
type diagAccumulator interface {
	AddError(summary, detail string)
	AddWarning(summary, detail string)
}

func appendBodyError(diags diagAccumulator, response *http.Response, err error) {
	if response == nil {
		diags.AddError(err.Error(), "API request failed")
		return
	}
	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		diags.AddError(readErr.Error(), "failed to read response body")
		return
	}
	diags.AddError(err.Error(), string(bodyBytes))
}

func parseOrderID(rawID string, fallbackAppID types.Int64, fallbackPhase types.String, resp *resource.ReadResponse) (int64, string, bool) {
	parts := strings.Split(rawID, "/")
	if len(parts) == 2 {
		appID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid state ID", "Could not parse application ID from state ID")
			return 0, "", false
		}
		return appID, parts[1], true
	}
	if !fallbackAppID.IsNull() && !fallbackPhase.IsNull() {
		return fallbackAppID.ValueInt64(), fallbackPhase.ValueString(), true
	}
	resp.Diagnostics.AddError("Invalid state ID", "State ID must be in the form {application_id}/{phase}")
	return 0, "", false
}

func intSliceToInt64TypeSlice(in []int64) []types.Int64 {
	out := make([]types.Int64, 0, len(in))
	for _, v := range in {
		out = append(out, types.Int64Value(v))
	}
	return out
}
