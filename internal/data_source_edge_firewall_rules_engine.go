package provider

import (
	"context"
	"io"

	"github.com/aziontech/azionapi-go-sdk/edgefirewall"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ datasource.DataSource              = &EdgeFirewallRulesEngineDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeFirewallRulesEngineDataSource{}
)

func dataSourceAzionEdgeFirewallRulesEngine() datasource.DataSource {
	return &EdgeFirewallRulesEngineDataSource{}
}

type EdgeFirewallRulesEngineDataSource struct {
	client *apiClient
}

type EdgeFirewallRulesEngineDataSourceModel struct {
	ID             types.String                     `tfsdk:"id"`
	EdgeFirewallID types.Int64                      `tfsdk:"edge_firewall_id"`
	Counter        types.Int64                      `tfsdk:"counter"`
	TotalPages     types.Int64                      `tfsdk:"total_pages"`
	Page           types.Int64                      `tfsdk:"page"`
	PageSize       types.Int64                      `tfsdk:"page_size"`
	SchemaVersion  types.Int64                      `tfsdk:"schema_version"`
	Links          EdgeFirewallsResponseLinks       `tfsdk:"links"`
	Results        []EdgeFirewallRulesEngineResults `tfsdk:"results"`
}

type EdgeFirewallRulesEngineResults struct {
	ID           types.Int64                            `tfsdk:"id"`
	LastEditor   types.String                           `tfsdk:"last_editor"`
	LastModified types.String                           `tfsdk:"last_modified"`
	Name         types.String                           `tfsdk:"name"`
	IsActive     types.Bool                             `tfsdk:"is_active"`
	Description  types.String                           `tfsdk:"description"`
	Behaviors    []EdgeFirewallRulesEngineBehaviorModel `tfsdk:"behaviors"`
	Criteria     []EdgeFirewallRulesEngineCriteriaModel `tfsdk:"criteria"`
	Order        types.Int64                            `tfsdk:"order"`
}

type EdgeFirewallRulesEngineBehaviorModel struct {
	Name        types.String                                 `tfsdk:"name"`
	Argument    EdgeFirewallRulesEngineBehaviorArgumentModel `tfsdk:"argument"`
	ArgumentInt types.Int64                                  `tfsdk:"argument_int"`
	ArgumentStr types.String                                 `tfsdk:"argument_str"`
	ArgumentNum types.Float64                                `tfsdk:"argument_num"`
}

type EdgeFirewallRulesEngineBehaviorArgumentModel struct {
	Type                    types.String `tfsdk:"type"`
	LimitBy                 types.String `tfsdk:"limit_by"`
	AverageRateLimitInt     types.Int64  `tfsdk:"average_rate_limit_int"`
	AverageRateLimitStr     types.String `tfsdk:"average_rate_limit_str"`
	MaximumBurstSizeInt     types.Int64  `tfsdk:"maximum_burst_size_int"`
	MaximumBurstSizeStr     types.String `tfsdk:"maximum_burst_size_str"`
	WafID                   types.Int64  `tfsdk:"waf_id"`
	Mode                    types.String `tfsdk:"mode"`
	SetWafRulesetAndWafMode types.Int64  `tfsdk:"set_waf_ruleset_and_waf_mode"`
	WafMode                 types.String `tfsdk:"waf_mode"`
	StatusCodeStr           types.String `tfsdk:"status_code_str"`
	StatusCodeInt           types.Int64  `tfsdk:"status_code_int"`
	ContentType             types.String `tfsdk:"content_type"`
	ContentBody             types.String `tfsdk:"content_body"`
}

type EdgeFirewallRulesEngineCriteriaModel struct {
	Entries []EdgeFirewallRulesEngineCriteriaEntryModel `tfsdk:"entries"`
}

type EdgeFirewallRulesEngineCriteriaEntryModel struct {
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Conditional types.String `tfsdk:"conditional"`
	InputValue  types.String `tfsdk:"input_value"`
}

func (ds *EdgeFirewallRulesEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.client = req.ProviderData.(*apiClient)
}

func (ds *EdgeFirewallRulesEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_rules_engine"
}

func (ds *EdgeFirewallRulesEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			// Parameters and querystrings
			"edge_firewall_id": schema.Int64Attribute{
				Description: "The edge firewall identifier.",
				Required:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of edge firewalls.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of edge firewalls.",
				Optional:    true,
			},
			// Response
			"counter": schema.Int64Attribute{
				Description: "The total number of edge firewalls.",
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
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the edge firewall rule set.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the edge firewall rule set.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the edge firewall rule set.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the edge firewall rule set.",
							Computed:    true,
						},
						"is_active": schema.BoolAttribute{
							Description: "Whether the edge firewall rule set is active.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the edge firewall rule set.",
							Computed:    true,
						},
						"behaviors": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "The name of the behavior.",
										Computed:    true,
									},
									"argument": schema.SingleNestedAttribute{
										Optional: true,
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Description: "Unit type of the rate limit.",
												Optional:    true,
											},
											"limit_by": schema.StringAttribute{
												Description: "Scope of the rate limit.",
												Optional:    true,
											},
											"average_rate_limit_int": schema.Int64Attribute{
												Description: "Average rate limit value (as integer) to use as threshold value.",
												Optional:    true,
											},
											"average_rate_limit_str": schema.StringAttribute{
												Description: "Average rate limit value (as string) to use as threshold value.",
												Optional:    true,
											},
											"maximum_burst_size_int": schema.Int64Attribute{
												Description: "Maximum burst size value (as integer) to use as threshold value.",
												Optional:    true,
											},
											"maximum_burst_size_str": schema.StringAttribute{
												Description: "Maximum burst size value (as string) to use as threshold value.",
												Optional:    true,
											},
											"waf_id": schema.Int64Attribute{
												Description: "ID of the related WAF.",
												Optional:    true,
											},
											"mode": schema.StringAttribute{
												Description: "Mode of the WAF rule.",
												Optional:    true,
											},
											"set_waf_ruleset_and_waf_mode": schema.Int64Attribute{
												Description: "ID of the related WAF ruleset.",
												Optional:    true,
											},
											"waf_mode": schema.StringAttribute{
												Description: "Mode of the WAF rule.",
												Optional:    true,
											},
											"status_code_str": schema.StringAttribute{
												Description: "Status code (as string) to be returned if the rule is match.",
												Optional:    true,
											},
											"status_code_int": schema.Int64Attribute{
												Description: "Status code (as integer) to be returned if the rule is match.",
												Optional:    true,
											},
											"content_type": schema.StringAttribute{
												Description: "Content type of the response to be returned if the rule is match.",
												Optional:    true,
											},
											"content_body": schema.StringAttribute{
												Description: "Content body of the response to be returned if the rule is match.",
												Optional:    true,
											},
										},
									},
								},
							},
						},
						"criteria": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"entries": schema.ListNestedAttribute{
										Computed: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"variable": schema.StringAttribute{
													Description: "The variable used in the edge firewall rule's criteria.",
													Computed:    true,
												},
												"operator": schema.StringAttribute{
													Description: "The operator used in the edge firewall rule's criteria.",
													Computed:    true,
												},
												"conditional": schema.StringAttribute{
													Description: "The conditional operator used in the edge firewall rule's criteria (e.g., if, and, or).",
													Computed:    true,
												},
												"input_value": schema.StringAttribute{
													Description: "The input value used in the edge firewall rule's criteria.",
													Computed:    true,
												},
											},
										},
									},
								},
							},
						},
						"order": schema.Int64Attribute{
							Description: "The execution order of the edge firewallrule set",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (ds *EdgeFirewallRulesEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeFirewallID types.Int64
	var page types.Int64
	var pageSize types.Int64

	diagsEdgeFirewallID := req.Config.GetAttribute(ctx, path.Root("edge_firewall_id"), &edgeFirewallID)
	resp.Diagnostics.Append(diagsEdgeFirewallID...)

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diagsPage...)
	if page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	rulesEngineResponse, response, err := ds.client.edgeFirewallApi.DefaultAPI.
		EdgeFirewallEdgeFirewallIdRulesEngineGet(ctx, edgeFirewallID.ValueInt64()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64()).
		Execute()
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

	rulesEngineResults := []EdgeFirewallRulesEngineResults{}
	for _, result := range rulesEngineResponse.Results {
		behaviors := []EdgeFirewallRulesEngineBehaviorModel{}
		for _, behavior := range result.Behaviors {
			// edgefirewall.NullArgumentBehavior
			if b, ok := behavior.GetActualInstance().(edgefirewall.NullArgumentBehavior); ok {
				newBehavior := EdgeFirewallRulesEngineBehaviorModel{
					Name: types.StringValue(b.GetName()),
				}
				if arg, ok := b.GetArgumentOk(); ok {
					newBehavior.ArgumentInt = types.Int64Value(int64(*arg))
				}
				behaviors = append(behaviors, newBehavior)
			}
		}

		criterias := []EdgeFirewallRulesEngineCriteriaModel{}
		for _, criteriaList := range result.Criteria {
			var newCriteria EdgeFirewallRulesEngineCriteriaModel
			for _, criteria := range criteriaList {
				newEntry := EdgeFirewallRulesEngineCriteriaEntryModel{
					Variable:    basetypes.NewStringValue(string(criteria.GetVariable())),
					Operator:    basetypes.NewStringValue(string(criteria.GetOperator())),
					Conditional: basetypes.NewStringValue(string(criteria.GetConditional())),
				}
				if _, ok := criteria.GetInputValueOk(); ok {
					newEntry.InputValue = basetypes.NewStringValue(string(criteria.GetInputValue()))
				}
				newCriteria.Entries = append(newCriteria.Entries, newEntry)
			}
			criterias = append(criterias, newCriteria)
		}

		rulesEngineResults = append(rulesEngineResults, EdgeFirewallRulesEngineResults{
			ID:           types.Int64Value(result.GetId()),
			LastEditor:   types.StringValue(result.GetLastEditor()),
			LastModified: types.StringValue(result.GetLastModified().String()),
			Name:         types.StringValue(result.GetName()),
			IsActive:     types.BoolValue(result.GetIsActive()),
			Description:  types.StringValue(result.GetDescription()),
			Behaviors:    behaviors,
			Criteria:     criterias,
			Order:        types.Int64Value(int64(result.GetOrder())),
		})
	}

	state := EdgeFirewallRulesEngineDataSourceModel{
		SchemaVersion:  types.Int64Value(int64(rulesEngineResponse.GetSchemaVersion())),
		EdgeFirewallID: edgeFirewallID,
		Counter:        types.Int64Value(rulesEngineResponse.GetCount()),
		TotalPages:     types.Int64Value(rulesEngineResponse.GetTotalPages()),
		Page:           page,
		PageSize:       pageSize,
		Links: EdgeFirewallsResponseLinks{
			Previous: types.StringPointerValue(rulesEngineResponse.Links.Previous.Get()),
			Next:     types.StringPointerValue(rulesEngineResponse.Links.Next.Get()),
		},
		Results: rulesEngineResults,
	}

	state.ID = types.StringValue("Get All Edge Firewall Rules Engine")
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
