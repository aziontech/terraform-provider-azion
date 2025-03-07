package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
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
	ID            types.String                           `tfsdk:"id"`
	ApplicationID types.Int64                            `tfsdk:"edge_application_id"`
	Counter       types.Int64                            `tfsdk:"counter"`
	Page          types.Int64                            `tfsdk:"page"`
	PageSize      types.Int64                            `tfsdk:"page_size"`
	TotalPages    types.Int64                            `tfsdk:"total_pages"`
	Links         *GetEdgeFunctionsInstanceResponseLinks `tfsdk:"links"`
	SchemaVersion types.Int64                            `tfsdk:"schema_version"`
	Results       []EdgeFunctionsInstanceResponse        `tfsdk:"results"`
}

type GetEdgeFunctionsInstanceResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFunctionsInstanceResponse struct {
	ID             types.Int64  `tfsdk:"id"`
	EdgeFunctionID types.Int64  `tfsdk:"edge_function_id"`
	Name           types.String `tfsdk:"name"`
	Args           types.String `tfsdk:"args"`
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
			"counter": schema.Int64Attribute{
				Description: "The total number of edge function instances.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of edge function instances.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of edge function instances.",
				Optional:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"links": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"previous": schema.StringAttribute{
						Computed: true,
					},
					"next": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The function identifier.",
							Computed:    true,
						},
						"edge_function_id": schema.Int64Attribute{
							Description: "Name of the function.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Language of the function.",
							Computed:    true,
						},
						"args": schema.StringAttribute{
							Description: "Code of the function.",
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

	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &EdgeApplicationId)
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

	edgeFunctionInstancesResponse, response, err := d.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesGet(ctx, EdgeApplicationId.ValueInt64()).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgeapplications.ApplicationInstancesGetResponse, *http.Response, error) {
				return d.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesGet(ctx, EdgeApplicationId.ValueInt64()).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
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

	var previous, next string
	if edgeFunctionInstancesResponse.Links.Previous.Get() != nil {
		previous = *edgeFunctionInstancesResponse.Links.Previous.Get()
	}
	if edgeFunctionInstancesResponse.Links.Next.Get() != nil {
		next = *edgeFunctionInstancesResponse.Links.Next.Get()
	}

	edgeApplicationsEdgeFunctionsInstanceState := EdgeFunctionsInstanceDataSourceModel{
		Page:          Page,
		PageSize:      PageSize,
		ApplicationID: EdgeApplicationId,
		SchemaVersion: types.Int64Value(edgeFunctionInstancesResponse.SchemaVersion),
		TotalPages:    types.Int64Value(edgeFunctionInstancesResponse.TotalPages),
		Counter:       types.Int64Value(edgeFunctionInstancesResponse.Count),
		Links: &GetEdgeFunctionsInstanceResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
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
			ID:             types.Int64Value(resultEdgeApplication.GetId()),
			EdgeFunctionID: types.Int64Value(resultEdgeApplication.GetEdgeFunctionId()),
			Name:           types.StringValue(resultEdgeApplication.GetName()),
			Args:           types.StringValue(jsonArgsStr),
		})
	}

	edgeApplicationsEdgeFunctionsInstanceState.ID = types.StringValue("Get All Edge Applications Edge Functions Instances")
	diags := resp.State.Set(ctx, &edgeApplicationsEdgeFunctionsInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
