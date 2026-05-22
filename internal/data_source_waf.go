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
	_ datasource.DataSource              = &WafDataSource{}
	_ datasource.DataSourceWithConfigure = &WafDataSource{}
)

func dataSourceAzionWaf() datasource.DataSource {
	return &WafDataSource{}
}

type WafDataSource struct {
	client *apiClient
}

type WafDataSourceModel struct {
	ID      types.String        `tfsdk:"id"`
	WafID   types.Int64         `tfsdk:"waf_id"`
	Results *WafResultDataModel `tfsdk:"results"`
}

type WafResultDataModel struct {
	ID             types.Int64             `tfsdk:"id"`
	Name           types.String            `tfsdk:"name"`
	Active         types.Bool              `tfsdk:"active"`
	LastEditor     types.String            `tfsdk:"last_editor"`
	LastModified   types.String            `tfsdk:"last_modified"`
	ProductVersion types.String            `tfsdk:"product_version"`
	EngineSettings *WafEngineSettingsModel `tfsdk:"engine_settings"`
}

type WafEngineSettingsModel struct {
	EngineVersion types.String                      `tfsdk:"engine_version"`
	Type          types.String                      `tfsdk:"type"`
	Attributes    *WafEngineSettingsAttributesModel `tfsdk:"attributes"`
}

type WafEngineSettingsAttributesModel struct {
	Rulesets   []types.Int64              `tfsdk:"rulesets"`
	Thresholds []WafThresholdWrapperModel `tfsdk:"thresholds"`
}

type WafThresholdWrapperModel struct {
	Threshold *WafThresholdConfigModel `tfsdk:"threshold"`
}

type WafThresholdConfigModel struct {
	Threat      types.String `tfsdk:"threat"`
	Sensitivity types.String `tfsdk:"sensitivity"`
}

func (o *WafDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf"
}

func (o *WafDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The ID of the WAF.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the WAF.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the WAF is active.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the WAF.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the WAF.",
						Computed:    true,
					},
					"engine_settings": schema.SingleNestedAttribute{
						Description: "Engine settings for the WAF.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"engine_version": schema.StringAttribute{
								Description: "Engine version for the WAF.",
								Computed:    true,
							},
							"type": schema.StringAttribute{
								Description: "Type of the WAF engine.",
								Computed:    true,
							},
							"attributes": schema.SingleNestedAttribute{
								Description: "Attributes for the WAF engine settings.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"rulesets": schema.ListAttribute{
										Description: "List of ruleset IDs.",
										Computed:    true,
										ElementType: types.Int64Type,
									},
									"thresholds": schema.ListNestedAttribute{
										Description: "Threshold configurations for the WAF.",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"threshold": schema.SingleNestedAttribute{
													Description: "A single threshold configuration.",
													Computed:    true,
													Attributes: map[string]schema.Attribute{
														"threat": schema.StringAttribute{
															Description: "The threat type for the threshold.",
															Computed:    true,
														},
														"sensitivity": schema.StringAttribute{
															Description: "The sensitivity level for the threshold.",
															Computed:    true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (o *WafDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var wafID types.Int64

	diagsWafID := req.Config.GetAttribute(ctx, path.Root("waf_id"), &wafID)
	resp.Diagnostics.Append(diagsWafID...)
	if resp.Diagnostics.HasError() {
		return
	}

	wafResponse, response, err := o.client.api.WAFsAPI.RetrieveWaf(ctx, wafID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			wafResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFResponse, *http.Response, error) {
				return o.client.api.WAFsAPI.RetrieveWaf(ctx, wafID.ValueInt64()).Execute()
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
	results := transformWAFToResultModel(wafResponse.GetData())

	state := WafDataSourceModel{
		ID:      types.StringValue("Get WAF By ID"),
		WafID:   wafID,
		Results: results,
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// transformWAFToResultModel transforms an SDK WAF to a Terraform model.
func transformWAFToResultModel(waf azionapi.WAF) *WafResultDataModel {
	result := &WafResultDataModel{
		ID:           types.Int64Value(waf.GetId()),
		Name:         types.StringValue(waf.GetName()),
		LastEditor:   types.StringValue(waf.GetLastEditor()),
		LastModified: types.StringValue(waf.GetLastModified().Format(time.RFC3339)),
	}

	// Optional active.
	if waf.HasActive() {
		result.Active = types.BoolValue(waf.GetActive())
	} else {
		result.Active = types.BoolNull()
	}

	// Optional product_version.
	if waf.HasProductVersion() {
		result.ProductVersion = types.StringValue(waf.GetProductVersion())
	} else {
		result.ProductVersion = types.StringNull()
	}

	// Optional engine_settings.
	if waf.HasEngineSettings() {
		engineSettings := waf.GetEngineSettings()
		result.EngineSettings = transformWAFEngineSettingsToModel(engineSettings)
	} else {
		result.EngineSettings = nil
	}

	return result
}

// transformWAFEngineSettingsToModel transforms SDK engine settings to Terraform model.
func transformWAFEngineSettingsToModel(engineSettings azionapi.WAFEngineSettingsField) *WafEngineSettingsModel {
	result := &WafEngineSettingsModel{}

	// Optional engine_version.
	if engineSettings.HasEngineVersion() {
		result.EngineVersion = types.StringValue(engineSettings.GetEngineVersion())
	} else {
		result.EngineVersion = types.StringNull()
	}

	// Optional type.
	if engineSettings.HasType() {
		result.Type = types.StringValue(engineSettings.GetType())
	} else {
		result.Type = types.StringNull()
	}

	// Optional attributes.
	if engineSettings.HasAttributes() {
		attrs := engineSettings.GetAttributes()
		result.Attributes = transformWAFEngineSettingsAttributesToModel(attrs)
	} else {
		result.Attributes = nil
	}

	return result
}

// transformWAFEngineSettingsAttributesToModel transforms SDK attributes to Terraform model.
func transformWAFEngineSettingsAttributesToModel(attrs azionapi.WAFEngineSettingsAttributesField) *WafEngineSettingsAttributesModel {
	result := &WafEngineSettingsAttributesModel{}

	// Optional rulesets.
	if attrs.HasRulesets() {
		rulesets := attrs.GetRulesets()
		var rulesetValues []types.Int64
		for _, r := range rulesets {
			rulesetValues = append(rulesetValues, types.Int64Value(r))
		}
		result.Rulesets = rulesetValues
	} else {
		result.Rulesets = nil
	}

	// Optional thresholds.
	if attrs.HasThresholds() {
		thresholds := attrs.GetThresholds()
		var thresholdValues []WafThresholdWrapperModel
		for _, t := range thresholds {
			thresholdModel := WafThresholdConfigModel{
				Threat: types.StringValue(t.GetThreat()),
			}
			if t.HasSensitivity() {
				thresholdModel.Sensitivity = types.StringValue(t.GetSensitivity())
			} else {
				thresholdModel.Sensitivity = types.StringNull()
			}
			thresholdValues = append(thresholdValues, WafThresholdWrapperModel{
				Threshold: &thresholdModel,
			})
		}
		result.Thresholds = thresholdValues
	} else {
		result.Thresholds = nil
	}

	return result
}
