package provider

import (
	"os"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/aziontech/azionapi-go-sdk/waf"

	"github.com/aziontech/azionapi-go-sdk/digital_certificates"
	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/azionapi-go-sdk/edgefirewall"
	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
	"github.com/aziontech/azionapi-go-sdk/edgefunctionsinstance_edgefirewall"
	"github.com/aziontech/azionapi-go-sdk/networklist"
	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
)

type apiClient struct {
	idnsConfig *idns.Configuration
	idnsApi    *idns.APIClient

	edgefunctionsConfig *edgefunctions.Configuration
	edgefunctionsApi    *edgefunctions.APIClient

	//TODO: remove this
	edgeApplicationsApi *edgeapplications.APIClient

	// V4 SDK (azion-api) - preferred for new implementations
	apiConfig *azionapi.Configuration
	api       *azionapi.APIClient

	// Legacy V4 SDK (edge-api) - kept for backward compatibility
	edgeConfig *edgeapi.Configuration
	edgeApi    *edgeapi.APIClient

	digitalCertificatesConfig *digital_certificates.Configuration
	digitalCertificatesApi    *digital_certificates.APIClient

	networkListConfig *networklist.Configuration
	networkListApi    *networklist.APIClient

	edgefirewallConfig *edgefirewall.Configuration
	edgeFirewallApi    *edgefirewall.APIClient

	edgefunctionsinstanceEdgefirewallConfig *edgefunctionsinstance_edgefirewall.Configuration
	edgefunctionsinstanceEdgefirewallApi    *edgefunctionsinstance_edgefirewall.APIClient

	wafConfig *waf.Configuration
	wafApi    *waf.APIClient
}

func Client(APIToken string, userAgent string) *apiClient {
	client := &apiClient{
		idnsConfig:                              idns.NewConfiguration(),
		edgefunctionsConfig:                     edgefunctions.NewConfiguration(),
		apiConfig:                               azionapi.NewConfiguration(),
		edgeConfig:                              edgeapi.NewConfiguration(),
		digitalCertificatesConfig:               digital_certificates.NewConfiguration(),
		networkListConfig:                       networklist.NewConfiguration(),
		edgefirewallConfig:                      edgefirewall.NewConfiguration(),
		edgefunctionsinstanceEdgefirewallConfig: edgefunctionsinstance_edgefirewall.NewConfiguration(),
		wafConfig:                               waf.NewConfiguration(),
	}

	envApiEntrypoint := os.Getenv("AZION_API_ENTRYPOINT")
	v4url := "https://api.azion.com/v4"

	// Always set v4 URL for V4 SDKs (azion-api and edge-api)
	client.apiConfig.Servers[0].URL = v4url
	client.edgeConfig.Servers[0].URL = v4url

	if envApiEntrypoint != "" {
		client.idnsConfig.Servers[0].URL = envApiEntrypoint
		client.edgefunctionsConfig.Servers[0].URL = envApiEntrypoint
		client.digitalCertificatesConfig.Servers[0].URL = envApiEntrypoint
		client.edgefirewallConfig.Servers[0].URL = envApiEntrypoint
		client.edgefunctionsinstanceEdgefirewallConfig.Servers[0].URL = envApiEntrypoint
		client.networkListConfig.Servers[0].URL = envApiEntrypoint
		client.wafConfig.Servers[0].URL = envApiEntrypoint
	}

	client.idnsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.idnsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.idnsConfig.UserAgent = userAgent
	client.idnsApi = idns.NewAPIClient(client.idnsConfig)

	client.edgefunctionsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgefunctionsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgefunctionsConfig.UserAgent = userAgent
	client.edgefunctionsApi = edgefunctions.NewAPIClient(client.edgefunctionsConfig)

	// V4 SDK (azion-api) - preferred for new implementations
	client.apiConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.apiConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.apiConfig.UserAgent = userAgent
	client.api = azionapi.NewAPIClient(client.apiConfig)

	// Legacy V4 SDK (edge-api) - kept for backward compatibility
	client.edgeConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgeConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgeConfig.UserAgent = userAgent
	client.edgeApi = edgeapi.NewAPIClient(client.edgeConfig)

	client.digitalCertificatesConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.digitalCertificatesConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.digitalCertificatesConfig.UserAgent = userAgent
	client.digitalCertificatesApi = digital_certificates.NewAPIClient(client.digitalCertificatesConfig)

	client.networkListConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.networkListConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.networkListConfig.UserAgent = userAgent
	client.networkListApi = networklist.NewAPIClient(client.networkListConfig)

	client.edgefirewallConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgefirewallConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgefirewallConfig.UserAgent = userAgent
	client.edgeFirewallApi = edgefirewall.NewAPIClient(client.edgefirewallConfig)

	client.edgefunctionsinstanceEdgefirewallConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgefunctionsinstanceEdgefirewallConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgefunctionsinstanceEdgefirewallConfig.UserAgent = userAgent
	client.edgefunctionsinstanceEdgefirewallApi = edgefunctionsinstance_edgefirewall.NewAPIClient(client.edgefunctionsinstanceEdgefirewallConfig)

	client.wafConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.wafConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.wafConfig.UserAgent = userAgent
	client.wafApi = waf.NewAPIClient(client.wafConfig)

	return client
}
