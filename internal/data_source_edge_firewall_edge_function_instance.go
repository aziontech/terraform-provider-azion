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

func dataSourceAzionFirewallFunctionInstance() datasource.DataSource {
	return &FirewallFunctionInstanceDataSource{}
}

type FirewallFunctionInstanceDataSource struct {
	client *apiClient
}

type FirewallFunctionInstanceDataSourceModel struct {
	ID         types.Int64                  `tfsdk:"id"`
	FirewallID types.Int64                  `tfsdk:"firewall_id"`
	Data       FirewallFunctionInstanceData `tfsdk:"data"`
}

type FirewallFunctionInstanceData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (f *FirewallFunctionInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	f.client = req.ProviderData.(*apiClient)
}

func (f *FirewallFunctionInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_function_instance"
}

func (f *FirewallFunctionInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "ID of the firewall function instance to retrieve.",
				Required:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Identifier of the Firewall",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "ID of the firewall function instance.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the firewall function instance.",
						Computed:    true,
					},
					"args": schema.StringAttribute{
						Description: "Arguments for the function instance.",
						Computed:    true,
					},
					"function": schema.Int64Attribute{
						Description: "ID of the Function for Firewall you wish to configure.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the function instance is active.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the firewall function instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the firewall function instance.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (f *FirewallFunctionInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var firewallID types.Int64
	var functionInstanceID types.Int64

	diagsFirewallId := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallId...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsFunctionInstanceId := req.Config.GetAttribute(ctx, path.Root("id"), &functionInstanceID)
	resp.Diagnostics.Append(diagsFunctionInstanceId...)
	if resp.Diagnostics.HasError() {
		return
	}

	firewallFunctionInstanceResponse, response, err := f.client.api.FirewallsFunctionAPI.
		RetrieveFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallFunctionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return f.client.api.FirewallsFunctionAPI.
					RetrieveFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(firewallFunctionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	data := FirewallFunctionInstanceData{
		ID:           types.Int64Value(firewallFunctionInstanceResponse.Data.GetId()),
		Name:         types.StringValue(firewallFunctionInstanceResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(firewallFunctionInstanceResponse.Data.GetFunction()),
		Active:       types.BoolValue(firewallFunctionInstanceResponse.Data.GetActive()),
		LastEditor:   types.StringValue(firewallFunctionInstanceResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(firewallFunctionInstanceResponse.Data.GetLastModified().Format(time.RFC3339)),
	}

	state := FirewallFunctionInstanceDataSourceModel{
		ID:         functionInstanceID,
		FirewallID: firewallID,
		Data:       data,
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
