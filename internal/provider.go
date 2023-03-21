package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/aziontech/azionapi-go-sdk/idns"
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
	version string
}

func New() provider.Provider {
	return &azionProvider{}
}

func (p *azionProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azion"
}

func (p *azionProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: fmt.Sprintf("The API Token for operations. Must provide `api_token`"),
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
	tflog.Debug(ctx, fmt.Sprintf("Configuring Azion Client"))
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

	ctx = tflog.SetField(ctx, "AZION_TERRAFORM_TOKEN", APIToken)

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
