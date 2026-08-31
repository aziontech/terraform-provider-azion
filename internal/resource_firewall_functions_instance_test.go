package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// importStateForTest runs ImportState against a freshly-nulled state, the same
// way the framework does before it calls Read.
func importStateForTest(ctx context.Context, t *testing.T, id string) resource.ImportStateResponse {
	t.Helper()

	r := &FirewallFunctionsInstanceResource{}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", schemaResp.Diagnostics)
	}

	importResp := resource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			Schema: schemaResp.Schema,
		},
	}

	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &importResp)

	return importResp
}

// TestFirewallFunctionsInstanceImportState covers the regression where importing
// left `data` null and Read could not decode the state into the resource model.
func TestFirewallFunctionsInstanceImportState(t *testing.T) {
	ctx := context.Background()

	importResp := importStateForTest(ctx, t, "12345/678")
	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState returned errors: %v", importResp.Diagnostics)
	}

	// This is the exact call Read makes first, and the one that used to fail with
	// "Received null value, however the target type cannot handle null values".
	var state FirewallFunctionInstanceResourceModel
	diags := importResp.State.Get(ctx, &state)
	if diags.HasError() {
		t.Fatalf("decoding imported state into the model failed: %v", diags)
	}

	if got := state.FirewallID.ValueInt64(); got != 12345 {
		t.Errorf("firewall_id = %d, want 12345", got)
	}
	if got := state.ID.ValueString(); got != "678" {
		t.Errorf("id = %q, want \"678\"", got)
	}
	if state.Data != nil {
		t.Errorf("data = %+v, want nil until Read populates it", state.Data)
	}
}

func TestFirewallFunctionsInstanceImportStateInvalidID(t *testing.T) {
	ctx := context.Background()

	for name, id := range map[string]string{
		"bare id":            "678",
		"empty":              "",
		"too many parts":     "12345/678/9",
		"non-numeric parent": "abc/678",
		"non-numeric child":  "12345/abc",
	} {
		t.Run(name, func(t *testing.T) {
			importResp := importStateForTest(ctx, t, id)
			if !importResp.Diagnostics.HasError() {
				t.Errorf("ImportState(%q) succeeded, want an error diagnostic", id)
			}
		})
	}
}
