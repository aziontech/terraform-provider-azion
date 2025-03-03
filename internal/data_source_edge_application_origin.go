package provider

import (
	"context"
	"io"

	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &OriginDataSource{}
	_ datasource.DataSourceWithConfigure = &OriginDataSource{}
)

func dataSourceAzionEdgeApplicationOrigin() datasource.DataSource {
	return &OriginDataSource{}
}

type OriginDataSource struct {
	client *apiClient
}

type OriginDataSourceModel struct {
	SchemaVersion types.Int64   `tfsdk:"schema_version"`
	ID            types.String  `tfsdk:"id"`
	ApplicationID types.Int64   `tfsdk:"edge_application_id"`
	Results       OriginResults `tfsdk:"origin"`
}

type OriginResults struct {
	OriginId                   types.Int64            `tfsdk:"origin_id"`
	OriginKey                  types.String           `tfsdk:"origin_key"`
	Name                       types.String           `tfsdk:"name"`
	OriginType                 types.String           `tfsdk:"origin_type"`
	Addresses                  []OriginAddressResults `tfsdk:"addresses"`
	OriginProtocolPolicy       types.String           `tfsdk:"origin_protocol_policy"`
	IsOriginRedirectionEnabled types.Bool             `tfsdk:"is_origin_redirection_enabled"`
	HostHeader                 types.String           `tfsdk:"host_header"`
	Method                     types.String           `tfsdk:"method"`
	OriginPath                 types.String           `tfsdk:"origin_path"`
	ConnectionTimeout          types.Int64            `tfsdk:"connection_timeout"`
	TimeoutBetweenBytes        types.Int64            `tfsdk:"timeout_between_bytes"`
	HMACAuthentication         types.Bool             `tfsdk:"hmac_authentication"`
	HMACRegionName             types.String           `tfsdk:"hmac_region_name"`
	HMACAccessKey              types.String           `tfsdk:"hmac_access_key"`
	HMACSecretKey              types.String           `tfsdk:"hmac_secret_key"`
}

type OriginAddressResults struct {
	Address    types.String `tfsdk:"address"`
	Weight     types.Int64  `tfsdk:"weight"`
	ServerRole types.String `tfsdk:"server_role"`
	IsActive   types.Bool   `tfsdk:"is_active"`
}

func (o *OriginDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *OriginDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_origin"
}

func (o *OriginDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "The edge application identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"origin": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"origin_id": schema.Int64Attribute{
						Description: "The origin identifier to target for the resource.",
						Computed:    true,
					},
					"origin_key": schema.StringAttribute{
						Description: "Origin key.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the origin.",
						Computed:    true,
					},
					"origin_type": schema.StringAttribute{
						Description: "Type of the origin.",
						Computed:    true,
					},
					"addresses": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"address": schema.StringAttribute{
									Description: "Address of the origin.",
									Computed:    true,
								},
								"weight": schema.Int64Attribute{
									Description: "Weight of the origin.",
									Computed:    true,
								},
								"server_role": schema.StringAttribute{
									Description: "Server role of the origin.",
									Computed:    true,
								},
								"is_active": schema.BoolAttribute{
									Description: "Status of the origin.",
									Computed:    true,
								},
							},
						},
					},
					"origin_protocol_policy": schema.StringAttribute{
						Description: "Origin protocol policy.",
						Computed:    true,
					},
					"is_origin_redirection_enabled": schema.BoolAttribute{
						Description: "Whether origin redirection is enabled.",
						Computed:    true,
					},
					"host_header": schema.StringAttribute{
						Description: "Host header value.",
						Computed:    true,
					},
					"method": schema.StringAttribute{
						Description: "HTTP method used by the origin.",
						Computed:    true,
					},
					"origin_path": schema.StringAttribute{
						Description: "Path of the origin.",
						Computed:    true,
					},
					"connection_timeout": schema.Int64Attribute{
						Description: "Connection timeout in seconds.",
						Computed:    true,
					},
					"timeout_between_bytes": schema.Int64Attribute{
						Description: "Timeout between bytes in seconds.",
						Computed:    true,
					},
					"hmac_authentication": schema.BoolAttribute{
						Description: "Whether HMAC authentication is enabled.",
						Computed:    true,
					},
					"hmac_region_name": schema.StringAttribute{
						Description: "HMAC region name.",
						Computed:    true,
					},
					"hmac_access_key": schema.StringAttribute{
						Description: "HMAC access key.",
						Computed:    true,
					},
					"hmac_secret_key": schema.StringAttribute{
						Description: "HMAC secret key.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (o *OriginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeApplicationID types.Int64
	var getOriginsKey types.String
	diags := req.Config.GetAttribute(ctx, path.Root("origin").AtName("origin_key"), &getOriginsKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if getOriginsKey.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	originResponse, response, err := o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(ctx, edgeApplicationID.ValueInt64(), getOriginsKey.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			err := utils.SleepAfter429(response)
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
				)
				return
			}
			originResponse, _, err = o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsOriginKeyGet(ctx, edgeApplicationID.ValueInt64(), getOriginsKey.ValueString()).Execute() //nolint
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
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

	var addresses []OriginAddressResults
	for _, addr := range originResponse.Results.Addresses {
		addresses = append(addresses, OriginAddressResults{
			Address:    types.StringValue(addr.GetAddress()),
			Weight:     types.Int64Value(addr.GetWeight()),
			ServerRole: types.StringValue(addr.GetServerRole()),
			IsActive:   types.BoolValue(addr.GetIsActive()),
		})
	}

	origin := OriginResults{
		OriginId:                   types.Int64Value(originResponse.Results.GetOriginId()),
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

	edgeApplicationOriginState := OriginDataSourceModel{
		SchemaVersion: types.Int64Value(originResponse.SchemaVersion),
		Results:       origin,
	}

	edgeApplicationOriginState.ID = types.StringValue("Get By Key Edge Application Origins")
	diags = resp.State.Set(ctx, &edgeApplicationOriginState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
