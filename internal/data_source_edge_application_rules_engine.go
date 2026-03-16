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
	ID            types.String             `tfsdk:"id"`
	ApplicationID types.Int64              `tfsdk:"edge_application_id"`
	Counter       types.Int64              `tfsdk:"counter"`
	TotalPages    types.Int64              `tfsdk:"total_pages"`
	Page          types.Int64              `tfsdk:"page"`
	PageSize      types.Int64              `tfsdk:"page_size"`
	Links         *LinksModel              `tfsdk:"links"`
	Results       []RulesEngineResultModel `tfsdk:"results"`
}

type LinksModel struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type RulesEngineResultModel struct {
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

type RulesEngineCriterionModel struct {
	Conditional types.String `tfsdk:"conditional"`
	Variable    types.String `tfsdk:"variable"`
	Operator    types.String `tfsdk:"operator"`
	Argument    types.String `tfsdk:"argument"`
}

type RulesEngineBehaviorModel struct {
	Type         types.String                   `tfsdk:"type"`
	Attributes   *RulesEngineBehaviorAttrsModel `tfsdk:"attributes"`
	CaptureAttrs *RulesEngineCaptureAttrsModel  `tfsdk:"capture_attributes"`
}

type RulesEngineBehaviorAttrsModel struct {
	Value types.String `tfsdk:"value"`
}

type RulesEngineCaptureAttrsModel struct {
	Subject       types.String `tfsdk:"subject"`
	Regex         types.String `tfsdk:"regex"`
	CapturedArray types.String `tfsdk:"captured_array"`
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
		},
	}
}

func (r *RulesEngineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var applicationID types.Int64
	var phase types.String
	var page types.Int64
	var pageSize types.Int64

	diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
	resp.Diagnostics.Append(diagsApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get phase from the first result element
	diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtListIndex(0).AtName("phase"), &phase)
	resp.Diagnostics.Append(diagsPhase...)
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

	phaseStr := phase.ValueString()
	var result RulesEngineDataSourceModel
	var response *http.Response
	var err error

	switch phaseStr {
	case "request":
		result, response, err = r.listRequestRules(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
	case "response":
		result, response, err = r.listResponseRules(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
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
				result, response, err = r.listRequestRulesWithRetry(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
			case "response":
				result, response, err = r.listResponseRulesWithRetry(ctx, applicationID.ValueInt64(), page.ValueInt64(), pageSize.ValueInt64(), phaseStr)
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
			return
		} else {
			resp.Diagnostics.AddError(err.Error(), "API request failed")
			return
		}
	}

	result.ApplicationID = applicationID
	result.ID = types.StringValue("Get All Edge Application Rules Engine")

	diags := resp.State.Set(ctx, &result)
	resp.Diagnostics.Append(diags...)
}

func (r *RulesEngineDataSource) listRequestRules(ctx context.Context, applicationID, page, pageSize int64, phase string) (RulesEngineDataSourceModel, *http.Response, error) {
	listReq := r.client.api.ApplicationsRequestRulesAPI.
		ListApplicationRequestRules(ctx, applicationID).
		Page(page).
		PageSize(pageSize)

	listResponse, response, err := listReq.Execute()
	if err != nil {
		return RulesEngineDataSourceModel{}, response, err
	}

	return transformPaginatedRequestPhaseRuleList(listResponse, phase), response, nil
}

func (r *RulesEngineDataSource) listRequestRulesWithRetry(ctx context.Context, applicationID, page, pageSize int64, phase string) (RulesEngineDataSourceModel, *http.Response, error) {
	listResponse, response, err := utils.RetryOn429(func() (*azionapi.PaginatedRequestPhaseRuleList, *http.Response, error) {
		return r.client.api.ApplicationsRequestRulesAPI.
			ListApplicationRequestRules(ctx, applicationID).
			Page(page).
			PageSize(pageSize).
			Execute()
	}, 5)
	if err != nil {
		return RulesEngineDataSourceModel{}, response, err
	}

	return transformPaginatedRequestPhaseRuleList(listResponse, phase), response, nil
}

func (r *RulesEngineDataSource) listResponseRules(ctx context.Context, applicationID, page, pageSize int64, phase string) (RulesEngineDataSourceModel, *http.Response, error) {
	listReq := r.client.api.ApplicationsResponseRulesAPI.
		ListApplicationResponseRules(ctx, applicationID).
		Page(page).
		PageSize(pageSize)

	listResponse, response, err := listReq.Execute()
	if err != nil {
		return RulesEngineDataSourceModel{}, response, err
	}

	return transformPaginatedResponsePhaseRuleList(listResponse, phase), response, nil
}

func (r *RulesEngineDataSource) listResponseRulesWithRetry(ctx context.Context, applicationID, page, pageSize int64, phase string) (RulesEngineDataSourceModel, *http.Response, error) {
	listResponse, response, err := utils.RetryOn429(func() (*azionapi.PaginatedResponsePhaseRuleList, *http.Response, error) {
		return r.client.api.ApplicationsResponseRulesAPI.
			ListApplicationResponseRules(ctx, applicationID).
			Page(page).
			PageSize(pageSize).
			Execute()
	}, 5)
	if err != nil {
		return RulesEngineDataSourceModel{}, response, err
	}

	return transformPaginatedResponsePhaseRuleList(listResponse, phase), response, nil
}

func transformPaginatedRequestPhaseRuleList(list *azionapi.PaginatedRequestPhaseRuleList, phase string) RulesEngineDataSourceModel {
	result := RulesEngineDataSourceModel{
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
		result.Results = append(result.Results, transformRequestPhaseRuleToListResult(rule, phase))
	}

	return result
}

func transformPaginatedResponsePhaseRuleList(list *azionapi.PaginatedResponsePhaseRuleList, phase string) RulesEngineDataSourceModel {
	result := RulesEngineDataSourceModel{
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
		result.Results = append(result.Results, transformResponsePhaseRuleToListResult(rule, phase))
	}

	return result
}

func transformRequestPhaseRuleToListResult(rule azionapi.RequestPhaseRule, phase string) RulesEngineResultModel {
	result := RulesEngineResultModel{
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
			val := getRulesBehaviorArgsValue(b.BehaviorArgs.Attributes.Value)
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

func transformResponsePhaseRuleToListResult(rule azionapi.ResponsePhaseRule, phase string) RulesEngineResultModel {
	result := RulesEngineResultModel{
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
			val := getRulesBehaviorArgsValue(b.BehaviorArgs.Attributes.Value)
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

func getRulesBehaviorArgsValue(value azionapi.BehaviorArgsAttributesValue) string {
	if value.String != nil {
		return *value.String
	}
	if value.Int64 != nil {
		return fmt.Sprintf("%d", *value.Int64)
	}
	return ""
}
