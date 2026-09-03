package provider

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataStreamTemplateSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	sr := &fwresource.SchemaResponse{}
	NewDataStreamTemplateResource().Schema(context.Background(), fwresource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestDataStreamTemplateSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	if diags := dataStreamTemplateSchema(t).Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// `custom` is set by the API — true for anything Terraform creates, false for
// Azion's built-in templates — so it must be Computed only. Accepting it as
// configuration would let a plan assert something the API decides.
func TestDataStreamTemplateCustomIsComputedOnly(t *testing.T) {
	ctx := context.Background()

	attribute, diags := dataStreamTemplateSchema(t).Schema.AttributeAtPath(ctx, tfPath(t, "template.custom"))
	if diags.HasError() {
		t.Fatalf("template.custom: %v", diags)
	}

	if !attribute.IsComputed() {
		t.Error("template.custom: expected Computed")
	}
	if attribute.IsOptional() || attribute.IsRequired() {
		t.Errorf("template.custom: expected Computed only, got optional=%v required=%v",
			attribute.IsOptional(), attribute.IsRequired())
	}
}

// The API strips leading and trailing whitespace from `data_set`. Since it is a
// Required attribute, Terraform demands the applied value match the
// configuration byte-for-byte, so a heredoc (which always ends in a newline)
// would fail apply outright if the trimmed echo were stored.
func TestPreferConfiguredDataSet(t *testing.T) {
	tests := []struct {
		name     string
		prior    types.String
		apiValue string
		want     string
	}{
		{
			name:     "heredoc trailing newline is preserved",
			prior:    types.StringValue("{\n  \"status\": \"$status\"\n}\n"),
			apiValue: "{\n  \"status\": \"$status\"\n}",
			want:     "{\n  \"status\": \"$status\"\n}\n",
		},
		{
			name:     "leading whitespace is preserved too",
			prior:    types.StringValue("  $status"),
			apiValue: "$status",
			want:     "  $status",
		},
		{
			name:     "an identical echo is stored as-is",
			prior:    types.StringValue("$status"),
			apiValue: "$status",
			want:     "$status",
		},
		{
			name:     "a genuine server-side rewrite is real drift and must surface",
			prior:    types.StringValue("$status"),
			apiValue: "$status $host",
			want:     "$status $host",
		},
		{
			name:     "no prior value, as on import: the API value is used",
			prior:    types.StringNull(),
			apiValue: "$status",
			want:     "$status",
		},
		{
			name:     "an unknown prior value is not carried over",
			prior:    types.StringUnknown(),
			apiValue: "$status",
			want:     "$status",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := preferConfiguredDataSet(test.prior, test.apiValue)

			if got.ValueString() != test.want {
				t.Errorf("got %q, want %q", got.ValueString(), test.want)
			}
		})
	}
}
