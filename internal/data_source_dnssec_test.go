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
				Config: providerConfig + `data "azion_dnssec" "test" { zone_id = "2580" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "zone_id", "2580"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.is_enabled", "true"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.status", "ready"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.delegation_signer.digesttype.id", "2"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.delegation_signer.digesttype.slug", "SHA256"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.delegation_signer.algorithmtype.id", "13"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.delegation_signer.algorithmtype.slug", "ECDSAP256SHA256"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.delegation_signer.digest", "3b7d6073c98645707d84e497a9263590c1ab00c494c3980305076b1add5fe781"),
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "dns_sec.delegation_signer.keytag", "42528"),
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.azion_dnssec.test", "id", "Get DNSSEC"),
				),
			},
		},
	})
}
