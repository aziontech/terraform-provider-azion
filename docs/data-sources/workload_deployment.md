---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_workload_deployment"
description: |-
  Provides a data source to read a specific workload deployment.
---

# azion_workload_deployment (Data Source)

Use this data source to read a specific workload deployment by its workload ID and deployment ID.

## Example Usage

```terraform
data "azion_workload_deployment" "example" {
  workload_id   = "12345"
  deployment_id = "67890"
}
```

## Argument Reference

* `workload_id` - (Required) The ID of the workload to which the deployment belongs.
* `deployment_id` - (Required) The ID of the deployment to retrieve.

## Attribute Reference

* `id` - Identifier of the data source.
* `data` - The deployment data.
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
