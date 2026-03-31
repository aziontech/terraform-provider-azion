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
	_ datasource.DataSource              = &WafRuleSetsDataSource{}
	_ datasource.DataSourceWithConfigure = &WafRuleSetsDataSource{}
)

func dataSourceAzionWafRuleSets() datasource.DataSource {
	return &WafRuleSetsDataSource{}
}

type WafRuleSetsDataSource struct {
	client *apiClient
}

type WafRuleSetsDataSourceModel struct {
	ID         types.String                  `tfsdk:"id"`
	WafID      types.Int64                   `tfsdk:"waf_id"`
	Counter    types.Int64                   `tfsdk:"counter"`
	TotalPages types.Int64                   `tfsdk:"total_pages"`
	Page       types.Int64                   `tfsdk:"page"`
	PageSize   types.Int64                   `tfsdk:"page_size"`
	Links      *WafRuleSetsResponseLinks     `tfsdk:"links"`
	Results    []WafRuleSetListItemDataModel `tfsdk:"results"`
}

type WafRuleSetsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type WafRuleSetListItemDataModel struct {
	ID           types.Int64                  `tfsdk:"id"`
	RuleID       types.Int64                  `tfsdk:"rule_id"`
	Name         types.String                 `tfsdk:"name"`
	Path         types.String                 `tfsdk:"path"`
	Conditions   []WafExceptionConditionModel `tfsdk:"conditions"`
	Operator     types.String                 `tfsdk:"operator"`
	Active       types.Bool                   `tfsdk:"active"`
	LastEditor   types.String                 `tfsdk:"last_editor"`
	LastModified types.String                 `tfsdk:"last_modified"`
}

func (o *WafRuleSetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafRuleSetsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_sets"
}

func (o *WafRuleSetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"counter": schema.Int64Attribute{
				Description: "The total number of WAF exceptions.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number.",
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
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
		},
	}
}

func (o *WafRuleSetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var wafID, page, pageSize types.Int64

	diagsWafID := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
	resp.Diagnostics.Append(diagsWafID...)
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

	if page.IsNull() || page.IsUnknown() || page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.IsNull() || pageSize.IsUnknown() || pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	listResponse, response, err := o.client.api.WAFsExceptionsAPI.ListWafExceptions(ctx, wafID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedWAFRuleList, *http.Response, error) {
				return o.client.api.WAFsExceptionsAPI.ListWafExceptions(ctx, wafID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
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

	// Transform links
	var previous, next string
	if listResponse.HasPrevious() {
		previous = listResponse.GetPrevious()
	}
	if listResponse.HasNext() {
		next = listResponse.GetNext()
	}

	// Transform results
	var results []WafRuleSetListItemDataModel
	for _, rule := range listResponse.GetResults() {
		results = append(results, transformWAFRuleToListItemModel(rule))
	}

	state := WafRuleSetsDataSourceModel{
		ID:         types.StringValue("Get All WAF Rule Sets"),
		WafID:      wafID,
		Results:    results,
		TotalPages: types.Int64Value(listResponse.GetTotalPages()),
		Page:       page,
		PageSize:   pageSize,
		Counter:    types.Int64Value(listResponse.GetCount()),
		Links: &WafRuleSetsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// transformWAFRuleToListItemModel transforms an SDK WAFRule to a Terraform list item model.
func transformWAFRuleToListItemModel(rule azionapi.WAFRule) WafRuleSetListItemDataModel {
	result := WafRuleSetListItemDataModel{
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
