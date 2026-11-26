package provider

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
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
// Each exception is one `rule_set` block, identified by its `name`. Internally
// the exceptions are handled as a map keyed by name. Names are not documented as
// unique per WAF, so when the API holds several exceptions with the same name,
// only one can take the name as its key; the rest are keyed (and shown in state
// with the name) "<name>#<exception_id>", which no configuration declares, so
// the plan removes them.
//
// Schema version 0 (v2.11.0) held the exceptions as a map nested attribute,
// `rule_sets = { "<name>" = { ... } }`. Version 1 holds them as a set of
// `rule_set` blocks; UpgradeState converts the old state (map key -> name).

var (
	_ resource.Resource                 = &wafRuleSetsResource{}
	_ resource.ResourceWithConfigure    = &wafRuleSetsResource{}
	_ resource.ResourceWithImportState  = &wafRuleSetsResource{}
	_ resource.ResourceWithModifyPlan   = &wafRuleSetsResource{}
	_ resource.ResourceWithUpgradeState = &wafRuleSetsResource{}
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
	ID       types.String           `tfsdk:"id"`
	WafID    types.Int64            `tfsdk:"waf_id"`
	RuleSets []WafRuleSetsItemModel `tfsdk:"rule_set"`
}

// WafRuleSetsItemModel is one exception, one `rule_set` block.
type WafRuleSetsItemModel struct {
	Name        types.String                        `tfsdk:"name"`
	ExceptionID types.Int64                         `tfsdk:"exception_id"`
	RuleID      types.Int64                         `tfsdk:"rule_id"`
	Path        types.String                        `tfsdk:"path"`
	Conditions  []WafExceptionConditionWrapperModel `tfsdk:"conditions"`
	Operator    types.String                        `tfsdk:"operator"`
	Active      types.Bool                          `tfsdk:"active"`
}

// wafRuleSetsByName indexes the exceptions by name, the key reconcile works
// with. The schema validator rejects duplicate names in configuration, and
// buildWafRuleSetsState gives every exception read from the API a distinct
// name, so no entry is lost.
func wafRuleSetsByName(items []WafRuleSetsItemModel) map[string]WafRuleSetsItemModel {
	result := make(map[string]WafRuleSetsItemModel, len(items))
	for _, item := range items {
		result[item.Name.ValueString()] = item
	}
	return result
}

// wafRuleSetsList is the inverse of wafRuleSetsByName, sorted by name so state
// is written in a stable order. The name is taken from the key.
func wafRuleSetsList(items map[string]WafRuleSetsItemModel) []WafRuleSetsItemModel {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]WafRuleSetsItemModel, 0, len(items))
	for _, name := range names {
		item := items[name]
		item.Name = types.StringValue(name)
		result = append(result, item)
	}
	return result
}

func (r *wafRuleSetsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_sets"
}

func (r *wafRuleSetsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version 1: rule_sets (map nested attribute) became rule_set blocks.
		Version:     1,
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
		},
		Blocks: map[string]schema.Block{
			"rule_set": wafRuleSetsBlock(),
		},
	}
}

// wafRuleSetsBlock is the repeated `rule_set` block. It is a set, not a list:
// the exceptions of a WAF have no order (the API lists them by ID), so a list
// would diff whenever the configuration order differs from the order state was
// written in, and reordering blocks would plan changes that apply nothing.
// Elements are matched by name (ModifyPlan, reconcile), not by position.
func wafRuleSetsBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "One exception of the WAF. Every exception of the WAF must be declared: those without a `rule_set` block are deleted on apply, and declaring no `rule_set` block at all deletes every exception of the WAF. `name` must be unique.",
		Validators: []validator.Set{
			wafRuleSetsNamesValidator{},
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Description: "Name of the WAF exception. Identifies the exception, so it must be unique among the `rule_set` blocks.",
					Required:    true,
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(1),
					},
				},
				// No UseStateForUnknown here: it only skips resources with no
				// state at all, so for an exception added to an existing
				// resource it copies a null prior value into the plan, and apply
				// fails with "inconsistent result after apply". ModifyPlan
				// carries the ID over for names state already holds.
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
			},
			Blocks: map[string]schema.Block{
				"conditions": wafExceptionConditionsBlock(
					"Required. Conditions for the WAF exception, one `conditions` block per condition, each holding a single `condition` block. At least one is required.",
					[]validator.String{
						stringvalidator.OneOf("generic", "specific_on_name", "specific_on_value"),
					},
				),
			},
		},
	}
}

// wafRuleSetsNamesValidator rejects two `rule_set` blocks with the same name,
// and names using the "<name>#<exception_id>" form reserved for duplicate-named
// exceptions read from the API, so a declared name can never collide with one.
type wafRuleSetsNamesValidator struct{}

func (wafRuleSetsNamesValidator) Description(_ context.Context) string {
	return "rule_set names must be unique and must not have the form <name>#<number>"
}

func (v wafRuleSetsNamesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (wafRuleSetsNamesValidator) ValidateSet(_ context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	seen := make(map[string]bool)
	for _, element := range req.ConfigValue.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			continue
		}
		name, ok := object.Attributes()["name"].(types.String)
		if !ok || name.IsNull() || name.IsUnknown() {
			continue
		}
		namePath := req.Path.AtSetValue(element).AtName("name")

		if isWafRuleSetsDuplicateKey(name.ValueString()) {
			resp.Diagnostics.AddAttributeError(
				namePath,
				"Reserved exception name",
				fmt.Sprintf("%q matches the \"<name>%s<exception_id>\" form used for duplicate-named exceptions read from the API. Choose a different name.", name.ValueString(), wafRuleSetsDuplicateSeparator),
			)
		}
		if seen[name.ValueString()] {
			resp.Diagnostics.AddAttributeError(
				namePath,
				"Duplicate exception name",
				fmt.Sprintf("More than one rule_set block is named %q. Each exception is identified by its name, so names must be unique.", name.ValueString()),
			)
		}
		seen[name.ValueString()] = true
	}
}

func (r *wafRuleSetsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

// ModifyPlan keeps the plan to what actually changes, matching exceptions to
// state by name.
//
// An exception whose declared fields match state is left untouched by apply, so
// its state is the exact plan. One that apply creates or updates gets the values
// the API supplies planned as unknown. The framework's own unknown marking cannot
// be relied on here: in a set block it locates each element in the configuration
// by value, and an element whose value differs from the configuration (a
// default applied, a prior value filled in) is not found and keeps its nulls,
// which apply would then contradict.
func (r *wafRuleSetsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var planRuleSets types.Set
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("rule_set"), &planRuleSets)...)
	if resp.Diagnostics.HasError() || planRuleSets.IsUnknown() {
		return
	}

	var plan, state, config WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// A name not known until apply cannot be matched to state.
	for _, planned := range plan.RuleSets {
		if planned.Name.IsUnknown() {
			return
		}
	}

	planned := wafRuleSetsByName(plan.RuleSets)
	current := wafRuleSetsByName(state.RuleSets)
	declared := wafRuleSetsByName(config.RuleSets)
	for name, item := range planned {
		// The framework applies active's default by finding the element in the
		// configuration by value. A proposed element holding prior values is
		// not equal to the configured one, so the lookup fails and a declared
		// `active = false` is planned as the default. Take it by name instead.
		if configured, ok := declared[name]; ok && !configured.Active.IsUnknown() {
			item.Active = configured.Active
			if item.Active.IsNull() {
				item.Active = types.BoolValue(defaultWAFRuleActive)
			}
		}

		held, ok := current[name]
		switch {
		case !ok:
			// Created by apply: the API assigns the ID.
			planned[name] = wafRuleSetsItemPlannedForApply(item, types.Int64Unknown())
		case wafRuleSetsItemNeedsUpdate(item, held):
			planned[name] = wafRuleSetsItemPlannedForApply(item, held.ExceptionID)
		default:
			planned[name] = held
		}
	}
	plan.RuleSets = wafRuleSetsList(planned)

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// wafRuleSetsItemPlannedForApply plans an exception that apply writes: what the
// configuration leaves to the API, and what the API sets on every write, is
// unknown until apply. A null planned value is one the configuration does not
// declare: Optional+Computed attributes in a block are null only then.
func wafRuleSetsItemPlannedForApply(item WafRuleSetsItemModel, exceptionID types.Int64) WafRuleSetsItemModel {
	item.ExceptionID = exceptionID
	if item.RuleID.IsNull() {
		item.RuleID = types.Int64Unknown()
	}
	if item.Path.IsNull() {
		item.Path = types.StringUnknown()
	}
	if item.Operator.IsNull() {
		item.Operator = types.StringUnknown()
	}

	conditions := make([]WafExceptionConditionWrapperModel, len(item.Conditions))
	for i, wrapper := range item.Conditions {
		if wrapper.Condition != nil {
			condition := *wrapper.Condition
			if condition.Name.IsNull() {
				condition.Name = types.StringUnknown()
			}
			if condition.Value.IsNull() {
				condition.Value = types.StringUnknown()
			}
			wrapper.Condition = &condition
		}
		conditions[i] = wrapper
	}
	if item.Conditions != nil {
		item.Conditions = conditions
	}

	return item
}

func (r *wafRuleSetsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wafID := plan.WafID.ValueInt64()
	planned := wafRuleSetsByName(plan.RuleSets)

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
		if _, declared := planned[key]; declared {
			adopted[key] = item
		} else {
			undeclared = append(undeclared, key)
		}
	}

	result, diags := r.reconcile(ctx, wafID, planned, adopted)
	resp.Diagnostics.Append(diags...)

	// Saved even on a partial failure, so state holds what was applied.
	plan.ID = types.StringValue(strconv.FormatInt(wafID, 10))
	plan.RuleSets = wafRuleSetsList(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	if len(undeclared) > 0 {
		sort.Strings(undeclared)
		resp.Diagnostics.AddWarning(
			"WAF has exceptions not declared in the configuration",
			fmt.Sprintf("WAF %d already holds exceptions this configuration does not declare: %v. They were left in place; the next plan shows them for removal. Declare them in rule_set blocks to keep them.", wafID, undeclared),
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

	ruleSets, duplicates := buildWafRuleSetsState(remote, wafRuleSetsByName(state.RuleSets))
	if len(duplicates) > 0 {
		resp.Diagnostics.AddWarning(
			"WAF has exceptions with duplicate names",
			fmt.Sprintf("WAF %d holds more than one exception with the same name. The extra ones are named %v in state and are planned for removal.", wafID, duplicates),
		)
	}

	state.ID = types.StringValue(strconv.FormatInt(wafID, 10))
	state.RuleSets = wafRuleSetsList(ruleSets)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wafRuleSetsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WafRuleSetsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, diags := r.reconcile(ctx, plan.WafID.ValueInt64(), wafRuleSetsByName(plan.RuleSets), wafRuleSetsByName(state.RuleSets))
	resp.Diagnostics.Append(diags...)

	// Saved even on a partial failure, so state holds what was applied.
	plan.ID = types.StringValue(strconv.FormatInt(plan.WafID.ValueInt64(), 10))
	plan.RuleSets = wafRuleSetsList(result)
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

	remaining, diags := r.reconcile(ctx, state.WafID.ValueInt64(), map[string]WafRuleSetsItemModel{}, wafRuleSetsByName(state.RuleSets))
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		// Keep the exceptions that could not be deleted in state.
		state.RuleSets = wafRuleSetsList(remaining)
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

// wafRuleSetsResourceModelV0 is the state of schema version 0 (v2.11.0), where
// the exceptions were a map keyed by name.
type wafRuleSetsResourceModelV0 struct {
	ID          types.String                      `tfsdk:"id"`
	WafID       types.Int64                       `tfsdk:"waf_id"`
	LastUpdated types.String                      `tfsdk:"last_updated"`
	RuleSets    map[string]wafRuleSetsItemModelV0 `tfsdk:"rule_sets"`
}

type wafRuleSetsItemModelV0 struct {
	ExceptionID  types.Int64                         `tfsdk:"exception_id"`
	RuleID       types.Int64                         `tfsdk:"rule_id"`
	Path         types.String                        `tfsdk:"path"`
	Conditions   []WafExceptionConditionWrapperModel `tfsdk:"conditions"`
	Operator     types.String                        `tfsdk:"operator"`
	Active       types.Bool                          `tfsdk:"active"`
	LastEditor   types.String                        `tfsdk:"last_editor"`
	LastModified types.String                        `tfsdk:"last_modified"`
}

// wafRuleSetsSchemaV0 is the schema version 0 shape, as far as decoding its
// state needs: types and Required/Optional/Computed only.
func wafRuleSetsSchemaV0() schema.Schema {
	condition := schema.SingleNestedAttribute{
		Required: true,
		Attributes: map[string]schema.Attribute{
			"match":          schema.StringAttribute{Required: true},
			"name":           schema.StringAttribute{Optional: true, Computed: true},
			"value":          schema.StringAttribute{Optional: true, Computed: true},
			"condition_type": schema.StringAttribute{Required: true},
		},
	}

	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true},
			"waf_id":       schema.Int64Attribute{Required: true},
			"last_updated": schema.StringAttribute{Computed: true},
			"rule_sets": schema.MapNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"exception_id": schema.Int64Attribute{Computed: true},
						"rule_id":      schema.Int64Attribute{Optional: true, Computed: true},
						"path":         schema.StringAttribute{Optional: true, Computed: true},
						"conditions": schema.ListNestedAttribute{
							Required: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{"condition": condition},
							},
						},
						"operator":      schema.StringAttribute{Optional: true, Computed: true},
						"active":        schema.BoolAttribute{Optional: true, Computed: true},
						"last_editor":   schema.StringAttribute{Computed: true},
						"last_modified": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

// UpgradeState converts version 0 state, `rule_sets = { "<name>" = {...} }`, to
// `rule_set` blocks: each map key becomes the block's `name`, and every other
// value is carried over unchanged, so a configuration rewritten to blocks plans
// no changes.
func (r *wafRuleSetsResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := wafRuleSetsSchemaV0()

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior wafRuleSetsResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				upgraded := upgradeWafRuleSetsStateV0(prior)
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

func upgradeWafRuleSetsStateV0(prior wafRuleSetsResourceModelV0) WafRuleSetsResourceModel {
	items := make(map[string]WafRuleSetsItemModel, len(prior.RuleSets))
	for name, item := range prior.RuleSets {
		items[name] = WafRuleSetsItemModel{
			ExceptionID: item.ExceptionID,
			RuleID:      item.RuleID,
			Path:        item.Path,
			Conditions:  item.Conditions,
			Operator:    item.Operator,
			Active:      item.Active,
		}
	}

	return WafRuleSetsResourceModel{
		ID:       prior.ID,
		WafID:    prior.WafID,
		RuleSets: wafRuleSetsList(items),
	}
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
		Name:        single.Name,
		ExceptionID: single.ID,
		RuleID:      single.RuleID,
		Path:        single.Path,
		Conditions:  single.Conditions,
		Operator:    single.Operator,
		Active:      single.Active,
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
