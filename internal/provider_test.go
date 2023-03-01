package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "azion" {
  api_token  = "4ebf949f99d68d2092dc50284ce7269ef6413193"
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"azion": providerserver.NewProtocol6WithError(New()),
	}
)
