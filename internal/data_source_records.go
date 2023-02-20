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
	_ datasource.DataSource              = &RecordsDataSource{}
	_ datasource.DataSourceWithConfigure = &RecordsDataSource{}
)

func dataSourceAzionRecords() datasource.DataSource {
	return &RecordsDataSource{}
}

type RecordsDataSource struct {
	client *idns.APIClient
}

type RecordsDataSourceModel struct {
	SchemaVersion types.Int64                `tfsdk:"schema_version"`
	Counter       types.Int64                `tfsdk:"counter"`
	TotalPages    types.Int64                `tfsdk:"total_pages"`
	Links         *GetZonesResponseLinks     `tfsdk:"links"`
	Results       *GetRecordsResponseResults `tfsdk:"results"`
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

func (d *RecordsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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
									Computed: true,
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

func (d *RecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Records"))
	zoneId := 2468 /*TODO read this from config*/
	recordsResponse, _, err := d.client.RecordsApi.GetZoneRecords(ctx, int32(zoneId)).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Azion Records",
			err.Error(),
		)
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
		Links: &GetZonesResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
		Results: &GetRecordsResponseResults{
			ZoneId: types.Int64Value(int64(*recordsResponse.Results.ZoneId)),
			Domain: types.StringValue(*recordsResponse.Results.Domain),
		},
	}

	for _, resultRecords := range recordsResponse.Results.Records {
		var r = Record{
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
