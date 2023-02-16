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
	SchemaVersion types.Int64            `tfsdk:"SchemaVersion"`
	Count         types.Int64            `tfsdk:"Count"`
	TotalPages    types.Int64            `tfsdk:"TotalPages"`
	Links         *GetZonesResponseLinks `tfsdk:"Links"`
	Results       []Zone                 `tfsdk:"Results"`
}

type GetZonesResponseLinks struct {
	Previous types.String `tfsdk:"Previous"`
	Next     types.String `tfsdk:"Next"`
}
type Zone struct {
	Id          types.Int64    `tfsdk:"Id"`
	Name        types.String   `tfsdk:"Name"`
	Domain      types.String   `tfsdk:"Domain"`
	IsActive    types.Bool     `tfsdk:"IsActive"`
	Retry       types.Int64    `tfsdk:"Retry"`
	NxTtl       types.Int64    `tfsdk:"NxTtl"`
	SoaTtl      types.Int64    `tfsdk:"SoaTtl"`
	Refresh     types.Int64    `tfsdk:"Refresh"`
	Expiry      types.Int64    `tfsdk:"Expiry"`
	Nameservers []types.String `tfsdk:"Nameservers"`
}

func (d *ZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*idns.APIClient)
}

func (d *ZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zones"
}

func (d *ZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"links": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"previous": schema.StringAttribute{
							Computed: true,
						},
						"next": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"total_pages": schema.Int64Attribute{
				Computed: true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							Required: true,
						},
						"is_active": schema.BoolAttribute{
							Required: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"id": schema.Int64Attribute{
							Required: true,
						},
					},
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
		},
	}

}

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Zones"))
	var state ZoneDataSourceModel
	zoneResponse, _, err := d.client.ZonesApi.GetZones(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Azion Zones",
			err.Error(),
		)
	}

	attributes := map[string]interface{}{
		//"count":          result.Count,
		"total_pages":    zoneResponse.TotalPages,
		"schema_version": zoneResponse.SchemaVersion,
	}

	links := map[string]interface{}{
		//"count":    result.Count,
		"previous": zoneResponse.Links.Previous,
		"next":     zoneResponse.Links.Next,
	}

	attributes["links"] = links

	var res []interface{}
	for _, result := range zoneResponse.Results {
		res = append(res, map[string]interface{}{
			"domain":    result.Domain,
			"is_active": result.IsActive,
			"name":      result.Name,
			"id":        result.Id,
		})
	}

	attributes["results"] = res
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
