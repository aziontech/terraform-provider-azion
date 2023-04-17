terraform {
  required_providers {
    azion = {
      source = "github.com/actions/azion"
#      version = ">= 0.13"
    }
  }
  required_version = ">= 0.13"
}
provider "azion" {
  api_token  = "azion9252873bb250dfac51625125cfd702e57af"
}
#####################################################
#####################-AZION ZONE-####################
#####################################################

#
#data "azion_zones" "dev" {
#}
#
#output "dev_zones" {
#  value = data.azion_zones.dev
#}

#####
#
#data "azion_zone" "dev2" {
#  id = "2580"
#}
#
#output "dev_zone" {
#  value = data.azion_zone.dev2
#}

#####

#resource "azion_zone" "dev3" {
#  zone = {
#    domain: "testtalk1.com",
#    is_active: true,
#    name: "test for tech talk terraform"
#  }
#}
#
#output "dev_Azion" {
#  value = azion_zone.dev3
#}

#####

#resource "azion_zone" "dev2" {
#  zone = {
#    domain: "test7.com",
#    is_active: true,
#    name: "test7 demonstracao1 terraform"
#  }
#}
#resource "azion_zone" "dev3" {
#  zone = {
#    domain: "test8.com",
#    is_active: true,
#    name: "test8 demonstracao1 terraform"
#  }
#}


#####################################################
#####################-AZION RECORDS-#################
#####################################################

#resource "azion_record" "dev" {
#  zone_id = "2658"
#  record = {
#    record_type= "A"
#    entry = "site"
#    answers_list = [
#      "8.8.8.8"
#    ]
#    # policy = "simple"
#    policy = "weighted"
#    weight = 50
#    description = "This is a description"
#    ttl = 20
#  }
#}
#output "dev_records" {
#  value = azion_record.dev
#}

#####################################################
#####################-AZION DNSSEC-##################
#####################################################

resource "azion_dnssec" "dev" {
  zone_id = "2580"
  dns_sec = {
    is_enabled = false
  }
}

output "dev_Azion" {
  value = azion_dnssec.dev
}














#
#locals {
#  domains = ["terraform11.azion", "terraform21.azion", "terraform31.azion"]
#  names = ["test1 show terraform", "test2 show terraform", "test3 show terraform"]
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


#### Azion RECORDS

