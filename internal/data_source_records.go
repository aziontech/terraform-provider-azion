package provider

import (
	"context"
	"fmt"
	"net/http"

	dnsapi "github.com/aziontech/azionapi-v4-go-sdk-dev/dns-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &RecordsDataSource{}
	_ datasource.DataSourceWithConfigure = &RecordsDataSource{}
)

func dataSourceAzionRecords() datasource.DataSource {
	return &RecordsDataSource{}
}

type RecordsDataSource struct {
	client *apiClient
}

type RecordsDataSourceModel struct {
	ZoneId  types.String `tfsdk:"zone_id"`
	Counter types.Int64  `tfsdk:"counter"`
	Results []RecordData `tfsdk:"results"`
	Id      types.String `tfsdk:"id"`
}

type RecordData struct {
	ID          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Name        types.String `tfsdk:"name"`
	Ttl         types.Int64  `tfsdk:"ttl"`
	Type        types.String `tfsdk:"type"`
	Rdata       types.List   `tfsdk:"rdata"`
	Policy      types.String `tfsdk:"policy"`
	Weight      types.Int64  `tfsdk:"weight"`
}

func (d *RecordsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *RecordsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_records"
}

func (d *RecordsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
			},
			"zone_id": schema.StringAttribute{
				Required:    true,
				Description: "The zone identifier to target for the resource.",
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of records.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "ID of the DNS record.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Description: "Record name.",
							Computed:    true,
						},
						"ttl": schema.Int64Attribute{
							Description: "Time-to-live defines max-time for packets life in seconds.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "DNS record type.",
							Computed:    true,
						},
						"rdata": schema.ListAttribute{
							Description: "Record data values (answers).",
							ElementType: types.StringType,
							Computed:    true,
						},
						"policy": schema.StringAttribute{
							Description: "Must be 'simple' or 'weighted'.",
							Computed:    true,
						},
						"weight": schema.Int64Attribute{
							Description: "Weight of the record (for weighted policies).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *RecordsDataSource) errorPrint(resp *datasource.ReadResponse, errCode int) {
	var usrMsg string
	switch errCode {
	case 404:
		usrMsg = "No Records Found"
	case 401:
		usrMsg = "Unauthorized Token"
	default:
		usrMsg = "Cannot read Azion response"
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	resp.Diagnostics.AddError(
		usrMsg,
		errMsg,
	)
}

func (d *RecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getZoneId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("zone_id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := getZoneId.ValueString()

	recordsResponse, httpResp, err := d.client.idnsApi.DNSRecordsAPI.
		ListDnsRecords(ctx, zoneID).Execute() //nolint
	if err != nil {
		if httpResp.StatusCode == 429 {
			recordsResponse, httpResp, err = utils.RetryOn429(func() (*dnsapi.PaginatedRecordList, *http.Response, error) {
				return d.client.idnsApi.DNSRecordsAPI.
					ListDnsRecords(ctx, zoneID).Execute() //nolint
			}, 5) // Maximum 5 retries

			if httpResp != nil {
				defer httpResp.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			d.errorPrint(resp, httpResp.StatusCode)
			return
		}
	}

	recordsState := RecordsDataSourceModel{
		ZoneId:  getZoneId,
		Counter: types.Int64Value(*recordsResponse.Count),
		Id:      types.StringValue("Get DNS records"),
	}

	for _, resultRecord := range recordsResponse.Results {
		var rdataValues []types.String
		for _, rdata := range resultRecord.Rdata {
			rdataValues = append(rdataValues, types.StringValue(rdata))
		}

		rdataList := utils.SliceStringTypeToList(rdataValues)

		var description string
		if resultRecord.Description != nil {
			description = *resultRecord.Description
		}

		var ttl int64
		if resultRecord.Ttl != nil {
			ttl = *resultRecord.Ttl
		}

		var policy string
		if resultRecord.Policy != nil {
			policy = *resultRecord.Policy
		}

		var weight int64
		if resultRecord.Weight != nil {
			weight = *resultRecord.Weight
		}

		record := RecordData{
			ID:          types.Int64Value(resultRecord.Id),
			Description: types.StringValue(description),
			Name:        types.StringValue(resultRecord.Name),
			Ttl:         types.Int64Value(ttl),
			Type:        types.StringValue(resultRecord.Type),
			Rdata:       rdataList,
			Policy:      types.StringValue(policy),
			Weight:      types.Int64Value(weight),
		}

		recordsState.Results = append(recordsState.Results, record)
	}

	diags = resp.State.Set(ctx, &recordsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
