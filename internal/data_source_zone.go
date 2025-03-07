package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/aziontech/terraform-provider-azion/internal/utils"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &ZoneDataSource{}
)

func dataSourceAzionZone() datasource.DataSource {
	return &ZoneDataSource{}
}

type ZoneDataSource struct {
	client *apiClient
}

type ZoneDataSourceModel struct {
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	Results       Zone         `tfsdk:"results"`
	ID            types.String `tfsdk:"id"`
}

type Zone struct {
	ZoneID      types.Int64  `tfsdk:"zone_id"`
	Name        types.String `tfsdk:"name"`
	Domain      types.String `tfsdk:"domain"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Retry       types.Int64  `tfsdk:"retry"`
	NxTtl       types.Int64  `tfsdk:"nxttl"`
	SoaTtl      types.Int64  `tfsdk:"soattl"`
	Refresh     types.Int64  `tfsdk:"refresh"`
	Expiry      types.Int64  `tfsdk:"expiry"`
	Nameservers types.List   `tfsdk:"nameservers"`
}

func (d *ZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_zone"
}

func (d *ZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Optional:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"zone_id": schema.Int64Attribute{
						Description: "The zone identifier to target for the resource.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the zone. Must provide only one of zone_id, name.",
						Computed:    true,
					},
					"domain": schema.StringAttribute{
						Computed:    true,
						Description: "Domain name attributed by Azion to this configuration.",
					},
					"is_active": schema.BoolAttribute{
						Computed:    true,
						Description: "Status of the zone.",
					},
					"retry": schema.Int64Attribute{
						Computed:    true,
						Description: "The rate at which a secondary server will retry to refresh the primary zone file if the initial refresh failed.",
					},
					"nxttl": schema.Int64Attribute{
						Computed:    true,
						Description: "In the event that requesting the domain results in a non-existent query (NXDOMAIN), this is the amount of time that is respected by the recursor to return the NXDOMAIN response.",
					},
					"soattl": schema.Int64Attribute{
						Computed:    true,
						Description: "The interval at which the SOA record itself is refreshed.",
					},
					"refresh": schema.Int64Attribute{
						Computed:    true,
						Description: "The interval at which secondary servers (secondary DNS) are set to refresh the primary zone file from the primary server.",
					},
					"expiry": schema.Int64Attribute{
						Computed:    true,
						Description: "If Refresh and Retry fail repeatedly, this is the time period after which the primary should be considered gone and no longer authoritative for the given zone.",
					},
					"nameservers": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getZoneId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneId, err := strconv.ParseInt(getZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	zoneResponse, response, err := d.client.idnsApi.ZonesAPI.GetZone(ctx, int32(zoneId)).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*idns.GetZoneResponse, *http.Response, error) {
				return d.client.idnsApi.ZonesAPI.GetZone(ctx, int32(zoneId)).Execute() //nolint
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

	var slice []types.String
	for _, Nameservers := range zoneResponse.Results.Nameservers {
		slice = append(slice, types.StringValue(Nameservers))
	}
	zoneState := ZoneDataSourceModel{
		SchemaVersion: types.Int64Value(int64(*zoneResponse.SchemaVersion)),
		Results: Zone{
			ZoneID:      types.Int64Value(int64(*zoneResponse.Results.Id)),
			Name:        types.StringValue(*zoneResponse.Results.Name),
			Domain:      types.StringValue(*zoneResponse.Results.Domain),
			IsActive:    types.BoolValue(*zoneResponse.Results.IsActive),
			NxTtl:       types.Int64Value(int64(*idns.NullableInt32.Get(zoneResponse.Results.NxTtl))),
			Retry:       types.Int64Value(int64(*idns.NullableInt32.Get(zoneResponse.Results.Retry))),
			Refresh:     types.Int64Value(int64(*idns.NullableInt32.Get(zoneResponse.Results.Refresh))),
			Expiry:      types.Int64Value(int64(*idns.NullableInt32.Get(zoneResponse.Results.Expiry))),
			SoaTtl:      types.Int64Value(int64(*idns.NullableInt32.Get(zoneResponse.Results.SoaTtl))),
			Nameservers: utils.SliceStringTypeToList(slice),
		},
	}
	zoneState.ID = types.StringValue("Get By ID Zone")
	diags = resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
