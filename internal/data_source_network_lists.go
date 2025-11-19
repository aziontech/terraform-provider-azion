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
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	Counter types.Int64           `tfsdk:"counter"`
	Results []NetworkListsResults `tfsdk:"results"`
	ID      types.String          `tfsdk:"id"`
}

type NetworkListsResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	Active       types.Bool   `tfsdk:"active"`
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
			"counter": schema.Int64Attribute{
				Description: "The total number of network lists.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "ID of the network list.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the network list.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of the network list.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the network list.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the network list.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the network list is active.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (n *NetworkListsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	networkListsResponse, response, err := n.client.edgeApi.NetworkListsAPI.
		ListNetworkLists(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			networkListsResponse, response, err = utils.RetryOn429(func() (*edgeapi.PaginatedNetworkListList, *http.Response, error) {
				return n.client.edgeApi.NetworkListsAPI.ListNetworkLists(ctx).Execute() //nolint
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

	var networkLists []NetworkListsResults
	for _, nl := range networkListsResponse.Results {
		networkList := NetworkListsResults{
			ID:           types.Int64Value(nl.GetId()),
			Name:         types.StringValue(nl.GetName()),
			Type:         types.StringValue(nl.GetType()),
			LastEditor:   types.StringValue(nl.GetLastEditor()),
			LastModified: types.StringValue(nl.GetLastModified().Format(time.RFC3339)),
			Active:       types.BoolValue(nl.GetActive()),
		}
		networkLists = append(networkLists, networkList)
	}

	networkListsState := NetworkListsDataSourceModel{
		Counter: types.Int64Value(networkListsResponse.GetCount()),
		Results: networkLists,
		ID:      types.StringValue("Get All Network Lists"),
	}

	diags := resp.State.Set(ctx, &networkListsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
