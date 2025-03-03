package provider

import (
	"context"
	"io"

	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &WafDataSource{}
	_ datasource.DataSourceWithConfigure = &WafDataSource{}
)

func dataSourceAzionWafRuleSets() datasource.DataSource {
	return &WafDataSource{}
}

type WafDataSource struct {
	client *apiClient
}

type WafDataSourceModel struct {
	SchemaVersion types.Int64          `tfsdk:"schema_version"`
	ID            types.String         `tfsdk:"id"`
	Counter       types.Int64          `tfsdk:"counter"`
	TotalPages    types.Int64          `tfsdk:"total_pages"`
	Page          types.Int64          `tfsdk:"page"`
	PageSize      types.Int64          `tfsdk:"page_size"`
	Links         *GetWafResponseLinks `tfsdk:"links"`
	Results       []WafResults         `tfsdk:"results"`
}

type GetWafResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type WafResults struct {
	ID                             types.Int64    `tfsdk:"waf_id"`
	Name                           types.String   `tfsdk:"name"`
	Mode                           types.String   `tfsdk:"mode"`
	Active                         types.Bool     `tfsdk:"active"`
	SQLInjection                   types.Bool     `tfsdk:"sql_injection"`
	SQLInjectionSensitivity        types.String   `tfsdk:"sql_injection_sensitivity"`
	RemoteFileInclusion            types.Bool     `tfsdk:"remote_file_inclusion"`
	RemoteFileInclusionSensitivity types.String   `tfsdk:"remote_file_inclusion_sensitivity"`
	DirectoryTraversal             types.Bool     `tfsdk:"directory_traversal"`
	DirectoryTraversalSensitivity  types.String   `tfsdk:"directory_traversal_sensitivity"`
	CrossSiteScripting             types.Bool     `tfsdk:"cross_site_scripting"`
	CrossSiteScriptingSensitivity  types.String   `tfsdk:"cross_site_scripting_sensitivity"`
	EvadingTricks                  types.Bool     `tfsdk:"evading_tricks"`
	EvadingTricksSensitivity       types.String   `tfsdk:"evading_tricks_sensitivity"`
	FileUpload                     types.Bool     `tfsdk:"file_upload"`
	FileUploadSensitivity          types.String   `tfsdk:"file_upload_sensitivity"`
	UnwantedAccess                 types.Bool     `tfsdk:"unwanted_access"`
	UnwantedAccessSensitivity      types.String   `tfsdk:"unwanted_access_sensitivity"`
	IdentifiedAttack               types.Bool     `tfsdk:"identified_attack"`
	IdentifiedAttackSensitivity    types.String   `tfsdk:"identified_attack_sensitivity"`
	BypassAddresses                []types.String `tfsdk:"bypass_addresses"`
}

func (o *WafDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_sets"
}

func (o *WafDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of Waf.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Waf.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of Waf.",
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
						"waf_id": schema.Int64Attribute{
							Description: "The WAF identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the WAF configuration.",
							Computed:    true,
						},
						"mode": schema.StringAttribute{
							Description: "WAF mode (e.g., counting).",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the WAF is active.",
							Computed:    true,
						},
						"sql_injection": schema.BoolAttribute{
							Description: "Enable SQL injection protection.",
							Computed:    true,
						},
						"sql_injection_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for SQL injection protection.",
							Computed:    true,
						},
						"remote_file_inclusion": schema.BoolAttribute{
							Description: "Enable remote file inclusion protection.",
							Computed:    true,
						},
						"remote_file_inclusion_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for remote file inclusion protection.",
							Computed:    true,
						},
						"directory_traversal": schema.BoolAttribute{
							Description: "Enable directory traversal protection.",
							Computed:    true,
						},
						"directory_traversal_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for directory traversal protection.",
							Computed:    true,
						},
						"cross_site_scripting": schema.BoolAttribute{
							Description: "Enable cross-site scripting protection.",
							Computed:    true,
						},
						"cross_site_scripting_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for cross-site scripting protection.",
							Computed:    true,
						},
						"evading_tricks": schema.BoolAttribute{
							Description: "Enable evading tricks protection.",
							Computed:    true,
						},
						"evading_tricks_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for evading tricks protection.",
							Computed:    true,
						},
						"file_upload": schema.BoolAttribute{
							Description: "Enable file upload protection.",
							Computed:    true,
						},
						"file_upload_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for file upload protection.",
							Computed:    true,
						},
						"unwanted_access": schema.BoolAttribute{
							Description: "Enable protection against unwanted access.",
							Computed:    true,
						},
						"unwanted_access_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for protection against unwanted access.",
							Computed:    true,
						},
						"identified_attack": schema.BoolAttribute{
							Description: "Enable protection against identified attacks.",
							Computed:    true,
						},
						"identified_attack_sensitivity": schema.StringAttribute{
							Description: "Sensitivity level for protection against identified attacks.",
							Computed:    true,
						},
						"bypass_addresses": schema.ListAttribute{
							Description: "List of bypass addresses.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (o *WafDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}
	if PageSize.ValueInt64() == 0 {
		PageSize = types.Int64Value(10)
	}

	wafResponse, response, err := o.client.wafApi.WAFAPI.ListAllWAFRulesets(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			err := utils.SleepAfter429(response)
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
				)
				return
			}
			wafResponse, _, err = o.client.wafApi.WAFAPI.ListAllWAFRulesets(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
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

	var previous, next string
	if wafResponse.GetLinks().Previous.Get() != nil {
		previous = *wafResponse.Links.Previous.Get()
	}
	if wafResponse.GetLinks().Next.Get() != nil {
		next = *wafResponse.Links.Next.Get()
	}

	var WafList []WafResults
	for _, waf := range wafResponse.Results {
		var sliceAddresses []types.String
		for _, Addresses := range waf.GetBypassAddresses() {
			sliceAddresses = append(sliceAddresses, types.StringValue(Addresses))
		}

		wafResults := WafResults{
			ID:                             types.Int64Value(waf.GetId()),
			Name:                           types.StringValue(waf.GetName()),
			Mode:                           types.StringValue(waf.GetMode()),
			Active:                         types.BoolValue(waf.GetActive()),
			BypassAddresses:                sliceAddresses,
			SQLInjection:                   types.BoolValue(waf.GetSqlInjection()),
			SQLInjectionSensitivity:        types.StringValue(string(waf.GetSqlInjectionSensitivity())),
			RemoteFileInclusion:            types.BoolValue(waf.GetRemoteFileInclusion()),
			RemoteFileInclusionSensitivity: types.StringValue(string(waf.GetRemoteFileInclusionSensitivity())),
			DirectoryTraversal:             types.BoolValue(waf.GetDirectoryTraversal()),
			DirectoryTraversalSensitivity:  types.StringValue(string(waf.GetDirectoryTraversalSensitivity())),
			CrossSiteScripting:             types.BoolValue(waf.GetCrossSiteScripting()),
			CrossSiteScriptingSensitivity:  types.StringValue(string(waf.GetCrossSiteScriptingSensitivity())),
			EvadingTricks:                  types.BoolValue(waf.GetEvadingTricks()),
			EvadingTricksSensitivity:       types.StringValue(string(waf.GetEvadingTricksSensitivity())),
			FileUpload:                     types.BoolValue(waf.GetFileUpload()),
			FileUploadSensitivity:          types.StringValue(string(waf.GetFileUploadSensitivity())),
			UnwantedAccess:                 types.BoolValue(waf.GetUnwantedAccess()),
			UnwantedAccessSensitivity:      types.StringValue(string(waf.GetUnwantedAccessSensitivity())),
			IdentifiedAttack:               types.BoolValue(waf.GetIdentifiedAttack()),
			IdentifiedAttackSensitivity:    types.StringValue(string(waf.GetIdentifiedAttackSensitivity())),
		}
		WafList = append(WafList, wafResults)
	}

	wafRuleSetState := WafDataSourceModel{
		SchemaVersion: types.Int64Value(int64(wafResponse.GetSchemaVersion())),
		Results:       WafList,
		TotalPages:    types.Int64Value(int64(wafResponse.GetTotalPages())),
		Page:          Page,
		PageSize:      PageSize,
		Counter:       types.Int64Value(int64(wafResponse.GetCount())),
		Links: &GetWafResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	wafRuleSetState.ID = types.StringValue("Get All Waf Rules Set")
	diags := resp.State.Set(ctx, &wafRuleSetState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
