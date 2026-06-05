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
	_ resource.Resource                = &wafResource{}
	_ resource.ResourceWithConfigure   = &wafResource{}
	_ resource.ResourceWithImportState = &wafResource{}
)

func WafResource() resource.Resource {
	return &wafResource{}
}

type wafResource struct {
	client *apiClient
}

type WafResourceModel struct {
	ID          types.String        `tfsdk:"id"`
	LastUpdated types.String        `tfsdk:"last_updated"`
	Result      *WafResourceResults `tfsdk:"result"`
}

type WafResourceResults struct {
	ID             types.Int64                     `tfsdk:"id"`
	Name           types.String                    `tfsdk:"name"`
	Active         types.Bool                      `tfsdk:"active"`
	LastEditor     types.String                    `tfsdk:"last_editor"`
	LastModified   types.String                    `tfsdk:"last_modified"`
	ProductVersion types.String                    `tfsdk:"product_version"`
	IsVersioned    types.Bool                      `tfsdk:"is_versioned"`
	Version        types.Int64                     `tfsdk:"version"`
	VersionState   types.String                    `tfsdk:"version_state"`
	VersionID      types.String                    `tfsdk:"version_id"`
	EngineSettings *WafEngineSettingsResourceModel `tfsdk:"engine_settings"`
}

type WafEngineSettingsResourceModel struct {
	EngineVersion types.String                              `tfsdk:"engine_version"`
	Type          types.String                              `tfsdk:"type"`
	Attributes    *WafEngineSettingsAttributesResourceModel `tfsdk:"attributes"`
}

type WafEngineSettingsAttributesResourceModel struct {
	Rulesets   []types.Int64                      `tfsdk:"rulesets"`
	Thresholds []WafThresholdWrapperResourceModel `tfsdk:"thresholds"`
}

type WafThresholdWrapperResourceModel struct {
	Threshold *WafThresholdConfigResourceModel `tfsdk:"threshold"`
}

type WafThresholdConfigResourceModel struct {
	Threat      types.String `tfsdk:"threat"`
	Sensitivity types.String `tfsdk:"sensitivity"`
}

func (r *wafResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf"
}

func (r *wafResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a WAF (Web Application Firewall) resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"result": schema.SingleNestedAttribute{
				Description: "The WAF configuration.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The ID of the WAF.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the WAF.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the WAF is active.",
						Optional:    true,
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
					"is_versioned": schema.BoolAttribute{
						Description: "Whether the WAF is versioned.",
						Computed:    true,
					},
					"version": schema.Int64Attribute{
						Description: "The current version of the WAF.",
						Computed:    true,
					},
					"version_state": schema.StringAttribute{
						Description: "The state of the current WAF version.",
						Computed:    true,
					},
					"version_id": schema.StringAttribute{
						Description: "The identifier of the current WAF version.",
						Computed:    true,
					},
					"engine_settings": schema.SingleNestedAttribute{
						Description: "Engine settings for the WAF.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"engine_version": schema.StringAttribute{
								Description: "Engine version for the WAF.",
								Optional:    true,
							},
							"type": schema.StringAttribute{
								Description: "Type of the WAF engine.",
								Optional:    true,
							},
							"attributes": schema.SingleNestedAttribute{
								Description: "Attributes for the WAF engine settings.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"rulesets": schema.ListAttribute{
										Description: "List of ruleset IDs.",
										Optional:    true,
										ElementType: types.Int64Type,
									},
									"thresholds": schema.SetNestedAttribute{
										Description: "Threshold configurations for the WAF. Order-insensitive; the API returns thresholds sorted alphabetically by threat.",
										Optional:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"threshold": schema.SingleNestedAttribute{
													Description: "A single threshold configuration.",
													Required:    true,
													Attributes: map[string]schema.Attribute{
														"threat": schema.StringAttribute{
															Description: "The threat type for the threshold.",
															Required:    true,
														},
														"sensitivity": schema.StringAttribute{
															Description: "The sensitivity level for the threshold.",
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
					},
				},
			},
		},
	}
}

func (r *wafResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *wafResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WafResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the WAF request.
	wafRequest := azionapi.NewWAFRequest(plan.Result.Name.ValueString())

	// Set optional fields.
	if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
		wafRequest.SetActive(plan.Result.Active.ValueBool())
	}

	// Save the plan's engine_settings to preserve it if not specified.
	planEngineSettings := plan.Result.EngineSettings

	// Set engine settings if provided.
	if plan.Result.EngineSettings != nil {
		engineSettings := buildWAFEngineSettingsRequest(plan.Result.EngineSettings)
		wafRequest.SetEngineSettings(engineSettings)
	}

	// Create the WAF.
	wafResponse, response, err := r.client.api.WAFsAPI.CreateWaf(ctx).WAFRequest(*wafRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			wafResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFResponse, *http.Response, error) {
				return r.client.api.WAFsAPI.CreateWaf(ctx).WAFRequest(*wafRequest).Execute()
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
	data := wafResponse.GetData()
	plan.Result = transformWAFToResourceModel(data)
	plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Only update engine_settings from API response if the plan had it specified.
	// This prevents Terraform from seeing an inconsistency when engine_settings was null in plan.
	if planEngineSettings == nil {
		plan.Result.EngineSettings = nil
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WafResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save the state's engine_settings to preserve it if it was null.
	stateEngineSettings := state.Result.EngineSettings

	var wafID int64
	var err error
	if state.ID.IsNull() {
		wafID = state.Result.ID.ValueInt64()
	} else {
		wafID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WAF ID",
			)
			return
		}
	}

	wafResponse, response, err := r.client.api.WAFsAPI.RetrieveWaf(ctx, wafID).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			wafResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFResponse, *http.Response, error) {
				return r.client.api.WAFsAPI.RetrieveWaf(ctx, wafID).Execute()
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

	data := wafResponse.GetData()
	state.Result = transformWAFToResourceModel(data)
	state.ID = types.StringValue(strconv.FormatInt(wafID, 10))

	// Only update engine_settings from API response if the state had it specified.
	// This prevents Terraform from seeing an inconsistency when engine_settings was null in state.
	if stateEngineSettings == nil {
		state.Result.EngineSettings = nil
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WafResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WafResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	var wafID int64
	var err error
	if state.ID.IsNull() {
		wafID = state.Result.ID.ValueInt64()
	} else {
		wafID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WAF ID",
			)
			return
		}
	}

	// Build the WAF request.
	wafRequest := azionapi.NewWAFRequest(plan.Result.Name.ValueString())

	// Set optional fields.
	if !plan.Result.Active.IsNull() && !plan.Result.Active.IsUnknown() {
		wafRequest.SetActive(plan.Result.Active.ValueBool())
	}

	// Save the plan's engine_settings to preserve it if not specified.
	planEngineSettings := plan.Result.EngineSettings

	// Set engine settings if provided.
	if plan.Result.EngineSettings != nil {
		engineSettings := buildWAFEngineSettingsRequest(plan.Result.EngineSettings)
		wafRequest.SetEngineSettings(engineSettings)
	}

	// Update the WAF.
	wafResponse, response, err := r.client.api.WAFsAPI.UpdateWaf(ctx, wafID).WAFRequest(*wafRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			wafResponse, response, err = utils.RetryOn429(func() (*azionapi.WAFResponse, *http.Response, error) {
				return r.client.api.WAFsAPI.UpdateWaf(ctx, wafID).WAFRequest(*wafRequest).Execute()
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
	data := wafResponse.GetData()
	plan.Result = transformWAFToResourceModel(data)
	plan.ID = types.StringValue(strconv.FormatInt(wafID, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Only update engine_settings from API response if the plan had it specified.
	// This prevents Terraform from seeing an inconsistency when engine_settings was null in plan.
	if planEngineSettings == nil {
		plan.Result.EngineSettings = nil
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WafResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var wafID int64
	var err error
	if state.ID.IsNull() {
		wafID = state.Result.ID.ValueInt64()
	} else {
		wafID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WAF ID",
			)
			return
		}
	}

	deleteResponse, response, err := r.client.api.WAFsAPI.DeleteWaf(ctx, wafID).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
				delResp, resp, err := r.client.api.WAFsAPI.DeleteWaf(ctx, wafID).Execute()
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

func (r *wafResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildWAFEngineSettingsRequest builds a WAFEngineSettingsFieldRequest from the Terraform model.
func buildWAFEngineSettingsRequest(model *WafEngineSettingsResourceModel) azionapi.WAFEngineSettingsFieldRequest {
	engineSettings := azionapi.NewWAFEngineSettingsFieldRequest()

	if !model.EngineVersion.IsNull() && !model.EngineVersion.IsUnknown() {
		engineSettings.SetEngineVersion(model.EngineVersion.ValueString())
	}

	if !model.Type.IsNull() && !model.Type.IsUnknown() {
		engineSettings.SetType(model.Type.ValueString())
	}

	if model.Attributes != nil {
		attrs := buildWAFEngineSettingsAttributesRequest(model.Attributes)
		engineSettings.SetAttributes(attrs)
	}

	return *engineSettings
}

// buildWAFEngineSettingsAttributesRequest builds a WAFEngineSettingsAttributesFieldRequest from the Terraform model.
func buildWAFEngineSettingsAttributesRequest(model *WafEngineSettingsAttributesResourceModel) azionapi.WAFEngineSettingsAttributesFieldRequest {
	attrs := azionapi.NewWAFEngineSettingsAttributesFieldRequest()

	if len(model.Rulesets) > 0 {
		var rulesets []int64
		for _, r := range model.Rulesets {
			rulesets = append(rulesets, r.ValueInt64())
		}
		attrs.SetRulesets(rulesets)
	}

	if len(model.Thresholds) > 0 {
		var thresholds []azionapi.ThresholdsConfigFieldRequest
		for _, wrapper := range model.Thresholds {
			if wrapper.Threshold == nil {
				continue
			}
			t := wrapper.Threshold
			threshold := azionapi.NewThresholdsConfigFieldRequest(t.Threat.ValueString())
			if !t.Sensitivity.IsNull() && !t.Sensitivity.IsUnknown() {
				threshold.SetSensitivity(t.Sensitivity.ValueString())
			}
			thresholds = append(thresholds, *threshold)
		}
		attrs.SetThresholds(thresholds)
	}

	return *attrs
}

// transformWAFToResourceModel transforms an SDK WAF to a Terraform resource model.
func transformWAFToResourceModel(waf azionapi.WAF) *WafResourceResults {
	result := &WafResourceResults{
		ID:           types.Int64Value(waf.GetId()),
		Name:         types.StringValue(waf.GetName()),
		LastEditor:   types.StringValue(waf.GetLastEditor()),
		LastModified: types.StringValue(waf.GetLastModified().Format(time.RFC3339)),
		IsVersioned:  types.BoolValue(waf.IsVersioned),
		Version:      types.Int64PointerValue(waf.Version.Get()),
		VersionState: types.StringPointerValue(waf.VersionState.Get()),
		VersionID:    types.StringPointerValue(waf.VersionId.Get()),
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
		result.EngineSettings = transformWAFEngineSettingsToResourceModel(engineSettings)
	} else {
		result.EngineSettings = nil
	}

	return result
}

// transformWAFEngineSettingsToResourceModel transforms SDK engine settings to Terraform resource model.
func transformWAFEngineSettingsToResourceModel(engineSettings azionapi.WAFEngineSettingsField) *WafEngineSettingsResourceModel {
	result := &WafEngineSettingsResourceModel{}

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
		result.Attributes = transformWAFEngineSettingsAttributesToResourceModel(attrs)
	} else {
		result.Attributes = nil
	}

	return result
}

// transformWAFEngineSettingsAttributesToResourceModel transforms SDK attributes to Terraform resource model.
func transformWAFEngineSettingsAttributesToResourceModel(attrs azionapi.WAFEngineSettingsAttributesField) *WafEngineSettingsAttributesResourceModel {
	result := &WafEngineSettingsAttributesResourceModel{}

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
		var thresholdValues []WafThresholdWrapperResourceModel
		for _, t := range thresholds {
			thresholdModel := WafThresholdConfigResourceModel{
				Threat: types.StringValue(t.GetThreat()),
			}
			if t.HasSensitivity() {
				thresholdModel.Sensitivity = types.StringValue(t.GetSensitivity())
			} else {
				thresholdModel.Sensitivity = types.StringNull()
			}
			thresholdValues = append(thresholdValues, WafThresholdWrapperResourceModel{
				Threshold: &thresholdModel,
			})
		}
		result.Thresholds = thresholdValues
	} else {
		result.Thresholds = nil
	}

	return result
}
