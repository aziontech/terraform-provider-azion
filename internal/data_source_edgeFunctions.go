package provider

import (
	"context"
	"fmt"
	"net/http"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &EdgeFunctionsDataSource{}
	_ datasource.DataSourceWithConfigure = &EdgeFunctionsDataSource{}
)

func dataSourceAzionEdgeFunctions() datasource.DataSource {
	return &EdgeFunctionsDataSource{}
}

type EdgeFunctionsDataSource struct {
	client *apiClient
}

type EdgeFunctionsDataSourceModel struct {
	Counter types.Int64            `tfsdk:"counter"`
	Results []EdgeFunctionsResults `tfsdk:"results"`
	ID      types.String           `tfsdk:"id"`
}

type GetEdgeFunctionsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFunctionsResults struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	LastEditor           types.String `tfsdk:"last_editor"`
	LastModified         types.String `tfsdk:"last_modified"`
	ProductVersion       types.String `tfsdk:"product_version"`
	Active               types.Bool   `tfsdk:"active"`
	Runtime              types.String `tfsdk:"runtime"`
	ExecutionEnvironment types.String `tfsdk:"execution_environment"`
	Code                 types.String `tfsdk:"code"`
	DefaultArgs          types.String `tfsdk:"default_args"`
	ReferenceCount       types.Int64  `tfsdk:"reference_count"`
	Version              types.String `tfsdk:"version"`
	Vendor               types.String `tfsdk:"vendor"`
}

func (d *EdgeFunctionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *EdgeFunctionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_functions"
}

func (d *EdgeFunctionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of edge functions.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
		},
	}
}

func (d *EdgeFunctionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	functionsResponse, response, err := d.client.api.FunctionsAPI.ListFunctions(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedFunctionsList, *http.Response, error) {
				return d.client.api.FunctionsAPI.ListFunctions(ctx).Execute() //nolint
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
			usrMsg, errMsg := errPrintFunctions(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	edgeFunctionsState := EdgeFunctionsDataSourceModel{
		Counter: types.Int64Value(*functionsResponse.Count),
	}

	for _, resultEdgeFunctions := range functionsResponse.GetResults() {
		defaultArgsStr := ""
		if resultEdgeFunctions.DefaultArgs != nil {
			var err error
			defaultArgsStr, err = utils.ConvertInterfaceToString(resultEdgeFunctions.DefaultArgs)
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"Failed to convert default_args to string",
				)
				return
			}
		}

		result := EdgeFunctionsResults{
			ID:             types.Int64Value(resultEdgeFunctions.Id),
			Name:           types.StringValue(resultEdgeFunctions.Name),
			Code:           types.StringValue(resultEdgeFunctions.Code),
			DefaultArgs:    types.StringValue(defaultArgsStr),
			Active:         types.BoolValue(*resultEdgeFunctions.Active),
			LastEditor:     types.StringValue(resultEdgeFunctions.LastEditor),
			ProductVersion: types.StringValue(resultEdgeFunctions.ProductVersion),
			Version:        types.StringValue(resultEdgeFunctions.Version),
			Vendor:         types.StringValue(resultEdgeFunctions.Vendor),
			ReferenceCount: types.Int64Value(resultEdgeFunctions.ReferenceCount),
		}

		// Set optional fields if they exist in the response
		if resultEdgeFunctions.Runtime != nil {
			result.Runtime = types.StringValue(*resultEdgeFunctions.Runtime)
		}

		edgeFunctionsState.Results = append(edgeFunctionsState.Results, result)
	}
	edgeFunctionsState.ID = types.StringValue("Get All Edge Functions")
	diags := resp.State.Set(ctx, &edgeFunctionsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintFunctions(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Edge Functions found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
