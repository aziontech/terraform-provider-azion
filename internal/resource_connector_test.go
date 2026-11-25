package provider

import (
	"context"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func connectorSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	sr := &fwresource.SchemaResponse{}
	NewConnectorResource().Schema(context.Background(), fwresource.SchemaRequest{}, sr)

	if sr.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", sr.Diagnostics)
	}

	return *sr
}

func TestConnectorSchemaImplementation(t *testing.T) {
	if diags := connectorSchema(t).Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them. A nested attribute here would silently accept a typo.
func TestConnectorNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()
	sr := connectorSchema(t)

	for _, name := range []string{
		"connector",
		"connector.storage_attributes",
		"connector.http_attributes",
		"connector.http_attributes.connection_options",
		"connector.http_attributes.modules",
		"connector.http_attributes.modules.load_balancer",
		"connector.http_attributes.modules.load_balancer.config",
		"connector.http_attributes.modules.origin_shield",
		"connector.http_attributes.modules.origin_shield.config",
		"connector.http_attributes.modules.origin_shield.config.origin_ip_acl",
		"connector.http_attributes.modules.origin_shield.config.hmac",
		"connector.http_attributes.modules.origin_shield.config.hmac.config",
		"connector.http_attributes.modules.origin_shield.config.hmac.config.attributes",
	} {
		if _, diags := sr.Schema.AttributeAtPath(ctx, tfPath(t, name)); !diags.HasError() {
			t.Errorf("%s: must be a block, not a nested attribute", name)
		}
	}

	if _, ok := sr.Schema.Blocks["connector"]; !ok {
		t.Fatal("connector: expected a top-level block")
	}
}

func testConnectorHTTPResponse() azionapi.Connector {
	method := "round_robin"
	dns := "both"
	enabled := true
	return azionapi.ConnectorHTTPAsConnector(&azionapi.ConnectorHTTP{
		Id:   42,
		Name: "origin",
		Type: "http",
		Attributes: azionapi.ConnectorHTTPAttributes{
			Addresses: []azionapi.Address{{Address: "origin.example.com"}},
			ConnectionOptions: &azionapi.HTTPConnectionOptions{
				DnsResolution: &dns,
			},
			Modules: &azionapi.HTTPModules{
				LoadBalancer: &azionapi.LoadBalancerModule{
					Enabled: &enabled,
					Config: *azionapi.NewNullableLoadBalancerModuleConfig(&azionapi.LoadBalancerModuleConfig{
						Method: &method,
					}),
				},
			},
		},
	})
}

// A block the configuration omits is planned as null, and Terraform core rejects
// a null block that comes back present. Blocks the API echoes but the prior
// plan/state did not declare must therefore stay null.
func TestConnectorPopulateKeepsUndeclaredBlocksNull(t *testing.T) {
	r := &connectorResource{}
	model := &connectorResourceResults{
		HTTPAttrs: &HTTPAttributesModel{
			Addresses: []AddressWrapperModel{{Endpoint: &AddressModel{Address: types.StringValue("origin.example.com")}}},
		},
	}

	r.populateConnectorFromResponse(context.Background(), model, testConnectorHTTPResponse())

	if model.HTTPAttrs == nil {
		t.Fatal("http_attributes: expected to be populated")
	}
	if model.HTTPAttrs.ConnectionOptions != nil {
		t.Error("connection_options: undeclared block must stay null")
	}
	if model.HTTPAttrs.Modules != nil {
		t.Error("modules: undeclared block must stay null")
	}
	if len(model.HTTPAttrs.Addresses) != 1 {
		t.Errorf("addresses: expected 1 entry, got %d", len(model.HTTPAttrs.Addresses))
	}
}

// Import has no prior value, so everything the API reports is kept.
func TestConnectorPopulateKeepsEverythingOnImport(t *testing.T) {
	r := &connectorResource{}
	model := &connectorResourceResults{}

	r.populateConnectorFromResponse(context.Background(), model, testConnectorHTTPResponse())

	if model.HTTPAttrs == nil || model.HTTPAttrs.ConnectionOptions == nil || model.HTTPAttrs.Modules == nil {
		t.Fatalf("import: expected every block the API reports, got %+v", model.HTTPAttrs)
	}
	lb := model.HTTPAttrs.Modules.LoadBalancer
	if lb == nil || lb.Config == nil || lb.Config.Method.ValueString() != "round_robin" {
		t.Errorf("import: expected load_balancer.config.method from the API, got %+v", lb)
	}
	if model.StorageAttrs != nil {
		t.Error("storage_attributes: must be null for an http connector")
	}
}
