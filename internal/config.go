package provider

import (
	"os"

	"github.com/aziontech/azionapi-go-sdk/domains"
	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
	"github.com/aziontech/azionapi-go-sdk/idns"
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
}

func Client(APIToken string, userAgent string) *apiClient {
	client := &apiClient{
		idnsConfig:             idns.NewConfiguration(),
		domainsConfig:          domains.NewConfiguration(),
		edgefunctionsConfig:    edgefunctions.NewConfiguration(),
		edgeApplicationsConfig: edgeapplications.NewConfiguration(),
	}

	envApiEntrypoint := os.Getenv("AZION_API_ENTRYPOINT")
	if envApiEntrypoint != "" {
		client.domainsConfig.Servers[0].URL = envApiEntrypoint
		client.idnsConfig.Servers[0].URL = envApiEntrypoint
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

	return client
}
