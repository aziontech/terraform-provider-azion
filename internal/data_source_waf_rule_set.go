package provider

import (
	"context"
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
	_ datasource.DataSource              = &WafRuleSetDataSource{}
	_ datasource.DataSourceWithConfigure = &WafRuleSetDataSource{}
)

func dataSourceAzionWafRuleSet() datasource.DataSource {
	return &WafRuleSetDataSource{}
}

type WafRuleSetDataSource struct {
	client *apiClient
}

type WafRuleSetDataSourceModel struct {
	ID          types.String               `tfsdk:"id"`
	WafID       types.Int64                `tfsdk:"waf_id"`
	ExceptionID types.Int64                `tfsdk:"exception_id"`
	Results     *WafRuleSetResultDataModel `tfsdk:"results"`
}

type WafRuleSetResultDataModel struct {
	ID           types.Int64                         `tfsdk:"id"`
	RuleID       types.Int64                         `tfsdk:"rule_id"`
	Name         types.String                        `tfsdk:"name"`
	Path         types.String                        `tfsdk:"path"`
	Conditions   []WafExceptionConditionWrapperModel `tfsdk:"conditions"`
	Operator     types.String                        `tfsdk:"operator"`
	Active       types.Bool                          `tfsdk:"active"`
	LastEditor   types.String                        `tfsdk:"last_editor"`
	LastModified types.String                        `tfsdk:"last_modified"`
}

// WafExceptionConditionWrapperModel wraps a single condition under a `condition` label.
type WafExceptionConditionWrapperModel struct {
	Condition *WafExceptionConditionModel `tfsdk:"condition"`
}

// WafExceptionConditionModel represents a polymorphic condition.
type WafExceptionConditionModel struct {
	// Common field for all condition types
	Match types.String `tfsdk:"match"`
	// For specific condition on name
	Name types.String `tfsdk:"name"`
	// For specific condition on value
	Value types.String `tfsdk:"value"`
	// Type indicator: "generic", "specific_on_name", or "specific_on_value"
	ConditionType types.String `tfsdk:"condition_type"`
}

func (o *WafRuleSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafRuleSetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_set"
}

func (o *WafRuleSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"waf_id": schema.Int64Attribute{
				Description: "The WAF identifier.",
				Required:    true,
			},
			"exception_id": schema.Int64Attribute{
				Description: "The WAF exception (rule set) identifier.",
				Required:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The ID of the WAF exception.",
						Computed:    true,
					},
					"rule_id": schema.Int64Attribute{
						Description: "The rule ID that this exception applies to. 0 means all rules.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the WAF exception.",
						Computed:    true,
					},
					"path": schema.StringAttribute{
						Description: "Path pattern for the exception.",
						Computed:    true,
					},
					"conditions": schema.ListNestedAttribute{
						Description: "Conditions for the WAF exception.",
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"condition": schema.SingleNestedAttribute{
									Description: "A single condition for the WAF exception.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"match": schema.StringAttribute{
											Description: "The match type for the condition.",
											Computed:    true,
										},
										"name": schema.StringAttribute{
											Description: "The name for specific condition on name.",
											Computed:    true,
										},
										"value": schema.StringAttribute{
											Description: "The value for specific condition on value.",
											Computed:    true,
										},
										"condition_type": schema.StringAttribute{
											Description: "Type of condition: generic, specific_on_name, or specific_on_value.",
											Computed:    true,
										},
									},
								},
							},
						},
					},
					"operator": schema.StringAttribute{
						Description: "The operator for the exception (regex or contains).",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the exception is active.",
						Computed:    true,
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

func (o *WafRuleSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var wafID, exceptionID types.Int64

	diagsWafID := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
	resp.Diagnostics.Append(diagsWafID...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsExceptionID := req.Config.GetAttribute(ctx, path.Root("exception_id"), &exceptionID)
	resp.Diagnostics.Append(diagsExceptionID...)
	if resp.Diagnostics.HasError() {
		return
	}

	exceptionResponse, response, err := o.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID.ValueInt64(), wafID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			exceptionResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFRuleResponse, *http.Response, error) {
				return o.client.api.WAFsExceptionsAPI.RetrieveWafException(ctx, exceptionID.ValueInt64(), wafID.ValueInt64()).Execute()
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

	// Transform the response to the model
	results := transformWAFRuleToResultModel(exceptionResponse.GetData())

	state := WafRuleSetDataSourceModel{
		ID:          types.StringValue("Get WAF Rule Set By ID"),
		WafID:       wafID,
		ExceptionID: exceptionID,
		Results:     results,
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// transformWAFRuleToResultModel transforms an SDK WAFRule to a Terraform model.
func transformWAFRuleToResultModel(rule azionapi.WAFRule) *WafRuleSetResultDataModel {
	result := &WafRuleSetResultDataModel{
		ID:           types.Int64Value(rule.GetId()),
		Name:         types.StringValue(rule.GetName()),
		LastEditor:   types.StringValue(rule.GetLastEditor()),
		LastModified: types.StringValue(rule.GetLastModified().Format(time.RFC3339)),
	}

	// Optional rule_id
	if rule.HasRuleId() {
		result.RuleID = types.Int64Value(rule.GetRuleId())
	} else {
		result.RuleID = types.Int64Null()
	}

	// Optional path
	if rule.HasPath() {
		result.Path = types.StringValue(rule.GetPath())
	} else {
		result.Path = types.StringNull()
	}

	// Optional operator
	if rule.HasOperator() {
		result.Operator = types.StringValue(rule.GetOperator())
	} else {
		result.Operator = types.StringNull()
	}

	// Optional active
	if rule.HasActive() {
		result.Active = types.BoolValue(rule.GetActive())
	} else {
		result.Active = types.BoolNull()
	}

	// Transform conditions
	conditions := rule.GetConditions()
	result.Conditions = transformWAFExceptionConditions(conditions)

	return result
}

// transformWAFExceptionConditions transforms SDK conditions to Terraform wrapped models.
func transformWAFExceptionConditions(conditions []azionapi.WAFExceptionCondition) []WafExceptionConditionWrapperModel {
	var result []WafExceptionConditionWrapperModel

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

		result = append(result, WafExceptionConditionWrapperModel{
			Condition: &model,
		})
	}

	return result
}
