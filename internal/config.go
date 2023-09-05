package provider

import (
	"github.com/aziontech/azionapi-go-sdk/idns"
	"os"

	"github.com/aziontech/azionapi-go-sdk/digital_certificates"
	"github.com/aziontech/azionapi-go-sdk/domains"
	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/azionapi-go-sdk/edgefirewall"
	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
	"github.com/aziontech/azionapi-go-sdk/networklist"
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
}

func Client(APIToken string, userAgent string) *apiClient {
	client := &apiClient{
		idnsConfig:                idns.NewConfiguration(),
		domainsConfig:             domains.NewConfiguration(),
		edgefunctionsConfig:       edgefunctions.NewConfiguration(),
		edgeApplicationsConfig:    edgeapplications.NewConfiguration(),
		digitalCertificatesConfig: digital_certificates.NewConfiguration(),
		networkListConfig:         networklist.NewConfiguration(),
		edgefirewallConfig:        edgefirewall.NewConfiguration(),
	}

	envApiEntrypoint := os.Getenv("AZION_API_ENTRYPOINT")
	if envApiEntrypoint != "" {
		client.domainsConfig.Servers[0].URL = envApiEntrypoint
		client.idnsConfig.Servers[0].URL = envApiEntrypoint
		client.edgefunctionsConfig.Servers[0].URL = envApiEntrypoint
		client.edgeApplicationsConfig.Servers[0].URL = envApiEntrypoint
		client.digitalCertificatesConfig.Servers[0].URL = envApiEntrypoint
		client.edgefirewallConfig.Servers[0].URL = envApiEntrypoint
		client.networkListConfig.Servers[0].URL = envApiEntrypoint
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

	return client
}
