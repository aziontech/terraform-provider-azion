package provider

import (
	"context"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
)

func dataSourceAzionEdgeFirewallEdgeFunctionsInstance() datasource.DataSource {
	return &EdgeFirewallEdgeFunctionsInstanceDataSource{}
}

type EdgeFirewallEdgeFunctionsInstanceDataSource struct {
	client *apiClient
}

type EdgeFirewallEdgeFunctionsInstanceDataSourceModel struct {
	ID             types.String                               `tfsdk:"id"`
	EdgeFirewallID types.Int64                                `tfsdk:"edge_firewall_id"`
	Counter        types.Int64                                `tfsdk:"counter"`
	TotalPages     types.Int64                                `tfsdk:"total_pages"`
	Page           types.Int64                                `tfsdk:"page"`
	PageSize       types.Int64                                `tfsdk:"page_size"`
	Links          *EdgeFirewallsResponseLinks                `tfsdk:"links"`
	SchemaVersion  types.Int64                                `tfsdk:"schema_version"`
	Results        []EdgeFirewallEdgeFunctionsInstanceResults `tfsdk:"results"`
}

type EdgeFirewallEdgeFunctionsInstanceResults struct {
	ID           types.Int64  `tfsdk:"edge_function_instance_id"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	Name         types.String `tfsdk:"name"`
	JsonArgs     types.String `tfsdk:"json_args"`
	EdgeFunction types.Int64  `tfsdk:"edge_function"`
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
						"edge_function_instance_id": schema.Int64Attribute{
							Description: "ID of the edge firewall edge functions instance.",
							Required:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the edge firewall edge functions instance.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the edge firewall edge functions instance.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the edge firewall edge functions instance.",
							Computed:    true,
						},
						"json_args": schema.StringAttribute{
							Description: "Requisition status code and message.",
							Computed:    true,
						},
						"edge_function": schema.Int64Attribute{
							Description: "ID of the Edge Function for Edge Firewall you with to configure.",
							Computed:    true},
					},
				},
			},
		},
	}
}

func (e *EdgeFirewallEdgeFunctionsInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var edgeFirewallID types.Int64

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

	EdgeFirewallFunctionsInstanceResponse, response, err := e.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.EdgeFirewallEdgeFirewallIdFunctionsInstancesGet(ctx, edgeFirewallID.ValueInt64()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64()).
		Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	var edgeFirewallsResults []EdgeFirewallEdgeFunctionsInstanceResults
	for _, results := range EdgeFirewallFunctionsInstanceResponse.Results {
		jsonArgsStr, err := utils.ConvertInterfaceToString(results.GetJsonArgs())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
		}
		GetEdgeFirewalls := EdgeFirewallEdgeFunctionsInstanceResults{
			ID:           types.Int64Value(results.GetId()),
			LastEditor:   types.StringValue(results.GetLastEditor()),
			LastModified: types.StringValue(results.GetLastModified()),
			Name:         types.StringValue(results.GetName()),
			EdgeFunction: types.Int64Value(results.GetEdgeFunction()),
			JsonArgs:     types.StringValue(jsonArgsStr),
		}
		edgeFirewallsResults = append(edgeFirewallsResults, GetEdgeFirewalls)
	}

	EdgeFirewallsState := EdgeFirewallEdgeFunctionsInstanceDataSourceModel{
		SchemaVersion:  types.Int64Value(EdgeFirewallFunctionsInstanceResponse.GetSchemaVersion()),
		ID:             types.StringValue("Get All Edge Firewall Edge Functions Instance"),
		EdgeFirewallID: edgeFirewallID,
		Counter:        types.Int64Value(EdgeFirewallFunctionsInstanceResponse.GetCount()),
		TotalPages:     types.Int64Value(EdgeFirewallFunctionsInstanceResponse.GetTotalPages()),
		Page:           page,
		PageSize:       pageSize,
		Links: &EdgeFirewallsResponseLinks{
			Previous: types.StringValue(EdgeFirewallFunctionsInstanceResponse.Links.GetPrevious()),
			Next:     types.StringValue(EdgeFirewallFunctionsInstanceResponse.Links.GetNext()),
		},
		Results: edgeFirewallsResults,
	}

	diags := resp.State.Set(ctx, &EdgeFirewallsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
