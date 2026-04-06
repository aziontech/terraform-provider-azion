---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_workload_deployment"
description: |-
  Provides a resource to manage workload deployments.
---

# azion_workload_deployment (Resource)

Provides a resource to manage workload deployments within Azion workloads.

## Example Usage

```terraform
resource "azion_workload_deployment" "example" {
  workload_id = 12345

  deployment = {
    name    = "My Deployment"
    current = true
    active  = true

    strategy = {
      type = "default"
      attributes = {
        application = 67890
      }
    }
  }
}
```

## Import

```sh
terraform import azion_workload_deployment.example 12345/67890
```

The import format is: `workloadID/deploymentID`

## Argument Reference

* `workload_id` - (Required) The ID of the workload to which the deployment belongs.
* `deployment` - (Required) The deployment configuration block.
  * `name` - (Required) Name of the deployment.
  * `current` - (Optional) Whether this is the current deployment. Defaults to `false`.
  * `active` - (Optional) Status of the deployment. Defaults to `false`.
  * `strategy` - (Required) Deployment strategy configuration.
    * `type` - (Required) Type of deployment strategy.
    * `attributes` - (Required) Strategy attributes.
      * `application` - (Required) Application ID for the deployment.
      * `firewall` - (Optional) Firewall ID for the deployment.
      * `custom_page` - (Optional) Custom page ID for the deployment.

## Attribute Reference

The following attributes are exported:

* `id` - The identifier of the resource in the format `workloadID/deploymentID`.
* `last_updated` - Timestamp of the last Terraform update of the resource.
* `deployment` - The deployment configuration.
  * `id` - The deployment identifier (computed).
  * `last_editor` - The last editor of the deployment.
  * `last_modified` - Last modified timestamp of the deployment.
  * `created_at` - Creation timestamp of the deployment.
