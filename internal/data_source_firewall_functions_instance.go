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

func dataSourceAzionFirewallFunctionsInstance() datasource.DataSource {
	return &FirewallFunctionsInstanceDataSource{}
}

type FirewallFunctionsInstanceDataSource struct {
	client *apiClient
}

type FirewallFunctionsInstanceDataSourceModel struct {
	ID         types.Int64                       `tfsdk:"id"`
	FirewallID types.Int64                       `tfsdk:"firewall_id"`
	Counter    types.Int64                       `tfsdk:"counter"`
	Page       types.Int64                       `tfsdk:"page"`
	PageSize   types.Int64                       `tfsdk:"page_size"`
	TotalPages types.Int64                       `tfsdk:"total_pages"`
	Results    []FirewallFunctionInstanceResults `tfsdk:"results"`
}

type FirewallFunctionInstanceResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (f *FirewallFunctionsInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	f.client = req.ProviderData.(*apiClient)
}

func (f *FirewallFunctionsInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_functions_instance"
}

func (f *FirewallFunctionsInstanceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"firewall_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Firewall",
				Required:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of firewall function instances.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of firewall function instances.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of firewall function instances.",
				Optional:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the firewall function instance.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (f *FirewallFunctionsInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page types.Int64
	var pageSize types.Int64
	var firewallID types.Int64

	diagsFirewallId := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallId...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	if page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	firewallFunctionInstancesResponse, response, err := f.client.api.FirewallsFunctionAPI.
		ListFirewallFunction(ctx, firewallID.ValueInt64()).
		Page(page.ValueInt64()).
		PageSize(pageSize.ValueInt64()).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			firewallFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*sdk.PaginatedFirewallFunctionInstanceList, *http.Response, error) {
				return f.client.api.FirewallsFunctionAPI.
					ListFirewallFunction(ctx, firewallID.ValueInt64()).Page(page.ValueInt64()).
					PageSize(pageSize.ValueInt64()).Execute() //nolint
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

	var functionInstancesResults []FirewallFunctionInstanceResults
	for _, result := range firewallFunctionInstancesResponse.GetResults() {
		jsonArgsStr, err := utils.ConvertInterfaceToString(result.GetArgs())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
		}

		functionInstance := FirewallFunctionInstanceResults{
			ID:           types.Int64Value(result.GetId()),
			Name:         types.StringValue(result.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(result.GetFunction()),
			Active:       types.BoolValue(result.GetActive()),
			LastEditor:   types.StringValue(result.GetLastEditor()),
			LastModified: types.StringValue(result.GetLastModified().Format(time.RFC3339)),
			CreatedAt:    types.StringValue(result.GetCreatedAt().Format(time.RFC3339)),
		}
		functionInstancesResults = append(functionInstancesResults, functionInstance)
	}

	state := FirewallFunctionsInstanceDataSourceModel{
		ID:         firewallID,
		FirewallID: firewallID,
		Counter:    types.Int64Value(firewallFunctionInstancesResponse.GetCount()),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: types.Int64Value(firewallFunctionInstancesResponse.GetTotalPages()),
		Results:    functionInstancesResults,
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
