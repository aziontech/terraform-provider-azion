package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CacheSettingDataSource{}
	_ datasource.DataSourceWithConfigure = &CacheSettingDataSource{}
)

func dataSourceAzionEdgeApplicationCacheSetting() datasource.DataSource {
	return &CacheSettingDataSource{}
}

type CacheSettingDataSource struct {
	client *apiClient
}

type CacheSettingDataSourceModel struct {
	ApplicationID types.Int64         `tfsdk:"edge_application_id"`
	SchemaVersion types.Int64         `tfsdk:"schema_version"`
	Results       CacheSettingResults `tfsdk:"results"`
	ID            types.String        `tfsdk:"id"`
}

type CacheSettingResults struct {
	CacheSettingID              types.Int64    `tfsdk:"cache_setting_id"`
	Name                        types.String   `tfsdk:"name"`
	BrowserCacheSettings        types.String   `tfsdk:"browser_cache_settings"`
	BrowserCacheSettingsMaxTtl  types.Int64    `tfsdk:"browser_cache_settings_maximum_ttl"`
	CdnCacheSettings            types.String   `tfsdk:"cdn_cache_settings"`
	CdnCacheSettingsMaxTtl      types.Int64    `tfsdk:"cdn_cache_settings_maximum_ttl"`
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

func (c *CacheSettingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c.client = req.ProviderData.(*apiClient)
}

func (c *CacheSettingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_cache_setting"
}

func (c *CacheSettingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true,
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Application",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"cache_setting_id": schema.Int64Attribute{
						Description: "The cache settings identifier to target for the resource.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the cache settings.",
						Computed:    true,
					},
					"browser_cache_settings": schema.StringAttribute{
						Description: "Browser cache settings value.",
						Computed:    true,
					},
					"browser_cache_settings_maximum_ttl": schema.Int64Attribute{
						Description: "Maximum TTL for browser cache settings.",
						Computed:    true,
					},
					"cdn_cache_settings": schema.StringAttribute{
						Description: "CDN cache settings value.",
						Computed:    true,
					},
					"cdn_cache_settings_maximum_ttl": schema.Int64Attribute{
						Description: "Maximum TTL for CDN cache settings.",
						Computed:    true,
					},
					"cache_by_query_string": schema.StringAttribute{
						Description: "Cache by query string settings value.",
						Computed:    true,
					},
					"query_string_fields": schema.ListAttribute{
						ElementType: types.StringType,
						Description: "Query string fields for cache settings.",
						Computed:    true,
					},
					"enable_query_string_sort": schema.BoolAttribute{
						Description: "Enable query string sorting for cache settings.",
						Computed:    true,
					},
					"cache_by_cookies": schema.StringAttribute{
						Description: "Cache by cookies settings value.",
						Computed:    true,
					},
					"cookie_names": schema.ListAttribute{
						ElementType: types.StringType,
						Description: "Cookie names for cache settings.",
						Computed:    true,
					},
					"adaptive_delivery_action": schema.StringAttribute{
						Description: "Adaptive delivery action settings value.",
						Computed:    true,
					},
					"device_group": schema.ListAttribute{
						Description: "Device group settings.",
						Computed:    true,
						ElementType: types.Int64Type,
					},
					"enable_caching_for_post": schema.BoolAttribute{
						Description: "Enable caching for POST requests.",
						Computed:    true,
					},
					"l2_caching_enabled": schema.BoolAttribute{
						Description: "Enable L2 caching.",
						Computed:    true,
					},
					"is_slice_configuration_enabled": schema.BoolAttribute{
						Description: "Enable slice configuration.",
						Computed:    true,
					},
					"is_slice_edge_caching_enabled": schema.BoolAttribute{
						Description: "Enable slice edge caching.",
						Computed:    true,
					},
					"is_slice_l2_caching_enabled": schema.BoolAttribute{
						Description: "Enable slice L2 caching.",
						Computed:    true,
					},
					"slice_configuration_range": schema.Int64Attribute{
						Description: "Slice configuration range.",
						Computed:    true,
					},
					"enable_caching_for_options": schema.BoolAttribute{
						Description: "Enable caching for OPTIONS requests.",
						Computed:    true,
					},
					"enable_stale_cache": schema.BoolAttribute{
						Description: "Enable stale cache.",
						Computed:    true,
					},
					"l2_region": schema.StringAttribute{
						Description: "L2 region settings value.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (c *CacheSettingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var CacheSettingId types.Int64
	var EdgeApplicationId types.Int64
	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &EdgeApplicationId)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsCacheSettingId := req.Config.GetAttribute(ctx, path.Root("results").AtName("cache_setting_id"), &CacheSettingId)
	resp.Diagnostics.Append(diagsCacheSettingId...)
	if resp.Diagnostics.HasError() {
		return
	}

	cacheSettingResponse, response, err := c.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsAPI.EdgeApplicationsEdgeApplicationIdCacheSettingsCacheSettingsIdGet(ctx, EdgeApplicationId.ValueInt64(), CacheSettingId.ValueInt64()).Execute()
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

	cacheSettingResult := CacheSettingResults{
		CacheSettingID:              types.Int64Value(cacheSettingResponse.Results.GetId()),
		Name:                        types.StringValue(cacheSettingResponse.Results.GetName()),
		BrowserCacheSettings:        types.StringValue(cacheSettingResponse.Results.GetBrowserCacheSettings()),
		BrowserCacheSettingsMaxTtl:  types.Int64Value(cacheSettingResponse.Results.GetBrowserCacheSettingsMaximumTtl()),
		CdnCacheSettings:            types.StringValue(cacheSettingResponse.Results.GetCdnCacheSettings()),
		CdnCacheSettingsMaxTtl:      types.Int64Value(cacheSettingResponse.Results.GetCdnCacheSettingsMaximumTtl()),
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

	edgeApplicationsCacheSettingsState := CacheSettingDataSourceModel{
		ApplicationID: EdgeApplicationId,
		SchemaVersion: types.Int64Value(cacheSettingResponse.SchemaVersion),
		Results:       cacheSettingResult,
	}

	edgeApplicationsCacheSettingsState.ID = types.StringValue("Get By ID Cache Settings")
	diags := resp.State.Set(ctx, &edgeApplicationsCacheSettingsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
