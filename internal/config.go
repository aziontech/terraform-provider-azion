package provider

import (
	"os"

	"github.com/aziontech/azionapi-go-sdk/domains"
	"github.com/aziontech/azionapi-go-sdk/idns"
)

type azClient struct {
	APIToken      string
	IdnsConfig    *idns.Configuration
	DomainsConfig *domains.Configuration
}

type apiClient struct {
	idnsApi    *idns.APIClient
	domainsApi *domains.APIClient
}

func Client() *apiClient {
	APIToken := os.Getenv("api_token")
	var config AzionProviderModel
	if !config.APIToken.IsNull() {
		APIToken = config.APIToken.ValueString()
	}

	client := &azClient{
		IdnsConfig:    idns.NewConfiguration(),
		DomainsConfig: domains.NewConfiguration(),
	}

	var apiClients *apiClient

	domainsConfig := client.DomainsConfig
	domainsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	apiClients.domainsApi = domains.NewAPIClient(domainsConfig)

	idnsConfig := client.IdnsConfig
	idnsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	apiClients.idnsApi = idns.NewAPIClient(idnsConfig)

	return apiClients
}
