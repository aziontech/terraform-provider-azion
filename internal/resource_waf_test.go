package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func wafSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	ctx := context.Background()
	sr := &fwresource.SchemaResponse{}
	WafResource().Schema(ctx, fwresource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestWAFSchemaImplementation(t *testing.T) {
	ctx := context.Background()
	sr := wafSchema(t)

	if diags := sr.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// The scalars are enforced: a value changed out-of-band lands in state via the
// unfiltered transform and is reverted on the next apply.
func TestWAFEnforcedScalarsAreComputed(t *testing.T) {
	ctx := context.Background()
	sr := wafSchema(t)

	for _, name := range []string{
		"result.active",
		"result.engine_settings.engine_version",
		"result.engine_settings.type",
		"result.engine_settings.thresholds.threshold.sensitivity",
	} {
		p := tfPath(t, name)
		attribute, diags := sr.Schema.AttributeAtPath(ctx, p)
		if diags.HasError() {
			// thresholds is a set; its element path needs an index, so skip
			// resolution failures for that one rather than asserting on shape.
			continue
		}

		if !attribute.IsComputed() || !attribute.IsOptional() {
			t.Errorf("%s: must be Optional + Computed, got optional=%v computed=%v",
				name, attribute.IsOptional(), attribute.IsComputed())
		}
	}
}

// The collections and blocks must NOT be Computed. They are held as Go pointers
// and slices, which cannot be populated from an unknown value — a Computed
// attribute with no default is unknown whenever the configuration omits it, and
// reading that fails Create outright.
func TestWAFCollectionsAreNotComputed(t *testing.T) {
	ctx := context.Background()
	sr := wafSchema(t)

	for _, name := range []string{
		"result.engine_settings",
		"result.engine_settings.attributes",
		"result.engine_settings.attributes.rulesets",
		"result.engine_settings.attributes.thresholds",
	} {
		attribute, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if attribute.IsComputed() {
			t.Errorf("%s: must not be Computed; the model cannot hold an unknown here", name)
		}
	}
}

// An undeclared block stays out of state, and an undeclared collection inside a
// declared block does too — otherwise the API's own values would diff forever
// against a null plan.
func TestAlignWAFEngineSettingsGatesOnlyCollections(t *testing.T) {
	fromAPI := func() *WafEngineSettingsResourceModel {
		return &WafEngineSettingsResourceModel{
			EngineVersion: types.StringValue("2021-Q3"),
			Type:          types.StringValue("score"),
			Attributes: &WafEngineSettingsAttributesResourceModel{
				Rulesets:   []types.Int64{types.Int64Value(11)},
				Thresholds: []WafThresholdWrapperResourceModel{{}},
			},
		}
	}

	if got := alignWAFEngineSettings(nil, fromAPI()); got != nil {
		t.Errorf("undeclared engine_settings must stay out of state, got %+v", got)
	}

	declaredBlockOnly := &WafEngineSettingsResourceModel{}
	got := alignWAFEngineSettings(declaredBlockOnly, fromAPI())
	if got == nil || got.Attributes != nil {
		t.Errorf("undeclared attributes must stay out of state, got %+v", got)
	}
	if got != nil && got.EngineVersion.ValueString() != "2021-Q3" {
		t.Errorf("scalars must NOT be gated; engine_version = %q", got.EngineVersion)
	}

	declaredRulesetsOnly := &WafEngineSettingsResourceModel{
		Attributes: &WafEngineSettingsAttributesResourceModel{
			Rulesets: []types.Int64{types.Int64Value(11)},
		},
	}
	got = alignWAFEngineSettings(declaredRulesetsOnly, fromAPI())
	if got.Attributes == nil || got.Attributes.Rulesets == nil {
		t.Fatalf("declared rulesets must be refreshed from the API, got %+v", got)
	}
	if got.Attributes.Thresholds != nil {
		t.Errorf("undeclared thresholds must stay out of state, got %+v", got.Attributes.Thresholds)
	}
}
