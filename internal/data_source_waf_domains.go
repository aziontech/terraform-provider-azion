package provider

import (
	"context"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &WafDomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &WafDomainsDataSource{}
)

func dataSourceAzionWafDomains() datasource.DataSource {
	return &WafDomainsDataSource{}
}

type WafDomainsDataSource struct {
	client *apiClient
}

type WafDomainsDataSourceModel struct {
	SchemaVersion types.Int64                `tfsdk:"schema_version"`
	ID            types.String               `tfsdk:"id"`
	WafID         types.Int64                `tfsdk:"waf_id"`
	Counter       types.Int64                `tfsdk:"counter"`
	TotalPages    types.Int64                `tfsdk:"total_pages"`
	Page          types.Int64                `tfsdk:"page"`
	PageSize      types.Int64                `tfsdk:"page_size"`
	Links         GetWafDomainsResponseLinks `tfsdk:"links"`
	Results       []WafDomainsResults        `tfsdk:"results"`
}

type GetWafDomainsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type WafDomainsResults struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Domain types.String `tfsdk:"domain"`
	Cnames types.List   `tfsdk:"cnames"`
}

func (o *WafDomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafDomainsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_domains"
}

func (o *WafDomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"counter": schema.Int64Attribute{
				Description: "The total number of Waf domains.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Waf domains.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of Waf domains.",
				Optional:    true,
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
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The WAF identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the WAF Domain configuration.",
							Computed:    true,
						},
						"domain": schema.StringAttribute{
							Description: "WAF mode (e.g., counting).",
							Computed:    true,
						},
						"cnames": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of domains.",
						},
					},
				},
			},
		},
	}
}

func (o *WafDomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var wafID types.Int64

	diagsCacheSettingId := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
	resp.Diagnostics.Append(diagsCacheSettingId...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	if page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	wafDomainsResponse, response, err := o.client.wafApi.WAFAPI.GetWAFDomains(ctx, wafID.ValueInt64()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64()).
		Execute()
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
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
	defer response.Body.Close()

	var WafList []WafDomainsResults
	for _, waf := range wafDomainsResponse.Results {
		var slice []types.String
		for _, Cnames := range waf.Cnames {
			slice = append(slice, types.StringValue(Cnames))
		}
		wafResults := WafDomainsResults{
			ID:     types.Int64Value(waf.GetId()),
			Name:   types.StringValue(waf.GetName()),
			Domain: types.StringValue(waf.GetDomain()),
			Cnames: utils.SliceStringTypeToList(slice),
		}
		WafList = append(WafList, wafResults)
	}

	wafRuleSetState := WafDomainsDataSourceModel{
		SchemaVersion: types.Int64Value(wafDomainsResponse.GetSchemaVersion()),
		WafID:         wafID,
		ID:            types.StringValue("Get All WAF domains"),
		Results:       WafList,
		TotalPages:    types.Int64Value(wafDomainsResponse.GetTotalPages()),
		Page:          page,
		PageSize:      pageSize,
		Counter:       types.Int64Value(wafDomainsResponse.GetCount()),
		Links: GetWafDomainsResponseLinks{
			Previous: types.StringValue(wafDomainsResponse.Links.GetPrevious()),
			Next:     types.StringValue(wafDomainsResponse.Links.GetNext()),
		},
	}

	diags := resp.State.Set(ctx, &wafRuleSetState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
