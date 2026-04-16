package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
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

func NewApplicationRulesEngineResource() resource.Resource {
	return &rulesEngineResource{}
}

type rulesEngineResource struct {
	client *apiClient
}

type RulesEngineResourceModel struct {
	SchemaVersion types.Int64                 `tfsdk:"schema_version"`
	RulesEngine   *RulesEngineResourceResults `tfsdk:"results"`
	ID            types.String                `tfsdk:"id"`
	ApplicationID types.Int64                 `tfsdk:"application_id"`
	LastUpdated   types.String                `tfsdk:"last_updated"`
}

type RulesEngineResourceResults struct {
	ID           types.Int64                        `tfsdk:"id"`
	Name         types.String                       `tfsdk:"name"`
	Phase        types.String                       `tfsdk:"phase"`
	Active       types.Bool                         `tfsdk:"active"`
	Behaviors    []RulesEngineBehaviorResourceModel `tfsdk:"behaviors"`
	Criteria     []CriteriaResourceModel            `tfsdk:"criteria"`
	Description  types.String                       `tfsdk:"description"`
	Order        types.Int64                        `tfsdk:"order"`
	LastEditor   types.String                       `tfsdk:"last_editor"`
	LastModified types.String                       `tfsdk:"last_modified"`
	CreatedAt    types.String                       `tfsdk:"created_at"`
}

type RulesEngineBehaviorResourceModel struct {
	Type         types.String                     `tfsdk:"type"`
	Attributes   *BehaviorAttributesResourceModel `tfsdk:"attributes"`
	CaptureAttrs *CaptureAttributesResourceModel  `tfsdk:"capture_attributes"`
}

type BehaviorAttributesResourceModel struct {
	Value types.String `tfsdk:"value"`
}

type CaptureAttributesResourceModel struct {
	Subject       types.String `tfsdk:"subject"`
	Regex         types.String `tfsdk:"regex"`
	CapturedArray types.String `tfsdk:"captured_array"`
}

type CriteriaResourceModel struct {
	Entries []RulesEngineResourceCriteria `tfsdk:"entries"`
}

type RulesEngineResourceCriteria struct {
	Conditional types.String `tfsdk:"conditional"`
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Argument    types.String `tfsdk:"argument"`
}

func (r *rulesEngineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_rule_engine"
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
			"application_id": schema.Int64Attribute{
				Description: "The application identifier.",
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
						Description: "The phase in which the rule is executed (request or response).",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the rule is active.",
						Optional:    true,
						Computed:    true,
					},
					"behaviors": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "The type of behavior.",
									Required:    true,
								},
								"attributes": schema.SingleNestedAttribute{
									Description: "Behavior attributes (for behaviors with args).",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"value": schema.StringAttribute{
											Description: "Value for the behavior.",
											Required:    true,
										},
									},
								},
								"capture_attributes": schema.SingleNestedAttribute{
									Description: "Capture attributes (for capture_match_groups).",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"subject": schema.StringAttribute{
											Description: "Subject for capture.",
											Required:    true,
										},
										"regex": schema.StringAttribute{
											Description: "Regex pattern.",
											Required:    true,
										},
										"captured_array": schema.StringAttribute{
											Description: "Captured array name.",
											Required:    true,
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
											"conditional": schema.StringAttribute{
												Description: "The conditional operator used in the rule's criteria (e.g., if, and, or).",
												Required:    true,
											},
											"variable": schema.StringAttribute{
												Description: "The variable used in the rule's criteria.",
												Required:    true,
											},
											"operator": schema.StringAttribute{
												Description: "The operator used in the rule's criteria.",
												Required:    true,
											},
											"argument": schema.StringAttribute{
												Description: "The argument used in the rule's criteria.",
												Optional:    true,
											},
										},
									},
								},
							},
						},
					},
					"description": schema.StringAttribute{
						Description: "The description of the rules engine rule.",
						Optional:    true,
						Computed:    true,
					},
					"order": schema.Int64Attribute{
						Description: "The order of the rule in the rules engine.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the rule.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "The last modified timestamp.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp.",
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
	var applicationID types.Int64
	var phase types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
	resp.Diagnostics.Append(diagsApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtName("phase"), &phase)
	resp.Diagnostics.Append(diagsPhase...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build criteria
	criteria := buildCriteriaRequestV4(plan.RulesEngine.Criteria)

	phaseStr := phase.ValueString()

	// Default Rule handling - it exists automatically, we just update it
	if phaseStr == "default" {
		if plan.RulesEngine.Name.ValueString() != "Default Rule" {
			resp.Diagnostics.AddError(
				"Name error",
				"you need to send a default name - 'Default Rule'",
			)
			return
		}

		// Find the Default Rule ID - it's in the request phase
		rulesResponse, response, err := r.client.api.ApplicationsRequestRulesAPI.
			ListApplicationRequestRules(ctx, applicationID.ValueInt64()).
			Page(1).PageSize(2).Execute()
		if err != nil {
			if response.StatusCode == 429 {
				rulesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedRequestPhaseRuleList, *http.Response, error) {
					return r.client.api.ApplicationsRequestRulesAPI.
						ListApplicationRequestRules(ctx, applicationID.ValueInt64()).
						Page(1).PageSize(2).Execute()
				}, 5)

				if response != nil {
					defer response.Body.Close()
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
				response.Body.Close()
				return
			}
		}
		if response != nil {
			defer response.Body.Close()
		}

		var ruleID int64
		for _, rule := range rulesResponse.Results {
			if rule.Name == "Default Rule" {
				ruleID = rule.GetId()
				break
			}
		}

		if ruleID == 0 {
			resp.Diagnostics.AddError(
				"Default Rule not found",
				"Could not find Default Rule in the request phase",
			)
			return
		}

		// Update the Default Rule
		behaviors := buildBehaviorsRequestV4(plan.RulesEngine.Behaviors)
		ruleRequest := azionapi.NewRequestPhaseRuleRequest(
			plan.RulesEngine.Name.ValueString(),
			criteria,
			behaviors,
		)

		if !plan.RulesEngine.Active.IsNull() && !plan.RulesEngine.Active.IsUnknown() {
			ruleRequest.SetActive(plan.RulesEngine.Active.ValueBool())
		}
		if !plan.RulesEngine.Description.IsNull() && !plan.RulesEngine.Description.IsUnknown() {
			ruleRequest.SetDescription(plan.RulesEngine.Description.ValueString())
		}

		updateResponse, response, err := r.client.api.ApplicationsRequestRulesAPI.
			UpdateApplicationRequestRule(ctx, applicationID.ValueInt64(), ruleID).
			RequestPhaseRuleRequest(*ruleRequest).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			handleResourceAPIError(resp, response, err)
			return
		}

		plan = buildStateFromResponse(updateResponse.Data, applicationID, types.StringValue("default"))
	} else {
		// Create new rule for request or response phase
		var rulesEngineResponse *azionapi.RequestPhaseRuleResponse
		var response *http.Response
		var err error

		if phaseStr == "request" {
			behaviors := buildBehaviorsRequestV4(plan.RulesEngine.Behaviors)
			ruleRequest := azionapi.NewRequestPhaseRuleRequest(
				plan.RulesEngine.Name.ValueString(),
				criteria,
				behaviors,
			)

			if !plan.RulesEngine.Active.IsNull() && !plan.RulesEngine.Active.IsUnknown() {
				ruleRequest.SetActive(plan.RulesEngine.Active.ValueBool())
			}
			if !plan.RulesEngine.Description.IsNull() && !plan.RulesEngine.Description.IsUnknown() {
				ruleRequest.SetDescription(plan.RulesEngine.Description.ValueString())
			}

			rulesEngineResponse, response, err = r.client.api.ApplicationsRequestRulesAPI.
				CreateApplicationRequestRule(ctx, applicationID.ValueInt64()).
				RequestPhaseRuleRequest(*ruleRequest).
				Execute()
			if response != nil {
				defer response.Body.Close()
			}
		} else if phaseStr == "response" {
			behaviors := buildBehaviorsResponseV4(plan.RulesEngine.Behaviors)
			responseRuleRequest := azionapi.NewResponsePhaseRuleRequest(
				plan.RulesEngine.Name.ValueString(),
				criteria,
				behaviors,
			)

			if !plan.RulesEngine.Active.IsNull() && !plan.RulesEngine.Active.IsUnknown() {
				responseRuleRequest.SetActive(plan.RulesEngine.Active.ValueBool())
			}
			if !plan.RulesEngine.Description.IsNull() && !plan.RulesEngine.Description.IsUnknown() {
				responseRuleRequest.SetDescription(plan.RulesEngine.Description.ValueString())
			}

			var responseRulesEngineResponse *azionapi.ResponsePhaseRuleResponse
			responseRulesEngineResponse, response, err = r.client.api.ApplicationsResponseRulesAPI.
				CreateApplicationResponseRule(ctx, applicationID.ValueInt64()).
				ResponsePhaseRuleRequest(*responseRuleRequest).
				Execute()
			if response != nil {
				defer response.Body.Close()
			}
			if err == nil {
				// Convert response to request phase rule response format for consistent handling
				rulesEngineResponse = &azionapi.RequestPhaseRuleResponse{
					Data: convertResponseToRequestPhaseRule(responseRulesEngineResponse.Data),
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Invalid phase value",
				fmt.Sprintf("Phase must be 'request', 'response', or 'default', got: %s", phaseStr),
			)
			return
		}

		if err != nil {
			handleResourceAPIError(resp, response, err)
			return
		}

		plan = buildStateFromResponse(rulesEngineResponse.Data, applicationID, phase)
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

	var applicationID int64
	var ruleID int64
	var phase string
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) == 3 {
		applicationID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		phase = valueFromCmd[1]
		ruleID = int64(utils.AtoiNoError(valueFromCmd[2], resp))
	} else if len(valueFromCmd) == 1 {
		applicationID = state.ApplicationID.ValueInt64()
		ruleID = state.RulesEngine.ID.ValueInt64()
		phase = state.RulesEngine.Phase.ValueString()
	} else {
		resp.Diagnostics.AddError(
			"Parameters error",
			"you need to pass <applicationID>/<phase>/<ruleEngineID>",
		)
		return
	}

	if ruleID == 0 {
		resp.Diagnostics.AddError(
			"Rules ID error",
			"rule ID cannot be 0",
		)
		return
	}

	// Default phase is actually stored in request phase
	apiPhase := phase
	if phase == "default" {
		apiPhase = "request"
	}

	var result *RulesEngineResourceResults

	if apiPhase == "request" {
		ruleResponse, response, err := r.client.api.ApplicationsRequestRulesAPI.
			RetrieveApplicationRequestRule(ctx, applicationID, ruleID).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
			handleResourceAPIError(resp, response, err)
			return
		}
		result = transformRuleToResultsModelFromResponse(ruleResponse.Data, phase)
	} else if apiPhase == "response" {
		ruleResponse, response, err := r.client.api.ApplicationsResponseRulesAPI.
			RetrieveApplicationResponseRule(ctx, applicationID, ruleID).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
			handleResourceAPIError(resp, response, err)
			return
		}
		result = transformRuleToResultsModelFromResponse(ruleResponse.Data, phase)
	}

	state.ApplicationID = types.Int64Value(applicationID)
	state.RulesEngine = result

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *rulesEngineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RulesEngineResourceModel
	var applicationID types.Int64
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
		applicationID = state.ApplicationID
	} else {
		applicationID = plan.ApplicationID
	}

	if plan.RulesEngine.ID.IsNull() || plan.RulesEngine.ID.ValueInt64() == 0 {
		ruleID = state.RulesEngine.ID
	} else {
		ruleID = plan.RulesEngine.ID
	}

	// Build criteria
	criteria := buildCriteriaRequestV4(plan.RulesEngine.Criteria)

	phaseStr := phase.ValueString()
	apiPhase := phaseStr
	if phaseStr == "default" {
		apiPhase = "request"
	}

	// Validate Default Rule name
	if state.RulesEngine.Phase.ValueString() == "default" {
		if plan.RulesEngine.Name.ValueString() != "Default Rule" {
			resp.Diagnostics.AddError(
				"Name error",
				"you need to send a default name - 'Default Rule'",
			)
			return
		}
	}

	var rulesEngineResponse *azionapi.RequestPhaseRuleResponse
	var response *http.Response
	var err error

	if apiPhase == "request" {
		behaviors := buildBehaviorsRequestV4(plan.RulesEngine.Behaviors)
		ruleRequest := azionapi.NewRequestPhaseRuleRequest(
			plan.RulesEngine.Name.ValueString(),
			criteria,
			behaviors,
		)

		if !plan.RulesEngine.Active.IsNull() && !plan.RulesEngine.Active.IsUnknown() {
			ruleRequest.SetActive(plan.RulesEngine.Active.ValueBool())
		}
		if !plan.RulesEngine.Description.IsNull() && !plan.RulesEngine.Description.IsUnknown() {
			ruleRequest.SetDescription(plan.RulesEngine.Description.ValueString())
		}

		rulesEngineResponse, response, err = r.client.api.ApplicationsRequestRulesAPI.
			UpdateApplicationRequestRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64()).
			RequestPhaseRuleRequest(*ruleRequest).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
	} else if apiPhase == "response" {
		behaviors := buildBehaviorsResponseV4(plan.RulesEngine.Behaviors)
		responseRuleRequest := azionapi.NewResponsePhaseRuleRequest(
			plan.RulesEngine.Name.ValueString(),
			criteria,
			behaviors,
		)

		if !plan.RulesEngine.Active.IsNull() && !plan.RulesEngine.Active.IsUnknown() {
			responseRuleRequest.SetActive(plan.RulesEngine.Active.ValueBool())
		}
		if !plan.RulesEngine.Description.IsNull() && !plan.RulesEngine.Description.IsUnknown() {
			responseRuleRequest.SetDescription(plan.RulesEngine.Description.ValueString())
		}

		var responseRulesEngineResponse *azionapi.ResponsePhaseRuleResponse
		responseRulesEngineResponse, response, err = r.client.api.ApplicationsResponseRulesAPI.
			UpdateApplicationResponseRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64()).
			ResponsePhaseRuleRequest(*responseRuleRequest).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err == nil {
			rulesEngineResponse = &azionapi.RequestPhaseRuleResponse{
				Data: convertResponseToRequestPhaseRule(responseRulesEngineResponse.Data),
			}
		}
	}

	if err != nil {
		handleResourceAPIError(resp, response, err)
		return
	}

	plan = buildStateFromResponse(rulesEngineResponse.Data, applicationID, phase)

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
			"Application ID error",
			"Application ID cannot be null",
		)
		return
	}

	if state.RulesEngine.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Rules Engine ID error",
			"Rules Engine ID cannot be null",
		)
		return
	}

	if state.RulesEngine.Phase.IsNull() {
		resp.Diagnostics.AddError(
			"Phase error",
			"Phase cannot be null",
		)
		return
	}

	// Default Rule cannot be deleted, so we just set its behavior to deliver and return
	if state.RulesEngine.Name == types.StringValue("Default Rule") {
		// Create a deliver behavior
		deliverBehavior := azionapi.NewBehaviorNoArgs("deliver")
		behaviorRequest := azionapi.BehaviorNoArgsAsRequestPhaseBehaviorRequest(deliverBehavior)
		behaviors := []azionapi.RequestPhaseBehaviorRequest{behaviorRequest}

		// Empty criteria for default
		criteria := [][]azionapi.ApplicationCriterionFieldRequest{}

		ruleRequest := azionapi.NewRequestPhaseRuleRequest(
			"Default Rule",
			criteria,
			behaviors,
		)

		_, response, err := r.client.api.ApplicationsRequestRulesAPI.
			UpdateApplicationRequestRule(ctx, state.ApplicationID.ValueInt64(), state.RulesEngine.ID.ValueInt64()).
			RequestPhaseRuleRequest(*ruleRequest).
			Execute()
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			handleResourceAPIError(resp, response, err)
			return
		}

		resp.Diagnostics.AddWarning(
			"Default Rule",
			"Default Rule cannot be deleted. Behaviors were set to default values instead.",
		)
	} else {
		phase := state.RulesEngine.Phase.ValueString()
		var response *http.Response
		var err error

		if phase == "request" {
			_, response, err = r.client.api.ApplicationsRequestRulesAPI.
				DeleteApplicationRequestRule(ctx, state.ApplicationID.ValueInt64(), state.RulesEngine.ID.ValueInt64()).
				Execute()
		} else if phase == "response" {
			_, response, err = r.client.api.ApplicationsResponseRulesAPI.
				DeleteApplicationResponseRule(ctx, state.ApplicationID.ValueInt64(), state.RulesEngine.ID.ValueInt64()).
				Execute()
		}

		if response != nil {
			defer response.Body.Close()
		}

		if err != nil {
			if response != nil && response.StatusCode != http.StatusNotFound {
				handleResourceAPIError(resp, response, err)
				return
			}
		}
	}
}

func (r *rulesEngineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Expected format: {application_id}/{phase}/{rule_id}",
		)
		return
	}

	applicationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid application ID",
			"Could not parse application ID",
		)
		return
	}

	phase := parts[1]

	ruleID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid rule ID",
			"Could not parse rule ID",
		)
		return
	}

	var state RulesEngineResourceModel
	state.ApplicationID = types.Int64Value(applicationID)
	state.RulesEngine = &RulesEngineResourceResults{
		ID:    types.Int64Value(ruleID),
		Phase: types.StringValue(phase),
	}
	state.ID = types.StringValue(fmt.Sprintf("%d/%s/%d", applicationID, phase, ruleID))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Helper functions

func buildCriteriaRequestV4(criteria []CriteriaResourceModel) [][]azionapi.ApplicationCriterionFieldRequest {
	var result [][]azionapi.ApplicationCriterionFieldRequest
	for _, criterion := range criteria {
		var criterionGroup []azionapi.ApplicationCriterionFieldRequest
		for _, c := range criterion.Entries {
			criterionField := azionapi.NewApplicationCriterionFieldRequest(
				c.Conditional.ValueString(),
				c.Variable.ValueString(),
				c.Operator.ValueString(),
			)
			if !c.Argument.IsNull() && !c.Argument.IsUnknown() {
				arg := azionapi.ApplicationCriterionArgumentRequest{
					String: c.Argument.ValueStringPointer(),
				}
				criterionField.SetArgument(arg)
			}
			criterionGroup = append(criterionGroup, *criterionField)
		}
		result = append(result, criterionGroup)
	}
	return result
}

func buildBehaviorsRequestV4(behaviors []RulesEngineBehaviorResourceModel) []azionapi.RequestPhaseBehaviorRequest {
	var result []azionapi.RequestPhaseBehaviorRequest
	for _, b := range behaviors {
		if b.Attributes != nil && !b.Attributes.Value.IsNull() {
			// Behavior with args
			value := azionapi.BehaviorArgsAttributesValue{
				String: b.Attributes.Value.ValueStringPointer(),
			}
			attrs := azionapi.NewBehaviorArgsAttributes(value)
			argsBehavior := azionapi.NewBehaviorArgs(b.Type.ValueString(), *attrs)
			behaviorRequest := azionapi.BehaviorArgsAsRequestPhaseBehaviorRequest(argsBehavior)
			result = append(result, behaviorRequest)
		} else if b.CaptureAttrs != nil {
			// Capture behavior
			captureAttrs := azionapi.NewBehaviorCaptureMatchGroupsAttributes(
				b.CaptureAttrs.Subject.ValueString(),
				b.CaptureAttrs.Regex.ValueString(),
				b.CaptureAttrs.CapturedArray.ValueString(),
			)
			captureBehavior := azionapi.NewBehaviorCapture(b.Type.ValueString(), *captureAttrs)
			behaviorRequest := azionapi.BehaviorCaptureAsRequestPhaseBehaviorRequest(captureBehavior)
			result = append(result, behaviorRequest)
		} else {
			// No args behavior
			noArgsBehavior := azionapi.NewBehaviorNoArgs(b.Type.ValueString())
			behaviorRequest := azionapi.BehaviorNoArgsAsRequestPhaseBehaviorRequest(noArgsBehavior)
			result = append(result, behaviorRequest)
		}
	}
	return result
}

func buildBehaviorsResponseV4(behaviors []RulesEngineBehaviorResourceModel) []azionapi.ResponsePhaseBehaviorRequest {
	var result []azionapi.ResponsePhaseBehaviorRequest
	for _, b := range behaviors {
		if b.Attributes != nil && !b.Attributes.Value.IsNull() {
			// Behavior with args
			value := azionapi.BehaviorArgsAttributesValue{
				String: b.Attributes.Value.ValueStringPointer(),
			}
			attrs := azionapi.NewBehaviorArgsAttributes(value)
			argsBehavior := azionapi.NewBehaviorArgs(b.Type.ValueString(), *attrs)
			behaviorRequest := azionapi.BehaviorArgsAsResponsePhaseBehaviorRequest(argsBehavior)
			result = append(result, behaviorRequest)
		} else if b.CaptureAttrs != nil {
			// Capture behavior
			captureAttrs := azionapi.NewBehaviorCaptureMatchGroupsAttributes(
				b.CaptureAttrs.Subject.ValueString(),
				b.CaptureAttrs.Regex.ValueString(),
				b.CaptureAttrs.CapturedArray.ValueString(),
			)
			captureBehavior := azionapi.NewBehaviorCapture(b.Type.ValueString(), *captureAttrs)
			behaviorRequest := azionapi.BehaviorCaptureAsResponsePhaseBehaviorRequest(captureBehavior)
			result = append(result, behaviorRequest)
		} else {
			// No args behavior
			noArgsBehavior := azionapi.NewBehaviorNoArgs(b.Type.ValueString())
			behaviorRequest := azionapi.BehaviorNoArgsAsResponsePhaseBehaviorRequest(noArgsBehavior)
			result = append(result, behaviorRequest)
		}
	}
	return result
}

func buildStateFromResponse(rule azionapi.RequestPhaseRule, applicationID types.Int64, phase types.String) RulesEngineResourceModel {
	result := transformRuleToResultsModel(rule, phase.ValueString())
	return RulesEngineResourceModel{
		ApplicationID: applicationID,
		ID:            types.StringValue(fmt.Sprintf("%d/%s/%d", applicationID.ValueInt64(), phase.ValueString(), result.ID.ValueInt64())),
		LastUpdated:   types.StringValue(time.Now().Format(time.RFC850)),
		RulesEngine:   result,
	}
}

func transformRuleToResultsModel(rule azionapi.RequestPhaseRule, phase string) *RulesEngineResourceResults {
	result := &RulesEngineResourceResults{
		ID:    types.Int64Value(rule.GetId()),
		Name:  types.StringValue(rule.GetName()),
		Phase: types.StringValue(phase),
		Order: types.Int64Value(rule.GetOrder()),
	}

	if rule.Active != nil {
		result.Active = types.BoolValue(*rule.Active)
	}
	if rule.Description != nil {
		result.Description = types.StringValue(*rule.Description)
	}
	if rule.LastEditor.IsSet() && rule.LastEditor.Get() != nil {
		result.LastEditor = types.StringValue(*rule.LastEditor.Get())
	}
	if rule.LastModified.IsSet() && rule.LastModified.Get() != nil {
		result.LastModified = types.StringValue(rule.LastModified.Get().Format(time.RFC3339))
	}
	// CreatedAt
	result.CreatedAt = types.StringValue(rule.GetCreatedAt().Format(time.RFC3339))

	// Transform criteria
	for _, criterionGroup := range rule.Criteria {
		var criterionSet []RulesEngineResourceCriteria
		for _, c := range criterionGroup {
			arg := getCriterionArgumentValue(c.Argument)
			var argValue types.String
			if arg == "" {
				argValue = types.StringNull()
			} else {
				argValue = types.StringValue(arg)
			}
			criterionSet = append(criterionSet, RulesEngineResourceCriteria{
				Conditional: types.StringValue(c.GetConditional()),
				Variable:    types.StringValue(c.GetVariable()),
				Operator:    types.StringValue(c.GetOperator()),
				Argument:    argValue,
			})
		}
		result.Criteria = append(result.Criteria, CriteriaResourceModel{
			Entries: criterionSet,
		})
	}

	// Transform behaviors
	for _, b := range rule.Behaviors {
		behavior := RulesEngineBehaviorResourceModel{}

		if b.BehaviorArgs != nil {
			behavior.Type = types.StringValue(b.BehaviorArgs.GetType())
			val := getBehaviorArgsValueV4(b.BehaviorArgs.Attributes.Value)
			behavior.Attributes = &BehaviorAttributesResourceModel{
				Value: types.StringValue(val),
			}
		} else if b.BehaviorCapture != nil {
			behavior.Type = types.StringValue(b.BehaviorCapture.GetType())
			behavior.CaptureAttrs = &CaptureAttributesResourceModel{
				Subject:       types.StringValue(b.BehaviorCapture.Attributes.GetSubject()),
				Regex:         types.StringValue(b.BehaviorCapture.Attributes.GetRegex()),
				CapturedArray: types.StringValue(b.BehaviorCapture.Attributes.GetCapturedArray()),
			}
		} else if b.BehaviorNoArgs != nil {
			behavior.Type = types.StringValue(b.BehaviorNoArgs.GetType())
		}
		result.Behaviors = append(result.Behaviors, behavior)
	}

	return result
}

func transformRuleToResultsModelFromResponse(rule azionapi.ResponsePhaseRule, phase string) *RulesEngineResourceResults {
	result := &RulesEngineResourceResults{
		ID:    types.Int64Value(rule.GetId()),
		Name:  types.StringValue(rule.GetName()),
		Phase: types.StringValue(phase),
		Order: types.Int64Value(rule.GetOrder()),
	}

	if rule.Active != nil {
		result.Active = types.BoolValue(*rule.Active)
	}
	if rule.Description != nil {
		result.Description = types.StringValue(*rule.Description)
	}
	if rule.LastEditor.IsSet() && rule.LastEditor.Get() != nil {
		result.LastEditor = types.StringValue(*rule.LastEditor.Get())
	}
	if rule.LastModified.IsSet() && rule.LastModified.Get() != nil {
		result.LastModified = types.StringValue(rule.LastModified.Get().Format(time.RFC3339))
	}
	// CreatedAt
	result.CreatedAt = types.StringValue(rule.GetCreatedAt().Format(time.RFC3339))

	// Transform criteria
	for _, criterionGroup := range rule.Criteria {
		var criterionSet []RulesEngineResourceCriteria
		for _, c := range criterionGroup {
			arg := getCriterionArgumentValue(c.Argument)
			var argValue types.String
			if arg == "" {
				argValue = types.StringNull()
			} else {
				argValue = types.StringValue(arg)
			}
			criterionSet = append(criterionSet, RulesEngineResourceCriteria{
				Conditional: types.StringValue(c.GetConditional()),
				Variable:    types.StringValue(c.GetVariable()),
				Operator:    types.StringValue(c.GetOperator()),
				Argument:    argValue,
			})
		}
		result.Criteria = append(result.Criteria, CriteriaResourceModel{
			Entries: criterionSet,
		})
	}

	// Transform behaviors
	for _, b := range rule.Behaviors {
		behavior := RulesEngineBehaviorResourceModel{}

		if b.BehaviorArgs != nil {
			behavior.Type = types.StringValue(b.BehaviorArgs.GetType())
			val := getBehaviorArgsValueV4(b.BehaviorArgs.Attributes.Value)
			behavior.Attributes = &BehaviorAttributesResourceModel{
				Value: types.StringValue(val),
			}
		} else if b.BehaviorCapture != nil {
			behavior.Type = types.StringValue(b.BehaviorCapture.GetType())
			behavior.CaptureAttrs = &CaptureAttributesResourceModel{
				Subject:       types.StringValue(b.BehaviorCapture.Attributes.GetSubject()),
				Regex:         types.StringValue(b.BehaviorCapture.Attributes.GetRegex()),
				CapturedArray: types.StringValue(b.BehaviorCapture.Attributes.GetCapturedArray()),
			}
		} else if b.BehaviorNoArgs != nil {
			behavior.Type = types.StringValue(b.BehaviorNoArgs.GetType())
		}
		result.Behaviors = append(result.Behaviors, behavior)
	}

	return result
}

func getBehaviorArgsValueV4(value azionapi.BehaviorArgsAttributesValue) string {
	if value.String != nil {
		return *value.String
	}
	if value.Int64 != nil {
		return fmt.Sprintf("%d", *value.Int64)
	}
	return ""
}

func getCriterionArgumentValue(arg azionapi.NullableApplicationCriterionArgument) string {
	if !arg.IsSet() {
		return ""
	}
	argValue := arg.Get()
	if argValue == nil {
		return ""
	}
	if argValue.String != nil {
		return *argValue.String
	}
	if argValue.Int64 != nil {
		return fmt.Sprintf("%d", *argValue.Int64)
	}
	return ""
}

func convertResponseToRequestPhaseRule(rule azionapi.ResponsePhaseRule) azionapi.RequestPhaseRule {
	// Convert criteria
	var criteria [][]azionapi.ApplicationCriterionField
	for _, group := range rule.Criteria {
		var criterionGroup []azionapi.ApplicationCriterionField
		for _, c := range group {
			criterionField := azionapi.NewApplicationCriterionField(
				c.GetConditional(),
				c.GetVariable(),
				c.GetOperator(),
			)
			if c.Argument.IsSet() {
				if arg := c.Argument.Get(); arg != nil {
					criterionField.SetArgument(*arg)
				}
			}
			criterionGroup = append(criterionGroup, *criterionField)
		}
		criteria = append(criteria, criterionGroup)
	}

	// Convert behaviors
	var behaviors []azionapi.RequestPhaseBehavior
	for _, b := range rule.Behaviors {
		if b.BehaviorArgs != nil {
			behaviors = append(behaviors, azionapi.BehaviorArgsAsRequestPhaseBehavior(b.BehaviorArgs))
		} else if b.BehaviorCapture != nil {
			behaviors = append(behaviors, azionapi.BehaviorCaptureAsRequestPhaseBehavior(b.BehaviorCapture))
		} else if b.BehaviorNoArgs != nil {
			behaviors = append(behaviors, azionapi.BehaviorNoArgsAsRequestPhaseBehavior(b.BehaviorNoArgs))
		}
	}

	result := azionapi.NewRequestPhaseRuleWithDefaults()
	result.SetId(rule.GetId())
	result.SetName(rule.GetName())
	result.SetCriteria(criteria)
	result.SetBehaviors(behaviors)

	if rule.Active != nil {
		result.SetActive(*rule.Active)
	}
	if rule.Description != nil {
		result.SetDescription(*rule.Description)
	}
	result.SetOrder(rule.GetOrder())
	if rule.LastEditor.IsSet() {
		if val := rule.LastEditor.Get(); val != nil {
			result.SetLastEditor(*val)
		}
	}
	if rule.LastModified.IsSet() {
		if val := rule.LastModified.Get(); val != nil {
			result.SetLastModified(*val)
		}
	}

	return *result
}

func handleResourceAPIError(resp interface{}, response *http.Response, err error) {
	switch r := resp.(type) {
	case *resource.CreateResponse:
		if response != nil && response.StatusCode == 429 {
			r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
		} else if response != nil {
			bodyBytes, _ := io.ReadAll(response.Body)
			r.Diagnostics.AddError(err.Error(), string(bodyBytes))
		} else {
			r.Diagnostics.AddError(err.Error(), "API request failed")
		}
	case *resource.ReadResponse:
		if response != nil && response.StatusCode == 429 {
			r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
		} else if response != nil {
			bodyBytes, _ := io.ReadAll(response.Body)
			r.Diagnostics.AddError(err.Error(), string(bodyBytes))
		} else {
			r.Diagnostics.AddError(err.Error(), "API request failed")
		}
	case *resource.UpdateResponse:
		if response != nil && response.StatusCode == 429 {
			r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
		} else if response != nil {
			bodyBytes, _ := io.ReadAll(response.Body)
			r.Diagnostics.AddError(err.Error(), string(bodyBytes))
		} else {
			r.Diagnostics.AddError(err.Error(), "API request failed")
		}
	case *resource.DeleteResponse:
		if response != nil && response.StatusCode == 429 {
			r.Diagnostics.AddError(err.Error(), "Rate limited. Please retry.")
		} else if response != nil {
			bodyBytes, _ := io.ReadAll(response.Body)
			r.Diagnostics.AddError(err.Error(), string(bodyBytes))
		} else {
			r.Diagnostics.AddError(err.Error(), "API request failed")
		}
	}
}
