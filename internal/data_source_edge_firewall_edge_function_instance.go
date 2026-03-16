package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
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
	ID             types.Int64                             `tfsdk:"id"`
	EdgeFirewallID types.Int64                             `tfsdk:"edge_firewall_id"`
	Data           EdgeFirewallEdgeFunctionInstanceResults `tfsdk:"data"`
}

type EdgeFirewallEdgeFunctionInstanceResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
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
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"edge_firewall_id": schema.Int64Attribute{
				Description: "Identifier of the Edge Firewall",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the edge firewall edge function instance.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the edge firewall edge function instance.",
						Computed:    true,
					},
					"args": schema.StringAttribute{
						Description: "Arguments for the edge function instance.",
						Computed:    true,
					},
					"function": schema.Int64Attribute{
						Description: "ID of the Edge Function for Edge Firewall you wish to configure.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the edge function instance is active.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the edge firewall edge function instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the edge firewall edge function instance.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (e *EdgeFirewallEdgeFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var edgeFirewallID types.Int64
	var edgeFunctionInstanceID types.Int64

	diagsEdgeFirewallId := req.Config.GetAttribute(ctx, path.Root("edge_firewall_id"), &edgeFirewallID)
	resp.Diagnostics.Append(diagsEdgeFirewallId...)
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFunctionInstanceID = types.Int64Value(0)

	EdgeFirewallFunctionsInstanceResponse, response, err := e.client.api.FirewallsFunctionAPI.
		RetrieveFirewallFunction(ctx, edgeFirewallID.ValueInt64(), edgeFunctionInstanceID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			EdgeFirewallFunctionsInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return e.client.api.FirewallsFunctionAPI.
					RetrieveFirewallFunction(ctx, edgeFirewallID.ValueInt64(), edgeFunctionInstanceID.ValueInt64()).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(EdgeFirewallFunctionsInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	GetEdgeFirewall := EdgeFirewallEdgeFunctionInstanceResults{
		ID:           types.Int64Value(EdgeFirewallFunctionsInstanceResponse.Data.GetId()),
		Name:         types.StringValue(EdgeFirewallFunctionsInstanceResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(EdgeFirewallFunctionsInstanceResponse.Data.GetFunction()),
		Active:       types.BoolValue(EdgeFirewallFunctionsInstanceResponse.Data.GetActive()),
		LastEditor:   types.StringValue(EdgeFirewallFunctionsInstanceResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(EdgeFirewallFunctionsInstanceResponse.Data.GetLastModified().Format(time.RFC3339)),
	}

	EdgeFirewallsState := EdgeFirewallEdgeFunctionInstanceDataSourceModel{
		ID:             edgeFirewallID,
		EdgeFirewallID: edgeFirewallID,
		Data:           GetEdgeFirewall,
	}

	diags := resp.State.Set(ctx, &EdgeFirewallsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
