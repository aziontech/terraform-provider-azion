terraform {
  required_providers {
    azion = {
      source = "aziontech/azion"
      version = "0.2.0"
    }
  }
}
provider "azion" {
  api_token  = "azion9252873bb250dfac51625125cfd702e57af"
}

resource "azion_zone" "dev" {
  zone = {
      domain: "test6.com",
      is_active: true,
      name: "test6 demonstracao1 terraform"
    }
}
resource "azion_zone" "dev2" {
  zone = {
    domain: "test7.com",
    is_active: true,
    name: "test7 demonstracao1 terraform"
  }
}
resource "azion_zone" "dev3" {
  zone = {
    domain: "test8.com",
    is_active: true,
    name: "test8 demonstracao1 terraform"
  }
}

output "dev_order" {
  value = azion_zone.dev
}







#
#locals {
#  domains = ["terraform1.azion", "terraform2.azion", "terraform3.azion"]
##  names = ["test1 show terraform", "test2 show terraform", "test3 show terraform"]
#}
#resource "azion_zone" "CreateDomains" {
#  for_each = toset(local.domains)
#    zone = {
#      domain: each.value,
#      is_active: true,
#      name: "test show terraform"
#    }
#}
#output "CreateDomains" {
#  value = azion_zone.CreateDomains
#}

#resource "azion_zone" "NameChange" {
#  for_each = toset(local.names)
#  zone = {
#    domain: each.value,
#    is_active: true,
#    name: "test show terraform"
#  }
#}
#output "NameChanges" {
#  value = azion_zone.NameChange
#}

