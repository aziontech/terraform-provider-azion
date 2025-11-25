package provider

import (
	"context"
	"io"
	"net/http"

	dnsapi "github.com/aziontech/azionapi-v4-go-sdk-dev/dns-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"

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
	Counter types.Int64  `tfsdk:"counter"`
	Results []ZoneResult `tfsdk:"results"`
	ID      types.String `tfsdk:"id"`
}

type ZoneResult struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Domain         types.String `tfsdk:"domain"`
	Active         types.Bool   `tfsdk:"active"`
	Nameservers    types.List   `tfsdk:"nameservers"`
	ProductVersion types.String `tfsdk:"product_version"`
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
			"counter": schema.Int64Attribute{
				Description: "The total number of zones.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
							Description: "Domain name attributed by Azion to this configuration.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Status of the zone.",
							Computed:    true,
						},
						"nameservers": schema.ListAttribute{
							Description: "List of nameservers for the zone.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "Product version of the zone.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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

	zoneState := ZonesDataSourceModel{
		Counter: types.Int64Value(int64(zoneResponse.GetCount())),
	}

	for _, resultZone := range zoneResponse.Results {
		var nameserverValues []types.String
		for _, ns := range resultZone.GetNameservers() {
			nameserverValues = append(nameserverValues, types.StringValue(ns))
		}

		nameserversList, diag := types.ListValueFrom(ctx, types.StringType, nameserverValues)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}

		zoneState.Results = append(zoneState.Results, ZoneResult{
			ID:             types.Int64Value(int64(resultZone.GetId())),
			Name:           types.StringValue(resultZone.GetName()),
			Domain:         types.StringValue(resultZone.GetDomain()),
			Active:         types.BoolValue(resultZone.GetActive()),
			Nameservers:    nameserversList,
			ProductVersion: types.StringValue("1"),
		})
	}

	zoneState.ID = types.StringValue("Get All Zones")
	diags := resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
