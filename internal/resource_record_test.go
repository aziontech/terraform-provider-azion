package provider

import (
	"testing"
)

func TestAccResourceRecord(t *testing.T) {
	// * * * * This test will be implemented
	// * * * * in the next project phase

	//	resource.Test(t, resource.TestCase{
	//		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
	//		Steps: []resource.TestStep{
	//			// Create and Read testing
	//			{
	//				Config: providerConfig + `
	//				resource "azion_records" "dev" {
	//					zone_id = 2638
	//					record = {
	//					  record_type= "A"
	//					  entry = "www"
	//					  answers_list = [
	//						  "1.1.1.1",
	//						  "8.8.8.8"
	//					  ]
	//					  policy = "simple"
	//					  ttl = 20
	//					}
	//				  }
	//
	// `,
	//
	//		Check: resource.ComposeAggregateTestCheckFunc(
	//			resource.TestCheckResourceAttr("azion_records.dev", "schema_version", "3"),
	//		),
	//	},
	//	// ImportState testing
	//	{
	//		ResourceName:            "azion_records.dev",
	//		ImportState:             true,
	//		ImportStateVerify:       true,
	//		ImportStateVerifyIgnore: []string{"last_updated"},
	//	},
	//	// Update and Read testing
	//	{
	//		Config: providerConfig + `
	//		resource "azion_records" "dev" {
	//			zone_id = 2638
	//			record = {
	//			  record_type= "A"
	//			  entry = "ww2"
	//			  answers_list = [
	//				"8.8.8.8",
	//				"7.7.7.7"
	//			  ]
	//			  policy = "simple"
	//			  ttl = 20
	//			}
	//		  }
	//
	// `,
	//
	//				Check: resource.ComposeAggregateTestCheckFunc(
	//					resource.TestCheckResourceAttr("azion_records.dev", "schema_version", "3"),
	//				),
	//			},
	//			// Delete testing automatically occurs in TestCase
	//		},
	//	})
}
