package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
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
		"azion": func() (tfprotov6.ProviderServer, error) {
			providers := []func() tfprotov6.ProviderServer{
				providerserver.NewProtocol6(New("test")),
			}

			return tf6muxserver.NewMuxServer(context.Background(), providers...)
		},
	}
)
