# Terraform Provider Azion - Code Generation Guide

This document provides comprehensive guidance for AI agents generating Terraform provider code from an OpenAPI specification. It documents the patterns, conventions, and implementation details used in this provider.
It should only be used when the agent is explicitly instructed to edit or update the specified data_sources and/or resources.
In order to edit/update a data_source or resource, the agent should receive a .yaml file OR should be prompted to look into the SDK code for updates. 

## Package-Specific Documentation

Use these files ONLY when prompted to edit/update said data_sources and/or resources.
For detailed documentation on specific packages, see the `agents/` folder:

### Applications
- **[agents/APPLICATIONS.md](agents/APPLICATIONS.md)** - Applications (Main Settings)
- **[agents/CACHE_SETTINGS.md](agents/CACHE_SETTINGS.md)** - Cache Settings
- **[agents/RULES_ENGINE.md](agents/RULES_ENGINE.md)** - Rules Engine
- **[agents/DEVICE_GROUPS.md](agents/DEVICE_GROUPS.md)** - Device Groups
- **[agents/FUNCTIONS_INSTANCES.md](agents/FUNCTIONS_INSTANCES.md)** - Application Function Instances

### Firewall
- **[agents/FIREWALL.md](agents/FIREWALL.md)** - Firewall (Main Settings)
- **[agents/FIREWALL_INSTANCE.md](agents/FIREWALL_INSTANCE.md)** - Firewall Function Instances
- **[agents/FIREWALL_RULES_ENGINE.md](agents/FIREWALL_RULES_ENGINE.md)** - Firewall Rules Engine

### Functions
- **[agents/FUNCTIONS.md](agents/FUNCTIONS.md)** - Functions

### Intelligent DNS
- **[agents/ZONES.md](agents/ZONES.md)** - DNS Zones
- **[agents/RECORDS.md](agents/RECORDS.md)** - DNS Records
- **[agents/DNSSEC.md](agents/DNSSEC.md)** - DNSSEC

### WAF
- **[agents/WAF.md](agents/WAF.md)** - WAF (Main Settings)
- **[agents/WAF_RULE_SETS.md](agents/WAF_RULE_SETS.md)** - WAF Rule Sets

### Other Resources
- **[agents/NETWORK_LISTS.md](agents/NETWORK_LISTS.md)** - Network Lists
- **[agents/CONNECTOR.md](agents/CONNECTOR.md)** - Connector
- **[agents/CUSTOM_PAGES.md](agents/CUSTOM_PAGES.md)** - Custom Pages
- **[agents/DIGITAL_CERTIFICATE.md](agents/DIGITAL_CERTIFICATE.md)** - Digital Certificates
- **[agents/WORKLOAD.md](agents/WORKLOAD.md)** - Workloads
- **[agents/WORKLOAD_DEPLOYMENT.md](agents/WORKLOAD_DEPLOYMENT.md)** - Workload Deployments

---

## Table of Contents

1. [Project Structure](#project-structure)
2. [SDK and API Client Configuration](#sdk-and-api-client-configuration)
3. [Error Handling](#error-handling)
4. [Type Conversions](#type-conversions)
5. [Testing Patterns](#testing-patterns)
6. [Documentation Generation](#documentation-generation)
7. [Provider Registration](#provider-registration)

---

## Project Structure

```
terraform-provider-azion/
├── main.go                    # Provider entry point
├── internal/
│   ├── config.go              # API client configuration
│   ├── provider.go            # Provider definition and registration
│   ├── data_source_*.go       # Data source implementations
│   ├── resource_*.go          # Resource implementations
│   ├── consts/
│   │   └── consts.go          # Constants
│   └── utils/
│       └── utils.go           # Utility functions
├── docs/
│   ├── index.md
│   ├── data-sources/          # Data source documentation
│   └── resources/             # Resource documentation
└── examples/
    ├── data-sources/          # Data source examples
    └── resources/             # Resource examples
```

### File Naming Conventions

- **Data Sources (singular)**: `data_source_<resource_name>.go` - For reading a single resource by ID
  - Example: `data_source_function.go` → `azion_function`
  
- **Data Sources (plural)**: `data_source_<resource_name>s.go` - For listing multiple resources
  - Example: `data_source_functions.go` → `azion_functions`

- **Resources**: `resource_<resource_name>.go` - For CRUD operations
  - Example: `resource_function.go` → `azion_function`

---

## SDK and API Client Configuration

### Multiple SDK Pattern

The provider uses **multiple SDKs** for different API endpoints. This is critical to understand:

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azionapi-v4-go-sdk-dev)
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // Legacy SDKs (azionapi-go-sdk)
    edgeApplicationsApi              *edgeapplications.APIClient
    edgefunctionsConfig              *edgefunctions.Configuration
    edgefunctionsApi                 *edgefunctions.APIClient
    digitalCertificatesConfig        *digital_certificates.Configuration
    digitalCertificatesApi           *digital_certificates.APIClient
    networkListConfig                *networklist.Configuration
    networkListApi                   *networklist.APIClient
    edgefirewallConfig               *edgefirewall.Configuration
    edgeFirewallApi                  *edgefirewall.APIClient
    // ... more SDK clients
}
```

### Important: SDK Import Path

**The V4 SDK import path is:**

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

Previously, the SDK used `edge-api` in the import path. This has been changed to `azion-api` in order to use a all-in-one SDK. YOU MUST USE THIS IMPORT FOR THE APIS THAT USE IT. Any reference to `edge-api` or `edgeApi` in legacy code or documentation should use `azion-api` or `api` respectively.

### Important: Naming Convention Difference

**The "edge" prefix is no longer used in V4 SDK.**

In the legacy SDKs, resources and API clients used the `edge` prefix (e.g., `edgeApplicationsApi`, `edgefunctionsApi`, `edgeFirewallApi`). This prefix has been **removed in the V4 SDK** to provide cleaner, more concise naming.

| Legacy SDK (with `edge` prefix) | V4 SDK (no prefix) |
|--------------------------------|-------------------|
| `edgeApplicationsApi` | `applicationsApi` |
| `edgefunctionsApi` | `functionsApi` |
| `edgeFirewallApi` | `firewallApi` |
| `edgeApplicationId` | `applicationId` |
| `edgeConfig` / `edgeApi` | `apiConfig` / `api` |

When implementing new resources using the V4 API:
- Use naming without the `edge` prefix for variables, structs, and function parameters
- The Terraform resource names still use `edge_application`, `edge_function`, etc. for backwards compatibility with existing configurations
- Only the internal Go code naming follows the new convention

### Client Configuration Pattern

```go
func Client(APIToken string, userAgent string) *apiClient {
    client := &apiClient{
        apiConfig: azionapi.NewConfiguration(),
        // ... initialize all configs
    }
    
    // V4 SDK uses fixed URL
    client.apiConfig.Servers[0].URL = "https://api.azion.com/v4"
    
    // Legacy SDKs can use custom entrypoint
    if envApiEntrypoint := os.Getenv("AZION_API_ENTRYPOINT"); envApiEntrypoint != "" {
        client.domainsConfig.Servers[0].URL = envApiEntrypoint
        // ... set for all legacy SDKs
    }
    
    // Add authentication headers
    client.apiConfig.AddDefaultHeader("Authorization", "token "+APIToken)
    client.apiConfig.AddDefaultHeader("Accept", "application/json; version=3")
    client.apiConfig.UserAgent = userAgent
    
    // Create API clients
    client.api = azionapi.NewAPIClient(client.apiConfig)
    
    return client
}
```

---

## Error Handling

### Standard Error Handling Pattern

```go
if err != nil {
    // 1. Check for 429 (rate limiting)
    if response.StatusCode == 429 {
        result, response, err = utils.RetryOn429(func() (*ResponseType, *http.Response, error) {
            return client.API.Method(ctx, params).Execute()
        }, 5)  // Max 5 retries
        
        if response != nil {
            defer response.Body.Close()
        }
        
        if err != nil {
            resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
            return
        }
    } else {
        // 2. Read error body for details
        bodyBytes, errReadAll := io.ReadAll(response.Body)
        if errReadAll != nil {
            resp.Diagnostics.AddError(errReadAll.Error(), "err")
        }
        bodyString := string(bodyBytes)
        resp.Diagnostics.AddError(err.Error(), bodyString)
        return
    }
}
```

### Special Error Codes

```go
// For Read operations - handle 404 specially
if response.StatusCode == http.StatusNotFound {
    resp.State.RemoveResource(ctx)  // Mark resource as deleted
    return
}

// Custom error messages for common codes
func errPrint(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "Resource not found"
    default:
        usrMsg = err.Error()
    }
    return usrMsg, fmt.Sprintf("%d - %s", errCode, usrMsg)
}
```

---

## Type Conversions

### Terraform Types to Go Types

```go
// String
name := plan.Name.ValueString()
namePtr := plan.Name.ValueStringPointer()

// Int64
id := plan.ID.ValueInt64()

// Bool
active := plan.Active.ValueBool()
activePtr := plan.Active.ValueBoolPointer()

// Check if set
if !plan.OptionalField.IsNull() && !plan.OptionalField.IsUnknown() {
    // Field is set
}
```

### Go Types to Terraform Types

```go
// String
types.StringValue(str)
types.StringPointerValue(&str)  // Returns null if nil

// Int64
types.Int64Value(int64Val)
types.Int64PointerValue(&int64Val)

// Bool
types.BoolValue(boolVal)
types.BoolPointerValue(&boolVal)

// List of strings
types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a"), types.StringValue("b")})
```

### JSON Field Handling

```go
// Convert JSON string from Terraform to interface{} for API
func ConvertStringToInterface(jsonArgs string) (interface{}, error) {
    var data map[string]interface{}
    err := json.Unmarshal([]byte(jsonArgs), &data)
    if err != nil {
        return nil, err
    }
    return data, nil
}

// Convert interface{} from API to JSON string for Terraform
func ConvertInterfaceToString(jsonArgs interface{}) (string, error) {
    jsonArgsStr, err := json.Marshal(jsonArgs)
    if err != nil {
        return "{}", nil
    }
    return string(jsonArgsStr), nil
}
```

### Time Formatting

```go
// From API response
lastModified := types.StringValue(response.Data.LastModified.Format(time.RFC3339))

// For last_updated field
lastUpdated := types.StringValue(time.Now().Format(time.RFC850))
```

---

## Testing Patterns

### Unit Test Structure

```go
package provider

import (
    "testing"
)

func TestResourceDataSource(t *testing.T) {
    // Test cases
}
```

### Acceptance Test Pattern

```go
func TestAccResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: providerConfig + `resource "azion_resource" "test" {
                    # resource configuration
                }`,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("azion_resource.test", "attribute", "value"),
                ),
            },
            // Import test
            {
                ResourceName:      "azion_resource.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

---

## Documentation Generation

### Data Source Documentation

Located in `docs/data-sources/<resource_name>.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_resource"
description: |-
  Provides a resource data source.
---

# azion_resource

Use this data source to read a specific resource.

## Example Usage

```hcl
data "azion_resource" "example" {
  id = "12345"
}
```

## Argument Reference

* `id` - (Required) The ID of the resource.

## Attribute Reference

* `data` - The resource data.
  * `name` - Name of the resource.
  * ...
```

### Resource Documentation

Located in `docs/resources/<resource_name>.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_resource"
description: |-
  Provides a resource.
---

# azion_resource

Creates a resource.

## Example Usage

```hcl
resource "azion_resource" "example" {
  # configuration
}
```

## Import

```sh
terraform import azion_resource.example 12345
```

## Argument Reference

* `attribute` - (Required) Description.
  * ...
```

---

## Provider Registration

All data sources and resources must be registered in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionResource,
        // ... register all data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewResource,
        // ... register all resources
    }
}
```

---

## Summary Checklist

When generating a new resource or data source from OpenAPI:

1. **Identify the correct SDK**: V4 (`azion-api`) or legacy (`edgeapplications`, etc.)
2. **Determine ID types**: `int64` or `string` based on SDK
3. **Determine update method**: PUT (full update) or PATCH (partial update)
4. **Create model structs**: With appropriate `tfsdk` tags
5. **Implement schema**: With correct Required/Optional/Computed
6. **Implement all methods**: Create, Read, Update, Delete, ImportState (for resources)
7. **Handle 429 errors**: Use `utils.RetryOn429`
8. **Handle optional fields**: Check `IsNull()` and `IsUnknown()`
9. **Transform nested objects**: Create helper functions if needed
10. **Handle JSON fields**: Use `utils.ConvertStringToInterface` and `utils.ConvertInterfaceToString`
11. **Register in provider.go**: Add to DataSources() or Resources()
12. **Generate documentation**: Create docs and examples
13. **Update example/test files**: After any schema changes, update the corresponding files
14. **Run linters**: After any change, run `golangci-lint run --config .golintci.yml ./internal/...`

---

## Running Linters After Structural Changes

**IMPORTANT**: After making any structural changes to Go code (adding functions, modifying function signatures, adding struct fields, etc.), you MUST run the linters to ensure code quality:

```bash
# Run golangci-lint with the project configuration
golangci-lint run --config .golintci.yml ./internal/...
```

### Key Linter Checks

The project enforces these important linter rules:

| Linter | Purpose | Common Fix |
|--------|---------|------------|
| `bodyclose` | Ensures HTTP response bodies are closed | Add `defer response.Body.Close()` after successful API calls |
| `contextcheck` | Ensures context is passed through function calls | Add `ctx context.Context` as first parameter and pass it to nested calls |
| `godot` | Ensures comments end with a period | Add `.` at end of comments |

### Response Body Closure Pattern

```go
// After successful API response
if response != nil {
    defer response.Body.Close()
}
```

### Context Propagation Pattern

```go
// Function signature with context
func populateResults(ctx context.Context, response *api.Response, plan *model) *model {
    // Use ctx instead of context.Background()
    listValue, _ := types.ListValueFrom(ctx, types.StringType, data)
    return result
}
```

---

## Updating Example and Test Files After Schema Changes

**CRITICAL**: Whenever you modify the schema or structure of a resource or data source, you MUST also update the corresponding example and test files to match the new format.

### Files to Update

When changing a resource/data source schema, check and update these files:

| File Type | Location | Purpose |
|-----------|----------|---------|
| Data Source Examples | `examples/data-sources/azion_<resource_name>/data-source.tf` | Example usage for data sources |
| Resource Examples | `examples/resources/azion_<resource_name>/resource.tf` | Example usage for resources |
| Functional Tests | `func-tests/main.tf` | Integration tests (if applicable) |
| Documentation | `docs/data-sources/<resource_name>.md` | Data source documentation |
| Documentation | `docs/resources/<resource_name>.md` | Resource documentation |

### Common Schema Changes Requiring Example Updates

- **Attribute name changes**: Update all references to the old attribute name
- **Nested structure changes**: Update to reflect new nesting hierarchy
- **Type changes**: Ensure example values match the new type (string vs int, etc.)
- **Required/Optional changes**: Add or remove required fields from examples
- **Block format changes**: Update from flat attributes to nested blocks or vice versa

### Example Update Process

1. After modifying a resource schema, locate the corresponding example files
2. Update the example Terraform configuration to use the new schema format
3. Ensure all required attributes are present in the example
4. Verify the example is valid by running `terraform validate` in the example directory
5. Update documentation to reflect the new schema

### Validation Command

To verify all example files are valid after changes:

```bash
for dir in $(find ./examples -type f -name '*.tf' -exec dirname {} \; | sort -u); do
  echo "===> Validating: $dir <==="
  (cd "$dir" && terraform init -backend=false && terraform validate)
done
```
