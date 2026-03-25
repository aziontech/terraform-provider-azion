# =====================================================
# CONNECTORS LIST DATA SOURCE
# =====================================================

# List all connectors in the account
data "azion_connectors" "all" {}

# =====================================================
# OUTPUTS
# =====================================================

output "all_connectors_count" {
  description = "Total count of connectors"
  value       = data.azion_connectors.all.counter
}

output "all_connectors_names" {
  description = "Names of all connectors"
  value       = [for c in data.azion_connectors.all.results : c.name]
}

output "all_connectors_types" {
  description = "Types of all connectors"
  value       = [for c in data.azion_connectors.all.results : c.type]
}

output "all_connectors_ids" {
  description = "IDs of all connectors"
  value       = [for c in data.azion_connectors.all.results : c.id]
}
