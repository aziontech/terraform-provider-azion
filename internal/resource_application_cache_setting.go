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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	CacheVaryByMethod      types.List                           `tfsdk:"cache_vary_by_method"`
	CacheVaryByQuerystring *CacheVaryByQuerystringResourceModel `tfsdk:"cache_vary_by_querystring"`
	CacheVaryByCookies     *CacheVaryByCookiesResourceModel     `tfsdk:"cache_vary_by_cookies"`
	CacheVaryByDevices     *CacheVaryByDevicesResourceModel     `tfsdk:"cache_vary_by_devices"`
}

type CacheVaryByQuerystringResourceModel struct {
	Behavior    types.String `tfsdk:"behavior"`
	Fields      types.List   `tfsdk:"fields"`
	SortEnabled types.Bool   `tfsdk:"sort_enabled"`
}

type CacheVaryByCookiesResourceModel struct {
	Behavior    types.String `tfsdk:"behavior"`
	CookieNames types.List   `tfsdk:"cookie_names"`
}

type CacheVaryByDevicesResourceModel struct {
	Behavior    types.String `tfsdk:"behavior"`
	DeviceGroup types.List   `tfsdk:"device_group"`
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
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Description: "Name of the cache setting.",
						Required:    true,
					},
					// Every optional attribute below is also Computed: the API fills in
					// the ones the configuration omits, and state has to hold what the
					// API actually stores so `terraform plan` can report remote changes
					// to attributes that were never configured locally. UseStateForUnknown
					// keeps those attributes stable in the plan output instead of turning
					// them into "(known after apply)" on unrelated updates.
					"browser_cache": schema.SingleNestedAttribute{
						Description: "Browser cache settings.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"behavior": schema.StringAttribute{
								Description: "Browser cache behavior: override, honor, no-cache.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"max_age": schema.Int64Attribute{
								Description: "Maximum TTL for browser cache.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Int64{
									int64planmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Cache settings modules.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"cache": schema.SingleNestedAttribute{
								Description: "Edge cache module settings.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"behavior": schema.StringAttribute{
										Description: "Cache behavior: honor, override.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"max_age": schema.Int64Attribute{
										Description: "Maximum TTL for edge cache.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"stale_cache": schema.SingleNestedAttribute{
										Description: "Stale cache settings.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.Object{
											objectplanmodifier.UseStateForUnknown(),
										},
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												PlanModifiers: []planmodifier.Bool{
													boolplanmodifier.UseStateForUnknown(),
												},
											},
										},
									},
									"large_file_cache": schema.SingleNestedAttribute{
										Description: "Large file cache settings.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.Object{
											objectplanmodifier.UseStateForUnknown(),
										},
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												PlanModifiers: []planmodifier.Bool{
													boolplanmodifier.UseStateForUnknown(),
												},
											},
											"offset": schema.Int64Attribute{
												Optional: true,
												Computed: true,
												PlanModifiers: []planmodifier.Int64{
													int64planmodifier.UseStateForUnknown(),
												},
											},
										},
									},
									"tiered_cache": schema.SingleNestedAttribute{
										Description: "Tiered cache settings.",
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.Object{
											objectplanmodifier.UseStateForUnknown(),
										},
										Attributes: map[string]schema.Attribute{
											"topology": schema.StringAttribute{
												Description: "Tiered cache topology: nearest-region, br-east-1, us-east-1.",
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.String{
													stringplanmodifier.UseStateForUnknown(),
												},
											},
											"enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												PlanModifiers: []planmodifier.Bool{
													boolplanmodifier.UseStateForUnknown(),
												},
											},
										},
									},
								},
							},
							"application_accelerator": schema.SingleNestedAttribute{
								Description: "Application accelerator module settings.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Object{
									objectplanmodifier.UseStateForUnknown(),
								},
								Attributes: map[string]schema.Attribute{
									"cache_vary_by_method": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
										Computed:    true,
										PlanModifiers: []planmodifier.List{
											listplanmodifier.UseStateForUnknown(),
										},
									},
									"cache_vary_by_querystring": schema.SingleNestedAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Object{
											objectplanmodifier.UseStateForUnknown(),
										},
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Query string behavior: ignore, all, allowlist, denylist.",
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.String{
													stringplanmodifier.UseStateForUnknown(),
												},
											},
											"fields": schema.ListAttribute{
												ElementType: types.StringType,
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.List{
													listplanmodifier.UseStateForUnknown(),
												},
											},
											"sort_enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												PlanModifiers: []planmodifier.Bool{
													boolplanmodifier.UseStateForUnknown(),
												},
											},
										},
									},
									"cache_vary_by_cookies": schema.SingleNestedAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Object{
											objectplanmodifier.UseStateForUnknown(),
										},
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Cookies behavior: ignore, all, allowlist, denylist.",
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.String{
													stringplanmodifier.UseStateForUnknown(),
												},
											},
											"cookie_names": schema.ListAttribute{
												ElementType: types.StringType,
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.List{
													listplanmodifier.UseStateForUnknown(),
												},
											},
										},
									},
									"cache_vary_by_devices": schema.SingleNestedAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Object{
											objectplanmodifier.UseStateForUnknown(),
										},
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Devices behavior: ignore, allowlist.",
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.String{
													stringplanmodifier.UseStateForUnknown(),
												},
											},
											"device_group": schema.ListAttribute{
												ElementType: types.Int64Type,
												Optional:    true,
												Computed:    true,
												PlanModifiers: []planmodifier.List{
													listplanmodifier.UseStateForUnknown(),
												},
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
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
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
	var config ApplicationCacheSettingsResourceModel
	var applicationID types.Int64
	// Read the configuration instead of the plan: every optional attribute is also
	// Computed, so the plan carries unknown values for whatever the configuration
	// omits, and unknowns cannot be decoded into the nested pointer structs of the
	// model. The configuration is also exactly what should be sent to the API.
	diags := req.Config.Get(ctx, &config)
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
	cacheSettingRequest := azionapi.NewCacheSettingRequest(config.CacheSetting.Name.ValueString())

	// Browser Cache
	if config.CacheSetting.BrowserCache != nil {
		browserCache := azionapi.NewBrowserCacheModuleRequest()
		if isSet(config.CacheSetting.BrowserCache.Behavior) {
			browserCache.SetBehavior(config.CacheSetting.BrowserCache.Behavior.ValueString())
		}
		if isSet(config.CacheSetting.BrowserCache.MaxAge) {
			browserCache.SetMaxAge(config.CacheSetting.BrowserCache.MaxAge.ValueInt64())
		}
		cacheSettingRequest.SetBrowserCache(*browserCache)
	}

	// Modules
	if config.CacheSetting.Modules != nil {
		modulesRequest := buildModulesRequest(config.CacheSetting.Modules)
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

	// State mirrors what the API actually stored, so a remote change to an attribute
	// that was never configured locally shows up on the next plan instead of being
	// silently absorbed. Values the API leaves out fall back to the configuration,
	// because an applied value must not contradict a known planned value.
	cacheSettingResult := transformCacheSettingResponseToResourceModel(&createdCacheSetting.Data)
	fillMissingFromConfig(cacheSettingResult, config.CacheSetting)

	config.CacheSetting = cacheSettingResult
	config.ApplicationID = applicationID
	config.ID = types.Int64Value(createdCacheSetting.Data.GetId())
	config.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &config)
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
	// Refresh state with everything the API reports, including attributes that were
	// never configured locally. Anything less hides remote changes from the plan.
	// Attributes absent from the configuration are Optional+Computed, so state
	// holding an API-supplied value does not produce a perpetual diff.
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
	var config ApplicationCacheSettingsResourceModel
	var applicationID types.Int64
	var cacheID types.Int64
	// Same reason as in Create: the plan carries unknown values for the
	// optional+computed attributes the configuration omits, so the request is built
	// from the configuration - which is also what should be sent to the API.
	diags := req.Config.Get(ctx, &config)
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

	if config.ApplicationID.IsNull() {
		applicationID = state.ApplicationID
	} else {
		applicationID = config.ApplicationID
	}

	if config.CacheSetting.ID.IsNull() || config.CacheSetting.ID.ValueInt64() == 0 {
		cacheID = state.CacheSetting.ID
	} else {
		cacheID = config.CacheSetting.ID
	}

	// Build patched request
	patchedRequest := azionapi.NewPatchedCacheSettingRequest()

	if isSet(config.CacheSetting.Name) {
		patchedRequest.SetName(config.CacheSetting.Name.ValueString())
	}

	// Browser Cache
	if config.CacheSetting.BrowserCache != nil {
		browserCache := azionapi.NewBrowserCacheModuleRequest()
		if isSet(config.CacheSetting.BrowserCache.Behavior) {
			browserCache.SetBehavior(config.CacheSetting.BrowserCache.Behavior.ValueString())
		}
		if isSet(config.CacheSetting.BrowserCache.MaxAge) {
			browserCache.SetMaxAge(config.CacheSetting.BrowserCache.MaxAge.ValueInt64())
		}
		patchedRequest.SetBrowserCache(*browserCache)
	}

	// Modules
	if config.CacheSetting.Modules != nil {
		modulesRequest := buildModulesRequest(config.CacheSetting.Modules)
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

	// See the comment in Create: state holds what the API stored, with the
	// configuration filling in whatever the response leaves out.
	cacheSettingResult := transformCacheSettingResponseToResourceModel(&updatedCacheSetting.Data)
	fillMissingFromConfig(cacheSettingResult, config.CacheSetting)

	state.CacheSetting = cacheSettingResult
	state.ApplicationID = applicationID
	state.ID = types.Int64Value(updatedCacheSetting.Data.GetId())
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &state)
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

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.ApplicationsCacheSettingsAPI.
			DeleteCacheSetting(ctx, applicationId, cacheSettingId).
			Execute()
	}, 5)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
		bodyBytes, errReadAll := io.ReadAll(response.Body)
		if errReadAll != nil {
			resp.Diagnostics.AddError(errReadAll.Error(), "err")
		}
		bodyString := string(bodyBytes)
		resp.Diagnostics.AddError(err.Error(), bodyString)
		return
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

// isSet reports whether an attribute carries a usable value. Optional+Computed
// attributes are unknown in the plan whenever the configuration omits them, and an
// unknown value must never be sent to the API as its Go zero value.
func isSet(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}

// stringSliceFromList converts a list attribute into the slice the SDK expects,
// returning nil when the list carries no usable value.
func stringSliceFromList(list types.List) []string {
	if !isSet(list) {
		return nil
	}
	values := make([]string, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		if value, ok := element.(types.String); ok && isSet(value) {
			values = append(values, value.ValueString())
		}
	}
	return values
}

// int64SliceFromList converts a list attribute into the slice the SDK expects,
// returning nil when the list carries no usable value.
func int64SliceFromList(list types.List) []int64 {
	if !isSet(list) {
		return nil
	}
	values := make([]int64, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		if value, ok := element.(types.Int64); ok && isSet(value) {
			values = append(values, value.ValueInt64())
		}
	}
	return values
}

// stringListValue builds a list attribute from an API response slice.
func stringListValue(values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, elements)
}

// int64ListValue builds a list attribute from an API response slice.
func int64ListValue(values []int64) types.List {
	if values == nil {
		return types.ListNull(types.Int64Type)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.Int64Value(value))
	}
	return types.ListValueMust(types.Int64Type, elements)
}

// Helper: Build Modules Request.
func buildModulesRequest(modules *CacheSettingsModulesResourceModel) *azionapi.CacheSettingsModulesRequest {
	modulesRequest := azionapi.NewCacheSettingsModulesRequest()

	if modules.Cache != nil {
		cacheRequest := azionapi.NewCacheSettingsCacheModuleRequest()

		if isSet(modules.Cache.Behavior) {
			cacheRequest.SetBehavior(modules.Cache.Behavior.ValueString())
		}
		if isSet(modules.Cache.MaxAge) {
			cacheRequest.SetMaxAge(modules.Cache.MaxAge.ValueInt64())
		}

		if modules.Cache.StaleCache != nil {
			staleCache := azionapi.NewStateCacheModuleRequest()
			if isSet(modules.Cache.StaleCache.Enabled) {
				staleCache.SetEnabled(modules.Cache.StaleCache.Enabled.ValueBool())
			}
			cacheRequest.SetStaleCache(*staleCache)
		}

		if modules.Cache.TieredCache != nil {
			tieredCache := azionapi.NewCacheSettingsTieredCacheModuleRequest()
			if isSet(modules.Cache.TieredCache.Topology) {
				tieredCache.SetTopology(modules.Cache.TieredCache.Topology.ValueString())
			}
			if isSet(modules.Cache.TieredCache.Enabled) {
				tieredCache.SetEnabled(modules.Cache.TieredCache.Enabled.ValueBool())
			}
			cacheRequest.SetTieredCache(*tieredCache)
		}

		if modules.Cache.LargeFileCache != nil {
			largeFileCache := azionapi.NewLargeFileCacheModuleRequest()
			if isSet(modules.Cache.LargeFileCache.Enabled) {
				largeFileCache.SetEnabled(modules.Cache.LargeFileCache.Enabled.ValueBool())
			}
			if isSet(modules.Cache.LargeFileCache.Offset) {
				largeFileCache.SetOffset(modules.Cache.LargeFileCache.Offset.ValueInt64())
			}
			cacheRequest.SetLargeFileCache(*largeFileCache)
		}

		modulesRequest.SetCache(*cacheRequest)
	}

	if modules.ApplicationAccelerator != nil {
		aa := modules.ApplicationAccelerator
		aaRequest := azionapi.NewCacheSettingsApplicationAcceleratorModuleRequest()

		if methods := stringSliceFromList(aa.CacheVaryByMethod); len(methods) > 0 {
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

	if isSet(qs.Behavior) {
		request.SetBehavior(qs.Behavior.ValueString())
	}
	if fields := stringSliceFromList(qs.Fields); len(fields) > 0 {
		request.SetFields(fields)
	}
	if isSet(qs.SortEnabled) {
		request.SetSortEnabled(qs.SortEnabled.ValueBool())
	}

	return request
}

func buildCookiesRequest(cookies *CacheVaryByCookiesResourceModel) *azionapi.CacheVaryByCookiesModuleRequest {
	request := azionapi.NewCacheVaryByCookiesModuleRequest()

	if isSet(cookies.Behavior) {
		request.SetBehavior(cookies.Behavior.ValueString())
	}
	if names := stringSliceFromList(cookies.CookieNames); len(names) > 0 {
		request.SetCookieNames(names)
	}

	return request
}

func buildDevicesRequest(devices *CacheVaryByDevicesResourceModel) *azionapi.CacheVaryByDevicesModuleRequest {
	request := azionapi.NewCacheVaryByDevicesModuleRequest()

	if isSet(devices.Behavior) {
		request.SetBehavior(devices.Behavior.ValueString())
	}
	if groups := int64SliceFromList(devices.DeviceGroup); len(groups) > 0 {
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

			// Tiered Cache. The field is nullable, so an explicit null has to stay
			// null in state instead of becoming an object with null attributes.
			if cache.TieredCache.Get() != nil {
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
			model.Modules.ApplicationAccelerator = &CacheSettingsAppAcceleratorResourceModel{
				CacheVaryByMethod: stringListValue(aa.GetCacheVaryByMethod()),
			}

			// Cache Vary By Querystring
			if aa.HasCacheVaryByQuerystring() {
				qs := aa.GetCacheVaryByQuerystring()
				model.Modules.ApplicationAccelerator.CacheVaryByQuerystring = &CacheVaryByQuerystringResourceModel{
					Fields: stringListValue(qs.GetFields()),
				}

				if qs.HasBehavior() {
					model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Behavior = types.StringValue(qs.GetBehavior())
				}
				if qs.HasSortEnabled() {
					model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.SortEnabled = types.BoolValue(qs.GetSortEnabled())
				}
			}

			// Cache Vary By Cookies
			if aa.HasCacheVaryByCookies() {
				cookies := aa.GetCacheVaryByCookies()
				model.Modules.ApplicationAccelerator.CacheVaryByCookies = &CacheVaryByCookiesResourceModel{
					CookieNames: stringListValue(cookies.GetCookieNames()),
				}

				if cookies.HasBehavior() {
					model.Modules.ApplicationAccelerator.CacheVaryByCookies.Behavior = types.StringValue(cookies.GetBehavior())
				}
			}

			// Cache Vary By Devices
			if aa.HasCacheVaryByDevices() {
				devices := aa.GetCacheVaryByDevices()
				model.Modules.ApplicationAccelerator.CacheVaryByDevices = &CacheVaryByDevicesResourceModel{
					DeviceGroup: int64ListValue(devices.GetDeviceGroup()),
				}

				if devices.HasBehavior() {
					model.Modules.ApplicationAccelerator.CacheVaryByDevices.Behavior = types.StringValue(devices.GetBehavior())
				}
			}
		}
	}

	return model
}

// fillMissingFromConfig copies configured values into result wherever the API
// response carried none. Terraform rejects an applied value that contradicts a
// known planned value, so an attribute the practitioner set must survive even if
// the API leaves it out of its response.
func fillMissingFromConfig(result, config *CacheSettingResourceModel) {
	if result == nil || config == nil {
		return
	}

	result.Name = orConfigured(result.Name, config.Name)

	if config.BrowserCache != nil {
		if result.BrowserCache == nil {
			result.BrowserCache = &BrowserCacheResourceModel{}
		}
		result.BrowserCache.Behavior = orConfigured(result.BrowserCache.Behavior, config.BrowserCache.Behavior)
		result.BrowserCache.MaxAge = orConfigured(result.BrowserCache.MaxAge, config.BrowserCache.MaxAge)
	}

	if config.Modules == nil {
		return
	}
	if result.Modules == nil {
		result.Modules = &CacheSettingsModulesResourceModel{}
	}

	if cacheConfig := config.Modules.Cache; cacheConfig != nil {
		if result.Modules.Cache == nil {
			result.Modules.Cache = &CacheSettingsCacheResourceModel{}
		}
		cache := result.Modules.Cache
		cache.Behavior = orConfigured(cache.Behavior, cacheConfig.Behavior)
		cache.MaxAge = orConfigured(cache.MaxAge, cacheConfig.MaxAge)

		if cacheConfig.StaleCache != nil {
			if cache.StaleCache == nil {
				cache.StaleCache = &StateCacheResourceModel{}
			}
			cache.StaleCache.Enabled = orConfigured(cache.StaleCache.Enabled, cacheConfig.StaleCache.Enabled)
		}

		if cacheConfig.LargeFileCache != nil {
			if cache.LargeFileCache == nil {
				cache.LargeFileCache = &LargeFileCacheResourceModel{}
			}
			cache.LargeFileCache.Enabled = orConfigured(cache.LargeFileCache.Enabled, cacheConfig.LargeFileCache.Enabled)
			cache.LargeFileCache.Offset = orConfigured(cache.LargeFileCache.Offset, cacheConfig.LargeFileCache.Offset)
		}

		if cacheConfig.TieredCache != nil {
			if cache.TieredCache == nil {
				cache.TieredCache = &CacheSettingsTieredCacheResourceModel{}
			}
			cache.TieredCache.Topology = orConfigured(cache.TieredCache.Topology, cacheConfig.TieredCache.Topology)
			cache.TieredCache.Enabled = orConfigured(cache.TieredCache.Enabled, cacheConfig.TieredCache.Enabled)
		}
	}

	acceleratorConfig := config.Modules.ApplicationAccelerator
	if acceleratorConfig == nil {
		return
	}
	if result.Modules.ApplicationAccelerator == nil {
		result.Modules.ApplicationAccelerator = &CacheSettingsAppAcceleratorResourceModel{
			CacheVaryByMethod: types.ListNull(types.StringType),
		}
	}
	accelerator := result.Modules.ApplicationAccelerator
	accelerator.CacheVaryByMethod = orConfigured(accelerator.CacheVaryByMethod, acceleratorConfig.CacheVaryByMethod)

	if querystringConfig := acceleratorConfig.CacheVaryByQuerystring; querystringConfig != nil {
		if accelerator.CacheVaryByQuerystring == nil {
			accelerator.CacheVaryByQuerystring = &CacheVaryByQuerystringResourceModel{
				Fields: types.ListNull(types.StringType),
			}
		}
		querystring := accelerator.CacheVaryByQuerystring
		querystring.Behavior = orConfigured(querystring.Behavior, querystringConfig.Behavior)
		querystring.Fields = orConfigured(querystring.Fields, querystringConfig.Fields)
		querystring.SortEnabled = orConfigured(querystring.SortEnabled, querystringConfig.SortEnabled)
	}

	if cookiesConfig := acceleratorConfig.CacheVaryByCookies; cookiesConfig != nil {
		if accelerator.CacheVaryByCookies == nil {
			accelerator.CacheVaryByCookies = &CacheVaryByCookiesResourceModel{
				CookieNames: types.ListNull(types.StringType),
			}
		}
		cookies := accelerator.CacheVaryByCookies
		cookies.Behavior = orConfigured(cookies.Behavior, cookiesConfig.Behavior)
		cookies.CookieNames = orConfigured(cookies.CookieNames, cookiesConfig.CookieNames)
	}

	if devicesConfig := acceleratorConfig.CacheVaryByDevices; devicesConfig != nil {
		if accelerator.CacheVaryByDevices == nil {
			accelerator.CacheVaryByDevices = &CacheVaryByDevicesResourceModel{
				DeviceGroup: types.ListNull(types.Int64Type),
			}
		}
		devices := accelerator.CacheVaryByDevices
		devices.Behavior = orConfigured(devices.Behavior, devicesConfig.Behavior)
		devices.DeviceGroup = orConfigured(devices.DeviceGroup, devicesConfig.DeviceGroup)
	}
}

// orConfigured keeps the API value when there is one and falls back to the
// configured value otherwise. An unknown configured value is never used: unknown
// may not reach state.
func orConfigured[T attr.Value](fromAPI, configured T) T {
	if !fromAPI.IsNull() {
		return fromAPI
	}
	if configured.IsUnknown() {
		return fromAPI
	}
	return configured
}
