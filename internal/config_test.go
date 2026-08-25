package provider

import "testing"

// TestV4EntrypointOverride pins the split between the two entrypoint variables.
// AZION_API_ENTRYPOINT must never reach the V4 clients: the legacy host serves
// no /v4 paths, so leaking it there breaks every V4 resource. And V4 has to be
// redirectable on its own, or a functional-test run cannot be aimed away from
// production - which is the whole reason AZION_API_V4_ENTRYPOINT exists.
func TestV4EntrypointOverride(t *testing.T) {
	const prod = "https://api.azion.com/v4"

	t.Run("defaults to production when unset", func(t *testing.T) {
		c := Client("tok", "ua")
		if got := c.apiConfig.Servers[0].URL; got != prod {
			t.Errorf("apiConfig = %q, want %q", got, prod)
		}
		if got := c.edgeConfig.Servers[0].URL; got != prod {
			t.Errorf("edgeConfig = %q, want %q", got, prod)
		}
	})

	t.Run("redirects both V4 clients when set", func(t *testing.T) {
		const stage = "https://stage.example.invalid/v4"
		t.Setenv("AZION_API_V4_ENTRYPOINT", stage)
		c := Client("tok", "ua")
		if got := c.apiConfig.Servers[0].URL; got != stage {
			t.Errorf("apiConfig = %q, want %q", got, stage)
		}
		if got := c.edgeConfig.Servers[0].URL; got != stage {
			t.Errorf("edgeConfig = %q, want %q", got, stage)
		}
	})

	t.Run("legacy entrypoint does not redirect V4", func(t *testing.T) {
		const legacy = "https://legacy.example.invalid"
		t.Setenv("AZION_API_ENTRYPOINT", legacy)
		c := Client("tok", "ua")
		if got := c.idnsConfig.Servers[0].URL; got != legacy {
			t.Errorf("idnsConfig = %q, want %q", got, legacy)
		}
		if got := c.apiConfig.Servers[0].URL; got != prod {
			t.Errorf("apiConfig leaked legacy entrypoint: %q, want %q", got, prod)
		}
	})
}
