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
	CreatedAt types.String `tfsdk:"created_at"`
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
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the device group.",
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
	result := ApplicationDeviceGroupsResults{
		ID:        types.Int64Value(deviceGroup.GetId()),
		Name:      types.StringValue(deviceGroup.GetName()),
		UserAgent: types.StringValue(deviceGroup.GetUserAgent()),
	}
	// Handle CreatedAt if it's set
	if deviceGroup.CreatedAt.IsSet() && deviceGroup.CreatedAt.Get() != nil {
		result.CreatedAt = types.StringValue(deviceGroup.GetCreatedAt().Format(time.RFC3339))
	}
	return result
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
