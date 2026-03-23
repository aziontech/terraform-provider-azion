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
