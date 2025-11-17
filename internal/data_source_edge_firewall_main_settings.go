package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataSourceAzionEdgeFirewalls() datasource.DataSource {
	return &EdgeFirewallsDataSource{}
}

type EdgeFirewallsDataSource struct {
	client *apiClient
}

type EdgeFirewallsDataSourceModel struct {
	Page     types.Int64            `tfsdk:"page"`
	PageSize types.Int64            `tfsdk:"page_size"`
	Counter  types.Int64            `tfsdk:"counter"`
	Results  []EdgeFirewallsResults `tfsdk:"results"`
}

type EdgeFirewallsResults struct {
	ID             types.Int64         `tfsdk:"id"`
	Name           types.String        `tfsdk:"name"`
	Modules        EdgeFirewallModules `tfsdk:"modules"`
	Debug          types.Bool          `tfsdk:"debug"`
	Active         types.Bool          `tfsdk:"active"`
	LastEditor     types.String        `tfsdk:"last_editor"`
	LastModified   types.String        `tfsdk:"last_modified"`
	ProductVersion types.String        `tfsdk:"product_version"`
}

func (e *EdgeFirewallsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	e.client = req.ProviderData.(*apiClient)
}

func (e *EdgeFirewallsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_main_settings"
}

func (e *EdgeFirewallsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"page": schema.Int64Attribute{
				Description: "The page number of firewalls.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number of firewalls.",
				Optional:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of firewalls.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
		},
	}
}

func (e *EdgeFirewallsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var Page types.Int64
	var PageSize types.Int64

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &Page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &PageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}
	if PageSize.ValueInt64() == 0 {
		PageSize = types.Int64Value(10)
	}

	EdgeFirewallsResponse, response, err := e.client.edgeApi.FirewallsAPI.ListFirewalls(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			EdgeFirewallsResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedFirewallList, *http.Response, error) {
				return e.client.edgeApi.FirewallsAPI.ListFirewalls(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
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

	var edgeFirewallsResults []EdgeFirewallsResults
	for _, results := range EdgeFirewallsResponse.Results {
		mods := results.GetModules()
		ddosProtection := mods.GetDdosProtection()
		functions := mods.GetFunctions()
		networkProtection := mods.GetNetworkProtection()
		waf := mods.GetWaf()

		modules := EdgeFirewallModules{
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

		GetEdgeFirewalls := EdgeFirewallsResults{
			ID:             types.Int64Value(results.GetId()),
			Name:           types.StringValue(results.GetName()),
			Modules:        modules,
			Debug:          types.BoolValue(results.GetDebug()),
			Active:         types.BoolValue(results.GetActive()),
			LastEditor:     types.StringValue(results.GetLastEditor()),
			LastModified:   types.StringValue(results.GetLastModified().Format(time.RFC3339)),
			ProductVersion: types.StringValue(results.GetProductVersion()),
		}
		edgeFirewallsResults = append(edgeFirewallsResults, GetEdgeFirewalls)
	}

	EdgeFirewallsState := EdgeFirewallsDataSourceModel{
		Page:     Page,
		PageSize: PageSize,
		Counter:  types.Int64Value(EdgeFirewallsResponse.GetCount()),
		Results:  edgeFirewallsResults,
	}

	diags := resp.State.Set(ctx, &EdgeFirewallsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
