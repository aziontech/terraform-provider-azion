package provider

//func TestAccResourceDnsSec(t *testing.T) {
//	resourceName := "azion_dnssec.examples"
//	resource.ParallelTest(t, resource.TestCase{
//		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
//		CheckDestroy:             testAccDNSSecResourceDestroy,
//		Steps: []resource.TestStep{
//			{
//				Config: testAccDNSSecResourceConfig(),
//				Check: resource.ComposeTestCheckFunc(
//					resource.TestCheckResourceAttr(resourceName, "schema_version", "3"),
//					resource.TestCheckResourceAttr(resourceName, "id", "2595"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.is_enabled", "true"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.status", "ready"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.delegation_signer.digesttype.id", "2"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.delegation_signer.digesttype.slug", "SHA256"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.delegation_signer.algorithmtype.id", "13"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.delegation_signer.algorithmtype.slug", "ECDSAP256SHA256"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.delegation_signer.digest", "35dbd2f5cd43d191d6f7c61f9c8d79149254186761a188b667f5ca78d0a3cc27"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.delegation_signer.keytag", "32597"),
//				),
//			},
//			{
//				Config: testAccDNSSecResourceConfigUpdate(),
//				Check: resource.ComposeTestCheckFunc(
//					resource.TestCheckResourceAttr(resourceName, "schema_version", "3"),
//					resource.TestCheckResourceAttr(resourceName, "id", "2595"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.is_enabled", "false"),
//					resource.TestCheckResourceAttr(resourceName, "dns_sec.status", "ready"),
//				),
//			},
//			{
//				ResourceName:            resourceName,
//				ImportState:             true,
//				ImportStateVerify:       true,
//				ImportStateVerifyIgnore: []string{"last_updated", "schema_version"},
//			},
//		},
//	})
//}
//
//func testAccDNSSecResourceDestroy(s *terraform.State) error {
//	return nil
//}
//
//func testAccDNSSecResourceConfig() string {
//	return `
//provider "azion" {
//  api_token  = "token"
//}
//resource "azion_dnssec" "examples" {
//  id = "2595"
//  dns_sec = {
//      is_enabled = true
//    }
//}
//`
//}
//
//func testAccDNSSecResourceConfigUpdate() string {
//	return `
//provider "azion" {
//  api_token  = "token"
//}
//resource "azion_dnssec" "examples" {
//  id = "2595"
//  dns_sec = {
//      is_enabled = false
//    }
//}
//`
//}
