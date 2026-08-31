package provider

import (
	"context"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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

// Azion API defaults for cache setting fields.
//
// The configuration is the desired state: any field a practitioner leaves out is
// reset to the value below on every apply, so a console change to an undeclared
// field is undone instead of being silently adopted. That makes these values
// enforcement targets, not cosmetics.
//
// The Azion OpenAPI specification declares no `default:` for these fields (the
// generated NewXWithDefaults constructors are empty), so they mirror what the
// API applies when a field is omitted from the request. A value that disagrees
// with the API surfaces as a diff on the first plan against a field nobody
// touched — loud rather than silent, but it must be corrected here.
//
// Every optional attribute needs one. A Computed attribute with no default is
// unknown in the plan whenever the configuration omits it, and reading an
// unknown object into the pointer fields of this model is a hard error
// ("the target type cannot handle unknown values"), so a missing default breaks
// Create rather than merely weakening enforcement.
const (
	defaultBrowserCacheBehavior = "honor"
	defaultBrowserCacheMaxAge   = int64(0)
	defaultCacheBehavior        = "override"
	defaultCacheMaxAge          = int64(60)
	defaultStaleCacheEnabled    = true
	defaultLargeFileEnabled     = false
	defaultLargeFileOffset      = int64(1024)
	defaultTieredCacheEnabled   = false
	defaultVaryBehavior         = "ignore"
	defaultQuerystringSort      = false
)

// Attribute types for the nested objects, needed to build object defaults whose
// types must match the schema exactly.
var (
	browserCacheAttrTypes = map[string]attr.Type{
		"behavior": types.StringType,
		"max_age":  types.Int64Type,
	}

	staleCacheAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
	}

	largeFileCacheAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
		"offset":  types.Int64Type,
	}

	tieredCacheAttrTypes = map[string]attr.Type{
		"topology": types.StringType,
		"enabled":  types.BoolType,
	}

	cacheModuleAttrTypes = map[string]attr.Type{
		"behavior":         types.StringType,
		"max_age":          types.Int64Type,
		"stale_cache":      types.ObjectType{AttrTypes: staleCacheAttrTypes},
		"large_file_cache": types.ObjectType{AttrTypes: largeFileCacheAttrTypes},
		"tiered_cache":     types.ObjectType{AttrTypes: tieredCacheAttrTypes},
	}

	varyByQuerystringAttrTypes = map[string]attr.Type{
		"behavior":     types.StringType,
		"fields":       types.ListType{ElemType: types.StringType},
		"sort_enabled": types.BoolType,
	}

	varyByCookiesAttrTypes = map[string]attr.Type{
		"behavior":     types.StringType,
		"cookie_names": types.ListType{ElemType: types.StringType},
	}

	varyByDevicesAttrTypes = map[string]attr.Type{
		"behavior":     types.StringType,
		"device_group": types.ListType{ElemType: types.Int64Type},
	}

	appAcceleratorAttrTypes = map[string]attr.Type{
		"cache_vary_by_method":      types.ListType{ElemType: types.StringType},
		"cache_vary_by_querystring": types.ObjectType{AttrTypes: varyByQuerystringAttrTypes},
		"cache_vary_by_cookies":     types.ObjectType{AttrTypes: varyByCookiesAttrTypes},
		"cache_vary_by_devices":     types.ObjectType{AttrTypes: varyByDevicesAttrTypes},
	}

	modulesAttrTypes = map[string]attr.Type{
		"cache":                   types.ObjectType{AttrTypes: cacheModuleAttrTypes},
		"application_accelerator": types.ObjectType{AttrTypes: appAcceleratorAttrTypes},
	}
)

var (
	browserCacheDefault = types.ObjectValueMust(browserCacheAttrTypes, map[string]attr.Value{
		"behavior": types.StringValue(defaultBrowserCacheBehavior),
		"max_age":  types.Int64Value(defaultBrowserCacheMaxAge),
	})

	staleCacheDefault = types.ObjectValueMust(staleCacheAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(defaultStaleCacheEnabled),
	})

	largeFileCacheDefault = types.ObjectValueMust(largeFileCacheAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(defaultLargeFileEnabled),
		"offset":  types.Int64Value(defaultLargeFileOffset),
	})

	// The API returns a null topology for a cache setting whose tiered cache was
	// never configured, so null is the value to enforce. Enforcing a concrete
	// topology instead makes every apply fail with "unexpected new value:
	// .tiered_cache.topology: was ..., but now null".
	tieredCacheDefault = types.ObjectValueMust(tieredCacheAttrTypes, map[string]attr.Value{
		"topology": types.StringNull(),
		"enabled":  types.BoolValue(defaultTieredCacheEnabled),
	})

	cacheModuleDefault = types.ObjectValueMust(cacheModuleAttrTypes, map[string]attr.Value{
		"behavior":         types.StringValue(defaultCacheBehavior),
		"max_age":          types.Int64Value(defaultCacheMaxAge),
		"stale_cache":      staleCacheDefault,
		"large_file_cache": largeFileCacheDefault,
		"tiered_cache":     tieredCacheDefault,
	})

	varyByQuerystringDefault = types.ObjectValueMust(varyByQuerystringAttrTypes, map[string]attr.Value{
		"behavior":     types.StringValue(defaultVaryBehavior),
		"fields":       types.ListValueMust(types.StringType, []attr.Value{}),
		"sort_enabled": types.BoolValue(defaultQuerystringSort),
	})

	varyByCookiesDefault = types.ObjectValueMust(varyByCookiesAttrTypes, map[string]attr.Value{
		"behavior":     types.StringValue(defaultVaryBehavior),
		"cookie_names": types.ListValueMust(types.StringType, []attr.Value{}),
	})

	varyByDevicesDefault = types.ObjectValueMust(varyByDevicesAttrTypes, map[string]attr.Value{
		"behavior":     types.StringValue(defaultVaryBehavior),
		"device_group": types.ListValueMust(types.Int64Type, []attr.Value{}),
	})

	appAcceleratorDefault = types.ObjectValueMust(appAcceleratorAttrTypes, map[string]attr.Value{
		"cache_vary_by_method":      types.ListValueMust(types.StringType, []attr.Value{}),
		"cache_vary_by_querystring": varyByQuerystringDefault,
		"cache_vary_by_cookies":     varyByCookiesDefault,
		"cache_vary_by_devices":     varyByDevicesDefault,
	})

	modulesDefault = types.ObjectValueMust(modulesAttrTypes, map[string]attr.Value{
		"cache":                   cacheModuleDefault,
		"application_accelerator": appAcceleratorDefault,
	})
)

// nullStringDefault enforces null for an omitted attribute. The framework ships
// stringdefault.StaticString only, but null is a real desired value here: the API
// reports no topology for a tiered cache that was never configured, and a
// Computed attribute left without any default would instead be unknown on every
// plan whose prior state is null, showing a change that never settles.
type nullStringDefault struct{}

func (d nullStringDefault) Description(_ context.Context) string {
	return "value defaults to null"
}

func (d nullStringDefault) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (d nullStringDefault) DefaultString(_ context.Context, _ defaults.StringRequest, resp *defaults.StringResponse) {
	resp.PlanValue = types.StringNull()
}

// topologyFollowsTieredCache defers to the API for a topology the configuration
// does not declare while tiered cache is being enabled. Defaulting it to null
// would be wrong there — the API assigns a topology when the module is on — and
// an attribute default cannot make that distinction, since defaults.StringRequest
// carries only a path, not the configuration.
type topologyFollowsTieredCache struct{}

func (m topologyFollowsTieredCache) Description(_ context.Context) string {
	return "leaves topology to the API when tiered cache is enabled without one"
}

func (m topologyFollowsTieredCache) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m topologyFollowsTieredCache) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// An explicitly configured topology is the desired state; leave it alone.
	if !req.ConfigValue.IsNull() {
		return
	}

	var enabled types.Bool
	diags := req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName("enabled"), &enabled)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Tiered cache off: the API reports no topology, so keep the null default and
	// reset a topology somebody set out-of-band.
	if !enabled.IsUnknown() && !enabled.ValueBool() {
		return
	}

	// Tiered cache on. The API owns the topology, so follow the value already in
	// state; forcing unknown once it is known would show "known after apply" on
	// every plan.
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		return
	}

	resp.PlanValue = types.StringUnknown()
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
						Description: "Browser cache settings. Omitting this block resets it to the Azion defaults on every apply.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(browserCacheDefault),
						Attributes: map[string]schema.Attribute{
							"behavior": schema.StringAttribute{
								Description: "Browser cache behavior: override, honor, no-cache.",
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString(defaultBrowserCacheBehavior),
							},
							"max_age": schema.Int64Attribute{
								Description: "Maximum TTL for browser cache.",
								Optional:    true,
								Computed:    true,
								Default:     int64default.StaticInt64(defaultBrowserCacheMaxAge),
							},
						},
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Cache settings modules.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(modulesDefault),
						Attributes: map[string]schema.Attribute{
							"cache": schema.SingleNestedAttribute{
								Description: "Edge cache module settings.",
								Optional:    true,
								Computed:    true,
								Default:     objectdefault.StaticValue(cacheModuleDefault),
								Attributes: map[string]schema.Attribute{
									"behavior": schema.StringAttribute{
										Description: "Cache behavior: honor, override.",
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString(defaultCacheBehavior),
									},
									"max_age": schema.Int64Attribute{
										Description: "Maximum TTL for edge cache.",
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(defaultCacheMaxAge),
									},
									"stale_cache": schema.SingleNestedAttribute{
										Description: "Stale cache settings.",
										Optional:    true,
										Computed:    true,
										Default:     objectdefault.StaticValue(staleCacheDefault),
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												Default:  booldefault.StaticBool(defaultStaleCacheEnabled),
											},
										},
									},
									"large_file_cache": schema.SingleNestedAttribute{
										Description: "Large file cache settings.",
										Optional:    true,
										Computed:    true,
										Default:     objectdefault.StaticValue(largeFileCacheDefault),
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												Default:  booldefault.StaticBool(defaultLargeFileEnabled),
											},
											"offset": schema.Int64Attribute{
												Optional: true,
												Computed: true,
												Default:  int64default.StaticInt64(defaultLargeFileOffset),
											},
										},
									},
									"tiered_cache": schema.SingleNestedAttribute{
										Description: "Tiered cache settings. Requires the tiered cache module on the application.",
										Optional:    true,
										Computed:    true,
										Default:     objectdefault.StaticValue(tieredCacheDefault),
										Attributes: map[string]schema.Attribute{
											"topology": schema.StringAttribute{
												Description: "Tiered cache topology: nearest-region, br-east-1, us-east-1.",
												Optional:    true,
												Computed:    true,
												Default:     nullStringDefault{},
												PlanModifiers: []planmodifier.String{
													topologyFollowsTieredCache{},
												},
											},
											"enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												Default:  booldefault.StaticBool(defaultTieredCacheEnabled),
											},
										},
									},
								},
							},
							"application_accelerator": schema.SingleNestedAttribute{
								Description: "Application accelerator module settings. Requires the application accelerator module on the application.",
								Optional:    true,
								Computed:    true,
								Default:     objectdefault.StaticValue(appAcceleratorDefault),
								Attributes: map[string]schema.Attribute{
									"cache_vary_by_method": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
										Computed:    true,
										Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
									},
									"cache_vary_by_querystring": schema.SingleNestedAttribute{
										Optional: true,
										Computed: true,
										Default:  objectdefault.StaticValue(varyByQuerystringDefault),
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Query string behavior: ignore, all, allowlist, denylist.",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString(defaultVaryBehavior),
											},
											"fields": schema.ListAttribute{
												ElementType: types.StringType,
												Optional:    true,
												Computed:    true,
												Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
											},
											"sort_enabled": schema.BoolAttribute{
												Optional: true,
												Computed: true,
												Default:  booldefault.StaticBool(defaultQuerystringSort),
											},
										},
									},
									"cache_vary_by_cookies": schema.SingleNestedAttribute{
										Optional: true,
										Computed: true,
										Default:  objectdefault.StaticValue(varyByCookiesDefault),
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Cookies behavior: ignore, all, allowlist, denylist.",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString(defaultVaryBehavior),
											},
											"cookie_names": schema.ListAttribute{
												ElementType: types.StringType,
												Optional:    true,
												Computed:    true,
												Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
											},
										},
									},
									"cache_vary_by_devices": schema.SingleNestedAttribute{
										Optional: true,
										Computed: true,
										Default:  objectdefault.StaticValue(varyByDevicesDefault),
										Attributes: map[string]schema.Attribute{
											"behavior": schema.StringAttribute{
												Description: "Devices behavior: ignore, allowlist.",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString(defaultVaryBehavior),
											},
											"device_group": schema.ListAttribute{
												ElementType: types.Int64Type,
												Optional:    true,
												Computed:    true,
												Default:     listdefault.StaticValue(types.ListValueMust(types.Int64Type, []attr.Value{})),
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

	// Build the V4 API request from the plan, which carries the schema defaults
	// for every field the configuration left out.
	cacheSettingRequest := buildCacheSettingRequest(plan.CacheSetting)

	// Call V4 API
	createdCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		CreateCacheSetting(ctx, applicationID.ValueInt64()).
		CacheSettingRequest(*cacheSettingRequest).
		Execute()
	if err != nil {
		if response != nil && response.StatusCode == 429 {
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// State mirrors exactly what the API returned. Every attribute is Computed,
	// so all of them must resolve to a known value here.
	cacheSettingData, ok := createdCacheSetting.GetDataOk()
	if !ok || cacheSettingData == nil {
		resp.Diagnostics.AddError("Empty response", "cacheSettingResponse has no data after successful API call")
		return
	}

	plan.CacheSetting = transformCacheSettingResponseToResourceModel(cacheSettingData)
	plan.ApplicationID = applicationID
	plan.ID = types.Int64Value(cacheSettingData.GetId())
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
	var cacheSettingId int64
	if state.CacheSetting != nil {
		cacheSettingId = state.CacheSetting.ID.ValueInt64()
	} else {
		cacheSettingId = state.ID.ValueInt64()
	}

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
				resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
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
	// Refresh state from the API without filtering: state has to mirror the
	// remote resource for Terraform to plan a revert when it was changed
	// out-of-band. Fields the configuration omits do not drift perpetually
	// because they are Computed with a schema default, so the plan resolves them
	// to that default rather than to null.
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

	// Full replacement, not a partial update: the configuration is the desired
	// state, so every field is asserted on every apply. A PATCH would leave a
	// console change to an undeclared field in place forever.
	cacheSettingRequest := buildCacheSettingRequest(plan.CacheSetting)

	// Call V4 API PUT
	updatedCacheSetting, response, err := r.client.api.ApplicationsCacheSettingsAPI.
		UpdateCacheSetting(ctx, applicationID.ValueInt64(), cacheID.ValueInt64()).
		CacheSettingRequest(*cacheSettingRequest).
		Execute()
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			updatedCacheSetting, response, err = utils.RetryOn429(func() (*azionapi.CacheSettingResponse, *http.Response, error) {
				return r.client.api.ApplicationsCacheSettingsAPI.
					UpdateCacheSetting(ctx, applicationID.ValueInt64(), cacheID.ValueInt64()).
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// State mirrors exactly what the API returned.
	cacheSettingData, ok := updatedCacheSetting.GetDataOk()
	if !ok || cacheSettingData == nil {
		resp.Diagnostics.AddError("Empty response", "cacheSettingResponse has no data after successful API call")
		return
	}

	plan.CacheSetting = transformCacheSettingResponseToResourceModel(cacheSettingData)
	plan.ApplicationID = applicationID
	plan.ID = types.Int64Value(cacheSettingData.GetId())
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
	var cacheSettingId int64
	if state.CacheSetting != nil {
		cacheSettingId = state.CacheSetting.ID.ValueInt64()
	} else {
		cacheSettingId = state.ID.ValueInt64()
	}

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
		resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
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
				resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
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
// buildCacheSettingRequest builds the full API payload from a plan. Shared by
// Create (POST) and Update (PUT): both send the complete desired state, so a
// field the practitioner omitted carries its schema default rather than being
// left to whatever the remote currently holds.
func buildCacheSettingRequest(cs *CacheSettingResourceModel) *azionapi.CacheSettingRequest {
	request := azionapi.NewCacheSettingRequest(cs.Name.ValueString())

	if cs.BrowserCache != nil {
		browserCache := azionapi.NewBrowserCacheModuleRequest()
		if isSet(cs.BrowserCache.Behavior) {
			browserCache.SetBehavior(cs.BrowserCache.Behavior.ValueString())
		}
		if isSet(cs.BrowserCache.MaxAge) {
			browserCache.SetMaxAge(cs.BrowserCache.MaxAge.ValueInt64())
		}
		request.SetBrowserCache(*browserCache)
	}

	if cs.Modules != nil {
		request.SetModules(*buildModulesRequest(cs.Modules))
	}

	return request
}

// isSet reports whether a value carries something to send to the API. Unknown is
// excluded as well as null, so a value Terraform has not resolved yet is omitted
// instead of being sent as a zero value.
func isSet(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}

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

		if aa.CacheVaryByMethod != nil {
			// Allocated rather than declared: the SDK omits a nil slice from the
			// body, so an empty list would leave the remote value untouched
			// instead of clearing it.
			methods := make([]string, 0, len(aa.CacheVaryByMethod))
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

	if isSet(qs.Behavior) {
		request.SetBehavior(qs.Behavior.ValueString())
	}
	if qs.Fields != nil {
		fields := make([]string, 0, len(qs.Fields))
		for _, f := range qs.Fields {
			fields = append(fields, f.ValueString())
		}
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
	if cookies.CookieNames != nil {
		names := make([]string, 0, len(cookies.CookieNames))
		for _, n := range cookies.CookieNames {
			names = append(names, n.ValueString())
		}
		request.SetCookieNames(names)
	}

	return request
}

func buildDevicesRequest(devices *CacheVaryByDevicesResourceModel) *azionapi.CacheVaryByDevicesModuleRequest {
	request := azionapi.NewCacheVaryByDevicesModuleRequest()

	if isSet(devices.Behavior) {
		request.SetBehavior(devices.Behavior.ValueString())
	}
	if devices.DeviceGroup != nil {
		groups := make([]int64, 0, len(devices.DeviceGroup))
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
				methods := aa.GetCacheVaryByMethod()
				model.Modules.ApplicationAccelerator.CacheVaryByMethod = make([]types.String, 0, len(methods))
				for _, method := range methods {
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
					fields := qs.GetFields()
					model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields = make([]types.String, 0, len(fields))
					for _, f := range fields {
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
					names := cookies.GetCookieNames()
					model.Modules.ApplicationAccelerator.CacheVaryByCookies.CookieNames = make([]types.String, 0, len(names))
					for _, cn := range names {
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
					groups := devices.GetDeviceGroup()
					model.Modules.ApplicationAccelerator.CacheVaryByDevices.DeviceGroup = make([]types.Int64, 0, len(groups))
					for _, dg := range groups {
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
