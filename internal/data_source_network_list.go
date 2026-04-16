package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &NetworkListDataSource{}
	_ datasource.DataSourceWithConfigure = &NetworkListDataSource{}
)

func dataSourceAzionNetworkList() datasource.DataSource {
	return &NetworkListDataSource{}
}

type NetworkListDataSource struct {
	client *apiClient
}

type NetworkListDataSourceModel struct {
	ID      types.Int64        `tfsdk:"id"`
	Results *NetworkListResult `tfsdk:"results"`
}

type NetworkListResult struct {
	ID           types.Int64  `tfsdk:"id"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	Items        types.List   `tfsdk:"items"`
}

func (n *NetworkListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	n.client = req.ProviderData.(*apiClient)
}

func (n *NetworkListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_list"
}

func (n *NetworkListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Identifier of the network list.",
				Required:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the network list.",
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
					"created_at": schema.StringAttribute{
						Description: "Creation timestamp of the network list.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the network list. Can be: asn, countries, or ip_cidr.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the network list.",
						Computed:    true,
					},
					"items": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "List of items in the network list. Contents depend on the type: country codes, IP addresses, or ASN numbers.",
					},
				},
			},
		},
	}
}

func (n *NetworkListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var networkListID types.Int64
	diagsID := req.Config.GetAttribute(ctx, path.Root("id"), &networkListID)
	resp.Diagnostics.Append(diagsID...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListResponse, response, err := n.client.api.NetworkListsAPI.RetrieveNetworkList(ctx, networkListID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			networkListResponse, response, err = utils.RetryOn429(func() (*azionapi.NetworkListResponse, *http.Response, error) {
				return n.client.api.NetworkListsAPI.RetrieveNetworkList(ctx, networkListID.ValueInt64()).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			if response != nil && response.Body != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(
						errReadAll.Error(),
						"error reading response from API",
					)
				}
				bodyString := string(bodyBytes)
				resp.Diagnostics.AddError(
					err.Error(),
					bodyString,
				)
			} else {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed",
				)
			}
			return
		}
	}

	networkListState := populateNetworkListResult(networkListResponse.GetData())
	diags := resp.State.Set(ctx, &networkListState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func populateNetworkListResult(data azionapi.NetworkList) NetworkListDataSourceModel {
	var itemsSlice []types.String
	for _, item := range data.GetItems() {
		itemsSlice = append(itemsSlice, types.StringValue(item))
	}

	return NetworkListDataSourceModel{
		ID: types.Int64Value(data.GetId()),
		Results: &NetworkListResult{
			ID:           types.Int64Value(data.GetId()),
			LastEditor:   types.StringValue(data.GetLastEditor()),
			LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
			CreatedAt:    types.StringValue(data.GetCreatedAt().Format(time.RFC3339)),
			Type:         types.StringValue(data.GetType()),
			Name:         types.StringValue(data.GetName()),
			Items:        utils.SliceStringTypeToList(itemsSlice),
		},
	}
}
