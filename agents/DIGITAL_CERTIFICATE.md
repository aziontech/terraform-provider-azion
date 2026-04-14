# Digital Certificates - Code Generation Guide

This document provides specific guidance for implementing Digital Certificate resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by ID)](#singular-data-source-read-by-id)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
3. [Resource Implementation](#resource-implementation)
4. [Schema Definition Patterns](#schema-definition-patterns)
5. [Error Handling](#error-handling)
6. [Type Conversions](#type-conversions)
7. [Common Issues](#common-issues)

---

## SDK Selection

Digital Certificates use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Digital Certificate (Singular Data Source) | `azion-api` (v4) | `api.DigitalCertificatesCertificatesAPI` | `https://api.azion.com/v4` |
| Digital Certificates (Plural Data Source) | `azion-api` (v4) | `api.DigitalCertificatesCertificatesAPI` | `https://api.azion.com/v4` |
| Digital Certificate (Resource) | `azion-api` (v4) | `api.DigitalCertificatesCertificatesAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `Certificate` |
| Update Request Type | `Certificate` (PUT) or `PatchedCertificate` (PATCH) |
| Response Type | `CertificateResponse` with `Data` field |
| List Response Type | `PaginatedCertificateList` |
| Create Pattern | `.CreateCertificate(ctx).Certificate(cert).Execute()` |
| Update Pattern | `.UpdateCertificate(ctx, id).Certificate(cert).Execute()` |
| Retrieve Pattern | `.RetrieveCertificate(ctx, certificateId).Execute()` |
| List Method | `.ListCertificates(ctx).Execute()` |
| Delete Method | `.DeleteCertificate(ctx, id).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azion-api) - preferred for all implementations
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    
    // Legacy SDKs (azionapi-go-sdk) - deprecated
    digitalCertificatesApi *digital_certificates.APIClient
    // ... more SDK clients
}
```

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by ID)

For reading a single Digital Certificate by its identifier:

**File:** `internal/data_source_digital_certificate.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &CertificateDataSource{}
    _ datasource.DataSourceWithConfigure = &CertificateDataSource{}
)

// Constructor function
func dataSourceAzionDigitalCertificate() datasource.DataSource {
    return &CertificateDataSource{}
}

// DataSource struct - holds the client
type CertificateDataSource struct {
    client *apiClient
}

// Model struct - represents Terraform state
type CertificateDataSourceModel struct {
    ID            types.String             `tfsdk:"id"`
    SchemaVersion types.Int64              `tfsdk:"schema_version"`
    Results       *CertificateResultsModel `tfsdk:"results"`
    CertificateID types.Int64              `tfsdk:"certificate_id"`
}

// Results struct - represents the API response data
type CertificateResultsModel struct {
    ID                 types.Int64    `tfsdk:"id"`
    Name               types.String   `tfsdk:"name"`
    Issuer             types.String   `tfsdk:"issuer"`
    SubjectName        []types.String `tfsdk:"subject_name"`
    Validity           types.String   `tfsdk:"validity"`
    Status             types.String   `tfsdk:"status"`
    StatusDetail       types.String   `tfsdk:"status_detail"`
    Type               types.String   `tfsdk:"certificate_type"`
    Managed            types.Bool     `tfsdk:"managed"`
    CSR                types.String   `tfsdk:"csr"`
    Challenge          types.String   `tfsdk:"challenge"`
    Authority          types.String   `tfsdk:"authority"`
    KeyAlgorithm       types.String   `tfsdk:"key_algorithm"`
    Active             types.Bool     `tfsdk:"active"`
    ProductVersion     types.String   `tfsdk:"product_version"`
    LastEditor         types.String   `tfsdk:"last_editor"`
    LastModified       types.String   `tfsdk:"last_modified"`
    RenewedAt          types.String   `tfsdk:"renewed_at"`
    CertificateContent types.String   `tfsdk:"certificate_content"`
    PrivateKey         types.String   `tfsdk:"private_key"`
}

// Configure - receives the API client
func (c *CertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c.client = req.ProviderData.(*apiClient)
}

// Metadata - sets the data source type name
func (c *CertificateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_digital_certificate"
}

// Schema - defines the Terraform schema
func (c *CertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "certificate_id": schema.Int64Attribute{
                Description: "Identifier of the certificate.",
                Required:    true,
            },
            "schema_version": schema.Int64Attribute{
                Description: "Schema Version.",
                Computed:    true,
            },
            "results": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    // ... define all nested attributes
                },
            },
        },
    }
}

// Read - performs the API call to retrieve the certificate
func (c *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var getCertificateID types.Int64
    diags := req.Config.GetAttribute(ctx, path.Root("certificate_id"), &getCertificateID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Call the V4 API
    certificateResponse, response, err := c.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, getCertificateID.ValueInt64()).Execute()
    if err != nil {
        // Handle 429 rate limiting with retry
        if response.StatusCode == 429 {
            certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
                return c.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, getCertificateID.ValueInt64()).Execute()
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            // Handle other errors
            bodyBytes, _ := io.ReadAll(response.Body)
            resp.Diagnostics.AddError(err.Error(), string(bodyBytes))
            return
        }
    }

    // Transform response to state model
    certificateState := populateCertificateResults(ctx, certificateResponse.GetData(), getCertificateID)
    certificateState.ID = types.StringValue("Get By ID Digital Certificate")
    diags = resp.State.Set(ctx, &certificateState)
    resp.Diagnostics.Append(diags...)
}
```

---

### Plural Data Source (List Multiple Resources)

For listing multiple Digital Certificates:

**File:** `internal/data_source_digital_certificates.go`

```go
package provider

import (
    "context"
    "io"
    "net/http"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Interface assertions
var (
    _ datasource.DataSource              = &DigitalCertificatesDataSource{}
    _ datasource.DataSourceWithConfigure = &DigitalCertificatesDataSource{}
)

// Constructor function
func dataSourceAzionDigitalCertificates() datasource.DataSource {
    return &DigitalCertificatesDataSource{}
}

// DataSource struct
type DigitalCertificatesDataSource struct {
    client *apiClient
}

// Model struct
type DigitalCertificatesDataSourceModel struct {
    ID            types.String           `tfsdk:"id"`
    Counter       types.Int64            `tfsdk:"counter"`
    TotalPages    types.Int64            `tfsdk:"total_pages"`
    Page          types.Int64            `tfsdk:"page"`
    PageSize      types.Int64            `tfsdk:"page_size"`
    Links         *CertificateLinksModel `tfsdk:"links"`
    SchemaVersion types.Int64            `tfsdk:"schema_version"`
    Results       []CertificatesResultModel `tfsdk:"results"`
}

// Links struct - for pagination
type CertificateLinksModel struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

// Result struct - for each certificate in the list
type CertificatesResultModel struct {
    ID             types.Int64    `tfsdk:"id"`
    Name           types.String   `tfsdk:"name"`
    Issuer             types.String   `tfsdk:"issuer"`
    SubjectName        []types.String `tfsdk:"subject_name"`
    Validity           types.String   `tfsdk:"validity"`
    Status         types.String   `tfsdk:"status"`
    StatusDetail   types.String   `tfsdk:"status_detail"`
    Type           types.String   `tfsdk:"certificate_type"`
    Managed        types.Bool     `tfsdk:"managed"`
    Challenge      types.String   `tfsdk:"challenge"`
    Authority      types.String   `tfsdk:"authority"`
    KeyAlgorithm   types.String   `tfsdk:"key_algorithm"`
    Active         types.Bool     `tfsdk:"active"`
    ProductVersion types.String   `tfsdk:"product_version"`
    LastEditor     types.String   `tfsdk:"last_editor"`
    LastModified   types.String   `tfsdk:"last_modified"`
    RenewedAt      types.String   `tfsdk:"renewed_at"`
}

// Read - performs the API call to list certificates
func (d *DigitalCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // Call the V4 API
    certificatesResponse, response, err := d.client.api.DigitalCertificatesCertificatesAPI.ListCertificates(ctx).Execute()
    if err != nil {
        // Handle errors same as singular data source
        if response.StatusCode == 429 {
            certificatesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCertificateList, *http.Response, error) {
                return d.client.api.DigitalCertificatesCertificatesAPI.ListCertificates(ctx).Execute()
            }, 5)
            // ... error handling
        }
    }

    // Transform response to state model
    state := populateCertificatesListResults(ctx, certificatesResponse)
    state.ID = types.StringValue("Get All Digital Certificates")
    diags := resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Feature | Singular (digital_certificate) | Plural (digital_certificates) |
|---------|-------------------------------|------------------------------|
| Input | `certificate_id` (Required) | None (or optional filters) |
| Output | `results` (SingleNestedAttribute) | `results` (ListNestedAttribute) |
| Response Type | `CertificateResponse` | `PaginatedCertificateList` |
| API Method | `RetrieveCertificate(ctx, id)` | `ListCertificates(ctx)` |
| Pagination | No | Yes (counter, total_pages, page, page_size, links) |
| Schema ID | `"Get By ID Digital Certificate"` | `"Get All Digital Certificates"` |

---

## Resource Implementation

For CRUD operations on Digital Certificates:

**File:** `internal/resource_digitalcertificate.go`

### Key Model Types

```go
// Resource model - represents Terraform state
type certificateResourceModel struct {
    SchemaVersion types.Int64              `tfsdk:"schema_version"`
    Results       *certificateResultsModel `tfsdk:"results"`
    ID            types.String             `tfsdk:"id"`
    LastUpdated   types.String             `tfsdk:"last_updated"`
}

// Results model - represents the certificate data
type certificateResultsModel struct {
    ID                 types.Int64    `tfsdk:"id"`
    Name               types.String   `tfsdk:"name"`
    Issuer             types.String   `tfsdk:"issuer"`
    SubjectName        types.List     `tfsdk:"subject_name"`
    Validity           types.String   `tfsdk:"validity"`
    Status             types.String   `tfsdk:"status"`
    StatusDetail       types.String   `tfsdk:"status_detail"`
    Type               types.String   `tfsdk:"certificate_type"`
    Managed            types.Bool     `tfsdk:"managed"`
    CSR                types.String   `tfsdk:"csr"`
    Challenge          types.String   `tfsdk:"challenge"`
    Authority          types.String   `tfsdk:"authority"`
    KeyAlgorithm       types.String   `tfsdk:"key_algorithm"`
    Active             types.Bool     `tfsdk:"active"`
    ProductVersion     types.String   `tfsdk:"product_version"`
    LastEditor         types.String   `tfsdk:"last_editor"`
    LastModified       types.String   `tfsdk:"last_modified"`
    RenewedAt          types.String   `tfsdk:"renewed_at"`
    CertificateContent types.String   `tfsdk:"certificate_content"`
    PrivateKey         types.String   `tfsdk:"private_key"`
}
```

### Create Operation

```go
func (r *certificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan certificateResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the certificate request for V4 API.
    // Note: Certificate and PrivateKey use NullableString type.
    certificateRequest := azionapi.Certificate{
        Name:        plan.Results.Name.ValueString(),
        Certificate: *azionapi.NewNullableString(plan.Results.CertificateContent.ValueStringPointer()),
        PrivateKey:  *azionapi.NewNullableString(plan.Results.PrivateKey.ValueStringPointer()),
    }

    // Call the V4 API.
    certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.CreateCertificate(ctx).Certificate(certificateRequest).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
                return r.client.api.DigitalCertificatesCertificatesAPI.CreateCertificate(ctx).Certificate(certificateRequest).Execute()
            }, 5) // Maximum 5 retries

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
    }

    // Populate the state from the API response.
    cert := certificateResponse.GetData()
    plan.Results = populateCertificateResultsFromAPI(ctx, cert, plan.Results.CertificateContent.ValueString(), plan.Results.PrivateKey.ValueString())
    plan.SchemaVersion = types.Int64Value(1)
    plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Operation

```go
func (r *certificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state certificateResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the certificate ID from state.
    certificateID, err := parseCertificateID(state.ID, state.Results.ID)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", err.Error())
        return
    }

    // Call the V4 API.
    certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, certificateID).Execute()
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle 429 and other errors...
    }

    // Preserve the private key and certificate content from state since API doesn't return them.
    privateKey := state.Results.PrivateKey.ValueString()
    certificateContent := state.Results.CertificateContent.ValueString()

    // Populate the state from the API response.
    cert := certificateResponse.GetData()
    state.Results = populateCertificateResultsFromAPI(ctx, cert, certificateContent, privateKey)
    state.SchemaVersion = types.Int64Value(1)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Update Operation

```go
func (r *certificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan certificateResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state certificateResourceModel
    diags = req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the certificate ID from state.
    certificateID, err := parseCertificateID(state.ID, state.Results.ID)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", err.Error())
        return
    }

    // Build the certificate request for V4 API.
    certificateRequest := azionapi.Certificate{
        Name:        plan.Results.Name.ValueString(),
        Certificate: *azionapi.NewNullableString(plan.Results.CertificateContent.ValueStringPointer()),
        PrivateKey:  *azionapi.NewNullableString(plan.Results.PrivateKey.ValueStringPointer()),
    }

    // Call the V4 API (using PUT for full update).
    certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.UpdateCertificate(ctx, certificateID).Certificate(certificateRequest).Execute()
    if err != nil {
        // Handle errors...
    }

    // Populate the state from the API response.
    cert := certificateResponse.GetData()
    plan.Results = populateCertificateResultsFromAPI(ctx, cert, plan.Results.CertificateContent.ValueString(), plan.Results.PrivateKey.ValueString())
    plan.SchemaVersion = types.Int64Value(1)
    plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Delete Operation

```go
func (r *certificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state certificateResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get the certificate ID from state.
    certificateID, err := parseCertificateID(state.ID, state.Results.ID)
    if err != nil {
        resp.Diagnostics.AddError("Value Conversion error", err.Error())
        return
    }

    // Call the V4 API to delete the certificate.
    _, response, err := r.client.api.DigitalCertificatesCertificatesAPI.DeleteCertificate(ctx, certificateID).Execute()
    if err != nil {
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.DigitalCertificatesCertificatesAPI.DeleteCertificate(ctx, certificateID).Execute()
            }, 5) // Maximum 5 retries

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
    }
}
```

### Helper Functions

```go
// parseCertificateID extracts the certificate ID from either the string ID or the int64 ID.
func parseCertificateID(stringID types.String, int64ID types.Int64) (int64, error) {
    if !stringID.IsNull() && !stringID.IsUnknown() {
        var id int64
        _, err := fmt.Sscanf(stringID.ValueString(), "%d", &id)
        if err != nil {
            return 0, fmt.Errorf("could not parse certificate ID: %w", err)
        }
        return id, nil
    }
    if !int64ID.IsNull() && !int64ID.IsUnknown() {
        return int64ID.ValueInt64(), nil
    }
    return 0, fmt.Errorf("no valid certificate ID found in state")
}

// populateCertificateResultsFromAPI transforms API response data to Terraform state model.
func populateCertificateResultsFromAPI(ctx context.Context, cert azionapi.Certificate, certificateContent, privateKey string) *certificateResultsModel {
    // Convert subject names to types.List.
    var subjectNameList types.List
    subjectNames := cert.GetSubjectName()
    if len(subjectNames) > 0 {
        subjectNameList, _ = types.ListValueFrom(ctx, types.StringType, subjectNames)
    } else {
        subjectNameList = types.ListNull(types.StringType)
    }

    var renewedAt string
    if cert.RenewedAt.IsSet() && cert.RenewedAt.Get() != nil {
        renewedAt = (*cert.RenewedAt.Get()).Format(time.RFC3339)
    }

    result := &certificateResultsModel{
        ID:                 types.Int64Value(cert.GetId()),
        Name:               types.StringValue(cert.GetName()),
        Issuer:             types.StringValue(cert.GetIssuer()),
        SubjectName:        subjectNameList,
        Validity:           types.StringValue(cert.GetValidity()),
        Status:             types.StringValue(cert.GetStatus()),
        StatusDetail:       types.StringValue(cert.GetStatusDetail()),
        Type:               types.StringValue(cert.GetType()),
        Managed:            types.BoolValue(cert.GetManaged()),
        CSR:                types.StringValue(cert.GetCsr()),
        Challenge:          types.StringValue(cert.GetChallenge()),
        Authority:          types.StringValue(cert.GetAuthority()),
        KeyAlgorithm:       types.StringValue(cert.GetKeyAlgorithm()),
        ProductVersion:     types.StringValue(cert.GetProductVersion()),
        LastEditor:         types.StringValue(cert.GetLastEditor()),
        LastModified:       types.StringValue(cert.GetLastModified().Format(time.RFC3339)),
        RenewedAt:          types.StringValue(renewedAt),
        CertificateContent: types.StringValue(certificateContent),
        PrivateKey:         types.StringValue(privateKey),
    }

    // Handle optional fields.
    if cert.Active != nil {
        result.Active = types.BoolValue(*cert.Active)
    }

    return result
}
```

---

## Schema Definition Patterns

### Required vs Optional vs Computed

| Attribute | Data Source (Singular) | Data Source (Plural) | Resource |
|-----------|----------------------|---------------------|----------|
| `id` | Computed | Computed | Computed |
| `certificate_id` | Required | N/A | N/A |
| `name` | Computed | Computed | Required |
| `certificate_content` | Computed | N/A | Required, Sensitive |
| `private_key` | Computed | N/A | Required, Sensitive |
| `managed` | Computed | Computed | Computed |

### Field Types

```go
// Simple types
Name:         schema.StringAttribute{}
ID:           schema.Int64Attribute{}
Managed:      schema.BoolAttribute{}

// Sensitive types
CertificateContent: schema.StringAttribute{
    Description: "The content of the certificate (PEM format).",
    Required:    true,
    Sensitive:   true,
}

// List types
SubjectName: schema.ListAttribute{
    Description: "Subject name of the certificate.",
    Computed:    true,
    ElementType: types.StringType,
}

// Nested object types
Results: schema.SingleNestedAttribute{
    Description: "The certificate details.",
    Required:    true,
    Attributes: map[string]schema.Attribute{
        // nested attributes
    },
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
```

---

## Type Conversions

### NullableString Handling

The V4 SDK uses `NullableString` for optional string fields like `Certificate` and `PrivateKey`:

```go
// Creating a NullableString from a pointer
cert := azionapi.Certificate{
    Name:        plan.Name.ValueString(),
    Certificate: *azionapi.NewNullableString(plan.CertificateContent.ValueStringPointer()),
    PrivateKey:  *azionapi.NewNullableString(plan.PrivateKey.ValueStringPointer()),
}

// Reading a NullableString
if cert.HasCertificate() {
    result.CertificateContent = types.StringValue(cert.GetCertificate())
}
```

### Time Formatting

```go
// From API response
lastModified := types.StringValue(response.Data.GetLastModified().Format(time.RFC3339))

// Handle nullable time
var renewedAt string
if cert.RenewedAt.IsSet() && cert.RenewedAt.Get() != nil {
    renewedAt = (*cert.RenewedAt.Get()).Format(time.RFC3339)
}
```

### SubjectName List Conversion

```go
// Convert subject names to types.List.
// Note: Use types.List instead of []types.String to handle unknown values during planning.
var subjectNameList types.List
subjectNames := cert.GetSubjectName()
if len(subjectNames) > 0 {
    subjectNameList, _ = types.ListValueFrom(ctx, types.StringType, subjectNames)
} else {
    subjectNameList = types.ListNull(types.StringType)
}
result.SubjectName = subjectNameList
```

---

## Common Issues

### 1. Using Legacy SDK Instead of V4

**Problem:** Code imports `digital_certificates` from legacy SDK.

**Solution:** Use `azion-api` package:

```go
// Wrong
import "github.com/aziontech/azionapi-go-sdk/digital_certificates"

// Correct
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### 2. Using "edge" Prefix in Variable Names

**Problem:** Variable names include "edge" prefix from legacy naming.

**Solution:** Use cleaner naming without "edge":

```go
// Wrong
edgeCertificateID
edgeCertificatesApi

// Correct
certificateID
api.DigitalCertificatesCertificatesAPI
```

### 3. Not Using NullableString for Certificate/PrivateKey

**Problem:** Using regular string pointer for `Certificate` and `PrivateKey` fields.

**Solution:** Use `NullableString` type:

```go
// Wrong
Certificate: plan.CertificateContent.ValueStringPointer()

// Correct
Certificate: *azionapi.NewNullableString(plan.CertificateContent.ValueStringPointer())
```

### 4. Not Handling Nullable Time Fields

**Problem:** `RenewedAt` field can be null, causing panics.

**Solution:** Check if the field is set:

```go
var renewedAt string
if cert.RenewedAt.IsSet() && cert.RenewedAt.Get() != nil {
    renewedAt = (*cert.RenewedAt.Get()).Format(time.RFC3339)
}
result.RenewedAt = types.StringValue(renewedAt)
```

### 5. Missing Response Body Closure

**Problem:** HTTP response body not closed, causing resource leaks.

**Solution:** Always close the response body after retry:

```go
if response != nil {
    defer response.Body.Close()
}
```

### 6. Wrong Delete Method Name

**Problem:** Using `DestroyCertificate` instead of `DeleteCertificate`.

**Solution:** Use the correct method name:

```go
// Wrong
r.client.api.DigitalCertificatesCertificatesAPI.DestroyCertificate(ctx, id)

// Correct
r.client.api.DigitalCertificatesCertificatesAPI.DeleteCertificate(ctx, id)
```

### 7. Not Returning Private Key/Certificate Content from State

**Problem:** API doesn't return private key and certificate content on Read operations.

**Solution:** Preserve these values from state:

```go
// Preserve the private key and certificate content from state since API doesn't return them.
privateKey := state.Results.PrivateKey.ValueString()
certificateContent := state.Results.CertificateContent.ValueString()

// Then pass them to populateCertificateResultsFromAPI
state.Results = populateCertificateResultsFromAPI(ctx, cert, certificateContent, privateKey)
```

---

## Summary Checklist

When generating or updating Digital Certificate data sources:

1. **Use V4 SDK**: Import from `github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api`
2. **Use correct naming**: Avoid "edge" prefix in variable names
3. **Use correct API client**: `client.api.DigitalCertificatesCertificatesAPI`
4. **Handle 429 errors**: Use `utils.RetryOn429`
5. **Handle nullable fields**: Check `IsSet()` and `Get()` for nullable types
6. **Use NullableString**: For `Certificate` and `PrivateKey` fields
7. **Close response bodies**: Add `defer response.Body.Close()` after retries
8. **Convert time fields**: Use `time.RFC3339` format
9. **Handle ID types**: Use `int64` for all operations
10. **Preserve sensitive fields**: Keep `private_key` and `certificate_content` from state on Read
11. **Register in provider.go**: Ensure data sources are registered

---

## API Reference

### Certificate Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `int64` | Unique identifier |
| `name` | `string` | Certificate name |
| `certificate` | `NullableString` | Certificate content (PEM format) |
| `private_key` | `NullableString` | Private key content (PEM format) |
| `issuer` | `NullableString` | Certificate issuer |
| `subject_name` | `[]string` | List of subject names |
| `validity` | `NullableString` | Validity period |
| `type` | `*string` | Certificate type (certificate, trusted_ca_certificate) |
| `managed` | `bool` | Whether managed by Azion |
| `status` | `string` | Status (active, pending, failed, challenge_verification) |
| `status_detail` | `string` | Detailed status information |
| `csr` | `NullableString` | Certificate Signing Request |
| `challenge` | `string` | Challenge type (dns, http) |
| `authority` | `string` | Certificate authority (lets_encrypt) |
| `key_algorithm` | `string` | Key algorithm |
| `active` | `*bool` | Whether certificate is active |
| `product_version` | `string` | Product version |
| `last_editor` | `string` | Last editor |
| `last_modified` | `time.Time` | Last modification timestamp |
| `renewed_at` | `NullableTime` | Renewal timestamp |

### API Endpoints

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List | `ListCertificates(ctx)` | `GET /workspace/tls/certificates` |
| Retrieve | `RetrieveCertificate(ctx, id)` | `GET /workspace/tls/certificates/{id}` |
| Create | `CreateCertificate(ctx).Certificate(cert)` | `POST /workspace/tls/certificates` |
| Update | `UpdateCertificate(ctx, id).Certificate(cert)` | `PUT /workspace/tls/certificates/{id}` |
| Delete | `DeleteCertificate(ctx, id)` | `DELETE /workspace/tls/certificates/{id}` |
