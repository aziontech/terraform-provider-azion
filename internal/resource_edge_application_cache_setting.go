package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &applicationCacheSettingsResource{}
	_ resource.ResourceWithConfigure   = &applicationCacheSettingsResource{}
	_ resource.ResourceWithImportState = &applicationCacheSettingsResource{}
)

func NewApplicationCacheSettingsResource() resource.Resource {
	return &applicationCacheSettingsResource{}
}

type applicationCacheSettingsResource struct {
	client *apiClient
}

// Resource model matching V4 API structure.
type ApplicationCacheSettingsResourceModel struct {
	ApplicationID types.Int64                `tfsdk:"application_id"`
	CacheSetting  *CacheSettingResourceModel `tfsdk:"cache_setting"`
	ID            types.Int64                `tfsdk:"id"`
	LastUpdated   types.String               `tfsdk:"last_updated"`
}

type CacheSettingResourceModel struct {
	ID           types.Int64                        `tfsdk:"id"`
	Name         types.String                       `tfsdk:"name"`
	BrowserCache *BrowserCacheResourceModel         `tfsdk:"browser_cache"`
	Modules      *CacheSettingsModulesResourceModel `tfsdk:"modules"`
	CreatedAt    types.String                       `tfsdk:"created_at"`
}

type BrowserCacheResourceModel struct {
	Behavior types.String `tfsdk:"behavior"`
	MaxAge   types.Int64  `tfsdk:"max_age"`
}

type CacheSettingsModulesResourceModel struct {
	Cache                  *CacheSettingsCacheResourceModel          `tfsdk:"cache"`
	ApplicationAccelerator *CacheSettingsAppAcceleratorResourceModel `tfsdk:"application_accelerator"`
}

type CacheSettingsCacheResourceModel struct {
	Behavior       types.String                           `tfsdk:"behavior"`
	MaxAge         types.Int64                            `tfsdk:"max_age"`
	StaleCache     *StateCacheResourceModel               `tfsdk:"stale_cache"`
	LargeFileCache *LargeFileCacheResourceModel           `tfsdk:"large_file_cache"`
	TieredCache    *CacheSettingsTieredCacheResourceModel `tfsdk:"tiered_cache"`
}

type CacheSettingsAppAcceleratorResourceModel struct {
	CacheVaryByMethod      []types.String                       `tfsdk:"cache_vary_by_method"`
	CacheVaryByQuerystring *CacheVaryByQuerystringResourceModel `tfsdk:"cache_vary_by_querystring"`
	CacheVaryByCookies     *CacheVaryByCookiesResourceModel     `tfsdk:"cache_vary_by_cookies"`
	CacheVaryByDevices     *CacheVaryByDevicesResourceModel     `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringResourceModel struct {
	Behavior    types.String   `tfsdk:"behavior"`
	Fields      []types.String `tfsdk:"fields"`
	SortEnabled types.Bool     `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesResourceModel struct {
	Behavior    types.String   `tfsdk:"behavior"`
	CookieNames []types.String `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesResourceModel struct {
	Behavior    types.String  `tfsdk:"behavior"`
	DeviceGroup []types.Int64 `tfsdk:"device_group"`
}

type StateCacheResourceModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type CacheSettingsTieredCacheResourceModel struct {
	Topology types.String `tfsdk:"topology"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

type LargeFileCacheResourceModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Offset  types.Int64 `tfsdk:"offset"`
}

func (r *applicationCacheSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_cache_setting"
}

func (r *applicationCacheSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Resource identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Application.",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"cache_setting": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Cache setting identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the cache setting.",
						Required:    true,
					},
					"browser_cache": schema.SingleNestedAttribute{
						Description: "Browser cache settings.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"behavior": schema.StringAttribute{
								Description: "Browser cache behavior: override, honor, no-cache.",
								Optional:    true,
							},
							"max_age": schema.Int64Attribute{
								Description: "Maximum TTL for browser cache.",
								Optional:    true,
							},
						},
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Cache settings modules.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"cache": schema.SingleNestedAttribute{
								Description: "Edge cache module settings.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"behavior": schema.StringAttribute{
										Description: "Cache behavior: honor, override.",
										Optional:    true,
									},
									"max_age": schema.Int64Attribute{
										Description: "Maximum TTL for edge cache.",
										Optional:    true,
									},
									"stale_cache": schema.SingleNestedAttribute{
										Description: "Stale cache settings.",
										Optional:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Optional: true,
											},
										},
									},
									"large_file_cache": schema.SingleNestedAttribute{
										Description: "Large file cache settings.",
										Optional:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Optional: true,
											},
											"offset": schema.Int64Attribute{
												Optional: true,
											},
										},
									},
									"tiered_cache": schema.SingleNestedAttribute{
										Description: "Tiered cache settings.",
										Optional:    true,
										Attributes: map[string]schema.Attribute{
											"topology": schema.StringAttribute{
												Description: "Tiered cache topology: nearest-region, br-east-1, us-east-1.",
												Optional:    true,
											},
											"enabled": schema.BoolAttribute{
												Optional: true,
											},
										},
									},
								},
							},
							"application_accelerator": schema.SingleNestedAttribute{
								Description: "Application accelerator module settings.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"cache_vary_by_method": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
									},
									"cache_vary_by_querystring": schema.SingleNestedAttribute{
										Optional: true,
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Query string behavior: ignore, all, allowlist, denylist.",
												Optional:    true,
											},
											"fields": schema.ListAttribute{
												ElementType: types.StringType,
												Optional:    true,
											},
											"sort_enabled": schema.BoolAttribute{
												Optional: true,
											},
										},
									},
									"cache_vary_by_cookies": schema.SingleNestedAttribute{
										Optional: true,
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Cookies behavior: ignore, all, allowlist, denylist.",
												Optional:    true,
											},
											"cookie_names": schema.ListAttribute{
												ElementType: types.StringType,
												Optional:    true,
											},
										},
									},
									"cache_vary_by_devices": schema.SingleNestedAttribute{
										Optional: true,
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Devices behavior: ignore, allowlist.",
												Optional:    true,
											},
											"device_group": schema.ListAttribute{
												ElementType: types.Int64Type,
												Optional:    true,
											},
										},
									},
								},
							},
						},
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *applicationCacheSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *applicationCacheSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationCacheSettingsResourceModel
	var applicationID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
	resp.Diagnostics.Append(diagsApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the V4 API request
	cacheSettingRequest := azionapi.NewCacheSettingRequest(plan.CacheSetting.Name.ValueString())

	// Browser Cache
	if plan.CacheSetting.BrowserCache != nil {
		browserCache := azionapi.NewBrowserCacheModuleRequest()
		if !plan.CacheSetting.BrowserCache.Behavior.IsNull() {
			browserCache.SetBehavior(plan.CacheSetting.BrowserCache.Behavior.ValueString())
		}
		if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() {
			browserCache.SetMaxAge(plan.CacheSetting.BrowserCache.MaxAge.ValueInt64())
		}
		cacheSettingRequest.SetBrowserCache(*browserCache)
	}

	// Modules
	if plan.CacheSetting.Modules != nil {
		modulesRequest := buildModulesRequest(plan.CacheSetting.Modules)
		cacheSettingRequest.SetModules(*modulesRequest)
	}

	// Call V4 API
	createdCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		CreateCacheSetting(ctx, applicationID.ValueInt64()).
		CacheSettingRequest(*cacheSettingRequest).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			createdCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
				return r.client.api.ApplicationsCacheSettingsAPI.
					CreateCacheSetting(ctx, applicationID.ValueInt64()).
					CacheSettingRequest(*cacheSettingRequest).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(errReadAll.Error(), "err")
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(err.Error(), bodyString)
			response.Body.Close()
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Build result starting with plan values (prevents inconsistency errors when nested objects were null)
	cacheSettingResult := &CacheSettingResourceModel{
		ID:           types.Int64Value(createdCacheSetting.Data.GetId()),
		Name:         types.StringValue(createdCacheSetting.Data.GetName()),
		BrowserCache: plan.CacheSetting.BrowserCache,
		Modules:      plan.CacheSetting.Modules,
	}

	// Only update browser_cache from API response if the plan had it specified
	// Start with plan values to preserve fields not set in the plan
	if plan.CacheSetting.BrowserCache != nil && createdCacheSetting.Data.HasBrowserCache() {
		bc := createdCacheSetting.Data.GetBrowserCache()
		cacheSettingResult.BrowserCache = &BrowserCacheResourceModel{
			Behavior: plan.CacheSetting.BrowserCache.Behavior,
			MaxAge:   plan.CacheSetting.BrowserCache.MaxAge,
		}
		// Only update behavior from API if it was set in the plan
		if !plan.CacheSetting.BrowserCache.Behavior.IsNull() && bc.HasBehavior() {
			cacheSettingResult.BrowserCache.Behavior = types.StringValue(bc.GetBehavior())
		}
		// Only update max_age from API if it was set in the plan
		if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() && bc.HasMaxAge() {
			cacheSettingResult.BrowserCache.MaxAge = types.Int64Value(bc.GetMaxAge())
		}
	}

	// Only update modules from API response if the plan had modules specified
	if plan.CacheSetting.Modules != nil && createdCacheSetting.Data.HasModules() {
		modulesResp := createdCacheSetting.Data.GetModules()
		modulesResult := &CacheSettingsModulesResourceModel{}

		// Cache module - only populate if specified in plan
		if plan.CacheSetting.Modules.Cache != nil && modulesResp.HasCache() {
			cacheResp := modulesResp.GetCache()
			cacheResult := &CacheSettingsCacheResourceModel{}

			// Only populate behavior if it was set in the plan
			if !plan.CacheSetting.Modules.Cache.Behavior.IsNull() && cacheResp.HasBehavior() {
				cacheResult.Behavior = types.StringValue(cacheResp.GetBehavior())
			}
			// Only populate max_age if it was set in the plan
			if !plan.CacheSetting.Modules.Cache.MaxAge.IsNull() && cacheResp.HasMaxAge() {
				cacheResult.MaxAge = types.Int64Value(cacheResp.GetMaxAge())
			}

			// StaleCache - only if in plan
			if plan.CacheSetting.Modules.Cache.StaleCache != nil && cacheResp.HasStaleCache() {
				sc := cacheResp.GetStaleCache()
				cacheResult.StaleCache = &StateCacheResourceModel{}
				// Only populate enabled if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.StaleCache.Enabled.IsNull() && sc.HasEnabled() {
					cacheResult.StaleCache.Enabled = types.BoolValue(sc.GetEnabled())
				}
			}

			// LargeFileCache - only if in plan
			if plan.CacheSetting.Modules.Cache.LargeFileCache != nil && cacheResp.HasLargeFileCache() {
				lfc := cacheResp.GetLargeFileCache()
				cacheResult.LargeFileCache = &LargeFileCacheResourceModel{}
				// Only populate enabled if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.LargeFileCache.Enabled.IsNull() && lfc.HasEnabled() {
					cacheResult.LargeFileCache.Enabled = types.BoolValue(lfc.GetEnabled())
				}
				// Only populate offset if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.LargeFileCache.Offset.IsNull() && lfc.HasOffset() {
					cacheResult.LargeFileCache.Offset = types.Int64Value(lfc.GetOffset())
				}
			}

			// TieredCache - only if in plan
			if plan.CacheSetting.Modules.Cache.TieredCache != nil && cacheResp.HasTieredCache() {
				tc := cacheResp.GetTieredCache()
				cacheResult.TieredCache = &CacheSettingsTieredCacheResourceModel{}
				// Only populate topology if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.TieredCache.Topology.IsNull() && tc.HasTopology() {
					cacheResult.TieredCache.Topology = types.StringValue(tc.GetTopology())
				}
				// Only populate enabled if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.TieredCache.Enabled.IsNull() && tc.HasEnabled() {
					cacheResult.TieredCache.Enabled = types.BoolValue(tc.GetEnabled())
				}
			}

			modulesResult.Cache = cacheResult
		}

		// ApplicationAccelerator - only if in plan
		if plan.CacheSetting.Modules.ApplicationAccelerator != nil && modulesResp.HasApplicationAccelerator() {
			aaResp := modulesResp.GetApplicationAccelerator()
			aaResult := &CacheSettingsAppAcceleratorResourceModel{}

			// CacheVaryByQuerystring - only if in plan
			if plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring != nil && aaResp.HasCacheVaryByQuerystring() {
				qsResp := aaResp.GetCacheVaryByQuerystring()
				qsResult := &CacheVaryByQuerystringResourceModel{}
				// Only populate behavior if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Behavior.IsNull() && qsResp.HasBehavior() {
					qsResult.Behavior = types.StringValue(qsResp.GetBehavior())
				}
				// Only populate fields if they were set in the plan
				if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields) > 0 && qsResp.HasFields() {
					for _, f := range qsResp.GetFields() {
						qsResult.Fields = append(qsResult.Fields, types.StringValue(f))
					}
				}
				// Only populate sort_enabled if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring.SortEnabled.IsNull() && qsResp.HasSortEnabled() {
					qsResult.SortEnabled = types.BoolValue(qsResp.GetSortEnabled())
				}
				aaResult.CacheVaryByQuerystring = qsResult
			}

			// CacheVaryByCookies - only if in plan
			if plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByCookies != nil && aaResp.HasCacheVaryByCookies() {
				cookiesResp := aaResp.GetCacheVaryByCookies()
				cookiesResult := &CacheVaryByCookiesResourceModel{}
				// Only populate behavior if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByCookies.Behavior.IsNull() && cookiesResp.HasBehavior() {
					cookiesResult.Behavior = types.StringValue(cookiesResp.GetBehavior())
				}
				// Only populate cookie_names if they were set in the plan
				if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames) > 0 && cookiesResp.HasCookieNames() {
					for _, cn := range cookiesResp.GetCookieNames() {
						cookiesResult.CookieNames = append(cookiesResult.CookieNames, types.StringValue(cn))
					}
				}
				aaResult.CacheVaryByCookies = cookiesResult
			}

			// CacheVaryByDevices - only if in plan
			if plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByDevices != nil && aaResp.HasCacheVaryByDevices() {
				devicesResp := aaResp.GetCacheVaryByDevices()
				devicesResult := &CacheVaryByDevicesResourceModel{}
				// Only populate behavior if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByDevices.Behavior.IsNull() && devicesResp.HasBehavior() {
					devicesResult.Behavior = types.StringValue(devicesResp.GetBehavior())
				}
				// Only populate device_group if they were set in the plan
				if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup) > 0 && devicesResp.HasDeviceGroup() {
					for _, dg := range devicesResp.GetDeviceGroup() {
						devicesResult.DeviceGroup = append(devicesResult.DeviceGroup, types.Int64Value(dg))
					}
				}
				aaResult.CacheVaryByDevices = devicesResult
			}

			// CacheVaryByMethod - only if in plan
			if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByMethod) > 0 && aaResp.HasCacheVaryByMethod() {
				for _, m := range aaResp.GetCacheVaryByMethod() {
					aaResult.CacheVaryByMethod = append(aaResult.CacheVaryByMethod, types.StringValue(m))
				}
			}

			modulesResult.ApplicationAccelerator = aaResult
		}

		cacheSettingResult.Modules = modulesResult
	}

	plan.CacheSetting = cacheSettingResult
	plan.ApplicationID = applicationID
	plan.ID = types.Int64Value(createdCacheSetting.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationCacheSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationCacheSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationId := state.ApplicationID.ValueInt64()
	cacheSettingId := state.CacheSetting.ID.ValueInt64()

	// Call V4 API to retrieve cache setting
	cacheSettingResponse, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		RetrieveCacheSetting(ctx, applicationId, cacheSettingId).
		Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
			cacheSettingResponse, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
				return r.client.api.ApplicationsCacheSettingsAPI.
					RetrieveCacheSetting(ctx, applicationId, cacheSettingId).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			if response != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(errReadAll.Error(), "err")
				}
				bodyString := string(bodyBytes)
				resp.Diagnostics.AddError(err.Error(), bodyString)
				response.Body.Close()
			}
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Debug: ensure we got a valid cache setting response
	if cacheSettingResponse == nil {
		resp.Diagnostics.AddError("Empty response", "cacheSettingResponse is nil after successful API call")
		return
	}

	// Update state with response - Read should return the full API state
	cacheSettingData, ok := cacheSettingResponse.GetDataOk()
	if !ok || cacheSettingData == nil {
		resp.Diagnostics.AddError("Empty response", "cacheSettingResponse has no data after successful API call")
		return
	}
	state.CacheSetting = transformCacheSettingResponseToResourceModel(cacheSettingData)
	// Preserve top-level ID from state if not already set (it should come from req.State.Get())
	// Only set it from CacheSetting.ID if state.ID is null/unknown
	if state.ID.IsNull() || state.ID.IsUnknown() {
		if state.CacheSetting != nil {
			state.ID = state.CacheSetting.ID
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationCacheSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationCacheSettingsResourceModel
	var applicationID types.Int64
	var cacheID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ApplicationCacheSettingsResourceModel
	diagsOrigin := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsOrigin...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ApplicationID.IsNull() {
		applicationID = state.ApplicationID
	} else {
		applicationID = plan.ApplicationID
	}

	if plan.ID.IsNull() || plan.CacheSetting.ID.ValueInt64() == 0 {
		cacheID = state.CacheSetting.ID
	} else {
		cacheID = plan.CacheSetting.ID
	}

	// Build patched request
	patchedRequest := azionapi.NewPatchedCacheSettingRequest()

	if !plan.CacheSetting.Name.IsNull() {
		patchedRequest.SetName(plan.CacheSetting.Name.ValueString())
	}

	// Browser Cache
	if plan.CacheSetting.BrowserCache != nil {
		browserCache := azionapi.NewBrowserCacheModuleRequest()
		if !plan.CacheSetting.BrowserCache.Behavior.IsNull() {
			browserCache.SetBehavior(plan.CacheSetting.BrowserCache.Behavior.ValueString())
		}
		if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() {
			browserCache.SetMaxAge(plan.CacheSetting.BrowserCache.MaxAge.ValueInt64())
		}
		patchedRequest.SetBrowserCache(*browserCache)
	}

	// Modules
	if plan.CacheSetting.Modules != nil {
		modulesRequest := buildModulesRequest(plan.CacheSetting.Modules)
		patchedRequest.SetModules(*modulesRequest)
	}

	// Call V4 API PATCH
	updatedCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		PartialUpdateCacheSetting(ctx, applicationID.ValueInt64(), cacheID.ValueInt64()).
		PatchedCacheSettingRequest(*patchedRequest).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			updatedCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
				return r.client.api.ApplicationsCacheSettingsAPI.
					PartialUpdateCacheSetting(ctx, applicationID.ValueInt64(), cacheID.ValueInt64()).
					PatchedCacheSettingRequest(*patchedRequest).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(errReadAll.Error(), "err")
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(err.Error(), bodyString)
			response.Body.Close()
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Build result starting with plan values (prevents inconsistency errors when nested objects were null)
	cacheSettingResult := &CacheSettingResourceModel{
		ID:           types.Int64Value(updatedCacheSetting.Data.GetId()),
		Name:         types.StringValue(updatedCacheSetting.Data.GetName()),
		BrowserCache: plan.CacheSetting.BrowserCache,
		Modules:      plan.CacheSetting.Modules,
	}

	// Only update browser_cache from API response if the plan had it specified
	// Start with plan values to preserve fields not set in the plan
	if plan.CacheSetting.BrowserCache != nil && updatedCacheSetting.Data.HasBrowserCache() {
		bc := updatedCacheSetting.Data.GetBrowserCache()
		cacheSettingResult.BrowserCache = &BrowserCacheResourceModel{
			Behavior: plan.CacheSetting.BrowserCache.Behavior,
			MaxAge:   plan.CacheSetting.BrowserCache.MaxAge,
		}
		// Only populate behavior if it was set in the plan
		if !plan.CacheSetting.BrowserCache.Behavior.IsNull() && bc.HasBehavior() {
			cacheSettingResult.BrowserCache.Behavior = types.StringValue(bc.GetBehavior())
		}
		// Only populate max_age if it was set in the plan
		if !plan.CacheSetting.BrowserCache.MaxAge.IsNull() && bc.HasMaxAge() {
			cacheSettingResult.BrowserCache.MaxAge = types.Int64Value(bc.GetMaxAge())
		}
	}

	// Only update modules from API response if the plan had modules specified
	if plan.CacheSetting.Modules != nil && updatedCacheSetting.Data.HasModules() {
		modulesResp := updatedCacheSetting.Data.GetModules()
		modulesResult := &CacheSettingsModulesResourceModel{}

		// Cache module - only populate if specified in plan
		if plan.CacheSetting.Modules.Cache != nil && modulesResp.HasCache() {
			cacheResp := modulesResp.GetCache()
			cacheResult := &CacheSettingsCacheResourceModel{}

			// Only populate behavior if it was set in the plan
			if !plan.CacheSetting.Modules.Cache.Behavior.IsNull() && cacheResp.HasBehavior() {
				cacheResult.Behavior = types.StringValue(cacheResp.GetBehavior())
			}
			// Only populate max_age if it was set in the plan
			if !plan.CacheSetting.Modules.Cache.MaxAge.IsNull() && cacheResp.HasMaxAge() {
				cacheResult.MaxAge = types.Int64Value(cacheResp.GetMaxAge())
			}

			// StaleCache - only if in plan
			if plan.CacheSetting.Modules.Cache.StaleCache != nil && cacheResp.HasStaleCache() {
				sc := cacheResp.GetStaleCache()
				cacheResult.StaleCache = &StateCacheResourceModel{}
				// Only populate enabled if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.StaleCache.Enabled.IsNull() && sc.HasEnabled() {
					cacheResult.StaleCache.Enabled = types.BoolValue(sc.GetEnabled())
				}
			}

			// LargeFileCache - only if in plan
			if plan.CacheSetting.Modules.Cache.LargeFileCache != nil && cacheResp.HasLargeFileCache() {
				lfc := cacheResp.GetLargeFileCache()
				cacheResult.LargeFileCache = &LargeFileCacheResourceModel{}
				// Only populate enabled if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.LargeFileCache.Enabled.IsNull() && lfc.HasEnabled() {
					cacheResult.LargeFileCache.Enabled = types.BoolValue(lfc.GetEnabled())
				}
				// Only populate offset if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.LargeFileCache.Offset.IsNull() && lfc.HasOffset() {
					cacheResult.LargeFileCache.Offset = types.Int64Value(lfc.GetOffset())
				}
			}

			// TieredCache - only if in plan
			if plan.CacheSetting.Modules.Cache.TieredCache != nil && cacheResp.HasTieredCache() {
				tc := cacheResp.GetTieredCache()
				cacheResult.TieredCache = &CacheSettingsTieredCacheResourceModel{}
				// Only populate topology if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.TieredCache.Topology.IsNull() && tc.HasTopology() {
					cacheResult.TieredCache.Topology = types.StringValue(tc.GetTopology())
				}
				// Only populate enabled if it was set in the plan
				if !plan.CacheSetting.Modules.Cache.TieredCache.Enabled.IsNull() && tc.HasEnabled() {
					cacheResult.TieredCache.Enabled = types.BoolValue(tc.GetEnabled())
				}
			}

			modulesResult.Cache = cacheResult
		}

		// ApplicationAccelerator - only if in plan
		if plan.CacheSetting.Modules.ApplicationAccelerator != nil && modulesResp.HasApplicationAccelerator() {
			aaResp := modulesResp.GetApplicationAccelerator()
			aaResult := &CacheSettingsAppAcceleratorResourceModel{}

			// CacheVaryByQuerystring - only if in plan
			if plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring != nil && aaResp.HasCacheVaryByQuerystring() {
				qsResp := aaResp.GetCacheVaryByQuerystring()
				qsResult := &CacheVaryByQuerystringResourceModel{}
				// Only populate behavior if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Behavior.IsNull() && qsResp.HasBehavior() {
					qsResult.Behavior = types.StringValue(qsResp.GetBehavior())
				}
				// Only populate fields if they were set in the plan
				if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields) > 0 && qsResp.HasFields() {
					for _, f := range qsResp.GetFields() {
						qsResult.Fields = append(qsResult.Fields, types.StringValue(f))
					}
				}
				// Only populate sort_enabled if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByQuerystring.SortEnabled.IsNull() && qsResp.HasSortEnabled() {
					qsResult.SortEnabled = types.BoolValue(qsResp.GetSortEnabled())
				}
				aaResult.CacheVaryByQuerystring = qsResult
			}

			// CacheVaryByCookies - only if in plan
			if plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByCookies != nil && aaResp.HasCacheVaryByCookies() {
				cookiesResp := aaResp.GetCacheVaryByCookies()
				cookiesResult := &CacheVaryByCookiesResourceModel{}
				// Only populate behavior if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByCookies.Behavior.IsNull() && cookiesResp.HasBehavior() {
					cookiesResult.Behavior = types.StringValue(cookiesResp.GetBehavior())
				}
				// Only populate cookie_names if they were set in the plan
				if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames) > 0 && cookiesResp.HasCookieNames() {
					for _, cn := range cookiesResp.GetCookieNames() {
						cookiesResult.CookieNames = append(cookiesResult.CookieNames, types.StringValue(cn))
					}
				}
				aaResult.CacheVaryByCookies = cookiesResult
			}

			// CacheVaryByDevices - only if in plan
			if plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByDevices != nil && aaResp.HasCacheVaryByDevices() {
				devicesResp := aaResp.GetCacheVaryByDevices()
				devicesResult := &CacheVaryByDevicesResourceModel{}
				// Only populate behavior if it was set in the plan
				if !plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByDevices.Behavior.IsNull() && devicesResp.HasBehavior() {
					devicesResult.Behavior = types.StringValue(devicesResp.GetBehavior())
				}
				// Only populate device_group if they were set in the plan
				if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup) > 0 && devicesResp.HasDeviceGroup() {
					for _, dg := range devicesResp.GetDeviceGroup() {
						devicesResult.DeviceGroup = append(devicesResult.DeviceGroup, types.Int64Value(dg))
					}
				}
				aaResult.CacheVaryByDevices = devicesResult
			}

			// CacheVaryByMethod - only if in plan
			if len(plan.CacheSetting.Modules.ApplicationAccelerator.CacheVaryByMethod) > 0 && aaResp.HasCacheVaryByMethod() {
				for _, m := range aaResp.GetCacheVaryByMethod() {
					aaResult.CacheVaryByMethod = append(aaResult.CacheVaryByMethod, types.StringValue(m))
				}
			}

			modulesResult.ApplicationAccelerator = aaResult
		}

		cacheSettingResult.Modules = modulesResult
	}

	plan.CacheSetting = cacheSettingResult
	plan.ApplicationID = applicationID
	plan.ID = types.Int64Value(updatedCacheSetting.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationCacheSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationCacheSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationId := state.ApplicationID.ValueInt64()
	cacheSettingId := state.CacheSetting.ID.ValueInt64()

	_, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		DeleteCacheSetting(ctx, applicationId, cacheSettingId).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.ApplicationsCacheSettingsAPI.
					DeleteCacheSetting(ctx, applicationId, cacheSettingId).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else if response.StatusCode != http.StatusNotFound {
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(errReadAll.Error(), "err")
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(err.Error(), bodyString)
			response.Body.Close()
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}
}

func (r *applicationCacheSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: {application_id}/{cache_setting_id}
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			"Expected format: {application_id}/{cache_setting_id}",
		)
		return
	}

	applicationId, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application ID", err.Error())
		return
	}

	cacheSettingId, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid cache setting ID", err.Error())
		return
	}

	// Set the application ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationId)...)

	// Read the cache setting using the V4 API
	cacheSettingResponse, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		RetrieveCacheSetting(ctx, applicationId, cacheSettingId).
		Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Cache setting not found", "")
			return
		}
		if response != nil && response.StatusCode == 429 {
			cacheSettingResponse, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
				return r.client.api.ApplicationsCacheSettingsAPI.
					RetrieveCacheSetting(ctx, applicationId, cacheSettingId).
					Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			if response != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(errReadAll.Error(), "err")
				}
				bodyString := string(bodyBytes)
				resp.Diagnostics.AddError(err.Error(), bodyString)
				response.Body.Close()
			}
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Ensure we got a valid response
	if cacheSettingResponse == nil {
		resp.Diagnostics.AddError("Empty response", "cacheSettingResponse is nil after successful API call")
		return
	}

	cacheSettingData, ok := cacheSettingResponse.GetDataOk()
	if !ok || cacheSettingData == nil {
		resp.Diagnostics.AddError("Empty response", "cacheSettingResponse has no data after successful API call")
		return
	}

	// Build state
	state := ApplicationCacheSettingsResourceModel{
		ApplicationID: types.Int64Value(applicationId),
		CacheSetting:  transformCacheSettingResponseToResourceModel(cacheSettingData),
		ID:            types.Int64Value(cacheSettingId),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Helper: Build Modules Request.
func buildModulesRequest(modules *CacheSettingsModulesResourceModel) *azionapi.CacheSettingsModulesRequest {
	modulesRequest := azionapi.NewCacheSettingsModulesRequest()

	if modules.Cache != nil {
		cacheRequest := azionapi.NewCacheSettingsCacheModuleRequest()

		if !modules.Cache.Behavior.IsNull() {
			cacheRequest.SetBehavior(modules.Cache.Behavior.ValueString())
		}
		if !modules.Cache.MaxAge.IsNull() {
			cacheRequest.SetMaxAge(modules.Cache.MaxAge.ValueInt64())
		}

		if modules.Cache.StaleCache != nil {
			staleCache := azionapi.NewStateCacheModuleRequest()
			if !modules.Cache.StaleCache.Enabled.IsNull() {
				staleCache.SetEnabled(modules.Cache.StaleCache.Enabled.ValueBool())
			}
			cacheRequest.SetStaleCache(*staleCache)
		}

		if modules.Cache.TieredCache != nil {
			tieredCache := azionapi.NewCacheSettingsTieredCacheModuleRequest()
			if !modules.Cache.TieredCache.Topology.IsNull() {
				tieredCache.SetTopology(modules.Cache.TieredCache.Topology.ValueString())
			}
			if !modules.Cache.TieredCache.Enabled.IsNull() {
				tieredCache.SetEnabled(modules.Cache.TieredCache.Enabled.ValueBool())
			}
			cacheRequest.SetTieredCache(*tieredCache)
		}

		if modules.Cache.LargeFileCache != nil {
			largeFileCache := azionapi.NewLargeFileCacheModuleRequest()
			if !modules.Cache.LargeFileCache.Enabled.IsNull() {
				largeFileCache.SetEnabled(modules.Cache.LargeFileCache.Enabled.ValueBool())
			}
			if !modules.Cache.LargeFileCache.Offset.IsNull() {
				largeFileCache.SetOffset(modules.Cache.LargeFileCache.Offset.ValueInt64())
			}
			cacheRequest.SetLargeFileCache(*largeFileCache)
		}

		modulesRequest.SetCache(*cacheRequest)
	}

	if modules.ApplicationAccelerator != nil {
		aa := modules.ApplicationAccelerator
		aaRequest := azionapi.NewCacheSettingsApplicationAcceleratorModuleRequest()

		if len(aa.CacheVaryByMethod) > 0 {
			var methods []string
			for _, m := range aa.CacheVaryByMethod {
				methods = append(methods, m.ValueString())
			}
			aaRequest.SetCacheVaryByMethod(methods)
		}

		if aa.CacheVaryByQuerystring != nil {
			qs := buildQuerystringRequest(aa.CacheVaryByQuerystring)
			aaRequest.SetCacheVaryByQuerystring(*qs)
		}

		if aa.CacheVaryByCookies != nil {
			cookies := buildCookiesRequest(aa.CacheVaryByCookies)
			aaRequest.SetCacheVaryByCookies(*cookies)
		}

		if aa.CacheVaryByDevices != nil {
			devices := buildDevicesRequest(aa.CacheVaryByDevices)
			aaRequest.SetCacheVaryByDevices(*devices)
		}

		modulesRequest.SetApplicationAccelerator(*aaRequest)
	}

	return modulesRequest
}

func buildQuerystringRequest(qs *CacheVaryByQuerystringResourceModel) *azionapi.CacheVaryByQuerystringModuleRequest {
	request := azionapi.NewCacheVaryByQuerystringModuleRequest()

	if !qs.Behavior.IsNull() {
		request.SetBehavior(qs.Behavior.ValueString())
	}
	if len(qs.Fields) > 0 {
		var fields []string
		for _, f := range qs.Fields {
			fields = append(fields, f.ValueString())
		}
		request.SetFields(fields)
	}
	if !qs.SortEnabled.IsNull() {
		request.SetSortEnabled(qs.SortEnabled.ValueBool())
	}

	return request
}

func buildCookiesRequest(cookies *CacheVaryByCookiesResourceModel) *azionapi.CacheVaryByCookiesModuleRequest {
	request := azionapi.NewCacheVaryByCookiesModuleRequest()

	if !cookies.Behavior.IsNull() {
		request.SetBehavior(cookies.Behavior.ValueString())
	}
	if len(cookies.CookieNames) > 0 {
		var names []string
		for _, n := range cookies.CookieNames {
			names = append(names, n.ValueString())
		}
		request.SetCookieNames(names)
	}

	return request
}

func buildDevicesRequest(devices *CacheVaryByDevicesResourceModel) *azionapi.CacheVaryByDevicesModuleRequest {
	request := azionapi.NewCacheVaryByDevicesModuleRequest()

	if !devices.Behavior.IsNull() {
		request.SetBehavior(devices.Behavior.ValueString())
	}
	if len(devices.DeviceGroup) > 0 {
		var groups []int64
		for _, g := range devices.DeviceGroup {
			groups = append(groups, g.ValueInt64())
		}
		request.SetDeviceGroup(groups)
	}

	return request
}

// Transform API response to resource model.
func transformCacheSettingResponseToResourceModel(cs *azionapi.CacheSetting) *CacheSettingResourceModel {
	if cs == nil {
		return nil
	}

	model := &CacheSettingResourceModel{
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
		model.BrowserCache = &BrowserCacheResourceModel{}
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
		model.Modules = &CacheSettingsModulesResourceModel{}

		// Cache (Edge Cache)
		if modules.HasCache() {
			cache := modules.GetCache()
			model.Modules.Cache = &CacheSettingsCacheResourceModel{}

			if cache.HasBehavior() {
				model.Modules.Cache.Behavior = types.StringValue(cache.GetBehavior())
			}
			if cache.HasMaxAge() {
				model.Modules.Cache.MaxAge = types.Int64Value(cache.GetMaxAge())
			}

			// Stale Cache
			if cache.HasStaleCache() {
				sc := cache.GetStaleCache()
				model.Modules.Cache.StaleCache = &StateCacheResourceModel{}
				if sc.HasEnabled() {
					model.Modules.Cache.StaleCache.Enabled = types.BoolValue(sc.GetEnabled())
				}
			}

			// Large File Cache
			if cache.HasLargeFileCache() {
				lfc := cache.GetLargeFileCache()
				model.Modules.Cache.LargeFileCache = &LargeFileCacheResourceModel{}
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
				model.Modules.Cache.TieredCache = &CacheSettingsTieredCacheResourceModel{}
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
			model.Modules.ApplicationAccelerator = &CacheSettingsAppAcceleratorResourceModel{}

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
				model.Modules.ApplicationAccelerator.CacheVaryByQuerystring = &CacheVaryByQuerystringResourceModel{}

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
				model.Modules.ApplicationAccelerator.CacheVaryByCookies = &CacheVaryByCookiesResourceModel{}

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
				model.Modules.ApplicationAccelerator.CacheVaryByDevices = &CacheVaryByDevicesResourceModel{}

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
