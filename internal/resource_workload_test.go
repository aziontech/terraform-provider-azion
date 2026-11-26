package provider

import (
	"context"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A default whose type disagrees with the schema is only rejected at plan
// time, so assert the schema here rather than leaving it to a practitioner's
// apply.
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

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them. A nested attribute here would silently accept a typo.
func TestWorkloadNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewWorkloadResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"workload",
		"workload.tls",
		"workload.protocols",
		"workload.protocols.http",
		"workload.mtls",
		"workload.mtls.config",
	} {
		if _, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name)); !diags.HasError() {
			t.Errorf("%s: must be a block, not a nested attribute", name)
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

// An omitted tls block must still send minimum_version = tls_1_3 and nothing
// else, exactly as when tls was a Computed object with that default.
func TestWorkloadTLSRequestOmittedBlockSendsDefaults(t *testing.T) {
	tls := workloadTLSRequest(nil)

	if got := tls.GetMinimumVersion(); got != defaultWorkloadTLSMinimumVersion {
		t.Errorf("minimum_version = %q, want %q", got, defaultWorkloadTLSMinimumVersion)
	}
	if tls.HasCertificate() || tls.HasCiphers() {
		t.Errorf("certificate/ciphers must not be sent for an omitted block: %+v", tls)
	}

	declared := workloadTLSRequest(&TLSWorkloadResourceModel{
		Certificate:    types.Int64Value(7),
		Ciphers:        types.Int64Unknown(),
		MinimumVersion: types.StringValue("tls_1_2"),
	})
	if declared.GetCertificate() != 7 || declared.GetMinimumVersion() != "tls_1_2" || declared.HasCiphers() {
		t.Errorf("declared block not sent as configured: %+v", declared)
	}
}

// An omitted protocols block, or a protocols block without http, must still
// send http_ports [80] and https_ports [443] and leave versions/quic_ports to
// the API.
func TestWorkloadProtocolsRequestOmittedBlockSendsDefaults(t *testing.T) {
	ctx := context.Background()

	for name, model := range map[string]*ProtocolsResourceModel{
		"no protocols block": nil,
		"no http block":      {},
	} {
		t.Run(name, func(t *testing.T) {
			protocols, diags := workloadProtocolsRequest(ctx, model)
			if diags.HasError() {
				t.Fatal(diags)
			}

			http := protocols.GetHttp()
			if got := http.GetHttpPorts(); len(got) != 1 || got[0] != 80 {
				t.Errorf("http_ports = %v, want [80]", got)
			}
			if got := http.GetHttpsPorts(); len(got) != 1 || got[0] != 443 {
				t.Errorf("https_ports = %v, want [443]", got)
			}
			if http.HasVersions() || http.HasQuicPorts() {
				t.Errorf("versions/quic_ports must be left to the API: %+v", http)
			}
		})
	}
}

// An omitted mtls block must still send enabled = false and no config.
func TestWorkloadMTLSRequestOmittedBlockSendsDefaults(t *testing.T) {
	ctx := context.Background()

	mtls, diags := workloadMTLSRequest(ctx, nil)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !mtls.HasEnabled() || mtls.GetEnabled() != defaultWorkloadMTLSEnabled {
		t.Errorf("enabled must be sent as %t: %+v", defaultWorkloadMTLSEnabled, mtls)
	}
	if mtls.HasConfig() {
		t.Errorf("config must not be sent for an omitted block: %+v", mtls)
	}
}

// Terraform rejects a block planned as absent that comes back present, so the
// API result must not add blocks the configuration left out. Import has no
// prior value and keeps everything.
func TestKeepDeclaredWorkloadBlocks(t *testing.T) {
	full := func() *workloadResourceResults {
		return &workloadResourceResults{
			Tls:       defaultWorkloadTLS(),
			Protocols: &ProtocolsResourceModel{Http: defaultWorkloadHTTP()},
			Mtls: &MTLSResourceModel{
				Enabled: types.BoolValue(true),
				Config:  &MTLSConfigResourceModel{Certificate: types.Int64Value(1)},
			},
		}
	}

	t.Run("no optional blocks", func(t *testing.T) {
		results := full()
		keepDeclaredWorkloadBlocks(&workloadResourceResults{}, results)
		if results.Tls != nil || results.Protocols != nil || results.Mtls != nil {
			t.Errorf("undeclared blocks kept: %+v", results)
		}
	})

	t.Run("outer blocks without inner blocks", func(t *testing.T) {
		results := full()
		keepDeclaredWorkloadBlocks(&workloadResourceResults{
			Protocols: &ProtocolsResourceModel{},
			Mtls:      &MTLSResourceModel{},
		}, results)
		if results.Tls != nil {
			t.Error("tls: kept, want dropped")
		}
		if results.Protocols == nil || results.Protocols.Http != nil {
			t.Errorf("protocols: want kept without http, got %+v", results.Protocols)
		}
		if results.Mtls == nil || results.Mtls.Config != nil {
			t.Errorf("mtls: want kept without config, got %+v", results.Mtls)
		}
	})

	t.Run("all declared", func(t *testing.T) {
		results := full()
		keepDeclaredWorkloadBlocks(full(), results)
		if results.Tls == nil || results.Protocols.Http == nil || results.Mtls.Config == nil {
			t.Errorf("declared blocks dropped: %+v", results)
		}
	})

	t.Run("import", func(t *testing.T) {
		results := full()
		keepDeclaredWorkloadBlocks(nil, results)
		if results.Tls == nil || results.Protocols.Http == nil || results.Mtls.Config == nil {
			t.Errorf("import dropped blocks: %+v", results)
		}
	})
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
