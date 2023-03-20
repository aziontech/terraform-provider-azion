package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "azion" {
  api_token  = "b79f65853ff3386e4b3678397a808fe0aaf6fd5c"
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"azion": providerserver.NewProtocol6WithError(New()),
	}
)
