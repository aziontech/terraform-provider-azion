package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/path"

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
	ApplicationID types.Int64                  `tfsdk:"edge_application_id"`
	SchemaVersion types.Int64                  `tfsdk:"schema_version"`
	Results       EdgeFunctionInstanceResponse `tfsdk:"results"`
}

type GetEdgeFunctionInstanceResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFunctionInstanceResponse struct {
	ID             types.Int64  `tfsdk:"id"`
	EdgeFunctionID types.Int64  `tfsdk:"edge_function_id"`
	Name           types.String `tfsdk:"name"`
	Args           types.String `tfsdk:"args"`
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
			"edge_application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Application",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
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
				},
			},
		},
	}
}

func (d *EdgeApplicationEdgeFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var EdgeApplicationId types.Int64
	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &EdgeApplicationId)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	var EdgeFunctionInstanceId types.Int64
	diagsEdgeFunctionInstanceId := req.Config.GetAttribute(ctx, path.Root("results").AtName("id"), &EdgeFunctionInstanceId)
	resp.Diagnostics.Append(diagsEdgeFunctionInstanceId...)
	if resp.Diagnostics.HasError() {
		return
	}
	edgeFunctionInstancesResponse, response, err := d.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesFunctionsInstancesIdGet(ctx, EdgeApplicationId.ValueInt64(), EdgeFunctionInstanceId.ValueInt64()).Execute() //nolint
	if err != nil {
		bodyBytes, errReadAll := io.ReadAll(response.Body)
		if errReadAll != nil {
			resp.Diagnostics.AddError(
				errReadAll.Error(),
				"err",
			)
		}
		bodyString := string(bodyBytes)
		resp.Diagnostics.AddError(
			err.Error(),
			bodyString,
		)
		return
	}

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Results.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	edgeApplicationsEdgeFunctionsInstanceState := EdgeFunctionInstanceDataSourceModel{
		ApplicationID: EdgeApplicationId,
		SchemaVersion: types.Int64Value(edgeFunctionInstancesResponse.SchemaVersion),
		Results: EdgeFunctionInstanceResponse{
			ID:             types.Int64Value(edgeFunctionInstancesResponse.Results.GetId()),
			EdgeFunctionID: types.Int64Value(edgeFunctionInstancesResponse.Results.GetEdgeFunctionId()),
			Name:           types.StringValue(edgeFunctionInstancesResponse.Results.GetName()),
			Args:           types.StringValue(jsonArgsStr),
		},
	}

	edgeApplicationsEdgeFunctionsInstanceState.ID = types.StringValue("Get By ID Edge Applications Edge Functions Instances")
	diags := resp.State.Set(ctx, &edgeApplicationsEdgeFunctionsInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
