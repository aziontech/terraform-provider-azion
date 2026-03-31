package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
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
	_ resource.Resource                = &wafRuleSetResource{}
	_ resource.ResourceWithConfigure   = &wafRuleSetResource{}
	_ resource.ResourceWithImportState = &wafRuleSetResource{}
)

func WafRuleSetResource() resource.Resource {
	return &wafRuleSetResource{}
}

type wafRuleSetResource struct {
	client *apiClient
}

type WafRuleSetResourceModel struct {
	ID          types.String               `tfsdk:"id"`
	WafID       types.Int64                `tfsdk:"waf_id"`
	LastUpdated types.String               `tfsdk:"last_updated"`
	Result      *WafRuleSetResourceResults `tfsdk:"result"`
}

type WafRuleSetResourceResults struct {
	ID           types.Int64                  `tfsdk:"exception_id"`
	RuleID       types.Int64                  `tfsdk:"rule_id"`
	Name         types.String                 `tfsdk:"name"`
	Path         types.String                 `tfsdk:"path"`
	Conditions   []WafExceptionConditionModel `tfsdk:"conditions"`
	Operator     types.String                 `tfsdk:"operator"`
	Active       types.Bool                   `tfsdk:"active"`
	LastEditor   types.String                 `tfsdk:"last_editor"`
	LastModified types.String                 `tfsdk:"last_modified"`
}

func (r *wafRuleSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_set"
}

func (r *wafRuleSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"waf_id": schema.Int64Attribute{
				Description: "The WAF identifier.",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"result": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"exception_id": schema.Int64Attribute{
						Description: "The ID of the WAF exception.",
						Computed:    true,
					},
					"rule_id": schema.Int64Attribute{
						Description: "The rule ID that this exception applies to. 0 means all rules.",
						Optional:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the WAF exception.",
						Required:    true,
					},
					"path": schema.StringAttribute{
						Description: "Path pattern for the exception.",
						Optional:    true,
					},
					"conditions": schema.ListNestedAttribute{
						Description: "Conditions for the WAF exception.",
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"match": schema.StringAttribute{
									Description: "The match type for the condition.",
									Required:    true,
								},
								"name": schema.StringAttribute{
									Description: "The name for specific condition on name.",
									Optional:    true,
								},
								"value": schema.StringAttribute{
									Description: "The value for specific condition on value.",
									Optional:    true,
								},
								"condition_type": schema.StringAttribute{
									Description: "Type of condition: generic, specific_on_name, or specific_on_value.",
									Required:    true,
								},
							},
						},
					},
					"operator": schema.StringAttribute{
						Description: "The operator for the exception (regex or contains).",
						Optional:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the exception is active.",
						Optional:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the exception.",
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

func (r *wafRuleSetResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *wafRuleSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WafRuleSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the conditions request.
	conditions, err := buildWAFExceptionConditionsRequest(plan.Result.Conditions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building conditions",
			err.Error(),
		)
		return
	}

	// Build the WAF exception request.
	wafRuleRequest := azionapi.NewWAFRuleRequest(plan.Result.Name.ValueString(), conditions)

	// Set optional fields.
	if !plan.Result.RuleID.IsNull() && !plan.Result.RuleID.IsUnknown() {
		wafRuleRequest.SetRuleId(plan.Result.RuleID.ValueInt64())
	}

	if !plan.Result.Path.IsNull() && !plan.Result.Path.IsUnknown() {
		wafRuleRequest.SetPath(plan.Result.Path.ValueString())
	}

	if !plan.Result.Operator.IsNull() && !plan.Result.Operator.IsUnknown() {
		wafRuleRequest.SetOperator(plan.Result.Operator.ValueString())
	}

	if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
		wafRuleRequest.SetActive(plan.Result.Active.ValueBool())
	}

	// Create the WAF exception.
	exceptionResponse, response, err := r.client.api.WAFsExceptionsAPI.CreateWafException(ctx, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			exceptionResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
				return r.client.api.WAFsExceptionsAPI.CreateWafException(ctx, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
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
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	// Transform the response to the model.
	data := exceptionResponse.GetData()
	plan.Result = transformWAFRuleToResourceModel(data)
	plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafRuleSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WafRuleSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var exceptionID int64
	var err error
	if state.ID.IsNull() {
		exceptionID = state.Result.ID.ValueInt64()
	} else {
		exceptionID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WAF Rule Set ID",
			)
			return
		}
	}

	exceptionResponse, response, err := r.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			exceptionResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
				return r.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
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
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	data := exceptionResponse.GetData()
	state.Result = transformWAFRuleToResourceModel(data)
	state.ID = types.StringValue(strconv.FormatInt(exceptionID, 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafRuleSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WafRuleSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WafRuleSetResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	var exceptionID int64
	var err error
	if state.ID.IsNull() {
		exceptionID = state.Result.ID.ValueInt64()
	} else {
		exceptionID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WAF Rule Set ID",
			)
			return
		}
	}

	// Build the conditions request.
	conditions, err := buildWAFExceptionConditionsRequest(plan.Result.Conditions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error building conditions",
			err.Error(),
		)
		return
	}

	// Build the WAF exception request.
	wafRuleRequest := azionapi.NewWAFRuleRequest(plan.Result.Name.ValueString(), conditions)

	// Set optional fields.
	if !plan.Result.RuleID.IsNull() && !plan.Result.RuleID.IsUnknown() {
		wafRuleRequest.SetRuleId(plan.Result.RuleID.ValueInt64())
	}

	if !plan.Result.Path.IsNull() && !plan.Result.Path.IsUnknown() {
		wafRuleRequest.SetPath(plan.Result.Path.ValueString())
	}

	if !plan.Result.Operator.IsNull() && !plan.Result.Operator.IsUnknown() {
		wafRuleRequest.SetOperator(plan.Result.Operator.ValueString())
	}

	if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
		wafRuleRequest.SetActive(plan.Result.Active.ValueBool())
	}

	// Update the WAF exception.
	exceptionResponse, response, err := r.client.api.WAFsExceptionsAPI.UpdateWafException(ctx, exceptionID, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			exceptionResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
				return r.client.api.WAFsExceptionsAPI.UpdateWafException(ctx, exceptionID, plan.WafID.ValueInt64()).WAFRuleRequest(*wafRuleRequest).Execute()
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
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	// Transform the response to the model.
	data := exceptionResponse.GetData()
	plan.Result = transformWAFRuleToResourceModel(data)
	plan.ID = types.StringValue(strconv.FormatInt(exceptionID, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafRuleSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WafRuleSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var exceptionID int64
	var err error
	if state.ID.IsNull() {
		exceptionID = state.Result.ID.ValueInt64()
	} else {
		exceptionID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WAF Rule Set ID",
			)
			return
		}
	}

	deleteResponse, response, err := r.client.api.WAFsExceptionsAPI.DeleteWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
				delResp, resp, err := r.client.api.WAFsExceptionsAPI.DeleteWafException(ctx, exceptionID, state.WafID.ValueInt64()).Execute()
				_ = delResp // Ignore the delete response in retry.
				return resp, err
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
			return
		}
	}

	// Close response body if not nil.
	if response != nil {
		defer response.Body.Close()
	}

	// Use deleteResponse to avoid unused variable error.
	_ = deleteResponse
}

func (r *wafRuleSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// transformWAFRuleToResourceModel transforms an SDK WAFRule to a Terraform resource model.
func transformWAFRuleToResourceModel(rule azionapi.WAFRule) *WafRuleSetResourceResults {
	result := &WafRuleSetResourceResults{
		ID:           types.Int64Value(rule.GetId()),
		Name:         types.StringValue(rule.GetName()),
		LastEditor:   types.StringValue(rule.GetLastEditor()),
		LastModified: types.StringValue(rule.GetLastModified().Format(time.RFC3339)),
	}

	// Optional rule_id.
	if rule.HasRuleId() {
		result.RuleID = types.Int64Value(rule.GetRuleId())
	} else {
		result.RuleID = types.Int64Null()
	}

	// Optional path.
	if rule.HasPath() {
		result.Path = types.StringValue(rule.GetPath())
	} else {
		result.Path = types.StringNull()
	}

	// Optional operator.
	if rule.HasOperator() {
		result.Operator = types.StringValue(rule.GetOperator())
	} else {
		result.Operator = types.StringNull()
	}

	// Optional active.
	if rule.HasActive() {
		result.Active = types.BoolValue(rule.GetActive())
	} else {
		result.Active = types.BoolNull()
	}

	// Transform conditions.
	conditions := rule.GetConditions()
	result.Conditions = transformWAFExceptionConditionsForResource(conditions)

	return result
}

// transformWAFExceptionConditionsForResource transforms SDK conditions to Terraform models for resources.
func transformWAFExceptionConditionsForResource(conditions []azionapi.WAFExceptionCondition) []WafExceptionConditionModel {
	var result []WafExceptionConditionModel

	for _, cond := range conditions {
		actualInstance := cond.GetActualInstance()
		if actualInstance == nil {
			continue
		}

		model := WafExceptionConditionModel{}

		switch c := actualInstance.(type) {
		case *azionapi.WAFExceptionGenericCondition:
			model.Match = types.StringValue(c.GetMatch())
			model.Name = types.StringNull()
			model.Value = types.StringNull()
			model.ConditionType = types.StringValue("generic")

		case *azionapi.WAFExceptionSpecificConditionOnName:
			model.Match = types.StringValue(c.GetMatch())
			model.Name = types.StringValue(c.GetName())
			model.Value = types.StringNull()
			model.ConditionType = types.StringValue("specific_on_name")

		case *azionapi.WAFExceptionSpecificConditionOnValue:
			model.Match = types.StringValue(c.GetMatch())
			model.Name = types.StringNull()
			model.Value = types.StringValue(c.GetValue())
			model.ConditionType = types.StringValue("specific_on_value")
		}

		result = append(result, model)
	}

	return result
}

// buildWAFExceptionConditionsRequest builds SDK conditions from Terraform models.
func buildWAFExceptionConditionsRequest(conditions []WafExceptionConditionModel) ([]azionapi.WAFExceptionConditionRequest, error) {
	var result []azionapi.WAFExceptionConditionRequest

	for _, c := range conditions {
		switch c.ConditionType.ValueString() {
		case "generic":
			generic := azionapi.NewWAFExceptionGenericConditionRequest(c.Match.ValueString())
			result = append(result, azionapi.WAFExceptionGenericConditionRequestAsWAFExceptionConditionRequest(generic))

		case "specific_on_name":
			specificName := azionapi.NewWAFExceptionSpecificConditionOnNameRequest(
				c.Match.ValueString(),
				c.Name.ValueString(),
			)
			result = append(result, azionapi.WAFExceptionSpecificConditionOnNameRequestAsWAFExceptionConditionRequest(specificName))

		case "specific_on_value":
			specificValue := azionapi.NewWAFExceptionSpecificConditionOnValueRequest(
				c.Match.ValueString(),
				c.Value.ValueString(),
			)
			result = append(result, azionapi.WAFExceptionSpecificConditionOnValueRequestAsWAFExceptionConditionRequest(specificValue))
		}
	}

	return result, nil
}
