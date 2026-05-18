package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &FirewallRulesEngineDataSource{}
	_ datasource.DataSourceWithConfigure = &FirewallRulesEngineDataSource{}
)

func dataSourceAzionFirewallRulesEngine() datasource.DataSource {
	return &FirewallRulesEngineDataSource{}
}

type FirewallRulesEngineDataSource struct {
	client *apiClient
}

type FirewallRulesEngineDataSourceModel struct {
	ID         types.String                     `tfsdk:"id"`
	FirewallID types.Int64                      `tfsdk:"firewall_id"`
	Counter    types.Int64                      `tfsdk:"counter"`
	TotalPages types.Int64                      `tfsdk:"total_pages"`
	Page       types.Int64                      `tfsdk:"page"`
	PageSize   types.Int64                      `tfsdk:"page_size"`
	Links      *LinksModel                      `tfsdk:"links"`
	Results    []FirewallRulesEngineResultModel `tfsdk:"results"`
}

type FirewallRulesEngineResultModel struct {
	ID           types.Int64                        `tfsdk:"id"`
	Name         types.String                       `tfsdk:"name"`
	Active       types.Bool                         `tfsdk:"active"`
	Criteria     []FirewallCriteriaDataModel        `tfsdk:"criteria"`
	Behaviors    []FirewallBehaviorWrapperDataModel `tfsdk:"behaviors"`
	Description  types.String                       `tfsdk:"description"`
	Order        types.Int64                        `tfsdk:"order"`
	LastEditor   types.String                       `tfsdk:"last_editor"`
	LastModified types.String                       `tfsdk:"last_modified"`
	CreatedAt    types.String                       `tfsdk:"created_at"`
}

func (r *FirewallRulesEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *FirewallRulesEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rules_engine"
}

func (r *FirewallRulesEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall identifier.",
				Required:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of rules.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size.",
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
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the rules engine rule.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the rules engine rule.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the rule is active.",
							Computed:    true,
						},
						"criteria": schema.ListNestedAttribute{
							Description: "Criteria for the rule.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"entries": schema.ListNestedAttribute{
										Description: "Criteria entries.",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"criterion": schema.SingleNestedAttribute{
													Description: "A single criterion entry.",
													Computed:    true,
													Attributes: map[string]schema.Attribute{
														"conditional": schema.StringAttribute{
															Description: "Conditional operator (if, and, or).",
															Computed:    true,
														},
														"variable": schema.StringAttribute{
															Description: "Variable to evaluate.",
															Computed:    true,
														},
														"operator": schema.StringAttribute{
															Description: "Comparison operator.",
															Computed:    true,
														},
														"argument": schema.StringAttribute{
															Description: "Argument for comparison.",
															Computed:    true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"behaviors": schema.ListNestedAttribute{
							Description: "Behaviors for the rule.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"behavior": schema.SingleNestedAttribute{
										Description: "A single behavior for the rule.",
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"type": schema.StringAttribute{
												Description: "Type of behavior.",
												Computed:    true,
											},
											"attributes": schema.SingleNestedAttribute{
												Description: "Behavior attributes.",
												Computed:    true,
												Attributes: map[string]schema.Attribute{
													"value": schema.Int64Attribute{
														Description: "Value for run_function behavior (function instance ID).",
														Computed:    true,
													},
													"status_code": schema.Int64Attribute{
														Description: "Status code for set_custom_response behavior.",
														Computed:    true,
													},
													"content_type": schema.StringAttribute{
														Description: "Content type for set_custom_response behavior.",
														Computed:    true,
													},
													"content_body": schema.StringAttribute{
														Description: "Content body for set_custom_response behavior.",
														Computed:    true,
													},
													"waf_id": schema.Int64Attribute{
														Description: "WAF ID for set_waf behavior.",
														Computed:    true,
													},
													"mode": schema.StringAttribute{
														Description: "Mode for set_waf behavior.",
														Computed:    true,
													},
													"type": schema.StringAttribute{
														Description: "Type for set_rate_limit behavior.",
														Computed:    true,
													},
													"limit_by": schema.StringAttribute{
														Description: "Limit by for set_rate_limit behavior.",
														Computed:    true,
													},
													"average_rate_limit": schema.Int64Attribute{
														Description: "Average rate limit for set_rate_limit behavior.",
														Computed:    true,
													},
													"maximum_burst_size": schema.Int64Attribute{
														Description: "Maximum burst size for set_rate_limit behavior.",
														Computed:    true,
													},
												},
											},
										},
									},
								},
							},
						},
						"description": schema.StringAttribute{
							Description: "Description of the rule.",
							Computed:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Order of the rule.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the rule.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the firewall rule.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *FirewallRulesEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var firewallID types.Int64
	var page types.Int64
	var pageSize types.Int64

	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
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

	// Set defaults
	if page.IsNull() || page.IsUnknown() {
		page = types.Int64Value(1)
	}
	if pageSize.IsNull() || pageSize.IsUnknown() {
		pageSize = types.Int64Value(10)
	}

	result, response, err := r.listRules(ctx, firewallID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64())
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			result, response, err = r.listRulesWithRetry(ctx, firewallID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64())

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else if response != nil {
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(errReadAll.Error(), "err")
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(err.Error(), bodyString)
			response.Body.Close()
			return
		} else {
			resp.Diagnostics.AddError(err.Error(), "API request failed")
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	result.FirewallID = firewallID
	result.ID = types.StringValue("Get All Firewall Rules Engine")

	diags := resp.State.Set(ctx, &result)
	resp.Diagnostics.Append(diags...)
}

func (r *FirewallRulesEngineDataSource) listRules(ctx context.Context, firewallID, page, pageSize int64) (FirewallRulesEngineDataSourceModel, *http.Response, error) {
	listReq := r.client.api.FirewallsRulesEngineAPI.
		ListFirewallRules(ctx, firewallID).
		Page(page).
		PageSize(pageSize)

	listResponse, response, err := listReq.Execute()
	if err != nil {
		return FirewallRulesEngineDataSourceModel{}, response, err
	}

	return transformPaginatedFirewallRuleList(listResponse), response, nil
}

func (r *FirewallRulesEngineDataSource) listRulesWithRetry(ctx context.Context, firewallID, page, pageSize int64) (FirewallRulesEngineDataSourceModel, *http.Response, error) {
	listResponse, response, err := utils.RetryOn429(func() (*azionapi.PaginatedFirewallRuleList, *http.Response, error) {
		return r.client.api.FirewallsRulesEngineAPI.
			ListFirewallRules(ctx, firewallID).
			Page(page).
			PageSize(pageSize).
			Execute()
	}, 5)
	if err != nil {
		return FirewallRulesEngineDataSourceModel{}, response, err
	}

	return transformPaginatedFirewallRuleList(listResponse), response, nil
}

func transformPaginatedFirewallRuleList(list *azionapi.PaginatedFirewallRuleList) FirewallRulesEngineDataSourceModel {
	result := FirewallRulesEngineDataSourceModel{
		Page:    types.Int64Value(int64(list.GetPage())),
		Counter: types.Int64Value(list.GetCount()),
	}

	if list.TotalPages != nil {
		result.TotalPages = types.Int64Value(*list.TotalPages)
	}
	if list.PageSize != nil {
		result.PageSize = types.Int64Value(*list.PageSize)
	}

	// Transform links
	var previous, next string
	if list.Previous.Get() != nil {
		previous = *list.Previous.Get()
	}
	if list.Next.Get() != nil {
		next = *list.Next.Get()
	}
	result.Links = &LinksModel{
		Previous: types.StringValue(previous),
		Next:     types.StringValue(next),
	}

	// Transform results
	for _, rule := range list.Results {
		result.Results = append(result.Results, transformFirewallRuleToListResult(rule))
	}

	return result
}

func transformFirewallRuleToListResult(rule azionapi.FirewallRule) FirewallRulesEngineResultModel {
	result := FirewallRulesEngineResultModel{
		ID:    types.Int64Value(rule.GetId()),
		Name:  types.StringValue(rule.GetName()),
		Order: types.Int64Value(rule.GetOrder()),
	}

	if rule.Active != nil {
		result.Active = types.BoolValue(*rule.Active)
	}
	if rule.Description != nil {
		result.Description = types.StringValue(*rule.Description)
	}
	result.LastEditor = types.StringValue(rule.GetLastEditor())
	result.LastModified = types.StringValue(rule.GetLastModified().Format(time.RFC3339))
	result.CreatedAt = types.StringValue(rule.GetCreatedAt().Format(time.RFC3339))

	// Transform criteria.
	for _, criterionGroup := range rule.Criteria {
		var entries []FirewallCriterionWrapperDataModel
		for _, c := range criterionGroup {
			arg := ""
			if c.Argument.Get() != nil {
				arg = fmt.Sprintf("%v", c.Argument.Get())
			}
			entries = append(entries, FirewallCriterionWrapperDataModel{
				Criterion: &FirewallCriteriaEntryDataModel{
					Conditional: types.StringValue(c.GetConditional()),
					Variable:    types.StringValue(c.GetVariable()),
					Operator:    types.StringValue(c.GetOperator()),
					Argument:    types.StringValue(arg),
				},
			})
		}
		result.Criteria = append(result.Criteria, FirewallCriteriaDataModel{
			Entries: entries,
		})
	}

	// Transform behaviors.
	for _, b := range rule.Behaviors {
		behavior := FirewallBehaviorDataModel{}

		if b.FirewallBehaviorArgs != nil {
			behavior.Type = types.StringValue(b.FirewallBehaviorArgs.GetType())
			attrs := transformFirewallBehaviorArgsToDataModel(b.FirewallBehaviorArgs.Attributes)
			behavior.Attributes = &attrs
		} else if b.FirewallBehaviorNoArgs != nil {
			behavior.Type = types.StringValue(b.FirewallBehaviorNoArgs.GetType())
		} else if b.FirewallBehaviorObjectArgs != nil {
			behavior.Type = types.StringValue(b.FirewallBehaviorObjectArgs.GetType())
			attrs := transformFirewallBehaviorObjectAttrsToDataModel(b.FirewallBehaviorObjectArgs.Attributes)
			behavior.Attributes = &attrs
		}
		result.Behaviors = append(result.Behaviors, FirewallBehaviorWrapperDataModel{
			Behavior: &behavior,
		})
	}

	return result
}
