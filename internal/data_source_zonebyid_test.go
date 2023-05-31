package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccZonebyIdDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "azion_zone" "test" { id = 2580 }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_zone.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.zone_id", "2580"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.domain", "test6.com"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.nameservers.0", "ns1.aziondns.net"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.nameservers.1", "ns2.aziondns.com"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.nameservers.2", "ns3.aziondns.org"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.retry", "7200"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.name", "test6 demonstracao1 terraform"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.soattl", "3600"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.is_active", "true"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.refresh", "43200"),
					resource.TestCheckResourceAttr("data.azion_zone.test", "results.expiry", "1209600"),
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.azion_zone.test", "id", "Get By ID Zone"),
				),
			},
		},
	})
}
