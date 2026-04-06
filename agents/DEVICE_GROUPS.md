# Application Device Groups - Code Generation Guide

This document provides specific guidance for implementing Application Device Group resources and data sources in the Terraform provider.

## Table of Contents

1. [Overview](#overview)
2. [SDK Selection](#sdk-selection)
3. [File Structure](#file-structure)
4. [Data Source Implementation](#data-source-implementation)
   - [Singular Data Source](#singular-data-source)
   - [Plural Data Source](#plural-data-source)
5. [Resource Implementation](#resource-implementation)
   - [Resource Model](#resource-model)
   - [Create Operation](#create-operation)
   - [Read Operation](#read-operation)
   - [Update Operation](#update-operation)
   - [Delete Operation](#delete-operation)
   - [Import State](#import-state)
6. [Schema Definition](#schema-definition)
7. [Error Handling](#error-handling)
8. [Composite ID Pattern](#composite-id-pattern)
9. [Provider Registration](#provider-registration)
10. [Documentation](#documentation)
11. [Examples](#examples)

---

## Overview

This guide covers both data sources and resources for Application Device Groups:

| Type | Terraform Name | File Name |
|------|---------------|-----------|
| Singular Data Source | `azion_application_device_group` | `data_source_application_device_group.go` |
| Plural Data Source | `azion_application_device_groups` | `data_source_application_device_groups.go` |
| Resource | `azion_application_device_group` | `resource_application_device_group.go` |

### Naming Convention

| Type | File Name | Terraform Resource Name |
|------|-----------|------------------------|
| Singular Data Source | `data_source_<name>.go` | `azion_<name>` |
| Plural Data Source | `data_source_<name>s.go` | `azion_<name>s` |
| Resource | `resource_<name>.go` | `azion_<name>` |
| Example Directory | `examples/<type>/azion_<name>/` | - |
| Documentation | `docs/<type>/<name>.md` | - |

### What are Device Groups?

Device groups allow you to categorize user agents (browsers, devices) using regular expression patterns. They are associated with an Edge Application and can be used in cache settings to vary content by device type.

---

## SDK Selection

Application Device Groups use the **V4 SDK (`azion-api`)**:

```go
import azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
```

### API Client Access

```go
// Access via the api client field
client.api.ApplicationsDeviceGroupsAPI
```

### Key SDK Types

| Type | Description |
|------|-------------|
| `DeviceGroupResponse` | Response type for single device group operations |
| `PaginatedDeviceGroupList` | Response type for list operations |
| `DeviceGroup` | Individual device group in results |
| `DeviceGroupRequest` | Request type for create/update operations |
| `ListDeviceGroups(ctx, applicationId)` | API method to list all device groups |
| `RetrieveDeviceGroup(ctx, applicationId, deviceGroupId)` | API method to get a single device group |
| `CreateDeviceGroup(ctx, applicationId)` | API method to create a device group |
| `UpdateDeviceGroup(ctx, applicationId, deviceGroupId)` | API method to update a device group |
| `DeleteDeviceGroup(ctx, applicationId, deviceGroupId)` | API method to delete a device group |

### Key SDK Features

| Feature | V4 SDK (`azion-api`) |
|---------|---------------------|
| ID Type | `int64` |
| Create Request Type | `DeviceGroupRequest` |
| Update Request Type | `DeviceGroupRequest` |
| Response Type | `DeviceGroupResponse` with `Data` field |
| List Response Type | `PaginatedDeviceGroupList` |
| Parent Resource | Application (required) |

---

## File Structure

```
terraform-provider-azion/
├── internal/
│   ├── data_source_application_device_group.go    # Singular data source
│   ├── data_source_application_device_groups.go   # Plural data source
│   └── resource_application_device_group.go       # Resource
├── docs/
│   ├── data-sources/
│   │   ├── application_device_group.md            # Singular data source docs
│   │   └── application_device_groups.md           # Plural data source docs
│   └── resources/
│       └── application_device_group.md            # Resource docs
└── examples/
    ├── data-sources/
    │   ├── azion_application_device_group/
    │   │   └── data-source.tf                     # Singular example
    │   └── azion_application_device_groups/
    │       └── data-source.tf                     # Plural example
    └── resources/
        └── azion_application_device_group/
            └── resource.tf                        # Resource example
```

---

## Data Source Implementation

### Singular Data Source

The singular data source reads a single device group by ID:

**File:** `internal/data_source_application_device_group.go`

```go
package provider

import (
    "context"
    "fmt"
    "net/http"
    "strconv"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &ApplicationDeviceGroupDataSource{}
    _ datasource.DataSourceWithConfigure = &ApplicationDeviceGroupDataSource{}
)

func dataSourceAzionApplicationDeviceGroup() datasource.DataSource {
    return &ApplicationDeviceGroupDataSource{}
}

type ApplicationDeviceGroupDataSource struct {
    client *apiClient
}

type ApplicationDeviceGroupDataSourceModel struct {
    ApplicationID types.Int64                `tfsdk:"application_id"`
    ID            types.String               `tfsdk:"id"`
    Data          ApplicationDeviceGroupData `tfsdk:"data"`
}

type ApplicationDeviceGroupData struct {
    ID        types.Int64  `tfsdk:"id"`
    Name      types.String `tfsdk:"name"`
    UserAgent types.String `tfsdk:"user_agent"`
}

func (d *ApplicationDeviceGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *ApplicationDeviceGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_device_group"
}

func (d *ApplicationDeviceGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the device group.",
                Required:    true,
            },
            "data": schema.SingleNestedAttribute{
                Computed: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The device group identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the device group.",
                        Computed:    true,
                    },
                    "user_agent": schema.StringAttribute{
                        Description: "Regular expression pattern to identify user agents.",
                        Computed:    true,
                    },
                },
            },
        },
    }
}

func (d *ApplicationDeviceGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64
    var deviceGroupID types.String

    diags := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    diags = req.Config.GetAttribute(ctx, path.Root("id"), &deviceGroupID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    deviceGroupIDInt, err := strconv.ParseInt(deviceGroupID.ValueString(), 10, 64)
    if err != nil {
        resp.Diagnostics.AddError(
            "Value Conversion error",
            "Could not convert device group ID to integer",
        )
        return
    }

    deviceGroupResponse, response, err := d.client.api.ApplicationsDeviceGroupsAPI.
        RetrieveDeviceGroup(ctx, applicationID.ValueInt64(), deviceGroupIDInt).Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            deviceGroupResponse, response, err = utils.RetryOn429(func() (*azionapi.DeviceGroupResponse, *http.Response, error) {
                return d.client.api.ApplicationsDeviceGroupsAPI.RetrieveDeviceGroup(ctx, applicationID.ValueInt64(), deviceGroupIDInt).Execute() //nolint
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
            usrMsg, errMsg := errPrintApplicationDeviceGroup(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    deviceGroupState := populateApplicationDeviceGroupResults(ctx, deviceGroupResponse.GetData())
    deviceGroupState.ApplicationID = applicationID
    deviceGroupState.ID = deviceGroupID

    diags = resp.State.Set(ctx, &deviceGroupState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}

func populateApplicationDeviceGroupResults(_ context.Context, deviceGroup azionapi.DeviceGroup) ApplicationDeviceGroupDataSourceModel {
    return ApplicationDeviceGroupDataSourceModel{
        Data: ApplicationDeviceGroupData{
            ID:        types.Int64Value(deviceGroup.GetId()),
            Name:      types.StringValue(deviceGroup.GetName()),
            UserAgent: types.StringValue(deviceGroup.GetUserAgent()),
        },
    }
}

// errPrintApplicationDeviceGroup returns user-friendly error messages for device group operations.
func errPrintApplicationDeviceGroup(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "Device Group not found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

### Plural Data Source Implementation

The plural data source lists multiple device groups:

**File:** `internal/data_source_application_device_groups.go`

```go
package provider

import (
    "context"
    "fmt"
    "net/http"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var (
    _ datasource.DataSource              = &ApplicationDeviceGroupsDataSource{}
    _ datasource.DataSourceWithConfigure = &ApplicationDeviceGroupsDataSource{}
)

func dataSourceAzionApplicationDeviceGroups() datasource.DataSource {
    return &ApplicationDeviceGroupsDataSource{}
}

type ApplicationDeviceGroupsDataSource struct {
    client *apiClient
}

type ApplicationDeviceGroupsDataSourceModel struct {
    ApplicationID types.Int64                      `tfsdk:"application_id"`
    Counter       types.Int64                      `tfsdk:"counter"`
    TotalPages    types.Int64                      `tfsdk:"total_pages"`
    Results       []ApplicationDeviceGroupsResults `tfsdk:"results"`
    ID            types.String                     `tfsdk:"id"`
}

type ApplicationDeviceGroupsResults struct {
    ID        types.Int64  `tfsdk:"id"`
    Name      types.String `tfsdk:"name"`
    UserAgent types.String `tfsdk:"user_agent"`
}

func (d *ApplicationDeviceGroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    d.client = req.ProviderData.(*apiClient)
}

func (d *ApplicationDeviceGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_application_device_groups"
}

func (d *ApplicationDeviceGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
            "id": schema.StringAttribute{
                Description: "Numeric identifier of the data source.",
                Computed:    true,
            },
            "counter": schema.Int64Attribute{
                Description: "The total count of device groups.",
                Computed:    true,
            },
            "total_pages": schema.Int64Attribute{
                Description: "The total number of pages.",
                Computed:    true,
            },
            "results": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.Int64Attribute{
                            Description: "The device group identifier.",
                            Computed:    true,
                        },
                        "name": schema.StringAttribute{
                            Description: "Name of the device group.",
                            Computed:    true,
                        },
                        "user_agent": schema.StringAttribute{
                            Description: "Regular expression pattern to identify user agents.",
                            Computed:    true,
                        },
                    },
                },
            },
        },
    }
}

func (d *ApplicationDeviceGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var applicationID types.Int64

    diags := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    deviceGroupsResponse, response, err := d.client.api.ApplicationsDeviceGroupsAPI.
        ListDeviceGroups(ctx, applicationID.ValueInt64()).Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            deviceGroupsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedDeviceGroupList, *http.Response, error) {
                return d.client.api.ApplicationsDeviceGroupsAPI.ListDeviceGroups(ctx, applicationID.ValueInt64()).Execute() //nolint
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
            usrMsg, errMsg := errPrintApplicationDeviceGroups(response.StatusCode, err)
            resp.Diagnostics.AddError(usrMsg, errMsg)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    deviceGroupsState := ApplicationDeviceGroupsDataSourceModel{
        ApplicationID: applicationID,
    }

    if deviceGroupsResponse.Count != nil {
        deviceGroupsState.Counter = types.Int64Value(*deviceGroupsResponse.Count)
    }

    if deviceGroupsResponse.TotalPages != nil {
        deviceGroupsState.TotalPages = types.Int64Value(*deviceGroupsResponse.TotalPages)
    }

    for _, deviceGroup := range deviceGroupsResponse.GetResults() {
        result := populateApplicationDeviceGroupsResults(ctx, deviceGroup)
        deviceGroupsState.Results = append(deviceGroupsState.Results, result)
    }

    deviceGroupsState.ID = types.StringValue("Get All Application Device Groups")

    diags = resp.State.Set(ctx, &deviceGroupsState)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}

func populateApplicationDeviceGroupsResults(_ context.Context, deviceGroup azionapi.DeviceGroup) ApplicationDeviceGroupsResults {
    return ApplicationDeviceGroupsResults{
        ID:        types.Int64Value(deviceGroup.GetId()),
        Name:      types.StringValue(deviceGroup.GetName()),
        UserAgent: types.StringValue(deviceGroup.GetUserAgent()),
    }
}

// errPrintApplicationDeviceGroups returns user-friendly error messages for device groups operations.
func errPrintApplicationDeviceGroups(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "No Device Groups found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

### Key Differences: Singular vs Plural Data Sources

| Feature | Singular (application_device_group) | Plural (application_device_groups) |
|---------|-------------------------------------|-----------------------------------|
| Input | `application_id` (Required), `id` (Required) | `application_id` (Required) |
| Output | `data` (SingleNestedAttribute) | `results` (ListNestedAttribute) |
| Response Type | `DeviceGroupResponse` | `PaginatedDeviceGroupList` |
| API Method | `RetrieveDeviceGroup(ctx, appId, groupId)` | `ListDeviceGroups(ctx, appId)` |
| Pagination | No | Yes (counter, total_pages) |
| Schema ID | Device group ID string | `"Get All Application Device Groups"` |

---

## Resource Implementation

Resources support full CRUD operations (Create, Read, Update, Delete) and Import.

### Resource Model

**File:** `internal/resource_application_device_group.go`

```go
package provider

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
    "time"

    azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
    "github.com/aziontech/terraform-provider-azion/internal/utils"
    "github.com/hashicorp/terraform-plugin-framework/diag"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
    _ resource.Resource                = &applicationDeviceGroupResource{}
    _ resource.ResourceWithConfigure   = &applicationDeviceGroupResource{}
    _ resource.ResourceWithImportState = &applicationDeviceGroupResource{}
)

func NewApplicationDeviceGroupResource() resource.Resource {
    return &applicationDeviceGroupResource{}
}

type applicationDeviceGroupResource struct {
    client *apiClient
}

// Main resource model.
type applicationDeviceGroupResourceModel struct {
    ApplicationID types.Int64                 `tfsdk:"application_id"`
    DeviceGroup   *deviceGroupResourceResults `tfsdk:"device_group"`
    ID            types.String                `tfsdk:"id"`
    LastUpdated   types.String                `tfsdk:"last_updated"`
    SchemaVersion types.Int64                 `tfsdk:"schema_version"`
}

// Device group results - all fields.
type deviceGroupResourceResults struct {
    ID        types.Int64  `tfsdk:"id"`
    Name      types.String `tfsdk:"name"`
    UserAgent types.String `tfsdk:"user_agent"`
}
```

### Create Operation

```go
func (r *applicationDeviceGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan applicationDeviceGroupResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build the device group request for V4 API.
    deviceGroupRequest := azionapi.DeviceGroupRequest{
        Name:      plan.DeviceGroup.Name.ValueString(),
        UserAgent: plan.DeviceGroup.UserAgent.ValueString(),
    }

    // Call the V4 API.
    deviceGroupResponse, response, err := r.client.api.ApplicationsDeviceGroupsAPI.
        CreateDeviceGroup(ctx, plan.ApplicationID.ValueInt64()).
        DeviceGroupRequest(deviceGroupRequest).
        Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            deviceGroupResponse, response, err = utils.RetryOn429(func() (*azionapi.DeviceGroupResponse, *http.Response, error) {
                return r.client.api.ApplicationsDeviceGroupsAPI.
                    CreateDeviceGroup(ctx, plan.ApplicationID.ValueInt64()).
                    DeviceGroupRequest(deviceGroupRequest).
                    Execute() //nolint
            }, 5) // Maximum 5 retries

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Populate the state from the API response.
    data := deviceGroupResponse.GetData()
    plan.DeviceGroup = &deviceGroupResourceResults{
        ID:        types.Int64Value(data.GetId()),
        Name:      types.StringValue(data.GetName()),
        UserAgent: types.StringValue(data.GetUserAgent()),
    }
    plan.SchemaVersion = types.Int64Value(1)
    plan.ID = types.StringValue(fmt.Sprintf("%d:%d", plan.ApplicationID.ValueInt64(), data.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Read Operation

```go
func (r *applicationDeviceGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state applicationDeviceGroupResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse the composite ID to get application_id and device_group_id.
    applicationID, deviceGroupID, err := parseDeviceGroupID(state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("ID Parsing Error", err.Error())
        return
    }

    // Call the V4 API.
    deviceGroupResponse, response, err := r.client.api.ApplicationsDeviceGroupsAPI.
        RetrieveDeviceGroup(ctx, applicationID, deviceGroupID).
        Execute() //nolint
    if err != nil {
        if response.StatusCode == http.StatusNotFound {
            resp.State.RemoveResource(ctx)
            return
        }

        if response.StatusCode == 429 {
            deviceGroupResponse, response, err = utils.RetryOn429(func() (*azionapi.DeviceGroupResponse, *http.Response, error) {
                return r.client.api.ApplicationsDeviceGroupsAPI.RetrieveDeviceGroup(ctx, applicationID, deviceGroupID).Execute() //nolint
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Populate the state from the API response.
    data := deviceGroupResponse.GetData()
    state.ApplicationID = types.Int64Value(applicationID)
    state.DeviceGroup = &deviceGroupResourceResults{
        ID:        types.Int64Value(data.GetId()),
        Name:      types.StringValue(data.GetName()),
        UserAgent: types.StringValue(data.GetUserAgent()),
    }
    state.SchemaVersion = types.Int64Value(1)

    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Update Operation

```go
func (r *applicationDeviceGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan applicationDeviceGroupResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    var state applicationDeviceGroupResourceModel
    diags = req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse the composite ID to get application_id and device_group_id.
    applicationID, deviceGroupID, err := parseDeviceGroupID(state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("ID Parsing Error", err.Error())
        return
    }

    // Build the device group request for V4 API.
    deviceGroupRequest := azionapi.DeviceGroupRequest{
        Name:      plan.DeviceGroup.Name.ValueString(),
        UserAgent: plan.DeviceGroup.UserAgent.ValueString(),
    }

    // Call the V4 API.
    deviceGroupResponse, response, err := r.client.api.ApplicationsDeviceGroupsAPI.
        UpdateDeviceGroup(ctx, applicationID, deviceGroupID).
        DeviceGroupRequest(deviceGroupRequest).
        Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            deviceGroupResponse, response, err = utils.RetryOn429(func() (*azionapi.DeviceGroupResponse, *http.Response, error) {
                return r.client.api.ApplicationsDeviceGroupsAPI.
                    UpdateDeviceGroup(ctx, applicationID, deviceGroupID).
                    DeviceGroupRequest(deviceGroupRequest).
                    Execute() //nolint
            }, 5) // Maximum 5 retries

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }

    // Populate the state from the API response.
    data := deviceGroupResponse.GetData()
    plan.ApplicationID = types.Int64Value(applicationID)
    plan.DeviceGroup = &deviceGroupResourceResults{
        ID:        types.Int64Value(data.GetId()),
        Name:      types.StringValue(data.GetName()),
        UserAgent: types.StringValue(data.GetUserAgent()),
    }
    plan.SchemaVersion = types.Int64Value(1)
    plan.ID = types.StringValue(fmt.Sprintf("%d:%d", applicationID, data.GetId()))
    plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }
}
```

### Delete Operation

```go
func (r *applicationDeviceGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state applicationDeviceGroupResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Parse the composite ID to get application_id and device_group_id.
    applicationID, deviceGroupID, err := parseDeviceGroupID(state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("ID Parsing Error", err.Error())
        return
    }

    // Call the V4 API.
    _, response, err := r.client.api.ApplicationsDeviceGroupsAPI.
        DeleteDeviceGroup(ctx, applicationID, deviceGroupID).
        Execute() //nolint
    if err != nil {
        if response.StatusCode == 429 {
            _, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
                return r.client.api.ApplicationsDeviceGroupsAPI.DeleteDeviceGroup(ctx, applicationID, deviceGroupID).Execute() //nolint
            }, 5)

            if response != nil {
                defer response.Body.Close()
            }

            if err != nil {
                resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
                return
            }
        } else if response.StatusCode == http.StatusNotFound {
            // Resource already deleted.
            return
        } else {
            bodyBytes, errReadAll := io.ReadAll(response.Body)
            if errReadAll != nil {
                resp.Diagnostics.AddError(errReadAll.Error(), "err")
            }
            bodyString := string(bodyBytes)
            resp.Diagnostics.AddError(err.Error(), bodyString)
            return
        }
    }

    if response != nil {
        defer response.Body.Close()
    }
}
```

### Import State

```go
func (r *applicationDeviceGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Parse the composite ID to get application_id and device_group_id.
    applicationID, deviceGroupID, err := parseDeviceGroupID(req.ID)
    if err != nil {
        resp.Diagnostics.AddError("ID Parsing Error", err.Error())
        return
    }

    // Set the application_id attribute.
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Set the device_group.id attribute.
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_group").AtName("id"), deviceGroupID)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Set the composite ID.
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
```

---

## Schema Definition

### Required vs Optional vs Computed

| Attribute | Data Source (Singular) | Data Source (Plural) | Resource |
|-----------|----------------------|---------------------|----------|
| `application_id` | Required | Required | Required |
| `id` | Required | Computed | Computed |
| `device_group.id` | N/A | N/A | Computed |
| `device_group.name` | Computed | Computed | Required |
| `device_group.user_agent` | Computed | Computed | Required |
| `last_updated` | N/A | N/A | Computed |
| `schema_version` | N/A | N/A | Computed |

### Resource Schema

```go
func (r *applicationDeviceGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Creates an application device group resource. Device groups allow you to categorize user agents (browsers, devices) using regular expression patterns.",
        Attributes: map[string]schema.Attribute{
            "application_id": schema.Int64Attribute{
                Description: "The application identifier.",
                Required:    true,
            },
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
            "schema_version": schema.Int64Attribute{
                Computed: true,
            },
            "device_group": schema.SingleNestedAttribute{
                Required: true,
                Attributes: map[string]schema.Attribute{
                    "id": schema.Int64Attribute{
                        Description: "The device group identifier.",
                        Computed:    true,
                    },
                    "name": schema.StringAttribute{
                        Description: "Name of the device group.",
                        Required:    true,
                    },
                    "user_agent": schema.StringAttribute{
                        Description: "Regular expression pattern to identify user agents.",
                        Required:    true,
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

// For Delete operations - handle 404 as already deleted
if response.StatusCode == http.StatusNotFound {
    return  // Resource already deleted, nothing to do
}
```

### Error Message Helper

```go
// errPrintApplicationDeviceGroup returns user-friendly error messages.
func errPrintApplicationDeviceGroup(errCode int, err error) (string, string) {
    var usrMsg string
    switch errCode {
    case 400:
        usrMsg = "Bad Request"
    case 401:
        usrMsg = "Unauthorized Token"
    case 404:
        usrMsg = "Device Group not found"
    default:
        usrMsg = err.Error()
    }

    errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
    return usrMsg, errMsg
}
```

---

## Composite ID Pattern

Device groups are child resources of applications, so they use a **composite ID** pattern.

### ID Format

The composite ID format is: `application_id:device_group_id`

Example: `12345:678` where:
- `12345` is the application ID
- `678` is the device group ID

### ID Parsing Function

```go
// parseDeviceGroupID parses the composite ID "application_id:device_group_id".
func parseDeviceGroupID(id string) (int64, int64, error) {
    parts := strings.Split(id, ":")
    if len(parts) != 2 {
        return 0, 0, fmt.Errorf("invalid ID format: expected 'application_id:device_group_id', got '%s'", id)
    }

    applicationID, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to parse application_id: %w", err)
    }

    deviceGroupID, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to parse device_group_id: %w", err)
    }

    return applicationID, deviceGroupID, nil
}
```

### ID Creation

```go
// When creating the ID from Create operation
plan.ID = types.StringValue(fmt.Sprintf("%d:%d", plan.ApplicationID.ValueInt64(), data.GetId()))
```

### Import ID Format

When importing, users must provide the composite ID:

```bash
terraform import azion_application_device_group.example "12345:678"
```

---

## Provider Registration

All data sources and resources must be registered in `internal/provider.go`:

```go
func (p *azionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        dataSourceAzionApplicationDeviceGroup,
        dataSourceAzionApplicationDeviceGroups,
        // ... other data sources
    }
}

func (p *azionProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewApplicationDeviceGroupResource,
        // ... other resources
    }
}
```

---

## Documentation

### Data Source Documentation (Singular)

Located in `docs/data-sources/application_device_group.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_application_device_group"
description: |-
  Provides a data source to read a specific Application Device Group.
---

# azion_application_device_group

Use this data source to read a specific Application Device Group.

## Example Usage

```hcl
data "azion_application_device_group" "example" {
  application_id = 12345
  id            = "678"
}
```

## Argument Reference

* `application_id` - (Required) The ID of the application.
* `id` - (Required) The ID of the device group.

## Attribute Reference

* `data` - The device group data.
  * `id` - The device group identifier.
  * `name` - Name of the device group.
  * `user_agent` - Regular expression pattern to identify user agents.
```

### Resource Documentation

Located in `docs/resources/application_device_group.md`:

```markdown
---
subcategory: ""
layout: "azion"
page_title: "Azion: azion_application_device_group"
description: |-
  Provides a resource to manage Application Device Groups.
---

# azion_application_device_group

Creates an Application Device Group resource.

## Example Usage

```hcl
resource "azion_application_device_group" "example" {
  application_id = 12345
  device_group = {
    name       = "Mobile Devices"
    user_agent = "(?i)(mobile|android|iphone)"
  }
}
```

## Import

```sh
terraform import azion_application_device_group.example "12345:678"
```

## Argument Reference

* `application_id` - (Required) The ID of the application.
* `device_group` - (Required) The device group configuration.
  * `name` - (Required) Name of the device group.
  * `user_agent` - (Required) Regular expression pattern to identify user agents.

## Attribute Reference

* `id` - The composite ID in format `application_id:device_group_id`.
* `last_updated` - Timestamp of the last Terraform update.
* `schema_version` - Schema version.
* `device_group.id` - The device group identifier.
```

---

## Examples

### Data Source Example (Singular)

**File:** `examples/data-sources/azion_application_device_group/data-source.tf`

```hcl
data "azion_application_device_group" "example" {
  application_id = 12345
  id            = "678"
}
```

### Data Source Example (Plural)

**File:** `examples/data-sources/azion_application_device_groups/data-source.tf`

```hcl
data "azion_application_device_groups" "example" {
  application_id = 12345
}
```

### Resource Example

**File:** `examples/resources/azion_application_device_group/resource.tf`

```hcl
resource "azion_application_device_group" "example" {
  application_id = 12345
  device_group = {
    name       = "Mobile Devices"
    user_agent = "(?i)(mobile|android|iphone)"
  }
}
```

---

## Summary Checklist

When generating or updating Application Device Group data sources or resources:

1. **Use V4 SDK**: Import from `github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api`
2. **Use correct naming**: Use `application_device_group` (not just `device_group`)
3. **Use correct API client**: `client.api.ApplicationsDeviceGroupsAPI`
4. **Handle parent resource**: Device groups require `application_id`
5. **Use composite ID**: Format `application_id:device_group_id`
6. **Implement parseDeviceGroupID**: Parse composite ID for Read, Update, Delete, Import
7. **Handle 429 errors**: Use `utils.RetryOn429`
8. **Close response bodies**: Add `defer response.Body.Close()` after retries
9. **Handle 404 on Read**: Call `resp.State.RemoveResource(ctx)`
10. **Handle 404 on Delete**: Return silently (resource already deleted)
11. **Register in provider.go**: Ensure all data sources and resources are registered
12. **Update documentation**: Create/update docs in `docs/` directory
13. **Create examples**: Add example files in `examples/` directory

---

## API Reference

### DeviceGroup Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `int64` | Unique identifier |
| `name` | `string` | Device group name |
| `user_agent` | `string` | Regular expression pattern to match user agents |

### DeviceGroupRequest Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Device group name (required) |
| `user_agent` | `string` | Regular expression pattern (required) |

### API Endpoints

| Operation | Method | Endpoint |
|-----------|--------|----------|
| List | `ListDeviceGroups(ctx, applicationId)` | `GET /applications/{applicationId}/device-groups` |
| Retrieve | `RetrieveDeviceGroup(ctx, applicationId, deviceGroupId)` | `GET /applications/{applicationId}/device-groups/{deviceGroupId}` |
| Create | `CreateDeviceGroup(ctx, applicationId).DeviceGroupRequest(req)` | `POST /applications/{applicationId}/device-groups` |
| Update | `UpdateDeviceGroup(ctx, applicationId, deviceGroupId).DeviceGroupRequest(req)` | `PUT /applications/{applicationId}/device-groups/{deviceGroupId}` |
| Delete | `DeleteDeviceGroup(ctx, applicationId, deviceGroupId)` | `DELETE /applications/{applicationId}/device-groups/{deviceGroupId}` |

---

## Common Issues

### 1. Missing Application ID

**Problem:** Forgetting to require `application_id` as a required field.

**Solution:** Device groups are child resources of applications and always require the parent ID.

### 2. Wrong ID Format on Import

**Problem:** Users providing only the device group ID instead of the composite ID.

**Solution:** Document the composite ID format clearly and provide helpful error messages:

```go
func parseDeviceGroupID(id string) (int64, int64, error) {
    parts := strings.Split(id, ":")
    if len(parts) != 2 {
        return 0, 0, fmt.Errorf("invalid ID format: expected 'application_id:device_group_id', got '%s'", id)
    }
    // ...
}
```

### 3. Not Closing Response Body

**Problem:** HTTP response body not closed after retry, causing resource leaks.

**Solution:** Always close the response body:

```go
if response != nil {
    defer response.Body.Close()
}
```

### 4. Using Wrong API Client Field

**Problem:** Using `client.applicationsApi` instead of `client.api.ApplicationsDeviceGroupsAPI`.

**Solution:** Device groups use the V4 SDK:

```go
// Wrong
client.applicationsApi.DeviceGroupsApi

// Correct
client.api.ApplicationsDeviceGroupsAPI
```

### 5. Not Handling 404 on Delete

**Problem:** Returning error when resource is already deleted.

**Solution:** Treat 404 as success on Delete:

```go
if response.StatusCode == http.StatusNotFound {
    return  // Resource already deleted
}
```
