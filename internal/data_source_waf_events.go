package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
)

var (
	_ datasource.DataSource              = &WafEventsDataSource{}
	_ datasource.DataSourceWithConfigure = &WafEventsDataSource{}
)

func dataSourceAzionWafEvents() datasource.DataSource {
	return &WafEventsDataSource{}
}

type WafEventsDataSource struct {
	client *apiClient
}

type WafEventsDataSourceModel struct {
	SchemaVersion types.Int64        `tfsdk:"schema_version"`
	ID            types.String       `tfsdk:"id"`
	WafID         types.Int64        `tfsdk:"waf_id"`
	DomainsIDs    []types.Int64      `tfsdk:"domains_ids"`
	NetworkListID types.Int64        `tfsdk:"network_list_id"`
	HourRange     types.Int64        `tfsdk:"hour_range"`
	Results       []WafEventsResults `tfsdk:"results"`
}

type WafEventsResults struct {
	CountryCount    types.Int64  `tfsdk:"country_count"`
	Top10Countries  types.List   `tfsdk:"top_10_countries"`
	Top10Ips        types.List   `tfsdk:"top_10_ips"`
	HitCount        types.Int64  `tfsdk:"hit_count"`
	RuleId          types.Int64  `tfsdk:"rule_id"`
	IpCount         types.Int64  `tfsdk:"ip_count"`
	MatchZone       types.String `tfsdk:"match_zone"`
	PathCount       types.Int64  `tfsdk:"path_count"`
	MatchesOn       types.String `tfsdk:"matches_on"`
	RuleDescription types.String `tfsdk:"rule_description"`
}

func (o *WafEventsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafEventsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_events"
}

func (o *WafEventsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"waf_id": schema.Int64Attribute{
				Description: "The WAF identifier.",
				Required:    true,
			},
			"domains_ids": schema.ListAttribute{
				ElementType: types.Int64Type,
				Description: "The total number of Waf.",
				Optional:    true,
			},
			"network_list_id": schema.Int64Attribute{
				Description: "The total number of Waf.",
				Optional:    true,
			},
			"hour_range": schema.Int64Attribute{
				Description: "The total number of Waf.",
				Optional:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"country_count": schema.Int64Attribute{
							Description: "The is the Country Count.",
							Computed:    true,
						},
						"top_10_countries": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "The is the Country Count.",
						},
						"top_10_ips": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "The is the Country Count.",
						},
						"hit_count": schema.Int64Attribute{
							Description: "The is the Hit Count.",
							Computed:    true,
						},
						"rule_id": schema.Int64Attribute{
							Description: "Rule id off WAF.",
							Computed:    true,
						},
						"ip_count": schema.Int64Attribute{
							Description: "The is the IP Count.",
							Computed:    true,
						},
						"match_zone": schema.StringAttribute{
							Description: "The is the Match Zone.",
							Computed:    true,
						},
						"path_count": schema.Int64Attribute{
							Description: "The is the Path Count.",
							Computed:    true,
						},
						"matches_on": schema.StringAttribute{
							Description: "The is the Matches On.",
							Computed:    true,
						},
						"rule_description": schema.StringAttribute{
							Description: "The is the Rule Description.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (o *WafEventsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var NetworkListID types.Int64
	var HourRange types.Int64
	var wafID types.Int64
	var domainIDs []types.Int64

	diags := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
	resp.Diagnostics.Append(diags...)

	diags = req.Config.GetAttribute(ctx, path.Root("network_list_id"), &NetworkListID)
	resp.Diagnostics.Append(diags...)

	diags = req.Config.GetAttribute(ctx, path.Root("domains_ids"), &domainIDs)
	resp.Diagnostics.Append(diags...)

	diags = req.Config.GetAttribute(ctx, path.Root("hour_range"), &HourRange)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ApiGetWAFEventsRequest := o.client.wafApi.WAFAPI.GetWAFEvents(ctx, wafID.ValueInt64())

	if HourRange.ValueInt64() != 0 {
		ApiGetWAFEventsRequest = ApiGetWAFEventsRequest.HourRange(HourRange.ValueInt64())
	}
	if NetworkListID.ValueInt64() != 0 {
		ApiGetWAFEventsRequest = ApiGetWAFEventsRequest.NetworkListId(NetworkListID.ValueInt64())
	}
	if len(domainIDs) != 0 {
		var slice []int64
		for _, domain := range domainIDs {
			slice = append(slice, domain.ValueInt64())
		}
		ApiGetWAFEventsRequest = ApiGetWAFEventsRequest.DomainsIds(slice)
	}

	wafDomainsResponse, response, err := ApiGetWAFEventsRequest.Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	var WafList []WafEventsResults
	for _, waf := range wafDomainsResponse.Results {
		//var slice []types.String
		//for _, top10Countries := range waf.Top10Countries {
		//	slice = append(slice, types.StringValue(Cnames))
		//}
		wafResults := WafEventsResults{
			CountryCount: types.Int64Value(waf.GetCountryCount()),
			//Top10Countries: types.ListValueMust(types.StringType, slice),
			//Top10Ips: types.ListValueMust(types.StringType, utils.StringSliceToTypesList(waf.GetTop10Ips())),
			HitCount:        types.Int64Value(waf.GetHitCount()),
			RuleId:          types.Int64Value(waf.GetRuleId()),
			IpCount:         types.Int64Value(waf.GetIpCount()),
			MatchZone:       types.StringValue(waf.GetMatchZone()),
			PathCount:       types.Int64Value(waf.GetPathCount()),
			MatchesOn:       types.StringValue(waf.GetMatchesOn()),
			RuleDescription: types.StringValue(waf.GetRuleDescription()),
		}
		WafList = append(WafList, wafResults)
	}

	wafRuleSetState := WafEventsDataSourceModel{
		SchemaVersion: types.Int64Value(wafDomainsResponse.GetSchemaVersion()),
		Results:       WafList,
		WafID:         wafID,
		HourRange:     HourRange,
		NetworkListID: NetworkListID,
		DomainsIDs:    domainIDs,
	}

	wafRuleSetState.ID = types.StringValue("Get All Waf Events")
	diags = resp.State.Set(ctx, &wafDomainsResponse)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
