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
				Config: providerConfig + `data "azion_records" "test" { zone_id = 2638 }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_records.test", "results.domain", "testrerecords.com"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.answers_list.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.answers_list.1", "8.8.8.8"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.description", ""),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.entry", "www"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.policy", "simple"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.record_type", "A"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.ttl", "3600"),

					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.1.answers_list.0", "www.azionterraform.com"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.1.description", ""),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.1.entry", "w3"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.1.policy", "simple"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.1.record_type", "CNAME"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.1.ttl", "3600"),

					resource.TestCheckResourceAttr("data.azion_records.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_records.test", "total_pages", "1"),
					resource.TestCheckResourceAttr("data.azion_records.test", "zone_id", "2638"),
				),
			},
		},
	})
}
