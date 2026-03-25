# =====================================================
# CONNECTOR DATA SOURCES
# =====================================================

# Read a single connector by ID
# The ID can be from an existing connector or from a resource reference
data "azion_connector" "by_id" {
  id = "12345"
}

# Example using a resource reference:
# data "azion_connector" "by_id" {
#   id = azion_connector.example.connector.id
# }

# =====================================================
# OUTPUTS
# =====================================================

output "connector_data" {
  description = "Data from the connector read by ID"
  value       = data.azion_connector.by_id.data
}

output "connector_name" {
  description = "Name of the connector"
  value       = data.azion_connector.by_id.data.name
}

output "connector_type" {
  description = "Type of the connector"
  value       = data.azion_connector.by_id.data.type
}

output "connector_active" {
  description = "Status of the connector"
  value       = data.azion_connector.by_id.data.active
}

output "connector_attributes" {
  description = "Attributes of the connector as JSON string"
  value       = data.azion_connector.by_id.data.attributes
}
