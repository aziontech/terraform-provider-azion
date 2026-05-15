# Architecture — terraform-provider-azion

## Overview

Terraform provider for managing Azion Edge Computing resources. Implements ~23 resources and ~35 data sources covering DNS zones, Edge Applications, Firewall, WAF, Edge Functions, Storage, Certificates, Connectors, and Workloads. Published to Terraform Registry as `aziontech/azion`.

## Technology Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.24 |
| Terraform SDK | Plugin Framework v1.4 + Plugin SDK v2.29 (mux) |
| Azion API | azionapi-v4-go-sdk (V4) + azionapi-go-sdk (V3 legacy) |
| Release | GoReleaser |
| Security | gosec, golangci-lint |
| Docs | terraform-plugin-docs (auto-generated from schema) |

## Provider Architecture

```
Terraform CLI
    │
    ▼
MUX Server (tf6muxserver)
    ├── Plugin Framework v1 (new resources)
    └── Plugin SDK v2 (legacy resources)
    │
    ▼
Provider Configuration
    │  api_token (required, 40 chars)
    │  env: AZION_API_TOKEN
    │
    ▼
API Client Initialization (internal/config.go)
    └── 9 SDK client instances
        ├── V4: DNSZonesAPI, StorageAPI, WorkloadsAPI, ConnectorsAPI, ...
        └── V3: EdgeFunctions, EdgeApplications, EdgeFirewall,
                DigitalCertificates, NetworkList, WAF, iDNS
```

## Module Structure

```
internal/
├── provider.go                Provider definition, schema, resource/data registration
├── config.go                  API client initialization (9 SDK clients)
├── consts/                    Constants
├── utils/
│   └── utils.go               Type conversions, retry logic, JSON helpers
│
├── Resources (~23)
│   ├── resource_zones.go               DNS zones
│   ├── resource_dns_record.go          DNS records
│   ├── resource_dnssec.go              DNSSEC
│   ├── resource_application_main_settings.go
│   ├── resource_application_cache_setting.go
│   ├── resource_application_rules_engine.go
│   ├── resource_application_function_instance.go
│   ├── resource_application_device_groups.go
│   ├── resource_application_origin.go
│   ├── resource_firewall_main_settings.go
│   ├── resource_firewall_rules_engine.go
│   ├── resource_firewall_function_instance.go
│   ├── resource_function.go            Edge Functions
│   ├── resource_waf.go                 WAF rules
│   ├── resource_waf_rule_set.go        WAF rule sets
│   ├── resource_certificate.go         Digital certificates
│   ├── resource_certificate_request.go Let's Encrypt integration
│   ├── resource_network_list.go        Network lists
│   ├── resource_bucket.go              Storage buckets
│   ├── resource_connector.go           HTTP/Storage connectors
│   ├── resource_workload.go            Managed workloads
│   ├── resource_workload_deployment.go Workload deployments
│   └── resource_custom_page.go         Custom error pages
│
└── Data Sources (~35)
    ├── data_source_zone.go / data_source_zones.go
    ├── data_source_application_*.go (6 variants)
    ├── data_source_firewall_*.go (3 variants)
    ├── data_source_function.go / data_source_functions.go
    ├── data_source_waf*.go (4 variants)
    ├── data_source_certificate*.go
    ├── data_source_network_list*.go
    ├── data_source_bucket*.go
    ├── data_source_connector*.go
    └── data_source_workload*.go
```

## Resource Lifecycle

Each resource implements the standard Terraform lifecycle:

```
Schema() → Configure() → Create() → Read() → Update() → Delete()
                                                    │
                                              ImportState()
```

**API interaction pattern**:
1. Parse Terraform plan into Go model struct
2. Convert to SDK request type
3. Call Azion API via SDK client
4. Retry on HTTP 429 (rate limit, up to 5 retries)
5. Parse response, convert to Terraform types
6. Update state with computed fields + LastUpdated timestamp

## Dual SDK Strategy

The provider uses two SDK generations to manage the migration from V3 to V4 APIs:

| SDK | API Version | Used For |
|-----|-------------|----------|
| azionapi-v4-go-sdk | V4 | DNS, Storage, Workloads, Connectors, newer resources |
| azionapi-go-sdk | V3 | Edge Functions, Applications, Firewall, WAF, Certificates |

A mux server combines both Framework v1 and SDK v2 providers, allowing gradual migration without breaking existing resources.

## Authentication

- **Token**: 40-character API token (`[A-Za-z0-9-_]{40}`)
- **Header**: `Authorization: token <API_TOKEN>`
- **Endpoint**: `https://api.azion.com/v4` (override via `AZION_API_ENTRYPOINT`)
- **User-Agent**: `terraform/<version> terraform-provider-azion/<version>`

## Release Process

1. Tag with `v*` pattern triggers release workflow
2. GoReleaser builds for Linux, macOS, Windows, FreeBSD (amd64, arm64, 386, arm)
3. ZIP archives with SHA256SUMS (GPG-signed)
4. Published to GitHub Releases
5. Terraform Registry picks up via `terraform-registry-manifest.json` (Protocol v6)
