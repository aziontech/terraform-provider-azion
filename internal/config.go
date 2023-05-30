package provider

import (
	"github.com/aziontech/azionapi-go-sdk/domains"
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
}

func Client(APIToken string, userAgent string) *apiClient {
	client := &apiClient{
		idnsConfig:          idns.NewConfiguration(),
		domainsConfig:       domains.NewConfiguration(),
		edgefunctionsConfig: edgefunctions.NewConfiguration(),
	}

	client.domainsApi = domains.NewAPIClient(client.domainsConfig)
	client.domainsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.domainsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.domainsConfig.UserAgent = userAgent

	client.idnsApi = idns.NewAPIClient(client.idnsConfig)
	client.idnsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.idnsConfig.UserAgent = userAgent

	client.edgefunctionsApi = edgefunctions.NewAPIClient(client.edgefunctionsConfig)
	client.edgefunctionsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	client.edgefunctionsConfig.AddDefaultHeader("Accept", "application/json; version=3")
	client.edgefunctionsConfig.UserAgent = userAgent

	return client
}
