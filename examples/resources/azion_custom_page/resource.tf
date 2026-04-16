resource "azion_custom_page" "example" {
  custom_page = {
    name   = "My Custom Error Pages"
    active = true
    pages = [
      {
        code = "404"
        page = {
          type = "application"
          attributes = {
            connector = 12345
            ttl       = 60
            uri       = "/errors/404.html"
          }
        }
      },
      {
        code = "500"
        page = {
          type = "application"
          attributes = {
            connector = 12345
            ttl       = 60
            uri       = "/errors/500.html"
          }
        }
      }
    ]
  }
}
