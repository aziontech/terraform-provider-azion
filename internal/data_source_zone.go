package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
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
	Data ZoneModel    `tfsdk:"data"`
	ID   types.String `tfsdk:"id"`
}

type ZoneModel struct {
	ZoneID         types.Int64  `tfsdk:"zone_id"`
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
					"zone_id": schema.Int64Attribute{
						Description: "The zone identifier to target for the resource.",
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
						Description: "List of nameservers for the zone.",
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

	zoneId, err := strconv.ParseInt(getZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	zoneResponse, response, err := d.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
				return d.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errPrintZone(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	zoneData := zoneResponse.GetData()

	// Convert nameservers to Terraform List
	var nameserversList types.List
	if zoneData.GetNameservers() != nil {
		nsSlice := make([]string, len(zoneData.GetNameservers()))
		for i, ns := range zoneData.GetNameservers() {
			nsSlice[i] = ns
		}
		nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		nameserversList = types.ListNull(types.StringType)
	}

	zoneState := ZoneDataSourceModel{
		Data: ZoneModel{
			ZoneID:         types.Int64Value(zoneData.GetId()),
			Name:           types.StringValue(zoneData.GetName()),
			Domain:         types.StringValue(zoneData.GetDomain()),
			Active:         types.BoolValue(zoneData.GetActive()),
			Nameservers:    nameserversList,
			ProductVersion: types.StringValue(zoneData.GetProductVersion()),
		},
	}

	zoneState.ID = types.StringValue("Get By ID Zone")
	diags = resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintZone(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "Zone not found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
