---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_certificate_request"
description: |-
  Provides a certificate request resource for Let's Encrypt certificates.
---

# azion_certificate_request

Provides a certificate request resource for Let's Encrypt certificates. This resource allows you to request SSL/TLS certificates from Let's Encrypt automatically.

~> **Note:** This resource only supports creation. Update operations are not available. Read and Delete operations use the standard digital certificates endpoint.

~> **Note about challenge types:**
Use `dns` challenge for DNS-based validation or `http` challenge for HTTP-based validation. The challenge type determines how Let's Encrypt will verify domain ownership.

## Example Usage

### DNS Challenge

```hcl
resource "azion_certificate_request" "example" {
  results = {
    name        = "my-letsencrypt-certificate"
    common_name = "example.com"
    challenge   = "dns"
    authority   = "lets_encrypt"
    
    alternative_names = [
      "www.example.com",
      "api.example.com"
    ]
  }
}
```

### HTTP Challenge

```hcl
resource "azion_certificate_request" "example" {
  results = {
    name        = "my-letsencrypt-certificate"
    common_name = "example.com"
    challenge   = "http"
    authority   = "lets_encrypt"
  }
}
```

### With Key Algorithm

```hcl
resource "azion_certificate_request" "example" {
  results = {
    name          = "my-letsencrypt-certificate"
    common_name   = "example.com"
    challenge     = "dns"
    authority     = "lets_encrypt"
    key_algorithm = "rsa_2048"
    
    alternative_names = [
      "www.example.com",
      "api.example.com"
    ]
  }
}
```

## Import

```sh
terraform import azion_certificate_request.example 12345
```

## Argument Reference

The following arguments are supported:

* `results` - (Required) The certificate request details. See [Results Structure](#results-structure).

### Results Structure

The `results` block supports the following attributes:

* `name` - (Required) Name of the certificate.
* `common_name` - (Required) Common Name (CN) for the certificate. This is the primary domain name.
* `challenge` - (Required) Challenge type for ACME certificate validation. Options: `dns` (Uses DNS to solve the ACME challenge), `http` (Uses HTTP to solve the ACME challenge).
* `authority` - (Required) Certificate authority. Options: `lets_encrypt`.
* `alternative_names` - (Optional) Subject Alternative Names (SANs) for the certificate. Additional domain names to include.
* `key_algorithm` - (Optional) Key algorithm used for the certificate. Options: `rsa_2048` (2048-bit RSA), `rsa_4096` (4096-bit RSA), `ecc_384` (384-bit Prime Field Curve).

## Attribute Reference

The following attributes are exported:

* `id` - Identifier of the resource.
* `schema_version` - Schema version of the resource.
* `last_updated` - Timestamp of the last Terraform update of the resource.
* `results` - The certificate request details. See [Results Structure](#results-structure) for the structure.

### Results Structure (Computed Attributes)

The `results` block also exports the following computed attributes:

* `id` - Identifier of the certificate.
* `issuer` - Issuer of the certificate.
* `subject_name` - Subject name of the certificate.
* `validity` - Validity of the certificate.
* `status` - Status of the certificate. Options: `pending`, `challenge_verification`, `active`, `inactive`, `expired`, `failed`.
* `status_detail` - Status detail of the certificate.
* `certificate_type` - Type of the certificate.
* `managed` - Whether the certificate is managed by Azion.
* `csr` - Certificate Signing Request (CSR).
* `active` - Whether the certificate is active.
* `product_version` - Product version of the certificate.
* `last_editor` - Last editor of the certificate.
* `last_modified` - Last modified timestamp of the certificate.
* `created_at` - Creation timestamp of the certificate.
* `renewed_at` - Renewal timestamp of the certificate.
* `certificate_content` - The content of the certificate (PEM format). This field is populated after the certificate is issued.
* `private_key` - Private key of the certificate (PEM format). This field is populated after the certificate is issued.

## Timeouts

This resource does not support custom timeouts.

## Limitations

~> **Note:** The Update operation is not supported for certificate requests. If you need to modify a certificate, you must destroy and recreate the resource.

~> **Note:** The certificate request process is asynchronous. The certificate will be in `pending` or `challenge_verification` status until Let's Encrypt validates the domain ownership. The certificate content and private key will only be available after the certificate is active.
