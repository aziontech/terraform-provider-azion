package provider

import (
	"context"
	"fmt"
	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"terraform-provider-azion/internal/utils"
)

var (
	_ datasource.DataSource              = &ZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &ZoneDataSource{}
)

func dataSourceAzionZone() datasource.DataSource {
	return &ZoneDataSource{}
}

type ZoneDataSource struct {
	client *idns.APIClient
}

type ZoneDataSourceModel struct {
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	Results       Zone         `tfsdk:"results"`
	ID            types.String `tfsdk:"id"`
}

type Zone struct {
	ID          types.Int64  `tfsdk:"zone_id"`
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
	d.client = req.ProviderData.(*idns.APIClient)
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (d *ZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"zone_id": schema.Int64Attribute{
						Required: true,
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Name description of the DNS.",
					},
					"domain": schema.StringAttribute{
						Computed:    true,
						Description: "Domain description of the DNS.",
					},
					"is_active": schema.BoolAttribute{
						Computed:    true,
						Description: "Enable description of the DNS.",
					},
					"retry": schema.Int64Attribute{
						Computed: true,
					},
					"nxttl": schema.Int64Attribute{
						Computed: true,
					},
					"soattl": schema.Int64Attribute{
						Computed: true,
					},
					"refresh": schema.Int64Attribute{
						Computed: true,
					},
					"expiry": schema.Int64Attribute{
						Computed: true,
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
	tflog.Debug(ctx, fmt.Sprintf("Reading Zones"))

	var getZoneId types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneId := int32(getZoneId.ValueInt64())

	zoneResponse, response, err := d.client.ZonesApi.GetZone(ctx, zoneId).Execute()
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
	var slice []types.String
	for _, Nameservers := range zoneResponse.Results.Nameservers {
		slice = append(slice, types.StringValue(Nameservers))
	}
	zoneState := ZoneDataSourceModel{
		SchemaVersion: types.Int64Value(int64(*zoneResponse.SchemaVersion)),
		Results: Zone{
			ID:          types.Int64Value(int64(*zoneResponse.Results.Id)),
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
