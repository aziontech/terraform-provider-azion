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
	_ datasource.DataSource              = &CacheSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &CacheSettingsDataSource{}
)

func dataSourceAzionEdgeApplicationCacheSettings() datasource.DataSource {
	return &CacheSettingsDataSource{}
}

type CacheSettingsDataSource struct {
	client *apiClient
}

type CacheSettingsDataSourceModel struct {
	ApplicationID types.Int64                    `tfsdk:"edge_application_id"`
	Counter       types.Int64                    `tfsdk:"counter"`
	Page          types.Int64                    `tfsdk:"page"`
	PageSize      types.Int64                    `tfsdk:"page_size"`
	TotalPages    types.Int64                    `tfsdk:"total_pages"`
	Links         *GetCacheSettingsResponseLinks `tfsdk:"links"`
	SchemaVersion types.Int64                    `tfsdk:"schema_version"`
	Results       []CacheSettingsResults         `tfsdk:"results"`
	ID            types.String                   `tfsdk:"id"`
}

type GetCacheSettingsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type CacheSettingsResults struct {
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

func (c *CacheSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c.client = req.ProviderData.(*apiClient)
}

func (c *CacheSettingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_cache_settings"
}

func (c *CacheSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"counter": schema.Int64Attribute{
				Description: "The total number of Cache Settings.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Cache Settings.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of Cache Settings.",
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
		},
	}
}

func (c *CacheSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var Page types.Int64
	var PageSize types.Int64
	var EdgeApplicationId types.Int64
	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &EdgeApplicationId)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}
	if PageSize.ValueInt64() == 0 {
		PageSize = types.Int64Value(10)
	}

	cacheSettingsResponse, response, err := c.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsApi.EdgeApplicationsEdgeApplicationIdCacheSettingsGet(ctx, EdgeApplicationId.ValueInt64()).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute()
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

	var previous, next string
	if cacheSettingsResponse.Links.Previous.Get() != nil {
		previous = *cacheSettingsResponse.Links.Previous.Get()
	}
	if cacheSettingsResponse.Links.Next.Get() != nil {
		next = *cacheSettingsResponse.Links.Next.Get()
	}

	edgeApplicationsCacheSettingsState := CacheSettingsDataSourceModel{
		Page:          Page,
		PageSize:      PageSize,
		ApplicationID: EdgeApplicationId,
		SchemaVersion: types.Int64Value(cacheSettingsResponse.SchemaVersion),
		TotalPages:    types.Int64Value(cacheSettingsResponse.TotalPages),
		Counter:       types.Int64Value(cacheSettingsResponse.Count),
		Links: &GetCacheSettingsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	for _, resultCacheSettings := range cacheSettingsResponse.Results {
		var CookieNames []types.String
		for _, cookieName := range resultCacheSettings.GetCookieNames() {
			CookieNames = append(CookieNames, types.StringValue(*cookieName))
		}
		var QueryStringFields []types.String
		for _, queryStringField := range resultCacheSettings.GetQueryStringFields() {
			QueryStringFields = append(QueryStringFields, types.StringValue(queryStringField))
		}
		var DeviceGroups []types.Int64
		for _, DeviceGroup := range resultCacheSettings.GetDeviceGroup() {
			DeviceGroups = append(DeviceGroups, types.Int64Value(int64(DeviceGroup)))
		}
		edgeApplicationsCacheSettingsState.Results = append(edgeApplicationsCacheSettingsState.Results, CacheSettingsResults{
			CacheSettingID:              types.Int64Value(resultCacheSettings.GetId()),
			Name:                        types.StringValue(resultCacheSettings.GetName()),
			BrowserCacheSettings:        types.StringValue(resultCacheSettings.GetBrowserCacheSettings()),
			BrowserCacheSettingsMaxTtl:  types.Int64Value(resultCacheSettings.GetBrowserCacheSettingsMaximumTtl()),
			CdnCacheSettings:            types.StringValue(resultCacheSettings.GetCdnCacheSettings()),
			CdnCacheSettingsMaxTtl:      types.Int64Value(resultCacheSettings.GetCdnCacheSettingsMaximumTtl()),
			CacheByQueryString:          types.StringValue(resultCacheSettings.GetCacheByQueryString()),
			QueryStringFields:           QueryStringFields,
			EnableQueryStringSort:       types.BoolValue(resultCacheSettings.GetEnableQueryStringSort()),
			CacheByCookies:              types.StringValue(resultCacheSettings.GetCacheByCookies()),
			CookieNames:                 CookieNames,
			AdaptiveDeliveryAction:      types.StringValue(resultCacheSettings.GetAdaptiveDeliveryAction()),
			DeviceGroup:                 DeviceGroups,
			EnableCachingForPost:        types.BoolValue(resultCacheSettings.GetEnableCachingForPost()),
			L2CachingEnabled:            types.BoolValue(resultCacheSettings.GetL2CachingEnabled()),
			IsSliceConfigurationEnabled: types.BoolValue(resultCacheSettings.GetIsSliceConfigurationEnabled()),
			IsSliceEdgeCachingEnabled:   types.BoolValue(resultCacheSettings.GetIsSliceEdgeCachingEnabled()),
			IsSliceL2CachingEnabled:     types.BoolValue(resultCacheSettings.GetIsSliceL2CachingEnabled()),
			SliceConfigurationRange:     types.Int64Value(resultCacheSettings.GetSliceConfigurationRange()),
			EnableCachingForOptions:     types.BoolValue(resultCacheSettings.GetEnableCachingForOptions()),
			EnableStaleCache:            types.BoolValue(resultCacheSettings.GetEnableStaleCache()),
			L2Region:                    types.StringValue(resultCacheSettings.GetL2Region()),
		})
	}

	edgeApplicationsCacheSettingsState.ID = types.StringValue("Get By ID Cache Settings")
	diags := resp.State.Set(ctx, &edgeApplicationsCacheSettingsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
