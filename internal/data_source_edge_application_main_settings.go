package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &EdgeApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeApplicationDataSource{}
)

func dataSourceAzionEdgeApplication() datasource.DataSource {
	return &EdgeApplicationDataSource{}
}

type EdgeApplicationDataSource struct {
	client *apiClient
}

type EdgeApplicationDataSourceModel struct {
	SchemaVersion types.Int64      `tfsdk:"schema_version"`
	Data          *ApplicationData `tfsdk:"data"`
	ID            types.String     `tfsdk:"id"`
}

// ApplicationData models the API response into the Terraform state, following the requested schema.
type ApplicationData struct {
	Id             types.Int64         `tfsdk:"id"`
	Name           types.String        `tfsdk:"name"`
	LastEditor     types.String        `tfsdk:"last_editor"`
	LastModified   types.String        `tfsdk:"last_modified"` // RFC3339 as string
	Modules        *ApplicationModules `tfsdk:"modules"`
	Active         types.Bool          `tfsdk:"active"`
	Debug          types.Bool          `tfsdk:"debug"`
	ProductVersion types.String        `tfsdk:"product_version"`
}

type ApplicationModules struct {
	Cache                  *CacheModule                  `tfsdk:"edge_cache"`
	Functions              *EdgeFunctionModule           `tfsdk:"functions"`
	ApplicationAccelerator *ApplicationAcceleratorModule `tfsdk:"application_accelerator"`
	ImageProcessor         *ImageProcessorModule         `tfsdk:"image_processor"`
}

type CacheModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type EdgeFunctionModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type ApplicationAcceleratorModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type ImageProcessorModule struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func (e *EdgeApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	e.client = req.ProviderData.(*apiClient)
}

func (e *EdgeApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_main_settings"
}

func (e *EdgeApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
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
						Computed:    true,
						Description: "Whether the Application is active.",
					},
					"debug": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether debug is enabled.",
					},
					"modules": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"edge_cache": schema.SingleNestedAttribute{
								Computed: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Computed: true},
								},
							},
							"functions": schema.SingleNestedAttribute{
								Computed: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Computed: true},
								},
							},
							"application_accelerator": schema.SingleNestedAttribute{
								Computed: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Computed: true},
								},
							},
							"image_processor": schema.SingleNestedAttribute{
								Computed: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (e *EdgeApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getEdgeApplicationId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getEdgeApplicationId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if getEdgeApplicationId.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Application ID error ",
			"empty application ID",
		)
		return
	}

	edgeApplicationId, err := strconv.ParseInt(getEdgeApplicationId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Application ID error ",
			"not a valid application ID (integer)",
		)
		return
	}

	applicationsResponse, response, err := e.client.edgeApi.ApplicationsAPI.RetrieveApplication(ctx, edgeApplicationId).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			applicationsResponse, response, err = utils.RetryOn429(func() (*sdk.ResponseRetrieveApplication, *http.Response, error) {
				return e.client.edgeApi.ApplicationsAPI.RetrieveApplication(ctx, edgeApplicationId).Execute() //nolint
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

	mods := applicationsResponse.Data.GetModules()
	cache := mods.GetEdgeCache()
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

	// Populate only safe fields to avoid SDK getter mismatches; leave others null.
	state := EdgeApplicationDataSourceModel{
		SchemaVersion: types.Int64Null(),
		Data: &ApplicationData{
			Id:             types.Int64Value(applicationsResponse.Data.GetId()),
			Name:           types.StringValue(applicationsResponse.Data.GetName()),
			Active:         types.BoolValue(applicationsResponse.Data.GetActive()),
			Debug:          types.BoolValue(applicationsResponse.Data.GetDebug()),
			Modules:        modules,
			LastEditor:     types.StringValue(applicationsResponse.Data.GetLastEditor()),
			LastModified:   types.StringValue(applicationsResponse.Data.GetLastModified().Format(time.RFC3339)),
			ProductVersion: types.StringValue(applicationsResponse.Data.GetProductVersion()),
		},
	}

	state.ID = types.StringValue("Get Application By ID")
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
