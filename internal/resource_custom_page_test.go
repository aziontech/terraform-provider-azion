package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
)

func customPageSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	sr := &fwresource.SchemaResponse{}
	NewCustomPageResource().Schema(context.Background(), fwresource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestCustomPageSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	if diags := customPageSchema(t).Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// active is enforced, so a page disabled outside Terraform is re-enabled.
func TestCustomPageActiveIsEnforced(t *testing.T) {
	ctx := context.Background()
	sr := customPageSchema(t)

	attribute, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, "custom_page.active"))
	if diags.HasError() {
		t.Fatalf("custom_page.active: %v", diags)
	}

	if !attribute.IsOptional() || !attribute.IsComputed() {
		t.Errorf("custom_page.active: expected Optional + Computed")
	}

	withDefault, ok := attribute.(interface{ BoolDefaultValue() defaults.Bool })
	if !ok || withDefault.BoolDefaultValue() == nil {
		t.Error("custom_page.active: expected a bool default")
	}
}

// The per-page attributes must stay Computed but must NOT carry defaults. `uri`
// and `custom_status_code` are nullable in the API, where null means "no
// override" — defaulting either would assert an override nobody asked for — and
// `ttl` has no documented default.
func TestCustomPagePageAttributesHaveNoDefaults(t *testing.T) {
	ctx := context.Background()
	sr := customPageSchema(t)

	attrsPath := path.Root("custom_page").AtName("pages").AtListIndex(0).
		AtName("entry").AtName("page").AtName("attributes")

	for _, name := range []string{"ttl", "uri", "custom_status_code"} {
		attribute, diags := sr.Schema.AttributeAtPath(ctx, attrsPath.AtName(name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsOptional() || !attribute.IsComputed() {
			t.Errorf("%s: expected Optional + Computed, got optional=%v computed=%v",
				name, attribute.IsOptional(), attribute.IsComputed())
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
		default:
			t.Errorf("%s: unexpected attribute type %T", name, attribute)
		}
	}
}
