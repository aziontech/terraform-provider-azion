package provider

import (
	"context"
	"io"
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
	_ resource.Resource                = &edgeApplicationCacheSettingsResource{}
	_ resource.ResourceWithConfigure   = &edgeApplicationCacheSettingsResource{}
	_ resource.ResourceWithImportState = &edgeApplicationCacheSettingsResource{}
)

func NewEdgeApplicationCacheSettingsResource() resource.Resource {
	return &edgeApplicationCacheSettingsResource{}
}

type edgeApplicationCacheSettingsResource struct {
	client *apiClient
}

type EdgeApplicationCacheSettingsResourceModel struct {
	SchemaVersion types.Int64                          `tfsdk:"schema_version"`
	CacheSettings *EdgeApplicationCacheSettingsResults `tfsdk:"cache_settings"`
	ID            types.String                         `tfsdk:"id"`
	ApplicationID types.Int64                          `tfsdk:"edge_application_id"`
	LastUpdated   types.String                         `tfsdk:"last_updated"`
}

type EdgeApplicationCacheSettingsResults struct {
	CacheSettingID              types.Int64    `tfsdk:"cache_setting_id"`
	Name                        types.String   `tfsdk:"name"`
	BrowserCacheSettings        types.String   `tfsdk:"browser_cache_settings"`
	BrowserCacheSettingsMaxTTL  types.Int64    `tfsdk:"browser_cache_settings_maximum_ttl"`
	CDNCacheSettings            types.String   `tfsdk:"cdn_cache_settings"`
	CDNCacheSettingsMaxTTL      types.Int64    `tfsdk:"cdn_cache_settings_maximum_ttl"`
	CacheByQueryString          types.String   `tfsdk:"cache_by_query_string"`
	QueryStringFields           []types.String `tfsdk:"query_string_fields"`
	EnableQueryStringSort       types.Bool     `tfsdk:"enable_query_string_sort"`
	CacheByCookies              types.String   `tfsdk:"cache_by_cookies"`
	CookieNames                 []types.String `tfsdk:"cookie_names"`
	AdaptiveDeliveryAction      types.String   `tfsdk:"adaptive_delivery_action"`
	DeviceGroup                 []types.Int64  `tfsdk:"device_group"`
	EnableCachingForPost        types.Bool     `tfsdk:"enable_caching_for_post"`
	L2CachingEnabled            types.Bool     `tfsdk:"l2_caching_enabled"`
	IsSliceConfigurationEnabled types.Bool     `tfsdk:"is_slice_configuration_enabled"`
	IsSliceEdgeCachingEnabled   types.Bool     `tfsdk:"is_slice_edge_caching_enabled"`
	IsSliceL2CachingEnabled     types.Bool     `tfsdk:"is_slice_l2_caching_enabled"`
	SliceConfigurationRange     types.Int64    `tfsdk:"slice_configuration_range"`
	EnableCachingForOptions     types.Bool     `tfsdk:"enable_caching_for_options"`
	EnableStaleCache            types.Bool     `tfsdk:"enable_stale_cache"`
	L2Region                    types.String   `tfsdk:"l2_region"`
}

func (r *edgeApplicationCacheSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_cache_setting"
}

func (r *edgeApplicationCacheSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Application",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"cache_settings": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"cache_setting_id": schema.Int64Attribute{
						Description: "The cache settings identifier to target for the resource.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the cache settings.",
						Required:    true,
					},
					"browser_cache_settings": schema.StringAttribute{
						Description: "Browser cache settings value.",
						Optional:    true,
						Computed:    true,
					},
					"browser_cache_settings_maximum_ttl": schema.Int64Attribute{
						Description: "Maximum TTL for browser cache settings.",
						Optional:    true,
						Computed:    true,
					},
					"cdn_cache_settings": schema.StringAttribute{
						Description: "CDN cache settings value.",
						Optional:    true,
						Computed:    true,
					},
					"cdn_cache_settings_maximum_ttl": schema.Int64Attribute{
						Description: "Maximum TTL for CDN cache settings.",
						Optional:    true,
						Computed:    true,
					},
					"cache_by_query_string": schema.StringAttribute{
						Description: "Cache by query string settings value.",
						Optional:    true,
						Computed:    true,
					},
					"query_string_fields": schema.ListAttribute{
						ElementType: types.StringType,
						Description: "Query string fields for cache settings.",
						Optional:    true,
					},
					"enable_query_string_sort": schema.BoolAttribute{
						Description: "Enable query string sorting for cache settings.",
						Computed:    true,
						Optional:    true,
					},
					"cache_by_cookies": schema.StringAttribute{
						Description: "Cache by cookies settings value.",
						Optional:    true,
						Computed:    true,
					},
					"cookie_names": schema.ListAttribute{
						ElementType: types.StringType,
						Description: "Cookie names for cache settings.",
						Optional:    true,
					},
					"adaptive_delivery_action": schema.StringAttribute{
						Description: "Adaptive delivery action settings value.",
						Optional:    true,
						Computed:    true,
					},
					"device_group": schema.ListAttribute{
						Description: "Device group settings.",
						Optional:    true,
						ElementType: types.Int64Type,
					},
					"enable_caching_for_post": schema.BoolAttribute{
						Description: "Enable caching for POST requests.",
						Optional:    true,
						Computed:    true,
					},
					"l2_caching_enabled": schema.BoolAttribute{
						Description: "Enable L2 caching.",
						Optional:    true,
						Computed:    true,
					},
					"is_slice_configuration_enabled": schema.BoolAttribute{
						Description: "Enable slice configuration.",
						Optional:    true,
						Computed:    true,
					},
					"is_slice_edge_caching_enabled": schema.BoolAttribute{
						Description: "Enable slice edge caching.",
						Optional:    true,
						Computed:    true,
					},
					"is_slice_l2_caching_enabled": schema.BoolAttribute{
						Description: "Enable slice L2 caching.",
						Optional:    true,
						Computed:    true,
					},
					"slice_configuration_range": schema.Int64Attribute{
						Description: "Slice configuration range.",
						Optional:    true,
						Computed:    true,
					},
					"enable_caching_for_options": schema.BoolAttribute{
						Description: "Enable caching for OPTIONS requests.",
						Optional:    true,
						Computed:    true,
					},
					"enable_stale_cache": schema.BoolAttribute{
						Description: "Enable stale cache.",
						Optional:    true,
						Computed:    true,
					},
					"l2_region": schema.StringAttribute{
						Description: "L2 region settings value.",
						Computed:    true,
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *edgeApplicationCacheSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *edgeApplicationCacheSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeApplicationCacheSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var edgeApplicationID types.Int64
	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CacheSettings.AdaptiveDeliveryAction.ValueString() == "whitelist" {
		if len(plan.CacheSettings.DeviceGroup) == 0 {
			resp.Diagnostics.AddError(
				"DeviceGroup error ",
				"When you set AdaptiveDeliveryAction with `whitelist` you should set at least one DeviceGroup",
			)
			return
		}
	} else {
		if plan.CacheSettings.AdaptiveDeliveryAction.ValueString() == "ignore" {
			if len(plan.CacheSettings.DeviceGroup) == 0 && plan.CacheSettings.DeviceGroup != nil {
				resp.Diagnostics.AddError(
					"DeviceGroup error ",
					"When you set AdaptiveDeliveryAction with `ignore` you should remove DeviceGroup from request or set null",
				)
				return
			}
		}
	}

	if plan.CacheSettings.CacheByCookies.ValueString() == "ignore" || plan.CacheSettings.CacheByCookies.ValueString() == "all" {
		if len(plan.CacheSettings.CookieNames) > 0 {
			resp.Diagnostics.AddError(
				"cookie_names and cache_by_cookies error ",
				"When you set cache_by_cookies with `ignore` or `all` you should remove cookie_names from request",
			)
			return
		}
	} else {
		if plan.CacheSettings.CacheByCookies.ValueString() == "whitelist" || plan.CacheSettings.CacheByCookies.ValueString() == "blacklist" {
			if len(plan.CacheSettings.CookieNames) == 0 && plan.CacheSettings.CookieNames == nil {
				resp.Diagnostics.AddError(
					"cookie_names error ",
					"You should set at least one cookie_names",
				)
				return
			}
		}
	}

	if plan.CacheSettings.CacheByQueryString.ValueString() == "ignore" || plan.CacheSettings.CacheByQueryString.ValueString() == "all" {
		if len(plan.CacheSettings.QueryStringFields) > 0 {
			resp.Diagnostics.AddError(
				"query_string_fields and cache_by_query_string error ",
				"When you set cache_by_query_string with `ignore` or `all` you should remove query_string_fields from request",
			)
			return
		}
	} else {
		if plan.CacheSettings.CacheByQueryString.ValueString() == "whitelist" || plan.CacheSettings.CacheByQueryString.ValueString() == "blacklist" {
			if len(plan.CacheSettings.QueryStringFields) == 0 && plan.CacheSettings.QueryStringFields == nil {
				resp.Diagnostics.AddError(
					"query_string_fields error ",
					"You should set at least one query_string_fields",
				)
				return
			}
		}
	}

	var CookieNamesRequest []string
	for _, cookieName := range plan.CacheSettings.CookieNames {
		CookieNamesRequest = append(CookieNamesRequest, cookieName.ValueString())
	}
	var QueryStringFieldsRequest []string
	for _, queryStringField := range plan.CacheSettings.QueryStringFields {
		QueryStringFieldsRequest = append(QueryStringFieldsRequest, queryStringField.ValueString())
	}
	var DeviceGroupsRequest []int32
	for _, DeviceGroup := range plan.CacheSettings.DeviceGroup {
		DeviceGroupsRequest = append(DeviceGroupsRequest, int32(DeviceGroup.ValueInt64()))
	}

	cacheSettings := edgeapplications.ApplicationCacheCreateRequest{
		Name:                           plan.CacheSettings.Name.ValueString(),
		BrowserCacheSettings:           edgeapplications.PtrString(plan.CacheSettings.BrowserCacheSettings.ValueString()),
		BrowserCacheSettingsMaximumTtl: edgeapplications.PtrInt64(plan.CacheSettings.BrowserCacheSettingsMaxTTL.ValueInt64()),
		CdnCacheSettings:               edgeapplications.PtrString(plan.CacheSettings.CDNCacheSettings.ValueString()),
		CdnCacheSettingsMaximumTtl:     edgeapplications.PtrInt64(plan.CacheSettings.CDNCacheSettingsMaxTTL.ValueInt64()),
		CacheByQueryString:             edgeapplications.PtrString(plan.CacheSettings.CacheByQueryString.ValueString()),
		QueryStringFields:              QueryStringFieldsRequest,
		EnableQueryStringSort:          edgeapplications.PtrBool(plan.CacheSettings.EnableQueryStringSort.ValueBool()),
		CacheByCookies:                 edgeapplications.PtrString(plan.CacheSettings.CacheByCookies.ValueString()),
		CookieNames:                    CookieNamesRequest,
		AdaptiveDeliveryAction:         edgeapplications.PtrString(plan.CacheSettings.AdaptiveDeliveryAction.ValueString()),
		DeviceGroup:                    DeviceGroupsRequest,
		EnableCachingForPost:           edgeapplications.PtrBool(plan.CacheSettings.EnableCachingForPost.ValueBool()),
		L2CachingEnabled:               edgeapplications.PtrBool(plan.CacheSettings.L2CachingEnabled.ValueBool()),
		IsSliceConfigurationEnabled:    edgeapplications.PtrBool(plan.CacheSettings.IsSliceConfigurationEnabled.ValueBool()),
		IsSliceEdgeCachingEnabled:      edgeapplications.PtrBool(plan.CacheSettings.IsSliceEdgeCachingEnabled.ValueBool()),
		IsSliceL2CachingEnabled:        edgeapplications.PtrBool(plan.CacheSettings.IsSliceL2CachingEnabled.ValueBool()),
		SliceConfigurationRange:        edgeapplications.PtrInt64(plan.CacheSettings.SliceConfigurationRange.ValueInt64()),
		EnableCachingForOptions:        edgeapplications.PtrBool(plan.CacheSettings.EnableCachingForOptions.ValueBool()),
		EnableStaleCache:               edgeapplications.PtrBool(plan.CacheSettings.EnableStaleCache.ValueBool()),
	}
	createdCacheSetting, response, err := r.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsApi.EdgeApplicationsEdgeApplicationIdCacheSettingsPost(ctx, edgeApplicationID.ValueInt64()).ApplicationCacheCreateRequest(cacheSettings).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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
	var CookieNames []types.String
	for _, cookieName := range createdCacheSetting.Results.GetCookieNames() {
		CookieNames = append(CookieNames, types.StringValue(cookieName))
	}
	var QueryStringFields []types.String
	for _, queryStringField := range createdCacheSetting.Results.GetQueryStringFields() {
		QueryStringFields = append(QueryStringFields, types.StringValue(queryStringField))
	}
	var DeviceGroups []types.Int64
	for _, DeviceGroup := range createdCacheSetting.Results.GetDeviceGroup() {
		DeviceGroups = append(DeviceGroups, types.Int64Value(int64(DeviceGroup)))
	}

	plan.CacheSettings = &EdgeApplicationCacheSettingsResults{
		CacheSettingID:              types.Int64Value(createdCacheSetting.Results.GetId()),
		Name:                        types.StringValue(createdCacheSetting.Results.GetName()),
		BrowserCacheSettings:        types.StringValue(createdCacheSetting.Results.GetBrowserCacheSettings()),
		BrowserCacheSettingsMaxTTL:  types.Int64Value(createdCacheSetting.Results.GetBrowserCacheSettingsMaximumTtl()),
		CDNCacheSettings:            types.StringValue(createdCacheSetting.Results.GetCdnCacheSettings()),
		CDNCacheSettingsMaxTTL:      types.Int64Value(createdCacheSetting.Results.GetCdnCacheSettingsMaximumTtl()),
		CacheByQueryString:          types.StringValue(createdCacheSetting.Results.GetCacheByQueryString()),
		QueryStringFields:           QueryStringFields,
		EnableQueryStringSort:       types.BoolValue(createdCacheSetting.Results.GetEnableQueryStringSort()),
		CacheByCookies:              types.StringValue(createdCacheSetting.Results.GetCacheByCookies()),
		CookieNames:                 CookieNames,
		AdaptiveDeliveryAction:      types.StringValue(createdCacheSetting.Results.GetAdaptiveDeliveryAction()),
		DeviceGroup:                 DeviceGroups,
		EnableCachingForPost:        types.BoolValue(createdCacheSetting.Results.GetEnableCachingForPost()),
		L2CachingEnabled:            types.BoolValue(createdCacheSetting.Results.GetL2CachingEnabled()),
		IsSliceConfigurationEnabled: types.BoolValue(createdCacheSetting.Results.GetIsSliceConfigurationEnabled()),
		IsSliceEdgeCachingEnabled:   types.BoolValue(createdCacheSetting.Results.GetIsSliceEdgeCachingEnabled()),
		IsSliceL2CachingEnabled:     types.BoolValue(createdCacheSetting.Results.GetIsSliceL2CachingEnabled()),
		SliceConfigurationRange:     types.Int64Value(createdCacheSetting.Results.GetSliceConfigurationRange()),
		EnableCachingForOptions:     types.BoolValue(createdCacheSetting.Results.GetEnableCachingForOptions()),
		EnableStaleCache:            types.BoolValue(createdCacheSetting.Results.GetEnableStaleCache()),
		L2Region:                    types.StringValue(createdCacheSetting.Results.GetL2Region()),
	}

	plan.SchemaVersion = types.Int64Value(*createdCacheSetting.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(createdCacheSetting.Results.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationCacheSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeApplicationCacheSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var EdgeApplicationId int64
	var CacheSettingId int64
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		EdgeApplicationId = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		CacheSettingId = int64(utils.AtoiNoError(valueFromCmd[1], resp))
	} else {
		EdgeApplicationId = state.ApplicationID.ValueInt64()
		CacheSettingId = state.CacheSettings.CacheSettingID.ValueInt64()
	}

	if CacheSettingId == 0 || CacheSettingId < 0 {
		resp.Diagnostics.AddError(
			"Cache settings ID error ",
			"is not null",
		)
		return
	}

	cacheSettingResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsApi.EdgeApplicationsEdgeApplicationIdCacheSettingsCacheSettingsIdGet(ctx, EdgeApplicationId, CacheSettingId).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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
	var CookieNames []types.String
	for _, cookieName := range cacheSettingResponse.Results.GetCookieNames() {
		CookieNames = append(CookieNames, types.StringValue(*cookieName))
	}
	var QueryStringFields []types.String
	for _, queryStringField := range cacheSettingResponse.Results.GetQueryStringFields() {
		QueryStringFields = append(QueryStringFields, types.StringValue(queryStringField))
	}
	var DeviceGroups []types.Int64
	for _, DeviceGroup := range cacheSettingResponse.Results.GetDeviceGroup() {
		DeviceGroups = append(DeviceGroups, types.Int64Value(int64(DeviceGroup)))
	}

	state.CacheSettings = &EdgeApplicationCacheSettingsResults{
		CacheSettingID:              types.Int64Value(cacheSettingResponse.Results.GetId()),
		Name:                        types.StringValue(cacheSettingResponse.Results.GetName()),
		BrowserCacheSettings:        types.StringValue(cacheSettingResponse.Results.GetBrowserCacheSettings()),
		BrowserCacheSettingsMaxTTL:  types.Int64Value(cacheSettingResponse.Results.GetBrowserCacheSettingsMaximumTtl()),
		CDNCacheSettings:            types.StringValue(cacheSettingResponse.Results.GetCdnCacheSettings()),
		CDNCacheSettingsMaxTTL:      types.Int64Value(cacheSettingResponse.Results.GetCdnCacheSettingsMaximumTtl()),
		CacheByQueryString:          types.StringValue(cacheSettingResponse.Results.GetCacheByQueryString()),
		QueryStringFields:           QueryStringFields,
		EnableQueryStringSort:       types.BoolValue(cacheSettingResponse.Results.GetEnableQueryStringSort()),
		CacheByCookies:              types.StringValue(cacheSettingResponse.Results.GetCacheByCookies()),
		CookieNames:                 CookieNames,
		AdaptiveDeliveryAction:      types.StringValue(cacheSettingResponse.Results.GetAdaptiveDeliveryAction()),
		DeviceGroup:                 DeviceGroups,
		EnableCachingForPost:        types.BoolValue(cacheSettingResponse.Results.GetEnableCachingForPost()),
		L2CachingEnabled:            types.BoolValue(cacheSettingResponse.Results.GetL2CachingEnabled()),
		IsSliceConfigurationEnabled: types.BoolValue(cacheSettingResponse.Results.GetIsSliceConfigurationEnabled()),
		IsSliceEdgeCachingEnabled:   types.BoolValue(cacheSettingResponse.Results.GetIsSliceEdgeCachingEnabled()),
		IsSliceL2CachingEnabled:     types.BoolValue(cacheSettingResponse.Results.GetIsSliceL2CachingEnabled()),
		SliceConfigurationRange:     types.Int64Value(cacheSettingResponse.Results.GetSliceConfigurationRange()),
		EnableCachingForOptions:     types.BoolValue(cacheSettingResponse.Results.GetEnableCachingForOptions()),
		EnableStaleCache:            types.BoolValue(cacheSettingResponse.Results.GetEnableStaleCache()),
		L2Region:                    types.StringValue(cacheSettingResponse.Results.GetL2Region()),
	}

	state.ID = types.StringValue(strconv.FormatInt(cacheSettingResponse.Results.GetId(), 10))
	state.ApplicationID = types.Int64Value(EdgeApplicationId)
	state.SchemaVersion = types.Int64Value(cacheSettingResponse.SchemaVersion)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationCacheSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeApplicationCacheSettingsResourceModel
	var edgeApplicationID types.Int64
	var CacheSettingId types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state EdgeApplicationCacheSettingsResourceModel
	diagsOrigin := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsOrigin...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.CacheSettings.CacheSettingID.IsNull() || plan.CacheSettings.CacheSettingID.ValueInt64() == 0 {
		CacheSettingId = state.CacheSettings.CacheSettingID
	} else {
		CacheSettingId = plan.CacheSettings.CacheSettingID
	}

	if CacheSettingId.IsNull() || CacheSettingId.ValueInt64() == 0 {
		resp.Diagnostics.AddError(
			"Cache settings ID error ",
			"is not null",
		)
		return
	}

	if plan.ApplicationID.IsNull() {
		edgeApplicationID = state.ApplicationID
	} else {
		edgeApplicationID = plan.ApplicationID
	}

	if edgeApplicationID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}

	if plan.CacheSettings.AdaptiveDeliveryAction.ValueString() == "whitelist" {
		if len(plan.CacheSettings.DeviceGroup) == 0 {
			resp.Diagnostics.AddError(
				"DeviceGroup error ",
				"When you set AdaptiveDeliveryAction with `whitelist` you should set at least one DeviceGroup",
			)
			return
		}
	} else {
		if plan.CacheSettings.AdaptiveDeliveryAction.ValueString() == "ignore" {
			if len(plan.CacheSettings.DeviceGroup) == 0 && plan.CacheSettings.DeviceGroup != nil {
				resp.Diagnostics.AddError(
					"DeviceGroup error ",
					"When you set AdaptiveDeliveryAction with `ignore` you should remove DeviceGroup from request or set null",
				)
				return
			}
		}
	}

	if plan.CacheSettings.CacheByCookies.ValueString() == "ignore" || plan.CacheSettings.CacheByCookies.ValueString() == "all" {
		if len(plan.CacheSettings.CookieNames) > 0 {
			resp.Diagnostics.AddError(
				"cookie_names and cache_by_cookies error ",
				"When you set cache_by_cookies with `ignore` or `all` you should remove cookie_names from request",
			)
			return
		}
	} else {
		if plan.CacheSettings.CacheByCookies.ValueString() == "whitelist" || plan.CacheSettings.CacheByCookies.ValueString() == "blacklist" {
			if len(plan.CacheSettings.CookieNames) == 0 && plan.CacheSettings.CookieNames == nil {
				resp.Diagnostics.AddError(
					"cookie_names error ",
					"You should set at least one cookie_names",
				)
				return
			}
		}
	}

	if plan.CacheSettings.CacheByQueryString.ValueString() == "ignore" || plan.CacheSettings.CacheByQueryString.ValueString() == "all" {
		if len(plan.CacheSettings.QueryStringFields) > 0 {
			resp.Diagnostics.AddError(
				"query_string_fields and cache_by_query_string error ",
				"When you set cache_by_query_string with `ignore` or `all` you should remove query_string_fields from request",
			)
			return
		}
	} else {
		if plan.CacheSettings.CacheByQueryString.ValueString() == "whitelist" || plan.CacheSettings.CacheByQueryString.ValueString() == "blacklist" {
			if len(plan.CacheSettings.QueryStringFields) == 0 && plan.CacheSettings.QueryStringFields == nil {
				resp.Diagnostics.AddError(
					"query_string_fields error ",
					"You should set at least one query_string_fields",
				)
				return
			}
		}
	}

	var CookieNamesRequest []string
	for _, cookieName := range plan.CacheSettings.CookieNames {
		CookieNamesRequest = append(CookieNamesRequest, cookieName.ValueString())
	}
	var QueryStringFieldsRequest []string
	for _, queryStringField := range plan.CacheSettings.QueryStringFields {
		QueryStringFieldsRequest = append(QueryStringFieldsRequest, queryStringField.ValueString())
	}
	var DeviceGroupsRequest []int32
	for _, DeviceGroup := range plan.CacheSettings.DeviceGroup {
		DeviceGroupsRequest = append(DeviceGroupsRequest, int32(DeviceGroup.ValueInt64()))
	}

	cacheSettings := edgeapplications.ApplicationCachePutRequest{
		Name:                           plan.CacheSettings.Name.ValueString(),
		BrowserCacheSettings:           edgeapplications.PtrString(plan.CacheSettings.BrowserCacheSettings.ValueString()),
		BrowserCacheSettingsMaximumTtl: edgeapplications.PtrInt64(plan.CacheSettings.BrowserCacheSettingsMaxTTL.ValueInt64()),
		CdnCacheSettings:               edgeapplications.PtrString(plan.CacheSettings.CDNCacheSettings.ValueString()),
		CdnCacheSettingsMaximumTtl:     edgeapplications.PtrInt64(plan.CacheSettings.CDNCacheSettingsMaxTTL.ValueInt64()),
		CacheByQueryString:             edgeapplications.PtrString(plan.CacheSettings.CacheByQueryString.ValueString()),
		QueryStringFields:              QueryStringFieldsRequest,
		EnableQueryStringSort:          edgeapplications.PtrBool(plan.CacheSettings.EnableQueryStringSort.ValueBool()),
		CacheByCookies:                 edgeapplications.PtrString(plan.CacheSettings.CacheByCookies.ValueString()),
		CookieNames:                    CookieNamesRequest,
		AdaptiveDeliveryAction:         edgeapplications.PtrString(plan.CacheSettings.AdaptiveDeliveryAction.ValueString()),
		DeviceGroup:                    DeviceGroupsRequest,
		EnableCachingForPost:           edgeapplications.PtrBool(plan.CacheSettings.EnableCachingForPost.ValueBool()),
		L2CachingEnabled:               edgeapplications.PtrBool(plan.CacheSettings.L2CachingEnabled.ValueBool()),
		IsSliceConfigurationEnabled:    edgeapplications.PtrBool(plan.CacheSettings.IsSliceConfigurationEnabled.ValueBool()),
		IsSliceEdgeCachingEnabled:      edgeapplications.PtrBool(plan.CacheSettings.IsSliceEdgeCachingEnabled.ValueBool()),
		IsSliceL2CachingEnabled:        edgeapplications.PtrBool(plan.CacheSettings.IsSliceL2CachingEnabled.ValueBool()),
		SliceConfigurationRange:        edgeapplications.PtrInt64(plan.CacheSettings.SliceConfigurationRange.ValueInt64()),
		EnableCachingForOptions:        edgeapplications.PtrBool(plan.CacheSettings.EnableCachingForOptions.ValueBool()),
		EnableStaleCache:               edgeapplications.PtrBool(plan.CacheSettings.EnableStaleCache.ValueBool()),
	}
	createdCacheSetting, response, err := r.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsApi.EdgeApplicationsEdgeApplicationIdCacheSettingsCacheSettingsIdPut(ctx, edgeApplicationID.ValueInt64(), CacheSettingId.ValueInt64()).ApplicationCachePutRequest(cacheSettings).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	var CookieNames []types.String
	for _, cookieName := range createdCacheSetting.Results.GetCookieNames() {
		CookieNames = append(CookieNames, types.StringValue(cookieName))
	}
	var QueryStringFields []types.String
	for _, queryStringField := range createdCacheSetting.Results.GetQueryStringFields() {
		QueryStringFields = append(QueryStringFields, types.StringValue(queryStringField))
	}
	var DeviceGroups []types.Int64
	for _, DeviceGroup := range createdCacheSetting.Results.GetDeviceGroup() {
		DeviceGroups = append(DeviceGroups, types.Int64Value(int64(DeviceGroup)))
	}

	plan.CacheSettings = &EdgeApplicationCacheSettingsResults{
		CacheSettingID:              types.Int64Value(createdCacheSetting.Results.GetId()),
		Name:                        types.StringValue(createdCacheSetting.Results.GetName()),
		BrowserCacheSettings:        types.StringValue(createdCacheSetting.Results.GetBrowserCacheSettings()),
		BrowserCacheSettingsMaxTTL:  types.Int64Value(createdCacheSetting.Results.GetBrowserCacheSettingsMaximumTtl()),
		CDNCacheSettings:            types.StringValue(createdCacheSetting.Results.GetCdnCacheSettings()),
		CDNCacheSettingsMaxTTL:      types.Int64Value(createdCacheSetting.Results.GetCdnCacheSettingsMaximumTtl()),
		CacheByQueryString:          types.StringValue(createdCacheSetting.Results.GetCacheByQueryString()),
		QueryStringFields:           QueryStringFields,
		EnableQueryStringSort:       types.BoolValue(createdCacheSetting.Results.GetEnableQueryStringSort()),
		CacheByCookies:              types.StringValue(createdCacheSetting.Results.GetCacheByCookies()),
		CookieNames:                 CookieNames,
		AdaptiveDeliveryAction:      types.StringValue(createdCacheSetting.Results.GetAdaptiveDeliveryAction()),
		DeviceGroup:                 DeviceGroups,
		EnableCachingForPost:        types.BoolValue(createdCacheSetting.Results.GetEnableCachingForPost()),
		L2CachingEnabled:            types.BoolValue(createdCacheSetting.Results.GetL2CachingEnabled()),
		IsSliceConfigurationEnabled: types.BoolValue(createdCacheSetting.Results.GetIsSliceConfigurationEnabled()),
		IsSliceEdgeCachingEnabled:   types.BoolValue(createdCacheSetting.Results.GetIsSliceEdgeCachingEnabled()),
		IsSliceL2CachingEnabled:     types.BoolValue(createdCacheSetting.Results.GetIsSliceL2CachingEnabled()),
		SliceConfigurationRange:     types.Int64Value(createdCacheSetting.Results.GetSliceConfigurationRange()),
		EnableCachingForOptions:     types.BoolValue(createdCacheSetting.Results.GetEnableCachingForOptions()),
		EnableStaleCache:            types.BoolValue(createdCacheSetting.Results.GetEnableStaleCache()),
		L2Region:                    types.StringValue(createdCacheSetting.Results.GetL2Region()),
	}

	plan.SchemaVersion = types.Int64Value(createdCacheSetting.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(createdCacheSetting.Results.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationCacheSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeApplicationCacheSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	edgeApplicationID := state.ApplicationID.ValueInt64()

	if state.CacheSettings.CacheSettingID.IsNull() {
		resp.Diagnostics.AddError(
			"Cache Setting ID error ",
			"is not null",
		)
		return
	}

	if state.ApplicationID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}
	response, err := r.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsApi.EdgeApplicationsEdgeApplicationIdCacheSettingsCacheSettingsIdDelete(ctx, edgeApplicationID, state.CacheSettings.CacheSettingID.ValueInt64()).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

func (r *edgeApplicationCacheSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
