package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
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
	TotalCount types.Int64       `tfsdk:"total_count"`
	Page       types.Int64       `tfsdk:"page"`
	PageSize   types.Int64       `tfsdk:"page_size"`
	Results    []ApplicationData `tfsdk:"results"`
	ID         types.String      `tfsdk:"id"`
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
			"total_count": schema.Int64Attribute{
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
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The Application identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the Application.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor identifier.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp.",
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "Product version.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the Application is active.",
							Computed:    true,
						},
						"debug": schema.BoolAttribute{
							Description: "Whether the Application is in debug mode.",
							Computed:    true,
						},
						"modules": schema.SingleNestedAttribute{
							Description: "Modules configuration.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"edge_cache": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"enabled": schema.BoolAttribute{
											Computed: true,
										},
									},
								},
								"functions": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"enabled": schema.BoolAttribute{
											Computed: true,
										},
									},
								},
								"application_accelerator": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"enabled": schema.BoolAttribute{
											Computed: true,
										},
									},
								},
								"image_processor": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"enabled": schema.BoolAttribute{
											Computed: true,
										},
									},
								},
							},
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

	appResponse, response, err := e.client.api.ApplicationsAPI.ListApplications(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			appResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedApplicationList, *http.Response, error) {
				return e.client.api.ApplicationsAPI.ListApplications(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
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

	appState := EdgeApplicationsDataSourceModel{
		Page:       Page,
		PageSize:   PageSize,
		TotalCount: types.Int64Value(*appResponse.Count),
	}

	for _, resultApplication := range appResponse.GetResults() {
		mods := resultApplication.GetModules()
		cache := mods.GetCache()
		functions := mods.GetFunctions()
		applicationAccelerator := mods.GetApplicationAccelerator()
		imageProcessor := mods.GetImageProcessor()

		modules := &ApplicationModules{
			Cache: &CacheModule{
				Enabled: types.BoolValue(cache.GetEnabled()),
			},
			Functions: &EdgeFunctionModule{
				Enabled: types.BoolValue(functions.GetEnabled()),
			},
			ApplicationAccelerator: &ApplicationAcceleratorModule{
				Enabled: types.BoolValue(applicationAccelerator.GetEnabled()),
			},
			ImageProcessor: &ImageProcessorModule{
				Enabled: types.BoolValue(imageProcessor.GetEnabled()),
			},
		}
		appState.Results = append(appState.Results, ApplicationData{
			Id:             types.Int64Value(resultApplication.GetId()),
			Name:           types.StringValue(resultApplication.GetName()),
			LastEditor:     types.StringValue(resultApplication.GetLastEditor()),
			LastModified:   types.StringValue(resultApplication.GetLastModified().Format(time.RFC3339)),
			Modules:        modules,
			ProductVersion: types.StringValue(resultApplication.GetProductVersion()),
			Active:         types.BoolValue(resultApplication.GetActive()),
			Debug:          types.BoolValue(resultApplication.GetDebug()),
		})
	}
	appState.ID = types.StringValue("Get All Edge Application")
	diags := resp.State.Set(ctx, &appState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
