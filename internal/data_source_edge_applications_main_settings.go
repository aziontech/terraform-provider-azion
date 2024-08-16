package provider

import (
	"context"
	"io"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &EdgeApplicationsDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeApplicationsDataSource{}
)

func dataSourceAzionEdgeApplications() datasource.DataSource {
	return &EdgeApplicationsDataSource{}
}

type EdgeApplicationsDataSource struct {
	client *apiClient
}

type EdgeApplicationsDataSourceModel struct {
	SchemaVersion types.Int64                       `tfsdk:"schema_version"`
	Counter       types.Int64                       `tfsdk:"counter"`
	TotalPages    types.Int64                       `tfsdk:"total_pages"`
	Page          types.Int64                       `tfsdk:"page"`
	PageSize      types.Int64                       `tfsdk:"page_size"`
	Links         *GetEdgeApplicationsResponseLinks `tfsdk:"links"`
	Results       []EdgeApplicationsResult          `tfsdk:"results"`
	ID            types.String                      `tfsdk:"id"`
}
type GetEdgeApplicationsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeApplicationsResult struct {
	ApplicationID types.Int64          `tfsdk:"application_id"`
	Name          types.String         `tfsdk:"name"`
	Active        types.Bool           `tfsdk:"active"`
	DebugRules    types.Bool           `tfsdk:"debug_rules"`
	LastEditor    types.String         `tfsdk:"last_editor"`
	LastModified  types.String         `tfsdk:"last_modified"`
	Origins       []ApplicationOrigins `tfsdk:"origins"`
}

type ApplicationOrigins struct {
	Name       types.String `tfsdk:"name"`
	OriginType types.String `tfsdk:"origin_type"`
	OriginID   types.String `tfsdk:"origin_id"`
}

func (e *EdgeApplicationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	e.client = req.ProviderData.(*apiClient)
}

func (e *EdgeApplicationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_applications_main_settings"
}

func (e *EdgeApplicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of edge applications.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of edge applications.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of edge applications.",
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
						"application_id": schema.Int64Attribute{
							Description: "The edge application identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the edge application.",
						},
						"active": schema.BoolAttribute{
							Computed:    true,
							Description: "Indicates if the edge application is active.",
						},
						"debug_rules": schema.BoolAttribute{
							Computed:    true,
							Description: "Indicates if debug rules are enabled for the edge application.",
						},
						"last_editor": schema.StringAttribute{
							Computed:    true,
							Description: "The email of the last editor of the edge application.",
						},
						"last_modified": schema.StringAttribute{
							Computed:    true,
							Description: "The timestamp of the last modification of the edge application.",
						},
						"origins": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Computed:    true,
										Description: "Name of the origin.",
									},
									"origin_type": schema.StringAttribute{
										Computed:    true,
										Description: "Type of the origin.",
									},
									"origin_id": schema.StringAttribute{
										Computed:    true,
										Description: "Identifier of the origin.",
									},
								},
							},
							Description: "List of origins associated with the edge application.",
						},
					},
				},
			},
		},
	}
}

func (e *EdgeApplicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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

	edgeAppResponse, response, err := e.client.edgeApplicationsApi.EdgeApplicationsMainSettingsAPI.EdgeApplicationsGet(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
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

	var previous, next string
	if edgeAppResponse.Links.Previous.Get() != nil {
		previous = *edgeAppResponse.Links.Previous.Get()
	}
	if edgeAppResponse.Links.Next.Get() != nil {
		next = *edgeAppResponse.Links.Next.Get()
	}

	edgeApplicationsState := EdgeApplicationsDataSourceModel{
		Page:          Page,
		PageSize:      PageSize,
		SchemaVersion: types.Int64Value(edgeAppResponse.SchemaVersion),
		TotalPages:    types.Int64Value(edgeAppResponse.TotalPages),
		Counter:       types.Int64Value(edgeAppResponse.Count),
		Links: &GetEdgeApplicationsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}
	for _, resultEdgeApplication := range edgeAppResponse.GetResults() {
		edgeApplicationsState.Results = append(edgeApplicationsState.Results, EdgeApplicationsResult{
			ApplicationID: types.Int64Value(resultEdgeApplication.GetId()),
			Name:          types.StringValue(resultEdgeApplication.GetName()),
			Active:        types.BoolValue(resultEdgeApplication.GetActive()),
			DebugRules:    types.BoolValue(resultEdgeApplication.GetDebugRules()),
			LastEditor:    types.StringValue(resultEdgeApplication.GetLastEditor()),
			LastModified:  types.StringValue(resultEdgeApplication.GetLastModified()),
			Origins:       GetOrigins(resultEdgeApplication.GetOrigins()),
		})
	}
	edgeApplicationsState.ID = types.StringValue("Get All Edge Application")
	diags := resp.State.Set(ctx, &edgeApplicationsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func GetOrigins(EdgeOrigins []edgeapplications.ApplicationOrigins) []ApplicationOrigins {
	var origins []ApplicationOrigins
	for _, origin := range EdgeOrigins {
		origins = append(origins, ApplicationOrigins{
			Name:       types.StringValue(*origin.Name),
			OriginType: types.StringValue(*origin.OriginType),
			OriginID:   types.StringValue(*origin.OriginId),
		})
	}

	return origins
}
