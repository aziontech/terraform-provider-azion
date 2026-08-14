package provider

import (
	"context"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
)

// The schema uses objectdefault.StaticValue for tls, protocols and mtls. A
// default whose attribute types disagree with the schema is only rejected at
// plan time, so assert it here rather than leaving it to a practitioner's apply.
func TestWorkloadSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewWorkloadResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResponse.Diagnostics)
	}

	if diags := schemaResponse.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// Every nested block needs a default. A Computed block without one is unknown in
// the plan when the configuration omits it, and workloadResourceResults reads
// nested blocks into pointer fields, which cannot hold unknown values — that
// fails Create with "the target type cannot handle unknown values".
func TestWorkloadNestedBlocksCannotPlanUnknown(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewWorkloadResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"workload.tls",
		"workload.protocols",
		"workload.protocols.http",
		"workload.mtls",
		"workload.mtls.config",
	} {
		attribute, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsComputed() {
			t.Errorf("%s: must be Computed so an omitted block resolves to its default", name)
			continue
		}

		withDefault, ok := attribute.(interface{ ObjectDefaultValue() defaults.Object })
		if !ok || withDefault.ObjectDefaultValue() == nil {
			t.Errorf("%s: Computed nested block needs a Default, or Create fails on an unknown object", name)
		}
	}
}

// Fields the API supplies but Terraform does not enforce must still be Computed,
// otherwise an unfiltered Read writes a value the plan holds as null and the
// resource diffs forever.
func TestWorkloadUnenforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewWorkloadResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"workload.active",
		"workload.domains",
		"workload.tls.certificate",
		"workload.tls.ciphers",
		"workload.protocols.http.versions",
		"workload.protocols.http.quic_ports",
		"workload.mtls.config.certificate",
		"workload.mtls.config.crl",
		"workload.mtls.config.verification",
	} {
		attribute, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsComputed() || !attribute.IsOptional() {
			t.Errorf("%s: must be Optional + Computed so the API can supply it without drifting", name)
		}
	}
}

// The API returns an all-null mtls.config for a workload with no mTLS. The plan
// holds null there while mtls is disabled, so writing that object into state
// fails the apply with "Provider produced inconsistent result after apply".
func TestPopulateWorkloadResultsIgnoresEmptyMTLSConfig(t *testing.T) {
	ctx := context.Background()

	mtls := azionapi.NewMTLS()
	mtls.SetEnabled(false)
	mtls.SetConfig(*azionapi.NewMTLSConfig()) // every field unset, as the API sends it

	result := populateWorkloadResults(ctx, &azionapi.WorkloadResponse{
		Data: azionapi.Workload{Name: "probe", Mtls: mtls},
	})

	if result.Mtls == nil {
		t.Fatal("mtls should be populated")
	}

	if result.Mtls.Config != nil {
		t.Errorf("an all-null config must be treated as absent, got %+v", result.Mtls.Config)
	}
}

// A config carrying real values must still reach state, otherwise drift in a
// declared mTLS configuration would be invisible.
func TestPopulateWorkloadResultsKeepsRealMTLSConfig(t *testing.T) {
	ctx := context.Background()

	config := azionapi.NewMTLSConfig()
	config.SetCertificate(4321)
	config.SetVerification("enforce")

	mtls := azionapi.NewMTLS()
	mtls.SetEnabled(true)
	mtls.SetConfig(*config)

	result := populateWorkloadResults(ctx, &azionapi.WorkloadResponse{
		Data: azionapi.Workload{Name: "probe", Mtls: mtls},
	})

	if result.Mtls == nil || result.Mtls.Config == nil {
		t.Fatal("a config with values must be populated")
	}

	if got := result.Mtls.Config.Certificate.ValueInt64(); got != 4321 {
		t.Errorf("certificate = %d, want 4321", got)
	}

	if got := result.Mtls.Config.Verification.ValueString(); got != "enforce" {
		t.Errorf("verification = %q, want \"enforce\"", got)
	}
}
