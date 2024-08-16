package provider

import (
	"context"
	"fmt"

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
	ZoneId        types.Int64                `tfsdk:"zone_id"`
	SchemaVersion types.Int64                `tfsdk:"schema_version"`
	TotalPages    types.Int64                `tfsdk:"total_pages"`
	Page          types.Int64                `tfsdk:"page"`
	PageSize      types.Int64                `tfsdk:"page_size"`
	Counter       types.Int64                `tfsdk:"counter"`
	Links         *GetRecordsResponseLinks   `tfsdk:"links"`
	Results       *GetRecordsResponseResults `tfsdk:"results"`
	Id            types.String               `tfsdk:"id"`
}

type GetRecordsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type GetRecordsResponseResults struct {
	ZoneId  types.Int64  `tfsdk:"zone_id"`
	Domain  types.String `tfsdk:"domain"`
	Records []Record     `tfsdk:"records"`
}

type Record struct {
	RecordId    types.Int64    `tfsdk:"record_id"`
	Entry       types.String   `tfsdk:"entry"`
	Description types.String   `tfsdk:"description"`
	AnswersList []types.String `tfsdk:"answers_list"`
	Policy      types.String   `tfsdk:"policy"`
	RecordType  types.String   `tfsdk:"record_type"`
	Ttl         types.Int64    `tfsdk:"ttl"`
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
			"zone_id": schema.Int64Attribute{
				Required:    true,
				Description: "The zone identifier to target for the resource.",
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
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
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"zone_id": schema.Int64Attribute{
						Description: "The zone identifier to target for the resource.",
						Computed:    true,
					},
					"domain": schema.StringAttribute{
						Computed:    true,
						Description: "Zone name of the found DNS record.",
					},
					"records": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"record_id": schema.Int64Attribute{
									Description: "The record identifier.",
									Computed:    true,
								},
								"entry": schema.StringAttribute{
									Computed:    true,
									Description: "The first part of domain or 'Name'.",
								},
								"description": schema.StringAttribute{
									Computed: true,
								},
								"answers_list": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "List of answers replied by DNS Authoritative to that Record.",
								},
								"policy": schema.StringAttribute{
									Computed:    true,
									Description: "Must be 'simple' or 'weighted'.",
								},
								"record_type": schema.StringAttribute{
									Computed:    true,
									Description: "DNS record type to filter record results on.",
								},
								"ttl": schema.Int64Attribute{
									Computed:    true,
									Description: "Time-to-live defines max-time for packets life in seconds.",
								},
							},
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
	var Page types.Int64
	var PageSize types.Int64
	var getZoneId types.Int64
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

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}

	if PageSize.ValueInt64() == 0 {
		PageSize = types.Int64Value(10)
	}

	diags := req.Config.GetAttribute(ctx, path.Root("zone_id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneId := int32(getZoneId.ValueInt64())

	recordsResponse, httpResp, err := d.client.idnsApi.RecordsAPI.GetZoneRecords(ctx, zoneId).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		d.errorPrint(resp, httpResp.StatusCode)
		return
	}

	var previous, next string
	if recordsResponse.Links != nil {
		if recordsResponse.Links.Previous.Get() != nil {
			previous = *recordsResponse.Links.Previous.Get()
		}
		if recordsResponse.Links.Next.Get() != nil {
			next = *recordsResponse.Links.Next.Get()
		}
	}

	recordsState := RecordsDataSourceModel{
		ZoneId:        getZoneId,
		SchemaVersion: types.Int64Value(int64(*recordsResponse.SchemaVersion)),
		TotalPages:    types.Int64Value(int64(*recordsResponse.TotalPages)),
		Page:          types.Int64Value(Page.ValueInt64()),
		PageSize:      types.Int64Value(PageSize.ValueInt64()),
		Counter:       types.Int64Value(int64(*recordsResponse.Count)),
		Links: &GetRecordsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
		Results: &GetRecordsResponseResults{},
	}
	recordsState.Id = types.StringValue("placeholder")

	if recordsResponse.Results.ZoneId != nil {
		recordsState.Results.ZoneId = types.Int64Value(int64(*recordsResponse.Results.ZoneId))
	}

	if recordsResponse.Results.ZoneDomain != nil {
		recordsState.Results.Domain = types.StringValue(*recordsResponse.Results.ZoneDomain)
	}

	for _, resultRecords := range recordsResponse.Results.Records {
		var r = Record{
			RecordId:    types.Int64Value(int64(*resultRecords.RecordId)),
			Entry:       types.StringValue(*resultRecords.Entry),
			Description: types.StringValue(*resultRecords.Description),
			Policy:      types.StringValue(*resultRecords.Policy),
			RecordType:  types.StringValue(*resultRecords.RecordType),
			Ttl:         types.Int64Value(int64(*resultRecords.Ttl)),
		}

		for _, answer := range resultRecords.AnswersList {
			r.AnswersList = append(r.AnswersList, types.StringValue(answer))
		}

		recordsState.Results.Records = append(recordsState.Results.Records, r)
	}

	diags = resp.State.Set(ctx, &recordsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
