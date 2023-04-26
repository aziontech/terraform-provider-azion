package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceDomain(t *testing.T) {
	// * * * * This test will be implemented
	// * * * * in the next project phase

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
					resource "azion_domain" "dev" {
						domain = {
						cnames: [
					"www.terraformexample3.com",
					"www.terraformexample4.com",
					]
					name = "Terraform-domain-example3"
					digital_certificate_id = null
					cname_access_only = false
					edge_application_id = 1681826892
					is_active = true
					}
					}
	
	`,

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("azion_domain.dev", "schema_version", "3"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "azion_domain.dev",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
			resource "azion_domain" "dev" {
			domain = {
			cnames: [
			"www.terraformexample5.com",
			"www.terraformexample6.com",
			]
			name = "Terraform-domain-example3"
			digital_certificate_id = null
			cname_access_only = false
			edge_application_id = 1681826892
			is_active = true
			}
			}
	
	`,

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("azion_domain", "schema_version", "3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
