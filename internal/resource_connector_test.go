package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
)

func TestConnectorResponseUnmarshalAcceptsNullMetadata(t *testing.T) {
	input := []byte(`{
		"data": {
			"id": 544,
			"name": "azion-faq-test",
			"last_editor": "artur.rossa+mts2025@azion.com",
			"last_modified": "2025-10-02T21:02:40.585182Z",
			"created_at": null,
			"active": true,
			"product_version": "1.0",
			"type": "storage",
			"is_versioned": null,
			"version": null,
			"version_state": null,
			"version_id": null,
			"attributes": {
				"bucket": "faq-test",
				"prefix": "20251002175613"
			}
		}
	}`)

	connector, err := decodeConnectorResponse(input)
	if err != nil {
		t.Fatalf("decodeConnectorResponse() returned error: %v", err)
	}

	if connector.Data.ConnectorStorage == nil {
		t.Fatalf("expected storage connector, got %#v", connector.Data.GetActualInstance())
	}
	if got := connector.Data.ConnectorStorage.GetId(); got != 544 {
		t.Fatalf("expected id 544, got %d", got)
	}
	if got := connector.Data.ConnectorStorage.GetCreatedAt(); !got.IsZero() {
		t.Fatalf("expected zero created_at for null response value, got %s", got)
	}
}

func TestSDKConnectorResponseStillRejectsNullMetadata(t *testing.T) {
	input := []byte(`{
		"data": {
			"id": 544,
			"name": "azion-faq-test",
			"last_editor": "artur.rossa+mts2025@azion.com",
			"last_modified": "2025-10-02T21:02:40.585182Z",
			"created_at": null,
			"active": true,
			"product_version": "1.0",
			"type": "storage",
			"is_versioned": null,
			"version": null,
			"version_state": null,
			"version_id": null,
			"attributes": {
				"bucket": "faq-test",
				"prefix": "20251002175613"
			}
		}
	}`)

	var connector sdkConnectorResponseAlias
	if err := json.Unmarshal(input, &connector); err == nil {
		t.Fatalf("expected SDK unmarshal to reject connector metadata fields")
	}
}

func TestConnectorResponseUnmarshalAcceptsHTTPNullMetadata(t *testing.T) {
	input := []byte(`{
		"data": {
			"id": 307,
			"name": "graphql-proxy",
			"last_editor": "artur.rossa+mts2025@azion.com",
			"last_modified": "2025-09-12T23:06:37.252469Z",
			"created_at": null,
			"active": true,
			"product_version": "1.0",
			"type": "http",
			"is_versioned": null,
			"version": null,
			"version_state": null,
			"version_id": null,
			"attributes": {
				"addresses": [{
					"active": true,
					"address": "api.azion.com",
					"http_port": 80,
					"https_port": 443,
					"modules": null
				}],
				"connection_options": {
					"dns_resolution": "both",
					"transport_policy": "preserve",
					"http_version_policy": "http1_1",
					"host": "api.azion.com",
					"path_prefix": "/",
					"following_redirect": false,
					"real_ip_header": "X-Real-IP",
					"real_port_header": "X-Real-PORT"
				},
				"modules": {
					"load_balancer": {"enabled": false, "config": null},
					"origin_shield": {"enabled": false, "config": null}
				}
			}
		}
	}`)

	connector, err := decodeConnectorResponse(input)
	if err != nil {
		t.Fatalf("decodeConnectorResponse() returned error: %v", err)
	}

	if connector.Data.ConnectorHTTP == nil {
		t.Fatalf("expected HTTP connector, got %#v", connector.Data.GetActualInstance())
	}
	if got := connector.Data.ConnectorHTTP.GetId(); got != 307 {
		t.Fatalf("expected id 307, got %d", got)
	}
	if got := connector.Data.ConnectorHTTP.Attributes.Addresses[0].Address; got != "api.azion.com" {
		t.Fatalf("expected address api.azion.com, got %q", got)
	}
}

func TestConnectorResponseUnmarshalAcceptsLiveIngestNullMetadata(t *testing.T) {
	input := []byte(`{
		"data": {
			"id": 9001,
			"name": "live-ingest",
			"last_editor": "artur.rossa+mts2025@azion.com",
			"last_modified": "2026-01-09T19:07:46.804972Z",
			"created_at": null,
			"active": true,
			"product_version": "1.0",
			"type": "live_ingest",
			"is_versioned": null,
			"version": null,
			"version_state": null,
			"version_id": null,
			"attributes": {"region": "us-east-1"}
		}
	}`)

	connector, err := decodeConnectorResponse(input)
	if err != nil {
		t.Fatalf("decodeConnectorResponse() returned error: %v", err)
	}

	if connector.Data.ConnectorLiveIngest == nil {
		t.Fatalf("expected live ingest connector, got %#v", connector.Data.GetActualInstance())
	}
	if got := connector.Data.ConnectorLiveIngest.Attributes.Region; got != "us-east-1" {
		t.Fatalf("expected region us-east-1, got %q", got)
	}
}

func TestRetrieveConnectorFallbackRewindsResponseBody(t *testing.T) {
	input := []byte(`{
		"data": {
			"id": 544,
			"name": "azion-faq-test",
			"last_editor": "artur.rossa+mts2025@azion.com",
			"last_modified": "2025-10-02T21:02:40.585182Z",
			"created_at": null,
			"active": true,
			"product_version": "1.0",
			"type": "storage",
			"is_versioned": null,
			"version": null,
			"version_state": null,
			"version_id": null,
			"attributes": {"bucket": "faq-test", "prefix": "20251002175613"}
		}
	}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspace/connectors/544" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(input)
	}))
	defer server.Close()

	config := azionapi.NewConfiguration()
	config.Servers[0].URL = server.URL
	resource := connectorResource{client: &apiClient{api: azionapi.NewAPIClient(config)}}

	connector, response, err := resource.retrieveConnector(t.Context(), 544)
	if err != nil {
		t.Fatalf("retrieveConnector() returned error: %v", err)
	}
	defer response.Body.Close()
	if connector.Data.ConnectorStorage == nil {
		t.Fatalf("expected storage connector, got %#v", connector.Data.GetActualInstance())
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	if string(body) != string(input) {
		t.Fatalf("expected rewound response body %s, got %s", input, body)
	}
}
