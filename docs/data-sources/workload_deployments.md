---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_workload_deployments"
description: |-
  Provides a data source to list all workload deployments.
---

# azion_workload_deployments (Data Source)

Use this data source to list all deployments for a specific workload.

## Example Usage

```terraform
data "azion_workload_deployments" "example" {
  workload_id = "12345"
}
```

## Argument Reference

* `workload_id` - (Required) The numeric identifier of the workload as a string.

## Attribute Reference

* `id` - Identifier of the data source.
* `deployments_count` - The total number of deployments.
* `results` - List of deployments.
  * `id` - The deployment identifier.
  * `name` - Name of the deployment.
  * `current` - Whether this is the current deployment.
  * `active` - Status of the deployment.
  * `strategy` - Deployment strategy configuration.
    * `type` - Type of deployment strategy.
    * `attributes` - Strategy attributes.
      * `application` - Application ID for the deployment.
      * `firewall` - Firewall ID for the deployment.
      * `custom_page` - Custom page ID for the deployment.
  * `last_editor` - The last editor of the deployment.
  * `last_modified` - Last modified timestamp of the deployment.
  * `created_at` - Creation timestamp of the deployment.
