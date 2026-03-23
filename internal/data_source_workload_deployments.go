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
	_ datasource.DataSource              = &WorkloadDeploymentsDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkloadDeploymentsDataSource{}
)

func dataSourceAzionWorkloadDeployments() datasource.DataSource {
	return &WorkloadDeploymentsDataSource{}
}

type WorkloadDeploymentsDataSource struct {
	client *apiClient
}

type WorkloadDeploymentsDataSourceModel struct {
	WorkloadID       types.String                      `tfsdk:"workload_id"`
	DeploymentsCount types.Int64                       `tfsdk:"deployments_count"`
	Results          []WorkloadDeploymentsResultsModel `tfsdk:"results"`
	ID               types.String                      `tfsdk:"id"`
}

type WorkloadDeploymentsResultsModel struct {
	ID           types.Int64               `tfsdk:"id"`
	Name         types.String              `tfsdk:"name"`
	Current      types.Bool                `tfsdk:"current"`
	Active       types.Bool                `tfsdk:"active"`
	Strategy     *DeploymentsStrategyModel `tfsdk:"strategy"`
	LastEditor   types.String              `tfsdk:"last_editor"`
	LastModified types.String              `tfsdk:"last_modified"`
}

type DeploymentsStrategyModel struct {
	Type       types.String                   `tfsdk:"type"`
	Attributes *DeploymentsStrategyAttrsModel `tfsdk:"attributes"`
}

type DeploymentsStrategyAttrsModel struct {
	Application types.Int64 `tfsdk:"application"`
	Firewall    types.Int64 `tfsdk:"firewall"`
	CustomPage  types.Int64 `tfsdk:"custom_page"`
}

func (d *WorkloadDeploymentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *WorkloadDeploymentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_deployments"
}

func (d *WorkloadDeploymentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"workload_id": schema.StringAttribute{
				Description: "Numeric identifier of the Workload.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"deployments_count": schema.Int64Attribute{
				Description: "The total number of deployments.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The deployment identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the deployment.",
							Computed:    true,
						},
						"current": schema.BoolAttribute{
							Description: "Whether this is the current deployment.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Status of the deployment.",
							Computed:    true,
						},
						"strategy": schema.SingleNestedAttribute{
							Description: "Deployment strategy configuration.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Type of deployment strategy.",
									Computed:    true,
								},
								"attributes": schema.SingleNestedAttribute{
									Description: "Strategy attributes.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"application": schema.Int64Attribute{
											Description: "Application ID for the deployment.",
											Computed:    true,
										},
										"firewall": schema.Int64Attribute{
											Description: "Firewall ID for the deployment.",
											Computed:    true,
										},
										"custom_page": schema.Int64Attribute{
											Description: "Custom page ID for the deployment.",
											Computed:    true,
										},
									},
								},
							},
						},
						"last_editor": schema.StringAttribute{
							Description: "The last editor of the deployment.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the deployment.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *WorkloadDeploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getWorkloadId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("workload_id"), &getWorkloadId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadID, err := strconv.ParseInt(getWorkloadId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			"Could not convert workload_id",
		)
		return
	}

	deploymentsResponse, response, err := d.client.api.WorkloadDeploymentsAPI.
		ListWorkloadDeployments(ctx, workloadID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			deploymentsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedWorkloadDeploymentList, *http.Response, error) {
				return d.client.api.WorkloadDeploymentsAPI.ListWorkloadDeployments(ctx, workloadID).Execute() //nolint
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
			usrMsg, errMsg := errPrintWorkloadDeployments(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	deploymentsState := WorkloadDeploymentsDataSourceModel{
		WorkloadID: getWorkloadId,
	}

	if deploymentsResponse.Count != nil {
		deploymentsState.DeploymentsCount = types.Int64Value(*deploymentsResponse.Count)
	}

	for _, resultDeployment := range deploymentsResponse.GetResults() {
		result := WorkloadDeploymentsResultsModel{
			ID:           types.Int64Value(resultDeployment.Id),
			Name:         types.StringValue(resultDeployment.Name),
			LastEditor:   types.StringValue(resultDeployment.LastEditor),
			LastModified: types.StringValue(resultDeployment.LastModified.Format(time.RFC850)),
		}

		// Set optional fields
		if resultDeployment.Current != nil {
			result.Current = types.BoolValue(*resultDeployment.Current)
		}

		if resultDeployment.Active != nil {
			result.Active = types.BoolValue(*resultDeployment.Active)
		}

		// Handle Strategy configuration
		strategy := resultDeployment.GetStrategy()
		strategyModel := &DeploymentsStrategyModel{
			Type: types.StringValue(strategy.GetType()),
		}

		attrs := strategy.GetAttributes()
		strategyAttrsModel := &DeploymentsStrategyAttrsModel{
			Application: types.Int64Value(attrs.GetApplication()),
		}

		// Handle optional firewall field
		if attrs.Firewall.IsSet() {
			firewall := attrs.Firewall.Get()
			if firewall != nil {
				strategyAttrsModel.Firewall = types.Int64Value(*firewall)
			}
		}

		// Handle optional custom_page field
		if attrs.CustomPage.IsSet() {
			customPage := attrs.CustomPage.Get()
			if customPage != nil {
				strategyAttrsModel.CustomPage = types.Int64Value(*customPage)
			}
		}

		strategyModel.Attributes = strategyAttrsModel
		result.Strategy = strategyModel

		deploymentsState.Results = append(deploymentsState.Results, result)
	}

	deploymentsState.ID = types.StringValue("Get All Workload Deployments")
	diags = resp.State.Set(ctx, &deploymentsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintWorkloadDeployments(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Workload Deployments found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
