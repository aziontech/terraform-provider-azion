package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &RulesEngineDataSource{}
	_ datasource.DataSourceWithConfigure = &RulesEngineDataSource{}
)

func dataSourceAzionEdgeApplicationRulesEngine() datasource.DataSource {
	return &RulesEngineDataSource{}
}

type RulesEngineDataSource struct {
	client *apiClient
}

type RulesEngineDataSourceModel struct {
	SchemaVersion types.Int64                              `tfsdk:"schema_version"`
	ID            types.String                             `tfsdk:"id"`
	ApplicationID types.Int64                              `tfsdk:"edge_application_id"`
	Counter       types.Int64                              `tfsdk:"counter"`
	TotalPages    types.Int64                              `tfsdk:"total_pages"`
	Page          types.Int64                              `tfsdk:"page"`
	PageSize      types.Int64                              `tfsdk:"page_size"`
	Links         *GetEdgeApplicationsOriginsResponseLinks `tfsdk:"links"`
	Results       []RulesEngineResultModel                 `tfsdk:"results"`
}

type RulesEngineResultModel struct {
	ID          types.Int64                `tfsdk:"id"`
	Name        types.String               `tfsdk:"name"`
	Phase       types.String               `tfsdk:"phase"`
	Behaviors   []RulesEngineBehaviorModel `tfsdk:"behaviors"`
	Criteria    []CriteriaModel            `tfsdk:"criteria"`
	IsActive    types.Bool                 `tfsdk:"is_active"`
	Order       types.Int64                `tfsdk:"order"`
	Description types.String               `tfsdk:"description"`
}

type RulesEngineBehaviorModel struct {
	Name               types.String `tfsdk:"name"`
	TargetCaptureMatch TargetObject `tfsdk:"target_object"`
}

type TargetObject struct {
	Target        types.String `tfsdk:"target"`
	CapturedArray types.String `tfsdk:"captured_array"`
	Subject       types.String `tfsdk:"subject"`
	Regex         types.String `tfsdk:"regex"`
}

type CriteriaModel struct {
	Entries []RulesEngineCriteria `tfsdk:"entries"`
}

type RulesEngineCriteria struct {
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Conditional types.String `tfsdk:"conditional"`
	InputValue  types.String `tfsdk:"input_value"`
}

func (r *RulesEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *RulesEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_rules_engine"
}

func (r *RulesEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"counter": schema.Int64Attribute{
				Description: "The total number of edge applications.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of edge applications.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of edge applications.",
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
				Required: true,
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
		},
	}
}
func (r *RulesEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeApplicationID types.Int64
	var phase types.String
	var Page types.Int64
	var PageSize types.Int64

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtListIndex(0).AtName("phase"), &phase)
	resp.Diagnostics.Append(diagsPhase...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	rulesEngineResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesGet(ctx, edgeApplicationID.ValueInt64(), phase.ValueString()).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute()
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

	var previous, next string
	if rulesEngineResponse.Links.Previous.Get() != nil {
		previous = *rulesEngineResponse.Links.Previous.Get()
	}
	if rulesEngineResponse.Links.Next.Get() != nil {
		next = *rulesEngineResponse.Links.Next.Get()
	}

	var rulesEngineResults []RulesEngineResultModel
	for _, rules := range rulesEngineResponse.Results {
		var behaviors []RulesEngineBehaviorModel
		for _, behavior := range rules.Behaviors {
			if behavior.RulesEngineBehaviorString != nil {
				behaviors = append(behaviors, RulesEngineBehaviorModel{
					Name: types.StringValue(behavior.RulesEngineBehaviorString.GetName()),
					TargetCaptureMatch: TargetObject{
						Target:        types.StringValue(behavior.RulesEngineBehaviorString.GetTarget()),
						CapturedArray: types.StringValue(""),
						Subject:       types.StringValue(""),
						Regex:         types.StringValue(""),
					},
				})
			} else {
				target := behavior.RulesEngineBehaviorObject.GetTarget()
				behaviors = append(behaviors, RulesEngineBehaviorModel{
					Name: types.StringValue(behavior.RulesEngineBehaviorObject.GetName()),
					TargetCaptureMatch: TargetObject{
						Target:        types.StringValue(""),
						CapturedArray: types.StringValue(target.GetCapturedArray()),
						Subject:       types.StringValue(target.GetSubject()),
						Regex:         types.StringValue(target.GetRegex()),
					},
				})
			}
		}

		var criteria []CriteriaModel
		for _, criterion := range rules.Criteria {
			var criterionSet []RulesEngineCriteria
			for _, criterionGroup := range criterion {
				criterionSet = append(criterionSet, RulesEngineCriteria{
					Variable:    types.StringValue(criterionGroup.GetVariable()),
					Operator:    types.StringValue(criterionGroup.GetOperator()),
					Conditional: types.StringValue(criterionGroup.GetConditional()),
					InputValue:  types.StringValue(criterionGroup.GetInputValue()),
				})
			}
			criteria = append(criteria, CriteriaModel{
				Entries: criterionSet,
			})
		}
		rulesEngineResults = append(rulesEngineResults, RulesEngineResultModel{
			ID:          types.Int64Value(rules.GetId()),
			Name:        types.StringValue(rules.GetName()),
			Phase:       types.StringValue(rules.GetPhase()),
			Behaviors:   behaviors,
			Criteria:    criteria,
			IsActive:    types.BoolValue(rules.GetIsActive()),
			Order:       types.Int64Value(rules.GetOrder()),
			Description: types.StringValue(rules.GetDescription()),
		})
	}

	edgeApplicationsRulesEngineState := RulesEngineDataSourceModel{
		SchemaVersion: types.Int64Value(rulesEngineResponse.SchemaVersion),
		ApplicationID: edgeApplicationID,
		Page:          Page,
		PageSize:      PageSize,
		Results:       rulesEngineResults,
		TotalPages:    types.Int64Value(rulesEngineResponse.TotalPages),
		Counter:       types.Int64Value(rulesEngineResponse.Count),
		Links: &GetEdgeApplicationsOriginsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	edgeApplicationsRulesEngineState.ID = types.StringValue("Get All Edge Application Rules Engine")
	diags := resp.State.Set(ctx, &edgeApplicationsRulesEngineState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
