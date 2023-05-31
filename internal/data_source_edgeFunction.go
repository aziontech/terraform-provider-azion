package provider

import (
	"context"
	"io"
	"strconv"

	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &EdgeFunctionDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeFunctionDataSource{}
)

func dataSourceAzionEdgeFunction() datasource.DataSource {
	return &EdgeFunctionDataSource{}
}

type EdgeFunctionDataSource struct {
	client *apiClient
}

type EdgeFunctionDataSourceModel struct {
	SchemaVersion types.Int64         `tfsdk:"schema_version"`
	Results       EdgeFunctionResults `tfsdk:"results"`
	ID            types.String        `tfsdk:"id"`
}

type GetEdgeFunctionResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFunctionResults struct {
	FunctionID     types.Int64  `tfsdk:"function_id"`
	Name           types.String `tfsdk:"name"`
	Language       types.String `tfsdk:"language"`
	Code           types.String `tfsdk:"code"`
	JSONArgs       types.Map    `tfsdk:"json_args"`
	FunctionToRun  types.String `tfsdk:"function_to_run"`
	InitiatorType  types.String `tfsdk:"initiator_type"`
	IsActive       types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	Modified       types.String `tfsdk:"modified"`
	ReferenceCount types.Int64  `tfsdk:"reference_count"`
	Version        types.String `tfsdk:"version"`
}

func (d *EdgeFunctionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *EdgeFunctionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_function"
}

func (d *EdgeFunctionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Optional:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"function_id": schema.Int64Attribute{
						Description: "The function identifier.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Computed:    true,
					},
					"language": schema.StringAttribute{
						Description: "Language of the function.",
						Computed:    true,
					},
					"code": schema.StringAttribute{
						Description: "Code of the function.",
						Computed:    true,
					},
					"json_args": schema.MapAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "JSON arguments of the function.",
					},
					"function_to_run": schema.StringAttribute{
						Description: "The function to run.",
						Computed:    true,
					},
					"initiator_type": schema.StringAttribute{
						Description: "Initiator type of the function.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the function.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the function.",
						Computed:    true,
					},
					"modified": schema.StringAttribute{
						Description: "Last modified timestamp of the function.",
						Computed:    true,
					},
					"reference_count": schema.Int64Attribute{
						Description: "The reference count of the function.",
						Computed:    true,
					},
					"version": schema.StringAttribute{
						Description: "Version of the function.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *EdgeFunctionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getEdgeFunctionId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getEdgeFunctionId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFunctionId, err := strconv.ParseUint(getEdgeFunctionId.ValueString(), 10, 16)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	functionsResponse, response, err := d.client.edgefunctionsApi.EdgeFunctionsApi.EdgeFunctionsIdGet(ctx, int64(edgeFunctionId)).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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
	jsonArgs, _ := utils.ConvertInterfaceToMap(functionsResponse.Results.JsonArgs)
	EdgeFunctionState := EdgeFunctionDataSourceModel{
		SchemaVersion: types.Int64Value(int64(*functionsResponse.SchemaVersion)),
		Results: EdgeFunctionResults{
			FunctionID:    types.Int64Value(*functionsResponse.Results.Id),
			Name:          types.StringValue(*functionsResponse.Results.Name),
			Language:      types.StringValue(*functionsResponse.Results.Language),
			Code:          types.StringValue(*functionsResponse.Results.Code),
			JSONArgs:      utils.MapToTypesMap(jsonArgs),
			InitiatorType: types.StringValue(*functionsResponse.Results.InitiatorType),
			IsActive:      types.BoolValue(*functionsResponse.Results.Active),
			LastEditor:    types.StringValue(*functionsResponse.Results.LastEditor),
			Modified:      types.StringValue(*functionsResponse.Results.Modified),
		},
	}
	if functionsResponse.Results.ReferenceCount != nil {
		EdgeFunctionState.Results.ReferenceCount = types.Int64Value(*functionsResponse.Results.ReferenceCount)
	}
	if functionsResponse.Results.FunctionToRun != nil {
		EdgeFunctionState.Results.FunctionToRun = types.StringValue(*functionsResponse.Results.FunctionToRun)
	}

	EdgeFunctionState.ID = types.StringValue("Get By Id Edge Function")
	diags = resp.State.Set(ctx, &EdgeFunctionState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}
