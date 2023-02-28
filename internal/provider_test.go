package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	providerConfig = `
provider "azion" {
  api_token  = "ae21724627cc5f3fac461060caf86ac0dacfa00f"
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"azion": providerserver.NewProtocol6WithError(New()),
	}
)
