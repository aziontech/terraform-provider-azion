package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	LastUpdated   types.String                         `tfsdk:"last_updated"`
}

type EdgeApplicationCacheSettingsResults struct {
	Name                        types.String `tfsdk:"name"`
	BrowserCacheSettings        types.String `tfsdk:"browser_cache_settings"`
	BrowserCacheSettingsMaxTTL  types.Int64  `tfsdk:"browser_cache_settings_maximum_ttl"`
	CDNCacheSettings            types.String `tfsdk:"cdn_cache_settings"`
	CDNCacheSettingsMaxTTL      types.Int64  `tfsdk:"cdn_cache_settings_maximum_ttl"`
	CacheByQueryString          types.String `tfsdk:"cache_by_query_string"`
	QueryStringFields           types.String `tfsdk:"query_string_fields"`
	EnableQueryStringSort       types.Bool   `tfsdk:"enable_query_string_sort"`
	CacheByCookies              types.String `tfsdk:"cache_by_cookies"`
	CookieNames                 types.String `tfsdk:"cookie_names"`
	AdaptiveDeliveryAction      types.String `tfsdk:"adaptive_delivery_action"`
	DeviceGroup                 types.List   `tfsdk:"device_group"`
	EnableCachingForPost        types.Bool   `tfsdk:"enable_caching_for_post"`
	L2CachingEnabled            types.Bool   `tfsdk:"l2_caching_enabled"`
	IsSliceConfigurationEnabled types.Bool   `tfsdk:"is_slice_configuration_enabled"`
	IsSliceEdgeCachingEnabled   types.Bool   `tfsdk:"is_slice_edge_caching_enabled"`
	IsSliceL2CachingEnabled     types.Bool   `tfsdk:"is_slice_l2_caching_enabled"`
	SliceConfigurationRange     types.String `tfsdk:"slice_configuration_range"`
	EnableCachingForOptions     types.Bool   `tfsdk:"enable_caching_for_options"`
	EnableStaleCache            types.Bool   `tfsdk:"enable_stale_cache"`
	L2Region                    types.String `tfsdk:"l2_region"`
}

func (r *edgeApplicationCacheSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "edge_application_cache_settings"
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
			"schema_version": schema.Int64Attribute{
				Required: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"cache_settings": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"browser_cache_settings": schema.StringAttribute{
						Optional: true,
					},
					"browser_cache_settings_maximum_ttl": schema.Int64Attribute{
						Optional: true,
					},
					"cdn_cache_settings": schema.StringAttribute{
						Optional: true,
					},
					"cdn_cache_settings_maximum_ttl": schema.Int64Attribute{
						Optional: true,
					},
					"cache_by_query_string": schema.StringAttribute{
						Optional: true,
					},
					"query_string_fields": schema.StringAttribute{
						Optional: true,
					},
					"enable_query_string_sort": schema.BoolAttribute{
						Optional: true,
					},
					"cache_by_cookies": schema.StringAttribute{
						Optional: true,
					},
					"cookie_names": schema.StringAttribute{
						Optional: true,
					},
					"adaptive_delivery_action": schema.StringAttribute{
						Optional: true,
					},
					"device_group": schema.ListAttribute{
						Optional: true,
					},
					"enable_caching_for_post": schema.BoolAttribute{
						Optional: true,
					},
					"l2_caching_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"is_slice_configuration_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"is_slice_edge_caching_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"is_slice_l2_caching_enabled": schema.BoolAttribute{
						Optional: true,
					},
					"slice_configuration_range": schema.StringAttribute{
						Optional: true,
					},
					"enable_caching_for_options": schema.BoolAttribute{
						Optional: true,
					},
					"enable_stale_cache": schema.BoolAttribute{
						Optional: true,
					},
					"l2_region": schema.StringAttribute{
						Optional: true,
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

	//cacheSettings := edgeapplications.ApplicationCacheCreateRequest{
	//	Name:                        plan.CacheSettings.Name.ValueString(),
	//	BrowserCacheSettings:        edgeapplications.PtrString(plan.CacheSettings.BrowserCacheSettings.ValueString()),
	//	BrowserCacheSettingsMaxTTL:  edgeapplications.PtrInt64(plan.CacheSettings.BrowserCacheSettingsMaximumTTL.ValueInt64()),
	//	CDNCacheSettings:            edgeapplications.PtrString(plan.CacheSettings.CDNCacheSettings.ValueString()),
	//	CDNCacheSettingsMaxTTL:      edgeapplications.PtrInt64(plan.CacheSettings.CDNCacheSettingsMaximumTTL.ValueInt64()),
	//	CacheByQueryString:          edgeapplications.PtrString(plan.CacheSettings.CacheByQueryString.ValueString()),
	//	QueryStringFields:           edgeapplications.PtrString(plan.CacheSettings.QueryStringFields.ValueString()),
	//	EnableQueryStringSort:       edgeapplications.PtrBool(plan.CacheSettings.EnableQueryStringSort.ValueBool()),
	//	CacheByCookies:              edgeapplications.PtrString(plan.CacheSettings.CacheByCookies.ValueString()),
	//	CookieNames:                 edgeapplications.PtrString(plan.CacheSettings.CookieNames.ValueString()),
	//	AdaptiveDeliveryAction:      edgeapplications.PtrString(plan.CacheSettings.AdaptiveDeliveryAction.ValueString()),
	//	DeviceGroup:                 edgeapplications.PtrString(plan.CacheSettings.DeviceGroup.ValueString()),
	//	EnableCachingForPost:        edgeapplications.PtrBool(plan.CacheSettings.EnableCachingForPost.ValueBool()),
	//	L2CachingEnabled:            edgeapplications.PtrBool(plan.CacheSettings.L2CachingEnabled.ValueBool()),
	//	IsSliceConfigurationEnabled: edgeapplications.PtrBool(plan.CacheSettings.IsSliceConfigurationEnabled.ValueBool()),
	//}
	//createdCacheSetting, response, err := r.client.edgeApplicationsApi.EdgeApplicationsCacheSettingsApi.EdgeApplicationsEdgeApplicationIdCacheSettingsPost(ctx, edgeid).ApplicationCacheCreateRequest(cacheSettings).Execute()
	//if err != nil {
	//	bodyBytes, erro := io.ReadAll(response.Body)
	//	if erro != nil {
	//		resp.Diagnostics.AddError(
	//			err.Error(),
	//			"err",
	//		)
	//	}
	//	bodyString := string(bodyBytes)
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		bodyString,
	//	)
	//	return
	//}
	//
	//model.ID = types.Int64(createdCacheSetting.ID)
	//resp.State = req.Plan
	//resp.State.Set(ctx, &model)
}

func (r *edgeApplicationCacheSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	//var model CacheSettingsResourceModel
	//if err := req.State.Get(ctx, &model); err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//cacheSetting, err := r.client.GetCacheSetting(ctx, model.ID.Value())
	//if err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//model = createModelFromCacheSetting(cacheSetting)
	//resp.State.Set(ctx, &model)
}

func (r *edgeApplicationCacheSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	//var model CacheSettingsResourceModel
	//if err := req.Plan.Get(ctx, &model); err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//cacheSetting := createCacheSettingFromModel(model)
	//updatedCacheSetting, err := r.client.UpdateCacheSetting(ctx, cacheSetting)
	//if err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//model.ID = types.Int64(updatedCacheSetting.ID)
	//resp.State = req.Plan
	//resp.State.Set(ctx, &model)
}

func (r *edgeApplicationCacheSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	//var model CacheSettingsResourceModel
	//if err := req.State.Get(ctx, &model); err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//err := r.client.DeleteCacheSetting(ctx, model.ID.Value())
	//if err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//resp.State.Set(ctx, nil)
}

func (r *edgeApplicationCacheSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	//id, err := strconv.ParseInt(req.ID, 10, 64)
	//if err != nil {
	//	resp.Error(err)
	//	return
	//}
	//
	//cacheSetting, err := r.client
}
