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
	Data          *NetworkListData `tfsdk:"data"`
	NetworkListID types.String     `tfsdk:"network_list_id"`
	ID            types.String     `tfsdk:"id"`
}

type NetworkListData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	Items        types.List   `tfsdk:"items"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	Active       types.Bool   `tfsdk:"active"`
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
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true,
			},
			"network_list_id": schema.StringAttribute{
				Description: "The network list identifier.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
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
					"items": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "List of items in the network list.",
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
	}
}

func (n *NetworkListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var networkListID types.String
	diags := req.Config.GetAttribute(ctx, path.Root("network_list_id"), &networkListID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListResponse, response, err := n.client.edgeApi.NetworkListsAPI.
		RetrieveNetworkList(ctx, networkListID.ValueString()).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			networkListResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseRetrieveNetworkListDetail, *http.Response, error) {
				return n.client.edgeApi.NetworkListsAPI.RetrieveNetworkList(ctx, networkListID.ValueString()).Execute() //nolint
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
			if response != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(
						errReadAll.Error(),
						"Failed to read response body",
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

	// Convert items to string slice
	var items []types.String
	data := networkListResponse.GetData()
	for _, item := range data.Items {
		items = append(items, types.StringValue(item))
	}

	networkListState := NetworkListDataSourceModel{
		NetworkListID: networkListID,
		Data: &NetworkListData{
			ID:           types.Int64Value(data.GetId()),
			Name:         types.StringValue(data.GetName()),
			Type:         types.StringValue(data.GetType()),
			Items:        utils.SliceStringTypeToList(items),
			LastEditor:   types.StringValue(data.GetLastEditor()),
			LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
			Active:       types.BoolValue(data.GetActive()),
		},
		ID: types.StringValue("Get Network List By ID"),
	}

	diags = resp.State.Set(ctx, &networkListState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
