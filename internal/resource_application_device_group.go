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
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	UserAgent    types.String `tfsdk:"user_agent"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *applicationDeviceGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_device_group"
}

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
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the device group.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "The last modified timestamp of the device group.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *applicationDeviceGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

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
	// Handle CreatedAt if it's set
	if data.CreatedAt.IsSet() && data.CreatedAt.Get() != nil {
		plan.DeviceGroup.CreatedAt = types.StringValue(data.GetCreatedAt().Format(time.RFC3339))
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
	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.ApplicationsDeviceGroupsAPI.DeleteDeviceGroup(ctx, applicationID, deviceGroupID).Execute() //nolint
	}, 5)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			// Resource already deleted.
			return
		}
		bodyBytes, errReadAll := io.ReadAll(response.Body)
		if errReadAll != nil {
			resp.Diagnostics.AddError(errReadAll.Error(), "err")
		}
		bodyString := string(bodyBytes)
		resp.Diagnostics.AddError(err.Error(), bodyString)
		return
	}
}

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
