package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tfPath turns a dotted attribute name into a schema path.
func tfPath(t *testing.T, dotted string) path.Path {
	t.Helper()

	segments := strings.Split(dotted, ".")
	result := path.Root(segments[0])

	for _, segment := range segments[1:] {
		result = result.AtName(segment)
	}

	return result
}

// The schema uses attribute defaults inside the blocks it enforces. A default
// whose type disagrees with the schema is only rejected at plan time, which would
// surface as a runtime error for practitioners rather than a build failure, so
// assert it here.
func TestApplicationCacheSettingSchemaImplementation(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationCacheSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResponse.Diagnostics)
	}

	if diags := schemaResponse.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Errorf("invalid schema implementation: %v", diags)
	}
}

// Every enforced field must be Computed, otherwise its default is never applied
// and a value changed in the console is silently kept instead of reverted. The
// objects that hold them are blocks, which cannot be Computed; an omitted block
// is covered by withCacheSettingDefaults instead.
func TestApplicationCacheSettingEnforcedFieldsAreComputed(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationCacheSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	enforced := []string{
		"cache_setting.browser_cache.behavior",
		"cache_setting.browser_cache.max_age",
		"cache_setting.modules.cache.behavior",
		"cache_setting.modules.cache.max_age",
		"cache_setting.modules.cache.stale_cache.enabled",
		"cache_setting.modules.cache.large_file_cache.enabled",
		"cache_setting.modules.cache.large_file_cache.offset",
		"cache_setting.modules.cache.tiered_cache.enabled",
		"cache_setting.modules.cache.tiered_cache.topology",
		"cache_setting.modules.application_accelerator.cache_vary_by_method",
		"cache_setting.modules.application_accelerator.cache_vary_by_querystring.behavior",
		"cache_setting.modules.application_accelerator.cache_vary_by_querystring.fields",
		"cache_setting.modules.application_accelerator.cache_vary_by_querystring.sort_enabled",
		"cache_setting.modules.application_accelerator.cache_vary_by_cookies.behavior",
		"cache_setting.modules.application_accelerator.cache_vary_by_cookies.cookie_names",
		"cache_setting.modules.application_accelerator.cache_vary_by_devices.behavior",
		"cache_setting.modules.application_accelerator.cache_vary_by_devices.device_group",
	}

	for _, name := range enforced {
		attribute, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name))
		if diags.HasError() {
			t.Errorf("%s: %v", name, diags)
			continue
		}

		if !attribute.IsComputed() {
			t.Errorf("%s: must be Computed for its default to be enforced", name)
		}

		if !attribute.IsOptional() {
			t.Errorf("%s: must stay Optional so it can be declared in configuration", name)
		}
	}
}

// Nested objects are blocks so Terraform rejects unknown arguments instead of
// dropping them. A nested attribute here would silently accept a typo.
func TestApplicationCacheSettingNestedObjectsAreBlocks(t *testing.T) {
	ctx := context.Background()

	schemaResponse := &fwresource.SchemaResponse{}
	NewApplicationCacheSettingsResource().Schema(ctx, fwresource.SchemaRequest{}, schemaResponse)

	for _, name := range []string{
		"cache_setting",
		"cache_setting.browser_cache",
		"cache_setting.modules",
		"cache_setting.modules.cache",
		"cache_setting.modules.cache.stale_cache",
		"cache_setting.modules.cache.large_file_cache",
		"cache_setting.modules.cache.tiered_cache",
		"cache_setting.modules.application_accelerator",
		"cache_setting.modules.application_accelerator.cache_vary_by_querystring",
		"cache_setting.modules.application_accelerator.cache_vary_by_cookies",
		"cache_setting.modules.application_accelerator.cache_vary_by_devices",
	} {
		if _, diags := schemaResponse.Schema.AttributeAtPath(ctx, tfPath(t, name)); !diags.HasError() {
			t.Errorf("%s: must be a block, not a nested attribute", name)
		}
	}
}

// fullyDefaultedCacheSetting is the plan the schema produced while the nested
// objects were attributes with object defaults and the configuration declared
// only the name: every object present and holding its defaults.
func fullyDefaultedCacheSetting() *CacheSettingResourceModel {
	return &CacheSettingResourceModel{
		Name:         types.StringValue("cache"),
		BrowserCache: defaultBrowserCacheModel(),
		Modules: &CacheSettingsModulesResourceModel{
			Cache: &CacheSettingsCacheResourceModel{
				Behavior:       types.StringValue(defaultCacheBehavior),
				MaxAge:         types.Int64Value(defaultCacheMaxAge),
				StaleCache:     defaultStaleCacheModel(),
				LargeFileCache: defaultLargeFileCacheModel(),
				TieredCache:    defaultTieredCacheModel(),
			},
			ApplicationAccelerator: &CacheSettingsAppAcceleratorResourceModel{
				CacheVaryByMethod:      []types.String{},
				CacheVaryByQuerystring: defaultQuerystringModel(),
				CacheVaryByCookies:     defaultCookiesModel(),
				CacheVaryByDevices:     defaultDevicesModel(),
			},
		},
	}
}

func cacheSettingRequestJSON(t *testing.T, cs *CacheSettingResourceModel) string {
	t.Helper()

	body, err := json.Marshal(buildCacheSettingRequest(cs))
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return string(body)
}

// An omitted block must still send its defaults, so the request is the same one
// sent while the nested objects were attributes with object defaults, and a
// console change to an undeclared field is still reset on apply.
func TestBuildCacheSettingRequestSendsDefaultsForOmittedBlocks(t *testing.T) {
	want := cacheSettingRequestJSON(t, fullyDefaultedCacheSetting())

	t.Run("no optional blocks", func(t *testing.T) {
		got := cacheSettingRequestJSON(t, &CacheSettingResourceModel{Name: types.StringValue("cache")})
		if got != want {
			t.Errorf("request:\n got %s\nwant %s", got, want)
		}
	})

	t.Run("empty modules block", func(t *testing.T) {
		got := cacheSettingRequestJSON(t, &CacheSettingResourceModel{
			Name:    types.StringValue("cache"),
			Modules: &CacheSettingsModulesResourceModel{},
		})
		if got != want {
			t.Errorf("request:\n got %s\nwant %s", got, want)
		}
	})

	t.Run("declared values are kept", func(t *testing.T) {
		expected := fullyDefaultedCacheSetting()
		expected.Modules.Cache.MaxAge = types.Int64Value(3600)
		expected.Modules.Cache.StaleCache = &StateCacheResourceModel{Enabled: types.BoolValue(false)}

		got := cacheSettingRequestJSON(t, &CacheSettingResourceModel{
			Name: types.StringValue("cache"),
			Modules: &CacheSettingsModulesResourceModel{
				Cache: &CacheSettingsCacheResourceModel{
					Behavior:   types.StringValue(defaultCacheBehavior),
					MaxAge:     types.Int64Value(3600),
					StaleCache: &StateCacheResourceModel{Enabled: types.BoolValue(false)},
				},
			},
		})
		if wantDeclared := cacheSettingRequestJSON(t, expected); got != wantDeclared {
			t.Errorf("request:\n got %s\nwant %s", got, wantDeclared)
		}
	})

	t.Run("plan is not modified", func(t *testing.T) {
		plan := &CacheSettingResourceModel{Name: types.StringValue("cache")}
		buildCacheSettingRequest(plan)
		if plan.BrowserCache != nil || plan.Modules != nil {
			t.Errorf("plan modified: %+v", plan)
		}
	})
}

// Terraform rejects a block planned as absent that comes back present, so the
// API result must not add blocks the configuration left out. Import has no
// prior value and keeps everything.
func TestKeepDeclaredCacheSettingBlocks(t *testing.T) {
	t.Run("no optional blocks", func(t *testing.T) {
		results := fullyDefaultedCacheSetting()
		keepDeclaredCacheSettingBlocks(&CacheSettingResourceModel{}, results)
		if results.BrowserCache != nil || results.Modules != nil {
			t.Errorf("undeclared blocks kept: %+v", results)
		}
	})

	t.Run("partially declared", func(t *testing.T) {
		results := fullyDefaultedCacheSetting()
		keepDeclaredCacheSettingBlocks(&CacheSettingResourceModel{
			BrowserCache: &BrowserCacheResourceModel{},
			Modules: &CacheSettingsModulesResourceModel{
				Cache: &CacheSettingsCacheResourceModel{TieredCache: &CacheSettingsTieredCacheResourceModel{}},
				ApplicationAccelerator: &CacheSettingsAppAcceleratorResourceModel{
					CacheVaryByCookies: &CacheVaryByCookiesResourceModel{},
				},
			},
		}, results)

		if results.BrowserCache == nil || results.Modules == nil || results.Modules.Cache == nil {
			t.Fatalf("declared blocks dropped: %+v", results)
		}
		cache := results.Modules.Cache
		if cache.TieredCache == nil {
			t.Error("tiered_cache: dropped, want kept")
		}
		if cache.StaleCache != nil || cache.LargeFileCache != nil {
			t.Errorf("undeclared cache blocks kept: %+v", cache)
		}
		accelerator := results.Modules.ApplicationAccelerator
		if accelerator == nil || accelerator.CacheVaryByCookies == nil {
			t.Fatalf("declared accelerator blocks dropped: %+v", accelerator)
		}
		if accelerator.CacheVaryByQuerystring != nil || accelerator.CacheVaryByDevices != nil {
			t.Errorf("undeclared accelerator blocks kept: %+v", accelerator)
		}
	})

	t.Run("modules without accelerator", func(t *testing.T) {
		results := fullyDefaultedCacheSetting()
		keepDeclaredCacheSettingBlocks(&CacheSettingResourceModel{
			Modules: &CacheSettingsModulesResourceModel{Cache: &CacheSettingsCacheResourceModel{}},
		}, results)
		if results.Modules.ApplicationAccelerator != nil {
			t.Error("application_accelerator: kept, want dropped")
		}
		if results.Modules.Cache == nil || results.Modules.Cache.StaleCache != nil {
			t.Errorf("cache: %+v", results.Modules.Cache)
		}
	})

	t.Run("import", func(t *testing.T) {
		results := fullyDefaultedCacheSetting()
		keepDeclaredCacheSettingBlocks(nil, results)
		if results.BrowserCache == nil || results.Modules == nil || results.Modules.ApplicationAccelerator.CacheVaryByDevices == nil {
			t.Errorf("import dropped blocks: %+v", results)
		}
	})
}
