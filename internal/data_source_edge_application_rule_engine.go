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
	_ datasource.DataSource              = &RuleEngineDataSource{}
	_ datasource.DataSourceWithConfigure = &RuleEngineDataSource{}
)

func dataSourceAzionApplicationRuleEngine() datasource.DataSource {
	return &RuleEngineDataSource{}
}

type RuleEngineDataSource struct {
	client *apiClient
}

type RuleEngineDataSourceModel struct {
	ID            types.String          `tfsdk:"id"`
	ApplicationID types.Int64           `tfsdk:"application_id"`
	Results       RuleEngineResultModel `tfsdk:"results"`
}

type RuleEngineResultModel struct {
	ID           types.Int64        `tfsdk:"id"`
	Name         types.String       `tfsdk:"name"`
	Phase        types.String       `tfsdk:"phase"`
	Active       types.Bool         `tfsdk:"active"`
	Criteria     [][]CriterionModel `tfsdk:"criteria"`
	Behaviors    []BehaviorModel    `tfsdk:"behaviors"`
	Description  types.String       `tfsdk:"description"`
	Order        types.Int64        `tfsdk:"order"`
	LastEditor   types.String       `tfsdk:"last_editor"`
	LastModified types.String       `tfsdk:"last_modified"`
}

type CriterionModel struct {
	Conditional types.String `tfsdk:"conditional"`
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Argument    types.String `tfsdk:"argument"`
}

type BehaviorModel struct {
	Type         types.String             `tfsdk:"type"`
	Attributes   *BehaviorAttributesModel `tfsdk:"attributes"`
	CaptureAttrs *CaptureAttributesModel  `tfsdk:"capture_attributes"`
}

type BehaviorAttributesModel struct {
	Value types.String `tfsdk:"value"`
}

type CaptureAttributesModel struct {
	Subject       types.String `tfsdk:"subject"`
	Regex         types.String `tfsdk:"regex"`
	CapturedArray types.String `tfsdk:"captured_array"`
}

func (r *RuleEngineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *RuleEngineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_rule_engine"
}

func (r *RuleEngineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"application_id": schema.Int64Attribute{
				Description: "The application identifier.",
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
					"phase": schema.StringAttribute{
						Description: "The phase in which the rule is executed (request or response).",
						Required:    true,
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
									Description: "Behavior attributes (for behaviors with args).",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"value": schema.StringAttribute{
											Description: "Value for the behavior.",
											Computed:    true,
										},
									},
								},
								"capture_attributes": schema.SingleNestedAttribute{
									Description: "Capture attributes (for capture_match_groups).",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"subject": schema.StringAttribute{
											Description: "Subject for capture.",
											Computed:    true,
										},
										"regex": schema.StringAttribute{
											Description: "Regex pattern.",
											Computed:    true,
										},
										"captured_array": schema.StringAttribute{
											Description: "Captured array name.",
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
				},
			},
		},
	}
}

func (r *RuleEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var applicationID types.Int64
	var ruleID types.Int64
	var phase types.String

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

	diagsRuleID := req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &ruleID)
	resp.Diagnostics.Append(diagsRuleID...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine which API to use based on phase
	phaseStr := phase.ValueString()
	var result RuleEngineResultModel
	var response *http.Response
	var err error

	switch phaseStr {
	case "request":
		result, response, err = r.readRequestRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
	case "response":
		result, response, err = r.readResponseRule(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
	default:
		resp.Diagnostics.AddError(
			"Invalid phase value",
			fmt.Sprintf("Phase must be 'request' or 'response', got: %s", phaseStr),
		)
		return
	}

	if err != nil {
		if response != nil && response.StatusCode == 429 {
			switch phaseStr {
			case "request":
				result, response, err = r.readRequestRuleWithRetry(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
			case "response":
				result, response, err = r.readResponseRuleWithRetry(ctx, applicationID.ValueInt64(), ruleID.ValueInt64(), phaseStr)
			}

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

	state := RuleEngineDataSourceModel{
		ApplicationID: applicationID,
		Results:       result,
	}
	state.ID = types.StringValue("Get By ID Application Rule Engine")

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RuleEngineDataSource) readRequestRule(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
	ruleResponse, response, err := r.client.api.ApplicationsRequestRulesAPI.
		RetrieveApplicationRequestRule(ctx, applicationID, ruleID).
		Execute()
	if err != nil {
		return RuleEngineResultModel{}, response, err
	}
	return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func (r *RuleEngineDataSource) readRequestRuleWithRetry(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
	ruleResponse, response, err := utils.RetryOn429(func() (*azionapi.ResponsePhaseRuleResponse, *http.Response, error) {
		return r.client.api.ApplicationsRequestRulesAPI.
			RetrieveApplicationRequestRule(ctx, applicationID, ruleID).
			Execute()
	}, 5)
	if err != nil {
		return RuleEngineResultModel{}, response, err
	}
	return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func (r *RuleEngineDataSource) readResponseRule(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
	ruleResponse, response, err := r.client.api.ApplicationsResponseRulesAPI.
		RetrieveApplicationResponseRule(ctx, applicationID, ruleID).
		Execute()
	if err != nil {
		return RuleEngineResultModel{}, response, err
	}
	return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func (r *RuleEngineDataSource) readResponseRuleWithRetry(ctx context.Context, applicationID, ruleID int64, phase string) (RuleEngineResultModel, *http.Response, error) {
	ruleResponse, response, err := utils.RetryOn429(func() (*azionapi.ResponsePhaseRuleResponse, *http.Response, error) {
		return r.client.api.ApplicationsResponseRulesAPI.
			RetrieveApplicationResponseRule(ctx, applicationID, ruleID).
			Execute()
	}, 5)
	if err != nil {
		return RuleEngineResultModel{}, response, err
	}
	return transformResponsePhaseRule(ruleResponse.Data, phase), response, nil
}

func transformResponsePhaseRule(rule azionapi.ResponsePhaseRule, phase string) RuleEngineResultModel {
	result := RuleEngineResultModel{
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
	if rule.LastEditor.Get() != nil {
		result.LastEditor = types.StringValue(*rule.LastEditor.Get())
	}
	if rule.LastModified.Get() != nil {
		result.LastModified = types.StringValue(rule.LastModified.Get().Format(time.RFC3339))
	}

	// Transform criteria
	for _, criterionGroup := range rule.Criteria {
		var group []CriterionModel
		for _, c := range criterionGroup {
			arg := ""
			if c.Argument.Get() != nil {
				arg = fmt.Sprintf("%v", c.Argument.Get())
			}
			group = append(group, CriterionModel{
				Conditional: types.StringValue(c.GetConditional()),
				Variable:    types.StringValue(c.GetVariable()),
				Operator:    types.StringValue(c.GetOperator()),
				Argument:    types.StringValue(arg),
			})
		}
		result.Criteria = append(result.Criteria, group)
	}

	// Transform behaviors
	for _, b := range rule.Behaviors {
		behavior := BehaviorModel{}

		if b.BehaviorArgs != nil {
			behavior.Type = types.StringValue(b.BehaviorArgs.GetType())
			val := getBehaviorArgsValue(b.BehaviorArgs.Attributes.Value)
			behavior.Attributes = &BehaviorAttributesModel{
				Value: types.StringValue(val),
			}
		} else if b.BehaviorCapture != nil {
			behavior.Type = types.StringValue(b.BehaviorCapture.GetType())
			behavior.CaptureAttrs = &CaptureAttributesModel{
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

func getBehaviorArgsValue(value azionapi.BehaviorArgsAttributesValue) string {
	if value.String != nil {
		return *value.String
	}
	if value.Int64 != nil {
		return fmt.Sprintf("%d", *value.Int64)
	}
	return ""
}
