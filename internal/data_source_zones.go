package provider

import (
	"context"
	"io"

	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ZonesDataSource{}
	_ datasource.DataSourceWithConfigure = &ZonesDataSource{}
)

func dataSourceAzionZones() datasource.DataSource {
	return &ZonesDataSource{}
}

type ZonesDataSource struct {
	client *apiClient
}

type ZonesDataSourceModel struct {
	SchemaVersion types.Int64            `tfsdk:"schema_version"`
	Counter       types.Int64            `tfsdk:"counter"`
	TotalPages    types.Int64            `tfsdk:"total_pages"`
	Page          types.Int64            `tfsdk:"page"`
	PageSize      types.Int64            `tfsdk:"page_size"`
	Links         *GetZonesResponseLinks `tfsdk:"links"`
	Results       []Zones                `tfsdk:"results"`
	ID            types.String           `tfsdk:"id"`
}

type GetZonesResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}
type Zones struct {
	ZoneId   types.Int64  `tfsdk:"zone_id"`
	Name     types.String `tfsdk:"name"`
	Domain   types.String `tfsdk:"domain"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

func (d *ZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *ZonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_zones"
}

func (d *ZonesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Zones.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number of Zones.",
				Optional:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of zones.",
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
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							Description: "Domain name attributed by Azion to this configuration.",
							Computed:    true,
						},
						"is_active": schema.BoolAttribute{
							Computed:    true,
							Description: "Status of the zone.",
						},
						"name": schema.StringAttribute{
							Description: "The name of the zone. Must provide only one of zone_id, name.",
							Computed:    true,
						},
						"zone_id": schema.Int64Attribute{
							Description: "The zone identifier to target for the resource.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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

	zoneResponse, response, err := d.client.idnsApi.ZonesAPI.GetZones(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			resp.Diagnostics.AddWarning(
				"Too many requests",
				"Terraform provider will wait some time before attempting this request again. Please wait.",
			)
			err := utils.SleepAfter429(response)
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
				)
				return
			}
			zoneResponse, _, err = d.client.idnsApi.ZonesAPI.GetZones(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
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

	zoneState := ZonesDataSourceModel{
		SchemaVersion: types.Int64Value(int64(zoneResponse.GetSchemaVersion())),
		TotalPages:    types.Int64Value(int64(zoneResponse.GetTotalPages())),
		Page:          types.Int64Value(Page.ValueInt64()),
		PageSize:      types.Int64Value(PageSize.ValueInt64()),
		Counter:       types.Int64Value(int64(zoneResponse.GetCount())),
		Links: &GetZonesResponseLinks{
			Previous: types.StringValue(zoneResponse.Links.GetPrevious()),
			Next:     types.StringValue(zoneResponse.Links.GetNext()),
		},
	}
	for _, resultZone := range zoneResponse.Results {
		zoneState.Results = append(zoneState.Results, Zones{
			ZoneId:   types.Int64Value(int64(resultZone.GetId())),
			Name:     types.StringValue(resultZone.GetName()),
			Domain:   types.StringValue(resultZone.GetDomain()),
			IsActive: types.BoolValue(resultZone.GetIsActive()),
		})
	}
	zoneState.ID = types.StringValue("Get All Zones")
	diags := resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
