package provider

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// wafRuleSetsTestProvider serves only azion_waf_rule_sets, so a plan can run
// through the framework's real PlanResourceChange pipeline — defaults, computed
// nil marking, attribute plan modifiers and then ModifyPlan — instead of
// calling ModifyPlan with a hand-built plan.
type wafRuleSetsTestProvider struct{}

func (wafRuleSetsTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azion"
}
func (wafRuleSetsTestProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {
}
func (wafRuleSetsTestProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (wafRuleSetsTestProvider) Resources(context.Context) []func() fwresource.Resource {
	return []func() fwresource.Resource{WafRuleSetsResource}
}
func (wafRuleSetsTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

// configItem is an exception as written in configuration: computed attributes
// are null, and so are the optional ones it does not declare.
func configItem(path string) WafRuleSetsItemModel {
	return WafRuleSetsItemModel{
		ExceptionID:  types.Int64Null(),
		RuleID:       types.Int64Value(0),
		Path:         types.StringValue(path),
		Conditions:   []WafExceptionConditionWrapperModel{genericCondition("any_url")},
		Operator:     types.StringValue("regex"),
		Active:       types.BoolNull(),
		LastEditor:   types.StringNull(),
		LastModified: types.StringNull(),
	}
}

// proposedNewState mimics Terraform core: configuration values, with computed
// attributes the configuration leaves null taken from prior state. rule_set is
// a set block, which core correlates with prior elements only by value, so this
// only fills in prior values for an exception whose declared fields equal the
// prior ones; any other exception keeps them null.
func proposedNewState(config, prior WafRuleSetsResourceModel) WafRuleSetsResourceModel {
	proposed := config
	proposed.ID = prior.ID
	proposed.LastUpdated = prior.LastUpdated
	proposed.RuleSets = nil

	held := wafRuleSetsByName(prior.RuleSets)
	for _, item := range config.RuleSets {
		if p, ok := held[item.Name.ValueString()]; ok && !wafRuleSetsItemNeedsUpdate(item, p) {
			item.ExceptionID = p.ExceptionID
			item.LastEditor = p.LastEditor
			item.LastModified = p.LastModified
			if item.Active.IsNull() {
				item.Active = p.Active
			}
		}
		proposed.RuleSets = append(proposed.RuleSets, item)
	}

	return proposed
}

func planThroughFramework(t *testing.T, config, prior WafRuleSetsResourceModel) WafRuleSetsResourceModel {
	t.Helper()
	ctx := context.Background()

	sr := &fwresource.SchemaResponse{}
	WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
	state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("encoding: %v", diags)
	}
	return planWithPrior(t, config, state.Raw, proposedNewState(config, prior))
}

// planWithPrior runs PlanResourceChange with a raw prior state, so a create
// (null prior state) can be planned as well.
func planWithPrior(t *testing.T, config WafRuleSetsResourceModel, prior tftypes.Value, proposed WafRuleSetsResourceModel) WafRuleSetsResourceModel {
	t.Helper()
	ctx := context.Background()

	sr := &fwresource.SchemaResponse{}
	WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
	s := sr.Schema
	typ := s.Type().TerraformType(ctx)

	toDynamic := func(m WafRuleSetsResourceModel) *tfprotov6.DynamicValue {
		state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(typ, nil)}
		if diags := state.Set(ctx, &m); diags.HasError() {
			t.Fatalf("encoding: %v", diags)
		}
		dv, err := tfprotov6.NewDynamicValue(typ, state.Raw)
		if err != nil {
			t.Fatalf("encoding: %v", err)
		}
		return &dv
	}

	priorDV, err := tfprotov6.NewDynamicValue(typ, prior)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	priorState := &priorDV

	server := providerserver.NewProtocol6(wafRuleSetsTestProvider{})()
	resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "azion_waf_rule_sets",
		Config:           toDynamic(config),
		PriorState:       priorState,
		ProposedNewState: toDynamic(proposed),
	})
	if err != nil {
		t.Fatalf("PlanResourceChange: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("PlanResourceChange: %s: %s", d.Summary, d.Detail)
		}
	}

	raw, err := resp.PlannedState.Unmarshal(typ)
	if err != nil {
		t.Fatalf("decoding plan: %v", err)
	}
	var planned WafRuleSetsResourceModel
	if diags := (tfsdk.Plan{Schema: s, Raw: raw}).Get(ctx, &planned); diags.HasError() {
		t.Fatalf("decoding plan: %v", diags)
	}
	return planned
}

// Regression: an exception added to an existing resource — declared in the
// configuration, or deleted outside Terraform and being recreated — must plan
// exception_id as unknown. UseStateForUnknown only skips resources with no
// state at all, so on a new map key it copied the element's null prior value
// into the plan, and apply failed with "Provider produced inconsistent result
// after apply: .rule_sets[...].exception_id: was null, but now ...".
func TestWAFRuleSetsPlanNewExceptionOnExistingResource(t *testing.T) {
	prior := wafRuleSetsModel(types.StringValue("before"), map[string]WafRuleSetsItemModel{
		"kept": wafItem(1, "/kept", true),
	})
	config := wafRuleSetsModel(types.StringNull(), map[string]WafRuleSetsItemModel{
		"kept":     configItem("/kept"),
		"recreate": configItem("/new"),
	})
	config.ID = types.StringNull()

	planned := planThroughFramework(t, config, prior)

	byName := wafRuleSetsByName(planned.RuleSets)
	if got := byName["recreate"].ExceptionID; !got.IsUnknown() {
		t.Errorf("new exception: exception_id planned as %s, want unknown", got)
	}
	if got := byName["recreate"].LastModified; !got.IsUnknown() {
		t.Errorf("new exception: last_modified planned as %s, want unknown", got)
	}
	if got := byName["kept"]; !got.ExceptionID.Equal(types.Int64Value(1)) || got.LastModified.IsUnknown() {
		t.Errorf("unchanged exception should plan as its state, got %+v", got)
	}
}

func TestWAFRuleSetsPlanChangedExceptionKeepsID(t *testing.T) {
	prior := wafRuleSetsModel(types.StringValue("before"), map[string]WafRuleSetsItemModel{
		"changed": wafItem(1, "/old", true),
	})
	config := wafRuleSetsModel(types.StringNull(), map[string]WafRuleSetsItemModel{
		"changed": configItem("/new"),
	})
	config.ID = types.StringNull()

	planned := planThroughFramework(t, config, prior)

	got := wafRuleSetsByName(planned.RuleSets)["changed"]
	if !got.ExceptionID.Equal(types.Int64Value(1)) {
		t.Errorf("updated exception must keep its exception_id, got %s", got.ExceptionID)
	}
	if !got.LastModified.IsUnknown() {
		t.Errorf("updated exception: last_modified planned as %s, want unknown", got.LastModified)
	}
}

func TestWAFRuleSetsPlanNoChangesIsEmpty(t *testing.T) {
	prior := wafRuleSetsModel(types.StringValue("before"), map[string]WafRuleSetsItemModel{
		"a": wafItem(1, "/a", true),
	})
	config := wafRuleSetsModel(types.StringNull(), map[string]WafRuleSetsItemModel{
		"a": configItem("/a"),
	})
	config.ID = types.StringNull()

	planned := planThroughFramework(t, config, prior)

	if !planned.LastUpdated.Equal(prior.LastUpdated) || !reflect.DeepEqual(planned.RuleSets, prior.RuleSets) {
		t.Errorf("plan should equal prior state, got %+v", planned)
	}
}

// Regression: the framework applies `active`'s default by looking the element
// up in the configuration by value. In a set block the proposed element (prior
// values filled in by core) is not equal to the configured one, so the lookup
// found nothing and `active = false` was planned as the default `true`.
func TestWAFRuleSetsPlanKeepsDeclaredActiveFalse(t *testing.T) {
	prior := wafRuleSetsModel(types.StringValue("before"), map[string]WafRuleSetsItemModel{
		"off": wafItem(1, "/off", false),
		"on":  wafItem(2, "/on", true),
	})
	off := configItem("/off")
	off.Active = types.BoolValue(false)
	config := wafRuleSetsModel(types.StringNull(), map[string]WafRuleSetsItemModel{
		"off": off,
		"on":  configItem("/on"),
	})
	config.ID = types.StringNull()

	planned := planThroughFramework(t, config, prior)

	if !reflect.DeepEqual(planned.RuleSets, prior.RuleSets) {
		t.Errorf("plan should equal prior state, got %+v", planned.RuleSets)
	}

	// Changing only active must plan exactly that value.
	off.Active = types.BoolValue(true)
	on := configItem("/on")
	on.Active = types.BoolValue(false)
	config = wafRuleSetsModel(types.StringNull(), map[string]WafRuleSetsItemModel{"off": off, "on": on})
	config.ID = types.StringNull()

	byName := wafRuleSetsByName(planThroughFramework(t, config, prior).RuleSets)
	if got := byName["off"].Active; !got.Equal(types.BoolValue(true)) {
		t.Errorf("off: active planned as %s, want true", got)
	}
	if got := byName["on"].Active; !got.Equal(types.BoolValue(false)) {
		t.Errorf("on: active planned as %s, want false", got)
	}
}

// Declaring no rule_set block deletes every exception: the plan holds no
// exception, and reconcile deletes everything state holds.
func TestWAFRuleSetsPlanZeroBlocksDeletesAll(t *testing.T) {
	prior := wafRuleSetsModel(types.StringValue("before"), map[string]WafRuleSetsItemModel{
		"a": wafItem(1, "/a", true),
		"b": wafItem(2, "/b", true),
	})
	config := wafRuleSetsModel(types.StringNull(), nil)
	config.ID = types.StringNull()

	planned := planThroughFramework(t, config, prior)

	if len(planned.RuleSets) != 0 {
		t.Errorf("zero rule_set blocks must plan no exception, got %+v", planned.RuleSets)
	}
	ops := planWafRuleSetsOps(wafRuleSetsByName(planned.RuleSets), wafRuleSetsByName(prior.RuleSets))
	if !reflect.DeepEqual(ops.Deletes, []string{"a", "b"}) || len(ops.Updates)+len(ops.Creates) != 0 {
		t.Errorf("ops = %+v, want both deleted", ops)
	}
}

// On create there is no prior state; the values the API supplies must still be
// planned unknown, or apply contradicts the plan.
func TestWAFRuleSetsPlanCreateMarksAPIValuesUnknown(t *testing.T) {
	ctx := context.Background()
	item := configItem("/a")
	item.RuleID = types.Int64Null()
	item.Operator = types.StringNull()
	config := wafRuleSetsModel(types.StringNull(), map[string]WafRuleSetsItemModel{"a": item})
	config.ID = types.StringNull()

	sr := &fwresource.SchemaResponse{}
	WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
	typ := sr.Schema.Type().TerraformType(ctx)
	nullState := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(typ, nil)}
	planned := planWithPrior(t, config, nullState.Raw, config)

	got := wafRuleSetsByName(planned.RuleSets)["a"]
	if !got.ExceptionID.IsUnknown() || !got.RuleID.IsUnknown() || !got.Operator.IsUnknown() || !got.LastModified.IsUnknown() {
		t.Errorf("undeclared values must be unknown on create, got %+v", got)
	}
	if !got.Path.Equal(types.StringValue("/a")) || !got.Active.Equal(types.BoolValue(true)) {
		t.Errorf("declared path and default active must be kept, got %+v", got)
	}
}

func TestWAFRuleSetsNamesValidator(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		names []string
		want  int
	}{
		"unique":           {[]string{"a", "b"}, 0},
		"duplicate name":   {[]string{"a", "a"}, 1},
		"reserved form":    {[]string{"a#12"}, 1},
		"no blocks at all": {nil, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			items := make([]WafRuleSetsItemModel, 0, len(tc.names))
			for i, n := range tc.names {
				// Distinct paths, so equal names are still distinct set elements.
				item := configItem(fmt.Sprintf("/%d", i))
				item.Name = types.StringValue(n)
				items = append(items, item)
			}
			model := wafRuleSetsModel(types.StringNull(), nil)
			model.RuleSets = items

			sr := &fwresource.SchemaResponse{}
			WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
			state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatalf("encoding: %v", diags)
			}
			var set types.Set
			if diags := state.GetAttribute(ctx, path.Root("rule_set"), &set); diags.HasError() {
				t.Fatalf("decoding: %v", diags)
			}

			resp := &validator.SetResponse{}
			wafRuleSetsNamesValidator{}.ValidateSet(ctx, validator.SetRequest{Path: path.Root("rule_set"), ConfigValue: set}, resp)
			if got := resp.Diagnostics.ErrorsCount(); got != tc.want {
				t.Errorf("errors = %d, want %d: %v", got, tc.want, resp.Diagnostics)
			}
		})
	}
}

// Version 0 state (v2.11.0), `rule_sets = { "<name>" = {...} }`, upgrades
// through the framework's real UpgradeResourceState to rule_set blocks named by
// the map key, with every other value unchanged.
func TestWAFRuleSetsUpgradeStateV0(t *testing.T) {
	ctx := context.Background()

	v0 := `{
	  "id": "42", "waf_id": 42, "last_updated": "Monday, 01-Jan-26 00:00:00 UTC",
	  "rule_sets": {
	    "allow-api": {
	      "exception_id": 7, "rule_id": 0, "path": "/api", "operator": "regex", "active": true,
	      "last_editor": "someone", "last_modified": "2026-01-01T00:00:00Z",
	      "conditions": [
	        {"condition": {"match": "any_url", "condition_type": "generic", "name": null, "value": null}},
	        {"condition": {"match": "specific_query_string_name", "condition_type": "specific_on_name", "name": "q", "value": null}}
	      ]
	    },
	    "second": {
	      "exception_id": 8, "rule_id": 1010, "path": "", "operator": "contains", "active": false,
	      "last_editor": "someone", "last_modified": "2026-01-01T00:00:00Z",
	      "conditions": [{"condition": {"match": "any_url", "condition_type": "generic", "name": null, "value": null}}]
	    }
	  }
	}`

	server := providerserver.NewProtocol6(wafRuleSetsTestProvider{})()
	resp, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "azion_waf_rule_sets",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: []byte(v0)},
	})
	if err != nil {
		t.Fatalf("UpgradeResourceState: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("UpgradeResourceState: %s: %s", d.Summary, d.Detail)
		}
	}

	sr := &fwresource.SchemaResponse{}
	WafRuleSetsResource().Schema(ctx, fwresource.SchemaRequest{}, sr)
	raw, err := resp.UpgradedState.Unmarshal(sr.Schema.Type().TerraformType(ctx))
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	var got WafRuleSetsResourceModel
	if diags := (tfsdk.State{Schema: sr.Schema, Raw: raw}).Get(ctx, &got); diags.HasError() {
		t.Fatalf("decoding: %v", diags)
	}

	if !got.ID.Equal(types.StringValue("42")) || !got.WafID.Equal(types.Int64Value(42)) || got.LastUpdated.ValueString() == "" {
		t.Errorf("top-level values not carried over: %+v", got)
	}
	byName := wafRuleSetsByName(got.RuleSets)
	if len(byName) != 2 {
		t.Fatalf("want 2 rule_set blocks, got %+v", got.RuleSets)
	}
	first := byName["allow-api"]
	if !first.Name.Equal(types.StringValue("allow-api")) || first.ExceptionID.ValueInt64() != 7 ||
		first.Path.ValueString() != "/api" || first.Operator.ValueString() != "regex" || !first.Active.ValueBool() ||
		len(first.Conditions) != 2 || first.Conditions[1].Condition.Name.ValueString() != "q" ||
		!first.Conditions[0].Condition.Name.IsNull() {
		t.Errorf("allow-api not carried over: %+v", first)
	}
	second := byName["second"]
	if second.ExceptionID.ValueInt64() != 8 || second.RuleID.ValueInt64() != 1010 || second.Active.ValueBool() || second.Operator.ValueString() != "contains" {
		t.Errorf("second not carried over: %+v", second)
	}
}

func TestWAFRuleSetsUpgradeStateV0EmptyMap(t *testing.T) {
	got := upgradeWafRuleSetsStateV0(wafRuleSetsResourceModelV0{
		ID:       types.StringValue("42"),
		WafID:    types.Int64Value(42),
		RuleSets: map[string]wafRuleSetsItemModelV0{},
	})
	if got.RuleSets == nil || len(got.RuleSets) != 0 {
		t.Errorf("an empty map must upgrade to no rule_set blocks, got %#v", got.RuleSets)
	}
}
