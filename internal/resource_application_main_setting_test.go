package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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
// reverted; without Optional it could no longer be declared.
func TestApplicationMainSettingEnforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationMainSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	enforced := []string{
		"application.active",
		"application.debug",
		"application.modules",
		"application.modules.cache",
		"application.modules.cache.enabled",
		"application.modules.functions",
		"application.modules.functions.enabled",
		"application.modules.application_accelerator",
		"application.modules.application_accelerator.enabled",
		"application.modules.image_processor",
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
