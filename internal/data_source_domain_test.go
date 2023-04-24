package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDomainDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "azion_domains" "test" { id = 2580 }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.domain_name", "t6sd3m27lf.map.azionedge.net"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.cname_access_only", "true"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.cnames.0", "www.terraformdomaintest.com.br"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.edge_application_id", "1681826892"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.id", "1681825953"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.is_active", "true"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.name", "Terraform-domain-test"),

					resource.TestCheckResourceAttr("data.azion_domains.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "total_pages", "1"),
				),
			},
		},
	})
}
