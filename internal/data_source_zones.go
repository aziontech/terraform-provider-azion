package provider

import (
	"context"
	"fmt"
	"net/http"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	TotalCount types.Int64         `tfsdk:"total_count"`
	TotalPages types.Int64         `tfsdk:"total_pages"`
	Page       types.Int64         `tfsdk:"page"`
	PageSize   types.Int64         `tfsdk:"page_size"`
	Links      *ZonesResponseLinks `tfsdk:"links"`
	Results    []ZonesModel        `tfsdk:"results"`
	ID         types.String        `tfsdk:"id"`
}

type ZonesResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type ZonesModel struct {
	ZoneID         types.Int64  `tfsdk:"zone_id"`
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
			"page": schema.Int64Attribute{
				Description: "The page number of Zones.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number of Zones.",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
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
						"zone_id": schema.Int64Attribute{
							Description: "The zone identifier to target for the resource.",
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

	if Page.IsNull() || Page.IsUnknown() {
		Page = types.Int64Value(1)
	}

	if PageSize.IsNull() || PageSize.IsUnknown() {
		PageSize = types.Int64Value(10)
	}

	zoneResponse, response, err := d.client.api.DNSZonesAPI.ListDnsZones(ctx).
		Page(Page.ValueInt64()).
		PageSize(PageSize.ValueInt64()).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedZoneList, *http.Response, error) {
				return d.client.api.DNSZonesAPI.ListDnsZones(ctx).
					Page(Page.ValueInt64()).
					PageSize(PageSize.ValueInt64()).
					Execute()
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
			usrMsg, errMsg := errPrintZones(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	zoneState := ZonesDataSourceModel{
		Page:     types.Int64Value(Page.ValueInt64()),
		PageSize: types.Int64Value(PageSize.ValueInt64()),
	}

	// Set optional pagination fields
	if zoneResponse.HasCount() {
		zoneState.TotalCount = types.Int64Value(zoneResponse.GetCount())
	} else {
		zoneState.TotalCount = types.Int64Value(0)
	}

	if zoneResponse.HasTotalPages() {
		zoneState.TotalPages = types.Int64Value(zoneResponse.GetTotalPages())
	} else {
		zoneState.TotalPages = types.Int64Value(0)
	}

	// Set links
	zoneState.Links = &ZonesResponseLinks{}
	if zoneResponse.HasPrevious() {
		zoneState.Links.Previous = types.StringValue(zoneResponse.GetPrevious())
	} else {
		zoneState.Links.Previous = types.StringValue("")
	}

	if zoneResponse.HasNext() {
		zoneState.Links.Next = types.StringValue(zoneResponse.GetNext())
	} else {
		zoneState.Links.Next = types.StringValue("")
	}

	// Process results
	if zoneResponse.HasResults() {
		for _, resultZone := range zoneResponse.GetResults() {
			var nameserversList types.List
			if resultZone.GetNameservers() != nil {
				nsSlice := make([]string, len(resultZone.GetNameservers()))
				for i, ns := range resultZone.GetNameservers() {
					nsSlice[i] = ns
				}
				var diagsList diag.Diagnostics
				nameserversList, diagsList = types.ListValueFrom(ctx, types.StringType, nsSlice)
				resp.Diagnostics.Append(diagsList...)
				if resp.Diagnostics.HasError() {
					return
				}
			} else {
				nameserversList = types.ListNull(types.StringType)
			}

			zoneState.Results = append(zoneState.Results, ZonesModel{
				ZoneID:         types.Int64Value(resultZone.GetId()),
				Name:           types.StringValue(resultZone.GetName()),
				Domain:         types.StringValue(resultZone.GetDomain()),
				Active:         types.BoolValue(resultZone.GetActive()),
				Nameservers:    nameserversList,
				ProductVersion: types.StringValue(resultZone.GetProductVersion()),
			})
		}
	}

	zoneState.ID = types.StringValue("Get All Zones")
	diags := resp.State.Set(ctx, &zoneState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintZones(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Zones found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
