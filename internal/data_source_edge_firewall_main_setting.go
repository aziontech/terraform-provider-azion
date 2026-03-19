package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataSourceAzionFirewall() datasource.DataSource {
	return &FirewallDataSource{}
}

type FirewallDataSource struct {
	client *apiClient
}

type FirewallDataSourceModel struct {
	ID         types.String    `tfsdk:"id"`
	FirewallID types.Int64     `tfsdk:"firewall_id"`
	Data       FirewallResults `tfsdk:"data"`
}

type FirewallModules struct {
	DdosProtection    *DdosProtectionModule    `tfsdk:"ddos_protection"`
	Functions         *FunctionsModule         `tfsdk:"functions"`
	NetworkProtection *NetworkProtectionModule `tfsdk:"network_protection"`
	WAF               *WAFModule               `tfsdk:"waf"`
}

type DdosProtectionModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type FunctionsModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type NetworkProtectionModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type WAFModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type FirewallResults struct {
	ID             types.Int64     `tfsdk:"id"`
	Name           types.String    `tfsdk:"name"`
	Modules        FirewallModules `tfsdk:"modules"`
	Debug          types.Bool      `tfsdk:"debug"`
	Active         types.Bool      `tfsdk:"active"`
	LastEditor     types.String    `tfsdk:"last_editor"`
	LastModified   types.String    `tfsdk:"last_modified"`
	ProductVersion types.String    `tfsdk:"product_version"`
}

func (f *FirewallDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	f.client = req.ProviderData.(*apiClient)
}

func (f *FirewallDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_main_setting"
}

func (f *FirewallDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall identifier.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the firewall rule set.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the firewall rule set.",
						Computed:    true,
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Modules configuration for the firewall.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"ddos_protection": schema.SingleNestedAttribute{
								Description: "DDoS protection module configuration.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether DDoS protection is enabled.",
										Computed:    true,
									},
								},
							},
							"functions": schema.SingleNestedAttribute{
								Description: "Functions module configuration.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether functions are enabled.",
										Computed:    true,
									},
								},
							},
							"network_protection": schema.SingleNestedAttribute{
								Description: "Network protection module configuration.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether network protection is enabled.",
										Computed:    true,
									},
								},
							},
							"waf": schema.SingleNestedAttribute{
								Description: "WAF module configuration.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Description: "Whether WAF is enabled.",
										Computed:    true,
									},
								},
							},
						},
					},
					"debug": schema.BoolAttribute{
						Description: "Whether debug is enabled for the rule set.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the firewall rule set is active.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the firewall rule set.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the firewall rule set.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the firewall rule set.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (f *FirewallDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getFirewallID types.Int64
	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &getFirewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallResponse, response, err := f.client.api.FirewallsAPI.RetrieveFirewall(ctx, getFirewallID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallResponse, *http.Response, error) {
				return f.client.api.FirewallsAPI.RetrieveFirewall(ctx, getFirewallID.ValueInt64()).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(
					errReadAll.Error(),
					"err",
				)
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(
				err.Error(),
				bodyString,
			)
			return
		}
	}

	mods := firewallResponse.Data.GetModules()
	ddosProtection := mods.GetDdosProtection()
	functions := mods.GetFunctions()
	networkProtection := mods.GetNetworkProtection()
	waf := mods.GetWaf()

	modules := FirewallModules{
		DdosProtection: &DdosProtectionModule{
			Enabled: types.BoolValue(ddosProtection.GetEnabled()),
		},
		Functions: &FunctionsModule{
			Enabled: types.BoolValue(functions.GetEnabled()),
		},
		NetworkProtection: &NetworkProtectionModule{
			Enabled: types.BoolValue(networkProtection.GetEnabled()),
		},
		WAF: &WAFModule{
			Enabled: types.BoolValue(waf.GetEnabled()),
		},
	}

	firewallResults := FirewallResults{
		ID:             types.Int64Value(firewallResponse.Data.GetId()),
		Name:           types.StringValue(firewallResponse.Data.GetName()),
		Modules:        modules,
		Debug:          types.BoolValue(firewallResponse.Data.GetDebug()),
		Active:         types.BoolValue(firewallResponse.Data.GetActive()),
		LastEditor:     types.StringValue(firewallResponse.Data.GetLastEditor()),
		LastModified:   types.StringValue(firewallResponse.Data.GetLastModified().Format(time.RFC3339)),
		ProductVersion: types.StringValue(firewallResponse.Data.GetProductVersion()),
	}

	firewallState := FirewallDataSourceModel{
		FirewallID: getFirewallID,
		Data:       firewallResults,
	}

	firewallState.ID = types.StringValue("Get Firewall by ID")
	diags := resp.State.Set(ctx, &firewallState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
