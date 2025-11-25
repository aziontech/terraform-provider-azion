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
				Config: providerConfig + `data "azion_intelligent_dns_zones" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "counter", "6"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.#", "6"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.0.id", "2580"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.1.id", "2581"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.2.id", "2583"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.3.id", "2595"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.4.id", "2638"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "results.5.id", "2643"),
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zones.test", "id", "Get All Zones"),
				),
			},
		},
	})
}
