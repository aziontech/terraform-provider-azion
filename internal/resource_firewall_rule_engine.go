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
	_ resource.Resource                = &firewallRuleEngineResource{}
	_ resource.ResourceWithConfigure   = &firewallRuleEngineResource{}
	_ resource.ResourceWithImportState = &firewallRuleEngineResource{}
)

func NewFirewallRuleEngineResource() resource.Resource {
	return &firewallRuleEngineResource{}
}

type firewallRuleEngineResource struct {
	client *apiClient
}

type FirewallRuleEngineResourceModel struct {
	ID          types.String                      `tfsdk:"id"`
	FirewallID  types.Int64                       `tfsdk:"firewall_id"`
	LastUpdated types.String                      `tfsdk:"last_updated"`
	Results     *FirewallRuleEngineResultResource `tfsdk:"results"`
}

type FirewallRuleEngineResultResource struct {
	ID           types.Int64                            `tfsdk:"id"`
	Name         types.String                           `tfsdk:"name"`
	Active       types.Bool                             `tfsdk:"active"`
	Criteria     []FirewallCriteriaResourceModel        `tfsdk:"criteria"`
	Behaviors    []FirewallBehaviorWrapperResourceModel `tfsdk:"behaviors"`
	Description  types.String                           `tfsdk:"description"`
	Order        types.Int64                            `tfsdk:"order"`
	LastEditor   types.String                           `tfsdk:"last_editor"`
	LastModified types.String                           `tfsdk:"last_modified"`
	CreatedAt    types.String                           `tfsdk:"created_at"`
}

type FirewallCriteriaResourceModel struct {
	Entries []FirewallCriterionWrapperResourceModel `tfsdk:"entries"`
}

type FirewallCriterionWrapperResourceModel struct {
	Criterion *FirewallCriteriaEntryResourceModel `tfsdk:"criterion"`
}

type FirewallCriteriaEntryResourceModel struct {
	Conditional types.String `tfsdk:"conditional"`
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Argument    types.String `tfsdk:"argument"`
}

type FirewallBehaviorWrapperResourceModel struct {
	Behavior *FirewallBehaviorResourceModel `tfsdk:"behavior"`
}

type FirewallBehaviorResourceModel struct {
	Type       types.String                        `tfsdk:"type"`
	Attributes *FirewallBehaviorAttrsResourceModel `tfsdk:"attributes"`
}

type FirewallBehaviorAttrsResourceModel struct {
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

func (r *firewallRuleEngineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rule_engine"
}

func (r *firewallRuleEngineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall identifier.",
				Required:    true,
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
					"active": schema.BoolAttribute{
						Description: "Whether the rule is active.",
						Optional:    true,
						Computed:    true,
					},
					"criteria": schema.ListNestedAttribute{
						Description: "Criteria for the rule.",
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"entries": schema.ListNestedAttribute{
									Required: true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"criterion": schema.SingleNestedAttribute{
												Description: "A single criterion entry.",
												Required:    true,
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
						},
					},
					"behaviors": schema.ListNestedAttribute{
						Description: "Behaviors for the rule.",
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"behavior": schema.SingleNestedAttribute{
									Description: "A single behavior to apply on this rule.",
									Required:    true,
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Description: "Type of behavior (e.g., run_function, set_custom_response, set_waf, set_rate_limit, drop).",
											Required:    true,
										},
										"attributes": schema.SingleNestedAttribute{
											Description: "Behavior attributes (depends on behavior type).",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"value": schema.Int64Attribute{
													Description: "Value for run_function behavior (function instance ID).",
													Optional:    true,
												},
												"status_code": schema.Int64Attribute{
													Description: "Status code for set_custom_response behavior.",
													Optional:    true,
												},
												"content_type": schema.StringAttribute{
													Description: "Content type for set_custom_response behavior.",
													Optional:    true,
												},
												"content_body": schema.StringAttribute{
													Description: "Content body for set_custom_response behavior.",
													Optional:    true,
												},
												"waf_id": schema.Int64Attribute{
													Description: "WAF ID for set_waf behavior.",
													Optional:    true,
												},
												"mode": schema.StringAttribute{
													Description: "Mode for set_waf behavior (logging or blocking).",
													Optional:    true,
												},
												"type": schema.StringAttribute{
													Description: "Type for set_rate_limit behavior (second or minute).",
													Optional:    true,
												},
												"limit_by": schema.StringAttribute{
													Description: "Limit by for set_rate_limit behavior (client_ip or global).",
													Optional:    true,
												},
												"average_rate_limit": schema.Int64Attribute{
													Description: "Average rate limit for set_rate_limit behavior.",
													Optional:    true,
												},
												"maximum_burst_size": schema.Int64Attribute{
													Description: "Maximum burst size for set_rate_limit behavior.",
													Optional:    true,
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
						Optional:    true,
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
						Description: "Creation timestamp.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *firewallRuleEngineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *firewallRuleEngineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallRuleEngineResourceModel
	var firewallID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build criteria
	criteria := buildFirewallCriteriaRequest(plan.Results.Criteria)

	// Build behaviors
	behaviors := buildFirewallBehaviorsRequest(plan.Results.Behaviors)

	// Create rule request
	ruleRequest := azionapi.NewFirewallRuleRequest(
		plan.Results.Name.ValueString(),
		criteria,
		behaviors,
	)

	if !plan.Results.Active.IsNull() && !plan.Results.Active.IsUnknown() {
		ruleRequest.SetActive(plan.Results.Active.ValueBool())
	}
	if !plan.Results.Description.IsNull() && !plan.Results.Description.IsUnknown() {
		ruleRequest.SetDescription(plan.Results.Description.ValueString())
	}

	// Create the rule
	ruleResponse, response, err := r.client.api.FirewallsRulesEngineAPI.
		CreateFirewallRule(ctx, firewallID.ValueInt64()).
		FirewallRuleRequest(*ruleRequest).
		Execute()
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			ruleResponse, response, err = utils.RetryOn429(func() (*azionapi.FirewallRuleResponse, *http.Response, error) {
				return r.client.api.FirewallsRulesEngineAPI.
					CreateFirewallRule(ctx, firewallID.ValueInt64()).
					FirewallRuleRequest(*ruleRequest).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			handleFirewallRuleResourceAPIError(resp, response, err)
			return
		}
	}

	plan = buildFirewallRuleStateFromResponse(ruleResponse.Data, firewallID)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallRuleEngineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallRuleEngineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var firewallID int64
	var ruleID int64
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) == 2 {
		firewallID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		ruleID = int64(utils.AtoiNoError(valueFromCmd[1], resp))
	} else if len(valueFromCmd) == 1 {
		firewallID = state.FirewallID.ValueInt64()
		ruleID = state.Results.ID.ValueInt64()
	} else {
		resp.Diagnostics.AddError(
			"Parameters error",
			"you need to pass <firewallID>/<ruleID>",
		)
		return
	}

	if ruleID == 0 {
		resp.Diagnostics.AddError(
			"Rule ID error",
			"rule ID cannot be 0",
		)
		return
	}

	ruleResponse, response, err := r.client.api.FirewallsRulesEngineAPI.
		RetrieveFirewallRule(ctx, firewallID, ruleID).
		Execute()
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			ruleResponse, response, err = utils.RetryOn429(func() (*azionapi.FirewallRuleResponse, *http.Response, error) {
				return r.client.api.FirewallsRulesEngineAPI.
					RetrieveFirewallRule(ctx, firewallID, ruleID).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			handleFirewallRuleResourceAPIError(resp, response, err)
			return
		}
	}

	state = buildFirewallRuleStateFromResponse(ruleResponse.Data, types.Int64Value(firewallID))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallRuleEngineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallRuleEngineResourceModel
	var firewallID types.Int64
	var ruleID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallRuleEngineResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FirewallID.IsNull() {
		firewallID = state.FirewallID
	} else {
		firewallID = plan.FirewallID
	}

	if plan.Results.ID.IsNull() || plan.Results.ID.ValueInt64() == 0 {
		ruleID = state.Results.ID
	} else {
		ruleID = plan.Results.ID
	}

	// Build criteria
	criteria := buildFirewallCriteriaRequest(plan.Results.Criteria)

	// Build behaviors
	behaviors := buildFirewallBehaviorsRequest(plan.Results.Behaviors)

	// Create rule request for update
	ruleRequest := azionapi.NewFirewallRuleRequest(
		plan.Results.Name.ValueString(),
		criteria,
		behaviors,
	)

	if !plan.Results.Active.IsNull() && !plan.Results.Active.IsUnknown() {
		ruleRequest.SetActive(plan.Results.Active.ValueBool())
	}
	if !plan.Results.Description.IsNull() && !plan.Results.Description.IsUnknown() {
		ruleRequest.SetDescription(plan.Results.Description.ValueString())
	}

	// Update the rule
	ruleResponse, response, err := r.client.api.FirewallsRulesEngineAPI.
		UpdateFirewallRule(ctx, firewallID.ValueInt64(), ruleID.ValueInt64()).
		FirewallRuleRequest(*ruleRequest).
		Execute()
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			ruleResponse, response, err = utils.RetryOn429(func() (*azionapi.FirewallRuleResponse, *http.Response, error) {
				return r.client.api.FirewallsRulesEngineAPI.
					UpdateFirewallRule(ctx, firewallID.ValueInt64(), ruleID.ValueInt64()).
					FirewallRuleRequest(*ruleRequest).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			handleFirewallRuleResourceAPIError(resp, response, err)
			return
		}
	}

	plan = buildFirewallRuleStateFromResponse(ruleResponse.Data, firewallID)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *firewallRuleEngineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallRuleEngineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.FirewallID.IsNull() {
		resp.Diagnostics.AddError(
			"Firewall ID error",
			"Firewall ID cannot be null",
		)
		return
	}

	if state.Results.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Rule ID error",
			"Rule ID cannot be null",
		)
		return
	}

	_, response, err := utils.RetryOn429Delete(func() (interface{}, *http.Response, error) {
		_, httpResp, e := r.client.api.FirewallsRulesEngineAPI.
			DeleteFirewallRule(ctx, state.FirewallID.ValueInt64(), state.Results.ID.ValueInt64()).
			Execute()
		return nil, httpResp, e
	}, 5)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode != http.StatusNotFound {
			handleFirewallRuleResourceAPIError(resp, response, err)
			return
		}
	}
}

func (r *firewallRuleEngineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Expected format: {firewall_id}/{rule_id}",
		)
		return
	}

	firewallID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid firewall ID",
			"Could not parse firewall ID",
		)
		return
	}

	ruleID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid rule ID",
			"Could not parse rule ID",
		)
		return
	}

	var state FirewallRuleEngineResourceModel
	state.FirewallID = types.Int64Value(firewallID)
	state.Results = &FirewallRuleEngineResultResource{
		ID: types.Int64Value(ruleID),
	}
	state.ID = types.StringValue(fmt.Sprintf("%d/%d", firewallID, ruleID))

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Helper functions for Firewall Rules Engine

func buildFirewallCriteriaRequest(criteria []FirewallCriteriaResourceModel) [][]azionapi.FirewallCriterionFieldRequest {
	var result [][]azionapi.FirewallCriterionFieldRequest
	for _, criterion := range criteria {
		var criterionGroup []azionapi.FirewallCriterionFieldRequest
		for _, entry := range criterion.Entries {
			if entry.Criterion == nil {
				continue
			}
			c := entry.Criterion
			criterionField := azionapi.NewFirewallCriterionFieldRequest(
				c.Conditional.ValueString(),
				c.Variable.ValueString(),
				c.Operator.ValueString(),
			)
			if !c.Argument.IsNull() && !c.Argument.IsUnknown() {
				arg := azionapi.FirewallCriterionArgumentRequest{
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

func buildFirewallBehaviorsRequest(behaviors []FirewallBehaviorWrapperResourceModel) []azionapi.FirewallBehaviorRequest {
	var result []azionapi.FirewallBehaviorRequest
	for _, wrapper := range behaviors {
		if wrapper.Behavior == nil {
			continue
		}
		b := wrapper.Behavior
		behaviorType := b.Type.ValueString()

		// Check if it's a behavior without arguments (like "drop")
		if b.Attributes == nil {
			noArgsBehavior := azionapi.NewFirewallBehaviorNoArgsRequest(behaviorType)
			behaviorRequest := azionapi.FirewallBehaviorNoArgsRequestAsFirewallBehaviorRequest(noArgsBehavior)
			result = append(result, behaviorRequest)
			continue
		}

		// Handle behaviors with arguments based on type
		switch behaviorType {
		case "run_function":
			attrs := azionapi.NewFirewallBehaviorRunFunctionAttributesRequest(b.Attributes.Value.ValueInt64())
			argsBehavior := azionapi.NewFirewallBehaviorArgsRequest(behaviorType, *attrs)
			behaviorRequest := azionapi.FirewallBehaviorArgsRequestAsFirewallBehaviorRequest(argsBehavior)
			result = append(result, behaviorRequest)

		case "set_custom_response":
			attrs := azionapi.NewFirewallBehaviorSetCustomResponseAttributesRequest(
				b.Attributes.StatusCode.ValueInt64(),
			)
			if !b.Attributes.ContentType.IsNull() && !b.Attributes.ContentType.IsUnknown() {
				attrs.SetContentType(b.Attributes.ContentType.ValueString())
			}
			if !b.Attributes.ContentBody.IsNull() && !b.Attributes.ContentBody.IsUnknown() {
				attrs.SetContentBody(b.Attributes.ContentBody.ValueString())
			}
			objAttrs := azionapi.FirewallBehaviorSetCustomResponseAttributesRequestAsFirewallBehaviorObjectArgsRequestAttributes(attrs)
			objectArgsBehavior := azionapi.NewFirewallBehaviorObjectArgsRequest(behaviorType, objAttrs)
			behaviorRequest := azionapi.FirewallBehaviorObjectArgsRequestAsFirewallBehaviorRequest(objectArgsBehavior)
			result = append(result, behaviorRequest)

		case "set_waf":
			attrs := azionapi.NewFirewallBehaviorSetWafAttributesRequest(
				b.Attributes.WafId.ValueInt64(),
				b.Attributes.Mode.ValueString(),
			)
			objAttrs := azionapi.FirewallBehaviorSetWafAttributesRequestAsFirewallBehaviorObjectArgsRequestAttributes(attrs)
			objectArgsBehavior := azionapi.NewFirewallBehaviorObjectArgsRequest(behaviorType, objAttrs)
			behaviorRequest := azionapi.FirewallBehaviorObjectArgsRequestAsFirewallBehaviorRequest(objectArgsBehavior)
			result = append(result, behaviorRequest)

		case "set_rate_limit":
			attrs := azionapi.NewFirewallBehaviorSetRateLimitAttributesRequest(
				b.Attributes.LimitBy.ValueString(),
				b.Attributes.AverageRateLimit.ValueInt64(),
			)
			if !b.Attributes.Type.IsNull() && !b.Attributes.Type.IsUnknown() {
				attrs.SetType(b.Attributes.Type.ValueString())
			}
			if !b.Attributes.MaximumBurstSize.IsNull() && !b.Attributes.MaximumBurstSize.IsUnknown() {
				maxBurstVal := b.Attributes.MaximumBurstSize.ValueInt64()
				attrs.SetMaximumBurstSize(maxBurstVal)
			}
			objAttrs := azionapi.FirewallBehaviorSetRateLimitAttributesRequestAsFirewallBehaviorObjectArgsRequestAttributes(attrs)
			objectArgsBehavior := azionapi.NewFirewallBehaviorObjectArgsRequest(behaviorType, objAttrs)
			behaviorRequest := azionapi.FirewallBehaviorObjectArgsRequestAsFirewallBehaviorRequest(objectArgsBehavior)
			result = append(result, behaviorRequest)

		default:
			// For unknown behavior types, treat as no-args behavior
			noArgsBehavior := azionapi.NewFirewallBehaviorNoArgsRequest(behaviorType)
			behaviorRequest := azionapi.FirewallBehaviorNoArgsRequestAsFirewallBehaviorRequest(noArgsBehavior)
			result = append(result, behaviorRequest)
		}
	}
	return result
}

func buildFirewallRuleStateFromResponse(rule azionapi.FirewallRule, firewallID types.Int64) FirewallRuleEngineResourceModel {
	result := transformFirewallRuleToResultModel(rule)
	return FirewallRuleEngineResourceModel{
		FirewallID:  firewallID,
		ID:          types.StringValue(fmt.Sprintf("%d/%d", firewallID.ValueInt64(), result.ID.ValueInt64())),
		LastUpdated: types.StringValue(time.Now().Format(time.RFC850)),
		Results:     result,
	}
}

func transformFirewallRuleToResultModel(rule azionapi.FirewallRule) *FirewallRuleEngineResultResource {
	result := &FirewallRuleEngineResultResource{
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
		var criterionSet []FirewallCriterionWrapperResourceModel
		for _, c := range criterionGroup {
			arg := getFirewallCriterionArgumentValue(c.Argument)
			var argValue types.String
			if arg == "" {
				argValue = types.StringNull()
			} else {
				argValue = types.StringValue(arg)
			}
			criterionSet = append(criterionSet, FirewallCriterionWrapperResourceModel{
				Criterion: &FirewallCriteriaEntryResourceModel{
					Conditional: types.StringValue(c.GetConditional()),
					Variable:    types.StringValue(c.GetVariable()),
					Operator:    types.StringValue(c.GetOperator()),
					Argument:    argValue,
				},
			})
		}
		result.Criteria = append(result.Criteria, FirewallCriteriaResourceModel{
			Entries: criterionSet,
		})
	}

	// Transform behaviors.
	for _, b := range rule.Behaviors {
		behavior := FirewallBehaviorResourceModel{}

		if b.FirewallBehaviorArgs != nil {
			behavior.Type = types.StringValue(b.FirewallBehaviorArgs.GetType())
			attrs := transformFirewallBehaviorArgsAttrs(b.FirewallBehaviorArgs.Attributes)
			behavior.Attributes = &attrs
		} else if b.FirewallBehaviorNoArgs != nil {
			behavior.Type = types.StringValue(b.FirewallBehaviorNoArgs.GetType())
		} else if b.FirewallBehaviorObjectArgs != nil {
			behavior.Type = types.StringValue(b.FirewallBehaviorObjectArgs.GetType())
			attrs := transformFirewallBehaviorObjectAttrs(b.FirewallBehaviorObjectArgs.Attributes)
			behavior.Attributes = &attrs
		}
		result.Behaviors = append(result.Behaviors, FirewallBehaviorWrapperResourceModel{
			Behavior: &behavior,
		})
	}

	return result
}

func getFirewallCriterionArgumentValue(arg azionapi.NullableFirewallCriterionArgument) string {
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

func transformFirewallBehaviorArgsAttrs(attrs azionapi.FirewallBehaviorRunFunctionAttributes) FirewallBehaviorAttrsResourceModel {
	return FirewallBehaviorAttrsResourceModel{
		Value: types.Int64Value(attrs.GetValue()),
	}
}

func transformFirewallBehaviorObjectAttrs(attrs azionapi.FirewallBehaviorObjectArgsAttributes) FirewallBehaviorAttrsResourceModel {
	result := FirewallBehaviorAttrsResourceModel{}

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

func handleFirewallRuleResourceAPIError(resp interface{}, response *http.Response, err error) {
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
