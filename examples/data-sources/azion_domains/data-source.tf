provider "azion" {
  api_token  = "<token>"
}

data "azion_domains" "dev" {
}

output "dev_domains" {
  value = data.azion_domains.dev
}