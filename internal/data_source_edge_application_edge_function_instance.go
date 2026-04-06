package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"

	edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &EdgeApplicationEdgeFunctionInstanceDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeApplicationEdgeFunctionInstanceDataSource{}
)

func dataSourceAzionEdgeApplicationEdgeFunctionInstance() datasource.DataSource {
	return &EdgeApplicationEdgeFunctionInstanceDataSource{}
}

type EdgeApplicationEdgeFunctionInstanceDataSource struct {
	client *apiClient
}

type EdgeFunctionInstanceDataSourceModel struct {
	ID            types.String                 `tfsdk:"id"`
	ApplicationID types.Int64                  `tfsdk:"application_id"`
	Data          EdgeFunctionInstanceResponse `tfsdk:"data"`
}

type EdgeFunctionInstanceResponse struct {
	ID           types.Int64  `tfsdk:"id"`
	FunctionID   types.Int64  `tfsdk:"function_id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (d *EdgeApplicationEdgeFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *EdgeApplicationEdgeFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_edge_function_instance"
}

func (d *EdgeApplicationEdgeFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"application_id": schema.StringAttribute{
				Description: "Numeric identifier of the Edge Application",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The function identifier.",
						Required:    true,
					},
					"edge_function_id": schema.Int64Attribute{
						Description: "The function identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Computed:    true,
					},
					"args": schema.StringAttribute{
						Description: "Code of the function.",
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
			},
		},
	}
}

func (d *EdgeApplicationEdgeFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var EdgeApplicationId types.Int64
	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("application_id"), &EdgeApplicationId)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	var EdgeFunctionInstanceId types.Int64
	diagsEdgeFunctionInstanceId := req.Config.GetAttribute(ctx, path.Root("data").AtName("id"), &EdgeFunctionInstanceId)
	resp.Diagnostics.Append(diagsEdgeFunctionInstanceId...)
	if resp.Diagnostics.HasError() {
		return
	}
	functionInstancesResponse, response, err := d.client.edgeApi.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, EdgeApplicationId.ValueInt64(), EdgeFunctionInstanceId.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionInstancesResponse, response, err = utils.RetryOn429(func() (*edgeapi.FunctionInstanceResponse, *http.Response, error) {
				return d.client.edgeApi.ApplicationsFunctionAPI.RetrieveApplicationFunctionInstance(ctx, EdgeApplicationId.ValueInt64(), EdgeFunctionInstanceId.ValueInt64()).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstancesResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"error reading args from response",
		)
	}
	edgeApplicationsEdgeFunctionsInstanceState := EdgeFunctionInstanceDataSourceModel{
		ApplicationID: EdgeApplicationId,
		Data: EdgeFunctionInstanceResponse{
			ID:           types.Int64Value(functionInstancesResponse.Data.GetId()),
			FunctionID:   types.Int64Value(functionInstancesResponse.Data.GetFunction()),
			Name:         types.StringValue(functionInstancesResponse.Data.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Active:       types.BoolValue(functionInstancesResponse.Data.GetActive()),
			LastEditor:   types.StringValue(functionInstancesResponse.Data.GetLastEditor()),
			LastModified: types.StringValue(functionInstancesResponse.Data.GetLastModified().Format(time.RFC3339)),
			CreatedAt:    types.StringValue(functionInstancesResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		},
	}

	edgeApplicationsEdgeFunctionsInstanceState.ID = types.StringValue("Get Functions Instances By Applications")
	diags := resp.State.Set(ctx, &edgeApplicationsEdgeFunctionsInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
