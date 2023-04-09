resource "azion_zone" "example" {
  zone = {
      domain: "example.com",
      is_active: true,
      name: "example"
    }
}
