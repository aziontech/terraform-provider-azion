package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/networklist"
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
	SchemaVersion types.Int64        `tfsdk:"schema_version"`
	Results       *NetworkListResult `tfsdk:"results"`
	NetworkListID types.String       `tfsdk:"network_list_id"`
	ID            types.String       `tfsdk:"id"`
}

type NetworkListResult struct {
	LastEditor     types.String `tfsdk:"last_editor"`
	LastModified   types.String `tfsdk:"last_modified"`
	ListType       types.String `tfsdk:"list_type"`
	Name           types.String `tfsdk:"name"`
	ItemsValuesStr types.List   `tfsdk:"items_values_str"`
	ItemsValuesInt types.List   `tfsdk:"items_values_int"`
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
				Description: "The edge application identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
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
					"items_values_str": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "List of countries in the network list.",
					},
					"items_values_int": schema.ListAttribute{
						Computed:    true,
						ElementType: types.Int64Type,
						Description: "List of countries in the network list.",
					},
				},
			},
		},
	}
}

func (n *NetworkListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid types.String
	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("network_list_id"), &uuid)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListsResponse, response, err := n.client.networkListApi.DefaultAPI.NetworkListsUuidGet(ctx, uuid.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*networklist.NetworkListUuidResponse, *http.Response, error) {
				return n.client.networkListApi.DefaultAPI.NetworkListsUuidGet(ctx, uuid.ValueString()).Execute() //nolint
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

	var sliceString []types.String
	for _, itemsValuesStr := range networkListsResponse.GetResults().NetworkListUuidResponseEntryString.GetItemsValues() {
		sliceString = append(sliceString, types.StringValue(itemsValuesStr))
	}
	var sliceInt []types.Int64
	for _, itemsValuesInt := range networkListsResponse.GetResults().NetworkListUuidResponseEntryInt.GetItemsValues() {
		sliceInt = append(sliceInt, types.Int64Value(int64(itemsValuesInt)))
	}

	if len(sliceString) != 0 {
		networkListsState := NetworkListDataSourceModel{
			SchemaVersion: types.Int64Value(networkListsResponse.GetSchemaVersion()),
			Results: &NetworkListResult{
				LastEditor:     types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryString.GetLastEditor()),
				LastModified:   types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryString.GetLastModified()),
				ListType:       types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryString.GetListType()),
				Name:           types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryString.GetName()),
				ItemsValuesStr: utils.SliceStringTypeToList(sliceString),
				ItemsValuesInt: types.ListValueMust(types.Int64Type, nil),
			},
			ID: types.StringValue("Get By ID Network List"),
		}
		diags := resp.State.Set(ctx, &networkListsState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		networkListsState := NetworkListDataSourceModel{
			SchemaVersion: types.Int64Value(networkListsResponse.GetSchemaVersion()),
			Results: &NetworkListResult{
				LastEditor:     types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryInt.GetLastEditor()),
				LastModified:   types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryInt.GetLastModified()),
				ListType:       types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryInt.GetListType()),
				Name:           types.StringValue(networkListsResponse.GetResults().NetworkListUuidResponseEntryInt.GetName()),
				ItemsValuesStr: types.ListValueMust(types.StringType, nil),
				ItemsValuesInt: utils.SliceIntTypeToList(sliceInt),
			},
			ID: types.StringValue("Get By ID Network List"),
		}
		diags := resp.State.Set(ctx, &networkListsState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}
