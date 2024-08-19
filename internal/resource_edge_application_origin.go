package provider

import (
	"context"
	"io"
	"strconv"
	"strings"
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
	_ resource.Resource                = &originResource{}
	_ resource.ResourceWithConfigure   = &originResource{}
	_ resource.ResourceWithImportState = &originResource{}
)

func NewEdgeApplicationOriginResource() resource.Resource {
	return &originResource{}
}

type originResource struct {
	client *apiClient
}

type OriginResourceModel struct {
	SchemaVersion types.Int64            `tfsdk:"schema_version"`
	Origin        *OriginResourceResults `tfsdk:"origin"`
	ID            types.String           `tfsdk:"id"`
	ApplicationID types.Int64            `tfsdk:"edge_application_id"`
	LastUpdated   types.String           `tfsdk:"last_updated"`
}

type OriginResourceResults struct {
	OriginID                   types.Int64     `tfsdk:"origin_id"`
	OriginKey                  types.String    `tfsdk:"origin_key"`
	Name                       types.String    `tfsdk:"name"`
	OriginType                 types.String    `tfsdk:"origin_type"`
	Addresses                  []OriginAddress `tfsdk:"addresses"`
	OriginProtocolPolicy       types.String    `tfsdk:"origin_protocol_policy"`
	IsOriginRedirectionEnabled types.Bool      `tfsdk:"is_origin_redirection_enabled"`
	HostHeader                 types.String    `tfsdk:"host_header"`
	Method                     types.String    `tfsdk:"method"`
	OriginPath                 types.String    `tfsdk:"origin_path"`
	ConnectionTimeout          types.Int64     `tfsdk:"connection_timeout"`
	TimeoutBetweenBytes        types.Int64     `tfsdk:"timeout_between_bytes"`
	HMACAuthentication         types.Bool      `tfsdk:"hmac_authentication"`
	HMACRegionName             types.String    `tfsdk:"hmac_region_name"`
	HMACAccessKey              types.String    `tfsdk:"hmac_access_key"`
	HMACSecretKey              types.String    `tfsdk:"hmac_secret_key"`
}

type OriginAddress struct {
	Address    types.String `tfsdk:"address"`
	Weight     types.Int64  `tfsdk:"weight"`
	ServerRole types.String `tfsdk:"server_role"`
	IsActive   types.Bool   `tfsdk:"is_active"`
}

func (r *originResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_origin"
}

func (r *originResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "The edge application identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"origin": schema.SingleNestedAttribute{
				Description: "Origin configuration.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"origin_id": schema.Int64Attribute{
						Description: "Origin identifier.",
						Computed:    true,
					},
					"origin_key": schema.StringAttribute{
						Description: "Origin key.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Origin name.",
						Required:    true,
					},
					"origin_type": schema.StringAttribute{
						Description: "Identifies the source of a record.\n" +
							"~> **Note about Origin Type**\n" +
							"Accepted values: `single_origin`(default), `load_balancer` and `live_ingest`\n\n",
						Optional: true,
						Computed: true,
					},
					"addresses": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"address": schema.StringAttribute{
									Description: "Address of the origin.",
									Required:    true,
								},
								"weight": schema.Int64Attribute{
									Description: "Weight of the origin.",
									Required:    false,
									Optional:    true,
									Computed:    true,
								},
								"server_role": schema.StringAttribute{
									Description: "Server role of the origin.",
									Required:    false,
									Optional:    true,
									Computed:    true,
								},
								"is_active": schema.BoolAttribute{
									Description: "Status of the origin.",
									Required:    false,
									Optional:    true,
									Computed:    true,
								},
							},
						},
					},
					"origin_protocol_policy": schema.StringAttribute{
						Description: "Protocols for connection to the origin.\n" +
							"~> **Note about Origin Protocol Policy**\n" +
							"Accepted values: `preserve`(default), `http` and `https`\n\n",
						Optional: true,
						Computed: true,
					},
					"is_origin_redirection_enabled": schema.BoolAttribute{
						Description: "Whether origin redirection is enabled.",
						Computed:    true,
					},
					"host_header": schema.StringAttribute{
						Description: "Host header value that will be delivered to the origin.\n" +
							"~> **Note about Host Header**\n" +
							"Accepted values: `${host}`(default) and must be specified with `$${host}`\n\n",
						Required: true,
					},
					"method": schema.StringAttribute{
						Description: "HTTP method used by the origin.",
						Computed:    true,
					},
					"origin_path": schema.StringAttribute{
						Description: "Path of the origin.",
						Optional:    true,
						Computed:    true,
					},
					"connection_timeout": schema.Int64Attribute{
						Description: "Connection timeout in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"timeout_between_bytes": schema.Int64Attribute{
						Description: "Timeout between bytes in seconds.",
						Optional:    true,
						Computed:    true,
					},
					"hmac_authentication": schema.BoolAttribute{
						Description: "Whether HMAC authentication is enabled.",
						Optional:    true,
						Computed:    true,
					},
					"hmac_region_name": schema.StringAttribute{
						Description: "HMAC region name.",
						Optional:    true,
						Computed:    true,
					},
					"hmac_access_key": schema.StringAttribute{
						Description: "HMAC access key.",
						Optional:    true,
						Computed:    true,
					},
					"hmac_secret_key": schema.StringAttribute{
						Description: "HMAC secret key.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *originResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *originResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OriginResourceModel
	var edgeApplicationID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	var addressesRequest []edgeapplications.CreateOriginsRequestAddresses
	for _, addr := range plan.Origin.Addresses {
		var serverRole *string = addr.ServerRole.ValueStringPointer()
		if addr.ServerRole.ValueString() == "" {
			serverRole = nil
		}

		var weight *int64 = addr.Weight.ValueInt64Pointer()
		if addr.Weight.ValueInt64() == 0 {
			weight = nil
		}

		requestAddresses := edgeapplications.CreateOriginsRequestAddresses{
			Address:    addr.Address.ValueString(),
			IsActive:   addr.IsActive.ValueBoolPointer(),
			Weight:     weight,
			ServerRole: serverRole,
		}
		addressesRequest = append(addressesRequest, requestAddresses)
	}

	var originProtocolPolicy string
	if plan.Origin.OriginProtocolPolicy.IsUnknown() {
		originProtocolPolicy = "preserve"
	} else {
		if plan.Origin.OriginProtocolPolicy.ValueString() == "" || plan.Origin.OriginProtocolPolicy.IsNull() {
			resp.Diagnostics.AddError("Origin Protocol Policy",
				"Is not null, Possible choices are: [preserve(default), http, https]")
			return
		}
		originProtocolPolicy = plan.Origin.OriginProtocolPolicy.ValueString()
	}

	var OriginType string
	if plan.Origin.OriginType.IsUnknown() {
		OriginType = "single_origin"
	} else {
		if plan.Origin.OriginType.ValueString() == "" || plan.Origin.OriginType.IsNull() {
			resp.Diagnostics.AddError("Origin Type",
				"Is not null, Possible choices are: [single_origin]")
			return
		}
		OriginType = plan.Origin.OriginType.ValueString()
	}

	originRequest := edgeapplications.CreateOriginsRequest{
		Name:                 plan.Origin.Name.ValueString(),
		Addresses:            addressesRequest,
		OriginType:           edgeapplications.PtrString(OriginType),
		OriginProtocolPolicy: edgeapplications.PtrString(originProtocolPolicy),
		HostHeader:           plan.Origin.HostHeader.ValueStringPointer(),
		OriginPath:           edgeapplications.PtrString(plan.Origin.OriginPath.ValueString()),
		HmacAuthentication:   edgeapplications.PtrBool(plan.Origin.HMACAuthentication.ValueBool()),
		HmacRegionName:       edgeapplications.PtrString(plan.Origin.HMACRegionName.ValueString()),
		HmacAccessKey:        edgeapplications.PtrString(plan.Origin.HMACAccessKey.ValueString()),
		HmacSecretKey:        edgeapplications.PtrString(plan.Origin.HMACSecretKey.ValueString()),
	}

	originResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsPost(ctx, edgeApplicationID.ValueInt64()).CreateOriginsRequest(originRequest).Execute() //nolint
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

	var addresses []OriginAddress
	for _, addr := range originResponse.Results.Addresses {
		addresses = append(addresses, OriginAddress{
			Address:    types.StringValue(addr.GetAddress()),
			Weight:     types.Int64Value(addr.GetWeight()),
			ServerRole: types.StringValue(addr.GetServerRole()),
			IsActive:   types.BoolValue(addr.GetIsActive()),
		})
	}

	plan.Origin = &OriginResourceResults{
		OriginID:                   types.Int64Value(originResponse.Results.GetOriginId()),
		OriginKey:                  types.StringValue(originResponse.Results.GetOriginKey()),
		Name:                       types.StringValue(originResponse.Results.GetName()),
		OriginType:                 types.StringValue(originResponse.Results.GetOriginType()),
		Addresses:                  addresses,
		OriginProtocolPolicy:       types.StringValue(originResponse.Results.GetOriginProtocolPolicy()),
		IsOriginRedirectionEnabled: types.BoolValue(originResponse.Results.GetIsOriginRedirectionEnabled()),
		HostHeader:                 types.StringValue(originResponse.Results.GetHostHeader()),
		Method:                     types.StringValue(originResponse.Results.GetMethod()),
		OriginPath:                 types.StringValue(originResponse.Results.GetOriginPath()),
		ConnectionTimeout:          types.Int64Value(originResponse.Results.GetConnectionTimeout()),
		TimeoutBetweenBytes:        types.Int64Value(originResponse.Results.GetTimeoutBetweenBytes()),
		HMACAuthentication:         types.BoolValue(originResponse.Results.GetHmacAuthentication()),
		HMACRegionName:             types.StringValue(originResponse.Results.GetHmacRegionName()),
		HMACAccessKey:              types.StringValue(originResponse.Results.GetHmacAccessKey()),
		HMACSecretKey:              types.StringValue(originResponse.Results.GetHmacSecretKey()),
	}

	plan.SchemaVersion = types.Int64Value(originResponse.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(*originResponse.Results.OriginId, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *originResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OriginResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var ApplicationID int64
	var OriginKey string
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		ApplicationID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		OriginKey = valueFromCmd[1]
	} else {
		ApplicationID = state.ApplicationID.ValueInt64()
		OriginKey = state.Origin.OriginKey.ValueString()
	}

	if OriginKey == "" {
		resp.Diagnostics.AddError(
			"Origin Key error ",
			"is not null",
		)
		return
	}

	originResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(ctx, ApplicationID, OriginKey).Execute() //nolint
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

	var addresses []OriginAddress
	for _, addr := range originResponse.Results.Addresses {
		addresses = append(addresses, OriginAddress{
			Address:    types.StringValue(addr.GetAddress()),
			Weight:     types.Int64Value(addr.GetWeight()),
			ServerRole: types.StringValue(addr.GetServerRole()),
			IsActive:   types.BoolValue(addr.GetIsActive()),
		})
	}

	state.Origin = &OriginResourceResults{
		OriginID:                   types.Int64Value(originResponse.Results.GetOriginId()),
		OriginKey:                  types.StringValue(originResponse.Results.GetOriginKey()),
		Name:                       types.StringValue(originResponse.Results.GetName()),
		OriginType:                 types.StringValue(originResponse.Results.GetOriginType()),
		Addresses:                  addresses,
		OriginProtocolPolicy:       types.StringValue(originResponse.Results.GetOriginProtocolPolicy()),
		IsOriginRedirectionEnabled: types.BoolValue(originResponse.Results.GetIsOriginRedirectionEnabled()),
		HostHeader:                 types.StringValue(originResponse.Results.GetHostHeader()),
		Method:                     types.StringValue(originResponse.Results.GetMethod()),
		OriginPath:                 types.StringValue(originResponse.Results.GetOriginPath()),
		ConnectionTimeout:          types.Int64Value(originResponse.Results.GetConnectionTimeout()),
		TimeoutBetweenBytes:        types.Int64Value(originResponse.Results.GetTimeoutBetweenBytes()),
		HMACAuthentication:         types.BoolValue(originResponse.Results.GetHmacAuthentication()),
		HMACRegionName:             types.StringValue(originResponse.Results.GetHmacRegionName()),
		HMACAccessKey:              types.StringValue(originResponse.Results.GetHmacAccessKey()),
		HMACSecretKey:              types.StringValue(originResponse.Results.GetHmacSecretKey()),
	}
	state.ID = types.StringValue(strconv.FormatInt(*originResponse.Results.OriginId, 10))
	state.ApplicationID = types.Int64Value(ApplicationID)
	state.SchemaVersion = types.Int64Value(originResponse.SchemaVersion)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *originResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OriginResourceModel
	var edgeApplicationID types.Int64
	var originKey types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OriginResourceModel
	diagsOrigin := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsOrigin...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Origin.OriginKey.ValueString() == "" {
		originKey = state.Origin.OriginKey
	} else {
		originKey = plan.Origin.OriginKey
	}

	if plan.ApplicationID.IsNull() {
		edgeApplicationID = state.ApplicationID
	} else {
		edgeApplicationID = plan.ApplicationID
	}

	var addressesRequest []edgeapplications.CreateOriginsRequestAddresses
	for _, addr := range plan.Origin.Addresses {	
		var serverRole *string = addr.ServerRole.ValueStringPointer()
		if addr.ServerRole.ValueString() == "" {
			serverRole = nil
		}

		var weight *int64 = addr.Weight.ValueInt64Pointer()
		if addr.Weight.ValueInt64() == 0 {
			weight = nil
		}

		addressesRequest = append(addressesRequest, edgeapplications.CreateOriginsRequestAddresses{
			Address: addr.Address.ValueString(),
			IsActive: addr.IsActive.ValueBoolPointer(),
			Weight: weight,
			ServerRole: serverRole,
		})
	}

	var originProtocolPolicy string
	if plan.Origin.OriginProtocolPolicy.IsUnknown() {
		originProtocolPolicy = "preserve"
	} else {
		if plan.Origin.OriginProtocolPolicy.ValueString() == "" || plan.Origin.OriginProtocolPolicy.IsNull() {
			resp.Diagnostics.AddError("Origin Protocol Policy",
				"Is not null, Possible choices are: [preserve(default), http, https]")
			return
		}
		originProtocolPolicy = plan.Origin.OriginProtocolPolicy.ValueString()
	}

	var OriginType string
	if plan.Origin.OriginType.IsUnknown() {
		OriginType = "single_origin"
	} else {
		if plan.Origin.OriginType.ValueString() == "" || plan.Origin.OriginType.IsNull() {
			resp.Diagnostics.AddError("Origin Type",
				"Is not null, Possible choices are: [single_origin]")
			return
		}
		OriginType = plan.Origin.OriginType.ValueString()
	}

	originRequest := edgeapplications.UpdateOriginsRequest{
		Name:                 plan.Origin.Name.ValueString(),
		Addresses:            addressesRequest,
		OriginType:           edgeapplications.PtrString(OriginType),
		OriginProtocolPolicy: edgeapplications.PtrString(originProtocolPolicy),
		HostHeader:           edgeapplications.PtrString(plan.Origin.HostHeader.ValueString()),
		OriginPath:           edgeapplications.PtrString(plan.Origin.OriginPath.ValueString()),
		HmacAuthentication:   edgeapplications.PtrBool(plan.Origin.HMACAuthentication.ValueBool()),
		HmacRegionName:       edgeapplications.PtrString(plan.Origin.HMACRegionName.ValueString()),
		HmacAccessKey:        edgeapplications.PtrString(plan.Origin.HMACAccessKey.ValueString()),
		HmacSecretKey:        edgeapplications.PtrString(plan.Origin.HMACSecretKey.ValueString()),
	}

	originResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsOriginKeyPut(ctx, edgeApplicationID.ValueInt64(), originKey.ValueString()).UpdateOriginsRequest(originRequest).Execute() //nolint
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

	var addresses []OriginAddress
	for _, addr := range originResponse.Results.Addresses {
		addresses = append(addresses, OriginAddress{
			Address:    types.StringValue(addr.GetAddress()),
			Weight:     types.Int64Value(addr.GetWeight()),
			ServerRole: types.StringValue(addr.GetServerRole()),
			IsActive:   types.BoolValue(addr.GetIsActive()),
		})
	}

	plan.Origin = &OriginResourceResults{
		OriginID:                   types.Int64Value(originResponse.Results.GetOriginId()),
		OriginKey:                  types.StringValue(originResponse.Results.GetOriginKey()),
		Name:                       types.StringValue(originResponse.Results.GetName()),
		OriginType:                 types.StringValue(originResponse.Results.GetOriginType()),
		Addresses:                  addresses,
		OriginProtocolPolicy:       types.StringValue(originResponse.Results.GetOriginProtocolPolicy()),
		IsOriginRedirectionEnabled: types.BoolValue(originResponse.Results.GetIsOriginRedirectionEnabled()),
		HostHeader:                 types.StringValue(originResponse.Results.GetHostHeader()),
		Method:                     types.StringValue(originResponse.Results.GetMethod()),
		OriginPath:                 types.StringValue(originResponse.Results.GetOriginPath()),
		ConnectionTimeout:          types.Int64Value(originResponse.Results.GetConnectionTimeout()),
		TimeoutBetweenBytes:        types.Int64Value(originResponse.Results.GetTimeoutBetweenBytes()),
		HMACAuthentication:         types.BoolValue(originResponse.Results.GetHmacAuthentication()),
		HMACRegionName:             types.StringValue(originResponse.Results.GetHmacRegionName()),
		HMACAccessKey:              types.StringValue(originResponse.Results.GetHmacAccessKey()),
		HMACSecretKey:              types.StringValue(originResponse.Results.GetHmacSecretKey()),
	}

	plan.SchemaVersion = types.Int64Value(originResponse.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(*originResponse.Results.OriginId, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *originResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OriginResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	edgeApplicationID := state.ApplicationID.ValueInt64()

	if state.Origin.OriginKey.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Origin Key error ",
			"is not null",
		)
		return
	}

	if state.ApplicationID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}
	response, err := r.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsOriginKeyDelete(ctx, edgeApplicationID, state.Origin.OriginKey.ValueString()).Execute() //nolint
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
}

func (r *originResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
