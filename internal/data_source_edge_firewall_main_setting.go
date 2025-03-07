package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/edgefirewall"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func dataSourceAzionEdgeFirewall() datasource.DataSource {
	return &EdgeFirewallDataSource{}
}

type EdgeFirewallDataSource struct {
	client *apiClient
}

type EdgeFirewallDataSourceModel struct {
	ID             types.String        `tfsdk:"id"`
	EdgeFirewallID types.Int64         `tfsdk:"edge_firewall_id"`
	SchemaVersion  types.Int64         `tfsdk:"schema_version"`
	Results        EdgeFirewallResults `tfsdk:"results"`
}

type EdgeFirewallResults struct {
	ID                       types.Int64  `tfsdk:"id"`
	LastEditor               types.String `tfsdk:"last_editor"`
	LastModified             types.String `tfsdk:"last_modified"`
	Name                     types.String `tfsdk:"name"`
	IsActive                 types.Bool   `tfsdk:"is_active"`
	EdgeFunctionsEnabled     types.Bool   `tfsdk:"edge_functions_enabled"`
	NetworkProtectionEnabled types.Bool   `tfsdk:"network_protection_enabled"`
	WAFEnabled               types.Bool   `tfsdk:"waf_enabled"`
	DebugRules               types.Bool   `tfsdk:"debug_rules"`
	Domains                  types.List   `tfsdk:"domains"`
}

func (e *EdgeFirewallDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	e.client = req.ProviderData.(*apiClient)
}

func (e *EdgeFirewallDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_main_setting"
}

func (e *EdgeFirewallDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"edge_firewall_id": schema.Int64Attribute{
				Description: "The edge firewall identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the edge firewall rule set.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the edge firewall rule set.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the edge firewall rule set.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the edge firewall rule set.",
						Computed:    true,
					},
					"is_active": schema.BoolAttribute{
						Description: "Whether the edge firewall rule set is active.",
						Computed:    true,
					},
					"edge_functions_enabled": schema.BoolAttribute{
						Description: "Whether edge functions are enabled for the rule set.",
						Computed:    true,
					},
					"network_protection_enabled": schema.BoolAttribute{
						Description: "Whether network protection is enabled for the rule set.",
						Computed:    true,
					},
					"waf_enabled": schema.BoolAttribute{
						Description: "Whether Web Application Firewall (WAF) is enabled for the rule set.",
						Computed:    true,
					},
					"debug_rules": schema.BoolAttribute{
						Description: "Whether debug rules are enabled for the rule set.",
						Computed:    true,
					},
					"domains": schema.ListAttribute{
						Computed:    true,
						ElementType: types.Int64Type,
						Description: "List of domains associated with the edge firewall rule set.",
					},
				},
			},
		},
	}
}

func (e *EdgeFirewallDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getEdgeFirewallID types.Int64
	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_firewall_id"), &getEdgeFirewallID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFirewallResponse, response, err := e.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidGet(ctx, getEdgeFirewallID.String()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgefirewall.EdgeFirewallResponse, *http.Response, error) {
				return e.client.edgeFirewallApi.DefaultAPI.EdgeFirewallUuidGet(ctx, getEdgeFirewallID.String()).Execute() //nolint
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

	var sliceInt []types.Int64
	for _, itemsValuesInt := range edgeFirewallResponse.Results.GetDomains() {
		sliceInt = append(sliceInt, types.Int64Value(int64(itemsValuesInt)))
	}

	edgeFirewallResults := EdgeFirewallResults{
		ID:                       types.Int64Value(edgeFirewallResponse.Results.GetId()),
		LastEditor:               types.StringValue(edgeFirewallResponse.Results.GetLastEditor()),
		LastModified:             types.StringValue(edgeFirewallResponse.Results.GetLastModified()),
		Name:                     types.StringValue(edgeFirewallResponse.Results.GetName()),
		IsActive:                 types.BoolValue(edgeFirewallResponse.Results.GetIsActive()),
		EdgeFunctionsEnabled:     types.BoolValue(edgeFirewallResponse.Results.GetEdgeFunctionsEnabled()),
		NetworkProtectionEnabled: types.BoolValue(edgeFirewallResponse.Results.GetNetworkProtectionEnabled()),
		WAFEnabled:               types.BoolValue(edgeFirewallResponse.Results.GetWafEnabled()),
		DebugRules:               types.BoolValue(edgeFirewallResponse.Results.GetDebugRules()),
		Domains:                  utils.SliceIntTypeToList(sliceInt),
	}

	edgeFirewallState := EdgeFirewallDataSourceModel{
		EdgeFirewallID: getEdgeFirewallID,
		SchemaVersion:  types.Int64Value(int64(edgeFirewallResponse.GetSchemaVersion())),
		Results:        edgeFirewallResults,
	}

	edgeFirewallState.ID = types.StringValue("Get By Id Edge Firewall")
	diags := resp.State.Set(ctx, &edgeFirewallState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
