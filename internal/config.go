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

	edgeAplicationsConfig *edgeapplications.Configuration
	edgeAplicationsApi    *edgeapplications.APIClient
}

func Client(APIToken string, userAgent string) *apiClient {
	client := &apiClient{
		idnsConfig:            idns.NewConfiguration(),
		domainsConfig:         domains.NewConfiguration(),
		edgefunctionsConfig:   edgefunctions.NewConfiguration(),
		edgeAplicationsConfig: edgeapplications.NewConfiguration(),
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

	client.edgeAplicationsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgeAplicationsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgeAplicationsConfig.UserAgent = userAgent
	client.edgeAplicationsApi = edgeapplications.NewAPIClient(client.edgeAplicationsConfig)

	return client
}
