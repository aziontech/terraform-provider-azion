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
	_ datasource.DataSource              = &ApplicationFunctionInstancesDataSource{}
	_ datasource.DataSourceWithConfigure = &ApplicationFunctionInstancesDataSource{}
)

func dataSourceAzionApplicationFunctionInstances() datasource.DataSource {
	return &ApplicationFunctionInstancesDataSource{}
}

type ApplicationFunctionInstancesDataSource struct {
	client *apiClient
}

type FunctionInstancesDataSourceModel struct {
	ID            types.Int64                `tfsdk:"id"`
	ApplicationID types.Int64                `tfsdk:"application_id"`
	Page          types.Int64                `tfsdk:"page"`
	PageSize      types.Int64                `tfsdk:"page_size"`
	TotalCount    types.Int64                `tfsdk:"total_count"`
	Results       []FunctionInstanceResponse `tfsdk:"results"`
}

type FunctionInstanceResponse struct {
	ID           types.Int64  `tfsdk:"id"`
	FunctionID   types.Int64  `tfsdk:"function_id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (d *ApplicationFunctionInstancesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *ApplicationFunctionInstancesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_function_instances"
}

func (d *ApplicationFunctionInstancesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Application.",
				Required:    true,
			},
			"page": schema.Int64Attribute{
				Description: "Page number for pagination.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "Number of items per page.",
				Optional:    true,
			},
			"total_count": schema.Int64Attribute{
				Description: "The total number of function instances.",
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
							Description: "The function identifier.",
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
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the function instance.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ApplicationFunctionInstancesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var applicationID types.Int64

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &applicationID)
	resp.Diagnostics.Append(diagsApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	if page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	functionInstancesResponse, response, err := d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionInstancesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedFunctionInstanceList, *http.Response, error) {
				return d.client.api.ApplicationsFunctionAPI.ListApplicationFunctionInstances(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute() //nolint
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

	state := FunctionInstancesDataSourceModel{
		ApplicationID: applicationID,
		Page:          page,
		PageSize:      pageSize,
		TotalCount:    types.Int64Value(functionInstancesResponse.GetCount()),
	}

	for _, result := range functionInstancesResponse.GetResults() {
		jsonArgsStr, err := utils.ConvertInterfaceToString(result.GetArgs())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"error reading args from response",
			)
		}
		state.Results = append(state.Results, FunctionInstanceResponse{
			ID:           types.Int64Value(result.GetId()),
			FunctionID:   types.Int64Value(result.GetFunction()),
			Name:         types.StringValue(result.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Active:       types.BoolValue(result.GetActive()),
			LastEditor:   types.StringValue(result.GetLastEditor()),
			LastModified: types.StringValue(result.GetLastModified().Format(time.RFC3339)),
			CreatedAt:    types.StringValue(result.GetCreatedAt().Format(time.RFC3339)),
		})
	}

	state.ID = types.Int64Value(0)
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
