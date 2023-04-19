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
  api_token  = "<token>"
}
`
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"azionProvider": func() (tfprotov6.ProviderServer, error) {
			ctx := context.Background()
			providers := []func() tfprotov6.ProviderServer{
				providerserver.NewProtocol6(New("test")),
			}

			return tf6muxserver.NewMuxServer(ctx, providers...)
		},
	}
)
