package main

import (
	"context"
	"flag"
	"log"
	"time"

	framework "github.com/aziontech/terraform-provider-azion/internal"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

// Run "go generate" to format example terraform files and generate the provider docs

// Format examples
//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

var (
	version string = "dev"
)

func main() {
log.Println("Garantindo que isto está rodando agora a partil de agora: ", time.Now().Hour(), " - ", time.Now().Minute())

	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	ctx := context.Background()
	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(framework.New(version)),
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)

	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt
	if debug {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/aziontech/azion",
		muxServer.ProviderServer,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err.Error())
	}
}
