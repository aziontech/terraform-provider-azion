package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDNSSecDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccDNSSecDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "zone_id", "2580"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "dnssec.is_enabled", "true"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "dnssec.status", "ready"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "delegation_signer.digest_type.id", "2"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "delegation_signer.digest_type.slug", "SHA256"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "delegation_signer.algorithm_type.id", "13"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "delegation_signer.algorithm_type.slug", "ECDSAP256SHA256"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "delegation_signer.digest", "3b7d6073c98645707d84e497a9263590c1ab00c494c3980305076b1add5fe781"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_dnssec.test", "delegation_signer.key_tag", "42528"),
				),
			},
		},
	})
}

func testAccDNSSecDataSourceConfig() string {
	return `
data "azion_intelligent_dns_dnssec" "test" { zone_id = 2580 }
`
}
