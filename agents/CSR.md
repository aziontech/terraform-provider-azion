# Certificate Signing Request (CSR) - Terraform Provider Implementation Guide

This document provides guidance for implementing the Certificate Signing Request (CSR) resource in the Terraform provider.

## API Information

- **SDK Package**: `github.com/aziontech/azionapi-v4-go-sdk-dev/tls-api`
- **API Service**: `DigitalCertificatesCertificateSigningRequestsAPIService` (for Create)
- **API Service**: `DigitalCertificatesCertificatesAPI` (for Read and Delete)
- **Endpoint**: `POST /workspace/tls/csr` (Create)
- **Endpoint**: `GET /workspace/tls/certificates/{id}` (Read)
- **Endpoint**: `DELETE /workspace/tls/certificates/{id}` (Delete)
- **Available Methods**: Create, Read, Delete (no Update)

## Important Characteristics

The CSR endpoint is a **create-only** endpoint. This means:
- Only the `Create` method is available via the CSR API
- **Read** uses the standard digital certificates endpoint
- **Delete** uses the standard digital certificates endpoint
- There is **no Update endpoint** to modify a CSR - changes require recreation

## SDK Types

### CertificateSigningRequest (Request Model)

The request model for creating a CSR:

```go
type CertificateSigningRequest struct {
    // Required fields
    Name              string  `json:"name"`
    CommonName        string  `json:"common_name"`
    Country           string  `json:"country"`
    State             string  `json:"state"`
    Locality          string  `json:"locality"`
    Organization      string  `json:"organization"`
    OrganizationUnity string  `json:"organization_unity"`
    Email             string  `json:"email"`

    // Optional fields
    Id               *int64           `json:"id,omitempty"`
    Certificate      NullableString   `json:"certificate,omitempty"`
    PrivateKey       NullableString   `json:"private_key,omitempty"`
    Issuer           NullableString   `json:"issuer,omitempty"`
    SubjectName      []string         `json:"subject_name,omitempty"`
    Validity         NullableString   `json:"validity,omitempty"`
    Type             *string          `json:"type,omitempty"`
    Managed          *bool            `json:"managed,omitempty"`
    Status           *string          `json:"status,omitempty"`
    StatusDetail     *string          `json:"status_detail,omitempty"`
    Csr              NullableString   `json:"csr,omitempty"`
    Challenge        *string          `json:"challenge,omitempty"`
    Authority        *string          `json:"authority,omitempty"`
    KeyAlgorithm     *string          `json:"key_algorithm,omitempty"`
    Active           *bool            `json:"active,omitempty"`
    ProductVersion   *string          `json:"product_version,omitempty"`
    LastEditor       *string          `json:"last_editor,omitempty"`
    CreatedAt        NullableTime     `json:"created_at,omitempty"`
    LastModified     *time.Time       `json:"last_modified,omitempty"`
    RenewedAt        NullableTime     `json:"renewed_at,omitempty"`
    AlternativeNames []string         `json:"alternative_names,omitempty"`
}
```

### CertificateResponse (Response Model)

The API returns a `CertificateResponse` which wraps a `Certificate` object:

```go
type CertificateResponse struct {
    State *string    `json:"state,omitempty"`
    Data  Certificate `json:"data"`
}
```

## Field Descriptions

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Name identifier for the CSR |
| `common_name` | string | Common Name (CN) for the certificate subject |
| `country` | string | Country code (e.g., "US", "BR") |
| `state` | string | State or province name |
| `locality` | string | City or locality name |
| `organization` | string | Organization name |
| `organization_unity` | string | Organizational unit name |
| `email` | string | Contact email address |

### Optional Fields

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `alternative_names` | []string | Subject Alternative Names (SANs) | - |
| `type` | string | Certificate type: `edge_certificate` or `trusted_ca_certificate` | - |
| `key_algorithm` | string | Key algorithm: `rsa_2048`, `rsa_4096`, or `ecc_384` | - |
| `active` | bool | Whether the certificate is active | - |

### Computed Fields (Returned by API)

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Unique identifier of the created certificate |
| `csr` | string | The generated Certificate Signing Request content |
| `status` | string | Status: `pending`, `challenge_verification`, `active`, `inactive`, `expired`, `failed` |
| `status_detail` | string | Detailed status information |
| `managed` | bool | Whether the certificate is managed by Azion |
| `issuer` | string | Certificate issuer |
| `subject_name` | []string | Subject names included in the certificate |
| `validity` | string | Certificate validity period |
| `challenge` | string | Challenge type: `dns` or `http` |
| `authority` | string | Certificate authority (e.g., `lets_encrypt`) |
| `product_version` | string | Product version |
| `last_editor` | string | Last user to modify the certificate |
| `created_at` | time | Creation timestamp |
| `last_modified` | time | Last modification timestamp |
| `renewed_at` | time | Renewal timestamp (for managed certificates) |

## Terraform Resource Implementation

### Resource Name

```
azion_certificate_signing_request
```

### Schema Definition

```go
func (r *csrResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Provides a certificate signing request (CSR) resource. This resource allows you to create CSRs for generating SSL/TLS certificates.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the resource.",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema version of the resource.",
                Computed:    true,
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update.",
                Computed:    true,
            },
            "results": schema.SingleNestedAttribute{
                Description: "The CSR details.",
                Required:    true,
                Attributes: map[string]schema.Attribute{
                    // Required input fields
                    "name": schema.StringAttribute{
                        Description: "Name of the CSR.",
                        Required:    true,
                    },
                    "common_name": schema.StringAttribute{
                        Description: "Common Name (CN) for the certificate.",
                        Required:    true,
                    },
                    "country": schema.StringAttribute{
                        Description: "Country code (e.g., US, BR).",
                        Required:    true,
                    },
                    "state": schema.StringAttribute{
                        Description: "State or province name.",
                        Required:    true,
                    },
                    "locality": schema.StringAttribute{
                        Description: "City or locality name.",
                        Required:    true,
                    },
                    "organization": schema.StringAttribute{
                        Description: "Organization name.",
                        Required:    true,
                    },
                    "organization_unity": schema.StringAttribute{
                        Description: "Organizational unit name.",
                        Required:    true,
                    },
                    "email": schema.StringAttribute{
                        Description: "Contact email address.",
                        Required:    true,
                    },
                    // Optional input fields
                    "alternative_names": schema.ListAttribute{
                        Description: "Subject Alternative Names (SANs).",
                        Optional:    true,
                        ElementType: types.StringType,
                    },
                    "certificate_type": schema.StringAttribute{
                        Description: "Type: edge_certificate or trusted_ca_certificate.",
                        Optional:    true,
                    },
                    "key_algorithm": schema.StringAttribute{
                        Description: "Key algorithm: rsa_2048, rsa_4096, or ecc_384.",
                        Optional:    true,
                    },
                    "active": schema.BoolAttribute{
                        Description: "Whether the certificate is active.",
                        Optional:    true,
                    },
                    // Computed fields (returned by API)
                    "id": schema.Int64Attribute{
                        Description: "Unique identifier of the certificate.",
                        Computed:    true,
                    },
                    "csr": schema.StringAttribute{
                        Description: "Generated CSR content.",
                        Computed:    true,
                    },
                    "status": schema.StringAttribute{
                        Description: "Status of the certificate.",
                        Computed:    true,
                    },
                    "status_detail": schema.StringAttribute{
                        Description: "Detailed status information.",
                        Computed:    true,
                    },
                    "managed": schema.BoolAttribute{
                        Description: "Whether managed by Azion.",
                        Computed:    true,
                    },
                    "issuer": schema.StringAttribute{
                        Description: "Certificate issuer.",
                        Computed:    true,
                    },
                    "subject_name": schema.ListAttribute{
                        Description: "Subject names.",
                        Computed:    true,
                        ElementType: types.StringType,
                    },
                    "validity": schema.StringAttribute{
                        Description: "Validity period.",
                        Computed:    true,
                    },
                    "challenge": schema.StringAttribute{
                        Description: "Challenge type.",
                        Computed:    true,
                    },
                    "authority": schema.StringAttribute{
                        Description: "Certificate authority.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor.",
                        Computed:    true,
                    },
                    "created_at": schema.StringAttribute{
                        Description: "Creation timestamp.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp.",
                        Computed:    true,
                    },
                    "renewed_at": schema.StringAttribute{
                        Description: "Renewal timestamp.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}
```

### Create Method

```go
func (r *csrResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan csrResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the CSR request
    csrRequest := azionapi.CertificateSigningRequest{
        Name:              plan.Results.Name.ValueString(),
        CommonName:        plan.Results.CommonName.ValueString(),
        Country:           plan.Results.Country.ValueString(),
        State:             plan.Results.State.ValueString(),
        Locality:          plan.Results.Locality.ValueString(),
        Organization:      plan.Results.Organization.ValueString(),
        OrganizationUnity: plan.Results.OrganizationUnity.ValueString(),
        Email:             plan.Results.Email.ValueString(),
    }

    // Set optional fields if provided
    if !plan.Results.AlternativeNames.IsNull() {
        var altNames []string
        diags = plan.Results.AlternativeNames.ElementsAs(ctx, &altNames, false)
        resp.Diagnostics.Append(diags...)
        csrRequest.AlternativeNames = altNames
    }

    if !plan.Results.Type.IsNull() {
        csrType := plan.Results.Type.ValueString()
        csrRequest.Type = &csrType
    }

    if !plan.Results.KeyAlgorithm.IsNull() {
        keyAlgo := plan.Results.KeyAlgorithm.ValueString()
        csrRequest.KeyAlgorithm = &keyAlgo
    }

    if !plan.Results.Active.IsNull() {
        active := plan.Results.Active.ValueBool()
        csrRequest.Active = &active
    }

    // Call the API
    certificateResponse, response, err := r.client.api.DigitalCertificatesCertificateSigningRequestsAPI.
        CreateCertificateSigningRequest(ctx).
        CertificateSigningRequest(csrRequest).
        Execute()
    
    if err != nil {
        // Handle 429 rate limiting
        if response.StatusCode == 429 {
            certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
                return r.client.api.DigitalCertificatesCertificateSigningRequestsAPI.
                    CreateCertificateSigningRequest(ctx).
                    CertificateSigningRequest(csrRequest).
                    Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            return
        }
    } else {
        if response != nil {
            defer response.Body.Close()
        }
    }

    // Populate state from response
    cert := certificateResponse.GetData()
    plan.Results = populateCSRResultsFromAPI(ctx, cert, plan.Results)
    plan.SchemaVersion = types.Int64Value(1)
    plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Method

The Read method uses the standard digital certificates endpoint:

```go
func (r *csrResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state csrResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the certificate ID from state
    certificateID, err := parseCSRID(state.ID, state.Results.ID)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", err.Error())
        return
    }

    // Call the digital certificates endpoint to read
    certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.
        RetrieveCertificate(ctx, certificateID).
        Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle 429 and other errors...
    }

    // Populate state from response, preserving input fields
    cert := certificateResponse.GetData()
    state.Results = populateCSRResultsFromAPI(ctx, cert, state.Results)
    state.SchemaVersion = types.Int64Value(1)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Delete Method

The Delete method uses the standard digital certificates endpoint:

```go
func (r *csrResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state csrResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the certificate ID from state
    certificateID, err := parseCSRID(state.ID, state.Results.ID)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", err.Error())
        return
    }

    // Call the digital certificates endpoint to delete
    _, response, err := r.client.api.DigitalCertificatesCertificatesAPI.
        DeleteCertificate(ctx, certificateID).
        Execute()
    if err != nil {
        // Handle 429 and other errors...
    }
}
```

### Import State

Import is supported via the digital certificates endpoint:

```go
func (r *csrResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

## Usage Example

```hcl
resource "azion_certificate_signing_request" "example" {
  results = {
    name              = "my-certificate"
    common_name       = "example.com"
    country           = "US"
    state             = "California"
    locality          = "San Francisco"
    organization      = "Example Corp"
    organization_unity = "IT Department"
    email             = "admin@example.com"
    
    # Optional fields
    alternative_names = ["www.example.com", "api.example.com"]
    certificate_type  = "edge_certificate"
    key_algorithm     = "rsa_2048"
    active            = true
  }
}

output "certificate_id" {
  value = azion_certificate_signing_request.example.results.id
}

output "csr_content" {
  value     = azion_certificate_signing_request.example.results.csr
  sensitive = true
}
```

## Import

```sh
terraform import azion_certificate_signing_request.example 12345
```

## Notes

1. **Import Support**: Import is supported via the digital certificates endpoint using the certificate ID.

2. **Force New on Changes**: Any change to the input fields should trigger resource recreation since there's no update API.

3. **Read and Delete**: These operations use the standard digital certificates API endpoint (`/workspace/tls/certificates/{id}`), not the CSR-specific endpoint.

4. **Certificate vs CSR**: After creating a CSR, you get a Certificate object back with an ID and status. The CSR field contains the actual CSR content that can be provided to a Certificate Authority.

5. **Preserving Input Fields**: Since the Certificate API response doesn't include all CSR input fields (like `common_name`, `country`, etc.), the Read method preserves these from the previous state.
