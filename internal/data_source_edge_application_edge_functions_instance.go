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
	_ datasource.DataSource              = &EdgeApplicationsEdgeFunctionInstanceDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeApplicationsEdgeFunctionInstanceDataSource{}
)

func dataSourceAzionEdgeApplicationsEdgeFunctionsInstance() datasource.DataSource {
	return &EdgeApplicationsEdgeFunctionInstanceDataSource{}
}

type EdgeApplicationsEdgeFunctionInstanceDataSource struct {
	client *apiClient
}

type EdgeFunctionsInstanceDataSourceModel struct {
	ID            types.Int64                     `tfsdk:"id"`
	ApplicationID types.Int64                     `tfsdk:"application_id"`
	TotalCount    types.Int64                     `tfsdk:"total_count"`
	Results       []EdgeFunctionsInstanceResponse `tfsdk:"results"`
}

type EdgeFunctionsInstanceResponse struct {
	ID           types.Int64  `tfsdk:"id"`
	FunctionID   types.Int64  `tfsdk:"function_id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (d *EdgeApplicationsEdgeFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *EdgeApplicationsEdgeFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_edge_functions_instance"
}

func (d *EdgeApplicationsEdgeFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Application",
				Required:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "The total number of edge function instances.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The function instance identifier.",
							Computed:    true,
						},
						"function_id": schema.Int64Attribute{
							Description: "The edge function identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the function instance.",
							Computed:    true,
						},
						"args": schema.StringAttribute{
							Description: "Arguments of the function instance.",
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
					},
				},
			},
		},
	}
}

func (d *EdgeApplicationsEdgeFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var Page types.Int64
	var PageSize types.Int64
	var EdgeApplicationId types.Int64
	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &Page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &PageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("application_id"), &EdgeApplicationId)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}
	if PageSize.ValueInt64() == 0 {
		PageSize = types.Int64Value(10)
	}

	edgeFunctionInstancesResponse, response, err := d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, EdgeApplicationId.ValueInt64()).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedFunctionInstanceList, *http.Response, error) {
				return d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, EdgeApplicationId.ValueInt64()).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
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

	edgeApplicationsEdgeFunctionsInstanceState := EdgeFunctionsInstanceDataSourceModel{
		ApplicationID: EdgeApplicationId,
		TotalCount:    types.Int64Value(edgeFunctionInstancesResponse.GetCount()),
	}

	for _, resultEdgeApplication := range edgeFunctionInstancesResponse.GetResults() {
		jsonArgsStr, err := utils.ConvertInterfaceToString(resultEdgeApplication.GetArgs())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"error reading args from response",
			)
		}
		edgeApplicationsEdgeFunctionsInstanceState.Results = append(edgeApplicationsEdgeFunctionsInstanceState.Results, EdgeFunctionsInstanceResponse{
			ID:           types.Int64Value(resultEdgeApplication.GetId()),
			FunctionID:   types.Int64Value(resultEdgeApplication.GetFunction()),
			Name:         types.StringValue(resultEdgeApplication.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Active:       types.BoolValue(resultEdgeApplication.GetActive()),
			LastEditor:   types.StringValue(resultEdgeApplication.GetLastEditor()),
			LastModified: types.StringValue(resultEdgeApplication.GetLastModified().Format(time.RFC3339)),
		})
	}

	edgeApplicationsEdgeFunctionsInstanceState.ID = types.Int64Value(0)
	diags := resp.State.Set(ctx, &edgeApplicationsEdgeFunctionsInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
