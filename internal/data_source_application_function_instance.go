package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ApplicationFunctionInstanceDataSource{}
	_ datasource.DataSourceWithConfigure = &ApplicationFunctionInstanceDataSource{}
)

func dataSourceAzionApplicationFunctionInstance() datasource.DataSource {
	return &ApplicationFunctionInstanceDataSource{}
}

type ApplicationFunctionInstanceDataSource struct {
	client *apiClient
}

type FunctionInstanceDataSourceModel struct {
	ID            types.Int64  `tfsdk:"id"`
	ApplicationID types.Int64  `tfsdk:"application_id"`
	FunctionID    types.Int64  `tfsdk:"function_id"`
	FunctionName  types.String `tfsdk:"name"`
	Args          types.String `tfsdk:"args"`
	Active        types.Bool   `tfsdk:"active"`
	LastEditor    types.String `tfsdk:"last_editor"`
	LastModified  types.String `tfsdk:"last_modified"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *ApplicationFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *ApplicationFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_function_instance"
}

func (d *ApplicationFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the function instance.",
				Required:    true,
			},
			"application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Application.",
				Required:    true,
			},
			"function_id": schema.Int64Attribute{
				Description: "The function identifier.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the function instance.",
				Computed:    true,
			},
			"args": schema.StringAttribute{
				Description: "Arguments of the function instance in JSON format.",
				Computed:    true,
			},
			"active": schema.BoolAttribute{
				Description: "Active status of the function instance.",
				Computed:    true,
			},
			"last_editor": schema.StringAttribute{
				Description: "Last editor of the function instance.",
				Computed:    true,
			},
			"last_modified": schema.StringAttribute{
				Description: "Last modified timestamp of the function instance.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the function instance.",
				Computed:    true,
			},
		},
	}
}

func (d *ApplicationFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var applicationID types.Int64
	diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
	resp.Diagnostics.Append(diagsApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	var functionInstanceID types.Int64
	diagsFunctionInstanceID := req.Config.GetAttribute(ctx, path.Root("id"), &functionInstanceID)
	resp.Diagnostics.Append(diagsFunctionInstanceID...)
	if resp.Diagnostics.HasError() {
		return
	}

	functionInstanceResponse, response, err := d.client.api.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, applicationID.ValueInt64(), functionInstanceID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
				return d.client.api.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, applicationID.ValueInt64(), functionInstanceID.ValueInt64()).Execute() //nolint
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
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(
					errReadAll.Error(),
					"error reading response from api",
				)
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(
				err.Error(),
				bodyString,
			)
			return
		}
	}

	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"error reading args from response",
		)
	}

	state := FunctionInstanceDataSourceModel{
		ID:            types.Int64Value(functionInstanceResponse.Data.GetId()),
		ApplicationID: applicationID,
		FunctionID:    types.Int64Value(functionInstanceResponse.Data.GetFunction()),
		FunctionName:  types.StringValue(functionInstanceResponse.Data.GetName()),
		Args:          types.StringValue(jsonArgsStr),
		Active:        types.BoolValue(functionInstanceResponse.Data.GetActive()),
		LastEditor:    types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
		LastModified:  types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC3339)),
		CreatedAt:     types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
