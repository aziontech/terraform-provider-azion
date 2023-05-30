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
				Config: providerConfig + `data "azion_domain" "test" { id = 1682377117 }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.domain_name", "yzi1x9djtz.map.azionedge.net"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.cname_access_only", "false"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.cnames.0", "www.terraformexample.com"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.edge_application_id", "1681826892"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.domain_id", "1682377117"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.is_active", "true"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "results.name", "Terraform-domain-example"),
					resource.TestCheckResourceAttr("data.azion_domain.test", "schema_version", "3"),
				),
			},
		},
	})
}
