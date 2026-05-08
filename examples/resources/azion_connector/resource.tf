# =====================================================
# CONNECTOR RESOURCES
# =====================================================

# Storage Connector Example
# Used to connect to object storage services
# NOTE: The bucket name must be a valid, existing bucket
resource "azion_connector" "storage_connector" {
  connector = {
    name   = "My Storage Connector"
    type   = "storage"
    active = true
    storage_attributes = {
      # Replace with a valid bucket name
      bucket = "my-bucket"
      prefix = "path/to/files/"
    }
  }
}

# HTTP Connector Example
# Used to connect to HTTP origins
resource "azion_connector" "http_connector" {
  connector = {
    name   = "My HTTP Connector"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          address   = "192.168.1.100"
          http_port = 80
          active    = true
        }
      ]
    }
  }
}

# HTTP Connector with all options
resource "azion_connector" "http_connector_full" {
  connector = {
    name   = "My HTTP Connector Full"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          address    = "192.168.1.100"
          http_port  = 80
          https_port = 443
          active     = true
          modules = {
            load_balancer = {
              server_role = "primary"
              weight      = 1
            }
          }
        }
      ]
      connection_options = {
        dns_resolution      = "both"
        following_redirect  = false
        host                = "$${host}"
        http_version_policy = "http1_1"
        path_prefix         = ""
        real_ip_header      = "X-Real-IP"
        real_port_header    = "X-Real-PORT"
        transport_policy    = "preserve"
      }
      modules = {
        load_balancer = {
          enabled = true
          config = {
            method             = "round_robin"
            max_retries        = 3
            connection_timeout = 60
            read_write_timeout = 120
          }
        }
        origin_shield = {
          enabled = false
        }
      }
    }
  }
}

# HTTP Connector with HMAC Origin Shield (S3-compatible)
# Used to connect to S3-compatible origins with HMAC authentication
resource "azion_connector" "s3_with_hmac" {
  connector = {
    name   = "S3 Connector with HMAC"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          address    = "my-bucket.s3.us-east-1.amazonaws.com"
          http_port  = 80
          https_port = 443
          active     = true
        }
      ]
      connection_options = {
        host             = "my-bucket.s3.amazonaws.com"
        transport_policy = "force_https"
      }
      modules = {
        origin_shield = {
          enabled = true
          config = {
            hmac = {
              enabled = true
              config = {
                type = "aws4_hmac_sha256"
                attributes = {
                  region     = "us-east-1"
                  service    = "s3"
                  access_key = "YOUR_ACCESS_KEY"
                  secret_key = "YOUR_SECRET_KEY"
                }
              }
            }
          }
        }
      }
    }
  }
}

# =====================================================
# DATA SOURCES
# =====================================================

# Read a single connector by ID
data "azion_connector" "by_id" {
  id = azion_connector.http_connector.connector.id
}

# List all connectors in the account
data "azion_connectors" "all" {}

# =====================================================
# OUTPUTS
# =====================================================

output "storage_connector_id" {
  description = "ID of the storage connector"
  value       = azion_connector.storage_connector.connector.id
}

output "storage_connector_name" {
  description = "Name of the storage connector"
  value       = azion_connector.storage_connector.connector.name
}

output "storage_connector_type" {
  description = "Type of the storage connector"
  value       = azion_connector.storage_connector.connector.type
}

output "storage_connector_attributes" {
  description = "Attributes of the storage connector"
  value       = azion_connector.storage_connector.connector.storage_attributes
}

output "http_connector_id" {
  description = "ID of the HTTP connector"
  value       = azion_connector.http_connector.connector.id
}

output "http_connector_name" {
  description = "Name of the HTTP connector"
  value       = azion_connector.http_connector.connector.name
}

output "http_connector_attributes" {
  description = "Attributes of the HTTP connector (includes API defaults)"
  value       = azion_connector.http_connector.connector.http_attributes
}

output "connector_by_id_data" {
  description = "Data from the connector read by ID"
  value       = data.azion_connector.by_id.data
}

output "all_connectors_count" {
  description = "Total count of connectors"
  value       = data.azion_connectors.all.counter
}

output "all_connectors_names" {
  description = "Names of all connectors"
  value       = [for c in data.azion_connectors.all.results : c.name]
}
