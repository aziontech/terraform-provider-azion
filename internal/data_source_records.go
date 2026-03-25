package provider

import (
	"context"
	"fmt"
	"net/http"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
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
	ZoneId     types.Int64              `tfsdk:"zone_id"`
	TotalPages types.Int64              `tfsdk:"total_pages"`
	Page       types.Int64              `tfsdk:"page"`
	PageSize   types.Int64              `tfsdk:"page_size"`
	Counter    types.Int64              `tfsdk:"counter"`
	Links      *RecordsResponseLinks    `tfsdk:"links"`
	Results    []RecordDataSourceResult `tfsdk:"results"`
	Id         types.String             `tfsdk:"id"`
}

type RecordsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type RecordDataSourceResult struct {
	RecordId    types.Int64    `tfsdk:"record_id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Rdata       []types.String `tfsdk:"rdata"`
	Policy      types.String   `tfsdk:"policy"`
	Type        types.String   `tfsdk:"type"`
	Ttl         types.Int64    `tfsdk:"ttl"`
	Weight      types.Int64    `tfsdk:"weight"`
}

func (d *RecordsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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
			"zone_id": schema.Int64Attribute{
				Required:    true,
				Description: "The zone identifier to target for the resource.",
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Records.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number of Records.",
				Optional:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of records.",
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
						"record_id": schema.Int64Attribute{
							Description: "The record identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the DNS record.",
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"rdata": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of answers replied by DNS Authoritative to that Record.",
						},
						"policy": schema.StringAttribute{
							Computed:    true,
							Description: "Must be 'simple' or 'weighted'.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "DNS record type (A, AAAA, ANAME, CNAME, MX, NS, PTR, SRV, TXT, CAA, DS).",
						},
						"ttl": schema.Int64Attribute{
							Computed:    true,
							Description: "Time-to-live defines max-time for packets life in seconds.",
						},
						"weight": schema.Int64Attribute{
							Computed:    true,
							Description: "Weight for weighted policy records.",
						},
					},
				},
			},
		},
	}
}

func (d *RecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var zoneId types.Int64

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsZoneId := req.Config.GetAttribute(ctx, path.Root("zone_id"), &zoneId)
	resp.Diagnostics.Append(diagsZoneId...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set default values for pagination.
	if page.IsNull() || page.IsUnknown() {
		page = types.Int64Value(1)
	}
	if pageSize.IsNull() || pageSize.IsUnknown() {
		pageSize = types.Int64Value(10)
	}

	// Build the API request.
	listRequest := d.client.api.DNSRecordsAPI.ListDnsRecords(ctx, zoneId.ValueInt64()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64())

	// Execute the request.
	recordsResponse, httpResp, err := listRequest.Execute() //nolint
	if err != nil {
		if httpResp.StatusCode == 429 {
			recordsResponse, httpResp, err = utils.RetryOn429(func() (*azionapi.PaginatedRecordList, *http.Response, error) {
				return d.client.api.DNSRecordsAPI.ListDnsRecords(ctx, zoneId.ValueInt64()).
					Page(page.ValueInt64()).
					PageSize(pageSize.ValueInt64()).
					Execute()
			}, 5) // Maximum 5 retries

			if httpResp != nil {
				defer httpResp.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errPrintRecords(httpResp.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if httpResp != nil {
		defer httpResp.Body.Close()
	}

	// Build the state from the response.
	recordsState := buildRecordsState(ctx, zoneId, page, pageSize, recordsResponse)

	// Set the state.
	diags := resp.State.Set(ctx, &recordsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// buildRecordsState constructs the state model from the API response.
func buildRecordsState(ctx context.Context, zoneId types.Int64, page types.Int64, pageSize types.Int64, response *azionapi.PaginatedRecordList) RecordsDataSourceModel {
	state := RecordsDataSourceModel{
		ZoneId:   zoneId,
		Page:     page,
		PageSize: pageSize,
		Links:    &RecordsResponseLinks{},
	}

	// Set counter.
	if response.Count != nil {
		state.Counter = types.Int64Value(*response.Count)
	}

	// Set total pages.
	if response.TotalPages != nil {
		state.TotalPages = types.Int64Value(*response.TotalPages)
	}

	// Set links.
	if response.HasPrevious() {
		state.Links.Previous = types.StringValue(response.GetPrevious())
	} else {
		state.Links.Previous = types.StringNull()
	}

	if response.HasNext() {
		state.Links.Next = types.StringValue(response.GetNext())
	} else {
		state.Links.Next = types.StringNull()
	}

	// Set results.
	if response.HasResults() {
		for _, record := range response.GetResults() {
			recordResult := RecordDataSourceResult{
				RecordId: types.Int64Value(record.GetId()),
				Name:     types.StringValue(record.GetName()),
				Type:     types.StringValue(record.GetType()),
			}

			// Set optional description.
			if record.HasDescription() {
				recordResult.Description = types.StringValue(record.GetDescription())
			} else {
				recordResult.Description = types.StringNull()
			}

			// Set optional TTL.
			if record.HasTtl() {
				recordResult.Ttl = types.Int64Value(record.GetTtl())
			} else {
				recordResult.Ttl = types.Int64Null()
			}

			// Set optional policy.
			if record.HasPolicy() {
				recordResult.Policy = types.StringValue(record.GetPolicy())
			} else {
				recordResult.Policy = types.StringNull()
			}

			// Set optional weight.
			if record.HasWeight() {
				recordResult.Weight = types.Int64Value(record.GetWeight())
			} else {
				recordResult.Weight = types.Int64Null()
			}

			// Set rdata list.
			rdata := record.GetRdata()
			rdataList := make([]types.String, len(rdata))
			for i, d := range rdata {
				rdataList[i] = types.StringValue(d)
			}
			recordResult.Rdata = rdataList

			state.Results = append(state.Results, recordResult)
		}
	}

	// Set placeholder ID.
	state.Id = types.StringValue("placeholder")

	return state
}

// errPrintRecords returns user-friendly error messages for records operations.
func errPrintRecords(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Records Found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
