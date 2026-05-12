---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_firewall_rule_engine"
description: |-
  Provides a firewall rules engine resource.
---

# azion_firewall_rule_engine (Resource)

Creates and manages a firewall rules engine rule. The firewall rules engine allows you to create conditional rules with behaviors to control request processing within a firewall.

## Example Usage

### With Parent Firewall

```terraform
# First, create the parent firewall
resource "azion_firewall_main_setting" "example" {
  data = {
    name   = "My Firewall"
    active = true
  }
}

# Then create the rule engine for that firewall
resource "azion_firewall_rule_engine" "example" {
  firewall_id = azion_firewall_main_setting.example.data.id
  results = {
    name        = "Block Specific Path"
    description = "Block requests to specific path"
    active      = true
    behaviors = [
      {
        type = "drop"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${request_uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "/admin.*"
          }
        ]
      }
    ]
  }
}
```

### Basic Rule with Drop Behavior

```terraform
resource "azion_firewall_rule_engine" "example" {
  firewall_id = 1234567890
  results = {
    name        = "Block Specific Path"
    description = "Block requests to specific path"
    active      = true
    behaviors = [
      {
        type = "drop"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${request_uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "/admin.*"
          }
        ]
      }
    ]
  }
}
```

### Rule with Run Function Behavior

```terraform
resource "azion_firewall_rule_engine" "example" {
  firewall_id = 1234567890
  results = {
    name        = "Run Function on API Requests"
    description = "Execute edge function for API requests"
    active      = true
    behaviors = [
      {
        type = "run_function"
        attributes = {
          value = 12345  # Function instance ID
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${request_uri}"
            operator    = "starts_with"
            conditional = "if"
            argument    = "/api/"
          }
        ]
      }
    ]
  }
}
```

### Rule with Set Custom Response Behavior

```terraform
resource "azion_firewall_rule_engine" "example" {
  firewall_id = 1234567890
  results = {
    name        = "Custom Response for Maintenance"
    description = "Return maintenance page"
    active      = true
    behaviors = [
      {
        type = "set_custom_response"
        attributes = {
          status_code  = 503
          content_type = "text/html"
          content_body = "<html><body><h1>Under Maintenance</h1></body></html>"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${host}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "maintenance.example.com"
          }
        ]
      }
    ]
  }
}
```

### Rule with Set WAF Behavior

```terraform
resource "azion_firewall_rule_engine" "example" {
  firewall_id = 1234567890
  results = {
    name        = "Enable WAF"
    description = "Enable WAF protection"
    active      = true
    behaviors = [
      {
        type = "set_waf"
        attributes = {
          waf_id = 98765
          mode   = "blocking"
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${host}"
            operator    = "is_equal"
            conditional = "if"
            argument    = "api.example.com"
          }
        ]
      }
    ]
  }
}
```

### Rule with Set Rate Limit Behavior

```terraform
resource "azion_firewall_rule_engine" "example" {
  firewall_id = 1234567890
  results = {
    name        = "Rate Limit API Requests"
    description = "Limit API request rate"
    active      = true
    behaviors = [
      {
        type = "set_rate_limit"
        attributes = {
          type               = "second"
          limit_by           = "client_ip"
          average_rate_limit = 100
          maximum_burst_size = 200
        }
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${request_uri}"
            operator    = "starts_with"
            conditional = "if"
            argument    = "/api/"
          }
        ]
      }
    ]
  }
}
```

## Supported Behaviors

| Behavior | Description | Requires Attributes |
|----------|-------------|---------------------|
| `deny` | Denies the request with a `403` response. | No |
| `drop` | Drops the connection without sending a response. | No |
| `set_rate_limit` | Applies a rate limit to matching requests. | Yes (`type`, `limit_by`, `average_rate_limit`, `maximum_burst_size`) |
| `set_waf` | Applies a WAF rule set to matching requests. | Yes (`waf_id`, `mode`) |
| `run_function` | Executes a function instance. | Yes (`value`: function instance ID) |
| `set_custom_response` | Returns a custom HTTP response. | Yes (`status_code`, `content_type`, `content_body`) |

### Complex Rule with Multiple Criteria

```terraform
resource "azion_firewall_rule_engine" "example" {
  firewall_id = 1234567890
  results = {
    name        = "Complex Rule"
    description = "Rule with multiple criteria groups"
    active      = true
    behaviors = [
      {
        type = "drop"
      }
    ]
    criteria = [
      {
        entries = [
          {
            variable    = "${request_uri}"
            operator    = "matches"
            conditional = "if"
            argument    = "/admin.*"
          }
        ]
      },
      {
        entries = [
          {
            variable    = "${network}"
            operator    = "is_in_list"
            conditional = "and"
            argument    = "12345"
          }
        ]
      }
    ]
  }
}
```

## Schema

### Required

- `firewall_id` (Number) The firewall identifier.
- `results` (Attributes) The rule configuration. (see [below for nested schema](#nestedatt--results))

### Read-Only

- `id` (String) The ID of this resource in the format `{firewall_id}/{rule_id}`.
- `last_updated` (String) Timestamp of the last Terraform update of the resource.

<a id="nestedatt--results"></a>
### Nested Schema for `results`

Required:

- `behaviors` (Attributes List) Behaviors for the rule. (see [below for nested schema](#nestedatt--results--behaviors))
- `criteria` (Attributes List) Criteria for the rule. (see [below for nested schema](#nestedatt--results--criteria))
- `name` (String) The name of the rule.

Optional:

- `active` (Boolean) Whether the rule is active. Default: `true`
- `description` (String) Description of the rule.

Read-Only:

- `created_at` (String) Creation timestamp.
- `id` (Int64) The ID of the rule.
- `last_editor` (String) Last editor of the rule.
- `last_modified` (String) Last modified timestamp.
- `order` (Int64) Order of the rule.

<a id="nestedatt--results--behaviors"></a>
### Nested Schema for `results.behaviors`

Required:

- `type` (String) Type of behavior. Valid values: `deny`, `drop`, `run_function`, `set_custom_response`, `set_rate_limit`, `set_waf`. See [Supported Behaviors](#supported-behaviors) for details.

Optional:

- `attributes` (Attributes) Behavior attributes. Required for `run_function`, `set_custom_response`, `set_waf`, and `set_rate_limit` behaviors. Not needed for `deny` or `drop`. (see [below for nested schema](#nestedatt--results--behaviors--attributes))

<a id="nestedatt--results--behaviors--attributes"></a>
### Nested Schema for `results.behaviors.attributes`

The attributes available depend on the behavior type:

**For `run_function` behavior:**
- `value` (Int64, Required) The function instance ID to execute.

**For `set_custom_response` behavior:**
- `status_code` (Int64, Required) The HTTP status code to return.
- `content_type` (String, Optional) The content type header value.
- `content_body` (String, Optional) The response body content.

**For `set_waf` behavior:**
- `waf_id` (Int64, Required) The WAF rule set ID to apply.
- `mode` (String, Required) The WAF mode. Valid values: `logging`, `blocking`.

**For `set_rate_limit` behavior:**
- `type` (String, Optional) The rate limit time window. Valid values: `second`, `minute`.
- `limit_by` (String, Required) How to identify clients. Valid values: `client_ip`, `global`.
- `average_rate_limit` (Int64, Required) Maximum requests per time window.
- `maximum_burst_size` (Int64, Optional) Maximum burst size allowed.

<a id="nestedatt--results--criteria"></a>
### Nested Schema for `results.criteria`

Required:

- `entries` (Attributes List) List of criteria entries. (see [below for nested schema](#nestedatt--results--criteria--entries))

<a id="nestedatt--results--criteria--entries"></a>
### Nested Schema for `results.criteria.entries`

Required:

- `conditional` (String) The conditional operator. Valid values: `if`, `and`, `or`.
- `variable` (String) The variable to evaluate. See Supported Variables below.
- `operator` (String) The comparison operator. See Supported Operators below.

Optional:

- `argument` (String) The argument for comparison. Required for most operators.

## Supported Variables

| Variable | Description | Operators |
|----------|-------------|-----------|
| `${header_accept}` | Accept header | matches, does_not_match |
| `${header_accept_encoding}` | Accept-Encoding header | matches, does_not_match |
| `${header_accept_language}` | Accept-Language header | matches, does_not_match |
| `${header_cookie}` | Cookie header | matches, does_not_match |
| `${header_origin}` | Origin header | matches, does_not_match |
| `${header_referer}` | Referer header | matches, does_not_match |
| `${header_user_agent}` | User-Agent header | matches, does_not_match |
| `${host}` | Host | is_equal, is_not_equal, matches, does_not_match |
| `${network}` | Network | is_in_list, is_not_in_list |
| `${request_args}` | Request arguments | is_equal, is_not_equal, matches, does_not_match, exists, does_not_exist |
| `${request_method}` | Request method | is_equal, is_not_equal |
| `${request_uri}` | Request URI | starts_with, does_not_starts_with, is_equal, is_not_equal, matches, does_not_match |
| `${scheme}` | Scheme | is_equal, is_not_equal |
| `${ssl_verification_status}` | SSL verification status | is_equal, is_not_equal |
| `${client_certificate_validation}` | Client certificate validation | is_equal, is_not_equal |

## Supported Operators

| Operator | Description |
|----------|-------------|
| `is_equal` | Equals |
| `is_not_equal` | Does not equal |
| `matches` | Matches regex |
| `does_not_match` | Does not match regex |
| `starts_with` | Starts with |
| `does_not_starts_with` | Does not start with |
| `is_in_list` | Is in network list |
| `is_not_in_list` | Is not in network list |
| `exists` | Header/argument exists |
| `does_not_exist` | Header/argument does not exist |

## Import

Import is supported using the following syntax:

```shell
terraform import azion_firewall_rule_engine.example <firewall_id>/<rule_id>
```

For example:

```shell
terraform import azion_firewall_rule_engine.example 1234567890/987654
```
