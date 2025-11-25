package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/aziontech/terraform-provider-azion/internal/utils"

	dnsapi "github.com/aziontech/azionapi-v4-go-sdk-dev/dns-api"
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
	Data *ZoneData    `tfsdk:"data"`
	ID   types.String `tfsdk:"id"`
}

type ZoneData struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Domain         types.String `tfsdk:"domain"`
	Active         types.Bool   `tfsdk:"active"`
	Nameservers    types.List   `tfsdk:"nameservers"`
	ProductVersion types.String `tfsdk:"product_version"`
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
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the zone.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the zone.",
						Computed:    true,
					},
					"domain": schema.StringAttribute{
						Computed:    true,
						Description: "Domain name attributed by Azion to this configuration.",
					},
					"active": schema.BoolAttribute{
						Computed:    true,
						Description: "Status of the zone.",
					},
					"nameservers": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"product_version": schema.StringAttribute{
						Computed:    true,
						Description: "Product version of the zone.",
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

	zoneResponse, response, err := d.client.idnsApi.DNSZonesAPI.ListDnsZones(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			zoneResponse, response, err = utils.RetryOn429(func() (*dnsapi.PaginatedZoneList, *http.Response, error) {
				return d.client.idnsApi.DNSZonesAPI.ListDnsZones(ctx).Execute() //nolint
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

	var selectedZone *dnsapi.Zone
	for _, z := range zoneResponse.Results {
		if int64(z.GetId()) == zoneId {
			zoneCopy := z
			selectedZone = &zoneCopy
			break
		}
	}

	if selectedZone == nil {
		resp.Diagnostics.AddError(
			"Zone not found",
			"No DNS zone found with the given ID",
		)
		return
	}

	var slice []types.String
	for _, ns := range selectedZone.GetNameservers() {
		slice = append(slice, types.StringValue(ns))
	}

	nameserversList := utils.SliceStringTypeToList(slice)

	zoneState := ZoneDataSourceModel{
		Data: &ZoneData{
			ID:             types.Int64Value(int64(selectedZone.GetId())),
			Name:           types.StringValue(selectedZone.GetName()),
			Domain:         types.StringValue(selectedZone.GetDomain()),
			Active:         types.BoolValue(selectedZone.GetActive()),
			Nameservers:    nameserversList,
			ProductVersion: types.StringValue("1"),
		},
	}
	zoneState.ID = types.StringValue("Get Zone by ID")
	diags = resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
