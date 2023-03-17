package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRecords(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
				resource "azion_records" "dev" {
					zone_id = 2553
					record = {
					  record_type= "A"
					  entry = "www"
					  answers_list = [
						  "1.1.1.1",
						  "8.8.8.8"
					  ]
					  policy = "simple"
					  ttl = 20
					}
				  }
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_records.test", "counter", "2"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.domain", "azionterraform.com"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.answers_list.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.answers_list.1", "8.8.8.8"),
					// resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.description", ""),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.entry", "www"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.policy", "simple"),
					// resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.record_id", "31755"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.record_type", "A"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.ttl", "3600"),

					resource.TestCheckResourceAttr("data.azion_records.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_records.test", "total_pages", "1"),
					resource.TestCheckResourceAttr("data.azion_records.test", "zone_id", "2553"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "azion_records.dev",
				ImportState:       true,
				ImportStateVerify: true,
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
				resource "azion_records" "dev" {
					zone_id = 2553
					record = {
					  record_type= "A"
					  entry = "ww2"
					  answers_list = [
						"8.8.8.8",
						"7.7.7.7"
					  ]
					  policy = "simple"
					  ttl = 20
					}
				  }
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_records.test", "counter", "2"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.domain", "azionterraform.com"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.answers_list.0", "8.8.8.8"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.answers_list.1", "7.7.7.7"),
					// resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.description", ""),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.entry", "ww2"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.policy", "simple"),
					// resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.record_id", "31755"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.record_type", "A"),
					resource.TestCheckResourceAttr("data.azion_records.test", "results.records.0.ttl", "3600"),

					resource.TestCheckResourceAttr("data.azion_records.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_records.test", "total_pages", "1"),
					resource.TestCheckResourceAttr("data.azion_records.test", "zone_id", "2595"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
