# Certificate Request Resource - Agent Documentation

This document provides detailed information about the `azion_certificate_request` resource implementation for AI agents working on this Terraform provider.

## Overview

The `azion_certificate_request` resource allows users to request SSL/TLS certificates from Let's Encrypt automatically. This is different from the standard `azion_digital_certificate` resource, which requires users to provide their own certificate and private key.

## SDK Information

### API Endpoints

| Operation | Endpoint | API Service |
|-----------|----------|-------------|
| Create | `POST /tls/certificates/request` | `DigitalCertificatesRequestACertificateAPIService` |
| Read | `GET /tls/certificates/{id}` | `DigitalCertificatesCertificatesAPI` |
| Delete | `DELETE /tls/certificates/{id}` | `DigitalCertificatesCertificatesAPI` |
| Update | Not supported | N/A |

### SDK Types

```go
// Request type for certificate request
azionapi.CertificateRequest

// Response type
azionapi.CertificateResponse

// Certificate type (returned by Read)
azionapi.Certificate
```

## Implementation Details

### Resource Creation

The Create operation uses the `DigitalCertificatesRequestACertificateAPI` service:

```go
certificateRequest := azionapi.NewCertificateRequest(
    name,        // Required: Name of the certificate
    challenge,   // Required: "dns" or "http"
    authority,   // Required: "lets_encrypt"
    commonName,  // Required: Primary domain name
)

// Set optional fields
certificateRequest.SetAlternativeNames([]string{"www.example.com", "api.example.com"})
certificateRequest.SetKeyAlgorithm("rsa_2048")

// Call the API
response, err := client.api.DigitalCertificatesRequestACertificateAPI.
    RequestCertificate(ctx).
    CertificateRequest(*certificateRequest).
    Execute()
```

### Resource Read

The Read operation uses the standard `DigitalCertificatesCertificatesAPI` service:

```go
response, err := client.api.DigitalCertificatesCertificatesAPI.
    RetrieveCertificate(ctx, certificateID).
    Execute()
```

### Resource Delete

The Delete operation uses the standard `DigitalCertificatesCertificatesAPI` service:

```go
_, err := client.api.DigitalCertificatesCertificatesAPI.
    DeleteCertificate(ctx, certificateID).
    Execute()
```

### Resource Update

**Not Supported.** The certificate request API does not support updates. If a user needs to modify a certificate, they must destroy and recreate the resource.

## Schema Definition

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | String | Name of the certificate |
| `common_name` | String | Primary domain name (CN) |
| `challenge` | String | ACME challenge type: `dns` or `http` |
| `authority` | String | Certificate authority: `lets_encrypt` |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `alternative_names` | List[String] | Subject Alternative Names (SANs) |
| `key_algorithm` | String | Key algorithm: `rsa_2048`, `rsa_4096`, `ecc_384` |

### Computed Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | Int64 | Certificate ID |
| `issuer` | String | Certificate issuer |
| `subject_name` | List[String] | Subject names |
| `validity` | String | Certificate validity period |
| `status` | String | Certificate status |
| `status_detail` | String | Detailed status |
| `certificate_type` | String | Type of certificate |
| `managed` | Bool | Whether managed by Azion |
| `csr` | String | CSR content |
| `active` | Bool | Whether certificate is active |
| `certificate_content` | String | Certificate PEM content (sensitive) |
| `private_key` | String | Private key PEM content (sensitive) |

## Challenge Types

### DNS Challenge

For DNS challenge, Let's Encrypt will create a TXT record at `_acme-challenge.<domain>`. The user must configure DNS to allow Azion to create this record, or manually configure it.

### HTTP Challenge

For HTTP challenge, Let's Encrypt will verify domain ownership by accessing `http://<domain>/.well-known/acme-challenge/<token>`. The user must ensure HTTP access is available on port 80.

## Certificate Status Values

| Status | Description |
|--------|-------------|
| `pending` | Certificate request submitted, waiting for validation |
| `challenge_verification` | ACME challenge verification in progress |
| `active` | Certificate issued and active |
| `inactive` | Certificate is inactive |
| `expired` | Certificate has expired |
| `failed` | Certificate issuance failed |

## Key Algorithms

| Algorithm | Description |
|-----------|-------------|
| `rsa_2048` | 2048-bit RSA key (default) |
| `rsa_4096` | 4096-bit RSA key |
| `ecc_384` | 384-bit Elliptic Curve key |

## Error Handling

### Standard Error Pattern

```go
if err != nil {
    if response.StatusCode == 429 {
        // Retry with exponential backoff
        response, err = utils.RetryOn429(func() (*Response, *http.Response, error) {
            return client.api...Execute()
        }, 5) // Max 5 retries
    } else {
        // Read error body
        bodyBytes, _ := io.ReadAll(response.Body)
        resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
        return
    }
}
```

### Update Not Supported Error

When the user attempts to update a certificate request:

```go
func (r *certificateRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    resp.Diagnostics.AddError(
        "Update not supported",
        "Certificate requests cannot be updated. To change a certificate, you must destroy and recreate the resource.",
    )
}
```

## Data Source Consideration

Due to the nature of this endpoint (only POST available), a data source is **not recommended** for certificate requests. Users should use the existing `azion_digital_certificate` data source to read certificates by ID, as the Read operation uses the same endpoint.

## File Structure

```
internal/
├── resource_certificate_request.go   # Resource implementation
docs/
├── resources/
│   └── certificate_request.md        # User documentation
examples/
└── resources/
    └── azion_certificate_request/
        └── resource.tf                # Example usage
```

## Registration

The resource must be registered in `internal/provider.go`:

```go
func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... other resources
        NewCertificateRequestResource,
    }
}
```

## Differences from Standard Certificate Resource

| Feature | Certificate Request | Digital Certificate |
|---------|---------------------|---------------------|
| Certificate Source | Let's Encrypt (auto) | User-provided |
| Required Fields | name, common_name, challenge, authority | name, certificate_content, private_key |
| Update Support | No | Yes |
| Validation | ACME challenge | None |
| Certificate Type | Managed | User-managed |

## Common Use Cases

1. **Automatic SSL/TLS**: Request certificates for domains hosted on Azion without managing certificate files
2. **Multi-domain Certificates**: Include multiple domains using `alternative_names`
3. **DNS-validated Certificates**: Use DNS challenge for domains without HTTP access
4. **HTTP-validated Certificates**: Use HTTP challenge for quick validation

## Testing Considerations

When testing this resource:
1. Use mock API responses for unit tests
2. Test with both DNS and HTTP challenge types
3. Verify error handling for 429 status codes
4. Test that Update returns appropriate error
5. Verify Read operation uses correct endpoint
