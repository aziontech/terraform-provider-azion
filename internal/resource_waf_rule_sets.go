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
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// azion_waf_rule_sets is the authoritative counterpart of azion_waf_rule_set.
//
// It owns the complete set of exceptions of one WAF: Read lists every exception
// the API holds, so one created, changed or removed outside Terraform (in Azion
// Console, for example) shows up in the plan, and apply converges the WAF back to
// the configuration — including deleting exceptions the configuration does not
// declare.
//
// Exceptions are keyed by name. Names are not documented as unique per WAF, so
// when the API holds several exceptions with the same name, only one can take the
// name as its key; the rest are keyed "<name>#<exception_id>", which no
// configuration declares, so the plan removes them.

var (
	_ resource.Resource                   = &wafRuleSetsResource{}
	_ resource.ResourceWithConfigure      = &wafRuleSetsResource{}
	_ resource.ResourceWithImportState    = &wafRuleSetsResource{}
	_ resource.ResourceWithModifyPlan     = &wafRuleSetsResource{}
	_ resource.ResourceWithValidateConfig = &wafRuleSetsResource{}
)

const (
	wafRuleSetsPageSize = 100
	// Upper bound on pages read, so a misbehaving pagination response cannot loop forever.
	wafRuleSetsMaxPages = 1000
	// Separates the name from the exception ID in the key of a duplicate-named exception.
	wafRuleSetsDuplicateSeparator = "#"
)

func WafRuleSetsResource() resource.Resource {
	return &wafRuleSetsResource{}
}

type wafRuleSetsResource struct {
	client *apiClient
}

type WafRuleSetsResourceModel struct {
	ID          types.String                    `tfsdk:"id"`
	WafID       types.Int64                     `tfsdk:"waf_id"`
	LastUpdated types.String                    `tfsdk:"last_updated"`
	RuleSets    map[string]WafRuleSetsItemModel `tfsdk:"rule_sets"`
}

// WafRuleSetsItemModel is one exception. Its name is the map key.
type WafRuleSetsItemModel struct {
	ExceptionID  types.Int64                         `tfsdk:"exception_id"`
	RuleID       types.Int64                         `tfsdk:"rule_id"`
	Path         types.String                        `tfsdk:"path"`
	Conditions   []WafExceptionConditionWrapperModel `tfsdk:"conditions"`
	Operator     types.String                        `tfsdk:"operator"`
	Active       types.Bool                          `tfsdk:"active"`
	LastEditor   types.String                        `tfsdk:"last_editor"`
	LastModified types.String                        `tfsdk:"last_modified"`
}

func (r *wafRuleSetsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_sets"
}

func (r *wafRuleSetsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Authoritatively manages every exception (rule set) of a WAF. Exceptions not declared in the configuration are deleted on apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the resource. Same as `waf_id`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"waf_id": schema.Int64Attribute{
				Description: "The WAF identifier. Changing it replaces the resource.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"rule_sets": schema.MapNestedAttribute{
				Description: "Every exception of the WAF, keyed by exception name. An empty map deletes all exceptions.",
				Required:    true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						// No UseStateForUnknown here: it only skips resources with no
						// state at all, so for a key added to an existing resource it
						// copies that element's null prior value into the plan, and
						// apply fails with "inconsistent result after apply". ModifyPlan
						// carries the ID over for keys state already holds.
						"exception_id": schema.Int64Attribute{
							Description: "The ID of the WAF exception.",
							Computed:    true,
						},
						"rule_id": schema.Int64Attribute{
							Description: "The rule ID that this exception applies to. 0 means all rules. Supplied by the API when not declared.",
							Optional:    true,
							Computed:    true,
						},
						"path": schema.StringAttribute{
							Description: "Path pattern for the exception. Supplied by the API when not declared.",
							Optional:    true,
							Computed:    true,
						},
						"conditions": schema.ListNestedAttribute{
							Description: "Conditions for the WAF exception. Each item must contain a single `condition` object.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"condition": schema.SingleNestedAttribute{
										Description: "A single condition for the WAF exception.",
										Required:    true,
										Attributes: map[string]schema.Attribute{
											"match": schema.StringAttribute{
												Description: "The match type for the condition.",
												Required:    true,
											},
											"name": schema.StringAttribute{
												Description: "The name for specific condition on name. Supplied by the API when not declared.",
												Optional:    true,
												Computed:    true,
											},
											"value": schema.StringAttribute{
												Description: "The value for specific condition on value. Supplied by the API when not declared.",
												Optional:    true,
												Computed:    true,
											},
											"condition_type": schema.StringAttribute{
												Description: "Type of condition: generic, specific_on_name, or specific_on_value.",
												Required:    true,
												Validators: []validator.String{
													stringvalidator.OneOf("generic", "specific_on_name", "specific_on_value"),
												},
											},
										},
									},
								},
							},
						},
						"operator": schema.StringAttribute{
							Description: "The operator for the exception: regex or contains. Supplied by the API when not declared.",
							Optional:    true,
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the exception is active. Defaults to `true`.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(defaultWAFRuleActive),
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the exception.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *wafRuleSetsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

// ValidateConfig rejects keys using the separator reserved for duplicate-named
// exceptions read from the API, so a declared key can never collide with one.
func (r *wafRuleSetsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var ruleSets types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("rule_sets"), &ruleSets)...)
	if resp.Diagnostics.HasError() || ruleSets.IsNull() || ruleSets.IsUnknown() {
		return
	}

	for key := range ruleSets.Elements() {
		if isWafRuleSetsDuplicateKey(key) {
			resp.Diagnostics.AddAttributeError(
				path.Root("rule_sets").AtMapKey(key),
				"Reserved exception name",
				fmt.Sprintf("%q matches the \"<name>%s<exception_id>\" form used for duplicate-named exceptions read from the API. Choose a different name.", key, wafRuleSetsDuplicateSeparator),
			)
		}
	}
}

// ModifyPlan keeps the plan to what actually changes. On any change, the
// framework marks every computed attribute the configuration leaves null as
// unknown — in every exception, not only the changed ones. An exception whose
// declared fields match state is left untouched by apply, so its state is the
// exact plan.
func (r *wafRuleSetsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var planRuleSets types.Map
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("rule_sets"), &planRuleSets)...)
	if resp.Diagnostics.HasError() || planRuleSets.IsNull() || planRuleSets.IsUnknown() {
		return
	}

	var plan, state WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for key, planned := range plan.RuleSets {
		current, ok := state.RuleSets[key]
		if !ok {
			// Created by apply: the API assigns these.
			planned.ExceptionID = types.Int64Unknown()
			planned.LastEditor = types.StringUnknown()
			planned.LastModified = types.StringUnknown()
			plan.RuleSets[key] = planned
			continue
		}
		if wafRuleSetsItemNeedsUpdate(planned, current) {
			planned.ExceptionID = current.ExceptionID
			plan.RuleSets[key] = planned
			continue
		}
		plan.RuleSets[key] = current
	}

	// Nothing to do: keep last_updated as well, so the plan is empty.
	if !wafRuleSetsHasChanges(plan.RuleSets, state.RuleSets) && plan.WafID.Equal(state.WafID) {
		plan.LastUpdated = state.LastUpdated
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *wafRuleSetsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wafID := plan.WafID.ValueInt64()

	remote, _, err := r.listAllExceptions(ctx, wafID)
	if err != nil {
		resp.Diagnostics.AddError("Could not list WAF exceptions", err.Error())
		return
	}

	// Adopt exceptions that already carry a declared name instead of creating a
	// second one with the same name. Exceptions the configuration does not declare
	// are left in place: deleting them here would remove something the plan never
	// showed. They are read back on the next refresh and planned for removal then.
	existing, _ := buildWafRuleSetsState(remote, nil)
	adopted := make(map[string]WafRuleSetsItemModel)
	var undeclared []string
	for key, item := range existing {
		if _, declared := plan.RuleSets[key]; declared {
			adopted[key] = item
		} else {
			undeclared = append(undeclared, key)
		}
	}

	result, diags := r.reconcile(ctx, wafID, plan.RuleSets, adopted)
	resp.Diagnostics.Append(diags...)

	// Saved even on a partial failure, so state holds what was applied.
	plan.ID = types.StringValue(strconv.FormatInt(wafID, 10))
	plan.RuleSets = result
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	if len(undeclared) > 0 {
		sort.Strings(undeclared)
		resp.Diagnostics.AddWarning(
			"WAF has exceptions not declared in the configuration",
			fmt.Sprintf("WAF %d already holds exceptions this configuration does not declare: %v. They were left in place; the next plan shows them for removal. Declare them in rule_sets to keep them.", wafID, undeclared),
		)
	}
}

func (r *wafRuleSetsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wafID := state.WafID.ValueInt64()

	remote, statusCode, err := r.listAllExceptions(ctx, wafID)
	if err != nil {
		if statusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not list WAF exceptions", err.Error())
		return
	}

	ruleSets, duplicates := buildWafRuleSetsState(remote, state.RuleSets)
	if len(duplicates) > 0 {
		resp.Diagnostics.AddWarning(
			"WAF has exceptions with duplicate names",
			fmt.Sprintf("WAF %d holds more than one exception with the same name. The extra ones are keyed %v and are planned for removal.", wafID, duplicates),
		)
	}

	state.ID = types.StringValue(strconv.FormatInt(wafID, 10))
	state.RuleSets = ruleSets
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wafRuleSetsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, diags := r.reconcile(ctx, plan.WafID.ValueInt64(), plan.RuleSets, state.RuleSets)
	resp.Diagnostics.Append(diags...)

	// Saved even on a partial failure, so state holds what was applied.
	plan.ID = types.StringValue(strconv.FormatInt(plan.WafID.ValueInt64(), 10))
	plan.RuleSets = result
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes every exception the resource holds. The resource is
// authoritative, so that is every exception of the WAF as of the last refresh.
func (r *wafRuleSetsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remaining, diags := r.reconcile(ctx, state.WafID.ValueInt64(), map[string]WafRuleSetsItemModel{}, state.RuleSets)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		// Keep the exceptions that could not be deleted in state.
		state.RuleSets = remaining
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

// ImportState takes the WAF ID. Read then pulls every exception of the WAF.
func (r *wafRuleSetsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	wafID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected the WAF ID, got "+req.ID)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("waf_id"), wafID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// reconcile converges the WAF from current to planned and returns the resulting
// exceptions. Deletes run first so a name freed by a delete can be reused by a
// create. On the first failure it stops and returns what was applied so far.
func (r *wafRuleSetsResource) reconcile(ctx context.Context, wafID int64, planned, current map[string]WafRuleSetsItemModel) (map[string]WafRuleSetsItemModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	result := make(map[string]WafRuleSetsItemModel, len(current))
	for key, item := range current {
		result[key] = item
	}

	ops := planWafRuleSetsOps(planned, current)

	for _, key := range ops.Deletes {
		if err := r.deleteException(ctx, wafID, current[key].ExceptionID.ValueInt64()); err != nil {
			diags.AddError(fmt.Sprintf("Could not delete WAF exception %q", key), err.Error())
			return result, diags
		}
		delete(result, key)
	}

	for _, key := range ops.Updates {
		prior := current[key]
		request := buildWafRuleSetsRequest(key, planned[key], &prior)
		rule, statusCode, err := r.updateException(ctx, wafID, prior.ExceptionID.ValueInt64(), request)
		if err != nil && statusCode == http.StatusNotFound {
			// Deleted outside Terraform since the last refresh: recreate it.
			rule, err = r.createException(ctx, wafID, request)
		}
		if err != nil {
			diags.AddError(fmt.Sprintf("Could not update WAF exception %q", key), err.Error())
			return result, diags
		}
		result[key] = wafRuleToItemModel(*rule)
	}

	for _, key := range ops.Creates {
		request := buildWafRuleSetsRequest(key, planned[key], nil)
		rule, err := r.createException(ctx, wafID, request)
		if err != nil {
			diags.AddError(fmt.Sprintf("Could not create WAF exception %q", key), err.Error())
			return result, diags
		}
		result[key] = wafRuleToItemModel(*rule)
	}

	return result, diags
}

// wafRuleSetsOps lists, by key and in a stable order, what reconcile does.
type wafRuleSetsOps struct {
	Deletes []string
	Updates []string
	Creates []string
}

func planWafRuleSetsOps(planned, current map[string]WafRuleSetsItemModel) wafRuleSetsOps {
	var ops wafRuleSetsOps

	for key := range current {
		if _, ok := planned[key]; !ok {
			ops.Deletes = append(ops.Deletes, key)
		}
	}

	for key, item := range planned {
		prior, ok := current[key]
		switch {
		case !ok:
			ops.Creates = append(ops.Creates, key)
		case wafRuleSetsItemNeedsUpdate(item, prior):
			ops.Updates = append(ops.Updates, key)
		}
	}

	sort.Strings(ops.Deletes)
	sort.Strings(ops.Updates)
	sort.Strings(ops.Creates)

	return ops
}

func wafRuleSetsHasChanges(planned, current map[string]WafRuleSetsItemModel) bool {
	ops := planWafRuleSetsOps(planned, current)
	return len(ops.Deletes)+len(ops.Updates)+len(ops.Creates) > 0
}

// wafRuleSetsItemNeedsUpdate reports whether a planned exception differs from
// the current one in a field the configuration declares. A planned value that is
// unknown is not declared — the API supplies it — so it never forces an update.
func wafRuleSetsItemNeedsUpdate(planned, current WafRuleSetsItemModel) bool {
	if differsWhenKnown(planned.RuleID, current.RuleID) ||
		differsWhenKnown(planned.Path, current.Path) ||
		differsWhenKnown(planned.Operator, current.Operator) ||
		differsWhenKnown(planned.Active, current.Active) {
		return true
	}

	if len(planned.Conditions) != len(current.Conditions) {
		return true
	}

	for i := range planned.Conditions {
		p, c := planned.Conditions[i].Condition, current.Conditions[i].Condition
		if p == nil || c == nil {
			if p != c {
				return true
			}
			continue
		}
		if differsWhenKnown(p.Match, c.Match) ||
			differsWhenKnown(p.ConditionType, c.ConditionType) ||
			differsWhenKnown(p.Name, c.Name) ||
			differsWhenKnown(p.Value, c.Value) {
			return true
		}
	}

	return false
}

func differsWhenKnown(planned, current attr.Value) bool {
	return !planned.IsUnknown() && !planned.Equal(current)
}

// buildWafRuleSetsState keys the exceptions read from the API by name. When
// several share a name, the one whose ID prior state already holds under that
// name keeps it — so a duplicate created outside Terraform is what gets planned
// for removal, not the managed exception — and the others are keyed
// "<name>#<exception_id>". It returns those keys.
func buildWafRuleSetsState(remote []azionapi.WAFRule, prior map[string]WafRuleSetsItemModel) (map[string]WafRuleSetsItemModel, []string) {
	result := make(map[string]WafRuleSetsItemModel, len(remote))
	var duplicates []string

	sorted := make([]azionapi.WAFRule, len(remote))
	copy(sorted, remote)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].GetId() < sorted[j].GetId() })

	placed := make(map[int64]bool, len(sorted))
	for _, rule := range sorted {
		if held, ok := prior[rule.GetName()]; ok && held.ExceptionID.ValueInt64() == rule.GetId() {
			result[rule.GetName()] = wafRuleToItemModel(rule)
			placed[rule.GetId()] = true
		}
	}

	for _, rule := range sorted {
		if placed[rule.GetId()] {
			continue
		}
		key := rule.GetName()
		if _, taken := result[key]; taken {
			key = fmt.Sprintf("%s%s%d", key, wafRuleSetsDuplicateSeparator, rule.GetId())
			duplicates = append(duplicates, key)
		}
		result[key] = wafRuleToItemModel(rule)
	}

	sort.Strings(duplicates)
	return result, duplicates
}

func isWafRuleSetsDuplicateKey(key string) bool {
	for i := len(key) - 1; i > 0; i-- {
		if key[i:i+1] == wafRuleSetsDuplicateSeparator {
			_, err := strconv.ParseInt(key[i+1:], 10, 64)
			return err == nil
		}
	}
	return false
}

func wafRuleToItemModel(rule azionapi.WAFRule) WafRuleSetsItemModel {
	single := transformWAFRuleToResourceModel(rule)
	return WafRuleSetsItemModel{
		ExceptionID:  single.ID,
		RuleID:       single.RuleID,
		Path:         single.Path,
		Conditions:   single.Conditions,
		Operator:     single.Operator,
		Active:       single.Active,
		LastEditor:   single.LastEditor,
		LastModified: single.LastModified,
	}
}

// buildWafRuleSetsRequest builds the PUT/POST body for one exception. The update
// endpoint is a PUT, so an optional field the configuration leaves to the API is
// sent with its current value rather than omitted, which could reset it.
func buildWafRuleSetsRequest(name string, item WafRuleSetsItemModel, prior *WafRuleSetsItemModel) azionapi.WAFRuleRequest {
	request := azionapi.NewWAFRuleRequest(name, buildWAFExceptionConditionsRequest(item.Conditions))

	ruleID, path, operator := item.RuleID, item.Path, item.Operator
	if prior != nil {
		if ruleID.IsUnknown() {
			ruleID = prior.RuleID
		}
		if path.IsUnknown() {
			path = prior.Path
		}
		if operator.IsUnknown() {
			operator = prior.Operator
		}
	}

	if !ruleID.IsNull() && !ruleID.IsUnknown() {
		request.SetRuleId(ruleID.ValueInt64())
	}
	if !path.IsNull() && !path.IsUnknown() {
		request.SetPath(path.ValueString())
	}
	if !operator.IsNull() && !operator.IsUnknown() {
		request.SetOperator(operator.ValueString())
	}
	if !item.Active.IsNull() && !item.Active.IsUnknown() {
		request.SetActive(item.Active.ValueBool())
	}

	return *request
}

// listAllExceptions reads every page of the WAF's exceptions. The returned
// status code is that of the failed request, or 0.
func (r *wafRuleSetsResource) listAllExceptions(ctx context.Context, wafID int64) ([]azionapi.WAFRule, int, error) {
	var rules []azionapi.WAFRule

	for page := int64(1); page <= wafRuleSetsMaxPages; page++ {
		list, response, err := utils.RetryOn429(func() (*azionapi.PaginatedWAFRuleList, *http.Response, error) {
			return r.client.api.WAFsExceptionsAPI.ListWafExceptions(ctx, wafID).Page(page).PageSize(wafRuleSetsPageSize).Execute()
		}, 5)
		if err != nil {
			return nil, utils.StatusCodeOf(response), wafRuleSetsAPIError(err, response)
		}
		if response != nil {
			_ = response.Body.Close()
		}

		results := list.GetResults()
		rules = append(rules, results...)

		if len(results) == 0 || (list.GetNext() == "" && page >= list.GetTotalPages()) {
			return rules, 0, nil
		}
	}

	return nil, 0, fmt.Errorf("WAF %d: more than %d pages of exceptions", wafID, wafRuleSetsMaxPages)
}

func (r *wafRuleSetsResource) createException(ctx context.Context, wafID int64, request azionapi.WAFRuleRequest) (*azionapi.WAFRule, error) {
	created, response, err := utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
		return r.client.api.WAFsExceptionsAPI.CreateWafException(ctx, wafID).WAFRuleRequest(request).Execute()
	}, 5)
	if err != nil {
		return nil, wafRuleSetsAPIError(err, response)
	}
	if response != nil {
		_ = response.Body.Close()
	}

	rule := created.GetData()
	return &rule, nil
}

func (r *wafRuleSetsResource) updateException(ctx context.Context, wafID, exceptionID int64, request azionapi.WAFRuleRequest) (*azionapi.WAFRule, int, error) {
	updated, response, err := utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
		return r.client.api.WAFsExceptionsAPI.UpdateWafException(ctx, exceptionID, wafID).WAFRuleRequest(request).Execute()
	}, 5)
	if err != nil {
		return nil, utils.StatusCodeOf(response), wafRuleSetsAPIError(err, response)
	}
	if response != nil {
		_ = response.Body.Close()
	}

	rule := updated.GetData()
	return &rule, 0, nil
}

// deleteException treats an exception that is already gone as deleted.
func (r *wafRuleSetsResource) deleteException(ctx context.Context, wafID, exceptionID int64) error {
	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.WAFsExceptionsAPI.DeleteWafException(ctx, exceptionID, wafID).Execute()
	}, 5)
	if err != nil {
		if utils.StatusCodeOf(response) == http.StatusNotFound {
			if response.Body != nil {
				_ = response.Body.Close()
			}
			return nil
		}
		return wafRuleSetsAPIError(err, response)
	}
	if response != nil {
		_ = response.Body.Close()
	}
	return nil
}

// wafRuleSetsAPIError joins the SDK error with the response body, which carries the API's
// explanation. ReadAPIErrorBody closes the body.
func wafRuleSetsAPIError(err error, response *http.Response) error {
	return fmt.Errorf("%w: %s", err, utils.ReadAPIErrorBody(response))
}
