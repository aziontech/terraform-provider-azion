package provider

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &OriginsDataSource{}
	_ datasource.DataSourceWithConfigure = &OriginsDataSource{}
)

func dataSourceAzionEdgeApplicationsOrigins() datasource.DataSource {
	return &OriginsDataSource{}
}

type OriginsDataSource struct {
	client *apiClient
}

type OriginsDataSourceModel struct {
	SchemaVersion types.Int64                              `tfsdk:"schema_version"`
	ID            types.String                             `tfsdk:"id"`
	ApplicationID types.Int64                              `tfsdk:"edge_application_id"`
	Counter       types.Int64                              `tfsdk:"counter"`
	TotalPages    types.Int64                              `tfsdk:"total_pages"`
	Page          types.Int64                              `tfsdk:"page"`
	PageSize      types.Int64                              `tfsdk:"page_size"`
	Links         *GetEdgeApplicationsOriginsResponseLinks `tfsdk:"links"`
	Results       []OriginsResults                         `tfsdk:"results"`
}

type GetEdgeApplicationsOriginsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type OriginsResults struct {
	OriginId                   types.Int64             `tfsdk:"origin_id"`
	OriginKey                  types.String            `tfsdk:"origin_key"`
	Name                       types.String            `tfsdk:"name"`
	OriginType                 types.String            `tfsdk:"origin_type"`
	Addresses                  []OriginsAddressResults `tfsdk:"addresses"`
	OriginProtocolPolicy       types.String            `tfsdk:"origin_protocol_policy"`
	IsOriginRedirectionEnabled types.Bool              `tfsdk:"is_origin_redirection_enabled"`
	HostHeader                 types.String            `tfsdk:"host_header"`
	Method                     types.String            `tfsdk:"method"`
	OriginPath                 types.String            `tfsdk:"origin_path"`
	ConnectionTimeout          types.Int64             `tfsdk:"connection_timeout"`
	TimeoutBetweenBytes        types.Int64             `tfsdk:"timeout_between_bytes"`
	HMACAuthentication         types.Bool              `tfsdk:"hmac_authentication"`
	HMACRegionName             types.String            `tfsdk:"hmac_region_name"`
	HMACAccessKey              types.String            `tfsdk:"hmac_access_key"`
	HMACSecretKey              types.String            `tfsdk:"hmac_secret_key"`
}

type OriginsAddressResults struct {
	Address    types.String `tfsdk:"address"`
	Weight     types.Int64  `tfsdk:"weight"`
	ServerRole types.String `tfsdk:"server_role"`
	IsActive   types.Bool   `tfsdk:"is_active"`
}

func (o *OriginsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *OriginsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_applications_origins"
}

func (o *OriginsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
						"origin_id": schema.Int64Attribute{
							Description: "The origin identifier to target for the resource.",
							Computed:    true,
						},
						"origin_key": schema.StringAttribute{
							Description: "Origin key.",
							Computed:    true,
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
		},
	}
}

func (o *OriginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeApplicationID types.Int64
	var Page types.Int64
	var PageSize types.Int64

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	originsResponse, response, err := o.client.edgeApplicationsApi.EdgeApplicationsOriginsAPI.EdgeApplicationsEdgeApplicationIdOriginsGet(ctx, edgeApplicationID.ValueInt64()).Execute()
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
	if originsResponse.Links.Previous.Get() != nil {
		previous = *originsResponse.Links.Previous.Get()
	}
	if originsResponse.Links.Next.Get() != nil {
		next = *originsResponse.Links.Next.Get()
	}

	log.Println("HERE 01")

	var origins []OriginsResults
	for _, origin := range originsResponse.Results {
		var addresses []OriginsAddressResults
		for _, addr := range origin.Addresses {
			addresses = append(addresses, OriginsAddressResults{
				Address:    types.StringValue(addr.GetAddress()),
				Weight:     types.Int64Value(addr.GetWeight()),
				ServerRole: types.StringValue(addr.GetServerRole()),
				IsActive:   types.BoolValue(addr.GetIsActive()),
			})
		}

		origins = append(origins, OriginsResults{
			OriginId:                   types.Int64Value(origin.GetOriginId()),
			OriginKey:                  types.StringValue(origin.GetOriginKey()),
			Name:                       types.StringValue(origin.GetName()),
			OriginType:                 types.StringValue(origin.GetOriginType()),
			Addresses:                  addresses,
			OriginProtocolPolicy:       types.StringValue(origin.GetOriginProtocolPolicy()),
			IsOriginRedirectionEnabled: types.BoolValue(origin.GetIsOriginRedirectionEnabled()),
			HostHeader:                 types.StringValue(origin.GetHostHeader()),
			Method:                     types.StringValue(origin.GetMethod()),
			OriginPath:                 types.StringValue(origin.GetOriginPath()),
			ConnectionTimeout:          types.Int64Value(origin.GetConnectionTimeout()),
			TimeoutBetweenBytes:        types.Int64Value(origin.GetTimeoutBetweenBytes()),
			HMACAuthentication:         types.BoolValue(origin.GetHmacAuthentication()),
			HMACRegionName:             types.StringValue(origin.GetHmacRegionName()),
			HMACAccessKey:              types.StringValue(origin.GetHmacAccessKey()),
			HMACSecretKey:              types.StringValue(origin.GetHmacSecretKey()),
		})
	}

	edgeApplicationsOriginsState := OriginsDataSourceModel{
		SchemaVersion: types.Int64Value(originsResponse.SchemaVersion),
		Results:       origins,
		TotalPages:    types.Int64Value(originsResponse.TotalPages),
		Counter:       types.Int64Value(originsResponse.Count),
		Links: &GetEdgeApplicationsOriginsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	edgeApplicationsOriginsState.ID = types.StringValue("Get All Edge Application Origins")
	diags := resp.State.Set(ctx, &edgeApplicationsOriginsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
