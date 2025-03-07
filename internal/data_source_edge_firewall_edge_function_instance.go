package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/edgefunctionsinstance_edgefirewall"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataSourceAzionEdgeFirewallEdgeFunctionInstance() datasource.DataSource {
	return &EdgeFirewallEdgeFunctionInstanceDataSource{}
}

type EdgeFirewallEdgeFunctionInstanceDataSource struct {
	client *apiClient
}

type EdgeFirewallEdgeFunctionInstanceDataSourceModel struct {
	ID             types.String                            `tfsdk:"id"`
	EdgeFirewallID types.Int64                             `tfsdk:"edge_firewall_id"`
	SchemaVersion  types.Int64                             `tfsdk:"schema_version"`
	Results        EdgeFirewallEdgeFunctionInstanceResults `tfsdk:"results"`
}

type EdgeFirewallEdgeFunctionInstanceResults struct {
	ID             types.Int64  `tfsdk:"edge_function_instance_id"`
	LastEditor     types.String `tfsdk:"last_editor"`
	LastModified   types.String `tfsdk:"last_modified"`
	Name           types.String `tfsdk:"name"`
	JsonArgs       types.String `tfsdk:"json_args"`
	EdgeFunctionID types.Int64  `tfsdk:"edge_function_id"`
}

func (e *EdgeFirewallEdgeFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	e.client = req.ProviderData.(*apiClient)
}

func (e *EdgeFirewallEdgeFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_edge_function_instance"
}

func (e *EdgeFirewallEdgeFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"edge_firewall_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Firewall",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"edge_function_instance_id": schema.Int64Attribute{
						Description: "ID of the edge firewall edge functions instance.",
						Required:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the edge firewall edge functions instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the edge firewall edge functions instance.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the edge firewall edge functions instance.",
						Computed:    true,
					},
					"json_args": schema.StringAttribute{
						Description: "Requisition status code and message.",
						Computed:    true,
					},
					"edge_function_id": schema.Int64Attribute{
						Description: "ID of the Edge Function for Edge Firewall you with to configure.",
						Computed:    true},
				},
			},
		},
	}
}

func (e *EdgeFirewallEdgeFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeFirewallID types.Int64
	var edgeFunctionInstanceID types.Int64

	diagsPhase := req.Config.GetAttribute(ctx, path.Root("results").AtName("edge_function_instance_id"), &edgeFunctionInstanceID)
	resp.Diagnostics.Append(diagsPhase...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsEdgeApplicationId := req.Config.GetAttribute(ctx, path.Root("edge_firewall_id"), &edgeFirewallID)
	resp.Diagnostics.Append(diagsEdgeApplicationId...)
	if resp.Diagnostics.HasError() {
		return
	}

	EdgeFirewallFunctionsInstanceResponse, response, err := e.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdGet(ctx, edgeFirewallID.ValueInt64(), edgeFunctionInstanceID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgefunctionsinstance_edgefirewall.EdgeFunctionsInstanceResponse, *http.Response, error) {
				return e.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdGet(ctx, edgeFirewallID.ValueInt64(), edgeFunctionInstanceID.ValueInt64()).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(EdgeFirewallFunctionsInstanceResponse.Results.GetJsonArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	GetEdgeFirewall := EdgeFirewallEdgeFunctionInstanceResults{
		ID:             types.Int64Value(EdgeFirewallFunctionsInstanceResponse.Results.GetId()),
		LastEditor:     types.StringValue(EdgeFirewallFunctionsInstanceResponse.Results.GetLastEditor()),
		LastModified:   types.StringValue(EdgeFirewallFunctionsInstanceResponse.Results.GetLastModified()),
		Name:           types.StringValue(EdgeFirewallFunctionsInstanceResponse.Results.GetName()),
		EdgeFunctionID: types.Int64Value(EdgeFirewallFunctionsInstanceResponse.Results.GetEdgeFunction()),
		JsonArgs:       types.StringValue(jsonArgsStr),
	}

	EdgeFirewallsState := EdgeFirewallEdgeFunctionInstanceDataSourceModel{
		SchemaVersion: types.Int64Value(int64(EdgeFirewallFunctionsInstanceResponse.GetSchemaVersion())),
		Results:       GetEdgeFirewall,
	}
	EdgeFirewallsState.EdgeFirewallID = edgeFirewallID
	EdgeFirewallsState.ID = types.StringValue("Get By Id Edge Firewall Edge Function Instance")

	diags := resp.State.Set(ctx, &EdgeFirewallsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
