package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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
// attributes the configuration leaves null taken from prior state — for map
// keys prior state holds. A new key has no prior value, so they stay null.
func proposedNewState(config, prior WafRuleSetsResourceModel) WafRuleSetsResourceModel {
	proposed := config
	proposed.ID = prior.ID
	proposed.LastUpdated = prior.LastUpdated
	proposed.RuleSets = map[string]WafRuleSetsItemModel{}

	for key, item := range config.RuleSets {
		if held, ok := prior.RuleSets[key]; ok {
			item.ExceptionID = held.ExceptionID
			item.LastEditor = held.LastEditor
			item.LastModified = held.LastModified
			if item.Active.IsNull() {
				item.Active = held.Active
			}
		}
		proposed.RuleSets[key] = item
	}

	return proposed
}

func planThroughFramework(t *testing.T, config, prior WafRuleSetsResourceModel) WafRuleSetsResourceModel {
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

	server := providerserver.NewProtocol6(wafRuleSetsTestProvider{})()
	resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "azion_waf_rule_sets",
		Config:           toDynamic(config),
		PriorState:       toDynamic(prior),
		ProposedNewState: toDynamic(proposedNewState(config, prior)),
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

	if got := planned.RuleSets["recreate"].ExceptionID; !got.IsUnknown() {
		t.Errorf("new exception: exception_id planned as %s, want unknown", got)
	}
	if got := planned.RuleSets["recreate"].LastModified; !got.IsUnknown() {
		t.Errorf("new exception: last_modified planned as %s, want unknown", got)
	}
	if got := planned.RuleSets["kept"]; !got.ExceptionID.Equal(types.Int64Value(1)) || got.LastModified.IsUnknown() {
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

	got := planned.RuleSets["changed"]
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

	if !planned.LastUpdated.Equal(prior.LastUpdated) || !planned.RuleSets["a"].LastModified.Equal(prior.RuleSets["a"].LastModified) {
		t.Errorf("plan should equal prior state, got %+v", planned)
	}
}
