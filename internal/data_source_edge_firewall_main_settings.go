package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/edgefirewall"
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
	Counter       types.Int64                 `tfsdk:"counter"`
	TotalPages    types.Int64                 `tfsdk:"total_pages"`
	Page          types.Int64                 `tfsdk:"page"`
	PageSize      types.Int64                 `tfsdk:"page_size"`
	Links         *EdgeFirewallsResponseLinks `tfsdk:"links"`
	SchemaVersion types.Int64                 `tfsdk:"schema_version"`
	Results       []EdgeFirewallsResults      `tfsdk:"results"`
}

type EdgeFirewallsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFirewallsResults struct {
	ID                       types.Int64  `tfsdk:"id"`
	LastEditor               types.String `tfsdk:"last_editor"`
	LastModified             types.String `tfsdk:"last_modified"`
	Name                     types.String `tfsdk:"name"`
	IsActive                 types.Bool   `tfsdk:"is_active"`
	EdgeFunctionsEnabled     types.Bool   `tfsdk:"edge_functions_enabled"`
	NetworkProtectionEnabled types.Bool   `tfsdk:"network_protection_enabled"`
	WAFEnabled               types.Bool   `tfsdk:"waf_enabled"`
	DebugRules               types.Bool   `tfsdk:"debug_rules"`
	Domains                  types.List   `tfsdk:"domains"`
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
			"counter": schema.Int64Attribute{
				Description: "The total number of edge firewalls.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of edge firewalls.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of edge firewalls.",
				Optional:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"links": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"previous": schema.StringAttribute{
						Computed: true,
					},
					"next": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "ID of the edge firewall rule set.",
							Required:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the edge firewall rule set.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the edge firewall rule set.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the edge firewall rule set.",
							Computed:    true,
						},
						"is_active": schema.BoolAttribute{
							Description: "Whether the edge firewall rule set is active.",
							Computed:    true,
						},
						"edge_functions_enabled": schema.BoolAttribute{
							Description: "Whether edge functions are enabled for the rule set.",
							Computed:    true,
						},
						"network_protection_enabled": schema.BoolAttribute{
							Description: "Whether network protection is enabled for the rule set.",
							Computed:    true,
						},
						"waf_enabled": schema.BoolAttribute{
							Description: "Whether Web Application Firewall (WAF) is enabled for the rule set.",
							Computed:    true,
						},
						"debug_rules": schema.BoolAttribute{
							Description: "Whether debug rules are enabled for the rule set.",
							Computed:    true,
						},
						"domains": schema.ListAttribute{
							Computed:    true,
							ElementType: types.Int64Type,
							Description: "List of domains associated with the edge firewall rule set.",
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

	EdgeFirewallsResponse, response, err := e.client.edgeFirewallApi.DefaultAPI.EdgeFirewallGet(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgefirewall.ListEdgeFirewallResponse, *http.Response, error) {
				return e.client.edgeFirewallApi.DefaultAPI.EdgeFirewallGet(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
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

	var previous, next string
	if EdgeFirewallsResponse.Links.Previous.Get() != nil {
		previous = *EdgeFirewallsResponse.Links.Previous.Get()
	}
	if EdgeFirewallsResponse.Links.Next.Get() != nil {
		next = *EdgeFirewallsResponse.Links.Next.Get()
	}

	var edgeFirewallsResults []EdgeFirewallsResults
	for _, results := range EdgeFirewallsResponse.Results {
		var sliceInt []types.Int64
		for _, itemsValuesInt := range results.GetDomains() {
			sliceInt = append(sliceInt, types.Int64Value(itemsValuesInt))
		}

		GetEdgeFirewalls := EdgeFirewallsResults{
			ID:                       types.Int64Value(results.GetId()),
			LastEditor:               types.StringValue(results.GetLastEditor()),
			LastModified:             types.StringValue(results.GetLastModified()),
			Name:                     types.StringValue(results.GetName()),
			IsActive:                 types.BoolValue(results.GetIsActive()),
			EdgeFunctionsEnabled:     types.BoolValue(results.GetEdgeFunctionsEnabled()),
			NetworkProtectionEnabled: types.BoolValue(results.GetNetworkProtectionEnabled()),
			WAFEnabled:               types.BoolValue(results.GetWafEnabled()),
			DebugRules:               types.BoolValue(results.GetDebugRules()),
			Domains:                  utils.SliceIntTypeToList(sliceInt),
		}
		edgeFirewallsResults = append(edgeFirewallsResults, GetEdgeFirewalls)
	}

	EdgeFirewallsState := EdgeFirewallsDataSourceModel{
		SchemaVersion: types.Int64Value(EdgeFirewallsResponse.GetSchemaVersion()),
		Counter:       types.Int64Value(EdgeFirewallsResponse.GetCount()),
		TotalPages:    types.Int64Value(EdgeFirewallsResponse.GetTotalPages()),
		Page:          Page,
		PageSize:      PageSize,
		Links: &EdgeFirewallsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
		Results: edgeFirewallsResults,
	}

	diags := resp.State.Set(ctx, &EdgeFirewallsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
