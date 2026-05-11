---
subcategory: "WAF"
layout: "azion"
page_title: "Guide: Implementing WAF Tuning Recommendations in Terraform"
description: |-
  Learn how to convert WAF Tuning recommendations from the Azion Console into Terraform configurations.
---

# Implementing WAF Tuning Recommendations in Terraform

WAF Tuning is a feature available in the Azion Console that analyzes your domain's traffic and provides customized recommendations for WAF rules and exceptions. Since this feature is only available through the Console (not via API), this guide explains how to take those recommendations and implement them using Terraform.

## Overview

### What is WAF Tuning?

WAF Tuning analyzes the traffic patterns of your domain combined with your firewall configuration (with WAF enabled) to recommend:

- **WAF Rule Exceptions**: Specific rules to bypass for legitimate traffic patterns
- **False Positive Mitigation**: Recommendations to reduce false positives without compromising security
- **Custom Traffic Allowances**: Rules to allow specific trusted traffic patterns

### Why Use Terraform for WAF Tuning?

While WAF Tuning recommendations are generated in the Console, managing them through Terraform provides:

- **Version Control**: Track changes to your WAF configuration over time
- **Reproducibility**: Easily replicate configurations across environments
- **Infrastructure as Code**: Maintain consistency and documentation
- **Audit Trail**: Clear history of when and why exceptions were added

## Process Overview

1. **Analyze Traffic** in the Azion Console using WAF Tuning
2. **Review Recommendations** and identify which exceptions to implement
3. **Convert Recommendations** to Terraform `azion_waf_rule_set` resources
4. **Apply Configuration** using Terraform
5. **Monitor and Iterate** as needed

## Step 1: Analyze Traffic in the Console

1. Log in to the [Azion Console](https://console.azion.com/)
2. Navigate to your **WAF**, which is configured in a Firewall
3. Access the **WAF Tuning** feature
4. Select the time period to analyze
5. Review the traffic patterns and detected anomalies

The Console will display recommendations based on:
- Frequent false positives
- Legitimate traffic blocked by WAF rules
- Traffic patterns that require specific exceptions

## Step 2: Understanding the Recommendations

WAF Tuning recommendations typically include:

| Recommendation Type | Description | Terraform Resource |
|-------------------|-------------|-------------------|
| Rule Exception | Bypass a specific WAF rule for certain conditions | `azion_waf_rule_set` |
| Path Whitelist | Allow specific URL paths | `azion_waf_rule_set` with `path` |
| Header Exception | Allow traffic with specific headers | `azion_waf_rule_set` with header conditions |
| Query String Exception | Allow specific query parameters | `azion_waf_rule_set` with query conditions |

## Step 3: Converting Recommendations to Terraform

### Basic Structure

Each WAF Tuning recommendation can be converted to an `azion_waf_rule_set` resource:

```hcl
resource "azion_waf_rule_set" "tuning_recommendation" {
  waf_id = azion_waf.main.id
  
  result = {
    name       = "Description from WAF Tuning"
    active     = true
    rule_id    = 0  # 0 = applies to all rules, or specify a rule ID
    path       = "/api/*"  # Optional: path pattern
    operator   = "regex"   # "regex" or "contains"
    
    conditions = [
      {
        match          = "any_url"
        condition_type = "generic"
      }
    ]
  }
}
```

### Common Recommendation Patterns

#### 1. Allow Specific API Endpoint

**Console Recommendation**: "Allow traffic to `/api/health` endpoint"

**Terraform Implementation**:

```hcl
resource "azion_waf_rule_set" "health_endpoint" {
  waf_id = azion_waf.main.id
  
  result = {
    name     = "Allow Health Check Endpoint"
    active   = true
    rule_id  = 0
    path     = "/api/health"
    operator = "regex"
    
    conditions = [
      {
        match          = "any_url"
        condition_type = "generic"
      }
    ]
  }
}
```

#### 2. Allow Specific Header Value

**Console Recommendation**: "Allow requests with `X-Internal-Service` header"

**Terraform Implementation**:

```hcl
resource "azion_waf_rule_set" "internal_service_header" {
  waf_id = azion_waf.main.id
  
  result = {
    name     = "Allow Internal Service Header"
    active   = true
    rule_id  = 0
    
    conditions = [
      {
        match          = "specific_http_header_name"
        name           = "X-Internal-Service"
        condition_type = "specific_on_name"
      }
    ]
  }
}
```

#### 3. Allow Specific Query Parameter

**Console Recommendation**: "Allow `callback` query parameter for JSONP"

**Terraform Implementation**:

```hcl
resource "azion_waf_rule_set" "jsonp_callback" {
  waf_id = azion_waf.main.id
  
  result = {
    name     = "Allow JSONP Callback Parameter"
    active   = true
    rule_id  = 0
    
    conditions = [
      {
        match          = "specific_query_string_name"
        name           = "callback"
        condition_type = "specific_on_name"
      }
    ]
  }
}
```

#### 4. Bypass Specific Rule for File Upload

**Console Recommendation**: "Rule ID 1001234 is blocking legitimate file uploads"

**Terraform Implementation**:

```hcl
resource "azion_waf_rule_set" "file_upload_exception" {
  waf_id = azion_waf.main.id
  
  result = {
    name     = "Allow File Uploads"
    active   = true
    rule_id  = 1001234  # Specific rule ID from recommendation
    path     = "/upload"
    operator = "regex"
    
    conditions = [
      {
        match          = "file_extension"
        condition_type = "generic"
      }
    ]
  }
}
```

#### 5. Multiple Conditions (Complex Exception)

**Console Recommendation**: "Allow internal API calls with specific header and path"

**Terraform Implementation**:

```hcl
resource "azion_waf_rule_set" "internal_api" {
  waf_id = azion_waf.main.id
  
  result = {
    name     = "Allow Internal API Access"
    active   = true
    rule_id  = 0
    path     = "/internal/*"
    operator = "regex"
    
    conditions = [
      {
        match          = "specific_http_header_name"
        name           = "X-Internal-Auth"
        condition_type = "specific_on_name"
      },
      {
        match          = "specific_http_header_value"
        value          = "trusted-service"
        condition_type = "specific_on_value"
      }
    ]
  }
}
```

## Step 4: Applying the Configuration

### Initialize Terraform

```bash
terraform init
```

### Plan the Changes

Review what will be created:

```bash
terraform plan
```

### Apply the Configuration

```bash
terraform apply
```

### Verify in Console

After applying, verify in the Azion Console that the exceptions appear correctly in your WAF configuration.

## Best Practices

### 1. Naming Conventions

Use descriptive names that include the source:

```hcl
name = "WAF Tuning: Description of the exception"
```

This makes it clear which exceptions were derived from WAF Tuning recommendations.

### 2. Document with Comments

Add comments explaining why each exception was created:

```hcl
# WAF Tuning Recommendation from 2024-01-15
# Reason: False positive on legitimate API calls from payment webhook
# Analysis Period: 30 days
resource "azion_waf_rule_set" "payment_webhook" {
  # ...
}
```

### 3. Use Variables for Environment Differences

```hcl
variable "environment" {
  type = string
}

resource "azion_waf_rule_set" "health_check" {
  waf_id = azion_waf.production.id
  
  result = {
    name   = "WAF Tuning (${var.environment}): Health Check Endpoint"
    # ...
  }
}
```

### 4. Least Privilege Principle

Be specific with exceptions to avoid over-broad allowances:

```hcl
# Good: Specific path
path = "/api/webhooks/payment"

# Avoid: Too broad
path = "/*"
```

### 5. Regular Review

Schedule regular reviews of WAF Tuning recommendations:

1. Re-run WAF Tuning analysis monthly
2. Remove exceptions that are no longer needed
3. Adjust sensitivity levels as traffic patterns change

### 6. Version Control the Analysis Date

Track when recommendations were generated:

```hcl
# WAF Tuning Analysis: 2024-01-15 to 2024-02-15
# Total blocked requests analyzed: 10,000
# False positive rate: 2.3%
```

## Condition Type Reference

### Generic Conditions

Use when matching any value of a type:

| Match Type | Description | Example Use Case |
|-----------|-------------|------------------|
| `any_url` | Any URL | Path-based exceptions |
| `any_http_header_name` | Any header name | Header presence check |
| `any_http_header_value` | Any header value | Any header value |
| `any_query_string_name` | Any query param | Parameter presence |
| `any_query_string_value` | Any query value | Any query value |
| `body_form_field_name` | Form field name | Form field presence |
| `body_form_field_value` | Form field value | Any form value |
| `file_extension` | File extension | File upload handling |
| `raw_body` | Raw request body | Body content matching |

### Specific Conditions on Name

Use when you need to match a specific name:

| Match Type | Requires `name` Field | Description |
|-----------|---------------------|-------------|
| `specific_http_header_name` | Yes | Match specific header name |
| `specific_query_string_name` | Yes | Match specific query parameter |
| `specific_body_form_field_name` | Yes | Match specific form field |

### Specific Conditions on Value

Use when you need to match a specific value:

| Match Type | Requires `value` Field | Description |
|-----------|----------------------|-------------|
| `specific_http_header_value` | Yes | Match specific header value |
| `specific_query_string_value` | Yes | Match specific query value |
| `specific_body_form_field_value` | Yes | Match specific form field value |

## Troubleshooting

### Exception Not Working

1. **Check `active` is `true`**: Inactive exceptions don't affect traffic
2. **Verify `waf_id` matches**: Ensure the WAF ID is correct
3. **Check `rule_id`**: Use `0` for all rules, or verify specific rule ID
4. **Validate path pattern**: Ensure regex pattern is correct
5. **Review operator**: Use `regex` for patterns, `contains` for substring matching

### False Positives Still Occurring

1. **Check condition type**: Ensure you're using the right condition type
2. **Add more specific conditions**: Combine multiple conditions
3. **Verify the match type**: Use specific conditions for targeted exceptions
4. **Review WAF Tuning again**: Traffic patterns may have changed

### Terraform State Issues

If you need to import existing exceptions:

```bash
terraform import azion_waf_rule_set.example <exception_id>
```

**Note**: The `waf_id` must be in the Terraform configuration for import to work.

## Related Resources

- [azion_waf](../resources/waf.md) - Main WAF configuration
- [azion_waf_rule_set](../resources/waf_rule_set.md) - WAF exceptions
- [azion_firewall_main_setting](../resources/firewall_main_setting.md) - Edge Firewall configuration

## Next Steps

1. Set up WAF Tuning analysis in the Azion Console
2. Review recommendations weekly or monthly
3. Convert recommendations to Terraform code
4. Test in a staging environment before production
5. Monitor WAF logs for continued effectiveness
