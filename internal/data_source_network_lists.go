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
	Counter    types.Int64                `tfsdk:"counter"`
	Page       types.Int64                `tfsdk:"page"`
	TotalPages types.Int64                `tfsdk:"total_pages"`
	Links      *NetworkListsResponseLinks `tfsdk:"links"`
	Results    []NetworkListsResults      `tfsdk:"results"`
	ID         types.String               `tfsdk:"id"`
}

type NetworkListsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type NetworkListsResults struct {
	ID           types.Int64  `tfsdk:"id"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	IsVersioned  types.Bool   `tfsdk:"is_versioned"`
	Version      types.Int64  `tfsdk:"version"`
	VersionState types.String `tfsdk:"version_state"`
	VersionID    types.String `tfsdk:"version_id"`
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
			"page": schema.Int64Attribute{
				Description: "The page number of network lists.",
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
						"is_versioned": schema.BoolAttribute{
							Description: "Whether the network list is versioned.",
							Computed:    true,
						},
						"version": schema.Int64Attribute{
							Description: "The current version of the network list.",
							Computed:    true,
						},
						"version_state": schema.StringAttribute{
							Description: "The state of the current network list version.",
							Computed:    true,
						},
						"version_id": schema.StringAttribute{
							Description: "The identifier of the current network list version.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (n *NetworkListsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	if page.IsNull() || page.IsUnknown() {
		page = types.Int64Value(1)
	}

	page32, err := utils.CheckInt64toInt32Security(page.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, page.ValueInt64())
		return
	}

	networkListsResponse, response, err := n.client.api.NetworkListsAPI.ListNetworkLists(ctx).Page(int64(page32)).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			networkListsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedNetworkListSummaryList, *http.Response, error) {
				return n.client.api.NetworkListsAPI.ListNetworkLists(ctx).Page(int64(page32)).Execute() //nolint
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

	var networkLists []NetworkListsResults
	for _, nl := range networkListsResponse.GetResults() {
		networkList := NetworkListsResults{
			ID:           types.Int64Value(nl.GetId()),
			LastEditor:   types.StringValue(nl.GetLastEditor()),
			LastModified: types.StringValue(nl.GetLastModified().Format(time.RFC3339)),
			CreatedAt:    types.StringValue(nl.GetCreatedAt().Format(time.RFC3339)),
			Type:         types.StringValue(nl.GetType()),
			Name:         types.StringValue(nl.GetName()),
			IsVersioned:  types.BoolValue(nl.IsVersioned),
			Version:      types.Int64PointerValue(nl.Version.Get()),
			VersionState: types.StringPointerValue(nl.VersionState.Get()),
			VersionID:    types.StringPointerValue(nl.VersionId.Get()),
		}
		networkLists = append(networkLists, networkList)
	}

	networkListsState := NetworkListsDataSourceModel{
		Counter:    types.Int64Value(networkListsResponse.GetCount()),
		Page:       types.Int64Value(networkListsResponse.GetPage()),
		TotalPages: types.Int64Value(networkListsResponse.GetTotalPages()),
		Links:      populateNetworkListsLinks(networkListsResponse),
		Results:    networkLists,
		ID:         types.StringValue("Get All Network Lists"),
	}

	diags := resp.State.Set(ctx, &networkListsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func populateNetworkListsLinks(response *azionapi.PaginatedNetworkListSummaryList) *NetworkListsResponseLinks {
	links := &NetworkListsResponseLinks{
		Previous: types.StringValue(response.GetPrevious()),
		Next:     types.StringValue(response.GetNext()),
	}
	return links
}
