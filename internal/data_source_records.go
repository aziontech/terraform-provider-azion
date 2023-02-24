package provider

import (
	"context"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &RecordsDataSource{}
	_ datasource.DataSourceWithConfigure = &RecordsDataSource{}
)

func dataSourceAzionRecords() datasource.DataSource {
	return &RecordsDataSource{}
}

type AzionZoneidModel struct {
	ZoneId types.Int64 `tfsdk:"zoneid"`
}

type RecordsDataSource struct {
	client *idns.APIClient
}

type RecordsDataSourceModel struct {
	SchemaVersion types.Int64                `tfsdk:"schema_version"`
	Counter       types.Int64                `tfsdk:"counter"`
	TotalPages    types.Int64                `tfsdk:"total_pages"`
	Links         *GetRecordsResponseLinks   `tfsdk:"links"`
	Results       *GetRecordsResponseResults `tfsdk:"results"`
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
	d.client = req.ProviderData.(*idns.APIClient)
}

func (d *RecordsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_records"
}

func (d *RecordsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"zoneid": schema.Int64Attribute{
				Required: true,
			},
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
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"zone_id": schema.Int64Attribute{
						Computed: true,
					},
					"domain": schema.StringAttribute{
						Computed: true,
					},
					"records": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"record_id": schema.Int64Attribute{
									Computed: true,
								},
								"entry": schema.StringAttribute{
									Computed: true,
								},
								"description": schema.StringAttribute{
									Computed: true,
								},
								"answers_list": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
								},
								"policy": schema.StringAttribute{
									Computed: true,
								},
								"record_type": schema.StringAttribute{
									Computed: true,
								},
								"ttl": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *RecordsDataSource) errorPrint(resp *datasource.ReadResponse, errMsg string) {
	var usrMsg string
	switch errMsg {
	case "404 Not Found":
		usrMsg = "No Records Found"
	case "401 Unauthorized":
		usrMsg = "Unauthorized Token"
	default:
		usrMsg = "Cannot read Azion response"
	}

	resp.Diagnostics.AddError(
		usrMsg,
		errMsg,
	)
}

func (d *RecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading Records")
	//zoneId := 2553 /*TODO read this from config*/

	var config AzionZoneidModel
	diags2 := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	ZoneId := config.ZoneId.ValueInt64()
	zoneId := ZoneId

	recordsResponse, _, err := d.client.RecordsApi.GetZoneRecords(ctx, int32(zoneId)).Execute()
	if err != nil {
		d.errorPrint(resp, err.Error())
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
		SchemaVersion: types.Int64Value(int64(*recordsResponse.SchemaVersion)),
		TotalPages:    types.Int64Value(int64(*recordsResponse.TotalPages)),
		Counter:       types.Int64Value(int64(*recordsResponse.Count)),
		Links: &GetRecordsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
		Results: &GetRecordsResponseResults{},
	}

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

	diags := resp.State.Set(ctx, &recordsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
