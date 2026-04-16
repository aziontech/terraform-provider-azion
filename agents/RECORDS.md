# DNS Records - Code Generation Guide

This document provides specific guidance for implementing DNS Records data sources and resources in the Terraform provider.

## Table of Contents

1. [SDK Selection](#sdk-selection)
2. [Data Source Implementation](#data-source-implementation)
3. [Resource Implementation](#resource-implementation)
4. [Schema Definition Patterns](#schema-definition-patterns)
5. [Error Handling](#error-handling)
6. [Type Conversions](#type-conversions)
7. [Common Issues](#common-issues)

---

## SDK Selection

DNS Records use the **V4 SDK (`azion-api`)** for all data sources and resources:

| Resource | SDK Package | Client Field | Base URL |
|----------|-------------|--------------|----------|
| Records (Data Source) | `azion-api` (v4) | `api.DNSRecordsAPI` | `https://api.azion.com/v4` |
| Record (Resource) | `azion-api` (v4) | `api.DNSRecordsAPI` | `https://api.azion.com/v4` |

### Important: SDK Import Path

**The V4 SDK import path is:**

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### Important: Naming Convention

**The "edge" prefix is NOT used for DNS records.**

When implementing DNS records data sources and resources:
- Use naming without the `edge` prefix for variables, structs, and function parameters
- The Terraform data source name uses `intelligent_dns_records` (following the Intelligent DNS naming pattern)
- The Terraform resource name uses `intelligent_dns_record`
- Internal Go code naming follows this convention

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Response Type | `PaginatedRecordList` |
| Record Type | `Record` |
| List Method | `.ListDnsRecords(ctx, zoneId).Page(page).PageSize(pageSize).Execute()` |
| Retrieve Method | `.RetrieveDnsRecord(ctx, recordId, zoneId).Execute()` |
| Create Method | `.CreateDnsRecord(ctx, zoneId).RecordRequest(req).Execute()` |
| Update Method | `.UpdateDnsRecord(ctx, recordId, zoneId).RecordRequest(req).Execute()` |
| Delete Method | `.DeleteDnsRecord(ctx, recordId, zoneId).Execute()` |

### Client Configuration

```go
// internal/config.go
type apiClient struct {
    // V4 SDK (azion-api) - used for DNS records
    apiConfig *azionapi.Configuration
    api       *azionapi.APIClient
    // ... other SDK clients
}
```

---

## Data Source Implementation

### File Structure

**File:** `internal/data_source_records.go`

### Data Source Model

The records data source uses a paginated list structure:

```go
type RecordsDataSourceModel struct {
    ZoneId     types.Int64              `tfsdk:"zone_id"`
    TotalPages types.Int64              `tfsdk:"total_pages"`
    Page       types.Int64              `tfsdk:"page"`
    PageSize   types.Int64              `tfsdk:"page_size"`
    Counter    types.Int64              `tfsdk:"counter"`
    Links      *RecordsResponseLinks    `tfsdk:"links"`
    Results    []RecordDataSourceResult `tfsdk:"results"`
    Id         types.String             `tfsdk:"id"`
}

type RecordsResponseLinks struct {
    Previous types.String `tfsdk:"previous"`
    Next     types.String `tfsdk:"next"`
}

type RecordDataSourceResult struct {
    RecordId    types.Int64    `tfsdk:"record_id"`
    Name        types.String   `tfsdk:"name"`
    Description types.String   `tfsdk:"description"`
    Rdata       []types.String `tfsdk:"rdata"`
    Policy      types.String   `tfsdk:"policy"`
    Type        types.String   `tfsdk:"type"`
    Ttl         types.Int64    `tfsdk:"ttl"`
    Weight      types.Int64    `tfsdk:"weight"`
}
```

### Read Operation

```go
func (d *RecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var page types.Int64
    var pageSize types.Int64
    var zoneId types.Int64

    // Get pagination parameters.
    diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
    resp.Diagnostics.Append(diagsPage...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
    resp.Diagnostics.Append(diagsPageSize...)
    if resp.Diagnostics.HasError() {
        return
    }

    diagsZoneId := req.Config.GetAttribute(ctx, path.Root("zone_id"), &zoneId)
    resp.Diagnostics.Append(diagsZoneId...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Set default values for pagination.
    if page.IsNull() || page.IsUnknown() {
        page = types.Int64Value(1)
    }
    if pageSize.IsNull() || pageSize.IsUnknown() {
        pageSize = types.Int64Value(10)
    }

    // Build the API request.
    listRequest := d.client.api.DNSRecordsAPI.ListDnsRecords(ctx, zoneId.ValueInt64()).
        Page(page.ValueInt64()).
        PageSize(pageSize.ValueInt64())

    // Execute the request.
    recordsResponse, httpResp, err := listRequest.Execute()
    if err != nil {
        // Handle 429 (rate limiting) with retry.
        if httpResp.StatusCode == 429 {
            recordsResponse, httpResp, err = utils.RetryOn429(func() (*azionapi.PaginatedRecordList, *http.Response, error) {
                return d.client.api.DNSRecordsAPI.ListDnsRecords(ctx, zoneId.ValueInt64()).
                    Page(page.ValueInt64()).
                    PageSize(pageSize.ValueInt64()).
                    Execute()
            }, 5)

            if httpResp != nil {
                defer httpResp.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(
                    err.Error(),
                    "API request failed after too many retries",
                )
                return
            }
        } else {
            usrMsg, errMsg := errPrintRecords(httpResp.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    if httpResp != nil {
        defer httpResp.Body.Close()
    }

    // Build the state from the response.
    recordsState := buildRecordsState(zoneId, page, pageSize, recordsResponse)

    // Set the state.
    diags := resp.State.Set(ctx, &recordsState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

---

## Resource Implementation

### File Structure

**File:** `internal/resource_record.go`

### Resource Model

The record resource uses a nested structure for the record configuration:

```go
type recordResourceModel struct {
    ZoneId      types.String  `tfsdk:"zone_id"`
    Record      *recordModel  `tfsdk:"record"`
    LastUpdated types.String  `tfsdk:"last_updated"`
}

type recordModel struct {
    Id          types.Int64    `tfsdk:"id"`
    Rdata       []types.String `tfsdk:"rdata"`
    Type        types.String   `tfsdk:"type"`
    Ttl         types.Int64    `tfsdk:"ttl"`
    Policy      types.String   `tfsdk:"policy"`
    Name        types.String   `tfsdk:"name"`
    Weight      types.Int64    `tfsdk:"weight"`
    Description types.String   `tfsdk:"description"`
}
```

### Create Operation

```go
func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan recordResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error ",
            "Could not convert Zone ID",
        )
        return
    }

    // Build the record request using SDK constructor.
    recordReq := azionapi.NewRecordRequest(
        plan.Record.Name.ValueString(),
        plan.Record.Type.ValueString(),
        buildRdataList(plan.Record.Rdata),
    )

    // Set TTL.
    recordReq.SetTtl(plan.Record.Ttl.ValueInt64())

    // Set policy.
    recordReq.SetPolicy(plan.Record.Policy.ValueString())

    // Set weight and description for weighted policy.
    if plan.Record.Policy.ValueString() == "weighted" {
        if !plan.Record.Weight.IsNull() && !plan.Record.Weight.IsUnknown() {
            recordReq.SetWeight(plan.Record.Weight.ValueInt64())
        }
        if !plan.Record.Description.IsNull() && !plan.Record.Description.IsUnknown() {
            recordReq.SetDescription(plan.Record.Description.ValueString())
        }
    }

    // Execute create request.
    createRecord, httpResponse, err := r.client.api.DNSRecordsAPI.CreateDnsRecord(ctx, zoneId).
        RecordRequest(*recordReq).Execute()
    if err != nil {
        // Handle 429 with retry...
    }

    if httpResponse != nil {
        defer httpResponse.Body.Close()
    }

    // Update plan with response.
    plan.Record = populateRecordModel(createRecord.GetData(), plan.Record.Policy.ValueString())
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}
```

### Read Operation

The Read operation uses `RetrieveDnsRecord` to fetch a single record by ID:

```go
func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state recordResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse zone_id and record_id from state.
    // Format: "zone_id/record_id" for import, or just "zone_id" for existing state.
    valueFromCmd := strings.Split(state.ZoneId.ValueString(), "/")
    zoneId, err := strconv.ParseInt(valueFromCmd[0], 10, 64)
    // ... error handling

    var recordId int64
    if len(valueFromCmd) > 1 {
        recordId, err = strconv.ParseInt(valueFromCmd[1], 10, 64)
    } else if state.Record != nil && !state.Record.Id.IsNull() {
        recordId = state.Record.Id.ValueInt64()
    }

    // Retrieve the record using V4 SDK.
    recordResponse, httpResponse, err := r.client.api.DNSRecordsAPI.RetrieveDnsRecord(ctx, recordId, zoneId).Execute()
    if err != nil {
        if httpResponse.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }
        // Handle other errors...
    }

    if httpResponse != nil {
        defer httpResponse.Body.Close()
    }

    // Update state.
    state.ZoneId = types.StringValue(valueFromCmd[0])
    state.Record = populateRecordModel(recordResponse.GetData(), "")

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

### Update Operation

```go
func (r *recordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan recordResourceModel
    diags := req.Plan.Get(ctx, &plan)
    // ... get plan

    var state recordResourceModel
    diags2 := req.State.Get(ctx, &state)
    // ... get state for record ID

    zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
    recordId := state.Record.Id.ValueInt64()

    // Build the record request (same as Create).
    recordReq := azionapi.NewRecordRequest(
        plan.Record.Name.ValueString(),
        plan.Record.Type.ValueString(),
        buildRdataList(plan.Record.Rdata),
    )
    recordReq.SetTtl(plan.Record.Ttl.ValueInt64())
    recordReq.SetPolicy(plan.Record.Policy.ValueString())
    // ... set optional fields

    // Execute update request.
    updateRecord, httpResponse, err := r.client.api.DNSRecordsAPI.UpdateDnsRecord(ctx, recordId, zoneId).
        RecordRequest(*recordReq).Execute()
    // ... handle errors and response
}
```

### Delete Operation

```go
func (r *recordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state recordResourceModel
    diags := req.State.Get(ctx, &state)
    // ... get state

    zoneId, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 64)
    recordId := state.Record.Id.ValueInt64()

    // Execute delete request.
    _, httpResponse, err := r.client.api.DNSRecordsAPI.DeleteDnsRecord(ctx, recordId, zoneId).Execute()
    // ... handle errors
}
```

### Import State

```go
func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Import format: "zone_id/record_id"
    resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
```

### Helper Functions

```go
// buildRdataList converts a slice of types.String to a slice of string.
func buildRdataList(rdata []types.String) []string {
    result := make([]string, len(rdata))
    for i, d := range rdata {
        result[i] = d.ValueString()
    }
    return result
}

// populateRecordModel populates a recordModel from an SDK Record.
func populateRecordModel(record azionapi.Record, policyOverride string) *recordModel {
    model := &recordModel{
        Id:   types.Int64Value(record.GetId()),
        Name: types.StringValue(record.GetName()),
        Type: types.StringValue(record.GetType()),
    }

    // Set TTL.
    if record.HasTtl() {
        model.Ttl = types.Int64Value(record.GetTtl())
    }

    // Set policy.
    if policyOverride != "" {
        model.Policy = types.StringValue(policyOverride)
    } else if record.HasPolicy() {
        model.Policy = types.StringValue(record.GetPolicy())
    }

    // Set weight and description for weighted policy.
    if model.Policy.ValueString() == "weighted" {
        if record.HasWeight() {
            model.Weight = types.Int64Value(record.GetWeight())
        }
        if record.HasDescription() {
            model.Description = types.StringValue(record.GetDescription())
        }
    }

    // Set rdata.
    rdata := record.GetRdata()
    model.Rdata = make([]types.String, len(rdata))
    for i, d := range rdata {
        model.Rdata[i] = types.StringValue(d)
    }

    return model
}
```

---

## Schema Definition Patterns

### Schema with Nested List Attribute

```go
func (d *RecordsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Optional: true,
            },
            "zone_id": schema.Int64Attribute{
                Required:    true,
                Description: "The zone identifier to target for the resource.",
            },
            "page": schema.Int64Attribute{
                Description: "The page number of Records.",
                Optional:    true,
            },
            "page_size": schema.Int64Attribute{
                Description: "The page size number of Records.",
                Optional:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total number of records.",
                Computed:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
                Computed:    true,
            },
            "links": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "previous": schema.StringAttribute{
                        Computed: true,
                    },
                    "next": schema.StringAttribute{
                        Computed: true,
                    },
                },
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "record_id": schema.Int64Attribute{
                            Description: "The record identifier.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Computed:    true,
                            Description: "The name of the DNS record.",
                        },
                        "description": schema.StringAttribute{
                            Computed: true,
                        },
                        "rdata": schema.ListAttribute{
                            Computed:    true,
                            ElementType: types.StringType,
                            Description: "List of answers replied by DNS Authoritative to that Record.",
                        },
                        "policy": schema.StringAttribute{
                            Computed:    true,
                            Description: "Must be 'simple' or 'weighted'.",
                        },
                        "type": schema.StringAttribute{
                            Computed:    true,
                            Description: "DNS record type (A, AAAA, ANAME, CNAME, MX, NS, PTR, SRV, TXT, CAA, DS).",
                        },
                        "ttl": schema.Int64Attribute{
                            Computed:    true,
                            Description: "Time-to-live defines max-time for packets life in seconds.",
                        },
                        "weight": schema.Int64Attribute{
                            Computed:    true,
                            Description: "Weight for weighted policy records.",
                        },
                    },
                },
            },
        },
    }
}
```

---

## Error Handling

### Standard Error Handler

```go
// errPrintRecords returns user-friendly error messages for records operations.
func errPrintRecords(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Records Found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

### Retry on Rate Limiting

Use `utils.RetryOn429` for handling rate limiting:

```go
if httpResp.StatusCode == 429 {
    recordsResponse, httpResp, err = utils.RetryOn429(func() (*azionapi.PaginatedRecordList, *http.Response, error) {
        return d.client.api.DNSRecordsAPI.ListDnsRecords(ctx, zoneId.ValueInt64()).
            Page(page.ValueInt64()).
            PageSize(pageSize.ValueInt64()).
            Execute()
    }, 5) // Maximum 5 retries

    if httpResp != nil {
        defer httpResp.Body.Close()
    }

    if err != nil {
        resp.Diagnostics.AddError(
            err.Error(),
            "API request failed after too many retries",
        )
        return
    }
}
```

---

## Type Conversions

### Building State from SDK Response

```go
// buildRecordsState constructs the state model from the API response.
func buildRecordsState(zoneId types.Int64, page types.Int64, pageSize types.Int64, response *azionapi.PaginatedRecordList) RecordsDataSourceModel {
    state := RecordsDataSourceModel{
        ZoneId:   zoneId,
        Page:     page,
        PageSize: pageSize,
        Links:    &RecordsResponseLinks{},
        Results:  []RecordDataSourceResult{}, // Initialize as empty slice to avoid null
    }

    // Set counter.
    if response.Count != nil {
        state.Counter = types.Int64Value(*response.Count)
    }

    // Set total pages.
    if response.TotalPages != nil {
        state.TotalPages = types.Int64Value(*response.TotalPages)
    }

    // Set links using SDK helper methods.
    if response.HasPrevious() {
        state.Links.Previous = types.StringValue(response.GetPrevious())
    } else {
        state.Links.Previous = types.StringNull()
    }

    if response.HasNext() {
        state.Links.Next = types.StringValue(response.GetNext())
    } else {
        state.Links.Next = types.StringNull()
    }

    // Set results.
    if response.HasResults() {
        for _, record := range response.GetResults() {
            recordResult := RecordDataSourceResult{
                RecordId: types.Int64Value(record.GetId()),
                Name:     types.StringValue(record.GetName()),
                Type:     types.StringValue(record.GetType()),
            }

            // Set optional description.
            if record.HasDescription() {
                recordResult.Description = types.StringValue(record.GetDescription())
            } else {
                recordResult.Description = types.StringNull()
            }

            // Set optional TTL.
            if record.HasTtl() {
                recordResult.Ttl = types.Int64Value(record.GetTtl())
            } else {
                recordResult.Ttl = types.Int64Null()
            }

            // Set optional policy.
            if record.HasPolicy() {
                recordResult.Policy = types.StringValue(record.GetPolicy())
            } else {
                recordResult.Policy = types.StringNull()
            }

            // Set optional weight.
            if record.HasWeight() {
                recordResult.Weight = types.Int64Value(record.GetWeight())
            } else {
                recordResult.Weight = types.Int64Null()
            }

            // Set rdata list.
            rdata := record.GetRdata()
            rdataList := make([]types.String, len(rdata))
            for i, d := range rdata {
                rdataList[i] = types.StringValue(d)
            }
            recordResult.Rdata = rdataList

            state.Results = append(state.Results, recordResult)
        }
    }

    // Set placeholder ID.
    state.Id = types.StringValue("placeholder")

    return state
}
```

### SDK Nullable Types

The V4 SDK provides helper methods for nullable fields:

```go
// Check if field is set.
if record.HasDescription() {
    // Field is present.
}

// Get field value (returns default if not set).
value := record.GetDescription()

// Get field with existence check.
value, ok := record.GetDescriptionOk()
```

---

## Common Issues

### 1. Missing `defer response.Body.Close()`

**Problem:** HTTP response body not closed leads to resource leaks.

**Solution:** Always close the response body after processing:

```go
if httpResp != nil {
    defer httpResp.Body.Close()
}
```

### 2. Incorrect Pagination Handling

**Problem:** Page and page size not properly validated before API call.

**Solution:** Always set default values for optional pagination:

```go
if page.IsNull() || page.IsUnknown() {
    page = types.Int64Value(1)
}
if pageSize.IsNull() || pageSize.IsUnknown() {
    pageSize = types.Int64Value(10)
}
```

### 3. Missing Optional Field Handling

**Problem:** Accessing nullable fields without checking if they exist causes panics or incorrect values.

**Solution:** Use `Has*` methods before accessing nullable fields:

```go
if record.HasDescription() {
    recordResult.Description = types.StringValue(record.GetDescription())
} else {
    recordResult.Description = types.StringNull()
}
```

### 4. Field Name Differences Between SDK Versions

**Problem:** V4 SDK uses different field names compared to legacy SDK.

| Legacy SDK (idns) | V4 SDK (`azion-api`) |
|-------------------|---------------------|
| `RecordId` | `Id` |
| `Entry` | `Name` |
| `RecordType` | `Type` |
| `AnswersList` | `Rdata` |
| `ZoneDomain` | Not in list response |

**Solution:** Use the correct field names from V4 SDK:

```go
// Legacy SDK (idns)
record.RecordId  // int64
record.Entry     // string
record.RecordType // string
record.AnswersList // []string

// V4 SDK (azion-api)
record.GetId()    // int64
record.GetName()  // string
record.GetType()  // string
record.GetRdata() // []string
```

### 5. Import Path Confusion

**Problem:** Using wrong import path for V4 SDK.

**Incorrect:**
```go
import "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
```

**Correct:**
```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### 6. RecordRequest Constructor

**Problem:** Not using the SDK constructor for creating request objects.

**Incorrect:**
```go
recordReq := azionapi.RecordRequest{
    Name: plan.Record.Name.ValueString(),
    Type: plan.Record.Type.ValueString(),
}
```

**Correct:** Use the constructor which ensures required fields are set:

```go
recordReq := azionapi.NewRecordRequest(
    plan.Record.Name.ValueString(),
    plan.Record.Type.ValueString(),
    buildRdataList(plan.Record.Rdata),
)
```

### 7. Int32 to Int64 Conversion

**Problem:** Legacy SDK used `int32` for IDs and TTL, V4 SDK uses `int64`.

**Solution:** Remove unnecessary `int32` conversions:

```go
// Legacy SDK (idns) - required int32 conversion
recordTlg32, err := utils.CheckInt64toInt32Security(plan.Record.Ttl.ValueInt64())

// V4 SDK (azion-api) - uses int64 directly
recordReq.SetTtl(plan.Record.Ttl.ValueInt64())
```

---

## Summary Checklist

When updating DNS Records data sources and resources:

1. **Use V4 SDK**: Import from `azion-api` package
2. **Use correct naming**: No `edge` prefix, use `id`, `name`, `type`, `rdata`
3. **Handle pagination**: Set default values for `page` and `page_size` (data source)
4. **Handle optional fields**: Use `Has*` methods before accessing nullable fields
5. **Handle rate limiting**: Use `utils.RetryOn429` for 429 errors
6. **Close response bodies**: Add `defer response.Body.Close()` after successful API calls
7. **Use SDK constructors**: Use `azionapi.NewRecordRequest()` for creating request objects
8. **Use int64 directly**: V4 SDK uses `int64` for IDs and TTL, no conversion needed
9. **Use Retrieve API**: Use `RetrieveDnsRecord(ctx, recordId, zoneId)` for single record reads
10. **Update documentation**: Keep docs and examples synchronized with schema changes
