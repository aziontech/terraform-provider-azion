package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/aziontech/terraform-provider-azion/internal/consts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &azionProvider{}
)

type AzionProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
}

type azionProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func (p *azionProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azion"
	resp.Version = p.version
}

func (p *azionProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Azion provider is used to interact with resources supported by Azion. The provider needs to be configured with the proper credentials before it can be used.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Description: "A registered token for Azion API - https://api.azion.com/#authentication-types. Alternatively, can be configured using the environment variable.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[A-Za-z0-9-_]{40}`),
						"API tokens must be 40 characters long and only contain characters a-z, A-Z, 0-9, hyphens and underscores",
					),
				},
			},
		},
	}
}

func (p *azionProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config AzionProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	APIToken := os.Getenv("api_token")

	if !config.APIToken.IsNull() {
		APIToken = config.APIToken.ValueString()
	}
	if resp.Diagnostics.HasError() {
		return
	}
	idnsConfig := idns.NewConfiguration()
	idnsConfig.AddDefaultHeader("Authorization", "token "+APIToken)
	userAgent := fmt.Sprintf(consts.UserAgentDefault, req.TerraformVersion, p.version)
	tflog.Debug(ctx, "Estou aqui "+userAgent)
	idnsConfig.UserAgent = userAgent

	client := idns.NewAPIClient(idnsConfig)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		dataSourceAzionZone,
		dataSourceAzionZones,
		dataSourceAzionRecords,
		dataSourceAzionDNSSec,
	}
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZoneResource,
		NewRecordResource,
		NewDnssecResource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &azionProvider{
			version: version,
		}
	}
}
