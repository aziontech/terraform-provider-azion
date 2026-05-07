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
	_ datasource.DataSource              = &functionDataSource{}
	_ datasource.DataSourceWithConfigure = &functionDataSource{}
)

func dataSourceAzionFunction() datasource.DataSource {
	return &functionDataSource{}
}

type functionDataSource struct {
	client *apiClient
}

type functionDataSourceModel struct {
	Data functionResults `tfsdk:"data"`
	ID   types.String        `tfsdk:"id"`
}

type GetFunctionResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type functionResults struct {
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

func (d *functionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *functionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (d *functionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Required:    true,
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

func (d *functionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getFunctionId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getFunctionId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	functionID, err := strconv.ParseInt(getFunctionId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	functionsResponse, response, err := d.client.api.FunctionsAPI.
		RetrieveFunction(ctx, functionID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionsResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionResponse, *http.Response, error) {
				return d.client.api.FunctionsAPI.RetrieveFunction(ctx, functionID).Execute() //nolint
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
	if functionsResponse.Data.DefaultArgs != nil {
		var err error
		defaultArgsStr, err = utils.ConvertInterfaceToString(functionsResponse.Data.DefaultArgs)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to convert default_args to string",
			)
			return
		}
	}

	FunctionState := functionDataSourceModel{
		Data: functionResults{
			ID:                   types.Int64Value(functionsResponse.Data.Id),
			Name:                 types.StringValue(functionsResponse.Data.Name),
			Code:                 types.StringValue(functionsResponse.Data.Code),
			DefaultArgs:          types.StringValue(defaultArgsStr),
			ExecutionEnvironment: types.StringValue(*functionsResponse.Data.ExecutionEnvironment),
			Active:               types.BoolValue(*functionsResponse.Data.Active),
			LastEditor:           types.StringValue(functionsResponse.Data.LastEditor),
			LastModified:         types.StringValue(functionsResponse.Data.LastModified.Format(time.RFC850)),
			ProductVersion:       types.StringValue(functionsResponse.Data.ProductVersion),
			Version:              types.StringValue(functionsResponse.Data.Version),
			Vendor:               types.StringValue(functionsResponse.Data.Vendor),
			ReferenceCount:       types.Int64Value(functionsResponse.Data.ReferenceCount),
		},
	}

	if functionsResponse.Data.Runtime != nil {
		FunctionState.Data.Runtime = types.StringValue(*functionsResponse.Data.Runtime)
	}

	FunctionState.ID = types.StringValue("Get By Id Function")
	diags = resp.State.Set(ctx, &FunctionState)
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
		usrMsg = "No Function found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
