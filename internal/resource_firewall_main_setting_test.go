package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// The schema uses objectdefault.StaticValue for modules and each module toggle.
// A default whose attribute types disagree with the schema is only rejected at
// plan time, so assert it here rather than leaving it to a practitioner's apply.
func TestFirewallMainSettingSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	FirewallMainSettingResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResponse.Diagnostics)
	}

	if diags := schemaResponse.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// Every enforced field must be Optional + Computed. Without Computed its default
// is never applied, so a module toggled in the console is kept instead of
// reverted; without Optional it could no longer be declared.
func TestFirewallMainSettingEnforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	FirewallMainSettingResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	enforced := []string{
		"data.active",
		"data.debug",
		"data.modules",
		"data.modules.functions",
		"data.modules.functions.enabled",
		"data.modules.network_protection",
		"data.modules.network_protection.enabled",
		"data.modules.waf",
		"data.modules.waf.enabled",
	}

	for _, name := range enforced {
		attribute, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsComputed() {
			t.Errorf("%s: must be Computed for its default to be enforced", name)
		}

		if !attribute.IsOptional() {
			t.Errorf("%s: must stay Optional so it can be declared in configuration", name)
		}
	}
}

// ddos_protection stays Optional even though the API accepts no value for it.
// Marking it read-only rejects configurations that already declare the block —
// including on destroy, which reads the configuration — so it must remain
// settable for backwards compatibility.
func TestFirewallDdosProtectionStaysConfigurable(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	FirewallMainSettingResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{"data.modules.ddos_protection", "data.modules.ddos_protection.enabled"} {
		attribute, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsOptional() {
			t.Errorf("%s: must stay Optional; marking it read-only breaks existing configurations", name)
		}

		if !attribute.IsComputed() {
			t.Errorf("%s: must be Computed so an omitted block resolves to its default instead of drifting", name)
		}
	}
}
