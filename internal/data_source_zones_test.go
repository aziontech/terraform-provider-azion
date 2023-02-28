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
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.counter", "2"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.links.#", "1"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.results.#", "2"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.results.0.id", "2580"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.results.1.id", "2581"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "zones.0.total_pages", "1"),
				),
			},
		},
	})
}
