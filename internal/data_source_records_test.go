package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRecordsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "azion_intelligent_dns_records" "test" { zone_id = 2638 }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.record_id", "32538"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.rdata.0", "8.8.8.8"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.description", "This is a description"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.name", "site"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.policy", "weighted"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.type", "A"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.0.ttl", "20"),

					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.record_id", "33364"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.rdata.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.rdata.1", "8.8.8.8"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.name", "www"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.policy", "simple"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.type", "A"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "results.1.ttl", "3600"),

					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "total_pages", "1"),
					resource.TestCheckResourceAttr("data.azion_intelligent_dns_records.test", "zone_id", "2638"),
				),
			},
		},
	})
}
