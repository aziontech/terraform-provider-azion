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
	_ datasource.DataSource              = &RuleEngineDataSource{}
	_ datasource.DataSourceWithConfigure = &RuleEngineDataSource{}
)

func dataSourceAzionEdgeApplicationRuleEngine() datasource.DataSource {
	return &RuleEngineDataSource{}
}

type RuleEngineDataSource struct {
	client *apiClient
}

type RuleEngineDataSourceModel struct {
	SchemaVersion types.Int64           `tfsdk:"schema_version"`
	ID            types.String          `tfsdk:"id"`
	ApplicationID types.Int64           `tfsdk:"edge_application_id"`
	Results       RuleEngineResultModel `tfsdk:"results"`
}

type RuleEngineResultModel struct {
	ID          types.Int64               `tfsdk:"id"`
	Name        types.String              `tfsdk:"name"`
	Phase       types.String              `tfsdk:"phase"`
	Behaviors   []RuleEngineBehaviorModel `tfsdk:"behaviors"`
	Criteria    []RuleEngineCriteriaModel `tfsdk:"criteria"`
	IsActive    types.Bool                `tfsdk:"is_active"`
	Order       types.Int64               `tfsdk:"order"`
	Description types.String              `tfsdk:"description"`
}

type RuleEngineBehaviorModel struct {
	Name               types.String  `tfsdk:"name"`
	TargetCaptureMatch TargetCapture `tfsdk:"target_object"`
}

type TargetCapture struct {
	Target        types.String `tfsdk:"target"`
	CapturedArray types.String `tfsdk:"captured_array"`
	Subject       types.String `tfsdk:"subject"`
	Regex         types.String `tfsdk:"regex"`
}

type RuleEngineCriteriaModel struct {
	Entries []RuleEngineCriteria `tfsdk:"entries"`
}

type RuleEngineCriteria struct {
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Conditional types.String `tfsdk:"conditional"`
	InputValue  types.String `tfsdk:"input_value"`
}

func (r *RuleEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *RuleEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_rule_engine"
}

func (r *RuleEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "The edge application identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
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
					"phase": schema.StringAttribute{
						Description: "The phase in which the rule is executed (e.g., default, request, response).",
						Required:    true,
					},
					"behaviors": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "The name of the behavior.",
									Computed:    true,
								},
								"target_object": schema.SingleNestedAttribute{
									Required: true,
									Attributes: map[string]schema.Attribute{
										"target": schema.StringAttribute{
											Description: "The target of the behavior.",
											Computed:    true,
										},
										"captured_array": schema.StringAttribute{
											Description: "The name of the behavior.",
											Computed:    true,
										},
										"subject": schema.StringAttribute{
											Description: "The target of the behavior.",
											Computed:    true,
										},
										"regex": schema.StringAttribute{
											Description: "The target of the behavior.",
											Computed:    true,
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
												Description: "The variable used in the rule's criteria.",
												Computed:    true,
											},
											"operator": schema.StringAttribute{
												Description: "The operator used in the rule's criteria.",
												Computed:    true,
											},
											"conditional": schema.StringAttribute{
												Description: "The conditional operator used in the rule's criteria (e.g., if, and, or).",
												Computed:    true,
											},
											"input_value": schema.StringAttribute{
												Description: "The input value used in the rule's criteria.",
												Computed:    true,
											},
										},
									},
								},
							},
						},
					},
					"is_active": schema.BoolAttribute{
						Description: "The status of the rules engine rule.",
						Computed:    true,
					},
					"order": schema.Int64Attribute{
						Description: "The order of the rule in the rules engine.",
						Computed:    true,
					},
					"description": schema.StringAttribute{
						Description: "The description of the rules engine rule.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *RuleEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeApplicationID types.Int64
	var ruleID types.Int64
	var phase types.String

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtName("phase"), &phase)
	resp.Diagnostics.Append(diagsPhase...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPage := req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &ruleID)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleEngineResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdGet(ctx, edgeApplicationID.ValueInt64(), phase.ValueString(), ruleID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			for response.StatusCode == 429 {
				err := utils.SleepAfter429(response)
				if err != nil {
					resp.Diagnostics.AddError(
						err.Error(),
						"err",
					)
					return
				}
				ruleEngineResponse, _, err = r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdGet(ctx, edgeApplicationID.ValueInt64(), phase.ValueString(), ruleID.ValueInt64()).Execute() //nolint
				if err != nil {
					resp.Diagnostics.AddError(
						err.Error(),
						"err",
					)
					return
				}
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

	var behaviors []RuleEngineBehaviorModel
	for _, behavior := range ruleEngineResponse.Results.Behaviors {
		if behavior.RulesEngineBehaviorString != nil {
			behaviors = append(behaviors, RuleEngineBehaviorModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorString.GetName()),
				TargetCaptureMatch: TargetCapture{
					Target:        types.StringValue(behavior.RulesEngineBehaviorString.GetTarget()),
					CapturedArray: types.StringValue(""),
					Subject:       types.StringValue(""),
					Regex:         types.StringValue(""),
				},
			})
		} else {
			target := behavior.RulesEngineBehaviorObject.GetTarget()
			behaviors = append(behaviors, RuleEngineBehaviorModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorObject.GetName()),
				TargetCaptureMatch: TargetCapture{
					Target:        types.StringValue(""),
					CapturedArray: types.StringValue(target.GetCapturedArray()),
					Subject:       types.StringValue(target.GetSubject()),
					Regex:         types.StringValue(target.GetRegex()),
				},
			})
		}
	}
	var criteria []RuleEngineCriteriaModel
	for _, criterion := range ruleEngineResponse.Results.Criteria {
		var criterionSet []RuleEngineCriteria
		for _, criterionGroup := range criterion {
			criterionSet = append(criterionSet, RuleEngineCriteria{
				Variable:    types.StringValue(criterionGroup.GetVariable()),
				Operator:    types.StringValue(criterionGroup.GetOperator()),
				Conditional: types.StringValue(criterionGroup.GetConditional()),
				InputValue:  types.StringValue(criterionGroup.GetInputValue()),
			})
		}
		criteria = append(criteria, RuleEngineCriteriaModel{
			Entries: criterionSet,
		})
	}
	rulesEngineResults := RuleEngineResultModel{
		ID:          types.Int64Value(ruleEngineResponse.Results.GetId()),
		Name:        types.StringValue(ruleEngineResponse.Results.GetName()),
		Phase:       types.StringValue(ruleEngineResponse.Results.GetPhase()),
		Behaviors:   behaviors,
		Criteria:    criteria,
		IsActive:    types.BoolValue(ruleEngineResponse.Results.GetIsActive()),
		Order:       types.Int64Value(ruleEngineResponse.Results.GetOrder()),
		Description: types.StringValue(ruleEngineResponse.Results.GetDescription()),
	}

	edgeApplicationsRuleEngineState := RuleEngineDataSourceModel{
		ApplicationID: edgeApplicationID,
		SchemaVersion: types.Int64Value(ruleEngineResponse.SchemaVersion),
		Results:       rulesEngineResults,
	}

	edgeApplicationsRuleEngineState.ID = types.StringValue("Get By ID Edge Application Rule Engine")
	diags := resp.State.Set(ctx, &edgeApplicationsRuleEngineState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
