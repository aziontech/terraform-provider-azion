package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDomainsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "azion_domains" "test" { }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.domain_name", "yzi1x9djtz.map.azionedge.net"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.cname_access_only", "false"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.cnames.0", "www.terraformexample.com"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.edge_application_id", "1681826892"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.id", "1682377117"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.is_active", "true"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "results.0.name", "Terraform-domain-example"),

					resource.TestCheckResourceAttr("data.azion_domains.test", "schema_version", "3"),
					resource.TestCheckResourceAttr("data.azion_domains.test", "total_pages", "1"),
				),
			},
		},
	})
}
