package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CacheSettingDataSource{}
	_ datasource.DataSourceWithConfigure = &CacheSettingDataSource{}
)

func dataSourceAzionApplicationCacheSetting() datasource.DataSource {
	return &CacheSettingDataSource{}
}

type CacheSettingDataSource struct {
	client *apiClient
}

type CacheSettingDataSourceModel struct {
	ApplicationID types.Int64        `tfsdk:"application_id"`
	Results       *CacheSettingModel `tfsdk:"results"`
	ID            types.Int64        `tfsdk:"id"`
}

// Model structs matching V4 API structure.
type CacheSettingModel struct {
	ID           types.Int64                `tfsdk:"id"`
	Name         types.String               `tfsdk:"name"`
	BrowserCache *BrowserCacheModuleModel   `tfsdk:"browser_cache"`
	Modules      *CacheSettingsModulesModel `tfsdk:"modules"`
	CreatedAt    types.String               `tfsdk:"created_at"`
}

type BrowserCacheModuleModel struct {
	Behavior types.String `tfsdk:"behavior"`
	MaxAge   types.Int64  `tfsdk:"max_age"`
}

type CacheSettingsModulesModel struct {
	Cache                  *CacheSettingsCacheModuleModel        `tfsdk:"cache"`
	ApplicationAccelerator *CacheSettingsApplicationAcceleratorModel `tfsdk:"application_accelerator"`
}

type CacheSettingsCacheModuleModel struct {
	Behavior       types.String                   `tfsdk:"behavior"`
	MaxAge         types.Int64                    `tfsdk:"max_age"`
	StaleCache     *StateCacheModuleModel         `tfsdk:"stale_cache"`
	LargeFileCache *LargeFileCacheModuleModel     `tfsdk:"large_file_cache"`
	TieredCache    *CacheSettingsTieredCacheModel `tfsdk:"tiered_cache"`
}

type CacheSettingsApplicationAcceleratorModel struct {
	CacheVaryByMethod      []types.String                     `tfsdk:"cache_vary_by_method"`
	CacheVaryByQuerystring *CacheVaryByQuerystringModuleModel `tfsdk:"cache_vary_by_querystring"`
	CacheVaryByCookies     *CacheVaryByCookiesModuleModel     `tfsdk:"cache_vary_by_cookies"`
	CacheVaryByDevices     *CacheVaryByDevicesModuleModel     `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringModuleModel struct {
	Behavior    types.String   `tfsdk:"behavior"`
	Fields      []types.String `tfsdk:"fields"`
	SortEnabled types.Bool     `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesModuleModel struct {
	Behavior    types.String   `tfsdk:"behavior"`
	CookieNames []types.String `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesModuleModel struct {
	Behavior    types.String  `tfsdk:"behavior"`
	DeviceGroup []types.Int64 `tfsdk:"device_group"`
}

type StateCacheModuleModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type CacheSettingsTieredCacheModel struct {
	Topology types.String `tfsdk:"topology"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

type LargeFileCacheModuleModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Offset  types.Int64 `tfsdk:"offset"`
}

func (d *CacheSettingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *CacheSettingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_cache_setting"
}

func (d *CacheSettingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Application.",
				Required:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The cache setting identifier.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the cache setting.",
						Computed:    true,
					},
					"browser_cache": schema.SingleNestedAttribute{
						Description: "Browser cache settings.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"behavior": schema.StringAttribute{
								Description: "Browser cache behavior: override, honor, no-cache.",
								Computed:    true,
							},
							"max_age": schema.Int64Attribute{
								Description: "Maximum TTL for browser cache.",
								Computed:    true,
							},
						},
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Cache settings modules.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"cache": schema.SingleNestedAttribute{
								Description: "Edge cache module settings.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"behavior": schema.StringAttribute{
										Description: "Cache behavior: honor, override.",
										Computed:    true,
									},
									"max_age": schema.Int64Attribute{
										Description: "Maximum TTL for edge cache.",
										Computed:    true,
									},
									"stale_cache": schema.SingleNestedAttribute{
										Description: "Stale cache settings.",
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Computed: true,
											},
										},
									},
									"large_file_cache": schema.SingleNestedAttribute{
										Description: "Large file cache settings.",
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Computed: true,
											},
											"offset": schema.Int64Attribute{
												Computed: true,
											},
										},
									},
									"tiered_cache": schema.SingleNestedAttribute{
										Description: "Tiered cache settings.",
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"topology": schema.StringAttribute{
												Description: "Tiered cache topology.",
												Computed:    true,
											},
											"enabled": schema.BoolAttribute{
												Computed: true,
											},
										},
									},
								},
							},
							"application_accelerator": schema.SingleNestedAttribute{
								Description: "Application accelerator module settings.",
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"cache_vary_by_method": schema.ListAttribute{
										ElementType: types.StringType,
										Computed:    true,
									},
									"cache_vary_by_querystring": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{Computed: true},
											"fields": schema.ListAttribute{
												ElementType: types.StringType,
												Computed:    true,
											},
											"sort_enabled": schema.BoolAttribute{Computed: true},
										},
									},
									"cache_vary_by_cookies": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{Computed: true},
											"cookie_names": schema.ListAttribute{
												ElementType: types.StringType,
												Computed:    true,
											},
										},
									},
									"cache_vary_by_devices": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{Computed: true},
											"device_group": schema.ListAttribute{
												ElementType: types.Int64Type,
												Computed:    true,
											},
										},
									},
								},
							},
						},
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the cache setting.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *CacheSettingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var applicationID types.Int64
	var cacheSettingID types.Int64

	diags := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &cacheSettingID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use raw HTTP request to work around SDK validation issue with data wrapper
	cacheSetting, err := retrieveCacheSettingRawDS(ctx, d.client, applicationID.ValueInt64(), cacheSettingID.ValueInt64())
	if err != nil {
		if err.Error() == "404" {
			resp.Diagnostics.AddError("Cache setting not found", "")
			return
		}
		resp.Diagnostics.AddError("Failed to retrieve cache setting", err.Error())
		return
	}

	// Transform API response to state model
	result := transformCacheSettingToModel(cacheSetting)

	state := CacheSettingDataSourceModel{
		ApplicationID: applicationID,
		Results:       result,
		ID:            types.Int64Value(cacheSettingID.ValueInt64()),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// retrieveCacheSettingRawDS makes a raw HTTP request and manually parses the response
// to work around SDK validation issues with the data wrapper.
func retrieveCacheSettingRawDS(ctx context.Context, client *apiClient, applicationId, cacheSettingId int64) (*azionapi.CacheSetting, error) {
	// Build the request URL
	url := fmt.Sprintf("https://api.azion.com/v4/edge_applications/%d/cache_settings/%d", applicationId, cacheSettingId)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers from the SDK config
	for k, v := range client.apiConfig.DefaultHeader {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("User-Agent", client.apiConfig.UserAgent)

	// Get HTTP client from config, or use default
	httpClient := client.apiConfig.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Execute the request
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status codes
	if httpResp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404")
	}
	if httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s - %s", httpResp.Status, string(bodyBytes))
	}

	// Handle rate limiting
	if httpResp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited")
	}

	// Parse the response - the API returns {"data": {...}} wrapper
	var wrapper struct {
		Data azionapi.CacheSetting `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &wrapper.Data, nil
}

// Transform function for converting SDK response to model.
func transformCacheSettingToModel(cs *azionapi.CacheSetting) *CacheSettingModel {
	if cs == nil {
		return nil
	}

	model := &CacheSettingModel{
		ID:   types.Int64Value(cs.GetId()),
		Name: types.StringValue(cs.GetName()),
	}

	// CreatedAt - handle NullableTime
	if cs.CreatedAt.IsSet() && cs.CreatedAt.Get() != nil {
		model.CreatedAt = types.StringValue(cs.GetCreatedAt().Format(time.RFC3339))
	}

	// Browser Cache
	if cs.HasBrowserCache() {
		bc := cs.GetBrowserCache()
		model.BrowserCache = &BrowserCacheModuleModel{}
		if bc.HasBehavior() {
			model.BrowserCache.Behavior = types.StringValue(bc.GetBehavior())
		}
		if bc.HasMaxAge() {
			model.BrowserCache.MaxAge = types.Int64Value(bc.GetMaxAge())
		}
	}

	// Modules
	if cs.HasModules() {
		modules := cs.GetModules()
		model.Modules = &CacheSettingsModulesModel{}

		// Cache (Edge Cache)
		if modules.HasCache() {
			cache := modules.GetCache()
			model.Modules.Cache = &CacheSettingsCacheModuleModel{}

			if cache.HasBehavior() {
				model.Modules.Cache.Behavior = types.StringValue(cache.GetBehavior())
			}
			if cache.HasMaxAge() {
				model.Modules.Cache.MaxAge = types.Int64Value(cache.GetMaxAge())
			}

			// Stale Cache
			if cache.HasStaleCache() {
				sc := cache.GetStaleCache()
				model.Modules.Cache.StaleCache = &StateCacheModuleModel{}
				if sc.HasEnabled() {
					model.Modules.Cache.StaleCache.Enabled = types.BoolValue(sc.GetEnabled())
				}
			}

			// Large File Cache
			if cache.HasLargeFileCache() {
				lfc := cache.GetLargeFileCache()
				model.Modules.Cache.LargeFileCache = &LargeFileCacheModuleModel{}
				if lfc.HasEnabled() {
					model.Modules.Cache.LargeFileCache.Enabled = types.BoolValue(lfc.GetEnabled())
				}
				if lfc.HasOffset() {
					model.Modules.Cache.LargeFileCache.Offset = types.Int64Value(lfc.GetOffset())
				}
			}

			// Tiered Cache
			if cache.HasTieredCache() {
				tc := cache.GetTieredCache()
				model.Modules.Cache.TieredCache = &CacheSettingsTieredCacheModel{}
				if tc.HasTopology() {
					model.Modules.Cache.TieredCache.Topology = types.StringValue(tc.GetTopology())
				}
				if tc.HasEnabled() {
					model.Modules.Cache.TieredCache.Enabled = types.BoolValue(tc.GetEnabled())
				}
			}
		}

		// Application Accelerator
		if modules.HasApplicationAccelerator() {
			aa := modules.GetApplicationAccelerator()
			model.Modules.ApplicationAccelerator = &CacheSettingsApplicationAcceleratorModel{}

			// Cache Vary By Method
			if aa.HasCacheVaryByMethod() {
				for _, method := range aa.GetCacheVaryByMethod() {
					model.Modules.ApplicationAccelerator.CacheVaryByMethod = append(
						model.Modules.ApplicationAccelerator.CacheVaryByMethod,
						types.StringValue(method),
					)
				}
			}

			// Cache Vary By Querystring
			if aa.HasCacheVaryByQuerystring() {
				qs := aa.GetCacheVaryByQuerystring()
				model.Modules.ApplicationAccelerator.CacheVaryByQuerystring = &CacheVaryByQuerystringModuleModel{}

				if qs.HasBehavior() {
					model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Behavior = types.StringValue(qs.GetBehavior())
				}
				if qs.HasFields() {
					for _, f := range qs.GetFields() {
						model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields = append(
							model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields,
							types.StringValue(f),
						)
					}
				}
				if qs.HasSortEnabled() {
					model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.SortEnabled = types.BoolValue(qs.GetSortEnabled())
				}
			}

			// Cache Vary By Cookies
			if aa.HasCacheVaryByCookies() {
				cookies := aa.GetCacheVaryByCookies()
				model.Modules.ApplicationAccelerator.CacheVaryByCookies = &CacheVaryByCookiesModuleModel{}

				if cookies.HasBehavior() {
					model.Modules.ApplicationAccelerator.CacheVaryByCookies.Behavior = types.StringValue(cookies.GetBehavior())
				}
				if cookies.HasCookieNames() {
					for _, cn := range cookies.GetCookieNames() {
						model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames = append(
							model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames,
							types.StringValue(cn),
						)
					}
				}
			}

			// Cache Vary By Devices
			if aa.HasCacheVaryByDevices() {
				devices := aa.GetCacheVaryByDevices()
				model.Modules.ApplicationAccelerator.CacheVaryByDevices = &CacheVaryByDevicesModuleModel{}

				if devices.HasBehavior() {
					model.Modules.ApplicationAccelerator.CacheVaryByDevices.Behavior = types.StringValue(devices.GetBehavior())
				}
				if devices.HasDeviceGroup() {
					for _, dg := range devices.GetDeviceGroup() {
						model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup = append(
							model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup,
							types.Int64Value(dg),
						)
					}
				}
			}
		}
	}

	return model
}
