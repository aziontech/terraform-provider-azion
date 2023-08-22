package provider

import (
	"context"
	"io"
	"strconv"
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
	EdgeFirewall            types.Bool      `tfsdk:"edge_firewall"`
	EdgeFunctions           types.Bool      `tfsdk:"edge_functions"`
	ImageOptimization       types.Bool      `tfsdk:"image_optimization"`
	LoadBalancer            types.Bool      `tfsdk:"load_balancer"`
	L2Caching               types.Bool      `tfsdk:"l2_caching"`
	RawLogs                 types.Bool      `tfsdk:"raw_logs"`
	WebApplicationFirewall  types.Bool      `tfsdk:"web_application_firewall"`
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
					"edge_firewall": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether the Edge Application has an edge firewall enabled.",
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
					"web_application_firewall": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Indicates whether a web application firewall is enabled for the Edge Application.",
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

func (r *edgeApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if plan.EdgeApplication.L2Caching.ValueBool() {
		resp.Diagnostics.AddError(
			"L2Caching error ",
			"When you create a Edge Application: L2Caching must be false or remove from request",
		)
	}

	if plan.EdgeApplication.LoadBalancer.ValueBool() {
		resp.Diagnostics.AddError(
			"LoadBalancer error ",
			"When you create a Edge Application: LoadBalancer must be false or remove from request",
		)
	}

	if plan.EdgeApplication.ApplicationAcceleration.ValueBool() {
		resp.Diagnostics.AddError(
			"ApplicationAcceleration error",
			"When you create a Edge Application: ApplicationAcceleration must be false or remove from request",
		)
	}

	if plan.EdgeApplication.DeviceDetection.ValueBool() {
		resp.Diagnostics.AddError(
			"DeviceDetection error ",
			"When you create a Edge Application: DeviceDetection must be false or remove from request",
		)
	}

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
		Name:             plan.EdgeApplication.Name.ValueString(),
		HttpPort:         sliceHTTPPort,
		HttpsPort:        sliceHTTPSPort,
		DebugRules:       edgeapplications.PtrBool(plan.EdgeApplication.DebugRules.ValueBool()),
		Http3:            edgeapplications.PtrBool(plan.EdgeApplication.HTTP3.ValueBool()),
		SupportedCiphers: edgeapplications.PtrString(plan.EdgeApplication.SupportedCiphers.ValueString()),
		DeliveryProtocol: edgeapplications.PtrString(plan.EdgeApplication.DeliveryProtocol.ValueString()),
	}

	createEdgeApplication, response, err := r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsApi.EdgeApplicationsPost(ctx).CreateApplicationRequest(edgeApplication).Execute()
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

	plan.EdgeApplication = &EdgeApplicationResults{
		ApplicationID:           types.Int64Value(createEdgeApplication.Results.GetId()),
		Name:                    types.StringValue(createEdgeApplication.Results.GetName()),
		DeliveryProtocol:        types.StringValue(createEdgeApplication.Results.GetDeliveryProtocol()),
		HTTPPort:                utils.ConvertInterfaceToFloat64List(createEdgeApplication.Results.HttpPort),
		HTTPSPort:               utils.ConvertInterfaceToFloat64List(createEdgeApplication.Results.HttpsPort),
		MinimumTLSVersion:       types.StringValue(createEdgeApplication.Results.GetMinimumTlsVersion()),
		Active:                  types.BoolValue(createEdgeApplication.Results.GetActive()),
		DebugRules:              types.BoolValue(createEdgeApplication.Results.GetDebugRules()),
		HTTP3:                   types.BoolValue(createEdgeApplication.Results.GetHttp3()),
		SupportedCiphers:        types.StringValue(createEdgeApplication.Results.GetSupportedCiphers()),
		ApplicationAcceleration: types.BoolValue(createEdgeApplication.Results.GetApplicationAcceleration()),
		Caching:                 types.BoolValue(createEdgeApplication.Results.GetCaching()),
		DeviceDetection:         types.BoolValue(createEdgeApplication.Results.GetDeviceDetection()),
		EdgeFirewall:            types.BoolValue(createEdgeApplication.Results.GetEdgeFirewall()),
		EdgeFunctions:           types.BoolValue(createEdgeApplication.Results.GetEdgeFunctions()),
		ImageOptimization:       types.BoolValue(createEdgeApplication.Results.GetImageOptimization()),
		LoadBalancer:            types.BoolValue(createEdgeApplication.Results.GetLoadBalancer()),
		L2Caching:               types.BoolValue(createEdgeApplication.Results.GetL2Caching()),
		RawLogs:                 types.BoolValue(createEdgeApplication.Results.GetRawLogs()),
		WebApplicationFirewall:  types.BoolValue(createEdgeApplication.Results.GetWebApplicationFirewall()),
	}

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

	stateEdgeApplication, response, err := r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsApi.EdgeApplicationsIdGet(ctx, state.ID.ValueString()).Execute()
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
		EdgeFirewall:            types.BoolValue(stateEdgeApplication.Results.GetEdgeFirewall()),
		EdgeFunctions:           types.BoolValue(stateEdgeApplication.Results.GetEdgeFunctions()),
		ImageOptimization:       types.BoolValue(stateEdgeApplication.Results.GetImageOptimization()),
		LoadBalancer:            types.BoolValue(stateEdgeApplication.Results.GetLoadBalancer()),
		RawLogs:                 types.BoolValue(stateEdgeApplication.Results.GetRawLogs()),
		L2Caching:               types.BoolValue(stateEdgeApplication.Results.GetL2Caching()),
		WebApplicationFirewall:  types.BoolValue(stateEdgeApplication.Results.GetWebApplicationFirewall()),
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
		Caching:                 edgeapplications.PtrBool(plan.EdgeApplication.Caching.ValueBool()),
		DeviceDetection:         edgeapplications.PtrBool(plan.EdgeApplication.DeviceDetection.ValueBool()),
		EdgeFirewall:            edgeapplications.PtrBool(plan.EdgeApplication.EdgeFirewall.ValueBool()),
		EdgeFunctions:           edgeapplications.PtrBool(plan.EdgeApplication.EdgeFunctions.ValueBool()),
		ImageOptimization:       edgeapplications.PtrBool(plan.EdgeApplication.ImageOptimization.ValueBool()),
		LoadBalancer:            edgeapplications.PtrBool(plan.EdgeApplication.LoadBalancer.ValueBool()),
		RawLogs:                 edgeapplications.PtrBool(plan.EdgeApplication.RawLogs.ValueBool()),
		WebApplicationFirewall:  edgeapplications.PtrBool(plan.EdgeApplication.WebApplicationFirewall.ValueBool()),
		DeliveryProtocol:        edgeapplications.PtrString(plan.EdgeApplication.DeliveryProtocol.ValueString()),
		L2Caching:               edgeapplications.PtrBool(plan.EdgeApplication.L2Caching.ValueBool()),
	}

	updateEdgeApplication, response, err := r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsApi.EdgeApplicationsIdPut(ctx, plan.ID.ValueString()).ApplicationPutRequest(edgeApplication).Execute()
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
		EdgeFirewall:            types.BoolValue(updateEdgeApplication.Results.GetEdgeFirewall()),
		EdgeFunctions:           types.BoolValue(updateEdgeApplication.Results.GetEdgeFunctions()),
		ImageOptimization:       types.BoolValue(updateEdgeApplication.Results.GetImageOptimization()),
		LoadBalancer:            types.BoolValue(updateEdgeApplication.Results.GetLoadBalancer()),
		RawLogs:                 types.BoolValue(updateEdgeApplication.Results.GetRawLogs()),
		L2Caching:               types.BoolValue(updateEdgeApplication.Results.GetL2Caching()),
		WebApplicationFirewall:  types.BoolValue(updateEdgeApplication.Results.GetWebApplicationFirewall()),
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

	response, err := r.client.edgeApplicationsApi.EdgeApplicationsMainSettingsApi.EdgeApplicationsIdDelete(ctx, state.ID.ValueString()).Execute()
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
}

func (r *edgeApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
