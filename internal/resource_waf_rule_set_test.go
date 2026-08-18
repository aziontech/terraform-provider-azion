package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
)

func wafRuleSetSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	sr := &fwresource.SchemaResponse{}
	WafRuleSetResource().Schema(context.Background(), fwresource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestWAFRuleSetSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	if diags := wafRuleSetSchema(t).Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// Every optional field must be Computed. The transform populates these from the
// API unconditionally, so an Optional-only attribute leaves state holding a value
// the plan wants to null — a diff that reappears on every plan and never
// converges.
func TestWAFRuleSetOptionalFieldsAreComputed(t *testing.T) {
	ctx := context.Background()
	sr := wafRuleSetSchema(t)

	for _, name := range []string{
		"result.rule_id",
		"result.path",
		"result.operator",
		"result.active",
	} {
		attribute, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsOptional() {
			t.Errorf("%s: must stay Optional so it can be declared", name)
		}

		if !attribute.IsComputed() {
			t.Errorf("%s: must be Computed, or omitting it diffs forever", name)
		}
	}
}

// active is the only field carrying a default. The others must NOT: rule_id
// selects which rule the exception targets, operator has two accepted values with
// no stated default, and path is free-form — a default for any of them would be a
// guess that either retargets the exception or diffs forever.
func TestWAFRuleSetEnforcesOnlyActive(t *testing.T) {
	ctx := context.Background()
	sr := wafRuleSetSchema(t)

	active, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, "result.active"))
	if diags.HasError() {
		t.Fatalf("result.active: %v", diags)
	}

	withDefault, ok := active.(interface{ BoolDefaultValue() defaults.Bool })
	if !ok || withDefault.BoolDefaultValue() == nil {
		t.Error("result.active: expected a bool default")
	}

	for _, name := range []string{"result.rule_id", "result.path", "result.operator"} {
		attribute, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		switch a := attribute.(type) {
		case interface{ Int64DefaultValue() defaults.Int64 }:
			if a.Int64DefaultValue() != nil {
				t.Errorf("%s: must not carry a default", name)
			}
		case interface{ StringDefaultValue() defaults.String }:
			if a.StringDefaultValue() != nil {
				t.Errorf("%s: must not carry a default", name)
			}
		}
	}
}
