package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataSourceAzionEdgeFirewallEdgeFunctionsInstance() datasource.DataSource {
	return &EdgeFirewallEdgeFunctionsInstanceDataSource{}
}

type EdgeFirewallEdgeFunctionsInstanceDataSource struct {
	client *apiClient
}

type EdgeFirewallEdgeFunctionsInstanceDataSourceModel struct {
	ID             types.String                               `tfsdk:"id"`
	EdgeFirewallID types.String                               `tfsdk:"edge_firewall_id"`
	Counter        types.Int64                                `tfsdk:"counter"`
	Page           types.Int64                                `tfsdk:"page"`
	PageSize       types.Int64                                `tfsdk:"page_size"`
	Results        []EdgeFirewallEdgeFunctionsInstanceResults `tfsdk:"results"`
}

type EdgeFirewallEdgeFunctionsInstanceResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (e *EdgeFirewallEdgeFunctionsInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	e.client = req.ProviderData.(*apiClient)
}

func (e *EdgeFirewallEdgeFunctionsInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_edge_functions_instance"
}

func (e *EdgeFirewallEdgeFunctionsInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"edge_firewall_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Firewall",
				Required:    true,
			},
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
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "ID of the edge firewall edge functions instance.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the edge firewall edge functions instance.",
							Computed:    true,
						},
						"args": schema.StringAttribute{
							Description: "Arguments for the edge function instance.",
							Computed:    true,
						},
						"function": schema.Int64Attribute{
							Description: "ID of the Edge Function for Edge Firewall you wish to configure.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the edge function instance is active.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the edge firewall edge functions instance.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the edge firewall edge functions instance.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (e *EdgeFirewallEdgeFunctionsInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var edgeFirewallID types.String

	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("edge_firewall_id"), &edgeFirewallID)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	if page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	EdgeFirewallFunctionsInstanceResponse, response, err := e.client.edgeApi.FirewallsFunctionAPI.
		ListFirewallFunction(ctx, edgeFirewallID.ValueString()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64()).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			EdgeFirewallFunctionsInstanceResponse, response, err = utils.RetryOn429(func() (*edgeapi.PaginatedFirewallFunctionInstanceList, *http.Response, error) {
				return e.client.edgeApi.FirewallsFunctionAPI.
					ListFirewallFunction(ctx, edgeFirewallID.ValueString()).Page(page.ValueInt64()).
					PageSize(pageSize.ValueInt64()).Execute() //nolint
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

	var edgeFirewallsResults []EdgeFirewallEdgeFunctionsInstanceResults
	for _, results := range EdgeFirewallFunctionsInstanceResponse.Results {
		jsonArgsStr, err := utils.ConvertInterfaceToString(results.GetArgs())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
		}

		GetEdgeFirewalls := EdgeFirewallEdgeFunctionsInstanceResults{
			ID:           types.Int64Value(results.GetId()),
			Name:         types.StringValue(results.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(results.GetFunction()),
			Active:       types.BoolValue(results.GetActive()),
			LastEditor:   types.StringValue(results.GetLastEditor()),
			LastModified: types.StringValue(results.GetLastModified().Format(time.RFC3339)),
		}
		edgeFirewallsResults = append(edgeFirewallsResults, GetEdgeFirewalls)
	}

	EdgeFirewallsState := EdgeFirewallEdgeFunctionsInstanceDataSourceModel{
		ID:             types.StringValue("Get All Edge Firewall Edge Functions Instance"),
		EdgeFirewallID: edgeFirewallID,
		Counter:        types.Int64Value(EdgeFirewallFunctionsInstanceResponse.GetCount()),
		Page:           page,
		PageSize:       pageSize,
		Results:        edgeFirewallsResults,
	}

	diags := resp.State.Set(ctx, &EdgeFirewallsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
