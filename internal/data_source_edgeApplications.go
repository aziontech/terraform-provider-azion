package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	SchemaVersion types.Int64                      `tfsdk:"schema_version"`
	Counter       types.Int64                      `tfsdk:"counter"`
	TotalPages    types.Int64                      `tfsdk:"total_pages"`
	Links         *GetEdgeAplicationsResponseLinks `tfsdk:"links"`
	Results       []EdgeApplicationsResult         `tfsdk:"results"`
	ID            types.String                     `tfsdk:"id"`
}
type GetEdgeAplicationsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeApplicationsResult struct {
	ID           types.Int64          `tfsdk:"id"`
	Name         types.String         `tfsdk:"name"`
	Active       types.Bool           `tfsdk:"active"`
	DebugRules   types.Bool           `tfsdk:"debug_rules"`
	LastEditor   types.String         `tfsdk:"last_editor"`
	LastModified types.String         `tfsdk:"last_modified"`
	Origins      []ApplicationOrigins `tfsdk:"origins"`
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
	resp.TypeName = req.ProviderTypeName + "_edge_applications"
}

func (e *EdgeApplicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of edge applications.",
				Computed:    true,
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
							Description: "The edge application identifier.",
							Required:    true,
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
	edgeAppResponse, response, err := e.client.edgeAplicationsApi.EdgeApplicationsMainSettingsApi.EdgeApplicationsGet(ctx).Execute()
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

	var previous, next string
	if edgeAppResponse.Links.Previous.Get() != nil {
		previous = *edgeAppResponse.Links.Previous.Get()
	}
	if edgeAppResponse.Links.Next.Get() != nil {
		next = *edgeAppResponse.Links.Next.Get()
	}

	edgeApplicationsState := EdgeApplicationsDataSourceModel{
		SchemaVersion: types.Int64Value(edgeAppResponse.SchemaVersion),
		TotalPages:    types.Int64Value(edgeAppResponse.TotalPages),
		Counter:       types.Int64Value(edgeAppResponse.Count),
		Links: &GetEdgeAplicationsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}
	var origins []ApplicationOrigins
	for _, resultEdgeApplication := range edgeAppResponse.GetResults() {
		for _, origin := range resultEdgeApplication.GetOrigins() {
			origins = append(origins, ApplicationOrigins{
				Name:       types.StringValue(*origin.Name),
				OriginType: types.StringValue(*origin.OriginType),
				OriginID:   types.StringValue(*origin.OriginId),
			})
		}
		edgeApplicationsState.Results = append(edgeApplicationsState.Results, EdgeApplicationsResult{
			ID:           types.Int64Value(*resultEdgeApplication.Id),
			Name:         types.StringValue(*resultEdgeApplication.Name),
			Active:       types.BoolValue(*resultEdgeApplication.Active),
			DebugRules:   types.BoolValue(*resultEdgeApplication.DebugRules),
			LastEditor:   types.StringValue(*resultEdgeApplication.LastEditor),
			LastModified: types.StringValue(*resultEdgeApplication.LastModified),
			Origins:      origins,
		})

	}

	edgeApplicationsState.ID = types.StringValue("Get All Edge Application")
	diags := resp.State.Set(ctx, &edgeApplicationsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
