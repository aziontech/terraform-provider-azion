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
	_ datasource.DataSource              = &FirewallRuleEngineDataSource{}
	_ datasource.DataSourceWithConfigure = &FirewallRuleEngineDataSource{}
)

func dataSourceAzionFirewallRuleEngine() datasource.DataSource {
	return &FirewallRuleEngineDataSource{}
}

type FirewallRuleEngineDataSource struct {
	client *apiClient
}

type FirewallRuleEngineDataSourceModel struct {
	ID         types.String                       `tfsdk:"id"`
	FirewallID types.Int64                        `tfsdk:"firewall_id"`
	Results    *FirewallRuleEngineResultDataModel `tfsdk:"results"`
}

type FirewallRuleEngineResultDataModel struct {
	ID           types.Int64                 `tfsdk:"id"`
	Name         types.String                `tfsdk:"name"`
	Active       types.Bool                  `tfsdk:"active"`
	Criteria     []FirewallCriteriaDataModel `tfsdk:"criteria"`
	Behaviors    []FirewallBehaviorDataModel `tfsdk:"behaviors"`
	Description  types.String                `tfsdk:"description"`
	Order        types.Int64                 `tfsdk:"order"`
	LastEditor   types.String                `tfsdk:"last_editor"`
	LastModified types.String                `tfsdk:"last_modified"`
	CreatedAt    types.String                `tfsdk:"created_at"`
}

type FirewallCriteriaDataModel struct {
	Entries []FirewallCriteriaEntryDataModel `tfsdk:"entries"`
}

type FirewallCriteriaEntryDataModel struct {
	Conditional types.String `tfsdk:"conditional"`
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Argument    types.String `tfsdk:"argument"`
}

type FirewallBehaviorDataModel struct {
	Type       types.String                    `tfsdk:"type"`
	Attributes *FirewallBehaviorAttrsDataModel `tfsdk:"attributes"`
}

type FirewallBehaviorAttrsDataModel struct {
	// For run_function behavior
	Value types.Int64 `tfsdk:"value"`
	// For set_custom_response behavior
	StatusCode  types.Int64  `tfsdk:"status_code"`
	ContentType types.String `tfsdk:"content_type"`
	ContentBody types.String `tfsdk:"content_body"`
	// For set_waf behavior
	WafId types.Int64  `tfsdk:"waf_id"`
	Mode  types.String `tfsdk:"mode"`
	// For set_rate_limit behavior
	Type             types.String `tfsdk:"type"`
	LimitBy          types.String `tfsdk:"limit_by"`
	AverageRateLimit types.Int64  `tfsdk:"average_rate_limit"`
	MaximumBurstSize types.Int64  `tfsdk:"maximum_burst_size"`
}

func (r *FirewallRuleEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *FirewallRuleEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule_engine"
}

func (r *FirewallRuleEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The ID of the rules engine rule.",
						Required:    true,
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
					"behaviors": schema.ListNestedAttribute{
						Description: "Behaviors for the rule.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
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
	}
}

func (r *FirewallRuleEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var firewallID types.Int64
	var ruleID types.Int64

	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsRuleID := req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &ruleID)
	resp.Diagnostics.Append(diagsRuleID...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, response, err := r.readRuleDataSource(ctx, firewallID.ValueInt64(), ruleID.ValueInt64())
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			result, response, err = r.readRuleDataSourceWithRetry(ctx, firewallID.ValueInt64(), ruleID.ValueInt64())

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

	state := FirewallRuleEngineDataSourceModel{
		FirewallID: firewallID,
		Results:    result,
	}
	state.ID = types.StringValue("Get By ID Firewall Rule Engine")

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *FirewallRuleEngineDataSource) readRuleDataSource(ctx context.Context, firewallID, ruleID int64) (*FirewallRuleEngineResultDataModel, *http.Response, error) {
	ruleResponse, response, err := r.client.api.FirewallsRulesEngineAPI.
		RetrieveFirewallRule(ctx, firewallID, ruleID).
		Execute()
	if err != nil {
		return nil, response, err
	}
	return transformFirewallRuleResponseToDataModel(ruleResponse.Data), response, nil
}

func (r *FirewallRuleEngineDataSource) readRuleDataSourceWithRetry(ctx context.Context, firewallID, ruleID int64) (*FirewallRuleEngineResultDataModel, *http.Response, error) {
	ruleResponse, response, err := utils.RetryOn429(func() (*azionapi.FirewallRuleResponse, *http.Response, error) {
		return r.client.api.FirewallsRulesEngineAPI.
			RetrieveFirewallRule(ctx, firewallID, ruleID).
			Execute()
	}, 5)
	if err != nil {
		return nil, response, err
	}
	return transformFirewallRuleResponseToDataModel(ruleResponse.Data), response, nil
}

func transformFirewallRuleResponseToDataModel(rule azionapi.FirewallRule) *FirewallRuleEngineResultDataModel {
	result := &FirewallRuleEngineResultDataModel{
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

	// Transform criteria
	for _, criterionGroup := range rule.Criteria {
		var entries []FirewallCriteriaEntryDataModel
		for _, c := range criterionGroup {
			arg := ""
			if c.Argument.Get() != nil {
				arg = fmt.Sprintf("%v", c.Argument.Get())
			}
			entries = append(entries, FirewallCriteriaEntryDataModel{
				Conditional: types.StringValue(c.GetConditional()),
				Variable:    types.StringValue(c.GetVariable()),
				Operator:    types.StringValue(c.GetOperator()),
				Argument:    types.StringValue(arg),
			})
		}
		result.Criteria = append(result.Criteria, FirewallCriteriaDataModel{
			Entries: entries,
		})
	}

	// Transform behaviors
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
		result.Behaviors = append(result.Behaviors, behavior)
	}

	return result
}

func transformFirewallBehaviorArgsToDataModel(attrs azionapi.FirewallBehaviorRunFunctionAttributes) FirewallBehaviorAttrsDataModel {
	return FirewallBehaviorAttrsDataModel{
		Value: types.Int64Value(attrs.GetValue()),
	}
}

func transformFirewallBehaviorObjectAttrsToDataModel(attrs azionapi.FirewallBehaviorObjectArgsAttributes) FirewallBehaviorAttrsDataModel {
	result := FirewallBehaviorAttrsDataModel{}

	if attrs.FirewallBehaviorSetCustomResponseAttributes != nil {
		result.StatusCode = types.Int64Value(attrs.FirewallBehaviorSetCustomResponseAttributes.GetStatusCode())
		if attrs.FirewallBehaviorSetCustomResponseAttributes.ContentType != nil {
			result.ContentType = types.StringValue(*attrs.FirewallBehaviorSetCustomResponseAttributes.ContentType)
		}
		if attrs.FirewallBehaviorSetCustomResponseAttributes.ContentBody != nil {
			result.ContentBody = types.StringValue(*attrs.FirewallBehaviorSetCustomResponseAttributes.ContentBody)
		}
	}

	if attrs.FirewallBehaviorSetWafAttributes != nil {
		result.WafId = types.Int64Value(attrs.FirewallBehaviorSetWafAttributes.GetWafId())
		result.Mode = types.StringValue(attrs.FirewallBehaviorSetWafAttributes.GetMode())
	}

	if attrs.FirewallBehaviorSetRateLimitAttributes != nil {
		if attrs.FirewallBehaviorSetRateLimitAttributes.Type != nil {
			result.Type = types.StringValue(*attrs.FirewallBehaviorSetRateLimitAttributes.Type)
		}
		result.LimitBy = types.StringValue(attrs.FirewallBehaviorSetRateLimitAttributes.GetLimitBy())
		result.AverageRateLimit = types.Int64Value(attrs.FirewallBehaviorSetRateLimitAttributes.GetAverageRateLimit())
		if attrs.FirewallBehaviorSetRateLimitAttributes.MaximumBurstSize.Get() != nil {
			result.MaximumBurstSize = types.Int64Value(*attrs.FirewallBehaviorSetRateLimitAttributes.MaximumBurstSize.Get())
		}
	}

	return result
}
