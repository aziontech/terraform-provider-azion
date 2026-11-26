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
func TestApplicationMainSettingSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationMainSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

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
// transformModuleIntoRequest instead.
func TestApplicationMainSettingEnforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationMainSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	enforced := []string{
		"application.active",
		"application.debug",
		"application.modules.cache.enabled",
		"application.modules.functions.enabled",
		"application.modules.application_accelerator.enabled",
		"application.modules.image_processor.enabled",
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

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them. A nested attribute here would silently accept a typo.
func TestApplicationMainSettingNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationMainSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"application",
		"application.modules",
		"application.modules.cache",
		"application.modules.functions",
		"application.modules.application_accelerator",
		"application.modules.image_processor",
	} {
		if _, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name)); !diags.HasError() {
			t.Errorf("%s: must be a block, not a nested attribute", name)
		}
	}
}

// An omitted block must still send its defaults, so a module toggled in the
// console is reset on apply exactly as before the conversion to blocks.
func TestTransformModuleIntoRequestSendsDefaultsForOmittedBlocks(t *testing.T) {
	disabled := types.BoolValue(false)
	cases := map[string]*ApplicationModules{
		"no modules block":  nil,
		"only cache block":  {Cache: &CacheModule{Enabled: disabled}},
		"empty cache block": {Cache: &CacheModule{Enabled: types.BoolNull()}},
	}

	for name, modules := range cases {
		t.Run(name, func(t *testing.T) {
			request := transformModuleIntoRequest(modules)

			wantCache := defaultModuleCacheEnabled
			if modules != nil && modules.Cache != nil && isSet(modules.Cache.Enabled) {
				wantCache = modules.Cache.Enabled.ValueBool()
			}

			got := map[string]bool{
				"cache":                   request.Cache.GetEnabled(),
				"functions":               request.Functions.GetEnabled(),
				"application_accelerator": request.ApplicationAccelerator.GetEnabled(),
				"image_processor":         request.ImageProcessor.GetEnabled(),
			}
			want := map[string]bool{
				"cache":                   wantCache,
				"functions":               defaultModuleFunctionsEnabled,
				"application_accelerator": defaultModuleAcceleratorEnabled,
				"image_processor":         defaultModuleImageProcessorEnabled,
			}
			for module, enabled := range want {
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
func TestKeepDeclaredModules(t *testing.T) {
	full := func() *ApplicationResults {
		return &ApplicationResults{Modules: &ApplicationModules{
			Cache:                  &CacheModule{Enabled: types.BoolValue(true)},
			Functions:              &FunctionModule{Enabled: types.BoolValue(true)},
			ApplicationAccelerator: &ApplicationAcceleratorModule{Enabled: types.BoolValue(false)},
			ImageProcessor:         &ImageProcessorModule{Enabled: types.BoolValue(false)},
		}}
	}

	t.Run("no modules block", func(t *testing.T) {
		results := full()
		keepDeclaredModules(&ApplicationResults{}, results)
		if results.Modules != nil {
			t.Errorf("modules = %+v, want nil", results.Modules)
		}
	})

	t.Run("only cache block", func(t *testing.T) {
		results := full()
		keepDeclaredModules(&ApplicationResults{Modules: &ApplicationModules{Cache: &CacheModule{}}}, results)
		if results.Modules.Cache == nil {
			t.Error("cache: dropped, want kept")
		}
		if results.Modules.Functions != nil || results.Modules.ApplicationAccelerator != nil || results.Modules.ImageProcessor != nil {
			t.Errorf("undeclared modules kept: %+v", results.Modules)
		}
	})

	t.Run("import", func(t *testing.T) {
		results := full()
		keepDeclaredModules(nil, results)
		if results.Modules == nil || results.Modules.ImageProcessor == nil {
			t.Errorf("import dropped modules: %+v", results.Modules)
		}
	})
}
