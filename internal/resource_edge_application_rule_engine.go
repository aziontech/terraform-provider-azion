package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &rulesEngineResource{}
	_ resource.ResourceWithConfigure   = &rulesEngineResource{}
	_ resource.ResourceWithImportState = &rulesEngineResource{}
)

func NewEdgeApplicationRulesEngineResource() resource.Resource {
	return &rulesEngineResource{}
}

type rulesEngineResource struct {
	client *apiClient
}

type RulesEngineResourceModel struct {
	SchemaVersion types.Int64                 `tfsdk:"schema_version"`
	RulesEngine   *RulesEngineResourceResults `tfsdk:"results"`
	ID            types.String                `tfsdk:"id"`
	ApplicationID types.Int64                 `tfsdk:"edge_application_id"`
	LastUpdated   types.String                `tfsdk:"last_updated"`
}

type RulesEngineResourceResults struct {
	ID          types.Int64                        `tfsdk:"id"`
	Name        types.String                       `tfsdk:"name"`
	Phase       types.String                       `tfsdk:"phase"`
	Behaviors   []RulesEngineBehaviorResourceModel `tfsdk:"behaviors"`
	Criteria    []CriteriaResourceModel            `tfsdk:"criteria"`
	IsActive    types.Bool                         `tfsdk:"is_active"`
	Order       types.Int64                        `tfsdk:"order"`
	Description types.String                       `tfsdk:"description"`
}

type RulesEngineBehaviorResourceModel struct {
	Name               types.String           `tfsdk:"name"`
	TargetCaptureMatch *TargetCaptureResource `tfsdk:"target_object"`
}

type TargetCaptureResource struct {
	Target        types.String `tfsdk:"target"`
	CapturedArray types.String `tfsdk:"captured_array"`
	Subject       types.String `tfsdk:"subject"`
	Regex         types.String `tfsdk:"regex"`
}

type CriteriaResourceModel struct {
	Entries []RulesEngineResourceCriteria `tfsdk:"entries"`
}

type RulesEngineResourceCriteria struct {
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Conditional types.String `tfsdk:"conditional"`
	InputValue  types.String `tfsdk:"input_value"`
}

func (r *rulesEngineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_rule_engine"
}

func (r *rulesEngineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "The edge application identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The ID of the rules engine rule.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the rules engine rule.",
						Required:    true,
					},
					"phase": schema.StringAttribute{
						Description: "The phase in which the rule is executed (e.g., default, request, response).",
						Required:    true,
					},
					"behaviors": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Description: "The name of the behavior.",
									Required:    true,
								},
								"target_object": schema.SingleNestedAttribute{
									Required: true,
									Attributes: map[string]schema.Attribute{
										"target": schema.StringAttribute{
											Description: "The target of the behavior.",
											Computed:    true,
											Optional:    true,
										},
										"captured_array": schema.StringAttribute{
											Description: "The name of the behavior.",
											Computed:    true,
											Optional:    true,
										},
										"subject": schema.StringAttribute{
											Description: "The target of the behavior.",
											Computed:    true,
											Optional:    true,
										},
										"regex": schema.StringAttribute{
											Description: "The target of the behavior.",
											Computed:    true,
											Optional:    true,
										},
									},
								},
							},
						},
					},
					"criteria": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"entries": schema.ListNestedAttribute{
									Required: true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"variable": schema.StringAttribute{
												Description: "The variable used in the rule's criteria.",
												Required:    true,
											},
											"operator": schema.StringAttribute{
												Description: "The operator used in the rule's criteria.",
												Required:    true,
											},
											"conditional": schema.StringAttribute{
												Description: "The conditional operator used in the rule's criteria (e.g., if, and, or).",
												Required:    true,
											},
											"input_value": schema.StringAttribute{
												Description: "The input value used in the rule's criteria.",
												Required:    true,
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
						Optional:    true,
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *rulesEngineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *rulesEngineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RulesEngineResourceModel
	var edgeApplicationID types.Int64
	var phase types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	var behaviors []edgeapplications.RulesEngineBehaviorEntry
	for _, behavior := range plan.RulesEngine.Behaviors {
		if behavior.Name.ValueString() == "capture_match_groups" && behavior.TargetCaptureMatch.Target.ValueString() != "" {
			resp.Diagnostics.AddError("capture_match_groups",
				"Behavior capture_match_groups can not have a target value.")
			return
		}
		switch {
		case behavior.TargetCaptureMatch == nil:
		case behavior.Name.ValueString() == "capture_match_groups":
			RulesEngineBehaviorObject := edgeapplications.RulesEngineBehaviorObject{
				Name: behavior.Name.ValueString(),
				Target: edgeapplications.RulesEngineBehaviorObjectTarget{
					CapturedArray: behavior.TargetCaptureMatch.CapturedArray.ValueStringPointer(),
					Subject:       behavior.TargetCaptureMatch.Subject.ValueStringPointer(),
					Regex:         behavior.TargetCaptureMatch.Regex.ValueStringPointer(),
				},
			}
			behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorObject: &RulesEngineBehaviorObject})
		case behavior.TargetCaptureMatch.Target.ValueString() != "":
			RulesEngineBehaviorString := edgeapplications.RulesEngineBehaviorString{
				Name:   behavior.Name.ValueString(),
				Target: behavior.TargetCaptureMatch.Target.ValueString(),
			}
			behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorString: &RulesEngineBehaviorString})
		case behavior.TargetCaptureMatch.CapturedArray.IsUnknown() && behavior.TargetCaptureMatch.Regex.IsUnknown() && behavior.TargetCaptureMatch.Subject.IsUnknown():
			RulesEngineBehaviorString := edgeapplications.RulesEngineBehaviorString{
				Name:   behavior.Name.ValueString(),
				Target: "",
			}
			behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorString: &RulesEngineBehaviorString})
		}
	}

	var criteria [][]edgeapplications.RulesEngineCriteria
	for _, criterion := range plan.RulesEngine.Criteria {
		var criterionSet []edgeapplications.RulesEngineCriteria
		for _, criterionGroup := range criterion.Entries {
			criterionSet = append(criterionSet, edgeapplications.RulesEngineCriteria{
				Variable:    criterionGroup.Variable.ValueString(),
				Operator:    criterionGroup.Operator.ValueString(),
				Conditional: criterionGroup.Conditional.ValueString(),
				InputValue:  criterionGroup.InputValue.ValueStringPointer(),
			})
		}
		criteria = append(criteria, criterionSet)
	}

	var rulesEngineResponse *edgeapplications.RulesEngineIdResponse
	var response *http.Response
	var err error
	if plan.RulesEngine.Phase.ValueString() == "default" {
		// the Default Rule is automatically created when the Edge Application is created, we need to fake its creation here

		if plan.RulesEngine.Name.ValueString() != "Default Rule" {
			resp.Diagnostics.AddError(
				"Name error",
				"you need to send a default name - 'Default Rule'",
			)
			return
		}

		// both the default and first rule have the `order = 1`, so we need to get both of them and check the name
		rulesResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesGet(ctx, edgeApplicationID.ValueInt64(), "request").OrderBy("order").PageSize(2).Page(1).Sort("asc").Execute() //nolint
		if err != nil {
			if response.StatusCode == 429 {
				rulesResponse, response, err = utils.RetryOn429(func() (*edgeapplications.RulesEngineResponse, *http.Response, error) {
					return r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesGet(ctx, edgeApplicationID.ValueInt64(), "request").OrderBy("order").PageSize(2).Page(1).Sort("asc").Execute() //nolint
				}, 5) // Maximum 5 retries

				if response != nil {
					defer response.Body.Close() // <-- Close the body here
				}

				if err != nil {
					resp.Diagnostics.AddError(
						err.Error(),
						"API request failed after too many retries",
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

		var ruleID int64
		if rulesResponse.Results[0].Name == "Default Rule" && rulesResponse.Results[0].Phase == "default" {
			ruleID = rulesResponse.Results[0].Id
		} else {
			ruleID = rulesResponse.Results[1].Id
		}

		rulesEngineRequest := edgeapplications.UpdateRulesEngineRequest{
			Name:        plan.RulesEngine.Name.ValueString(),
			Description: plan.RulesEngine.Description.ValueStringPointer(),
			Behaviors:   behaviors,
			Criteria:    criteria,
		}

		rulesEngineResponse, response, err = r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdPut(ctx, edgeApplicationID.ValueInt64(), "request", ruleID).UpdateRulesEngineRequest(rulesEngineRequest).Execute() //nolint
		if err != nil {
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
	} else {
		rulesEngineRequest := edgeapplications.CreateRulesEngineRequest{
			Name:        plan.RulesEngine.Name.ValueString(),
			Description: plan.RulesEngine.Description.ValueStringPointer(),
			Behaviors:   behaviors,
			Criteria:    criteria,
		}

		rulesEngineResponse, response, err = r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.
			EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesPost(ctx, edgeApplicationID.ValueInt64(), phase.ValueString()).
			CreateRulesEngineRequest(rulesEngineRequest).Execute() //nolint
	}

	if err != nil {
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

	var behaviorResponse []RulesEngineBehaviorResourceModel
	for _, behavior := range rulesEngineResponse.Results.Behaviors {
		if behavior.RulesEngineBehaviorString != nil {
			behaviorResponse = append(behaviorResponse, RulesEngineBehaviorResourceModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorString.GetName()),
				TargetCaptureMatch: &TargetCaptureResource{
					Target: types.StringValue(behavior.RulesEngineBehaviorString.GetTarget()),
				},
			})
		} else {
			target := behavior.RulesEngineBehaviorObject.GetTarget()
			behaviorResponse = append(behaviorResponse, RulesEngineBehaviorResourceModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorObject.GetName()),
				TargetCaptureMatch: &TargetCaptureResource{
					CapturedArray: types.StringValue(target.GetCapturedArray()),
					Subject:       types.StringValue(target.GetSubject()),
					Regex:         types.StringValue(target.GetRegex()),
				},
			})
		}
	}

	var criteriaResult []CriteriaResourceModel
	for _, criterion := range rulesEngineResponse.Results.Criteria {
		var criterionSet []RulesEngineResourceCriteria
		for _, criterionGroup := range criterion {
			criterionSet = append(criterionSet, RulesEngineResourceCriteria{
				Variable:    types.StringValue(criterionGroup.GetVariable()),
				Operator:    types.StringValue(criterionGroup.GetOperator()),
				Conditional: types.StringValue(criterionGroup.GetConditional()),
				InputValue:  types.StringValue(criterionGroup.GetInputValue()),
			})
		}
		criteriaResult = append(criteriaResult, CriteriaResourceModel{
			Entries: criterionSet,
		})
	}
	rulesEngineResults := &RulesEngineResourceResults{
		ID:          types.Int64Value(rulesEngineResponse.Results.GetId()),
		Name:        types.StringValue(rulesEngineResponse.Results.GetName()),
		Phase:       types.StringValue(rulesEngineResponse.Results.GetPhase()),
		Behaviors:   behaviorResponse,
		Criteria:    criteriaResult,
		IsActive:    types.BoolValue(rulesEngineResponse.Results.GetIsActive()),
		Order:       types.Int64Value(rulesEngineResponse.Results.GetOrder()),
		Description: types.StringValue(rulesEngineResponse.Results.GetDescription()),
	}

	plan = RulesEngineResourceModel{
		ApplicationID: edgeApplicationID,
		ID:            types.StringValue(strconv.FormatInt(rulesEngineResponse.Results.GetId(), 10)),
		LastUpdated:   types.StringValue(time.Now().Format(time.RFC850)),
		SchemaVersion: types.Int64Value(rulesEngineResponse.SchemaVersion),
		RulesEngine:   rulesEngineResults,
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *rulesEngineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RulesEngineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var edgeApplicationID int64
	var ruleID int64
	var phase string
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) == 3 {
		edgeApplicationID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		phase = valueFromCmd[1]
		ruleID = int64(utils.AtoiNoError(valueFromCmd[2], resp))
	} else {
		if len(valueFromCmd) == 1 {
			edgeApplicationID = state.ApplicationID.ValueInt64()
			ruleID = state.RulesEngine.ID.ValueInt64()
			phase = state.RulesEngine.Phase.ValueString()
		} else {
			resp.Diagnostics.AddError(
				"Parameters error",
				"you need to pass <edgeApplicationID>/<phase>/<ruleEngineID>",
			)
			return
		}
	}

	if ruleID == 0 {
		resp.Diagnostics.AddError(
			"Rules ID id error ",
			"is not null",
		)
		return
	}

	if phase == "default" {
		phase = "request"
	}

	ruleEngineResponse, response, err := r.client.edgeApplicationsApi.
		EdgeApplicationsRulesEngineAPI.
		EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdGet(
			ctx, edgeApplicationID, phase, ruleID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			ruleEngineResponse, response, err = utils.RetryOn429(func() (*edgeapplications.RulesEngineIdResponse, *http.Response, error) {
				return r.client.edgeApplicationsApi.
					EdgeApplicationsRulesEngineAPI.
					EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdGet(ctx, edgeApplicationID, phase, ruleID).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
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

	var behaviorResponse []RulesEngineBehaviorResourceModel
	for _, behavior := range ruleEngineResponse.Results.Behaviors {
		if behavior.RulesEngineBehaviorString != nil {
			behaviorResponse = append(behaviorResponse, RulesEngineBehaviorResourceModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorString.GetName()),
				TargetCaptureMatch: &TargetCaptureResource{
					Target: types.StringValue(behavior.RulesEngineBehaviorString.GetTarget()),
				},
			})
		} else {
			target := behavior.RulesEngineBehaviorObject.GetTarget()
			behaviorResponse = append(behaviorResponse, RulesEngineBehaviorResourceModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorObject.GetName()),
				TargetCaptureMatch: &TargetCaptureResource{
					CapturedArray: types.StringValue(target.GetCapturedArray()),
					Subject:       types.StringValue(target.GetSubject()),
					Regex:         types.StringValue(target.GetRegex()),
				},
			})
		}
	}

	var criteria []CriteriaResourceModel
	for _, criterion := range ruleEngineResponse.Results.Criteria {
		var criterionSet []RulesEngineResourceCriteria
		for _, criterionGroup := range criterion {
			criterionSet = append(criterionSet, RulesEngineResourceCriteria{
				Variable:    types.StringValue(criterionGroup.GetVariable()),
				Operator:    types.StringValue(criterionGroup.GetOperator()),
				Conditional: types.StringValue(criterionGroup.GetConditional()),
				InputValue:  types.StringValue(criterionGroup.GetInputValue()),
			})
		}
		criteria = append(criteria, CriteriaResourceModel{
			Entries: criterionSet,
		})
	}
	state.RulesEngine = &RulesEngineResourceResults{
		ID:          types.Int64Value(ruleEngineResponse.Results.GetId()),
		Name:        types.StringValue(ruleEngineResponse.Results.GetName()),
		Phase:       types.StringValue(ruleEngineResponse.Results.GetPhase()),
		Behaviors:   behaviorResponse,
		Criteria:    criteria,
		IsActive:    types.BoolValue(ruleEngineResponse.Results.GetIsActive()),
		Order:       types.Int64Value(ruleEngineResponse.Results.GetOrder()),
		Description: types.StringValue(ruleEngineResponse.Results.GetDescription()),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *rulesEngineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RulesEngineResourceModel
	var edgeApplicationID types.Int64
	var ruleID types.Int64
	var phase types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RulesEngineResourceModel
	diagsOrigin := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsOrigin...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.RulesEngine.Phase.ValueString() == "" {
		phase = state.RulesEngine.Phase
	} else {
		if state.RulesEngine.Phase.ValueString() == "default" && plan.RulesEngine.Phase.ValueString() != "default" {
			resp.Diagnostics.AddError(
				"Phase error",
				"you need to send a default phase",
			)
			return
		}
		phase = plan.RulesEngine.Phase
	}

	if plan.ApplicationID.IsNull() {
		edgeApplicationID = state.ApplicationID
	} else {
		edgeApplicationID = plan.ApplicationID
	}

	if plan.RulesEngine.ID.IsNull() || plan.RulesEngine.ID.ValueInt64() == 0 {
		ruleID = state.RulesEngine.ID
	} else {
		ruleID = plan.RulesEngine.ID
	}

	var behaviors []edgeapplications.RulesEngineBehaviorEntry
	for _, behavior := range plan.RulesEngine.Behaviors {
		if behavior.Name.ValueString() == "capture_match_groups" && behavior.TargetCaptureMatch.Target.ValueString() != "" {
			resp.Diagnostics.AddError("capture_match_groups",
				"Behavior capture_match_groups can not have a target value.")
			return
		}
		switch {
		case behavior.TargetCaptureMatch == nil:
		case behavior.Name.ValueString() == "capture_match_groups":
			RulesEngineBehaviorObject := edgeapplications.RulesEngineBehaviorObject{
				Name: behavior.Name.ValueString(),
				Target: edgeapplications.RulesEngineBehaviorObjectTarget{
					CapturedArray: behavior.TargetCaptureMatch.CapturedArray.ValueStringPointer(),
					Subject:       behavior.TargetCaptureMatch.Subject.ValueStringPointer(),
					Regex:         behavior.TargetCaptureMatch.Regex.ValueStringPointer(),
				},
			}
			behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorObject: &RulesEngineBehaviorObject})
		case behavior.TargetCaptureMatch.Target.ValueString() != "":
			RulesEngineBehaviorString := edgeapplications.RulesEngineBehaviorString{
				Name:   behavior.Name.ValueString(),
				Target: behavior.TargetCaptureMatch.Target.ValueString(),
			}
			behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorString: &RulesEngineBehaviorString})
		case behavior.TargetCaptureMatch.CapturedArray.IsUnknown() && behavior.TargetCaptureMatch.Regex.IsUnknown() && behavior.TargetCaptureMatch.Subject.IsUnknown():
			RulesEngineBehaviorString := edgeapplications.RulesEngineBehaviorString{
				Name:   behavior.Name.ValueString(),
				Target: "",
			}
			behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorString: &RulesEngineBehaviorString})
		}
	}

	var criteria [][]edgeapplications.RulesEngineCriteria
	for _, criterion := range plan.RulesEngine.Criteria {
		var criterionSet []edgeapplications.RulesEngineCriteria
		for _, criterionGroup := range criterion.Entries {
			criterionSet = append(criterionSet, edgeapplications.RulesEngineCriteria{
				Variable:    criterionGroup.Variable.ValueString(),
				Operator:    criterionGroup.Operator.ValueString(),
				Conditional: criterionGroup.Conditional.ValueString(),
				InputValue:  criterionGroup.InputValue.ValueStringPointer(),
			})
		}
		criteria = append(criteria, criterionSet)
	}

	var rulesEngineRequest edgeapplications.UpdateRulesEngineRequest
	if state.RulesEngine.Phase.ValueString() == "default" {
		if plan.RulesEngine.Name.ValueString() != "Default Rule" {
			resp.Diagnostics.AddError(
				"Name error",
				"you need to send a default name - 'Default Rule'",
			)
			return
		}
	}

	rulesEngineRequest = edgeapplications.UpdateRulesEngineRequest{
		Name:        plan.RulesEngine.Name.ValueString(),
		Description: plan.RulesEngine.Description.ValueStringPointer(),
		Behaviors:   behaviors,
		Criteria:    criteria,
	}

	if phase.ValueString() == "default" {
		phase = types.StringValue("request")
	}

	rulesEngineResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdPut(ctx, edgeApplicationID.ValueInt64(), phase.ValueString(), ruleID.ValueInt64()).UpdateRulesEngineRequest(rulesEngineRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			rulesEngineResponse, response, err = utils.RetryOn429(func() (*edgeapplications.RulesEngineIdResponse, *http.Response, error) {
				return r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdPut(ctx, edgeApplicationID.ValueInt64(), phase.ValueString(), ruleID.ValueInt64()).UpdateRulesEngineRequest(rulesEngineRequest).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
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

	var behaviorResponse []RulesEngineBehaviorResourceModel
	for _, behavior := range rulesEngineResponse.Results.Behaviors {
		if behavior.RulesEngineBehaviorString != nil {
			behaviorResponse = append(behaviorResponse, RulesEngineBehaviorResourceModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorString.GetName()),
				TargetCaptureMatch: &TargetCaptureResource{
					Target: types.StringValue(behavior.RulesEngineBehaviorString.GetTarget()),
				},
			})
		} else {
			target := behavior.RulesEngineBehaviorObject.GetTarget()
			behaviorResponse = append(behaviorResponse, RulesEngineBehaviorResourceModel{
				Name: types.StringValue(behavior.RulesEngineBehaviorObject.GetName()),
				TargetCaptureMatch: &TargetCaptureResource{
					CapturedArray: types.StringValue(target.GetCapturedArray()),
					Subject:       types.StringValue(target.GetSubject()),
					Regex:         types.StringValue(target.GetRegex()),
				},
			})
		}
	}

	var criteriaResult []CriteriaResourceModel
	for _, criterion := range rulesEngineResponse.Results.Criteria {
		var criterionSet []RulesEngineResourceCriteria
		for _, criterionGroup := range criterion {
			criterionSet = append(criterionSet, RulesEngineResourceCriteria{
				Variable:    types.StringValue(criterionGroup.GetVariable()),
				Operator:    types.StringValue(criterionGroup.GetOperator()),
				Conditional: types.StringValue(criterionGroup.GetConditional()),
				InputValue:  types.StringValue(criterionGroup.GetInputValue()),
			})
		}
		criteriaResult = append(criteriaResult, CriteriaResourceModel{
			Entries: criterionSet,
		})
	}
	rulesEngineResults := &RulesEngineResourceResults{
		ID:          types.Int64Value(rulesEngineResponse.Results.GetId()),
		Name:        types.StringValue(rulesEngineResponse.Results.GetName()),
		Phase:       types.StringValue(rulesEngineResponse.Results.GetPhase()),
		Behaviors:   behaviorResponse,
		Criteria:    criteriaResult,
		IsActive:    types.BoolValue(rulesEngineResponse.Results.GetIsActive()),
		Order:       types.Int64Value(rulesEngineResponse.Results.GetOrder()),
		Description: types.StringValue(rulesEngineResponse.Results.GetDescription()),
	}

	plan = RulesEngineResourceModel{
		ApplicationID: edgeApplicationID,
		ID:            plan.ID,
		LastUpdated:   types.StringValue(time.Now().Format(time.RFC850)),
		SchemaVersion: types.Int64Value(rulesEngineResponse.SchemaVersion),
		RulesEngine:   rulesEngineResults,
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *rulesEngineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RulesEngineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ApplicationID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}

	if state.RulesEngine.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Rules Engine ID error ",
			"is not null",
		)
		return
	}

	if state.RulesEngine.Phase.IsNull() {
		resp.Diagnostics.AddError(
			"Phase error ",
			"is not null",
		)
		return
	}

	// Default Rule cannot be deleted, so we just set its behavior to no_content and return
	if state.RulesEngine.Name == types.StringValue("Default Rule") {
		var behaviors []edgeapplications.RulesEngineBehaviorEntry
		RulesEngineBehaviorString := edgeapplications.RulesEngineBehaviorString{
			Name:   "no_content",
			Target: "",
		}
		behaviors = append(behaviors, edgeapplications.RulesEngineBehaviorEntry{RulesEngineBehaviorString: &RulesEngineBehaviorString})
		var rulesEngineRequest edgeapplications.PatchRulesEngineRequest
		rulesEngineRequest.Behaviors = behaviors

		_, response, err := r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdPatch(ctx, state.ApplicationID.ValueInt64(), "request", state.RulesEngine.ID.ValueInt64()).PatchRulesEngineRequest(rulesEngineRequest).Execute() //nolint
		if err != nil {
			if response.StatusCode == 429 {
				_, response, err = utils.RetryOn429(func() (*edgeapplications.RulesEngineIdResponse, *http.Response, error) {
					return r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdPatch(ctx, state.ApplicationID.ValueInt64(), "request", state.RulesEngine.ID.ValueInt64()).PatchRulesEngineRequest(rulesEngineRequest).Execute() //nolint
				}, 5) // Maximum 5 retries

				if response != nil {
					defer response.Body.Close() // <-- Close the body here
				}

				if err != nil {
					resp.Diagnostics.AddError(
						err.Error(),
						"API request failed after too many retries",
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
		resp.Diagnostics.AddWarning(
			"Default Rule",
			"Default Rule cannot be deleted. Behaviors were set to default values instead.",
		)
	} else {
		response, err := r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdDelete(ctx, state.ApplicationID.ValueInt64(), state.RulesEngine.Phase.ValueString(), state.RulesEngine.ID.ValueInt64()).Execute() //nolint
		if err != nil {
			if response.StatusCode == 429 {
				response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
					return r.client.edgeApplicationsApi.EdgeApplicationsRulesEngineAPI.EdgeApplicationsEdgeApplicationIdRulesEnginePhaseRulesRuleIdDelete(ctx, state.ApplicationID.ValueInt64(), state.RulesEngine.Phase.ValueString(), state.RulesEngine.ID.ValueInt64()).Execute() //nolint
				}, 5) // Maximum 5 retries

				if response != nil {
					defer response.Body.Close() // <-- Close the body here
				}

				if err != nil {
					resp.Diagnostics.AddError(
						err.Error(),
						"API request failed after too many retries",
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
	}

}

func (r *rulesEngineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
