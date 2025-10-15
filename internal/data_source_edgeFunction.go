package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
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
	Data          EdgeFunctionResults `tfsdk:"data"`
	ID            types.String        `tfsdk:"id"`
}

type GetEdgeFunctionResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFunctionResults struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	LastEditor           types.String `tfsdk:"last_editor"`
	LastModified         types.String `tfsdk:"last_modified"`
	ProductVersion       types.String `tfsdk:"product_version"`
	IsActive             types.Bool   `tfsdk:"active"`
	Runtime              types.String `tfsdk:"runtime"`
	ExecutionEnvironment types.String `tfsdk:"execution_environment"`
	Code                 types.String `tfsdk:"code"`
	DefaultArgs          types.String `tfsdk:"default_args"`
	ReferenceCount       types.Int64  `tfsdk:"reference_count"`
	Version              types.String `tfsdk:"version"`
	Vendor               types.String `tfsdk:"vendor"`
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
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The function identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the function.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the function.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the function.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the function.",
						Computed:    true,
					},
					"runtime": schema.StringAttribute{
						Description: "Runtime of the function.",
						Computed:    true,
					},
					"execution_environment": schema.StringAttribute{
						Description: "Execution environment of the function.",
						Computed:    true,
					},
					"code": schema.StringAttribute{
						Description: "Code of the function.",
						Computed:    true,
					},
					"default_args": schema.StringAttribute{
						Description: "Default arguments of the function as JSON.",
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
					"vendor": schema.StringAttribute{
						Description: "Vendor of the function.",
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

	edgeFunctionID, err := strconv.ParseInt(getEdgeFunctionId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	functionsResponse, response, err := d.client.edgefunctionsApi.EdgeFunctionsAPI.
		EdgeFunctionsIdGet(ctx, edgeFunctionID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionsResponse, response, err = utils.RetryOn429(func() (*edgefunctions.EdgeFunctionResponse, *http.Response, error) {
				return d.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsIdGet(ctx, edgeFunctionID).Execute() //nolint
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
			usrMsg, errMsg := errPrint(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	defaultArgsStr := ""
	if functionsResponse.Results.JsonArgs != nil {
		var err error
		defaultArgsStr, err = utils.ConvertInterfaceToString(functionsResponse.Results.JsonArgs)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to convert default_args to string",
			)
			return
		}
	}

	EdgeFunctionState := EdgeFunctionDataSourceModel{
		SchemaVersion: types.Int64Value(int64(*functionsResponse.SchemaVersion)),
		Data: EdgeFunctionResults{
			ID:                   types.Int64Value(*functionsResponse.Results.Id),
			Name:                 types.StringValue(*functionsResponse.Results.Name),
			LastEditor:           types.StringValue(*functionsResponse.Results.LastEditor),
			LastModified:         types.StringValue(*functionsResponse.Results.Modified),
			IsActive:             types.BoolValue(*functionsResponse.Results.Active),
			Code:                 types.StringValue(*functionsResponse.Results.Code),
			DefaultArgs:          types.StringValue(defaultArgsStr),
			ReferenceCount:       types.Int64Value(int64(0)),
			ProductVersion:       types.StringValue(""),
			Runtime:              types.StringValue(""),
			ExecutionEnvironment: types.StringValue(""),
			Version:              types.StringValue(""),
			Vendor:               types.StringValue(""),
		},
	}

	// Set optional fields if they exist in the response
	if functionsResponse.Results.ReferenceCount != nil {
		EdgeFunctionState.Data.ReferenceCount = types.Int64Value(*functionsResponse.Results.ReferenceCount)
	}
	// Map old API fields to new schema fields
	if functionsResponse.Results.Language != nil {
		EdgeFunctionState.Data.Runtime = types.StringValue(*functionsResponse.Results.Language)
	}
	if functionsResponse.Results.InitiatorType != nil {
		EdgeFunctionState.Data.ExecutionEnvironment = types.StringValue(*functionsResponse.Results.InitiatorType)
	}

	EdgeFunctionState.ID = types.StringValue("Get By Id Edge Function")
	diags = resp.State.Set(ctx, &EdgeFunctionState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrint(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Edge Function found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
