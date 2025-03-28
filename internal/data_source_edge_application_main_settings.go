package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
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
	SchemaVersion types.Int64            `tfsdk:"schema_version"`
	Results       *EdgeApplicationResult `tfsdk:"results"`
	ID            types.String           `tfsdk:"id"`
}

type EdgeApplicationResult struct {
	ApplicationID           types.Int64     `tfsdk:"application_id"`
	Name                    types.String    `tfsdk:"name"`
	DeliveryProtocol        types.String    `tfsdk:"delivery_protocol"`
	HTTPPort                []types.Float64 `tfsdk:"http_port"`
	HTTPSPort               []types.Float64 `tfsdk:"https_port"`
	MinimumTLSVersion       types.String    `tfsdk:"minimum_tls_version"`
	Active                  types.Bool      `tfsdk:"active"`
	DebugRules              types.Bool      `tfsdk:"debug_rules"`
	HTTP3                   types.Bool      `tfsdk:"http3"`
	SupportedCiphers        types.String    `tfsdk:"supported_ciphers"`
	ApplicationAcceleration types.Bool      `tfsdk:"application_acceleration"`
	Caching                 types.Bool      `tfsdk:"caching"`
	DeviceDetection         types.Bool      `tfsdk:"device_detection"`
	EdgeFunctions           types.Bool      `tfsdk:"edge_functions"`
	ImageOptimization       types.Bool      `tfsdk:"image_optimization"`
	LoadBalancer            types.Bool      `tfsdk:"load_balancer"`
	L2Caching               types.Bool      `tfsdk:"l2_caching"`
	RawLogs                 types.Bool      `tfsdk:"raw_logs"`
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
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"application_id": schema.Int64Attribute{
						Description: "The Edge Application identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the Edge Application.",
						Computed:    true,
					},
					"delivery_protocol": schema.StringAttribute{
						Description: "The delivery protocol of the Edge Application.",
						Computed:    true,
					},
					"http_port": schema.ListAttribute{
						Computed:    true,
						ElementType: types.Float64Type,
						Description: "The HTTP port(s) for the Edge Application.",
					},
					"https_port": schema.ListAttribute{
						Computed:    true,
						ElementType: types.Float64Type,
						Description: "The HTTPS port(s) for the Edge Application.",
					},
					"minimum_tls_version": schema.StringAttribute{
						Computed:    true,
						Description: "The minimum TLS version supported by the Edge Application.",
					},
					"active": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether the Edge Application is active.",
					},
					"debug_rules": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether debug rules are enabled for the Edge Application.",
					},
					"http3": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether HTTP/3 is enabled for the Edge Application.",
					},
					"supported_ciphers": schema.StringAttribute{
						Computed:    true,
						Description: "The supported ciphers for the Edge Application.",
					},
					"application_acceleration": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether application acceleration is enabled for the Edge Application.",
					},
					"caching": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether caching is enabled for the Edge Application.",
					},
					"device_detection": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether device detection is enabled for the Edge Application.",
					},
					"edge_functions": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether edge functions are enabled for the Edge Application.",
					},
					"image_optimization": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether image optimization is enabled for the Edge Application.",
					},
					"load_balancer": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether load balancing is enabled for the Edge Application.",
					},
					"l2_caching": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether l2 caching is enabled for the Edge Application.",
					},
					"raw_logs": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether raw logs are enabled for the Edge Application.",
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
			"Edge Application ID error ",
			"is not null",
		)
		return
	}

	edgeApplicationsResponse, response, err := e.client.edgeApplicationsApi.EdgeApplicationsMainSettingsAPI.EdgeApplicationsIdGet(ctx, getEdgeApplicationId.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeApplicationsResponse, response, err = utils.RetryOn429(func() (*edgeapplications.GetApplicationResponse, *http.Response, error) {
				return e.client.edgeApplicationsApi.EdgeApplicationsMainSettingsAPI.EdgeApplicationsIdGet(ctx, getEdgeApplicationId.ValueString()).Execute() //nolint
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

	EdgeApplicationState := EdgeApplicationDataSourceModel{
		SchemaVersion: types.Int64Value(edgeApplicationsResponse.SchemaVersion),
		Results: &EdgeApplicationResult{
			ApplicationID:           types.Int64Value(edgeApplicationsResponse.Results.GetId()),
			Name:                    types.StringValue(edgeApplicationsResponse.Results.GetName()),
			DeliveryProtocol:        types.StringValue(edgeApplicationsResponse.Results.GetDeliveryProtocol()),
			HTTPPort:                utils.ConvertInterfaceToFloat64List(edgeApplicationsResponse.Results.HttpPort),
			HTTPSPort:               utils.ConvertInterfaceToFloat64List(edgeApplicationsResponse.Results.HttpsPort),
			MinimumTLSVersion:       types.StringValue(edgeApplicationsResponse.Results.GetMinimumTlsVersion()),
			Active:                  types.BoolValue(edgeApplicationsResponse.Results.GetActive()),
			DebugRules:              types.BoolValue(edgeApplicationsResponse.Results.GetDebugRules()),
			HTTP3:                   types.BoolValue(edgeApplicationsResponse.Results.GetHttp3()),
			SupportedCiphers:        types.StringValue(edgeApplicationsResponse.Results.GetSupportedCiphers()),
			ApplicationAcceleration: types.BoolValue(edgeApplicationsResponse.Results.GetApplicationAcceleration()),
			Caching:                 types.BoolValue(edgeApplicationsResponse.Results.GetCaching()),
			DeviceDetection:         types.BoolValue(edgeApplicationsResponse.Results.GetDeviceDetection()),
			EdgeFunctions:           types.BoolValue(edgeApplicationsResponse.Results.GetEdgeFunctions()),
			ImageOptimization:       types.BoolValue(edgeApplicationsResponse.Results.GetImageOptimization()),
			LoadBalancer:            types.BoolValue(edgeApplicationsResponse.Results.GetLoadBalancer()),
			L2Caching:               types.BoolValue(edgeApplicationsResponse.Results.GetL2Caching()),
			RawLogs:                 types.BoolValue(edgeApplicationsResponse.Results.GetRawLogs()),
		},
	}

	EdgeApplicationState.ID = types.StringValue("Get By ID Edge Application")
	diags = resp.State.Set(ctx, &EdgeApplicationState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
