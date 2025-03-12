package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
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
	SchemaVersion types.Int64                    `tfsdk:"schema_version"`
	Counter       types.Int64                    `tfsdk:"counter"`
	TotalPages    types.Int64                    `tfsdk:"total_pages"`
	Links         *GetEdgeFunctionsResponseLinks `tfsdk:"links"`
	Results       []EdgeFunctionsResults         `tfsdk:"results"`
	ID            types.String                   `tfsdk:"id"`
}

type GetEdgeFunctionsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type EdgeFunctionsResults struct {
	FunctionID     types.Int64  `tfsdk:"function_id"`
	Name           types.String `tfsdk:"name"`
	Language       types.String `tfsdk:"language"`
	Code           types.String `tfsdk:"code"`
	JSONArgs       types.String `tfsdk:"json_args"`
	FunctionToRun  types.String `tfsdk:"function_to_run"`
	InitiatorType  types.String `tfsdk:"initiator_type"`
	IsActive       types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	Modified       types.String `tfsdk:"modified"`
	ReferenceCount types.Int64  `tfsdk:"reference_count"`
	Version        types.String `tfsdk:"version"`
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
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of zones.",
				Computed:    true,
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
						"function_id": schema.Int64Attribute{
							Description: "The function identifier.",
							Computed:    true,
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
						"json_args": schema.StringAttribute{
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
		},
	}
}

func (d *EdgeFunctionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	functionsResponse, response, err := d.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsGet(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionsResponse, response, err = utils.RetryOn429(func() (*edgefunctions.ListEdgeFunctionResponse, *http.Response, error) {
				return d.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsGet(ctx).Execute() //nolint
			}, 15) // Maximum 15 retries

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
	}

	var previous, next string
	if functionsResponse.Links != nil {
		if functionsResponse.Links.Previous.Get() != nil {
			previous = *functionsResponse.Links.Previous.Get()
		}
		if functionsResponse.Links.Next.Get() != nil {
			next = *functionsResponse.Links.Next.Get()
		}
	}

	edgeFunctionsState := EdgeFunctionsDataSourceModel{
		SchemaVersion: types.Int64Value(*functionsResponse.SchemaVersion),
		TotalPages:    types.Int64Value(*functionsResponse.TotalPages),
		Counter:       types.Int64Value(*functionsResponse.Count),
		Links: &GetEdgeFunctionsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	for _, resultEdgeFunctions := range functionsResponse.GetResults() {
		jsonArgsStr, err := utils.ConvertInterfaceToString(resultEdgeFunctions.JsonArgs)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
		}
		if resp.Diagnostics.HasError() {
			return
		}
		edgeFunctionsState.Results = append(edgeFunctionsState.Results, EdgeFunctionsResults{
			FunctionID:    types.Int64Value(*resultEdgeFunctions.Id),
			Name:          types.StringValue(*resultEdgeFunctions.Name),
			Language:      types.StringValue(*resultEdgeFunctions.Language),
			Code:          types.StringValue(*resultEdgeFunctions.Code),
			JSONArgs:      types.StringValue(jsonArgsStr),
			InitiatorType: types.StringValue(*resultEdgeFunctions.InitiatorType),
			IsActive:      types.BoolValue(*resultEdgeFunctions.Active),
			LastEditor:    types.StringValue(*resultEdgeFunctions.LastEditor),
			Modified:      types.StringValue(*resultEdgeFunctions.Modified),
		})
		if resultEdgeFunctions.ReferenceCount != nil {
			edgeFunctionsState.Results[len(edgeFunctionsState.Results)-1].ReferenceCount = types.Int64Value(*resultEdgeFunctions.ReferenceCount)
		}
		if resultEdgeFunctions.FunctionToRun != nil {
			edgeFunctionsState.Results[len(edgeFunctionsState.Results)-1].FunctionToRun = types.StringValue(*resultEdgeFunctions.FunctionToRun)
		}
	}
	edgeFunctionsState.ID = types.StringValue("Get All Edge Functions")
	diags := resp.State.Set(ctx, &edgeFunctionsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
