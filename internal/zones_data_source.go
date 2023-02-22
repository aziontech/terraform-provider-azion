package provider

import (
	"context"
	"fmt"
	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	SchemaVersion types.Int64            `tfsdk:"schema_version"`
	Counter       types.Int64            `tfsdk:"counter"`
	TotalPages    types.Int64            `tfsdk:"total_pages"`
	Links         *GetZonesResponseLinks `tfsdk:"links"`
	Results       []Zone                 `tfsdk:"results"`
}

type GetZonesResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}
type Zone struct {
	Id       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Domain   types.String `tfsdk:"domain"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

func (d *ZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*idns.APIClient)
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zones"
}

func (d *ZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"counter": schema.Int64Attribute{
				Computed: true,
			},
			"total_pages": schema.Int64Attribute{
				Computed: true,
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
							Computed: true,
						},
						"is_active": schema.BoolAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"id": schema.Int64Attribute{
							Computed: true,
						},
					},
				},
			},
		},
	}

}

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Zones"))
	zoneResponse, _, err := d.client.ZonesApi.GetZones(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Token has expired",
			err.Error(),
		)
		return
	}
	var previous, next string
	if zoneResponse.Links != nil {
		if zoneResponse.Links.Previous.Get() != nil {
			previous = *zoneResponse.Links.Previous.Get()
		}
		if zoneResponse.Links.Next.Get() != nil {
			next = *zoneResponse.Links.Next.Get()
		}
	}
	zoneState := ZoneDataSourceModel{
		SchemaVersion: types.Int64Value(int64(*zoneResponse.SchemaVersion)),
		TotalPages:    types.Int64Value(int64(*zoneResponse.TotalPages)),
		Counter:       types.Int64Value(int64(*zoneResponse.Count)),
		Links: &GetZonesResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}
	for _, resultZone := range zoneResponse.Results {
		zoneState.Results = append(zoneState.Results, Zone{
			Domain:   types.StringValue(*resultZone.Domain),
			IsActive: types.BoolValue(*resultZone.IsActive),
			Name:     types.StringValue(*resultZone.Name),
			Id:       types.Int64Value(int64(*resultZone.Id)),
		})
	}
	diags := resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
