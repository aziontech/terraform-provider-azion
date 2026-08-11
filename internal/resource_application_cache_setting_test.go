package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// tfPath turns a dotted attribute name into a schema path.
func tfPath(t *testing.T, dotted string) path.Path {
	t.Helper()

	segments := strings.Split(dotted, ".")
	result := path.Root(segments[0])

	for _, segment := range segments[1:] {
		result = result.AtName(segment)
	}

	return result
}

// The schema uses objectdefault.StaticValue for the nested blocks it enforces.
// A default whose attribute types disagree with the schema is only rejected at
// plan time, which would surface as a runtime error for practitioners rather
// than a build failure, so assert it here.
func TestApplicationCacheSettingSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationCacheSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResponse.Diagnostics)
	}

	if diags := schemaResponse.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// Every enforced field must be Computed, otherwise its default is never applied
// and a value changed in the console is silently kept instead of reverted.
func TestApplicationCacheSettingEnforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationCacheSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	enforced := []string{
		"cache_setting.browser_cache",
		"cache_setting.browser_cache.behavior",
		"cache_setting.browser_cache.max_age",
		"cache_setting.modules.cache",
		"cache_setting.modules.cache.behavior",
		"cache_setting.modules.cache.max_age",
		"cache_setting.modules.cache.stale_cache",
		"cache_setting.modules.cache.stale_cache.enabled",
		"cache_setting.modules.cache.large_file_cache",
		"cache_setting.modules.cache.large_file_cache.enabled",
		"cache_setting.modules.cache.large_file_cache.offset",
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
