package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &edgeApplicationResource{}
	_ resource.ResourceWithConfigure   = &edgeApplicationResource{}
	_ resource.ResourceWithImportState = &edgeApplicationResource{}
)

func NewEdgeApplicationMainSettingsResource() resource.Resource {
	return &edgeApplicationResource{}
}

type edgeApplicationResource struct {
	client *apiClient
}

type EdgeApplicationResourceModel struct {
	SchemaVersion   types.Int64             `tfsdk:"schema_version"`
	EdgeApplication *EdgeApplicationResults `tfsdk:"edge_application"`
	ID              types.String            `tfsdk:"id"`
	LastUpdated     types.String            `tfsdk:"last_updated"`
}

type EdgeApplicationResults struct {
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

func (r *edgeApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_main_setting"
}

func (r *edgeApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"edge_application": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"application_id": schema.Int64Attribute{
						Description: "The Edge Application identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the Edge Application.",
						Required:    true,
					},
					"delivery_protocol": schema.StringAttribute{
						Description: "The delivery protocol of the Edge Application.",
						Required:    true,
					},
					"http_port": schema.ListAttribute{
						Required:    true,
						ElementType: types.Float64Type,
						Description: "The HTTP port(s) for the Edge Application.",
					},
					"https_port": schema.ListAttribute{
						Required:    true,
						ElementType: types.Float64Type,
						Description: "The HTTPS port(s) for the Edge Application.",
					},
					"minimum_tls_version": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The minimum TLS version supported by the Edge Application.",
					},
					"active": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates whether the Edge Application is active.",
					},
					"debug_rules": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether debug rules are enabled for the Edge Application.",
					},
					"http3": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether HTTP/3 is enabled for the Edge Application.",
					},
					"supported_ciphers": schema.StringAttribute{
						Required:    true,
						Description: "The supported ciphers for the Edge Application.",
					},
					"application_acceleration": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether application acceleration is enabled for the Edge Application.",
					},
					"caching": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether caching is enabled for the Edge Application.",
					},
					"device_detection": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether device detection is enabled for the Edge Application.",
					},
					"edge_functions": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether edge functions are enabled for the Edge Application.",
					},
					"image_optimization": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether image optimization is enabled for the Edge Application.",
					},
					"load_balancer": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether load balancing is enabled for the Edge Application.",
					},
					"l2_caching": schema.BoolAttribute{
						Computed:    true,
						Optional:    true,
						Description: "Indicates whether l2 caching is enabled for the Edge Application.",
					},
					"raw_logs": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether raw logs are enabled for the Edge Application.",
					},
				},
			},
		},
	}
}

func (r *edgeApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

var mutex sync.Mutex

func (r *edgeApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	mutex.Lock()
	defer mutex.Unlock()

	if resp.Diagnostics.HasError() {
		return
	}

	sliceHTTPPort, err := utils.ConvertFloat64ToInterface(plan.EdgeApplication.HTTPPort)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	sliceHTTPSPort, err := utils.ConvertFloat64ToInterface(plan.EdgeApplication.HTTPSPort)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	edgeApplication := edgeapplications.CreateApplicationRequest{
		Name:              plan.EdgeApplication.Name.ValueString(),
		HttpPort:          sliceHTTPPort,
		HttpsPort:         sliceHTTPSPort,
		MinimumTlsVersion: edgeapplications.PtrString(plan.EdgeApplication.MinimumTLSVersion.ValueString()),
		DebugRules:        edgeapplications.PtrBool(plan.EdgeApplication.DebugRules.ValueBool()),
		Http3:             edgeapplications.PtrBool(plan.EdgeApplication.HTTP3.ValueBool()),
		SupportedCiphers:  edgeapplications.PtrString(plan.EdgeApplication.SupportedCiphers.ValueString()),
		DeliveryProtocol:  edgeapplications.PtrString(plan.EdgeApplication.DeliveryProtocol.ValueString()),
	}

	createEdgeApplication, response, err := r.client.edgeApplicationsApi.
		EdgeApplicationsMainSettingsAPI.EdgeApplicationsPost(ctx).
		CreateApplicationRequest(edgeApplication).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createEdgeApplication, response, err = utils.RetryOn429(func() (*edgeapplications.CreateApplicationResult, *http.Response, error) {
				return r.client.edgeApplicationsApi.
					EdgeApplicationsMainSettingsAPI.EdgeApplicationsPost(ctx).
					CreateApplicationRequest(edgeApplication).Execute() //nolint
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

	requestUpdate := edgeapplications.ApplicationUpdateRequest{}
	if plan.EdgeApplication.L2Caching.ValueBool() {
		requestUpdate.L2Caching = plan.EdgeApplication.L2Caching.ValueBoolPointer()
	}

	if plan.EdgeApplication.EdgeFunctions.ValueBool() {
		requestUpdate.EdgeFunctions = plan.EdgeApplication.EdgeFunctions.ValueBoolPointer()
	}

	if plan.EdgeApplication.LoadBalancer.ValueBool() {
		requestUpdate.LoadBalancer = plan.EdgeApplication.LoadBalancer.ValueBoolPointer()
	}

	if plan.EdgeApplication.ApplicationAcceleration.ValueBool() {
		requestUpdate.ApplicationAcceleration = plan.EdgeApplication.ApplicationAcceleration.ValueBoolPointer()
	}

	if plan.EdgeApplication.DeviceDetection.ValueBool() {
		requestUpdate.DeviceDetection = plan.EdgeApplication.DeviceDetection.ValueBoolPointer()
	}

	if plan.EdgeApplication.ImageOptimization.ValueBool() {
		requestUpdate.ImageOptimization = plan.EdgeApplication.ImageOptimization.ValueBoolPointer()
	}

	if plan.EdgeApplication.RawLogs.ValueBool() {
		requestUpdate.RawLogs = plan.EdgeApplication.RawLogs.ValueBoolPointer()
	}

	ID := strconv.Itoa(int(createEdgeApplication.Results.GetId()))

	updateEdgeApplication, response, err := r.client.edgeApplicationsApi.
		EdgeApplicationsMainSettingsAPI.
		EdgeApplicationsIdPatch(ctx, ID).
		ApplicationUpdateRequest(requestUpdate).Execute() //nolint
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

	edgeAppResults := &EdgeApplicationResults{
		ApplicationID:     types.Int64Value(createEdgeApplication.Results.GetId()),
		Name:              types.StringValue(createEdgeApplication.Results.GetName()),
		DeliveryProtocol:  types.StringValue(createEdgeApplication.Results.GetDeliveryProtocol()),
		HTTPPort:          utils.ConvertInterfaceToFloat64List(createEdgeApplication.Results.HttpPort),
		HTTPSPort:         utils.ConvertInterfaceToFloat64List(createEdgeApplication.Results.HttpsPort),
		MinimumTLSVersion: types.StringValue(createEdgeApplication.Results.GetMinimumTlsVersion()),
		Active:            types.BoolValue(createEdgeApplication.Results.GetActive()),
		DebugRules:        types.BoolValue(createEdgeApplication.Results.GetDebugRules()),
		HTTP3:             types.BoolValue(createEdgeApplication.Results.GetHttp3()),
		SupportedCiphers:  types.StringValue(createEdgeApplication.Results.GetSupportedCiphers()),
		Caching:           types.BoolValue(createEdgeApplication.Results.GetCaching()),
	}

	if requestUpdate.L2Caching == nil {
		edgeAppResults.L2Caching = types.BoolValue(createEdgeApplication.Results.GetL2Caching())
	} else {
		edgeAppResults.L2Caching = types.BoolValue(updateEdgeApplication.Results.GetL2Caching())
	}

	if requestUpdate.EdgeFunctions == nil {
		edgeAppResults.EdgeFunctions = types.BoolValue(createEdgeApplication.Results.GetEdgeFunctions())
	} else {
		edgeAppResults.EdgeFunctions = types.BoolValue(updateEdgeApplication.Results.GetEdgeFunctions())
	}

	if requestUpdate.LoadBalancer == nil {
		edgeAppResults.LoadBalancer = types.BoolValue(createEdgeApplication.Results.GetLoadBalancer())
	} else {
		edgeAppResults.LoadBalancer = types.BoolValue(updateEdgeApplication.Results.GetLoadBalancer())
	}

	if requestUpdate.ApplicationAcceleration == nil {
		edgeAppResults.ApplicationAcceleration = types.BoolValue(createEdgeApplication.Results.GetApplicationAcceleration())
	} else {
		edgeAppResults.ApplicationAcceleration = types.BoolValue(updateEdgeApplication.Results.GetApplicationAcceleration())
	}

	if requestUpdate.DeviceDetection == nil {
		edgeAppResults.DeviceDetection = types.BoolValue(createEdgeApplication.Results.GetDeviceDetection())
	} else {
		edgeAppResults.DeviceDetection = types.BoolValue(updateEdgeApplication.Results.GetDeviceDetection())
	}

	if requestUpdate.ImageOptimization == nil {
		edgeAppResults.ImageOptimization = types.BoolValue(createEdgeApplication.Results.GetImageOptimization())
	} else {
		edgeAppResults.ImageOptimization = types.BoolValue(updateEdgeApplication.Results.GetImageOptimization())
	}

	if requestUpdate.RawLogs == nil {
		edgeAppResults.RawLogs = types.BoolValue(createEdgeApplication.Results.GetRawLogs())
	} else {
		edgeAppResults.RawLogs = types.BoolValue(updateEdgeApplication.Results.GetRawLogs())
	}

	plan.EdgeApplication = edgeAppResults
	plan.SchemaVersion = types.Int64Value(createEdgeApplication.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(createEdgeApplication.Results.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateEdgeApplication, response, err := r.client.edgeApplicationsApi.
		EdgeApplicationsMainSettingsAPI.
		EdgeApplicationsIdGet(ctx, state.ID.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			stateEdgeApplication, response, err = utils.RetryOn429(func() (*edgeapplications.GetApplicationResponse, *http.Response, error) {
				return r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsAPI.EdgeApplicationsIdGet(ctx, state.ID.ValueString()).Execute() //nolint
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

	sliceHTTPPort := utils.ConvertInterfaceToFloat64List(stateEdgeApplication.Results.HttpPort)

	sliceHTTPSPort := utils.ConvertInterfaceToFloat64List(stateEdgeApplication.Results.HttpsPort)

	state.EdgeApplication = &EdgeApplicationResults{
		ApplicationID:           types.Int64Value(stateEdgeApplication.Results.GetId()),
		Name:                    types.StringValue(stateEdgeApplication.Results.GetName()),
		DeliveryProtocol:        types.StringValue(stateEdgeApplication.Results.GetDeliveryProtocol()),
		HTTPPort:                sliceHTTPPort,
		HTTPSPort:               sliceHTTPSPort,
		MinimumTLSVersion:       types.StringValue(stateEdgeApplication.Results.GetMinimumTlsVersion()),
		Active:                  types.BoolValue(stateEdgeApplication.Results.GetActive()),
		DebugRules:              types.BoolValue(stateEdgeApplication.Results.GetDebugRules()),
		HTTP3:                   types.BoolValue(stateEdgeApplication.Results.GetHttp3()),
		SupportedCiphers:        types.StringValue(stateEdgeApplication.Results.GetSupportedCiphers()),
		ApplicationAcceleration: types.BoolValue(stateEdgeApplication.Results.GetApplicationAcceleration()),
		Caching:                 types.BoolValue(stateEdgeApplication.Results.GetCaching()),
		DeviceDetection:         types.BoolValue(stateEdgeApplication.Results.GetDeviceDetection()),
		EdgeFunctions:           types.BoolValue(stateEdgeApplication.Results.GetEdgeFunctions()),
		ImageOptimization:       types.BoolValue(stateEdgeApplication.Results.GetImageOptimization()),
		LoadBalancer:            types.BoolValue(stateEdgeApplication.Results.GetLoadBalancer()),
		RawLogs:                 types.BoolValue(stateEdgeApplication.Results.GetRawLogs()),
		L2Caching:               types.BoolValue(stateEdgeApplication.Results.GetL2Caching()),
	}
	state.ID = types.StringValue(strconv.FormatInt(stateEdgeApplication.Results.GetId(), 10))
	state.SchemaVersion = types.Int64Value(stateEdgeApplication.SchemaVersion)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sliceHTTPPort, err := utils.ConvertFloat64ToInterface(plan.EdgeApplication.HTTPPort)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	sliceHTTPSPort, err := utils.ConvertFloat64ToInterface(plan.EdgeApplication.HTTPSPort)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	edgeApplication := edgeapplications.ApplicationPutRequest{
		Name:                    plan.EdgeApplication.Name.ValueString(),
		HttpPort:                sliceHTTPPort,
		HttpsPort:               sliceHTTPSPort,
		MinimumTlsVersion:       edgeapplications.PtrString(plan.EdgeApplication.MinimumTLSVersion.ValueString()),
		DebugRules:              edgeapplications.PtrBool(plan.EdgeApplication.DebugRules.ValueBool()),
		Http3:                   edgeapplications.PtrBool(plan.EdgeApplication.HTTP3.ValueBool()),
		SupportedCiphers:        edgeapplications.PtrString(plan.EdgeApplication.SupportedCiphers.ValueString()),
		ApplicationAcceleration: edgeapplications.PtrBool(plan.EdgeApplication.ApplicationAcceleration.ValueBool()),
		DeviceDetection:         edgeapplications.PtrBool(plan.EdgeApplication.DeviceDetection.ValueBool()),
		EdgeFunctions:           edgeapplications.PtrBool(plan.EdgeApplication.EdgeFunctions.ValueBool()),
		ImageOptimization:       edgeapplications.PtrBool(plan.EdgeApplication.ImageOptimization.ValueBool()),
		LoadBalancer:            edgeapplications.PtrBool(plan.EdgeApplication.LoadBalancer.ValueBool()),
		RawLogs:                 edgeapplications.PtrBool(plan.EdgeApplication.RawLogs.ValueBool()),
		DeliveryProtocol:        edgeapplications.PtrString(plan.EdgeApplication.DeliveryProtocol.ValueString()),
		L2Caching:               edgeapplications.PtrBool(plan.EdgeApplication.L2Caching.ValueBool()),
	}

	updateEdgeApplication, response, err := r.client.edgeApplicationsApi.
		EdgeApplicationsMainSettingsAPI.
		EdgeApplicationsIdPut(ctx, plan.ID.ValueString()).
		ApplicationPutRequest(edgeApplication).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateEdgeApplication, response, err = utils.RetryOn429(func() (*edgeapplications.ApplicationPutResult, *http.Response, error) {
				return r.client.edgeApplicationsApi.
					EdgeApplicationsMainSettingsAPI.
					EdgeApplicationsIdPut(ctx, plan.ID.ValueString()).
					ApplicationPutRequest(edgeApplication).Execute() //nolint
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

	sliceHTTPPortResult := utils.ConvertInterfaceToFloat64List(updateEdgeApplication.Results.HttpPort)

	sliceHTTPSPortResult := utils.ConvertInterfaceToFloat64List(updateEdgeApplication.Results.HttpsPort)

	plan.EdgeApplication = &EdgeApplicationResults{
		ApplicationID:           types.Int64Value(updateEdgeApplication.Results.GetId()),
		Name:                    types.StringValue(updateEdgeApplication.Results.GetName()),
		DeliveryProtocol:        types.StringValue(updateEdgeApplication.Results.GetDeliveryProtocol()),
		HTTPPort:                sliceHTTPPortResult,
		HTTPSPort:               sliceHTTPSPortResult,
		MinimumTLSVersion:       types.StringValue(updateEdgeApplication.Results.GetMinimumTlsVersion()),
		Active:                  types.BoolValue(updateEdgeApplication.Results.GetActive()),
		DebugRules:              types.BoolValue(updateEdgeApplication.Results.GetDebugRules()),
		HTTP3:                   types.BoolValue(updateEdgeApplication.Results.GetHttp3()),
		SupportedCiphers:        types.StringValue(updateEdgeApplication.Results.GetSupportedCiphers()),
		ApplicationAcceleration: types.BoolValue(updateEdgeApplication.Results.GetApplicationAcceleration()),
		Caching:                 types.BoolValue(updateEdgeApplication.Results.GetCaching()),
		DeviceDetection:         types.BoolValue(updateEdgeApplication.Results.GetDeviceDetection()),
		EdgeFunctions:           types.BoolValue(updateEdgeApplication.Results.GetEdgeFunctions()),
		ImageOptimization:       types.BoolValue(updateEdgeApplication.Results.GetImageOptimization()),
		LoadBalancer:            types.BoolValue(updateEdgeApplication.Results.GetLoadBalancer()),
		RawLogs:                 types.BoolValue(updateEdgeApplication.Results.GetRawLogs()),
		L2Caching:               types.BoolValue(updateEdgeApplication.Results.GetL2Caching()),
	}

	plan.SchemaVersion = types.Int64Value(updateEdgeApplication.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(updateEdgeApplication.Results.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsAPI.
		EdgeApplicationsIdDelete(ctx, state.ID.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
				return r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsAPI.EdgeApplicationsIdDelete(ctx, state.ID.ValueString()).Execute() //nolint
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
}

func (r *edgeApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
