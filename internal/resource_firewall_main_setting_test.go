package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
// reverted; without Optional it could no longer be declared. The objects that
// hold them are blocks, which cannot be Computed; an omitted block is covered by
// buildFirewallModulesRequest instead.
func TestFirewallMainSettingEnforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	FirewallMainSettingResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	enforced := []string{
		"data.active",
		"data.debug",
		"data.modules.functions.enabled",
		"data.modules.network_protection.enabled",
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

// ddos_protection stays configurable even though the API accepts no value for
// it. Marking it read-only rejects configurations that already declare the
// block — including on destroy, which reads the configuration — so it must
// remain settable for backwards compatibility. The block itself is optional by
// nature; its flag must stay Optional + Computed.
func TestFirewallDdosProtectionStaysConfigurable(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	FirewallMainSettingResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	if _, ok := schemaResponse.Schema.Blocks["data"].GetNestedObject().GetBlocks()["modules"].GetNestedObject().GetBlocks()["ddos_protection"]; !ok {
		t.Error("data.modules.ddos_protection: block missing")
	}

	for _, name := range []string{"data.modules.ddos_protection.enabled"} {
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

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them. A nested attribute here would silently accept a typo.
func TestFirewallMainSettingNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	FirewallMainSettingResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"data",
		"data.modules",
		"data.modules.ddos_protection",
		"data.modules.functions",
		"data.modules.network_protection",
		"data.modules.waf",
	} {
		if _, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name)); !diags.HasError() {
			t.Errorf("%s: must be a block, not a nested attribute", name)
		}
	}
}

// An omitted block must still send its defaults, so a module toggled in the
// console is reset on apply exactly as before the conversion to blocks.
func TestBuildFirewallModulesRequestSendsDefaultsForOmittedBlocks(t *testing.T) {
	cases := map[string]struct {
		modules *FirewallResourceModules
		want    map[string]bool
	}{
		"no modules block": {
			modules: nil,
			want: map[string]bool{
				"functions":          defaultFirewallFunctions,
				"network_protection": defaultFirewallNetworkProtection,
				"waf":                defaultFirewallWAF,
			},
		},
		"only waf block": {
			modules: &FirewallResourceModules{WAF: &WAFModule{Enabled: types.BoolValue(true)}},
			want: map[string]bool{
				"functions":          defaultFirewallFunctions,
				"network_protection": defaultFirewallNetworkProtection,
				"waf":                true,
			},
		},
		"empty functions block": {
			modules: &FirewallResourceModules{Functions: &FunctionsModule{Enabled: types.BoolNull()}},
			want: map[string]bool{
				"functions":          defaultFirewallFunctions,
				"network_protection": defaultFirewallNetworkProtection,
				"waf":                defaultFirewallWAF,
			},
		},
		"every block declared": {
			modules: &FirewallResourceModules{
				Functions:         &FunctionsModule{Enabled: types.BoolValue(false)},
				NetworkProtection: &NetworkProtectionModule{Enabled: types.BoolValue(false)},
				WAF:               &WAFModule{Enabled: types.BoolValue(true)},
			},
			want: map[string]bool{
				"functions":          false,
				"network_protection": false,
				"waf":                true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			request := buildFirewallModulesRequest(tc.modules)

			if request.Functions == nil || request.NetworkProtection == nil || request.Waf == nil {
				t.Fatalf("every settable module must be sent: %+v", request)
			}

			got := map[string]bool{
				"functions":          request.Functions.GetEnabled(),
				"network_protection": request.NetworkProtection.GetEnabled(),
				"waf":                request.Waf.GetEnabled(),
			}
			for module, enabled := range tc.want {
				if got[module] != enabled {
					t.Errorf("%s: enabled = %t, want %t", module, got[module], enabled)
				}
			}
		})
	}
}

// Terraform rejects a block planned as absent that comes back present, so the
// API result must not add blocks the configuration left out. Import has no
// prior value and keeps everything.
func TestKeepDeclaredFirewallModules(t *testing.T) {
	full := func() *FirewallResourceResults {
		return &FirewallResourceResults{Modules: &FirewallResourceModules{
			DdosProtection:    &DdosProtectionModule{Enabled: types.BoolValue(true)},
			Functions:         &FunctionsModule{Enabled: types.BoolValue(true)},
			NetworkProtection: &NetworkProtectionModule{Enabled: types.BoolValue(true)},
			WAF:               &WAFModule{Enabled: types.BoolValue(false)},
		}}
	}

	t.Run("no modules block", func(t *testing.T) {
		results := full()
		keepDeclaredFirewallModules(&FirewallResourceResults{}, results)
		if results.Modules != nil {
			t.Errorf("modules = %+v, want nil", results.Modules)
		}
	})

	t.Run("only waf block", func(t *testing.T) {
		results := full()
		keepDeclaredFirewallModules(&FirewallResourceResults{Modules: &FirewallResourceModules{WAF: &WAFModule{}}}, results)
		if results.Modules.WAF == nil {
			t.Error("waf: dropped, want kept")
		}
		if results.Modules.DdosProtection != nil || results.Modules.Functions != nil || results.Modules.NetworkProtection != nil {
			t.Errorf("undeclared modules kept: %+v", results.Modules)
		}
	})

	t.Run("import", func(t *testing.T) {
		results := full()
		keepDeclaredFirewallModules(nil, results)
		if results.Modules == nil || results.Modules.DdosProtection == nil || results.Modules.WAF == nil {
			t.Errorf("import dropped modules: %+v", results.Modules)
		}
	})
}
