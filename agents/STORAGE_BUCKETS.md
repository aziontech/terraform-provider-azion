# Storage Buckets - Code Generation Guide

This document provides specific guidance for implementing Storage Bucket resources and data sources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source (Read by Name)](#singular-data-source-read-by-name)
   - [Plural Data Source (List Multiple Resources)](#plural-data-source-list-multiple-resources)
   - [Key Differences: Singular vs Plural Data Sources](#key-differences-singular-vs-plural-data-sources)
3. [Schema Definition Patterns](#schema-definition-patterns)
4. [Error Handling](#error-handling)
5. [Type Conversions](#type-conversions)
6. [Common Issues](#common-issues)

---

## SDK Selection

Storage Buckets use the **V4 SDK (`azion-api`)** for all resources and data sources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Bucket (Singular Data Source) | `azion-api` (v4) | `api.StorageBucketsAPI` | `https://api.azion.com/v4` |
| Buckets (Plural Data Source) | `azion-api` (v4) | `api.StorageBucketsAPI` | `https://api.azion.com/v4` |
| Bucket (Resource) | `azion-api` (v4) | `api.StorageBucketsAPI` | `https://api.azion.com/v4` |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `string` (bucket name) |
| Create Request Type | `BucketCreateRequest` |
| Update Request Type | `PatchedBucketRequest` |
| Response Type | `BucketCreateResponse` with `Data` field |
| List Response Type | `PaginatedBucketList` |
| Create Pattern | `.CreateBucket(ctx).BucketCreateRequest(req).Execute()` |
| Update Pattern | `.UpdateBucket(ctx, bucketName).PatchedBucketRequest(req).Execute()` |
| Retrieve Pattern | `.RetrieveBucket(ctx, bucketName).Execute()` |
| List Method | `.ListBuckets(ctx).Execute()` |
| Delete Method | `.DeleteBucket(ctx, bucketName).Execute()` |

### Import Statement

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

---

## Data Source Implementation

### Singular Data Source (Read by Name)

For reading a single Bucket by its name:

**File:** `internal/data_source_bucket.go`

```go
package provider

import (
    "context"
    "fmt"
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
    _ datasource.DataSource              = &BucketDataSource{}
    _ datasource.DataSourceWithConfigure = &BucketDataSource{}
)

func dataSourceAzionBucket() datasource.DataSource {
    return &BucketDataSource{}
}

type BucketDataSource struct {
    client *apiClient
}

type BucketDataSourceModel struct {
    Name types.String        `tfsdk:"name"`
    Data BucketResultsModel  `tfsdk:"data"`
    ID   types.String        `tfsdk:"id"`
}

type BucketResultsModel struct {
    Name            types.String `tfsdk:"name"`
    WorkloadsAccess types.String `tfsdk:"workloads_access"`
    LastEditor      types.String `tfsdk:"last_editor"`
    LastModified    types.String `tfsdk:"last_modified"`
    ProductVersion  types.String `tfsdk:"product_version"`
}
```

#### Key Points for Singular Data Source

1. **Uses `name` as the identifier**: Buckets are identified by their name (string), not a numeric ID
2. **Required attribute**: `name` is a required attribute for reading a specific bucket
3. **API Call**: Uses `RetrieveBucket(ctx, bucketName).Execute()`
4. **Response type**: Returns `BucketCreateResponse` with a `Data` field containing bucket details

### Plural Data Source (List Multiple Resources)

For listing multiple Buckets:

**File:** `internal/data_source_buckets.go`

```go
package provider

import (
    "context"
    "fmt"
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
    _ datasource.DataSource              = &BucketsDataSource{}
    _ datasource.DataSourceWithConfigure = &BucketsDataSource{}
)

func dataSourceAzionBuckets() datasource.DataSource {
    return &BucketsDataSource{}
}

type BucketsDataSource struct {
    client *apiClient
}

type BucketsDataSourceModel struct {
    Counter    types.Int64            `tfsdk:"counter"`
    TotalPages types.Int64            `tfsdk:"total_pages"`
    Page       types.Int64            `tfsdk:"page"`
    PageSize   types.Int64            `tfsdk:"page_size"`
    Results    []BucketsResultsModel  `tfsdk:"results"`
    ID         types.String           `tfsdk:"id"`
}

type BucketsResultsModel struct {
    Name            types.String `tfsdk:"name"`
    WorkloadsAccess types.String `tfsdk:"workloads_access"`
    LastEditor      types.String `tfsdk:"last_editor"`
    LastModified    types.String `tfsdk:"last_modified"`
    ProductVersion  types.String `tfsdk:"product_version"`
}
```

#### Key Points for Plural Data Source

1. **No required attributes**: Lists all buckets by default
2. **API Call**: Uses `ListBuckets(ctx).Execute()`
3. **Response type**: Returns `PaginatedBucketList` with pagination fields and a `Results` array
4. **Pagination fields**: `counter`, `total_pages`, `page`, `page_size`

### Key Differences: Singular vs Plural Data Sources

| Aspect | Singular (`azion_bucket`) | Plural (`azion_buckets`) |
|--------|---------------------------|--------------------------|
| **Purpose** | Read a specific bucket | List all buckets |
| **Required Attribute** | `name` (string) | None |
| **API Method** | `RetrieveBucket(ctx, name)` | `ListBuckets(ctx)` |
| **Response Type** | `BucketCreateResponse` | `PaginatedBucketList` |
| **Data Structure** | `SingleNestedAttribute` | `ListNestedAttribute` |
| **ID Field** | Set to bucket name | Set to "buckets" |

---

## Schema Definition Patterns

### Singular Data Source Schema

```go
func (d *BucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "name": schema.StringAttribute{
                Description: "Name of the bucket to retrieve.",
                Required:    true,
            },
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "data": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "name": schema.StringAttribute{
                        Description: "Name of the bucket.",
                        Computed:    true,
                    },
                    "workloads_access": schema.StringAttribute{
                        Description: "Access type for workloads: read_only, read_write, or restricted.",
                        Computed:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "Last editor of the bucket.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the bucket.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version of the bucket.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}
```

### Plural Data Source Schema

```go
func (d *BucketsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Identifier of the data source.",
                Computed:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total count of buckets.",
                Computed:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
                Computed:    true,
            },
            "page": schema.Int64Attribute{
                Description: "The current page number.",
                Computed:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The number of items per page.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        // Same as singular data attributes
                    },
                },
            },
        },
    }
}
```

---

## Error Handling

### Standard Error Handling Pattern

```go
func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var bucketName types.String
    diags := req.Config.GetAttribute(ctx, path.Root("name"), &bucketName)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    bucketResponse, response, err := d.client.api.StorageBucketsAPI.
        RetrieveBucket(ctx, bucketName.ValueString()).
        Execute() //nolint
    if err != nil {
        // Handle rate limiting (429)
        if response.StatusCode == 429 {
            bucketResponse, response, err = utils.RetryOn429(func() (*azionapi.BucketCreateResponse, *http.Response, error) {
                return d.client.api.StorageBucketsAPI.RetrieveBucket(ctx, bucketName.ValueString()).Execute() //nolint
            }, 5) // Maximum 5 retries

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            usrMsg, errMsg := errPrintBucket(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }
    
    // ... populate state
}
```

### Error Message Function

```go
func errPrintBucket(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "Bucket not found"
    case 403:
        usrMsg = "Forbidden"
    case 405:
        usrMsg = "Method Not Allowed"
    case 406:
        usrMsg = "Not Acceptable"
    default:
        usrMsg = err.Error()
    }
    return usrMsg, fmt.Sprintf("%d - %s", errCode, usrMsg)
}
```

---

## Type Conversions

### Time Formatting

The `last_modified` field from the API is a `time.Time` type. Convert it to RFC3339 string:

```go
LastModified: types.StringValue(bucketResponse.Data.LastModified.Format(time.RFC3339)),
```

### String Values

For simple string fields:

```go
Name:            types.StringValue(bucketResponse.Data.Name),
WorkloadsAccess: types.StringValue(bucketResponse.Data.WorkloadsAccess),
LastEditor:      types.StringValue(bucketResponse.Data.LastEditor),
ProductVersion:  types.StringValue(bucketResponse.Data.ProductVersion),
```

### Handling Nullable Fields

For optional fields that may be nil in the response:

```go
if bucketsResponse.Count != nil {
    bucketsState.Counter = types.Int64Value(*bucketsResponse.Count)
}

if bucketsResponse.TotalPages != nil {
    bucketsState.TotalPages = types.Int64Value(*bucketsResponse.TotalPages)
}
```

---

## Common Issues

### Issue 1: Using Wrong SDK Client

**Problem**: Using `edgeApi` instead of `api` for V4 resources.

**Solution**: Always use `d.client.api.StorageBucketsAPI` for bucket operations.

```go
// Correct
d.client.api.StorageBucketsAPI.RetrieveBucket(ctx, name).Execute()

// Incorrect
d.client.edgeApi.StorageBucketsAPI.RetrieveBucket(ctx, name).Execute()
```

### Issue 2: Not Closing Response Body

**Problem**: Forgetting to close HTTP response body causes resource leaks.

**Solution**: Always close the response body after successful API calls.

```go
if response != nil {
    defer response.Body.Close()
}
```

### Issue 3: Missing nolint Comment

**Problem**: The `Execute()` method may trigger a linting warning.

**Solution**: Add `//nolint` comment after Execute().

```go
bucketResponse, response, err := d.client.api.StorageBucketsAPI.
    RetrieveBucket(ctx, bucketName.ValueString()).
    Execute() //nolint
```

### Issue 4: String vs Int64 ID

**Problem**: Buckets use string names as identifiers, not numeric IDs like other resources.

**Solution**: Use `types.String` for the ID field and `schema.StringAttribute` for the identifier.

```go
// For buckets (string ID)
type BucketDataSourceModel struct {
    Name types.String `tfsdk:"name"`
    ID   types.String `tfsdk:"id"`
}

// For other resources (int64 ID)
type WorkloadDataSourceModel struct {
    ID   types.String    `tfsdk:"id"`  // String in state
    Data WorkloadResults `tfsdk:"data"`
}

// In the model, ID is int64
type WorkloadResults struct {
    ID types.Int64 `tfsdk:"id"`
}
```

### Issue 5: Not Handling Pagination in List Response

**Problem**: The `PaginatedBucketList` response includes pagination fields that should be exposed.

**Solution**: Always map pagination fields from the response.

```go
if bucketsResponse.Count != nil {
    bucketsState.Counter = types.Int64Value(*bucketsResponse.Count)
}
if bucketsResponse.TotalPages != nil {
    bucketsState.TotalPages = types.Int64Value(*bucketsResponse.TotalPages)
}
if bucketsResponse.Page != nil {
    bucketsState.Page = types.Int64Value(*bucketsResponse.Page)
}
if bucketsResponse.PageSize != nil {
    bucketsState.PageSize = types.Int64Value(*bucketsResponse.PageSize)
}
```

---

## API Response Structures

### BucketCreateResponse (RetrieveBucket)

```go
type BucketCreateResponse struct {
    State *string      `json:"state,omitempty"`
    Data  BucketCreate `json:"data"`
}

type BucketCreate struct {
    Name            string    `json:"name"`
    WorkloadsAccess string    `json:"workloads_access"`
    LastEditor      string    `json:"last_editor"`
    LastModified    time.Time `json:"last_modified"`
    ProductVersion  string    `json:"product_version"`
}
```

### PaginatedBucketList (ListBuckets)

```go
type PaginatedBucketList struct {
    Count      *int64   `json:"count,omitempty"`
    TotalPages *int64   `json:"total_pages,omitempty"`
    Page       *int64   `json:"page,omitempty"`
    PageSize   *int64   `json:"page_size,omitempty"`
    Next       *string  `json:"next,omitempty"`
    Previous   *string  `json:"previous,omitempty"`
    Results    []Bucket `json:"results,omitempty"`
}

type Bucket struct {
    Name            string    `json:"name"`
    WorkloadsAccess string    `json:"workloads_access"`
    LastEditor      string    `json:"last_editor"`
    LastModified    time.Time `json:"last_modified"`
    ProductVersion  string    `json:"product_version"`
}
```

---

## Resource Implementation

### Resource File Structure

**File:** `internal/resource_bucket.go`

### Resource Model

```go
type bucketResourceModel struct {
    Bucket      *bucketResourceResults `tfsdk:"bucket"`
    ID          types.String           `tfsdk:"id"`
    LastUpdated types.String           `tfsdk:"last_updated"`
}

type bucketResourceResults struct {
    Name            types.String `tfsdk:"name"`
    WorkloadsAccess types.String `tfsdk:"workloads_access"`
    LastEditor      types.String `tfsdk:"last_editor"`
    LastModified    types.String `tfsdk:"last_modified"`
    ProductVersion  types.String `tfsdk:"product_version"`
}
```

### Resource Schema

```go
func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Resource for managing Azion Storage Buckets.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "last_updated": schema.StringAttribute{
                Description: "Timestamp of the last Terraform update of the resource.",
                Computed:    true,
            },
            "bucket": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "name": schema.StringAttribute{
                        Description: "Name of the bucket. This field is immutable and cannot be updated after creation.",
                        Required:    true,
                        PlanModifiers: []planmodifier.String{
                            stringplanmodifier.RequiresReplace(),
                        },
                    },
                    "workloads_access": schema.StringAttribute{
                        Description: "Access type for workloads: read_only, read_write, or restricted.",
                        Required:    true,
                    },
                    "last_editor": schema.StringAttribute{
                        Description: "The last editor of the bucket.",
                        Computed:    true,
                    },
                    "last_modified": schema.StringAttribute{
                        Description: "Last modified timestamp of the bucket.",
                        Computed:    true,
                    },
                    "product_version": schema.StringAttribute{
                        Description: "Product version of the bucket.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}
```

### Key Points for Resource

1. **Bucket name is immutable**: The `name` field uses `stringplanmodifier.RequiresReplace()` to trigger resource replacement if changed
2. **String identifier**: Unlike other resources that use numeric IDs, buckets use their name as the identifier
3. **Create API**: Uses `CreateBucket(ctx).BucketCreateRequest(req).Execute()`
4. **Read API**: Uses `RetrieveBucket(ctx, bucketName).Execute()`
5. **Update API**: Uses `UpdateBucket(ctx, bucketName).PatchedBucketRequest(req).Execute()`
6. **Delete API**: Uses `DeleteBucket(ctx, bucketName).Execute()`

### Create Operation

```go
func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan bucketResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    bucket := azionapi.NewBucketCreateRequest(
        plan.Bucket.Name.ValueString(),
        plan.Bucket.WorkloadsAccess.ValueString(),
    )

    createBucket, response, err := r.client.api.StorageBucketsAPI.
        CreateBucket(ctx).
        BucketCreateRequest(*bucket).
        Execute() //nolint
    if err != nil {
        // Handle 429 rate limiting and other errors
        // ...
    }
    if response != nil {
        defer response.Body.Close()
    }

    plan.Bucket = populateBucketResults(createBucket)
    plan.ID = types.StringValue(createBucket.Data.Name)
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Operation

```go
func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state bucketResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var bucketName string
    if state.Bucket != nil {
        bucketName = state.Bucket.Name.ValueString()
    } else {
        bucketName = state.ID.ValueString()
    }

    getBucket, response, err := r.client.api.StorageBucketsAPI.
        RetrieveBucket(ctx, bucketName).
        Execute() //nolint
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle other errors...
    }
    if response != nil {
        defer response.Body.Close()
    }

    state.Bucket = populateBucketResults(getBucket)
    state.ID = types.StringValue(getBucket.Data.Name)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Update Operation

The update operation uses PATCH semantics - only the `workloads_access` field can be updated:

```go
func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan bucketResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state bucketResourceModel
    diagsState := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diagsState...)
    if resp.Diagnostics.HasError() {
        return
    }

    bucketName := state.Bucket.Name.ValueString()
    updateBucketRequest := azionapi.NewPatchedBucketRequest()

    if !plan.Bucket.WorkloadsAccess.IsNull() && !plan.Bucket.WorkloadsAccess.IsUnknown() {
        updateBucketRequest.SetWorkloadsAccess(plan.Bucket.WorkloadsAccess.ValueString())
    }

    updateBucket, response, err := r.client.api.StorageBucketsAPI.
        UpdateBucket(ctx, bucketName).
        PatchedBucketRequest(*updateBucketRequest).
        Execute() //nolint
    // Handle errors and populate state...
}
```

### Delete Operation

```go
func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state bucketResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    bucketName := state.Bucket.Name.ValueString()

    _, response, err := r.client.api.StorageBucketsAPI.
        DeleteBucket(ctx, bucketName).
        Execute() //nolint
    // Handle errors...
}
```

### Import State

Buckets can be imported using their name:

```go
func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

Example import command:
```bash
terraform import azion_bucket.example my-bucket-name
```

### SDK Request/Response Types

| Operation | Request Type | Response Type |
|-----------|-------------|---------------|
| Create | `BucketCreateRequest` | `BucketCreateResponse` |
| Read | N/A | `BucketCreateResponse` |
| Update | `PatchedBucketRequest` | `BucketCreateResponse` |
| Delete | N/A | `DeleteResponse` |

### BucketCreateRequest

```go
type BucketCreateRequest struct {
    Name            string `json:"name"`
    WorkloadsAccess string `json:"workloads_access"`
}

// Constructor
azionapi.NewBucketCreateRequest(name, workloadsAccess string) *BucketCreateRequest
```

### PatchedBucketRequest

```go
type PatchedBucketRequest struct {
    WorkloadsAccess *string `json:"workloads_access,omitempty"`
}

// Constructor
azionapi.NewPatchedBucketRequest() *PatchedBucketRequest

// Setter
updateBucketRequest.SetWorkloadsAccess(value string)
```

---

## Registration in provider.go

Add the data sources and resource to the provider:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionBucket,
        dataSourceAzionBuckets,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewBucketResource,
        // ... other resources
    }
}
```

---

## Summary Checklist

When implementing Storage Bucket data sources and resources:

- [x] Use V4 SDK (`azion-api`) via `client.api.StorageBucketsAPI`
- [x] Buckets use string names as identifiers (not numeric IDs)
- [x] Handle 429 errors with `utils.RetryOn429`
- [x] Close response body with `defer response.Body.Close()`
- [x] Add `//nolint` comment after `Execute()` calls
- [x] Map all fields from API response to Terraform state
- [x] Convert `time.Time` to RFC3339 string format
- [x] Register data sources in `provider.go`
- [x] Register resource in `provider.go`
- [x] Create documentation in `docs/data-sources/` and `docs/resources/`
- [x] Create examples in `examples/data-sources/` and `examples/resources/`
- [x] Run linters: `golangci-lint run --config .golintci.yml ./internal/...`

### Resource-Specific Checklist

- [x] Name field uses `RequiresReplace()` plan modifier (immutable)
- [x] Implement Create, Read, Update, Delete, and ImportState methods
- [x] Use `NewBucketCreateRequest` for creating buckets
- [x] Use `NewPatchedBucketRequest` for updating buckets
- [x] Handle 404 in Read to remove resource from state
- [x] Use bucket name as the ID for import
