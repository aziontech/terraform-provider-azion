package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccZonesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "azion_zones" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_zones.test", "counter", "3"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "links.#", "0"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.#", "3"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.0.id", "2580"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.1.id", "2581"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.2.id", "2583"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "total_pages", "1"),
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.azion_zones.test", "id", "placeholder"),
				),
			},
		},
	})
}
