package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	CreatedAt types.String `tfsdk:"created_at"`
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
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the device group.",
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
	data := ApplicationDeviceGroupData{
		ID:        types.Int64Value(deviceGroup.GetId()),
		Name:      types.StringValue(deviceGroup.GetName()),
		UserAgent: types.StringValue(deviceGroup.GetUserAgent()),
	}
	// Handle CreatedAt if it's set
	if deviceGroup.CreatedAt.IsSet() && deviceGroup.CreatedAt.Get() != nil {
		data.CreatedAt = types.StringValue(deviceGroup.GetCreatedAt().Format(time.RFC3339))
	}
	return ApplicationDeviceGroupDataSourceModel{
		Data: data,
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
