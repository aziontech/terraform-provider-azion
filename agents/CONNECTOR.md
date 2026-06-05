# Connectors - Code Generation Guide

This document provides specific guidance for implementing Connectors resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Resource Implementation](#resource-implementation)
3. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
4. [Connector Types and Polymorphism](#connector-types-and-polymorphism)
5. [Schema Definition Patterns](#schema-definition-patterns)
6. [Error Handling](#error-handling)
7. [Type Conversions](#type-conversions)
8. [Common Issues](#common-issues)

---

## SDK Selection

Connectors use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Connector (Singular Data Source) | `azion-api` (v4) | `api.ConnectorsAPI` | `https://api.azion.com/v4` |
| Connectors (Plural Data Source) | `azion-api` (v4) | `api.ConnectorsAPI` | `https://api.azion.com/v4` |
| Connector (Resource) | `azion-api` (v4) | `api.ConnectorsAPI` | `https://api.azion.com/v4` |

### Important: SDK Import Path

**The V4 SDK import path is:**

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### Important: Naming Convention

**The "edge" prefix is NOT used for connectors.**

When implementing connector resources:
- Use naming without the `edge` prefix for variables, structs, and function parameters
- The Terraform resource names use `connector` (not `edge_connector`)
- Internal Go code naming follows this convention

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Response Type | `ConnectorResponse` with `Data` field |
| Data Type | `Connector` (polymorphic - see below) |
| List Response Type | `PaginatedConnectorList` |
| Retrieve Method | `.RetrieveConnector(ctx, connectorId).Execute()` |
| List Method | `.ListConnectors(ctx).Execute()` |
| Create Method | `.CreateConnector(ctx).ConnectorRequest(req).Execute()` |
| Update Method | `.PartialUpdateConnector(ctx, connectorId).PatchedConnectorRequest(req).Execute()` |
| Delete Method | `.DeleteConnector(ctx, connectorId).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azion-api) - used for connectors
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    // ... other SDK clients
}
```

---

## Resource Implementation

### File Structure

**File:** `internal/resource_connector.go`

### Resource Model

The connector resource uses nested attributes for type-specific configuration:

```go
type connectorResourceModel struct {
    Connector     *connectorResourceResults `tfsdk:"connector"`
    ID            types.String              `tfsdk:"id"`
    LastUpdated   types.String              `tfsdk:"last_updated"`
    SchemaVersion types.Int64               `tfsdk:"schema_version"`
}

type connectorResourceResults struct {
    ID             types.Int64             `tfsdk:"id"`
    Name           types.String            `tfsdk:"name"`
    LastEditor     types.String            `tfsdk:"last_editor"`
    LastModified   types.String            `tfsdk:"last_modified"`
    CreatedAt      types.String            `tfsdk:"created_at"`
    ProductVersion types.String            `tfsdk:"product_version"`
    Active         types.Bool              `tfsdk:"active"`
    Type           types.String            `tfsdk:"type"`
    IsVersioned    types.Bool              `tfsdk:"is_versioned"`
    Version        types.Int64             `tfsdk:"version"`
    VersionState   types.String            `tfsdk:"version_state"`
    VersionID      types.String            `tfsdk:"version_id"`
    StorageAttrs   *StorageAttributesModel `tfsdk:"storage_attributes"`
    HTTPAttrs      *HTTPAttributesModel    `tfsdk:"http_attributes"`
}

// Storage connector attributes
type StorageAttributesModel struct {
    Bucket types.String `tfsdk:"bucket"`
    Prefix types.String `tfsdk:"prefix"`
}

// HTTP connector attributes
// Note: ConnectionOptions and Modules use types.Object because they are
// both Optional and Computed - the API returns default values
type HTTPAttributesModel struct {
    Addresses         []AddressModel     `tfsdk:"addresses"`
    ConnectionOptions types.Object       `tfsdk:"connection_options"`
    Modules           types.Object       `tfsdk:"modules"`
}
```

### Important: Handling Computed Nested Attributes

When nested attributes have both `Optional: true` and `Computed: true`, you **must** use `types.Object` instead of pointer types. This is because Terraform needs to handle unknown values during planning.

**Wrong (pointer types can't handle unknown values):**
```go
type HTTPAttributesModel struct {
    ConnectionOptions *HTTPConnectionOptionsModel `tfsdk:"connection_options"` // WRONG!
}
```

**Correct (use types.Object for Optional+Computed):**
```go
type HTTPAttributesModel struct {
    ConnectionOptions types.Object `tfsdk:"connection_options"` // Correct
}
```

### Schema Definition

```go
func (r *connectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "connector": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "name": schema.StringAttribute{Required: true},
                    "type": schema.StringAttribute{Required: true},
                    "is_versioned": schema.BoolAttribute{
                        Description: "Whether the connector is versioned.",
                        Computed:    true,
                    },
                    "version": schema.Int64Attribute{
                        Description: "The current version of the connector.",
                        Computed:    true,
                    },
                    "version_state": schema.StringAttribute{
                        Description: "The state of the current connector version.",
                        Computed:    true,
                    },
                    "version_id": schema.StringAttribute{
                        Description: "The identifier of the current connector version.",
                        Computed:    true,
                    },
                    // Storage attributes - nested inside connector
                    "storage_attributes": schema.SingleNestedAttribute{
                        Optional: true,
                        Attributes: map[string]schema.Attribute{
                            "bucket": schema.StringAttribute{Required: true},
                            "prefix": schema.StringAttribute{Optional: true},
                        },
                    },
                    // HTTP attributes - nested inside connector
                    "http_attributes": schema.SingleNestedAttribute{
                        Optional: true,
                        Attributes: map[string]schema.Attribute{
                            "addresses": schema.ListNestedAttribute{Required: true, /* ... */},
                            "connection_options": schema.SingleNestedAttribute{
                                Optional: true,
                                Computed: true, // API provides defaults
                                Attributes: map[string]schema.Attribute{ /* ... */ },
                            },
                            "modules": schema.SingleNestedAttribute{
                                Optional: true,
                                Computed: true, // API provides defaults
                                Attributes: map[string]schema.Attribute{ /* ... */ },
                            },
                        },
                    },
                },
            },
        },
    }
}
```

### Create Method

The Create method handles polymorphic connector types with proper error handling:

```go
import (
    "github.com/aziontech/terraform-provider-azion/internal/utils"
)

func (r *connectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan connectorResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    connectorType := plan.Connector.Type.ValueString()
    var connectorId int64

    switch connectorType {
    case "storage":
        connectorReq, err := buildStorageConnectorRequest(plan.Connector)
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "Failed to build storage connector request")
            return
        }
        createConnector, response, err := r.client.api.ConnectorsAPI.CreateConnector(ctx).ConnectorRequest(connectorReq).Execute()
        if response != nil {
            defer response.Body.Close()  // Close body BEFORE error check
        }
        if err != nil {
            if response != nil && response.StatusCode == http.StatusTooManyRequests {
                // Retry on 429 rate limiting
                createConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
                    return r.client.api.ConnectorsAPI.CreateConnector(ctx).ConnectorRequest(connectorReq).Execute()
                }, 5)
                if response != nil {
                    defer response.Body.Close()
                }
                if err != nil {
                    resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                    return
                }
            } else {
                addConnectorAPIError(&resp.Diagnostics, err, response, "create")
                return
            }
        }
        connectorId = getConnectorId(createConnector.GetData())

    case "http":
        connectorReq, err := r.buildHTTPConnectorRequest(ctx, plan.Connector)
        // ... similar pattern with error handling
    }
    
    // Read back to get API defaults
    getConnector, response, err := r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute()
    if response != nil {
        defer response.Body.Close()  // Close body BEFORE error check
    }
    if err != nil {
        // Handle 429 and other errors similarly
    }
    
    // Populate response and set state
    r.populateConnectorFromResponse(ctx, plan.Connector, getConnector.GetData())
    plan.ID = types.StringValue(strconv.FormatInt(plan.Connector.ID.ValueInt64(), 10))
    // ... set state
}
```

### Important: Response Body Closure Pattern

**CRITICAL**: Always close HTTP response bodies to prevent resource leaks. The `defer response.Body.Close()` must be placed BEFORE the error check:

```go
// WRONG - body not closed on error paths
result, response, err := client.API.Method(ctx).Execute()
if err != nil {
    return  // Body leaked!
}
if response != nil {
    defer response.Body.Close()
}

// CORRECT - body always closed
result, response, err := client.API.Method(ctx).Execute()
if response != nil {
    defer response.Body.Close()  // Placed BEFORE error check
}
if err != nil {
    // error handling
    return
}
```

### Important: Rate Limiting (429) Retry Pattern

Use `utils.RetryOn429` for automatic retries on rate limiting:

```go
if response != nil && response.StatusCode == http.StatusTooManyRequests {
    result, response, err = utils.RetryOn429(func() (*azionapi.SomeResponse, *http.Response, error) {
        return client.API.Method(ctx).Execute()
    }, 5)  // Max 5 retries
    if response != nil {
        defer response.Body.Close()
    }
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
        return
    }
}
```

For delete operations (which return different types), use `utils.RetryOn429Delete`:

```go
if response != nil && response.StatusCode == http.StatusTooManyRequests {
    _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
        return client.API.DeleteMethod(ctx, id).Execute()
    }, 5)
    if response != nil {
        defer response.Body.Close()
    }
    if err != nil {
        resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
        return
    }
}
```

### Helper Functions for Building Requests

```go
func buildStorageConnectorRequest(connector *connectorResourceResults) (azionapi.ConnectorRequest, error) {
    if connector.StorageAttrs == nil {
        return azionapi.ConnectorRequest{}, fmt.Errorf("storage_attributes is required")
    }

    attrs := azionapi.ConnectorStorageAttributesRequest{
        Bucket: connector.StorageAttrs.Bucket.ValueString(),
    }
    
    if !connector.StorageAttrs.Prefix.IsNull() {
        attrs.SetPrefix(connector.StorageAttrs.Prefix.ValueString())
    }

    req := azionapi.NewConnectorStorageRequest(
        connector.Name.ValueString(),
        connector.Type.ValueString(),
        attrs,
    )

    return azionapi.ConnectorStorageRequestAsConnectorRequest(req), nil
}

func (r *connectorResource) buildHTTPConnectorRequest(ctx context.Context, connector *connectorResourceResults) (azionapi.ConnectorRequest, error) {
    if connector.HTTPAttrs == nil {
        return azionapi.ConnectorRequest{}, fmt.Errorf("http_attributes is required")
    }

    // Build addresses
    var addresses []azionapi.AddressRequest
    for _, addr := range connector.HTTPAttrs.Addresses {
        address := azionapi.NewAddressRequest(addr.Address.ValueString())
        // ... set optional fields
        addresses = append(addresses, *address)
    }

    attrs := azionapi.NewConnectorHTTPAttributesRequest(addresses)

    // Handle connection_options (types.Object)
    if !connector.HTTPAttrs.ConnectionOptions.IsNull() {
        var connOptsModel HTTPConnectionOptionsModel
        diags := connector.HTTPAttrs.ConnectionOptions.As(ctx, &connOptsModel, basetypes.ObjectAsOptions{})
        if diags.HasError() {
            return azionapi.ConnectorRequest{}, fmt.Errorf("failed to parse connection_options")
        }
        // ... build connection options request
    }

    // Handle modules (types.Object)
    if !connector.HTTPAttrs.Modules.IsNull() {
        var modulesModel HTTPModulesModel
        diags := connector.HTTPAttrs.Modules.As(ctx, &modulesModel, basetypes.ObjectAsOptions{})
        // ... build modules request
    }

    req := azionapi.NewConnectorHTTPRequest(
        connector.Name.ValueString(),
        connector.Type.ValueString(),
        *attrs,
    )

    return azionapi.ConnectorHTTPRequestAsConnectorRequest(req), nil
}
```

### Populate Response Helper

```go
func (r *connectorResource) populateConnectorFromResponse(ctx context.Context, model *connectorResourceResults, connector azionapi.Connector) {
    actualConnector := connector.GetActualInstance()
    
    switch c := actualConnector.(type) {
    case *azionapi.ConnectorStorage:
        // Storage connector
        model.ID = types.Int64Value(c.Id)
        model.Name = types.StringValue(c.Name)
        model.Type = types.StringValue(c.Type)
        model.Active = types.BoolPointerValue(c.Active)
        model.IsVersioned = types.BoolValue(c.IsVersioned)
        model.Version = types.Int64PointerValue(c.Version.Get())
        model.VersionState = types.StringPointerValue(c.VersionState.Get())
        model.VersionID = types.StringPointerValue(c.VersionId.Get())
        
        model.StorageAttrs = &StorageAttributesModel{
            Bucket: types.StringValue(c.Attributes.Bucket),
        }
        if c.Attributes.Prefix != nil {
            model.StorageAttrs.Prefix = types.StringValue(*c.Attributes.Prefix)
        }
        model.HTTPAttrs = nil

    case *azionapi.ConnectorHTTP:
        // HTTP connector
        model.ID = types.Int64Value(c.Id)
        model.Name = types.StringValue(c.Name)
        model.Type = types.StringValue(c.Type)
        model.Active = types.BoolPointerValue(c.Active)
        model.IsVersioned = types.BoolValue(c.IsVersioned)
        model.Version = types.Int64PointerValue(c.Version.Get())
        model.VersionState = types.StringPointerValue(c.VersionState.Get())
        model.VersionID = types.StringPointerValue(c.VersionId.Get())
        
        httpAttrs := &HTTPAttributesModel{}
        
        // Populate addresses
        for _, addr := range c.Attributes.Addresses {
            // ... build address model
        }
        
        // Populate connection_options
        if c.Attributes.ConnectionOptions != nil {
            // ... build connection options model
        }
        
        // Populate modules
        if c.Attributes.Modules != nil {
            // ... build modules model
        }
        
        model.HTTPAttrs = httpAttrs
        model.StorageAttrs = nil
    }
}

// attrTypes methods are required for types.Object conversion
func (m HTTPConnectionOptionsModel) attrTypes() map[string]attr.Type {
    return map[string]attr.Type{
        "dns_resolution":      types.StringType,
        "following_redirect":  types.BoolType,
        "host":                types.StringType,
        "http_version_policy": types.StringType,
        "path_prefix":         types.StringType,
        "real_ip_header":      types.StringType,
        "real_port_header":    types.StringType,
        "transport_policy":    types.StringType,
    }
}
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Connector by its identifier:

**File:** `internal/data_source_connector.go`

```go
type ConnectorDataSourceModel struct {
    Data ConnectorResults `tfsdk:"data"`
    ID   types.String     `tfsdk:"id"`
}

type ConnectorResults struct {
    ID             types.Int64  `tfsdk:"id"`
    Name           types.String `tfsdk:"name"`
    LastEditor     types.String `tfsdk:"last_editor"`
    LastModified   types.String `tfsdk:"last_modified"`
    CreatedAt      types.String `tfsdk:"created_at"`
    ProductVersion types.String `tfsdk:"product_version"`
    Active         types.Bool   `tfsdk:"active"`
    Type           types.String `tfsdk:"type"`
    IsVersioned    types.Bool   `tfsdk:"is_versioned"`
    Version        types.Int64  `tfsdk:"version"`
    VersionState   types.String `tfsdk:"version_state"`
    VersionID      types.String `tfsdk:"version_id"`
    Attributes     types.String `tfsdk:"attributes"` // JSON string
}
```

### Read Method for Singular Data Source

```go
func (d *ConnectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getConnectorId types.String
    diags := req.Config.GetAttribute(ctx, path.Root("id"), &getConnectorId)
    // ...
    
    connectorResponse, response, err := d.client.api.ConnectorsAPI.
        RetrieveConnector(ctx, connectorID).Execute()
    // ...
}
```

---

## Connector Types and Polymorphism

Connectors are polymorphic - they can be one of several types, each with different attribute structures.

### Available Connector Types

| Type | Description | Attributes Structure |
|------|-------------|---------------------|
| `storage` | Storage connector | `ConnectorStorageAttributes` |
| `http` | HTTP connector | `ConnectorHTTPAttributes` |

**Note:** `live_ingest` is no longer supported.

### SDK Types

```go
// Polymorphic wrapper - can contain any connector type
type Connector struct {
    ConnectorStorage *ConnectorStorage  // For storage type
    ConnectorHTTP    *ConnectorHTTP     // For HTTP type
}

// Storage connector
type ConnectorStorage struct {
    Id             int64
    Name           string
    LastEditor     string
    LastModified   time.Time
    Active         *bool
    ProductVersion string
    Type           string  // "storage"
    IsVersioned    bool
    Version        NullableInt64
    VersionState   NullableString
    VersionId      NullableString
    Attributes     ConnectorStorageAttributes
}

type ConnectorStorageAttributes struct {
    Bucket string
    Prefix *string
}

// HTTP connector
type ConnectorHTTP struct {
    Id             int64
    Name           string
    LastEditor     string
    LastModified   time.Time
    Active         *bool
    ProductVersion string
    Type           string  // "http"
    IsVersioned    bool
    Version        NullableInt64
    VersionState   NullableString
    VersionId      NullableString
    Attributes     ConnectorHTTPAttributes
}

type ConnectorHTTPAttributes struct {
    Addresses         []Address
    ConnectionOptions *HTTPConnectionOptions
    Modules           *HTTPModules
}
```

---

## Schema Definition Patterns

### Terraform Configuration Example

```hcl
# =====================================================
# CONNECTOR RESOURCES
# =====================================================

# Storage Connector Example
# Used to connect to object storage services
# NOTE: The bucket name must be a valid, existing bucket
# You can either:
#   1. Use an existing bucket name directly (string value)
#   2. Reference a bucket created via azion_bucket resource
resource "azion_connector" "storage_connector" {
  connector = {
    name   = "tf-test-storage-connector"
    type   = "storage"
    active = true
    storage_attributes = {
      # Option 1: Use an existing bucket name directly
      # bucket = "my-existing-bucket"
      
      # Option 2: Reference a bucket resource (recommended)
      bucket = azion_bucket.example.bucket.name
      prefix = "path/to/files/"
    }
  }
  
  # When referencing a bucket resource, add depends_on to ensure
  # the bucket is created before the connector
  depends_on = [
    azion_bucket.example
  ]
}

# Example bucket resource (if using Option 2 above)
# resource "azion_bucket" "example" {
#   bucket = {
#     name             = "my-terraform-bucket"
#     workloads_access = "read_write"
#   }
# }

# HTTP Connector Example
# Used to connect to HTTP origins
resource "azion_connector" "http_connector" {
  connector = {
    name   = "tf-test-http-connector"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          endpoint = {
            address   = "192.168.1.100"
            http_port = 80
            active    = true
          }
        }
      ]
    }
  }
}

# HTTP Connector with all options
resource "azion_connector" "http_connector_full" {
  connector = {
    name   = "tf-test-http-connector-full"
    type   = "http"
    active = true
    http_attributes = {
      addresses = [
        {
          endpoint = {
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
            method              = "round_robin"
            max_retries         = 3
            connection_timeout  = 60
            read_write_timeout  = 120
          }
        }
        origin_shield = {
          enabled = false
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
```

---

## Error Handling

### Standard Error Handling Pattern

The connector resource uses comprehensive error handling with:

1. **Response body closure** - Always close response bodies, even on error paths
2. **Rate limiting (429) retry** - Automatic retries with exponential backoff
3. **Detailed error messages** - Include API response body in diagnostics

### Error Handling Helper Function

```go
// addConnectorAPIError adds an appropriate error to diagnostics based on the API response.
func addConnectorAPIError(diagnostics *diag.Diagnostics, err error, response *http.Response, operation string) {
    if response == nil {
        diagnostics.AddError(err.Error(), "No response received")
        return
    }

    bodyBytes, errReadAll := io.ReadAll(response.Body)
    if errReadAll != nil {
        diagnostics.AddError(errReadAll.Error(), "Failed to read response body")
        return
    }
    bodyString := string(bodyBytes)
    diagnostics.AddError(
        fmt.Sprintf("API Error during %s", operation),
        bodyString,
    )
}
```

### Handling 404 Not Found

For Read operations, handle 404 specially to remove the resource from state:

```go
if response != nil && response.StatusCode == http.StatusNotFound {
    resp.State.RemoveResource(ctx)
    return
}
```

For Delete operations, 404 is not an error (resource already deleted):

```go
if response != nil && response.StatusCode == http.StatusNotFound {
    return  // Resource already deleted, success
}
```

---

## Common Issues

### 1. "Provider returned invalid result object after apply" - Unknown Values

**Problem:** When using `Optional: true` and `Computed: true` with pointer types:
```
Error: Provider returned invalid result object after apply
After the apply operation, the provider still indicated an unknown value
```

**Solution:** Use `types.Object` instead of pointer types for nested attributes that are both Optional and Computed:

```go
// Wrong
ConnectionOptions *HTTPConnectionOptionsModel `tfsdk:"connection_options"`

// Correct
ConnectionOptions types.Object `tfsdk:"connection_options"`
```

### 2. "Provider produced inconsistent result after apply"

**Problem:** API returns default values that weren't in the plan:
```
Error: Provider produced inconsistent result after apply
unexpected new value: .connection_options: was null, but now cty.ObjectVal(...)
```

**Solution:** Add `Computed: true` to the schema for attributes where the API provides defaults:

```go
"connection_options": schema.SingleNestedAttribute{
    Optional: true,
    Computed: true, // Important: API provides defaults
    Attributes: map[string]schema.Attribute{ /* ... */ },
},
```

### 3. Converting types.Object to Model

Use `As()` with `basetypes.ObjectAsOptions{}`:

```go
var model HTTPConnectionOptionsModel
diags := objectValue.As(ctx, &model, basetypes.ObjectAsOptions{})
```

### 4. Converting Model to types.Object

Use `types.ObjectValueFrom()` with attrTypes:

```go
objectValue, diags := types.ObjectValueFrom(ctx, model.attrTypes(), model)
```

---

## Summary Checklist

When implementing the connector resource:

1. [ ] Use `types.Object` for nested attributes with `Optional: true` and `Computed: true`
2. [ ] Add `Computed: true` for attributes where API returns defaults
3. [ ] Implement `attrTypes()` method for each model used as `types.Object`
4. [ ] Use `basetypes.ObjectAsOptions{}` when calling `As()` on types.Object
5. [ ] Use `types.ObjectValueFrom()` when converting models to types.Object
6. [ ] Nest `storage_attributes` and `http_attributes` inside `connector` block
7. [ ] Only support `storage` and `http` types (no `live_ingest`)
8. [ ] Close response bodies BEFORE error checking (`defer response.Body.Close()`)
9. [ ] Implement 429 retry logic using `utils.RetryOn429`
10. [ ] Handle 404 specially in Read (remove from state) and Delete (ignore) operations
