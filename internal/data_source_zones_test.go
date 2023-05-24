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
					resource.TestCheckResourceAttr("data.azion_zones.test", "counter", "6"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "links.#", "0"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.#", "6"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.0.zone_id", "2580"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.1.zone_id", "2581"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.2.zone_id", "2583"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.3.zone_id", "2595"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.4.zone_id", "2638"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "results.5.zone_id", "2643"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_zones.test", "total_pages", "1"),
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.azion_zones.test", "id", "Get All Zones"),
				),
			},
		},
	})
}
