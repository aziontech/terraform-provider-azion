# Runbook — terraform-provider-azion

## Service Identity

| Field | Value |
|-------|-------|
| Name | terraform-provider-azion |
| Type | Terraform Provider |
| Registry | `registry.terraform.io/aziontech/azion` |
| Language | Go 1.24 |
| License | MPL 2.0 |

## Local Development

```bash
# Install dependencies
go mod tidy

# Build and install provider locally
make install

# Install dev version (to ~/.terraform.d/)
make install-dev

# Format code
make fmt

# Run linter
make lint

# Run security checks
make sec

# Run unit tests
make test

# Run acceptance tests (creates real Azion resources)
export AZION_API_TOKEN="your-token"
make testacc

# Run functional tests
make func-init
make func-plan
make func-apply
make func-destroy

# Debug with delve
make debug
```

## CI/CD

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| code-check-and-docs.yml | PR | go mod tidy, fmt, gosec, vet, lint |
| tests.yml | PR | Unit and acceptance tests |
| release.yml | v* tag | GoReleaser build + GitHub Release |
| deploy_main.yml | Push to main | Main branch deployment |
| ci-compliance.yml | PR, weekly | Azion compliance checks |
| ci-security.yml | PR, weekly | Security scanning |
| scc-checker.yml | PR | Source code counter |
| cla.yml | PR | CLA enforcement |

## Release Process

1. Create and push a `v*` tag (e.g., `v1.2.0`)
2. Release workflow triggers GoReleaser
3. GoReleaser builds multi-platform binaries (Linux, macOS, Windows, FreeBSD)
4. SHA256SUMS generated and GPG-signed
5. Published to GitHub Releases
6. Terraform Registry picks up the release automatically

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `AZION_API_TOKEN` | Yes | 40-character Azion API token |
| `AZION_API_ENTRYPOINT` | No | API endpoint override (default: `https://api.azion.com/v4`) |
| `TF_ACC` | Tests | Set to `1` to run acceptance tests |

## Common Issues

### 1. API Rate Limiting (HTTP 429)

**Symptoms**: Terraform operations fail with "too many requests" errors.

**Resolution**: The provider has built-in retry logic (up to 5 retries with backoff). If rate limiting persists, reduce parallelism with `terraform apply -parallelism=2`. For large infrastructure, consider breaking into smaller apply operations.

### 2. Token Authentication Errors

**Symptoms**: 401 or 403 errors on API calls.

**Resolution**: Verify the API token is exactly 40 characters matching `[A-Za-z0-9-_]{40}`. Ensure the token has the necessary permissions for the resources being managed. Set via `AZION_API_TOKEN` env var or in provider configuration.

### 3. Acceptance Test Failures

**Symptoms**: `make testacc` fails creating/modifying resources.

**Resolution**: Acceptance tests create real Azion resources. Ensure `AZION_API_TOKEN` is set with sufficient permissions. Some tests may fail if resources already exist from previous interrupted runs — clean up manually via Azion console.

### 4. SDK Version Mismatch

**Symptoms**: Compilation errors after updating API SDKs.

**Resolution**: The provider uses two SDK generations (V3 and V4). When updating, ensure both `azionapi-v4-go-sdk-dev` and `azionapi-go-sdk` are compatible. Run `go mod tidy` after updates.

### 5. Import State Errors

**Symptoms**: `terraform import` fails with unexpected ID format.

**Resolution**: Resource IDs are stored as strings (converted from API numeric IDs). Use the correct format: simple numeric ID for most resources, or composite `edge_application_id/resource_id` for nested resources.

## Escalation

| Level | Contact | When |
|-------|---------|------|
| L1 | Team Dev Tools & Integrations | Provider bugs, resource issues |
| L2 | Platform team | API-side issues, authentication |
| L3 | HashiCorp | Terraform SDK or Registry issues |
