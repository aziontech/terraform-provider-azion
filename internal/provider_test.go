package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "azion" {
  api_token  = "azion9252873bb250dfac51625125cfd702e57af"
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"azion": providerserver.NewProtocol6WithError(New()),
	}
)
