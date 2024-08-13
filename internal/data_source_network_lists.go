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

var (
	_ datasource.DataSource              = &NetworkListsDataSource{}
	_ datasource.DataSourceWithConfigure = &NetworkListsDataSource{}
)

func dataSourceAzionNetworkLists() datasource.DataSource {
	return &NetworkListsDataSource{}
}

type NetworkListsDataSource struct {
	client *apiClient
}

type NetworkListsDataSourceModel struct {
	SchemaVersion types.Int64                `tfsdk:"schema_version"`
	Counter       types.Int64                `tfsdk:"counter"`
	Page          types.Int64                `tfsdk:"page"`
	TotalPages    types.Int64                `tfsdk:"total_pages"`
	Links         *NetworkListsResponseLinks `tfsdk:"links"`
	Results       []NetworkListsResults      `tfsdk:"results"`
	ID            types.String               `tfsdk:"id"`
}

type NetworkListsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type NetworkListsResults struct {
	ID           types.Int64  `tfsdk:"id"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	ListType     types.String `tfsdk:"list_type"`
	Name         types.String `tfsdk:"name"`
	CountryList  types.List   `tfsdk:"country_list"`
	IPList       types.List   `tfsdk:"ip_list"`
}

func (n *NetworkListsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	n.client = req.ProviderData.(*apiClient)
}

func (n *NetworkListsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_lists"
}

func (n *NetworkListsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of Cache Settings.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Cache Settings.",
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
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "ID of the network list.",
							Required:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the network list.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the network list.",
							Computed:    true,
						},
						"list_type": schema.StringAttribute{
							Description: "Type of the network list.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the network list.",
							Computed:    true,
						},
						"country_list": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of countries in the network list.",
						},
						"ip_list": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of IP addresses in the network list.",
						},
					},
				},
			},
		},
	}
}

func (n *NetworkListsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var Page types.Int64
	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &Page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}

	networkListsResponse, response, err := n.client.networkListApi.DefaultAPI.NetworkListsGet(ctx).Page(int32(Page.ValueInt64())).Execute()
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
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
	defer response.Body.Close()

	var networkLists []NetworkListsResults
	for _, nl := range networkListsResponse.Results {
		var sliceCountryList []types.String
		for _, countryList := range nl.GetCountryList() {
			sliceCountryList = append(sliceCountryList, types.StringValue(countryList))
		}

		var sliceIpList []types.String
		for _, ipList := range nl.GetIpList() {
			sliceIpList = append(sliceIpList, types.StringValue(ipList))
		}

		networkList := NetworkListsResults{
			ID:           types.Int64Value(nl.GetId()),
			LastEditor:   types.StringValue(nl.GetLastEditor()),
			LastModified: types.StringValue(nl.GetLastModified()),
			ListType:     types.StringValue(nl.GetListType()),
			Name:         types.StringValue(nl.GetName()),
			CountryList:  utils.SliceStringTypeToList(sliceCountryList),
			IPList:       utils.SliceStringTypeToList(sliceIpList),
		}
		networkLists = append(networkLists, networkList)
	}

	networkListsState := NetworkListsDataSourceModel{
		SchemaVersion: types.Int64Value(networkListsResponse.GetSchemaVersion()),
		Counter:       types.Int64Value(networkListsResponse.GetCount()),
		Page:          types.Int64Value(Page.ValueInt64()),
		TotalPages:    types.Int64Value(networkListsResponse.GetTotalPages()),
		Links: &NetworkListsResponseLinks{
			Previous: types.StringValue(networkListsResponse.Links.GetPrevious()),
			Next:     types.StringValue(networkListsResponse.Links.GetNext()),
		},
		Results: networkLists,
		ID:      types.StringValue("Get All Network Lists"),
	}

	diags := resp.State.Set(ctx, &networkListsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
