package provider

import (
	"os"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/aziontech/azionapi-go-sdk/waf"

	"github.com/aziontech/azionapi-go-sdk/digital_certificates"
	"github.com/aziontech/azionapi-go-sdk/domains"
	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/azionapi-go-sdk/edgefirewall"
	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
	"github.com/aziontech/azionapi-go-sdk/edgefunctionsinstance_edgefirewall"
	"github.com/aziontech/azionapi-go-sdk/networklist"
	"github.com/aziontech/azionapi-go-sdk/variables"
	"github.com/aziontech/azionapi-v4-go-sdk/edge"
)

type apiClient struct {
	idnsConfig *idns.Configuration
	idnsApi    *idns.APIClient

	domainsConfig *domains.Configuration
	domainsApi    *domains.APIClient

	edgefunctionsConfig *edgefunctions.Configuration
	edgefunctionsApi    *edgefunctions.APIClient

	edgeApplicationsConfig *edgeapplications.Configuration
	edgeApplicationsApi    *edgeapplications.APIClient

	digitalCertificatesConfig *digital_certificates.Configuration
	digitalCertificatesApi    *digital_certificates.APIClient

	networkListConfig *networklist.Configuration
	networkListApi    *networklist.APIClient

	edgefirewallConfig *edgefirewall.Configuration
	edgeFirewallApi    *edgefirewall.APIClient

	edgefunctionsinstanceEdgefirewallConfig *edgefunctionsinstance_edgefirewall.Configuration
	edgefunctionsinstanceEdgefirewallApi    *edgefunctionsinstance_edgefirewall.APIClient

	variablesConfig *variables.Configuration
	variablesApi    *variables.APIClient

	wafConfig *waf.Configuration
	wafApi    *waf.APIClient

	workloadConfig *edge.Configuration
	workloadsApi   *edge.APIClient
}

func Client(APIToken string, userAgent string) *apiClient {
	client := &apiClient{
		idnsConfig:                              idns.NewConfiguration(),
		domainsConfig:                           domains.NewConfiguration(),
		edgefunctionsConfig:                     edgefunctions.NewConfiguration(),
		edgeApplicationsConfig:                  edgeapplications.NewConfiguration(),
		digitalCertificatesConfig:               digital_certificates.NewConfiguration(),
		networkListConfig:                       networklist.NewConfiguration(),
		edgefirewallConfig:                      edgefirewall.NewConfiguration(),
		edgefunctionsinstanceEdgefirewallConfig: edgefunctionsinstance_edgefirewall.NewConfiguration(),
		variablesConfig:                         variables.NewConfiguration(),
		wafConfig:                               waf.NewConfiguration(),
		workloadConfig:                          edge.NewConfiguration(),
	}

	envApiEntrypoint := os.Getenv("AZION_API_ENTRYPOINT")
	if envApiEntrypoint != "" {
		client.domainsConfig.Servers[0].URL = envApiEntrypoint
		client.idnsConfig.Servers[0].URL = envApiEntrypoint
		client.edgefunctionsConfig.Servers[0].URL = envApiEntrypoint
		client.edgeApplicationsConfig.Servers[0].URL = envApiEntrypoint
		client.digitalCertificatesConfig.Servers[0].URL = envApiEntrypoint
		client.edgefirewallConfig.Servers[0].URL = envApiEntrypoint
		client.edgefunctionsinstanceEdgefirewallConfig.Servers[0].URL = envApiEntrypoint
		client.networkListConfig.Servers[0].URL = envApiEntrypoint
		client.variablesConfig.Servers[0].URL = envApiEntrypoint
		client.wafConfig.Servers[0].URL = envApiEntrypoint
	}

	client.domainsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.domainsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.domainsConfig.UserAgent = userAgent
	client.domainsApi = domains.NewAPIClient(client.domainsConfig)

	client.idnsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.idnsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.idnsConfig.UserAgent = userAgent
	client.idnsApi = idns.NewAPIClient(client.idnsConfig)

	client.edgefunctionsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgefunctionsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgefunctionsConfig.UserAgent = userAgent
	client.edgefunctionsApi = edgefunctions.NewAPIClient(client.edgefunctionsConfig)

	client.edgeApplicationsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgeApplicationsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgeApplicationsConfig.UserAgent = userAgent
	client.edgeApplicationsApi = edgeapplications.NewAPIClient(client.edgeApplicationsConfig)

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

	client.variablesConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.variablesConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.variablesConfig.UserAgent = userAgent
	client.variablesApi = variables.NewAPIClient(client.variablesConfig)

	client.wafConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.wafConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.wafConfig.UserAgent = userAgent
	client.wafApi = waf.NewAPIClient(client.wafConfig)

	client.workloadConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.workloadConfig.AddDefaultHeader("Accept", "application/json")
	client.workloadConfig.UserAgent = userAgent
	client.workloadConfig.Servers[0].URL = "https://api.azion.com/v4"
	client.workloadsApi = edge.NewAPIClient(client.workloadConfig)

	envApiV4Entrypoint := os.Getenv("AZION_API_V4_ENTRYPOINT")
	if envApiV4Entrypoint != "" {
		client.workloadConfig.Servers[0].URL = envApiV4Entrypoint
	}

	return client
}
