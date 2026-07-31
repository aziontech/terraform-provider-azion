package provider

import (
	"context"
	"fmt"
	"testing"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func cacheSettingSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &fwresource.SchemaResponse{}
	NewApplicationCacheSettingsResource().Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// TestApplicationCacheSettingOptionalAttributesAreComputed guards the drift
// detection fix. Read writes back everything the API reports, including attributes
// that were never configured locally, which only stays free of perpetual diffs
// while every optional attribute is also computed.
func TestApplicationCacheSettingOptionalAttributesAreComputed(t *testing.T) {
	var walk func(path string, attributes map[string]schema.Attribute)
	walk = func(path string, attributes map[string]schema.Attribute) {
		for name, attribute := range attributes {
			current := name
			if path != "" {
				current = path + "." + name
			}

			if attribute.IsOptional() && !attribute.IsComputed() {
				t.Errorf("%s is optional but not computed: an API-supplied value would cause a perpetual diff", current)
			}

			if nested, ok := attribute.(schema.SingleNestedAttribute); ok {
				walk(current, nested.Attributes)
			}
		}
	}

	walk("", cacheSettingSchema(t).Attributes)
}

// TestTransformCacheSettingResponseKeepsAPIValues asserts the response transform
// reports every value the API returned. Gating any of them on what state already
// held is what used to hide remote changes from the plan.
func TestTransformCacheSettingResponseKeepsAPIValues(t *testing.T) {
	response := azionapi.NewCacheSetting("example")
	response.SetId(42)

	browserCache := azionapi.NewBrowserCacheModule()
	browserCache.SetBehavior("override")
	browserCache.SetMaxAge(120)
	response.SetBrowserCache(*browserCache)

	staleCache := azionapi.NewStateCacheModule()
	staleCache.SetEnabled(true)

	cache := azionapi.NewCacheSettingsCacheModule()
	cache.SetBehavior("honor")
	cache.SetMaxAge(60)
	cache.SetStaleCache(*staleCache)

	querystring := azionapi.NewCacheVaryByQuerystringModule()
	querystring.SetBehavior("allowlist")
	querystring.SetFields([]string{"page"})

	accelerator := azionapi.NewCacheSettingsApplicationAcceleratorModule()
	accelerator.SetCacheVaryByMethod([]string{"GET"})
	accelerator.SetCacheVaryByQuerystring(*querystring)

	modules := azionapi.NewCacheSettingsModules()
	modules.SetCache(*cache)
	modules.SetApplicationAccelerator(*accelerator)
	response.SetModules(*modules)

	model := transformCacheSettingResponseToResourceModel(response)

	if model.BrowserCache == nil {
		t.Fatal("browser_cache missing from state")
	}
	if got := model.BrowserCache.MaxAge; got != types.Int64Value(120) {
		t.Errorf("browser_cache.max_age = %v, want 120", got)
	}
	if model.Modules == nil || model.Modules.Cache == nil {
		t.Fatal("modules.cache missing from state")
	}
	if got := model.Modules.Cache.MaxAge; got != types.Int64Value(60) {
		t.Errorf("modules.cache.max_age = %v, want 60", got)
	}
	if model.Modules.Cache.StaleCache == nil {
		t.Fatal("modules.cache.stale_cache missing from state")
	}
	if got := model.Modules.Cache.StaleCache.Enabled; got != types.BoolValue(true) {
		t.Errorf("modules.cache.stale_cache.enabled = %v, want true", got)
	}
	if model.Modules.ApplicationAccelerator == nil {
		t.Fatal("modules.application_accelerator missing from state")
	}
	if got := fmt.Sprint(model.Modules.ApplicationAccelerator.CacheVaryByMethod); got != `["GET"]` {
		t.Errorf("cache_vary_by_method = %s, want [\"GET\"]", got)
	}
	if model.Modules.ApplicationAccelerator.CacheVaryByQuerystring == nil {
		t.Fatal("cache_vary_by_querystring missing from state")
	}
	if got := fmt.Sprint(model.Modules.ApplicationAccelerator.CacheVaryByQuerystring.Fields); got != `["page"]` {
		t.Errorf("cache_vary_by_querystring.fields = %s, want [\"page\"]", got)
	}
}

// TestFillMissingFromConfigKeepsConfiguredValues covers the consistency rule the
// API response alone cannot satisfy: a value the practitioner configured must
// survive into state even when the response omits it.
func TestFillMissingFromConfigKeepsConfiguredValues(t *testing.T) {
	result := &CacheSettingResourceModel{
		ID:   types.Int64Value(42),
		Name: types.StringValue("example"),
	}
	config := &CacheSettingResourceModel{
		Name: types.StringValue("example"),
		BrowserCache: &BrowserCacheResourceModel{
			Behavior: types.StringValue("override"),
			MaxAge:   types.Int64Unknown(),
		},
		Modules: &CacheSettingsModulesResourceModel{
			Cache: &CacheSettingsCacheResourceModel{
				Behavior: types.StringValue("honor"),
				TieredCache: &CacheSettingsTieredCacheResourceModel{
					Topology: types.StringValue("near-edge"),
				},
			},
		},
	}

	fillMissingFromConfig(result, config)

	if result.BrowserCache == nil || result.BrowserCache.Behavior != types.StringValue("override") {
		t.Errorf("configured browser_cache.behavior was dropped: %+v", result.BrowserCache)
	}
	if result.BrowserCache != nil && !result.BrowserCache.MaxAge.IsNull() {
		t.Errorf("unknown browser_cache.max_age must not reach state, got %v", result.BrowserCache.MaxAge)
	}
	if result.Modules == nil || result.Modules.Cache == nil {
		t.Fatal("configured modules.cache was dropped")
	}
	if result.Modules.Cache.Behavior != types.StringValue("honor") {
		t.Errorf("configured modules.cache.behavior = %v, want honor", result.Modules.Cache.Behavior)
	}
	if result.Modules.Cache.TieredCache == nil ||
		result.Modules.Cache.TieredCache.Topology != types.StringValue("near-edge") {
		t.Errorf("configured modules.cache.tiered_cache was dropped: %+v", result.Modules.Cache.TieredCache)
	}
}
