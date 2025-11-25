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
				Config: providerConfig + `data "azion_intelligent_dns_zone" "test" { id = 2580 }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.id", "2580"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.domain", "test6.com"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.nameservers.0", "ns1.aziondns.net"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.nameservers.1", "ns2.aziondns.com"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.nameservers.2", "ns3.aziondns.org"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.name", "test6 demonstracao1 terraform"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "data.active", "true"),
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_zone.test", "id", "Get Zone by ID"),
				),
			},
		},
	})
}
