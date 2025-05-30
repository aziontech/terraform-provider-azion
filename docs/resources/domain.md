---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azion_domain Resource - terraform-provider-azion"
subcategory: ""
description: |-
  
---

# azion_domain (Resource)



## Example Usage

```terraform
resource "azion_domain" "example" {
  domain = {
    cnames : [
      "www.example.com",
      "www.example2.com"
    ]
    name                   = "Terraform-domain-example"
    digital_certificate_id = null
    cname_access_only      = false
    edge_application_id    = 1234567890
    is_active              = true
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `domain` (Attributes) (see [below for nested schema](#nestedatt--domain))

### Read-Only

- `id` (String) The ID of this resource.
- `last_updated` (String) Timestamp of the last Terraform update of the resource.
- `schema_version` (Number)

<a id="nestedatt--domain"></a>
### Nested Schema for `domain`

Required:

- `cname_access_only` (Boolean) Allow access to your URL only via provided CNAMEs.
- `cnames` (Set of String) List of domains to use as URLs for your files.
- `edge_application_id` (Number) Edge Application associated ID.
- `is_active` (Boolean) Make access to your URL only via provided CNAMEs.
- `name` (String) Name of this entry.

Optional:

- `digital_certificate_id` (Number) Digital Certificate associated ID.
- `environment` (String) Accepted values: production | preview

Read-Only:

- `domain_name` (String) Domain name attributed by Azion to this configuration.
- `id` (Number) Identification of this entry.

## Import

Import is supported using the following syntax:

```shell
terraform import azion_domain.example <domain_id>
```
