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
	_ datasource.DataSource              = &WorkloadDeploymentDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkloadDeploymentDataSource{}
)

func dataSourceAzionWorkloadDeployment() datasource.DataSource {
	return &WorkloadDeploymentDataSource{}
}

type WorkloadDeploymentDataSource struct {
	client *apiClient
}

type WorkloadDeploymentDataSourceModel struct {
	WorkloadID   types.String                   `tfsdk:"workload_id"`
	DeploymentID types.String                   `tfsdk:"deployment_id"`
	Data         WorkloadDeploymentResultsModel `tfsdk:"data"`
	ID           types.String                   `tfsdk:"id"`
}

type WorkloadDeploymentResultsModel struct {
	ID           types.Int64              `tfsdk:"id"`
	Name         types.String             `tfsdk:"name"`
	Current      types.Bool               `tfsdk:"current"`
	Active       types.Bool               `tfsdk:"active"`
	Strategy     *DeploymentStrategyModel `tfsdk:"strategy"`
	LastEditor   types.String             `tfsdk:"last_editor"`
	LastModified types.String             `tfsdk:"last_modified"`
	CreatedAt    types.String             `tfsdk:"created_at"`
}

type DeploymentStrategyModel struct {
	Type       types.String                  `tfsdk:"type"`
	Attributes *DeploymentStrategyAttrsModel `tfsdk:"attributes"`
}

type DeploymentStrategyAttrsModel struct {
	Application types.Int64 `tfsdk:"application"`
	Firewall    types.Int64 `tfsdk:"firewall"`
	CustomPage  types.Int64 `tfsdk:"custom_page"`
}

func (d *WorkloadDeploymentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *WorkloadDeploymentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_deployment"
}

func (d *WorkloadDeploymentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"workload_id": schema.StringAttribute{
				Description: "Numeric identifier of the Workload.",
				Required:    true,
			},
			"deployment_id": schema.StringAttribute{
				Description: "Numeric identifier of the Deployment.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
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
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the deployment.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *WorkloadDeploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getWorkloadId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("workload_id"), &getWorkloadId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var getDeploymentId types.String
	diags = req.Config.GetAttribute(ctx, path.Root("deployment_id"), &getDeploymentId)
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

	deploymentID, err := strconv.ParseInt(getDeploymentId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			"Could not convert deployment_id",
		)
		return
	}

	deploymentResponse, response, err := d.client.api.WorkloadDeploymentsAPI.
		RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			deploymentResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
				return d.client.api.WorkloadDeploymentsAPI.RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute() //nolint
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
			usrMsg, errMsg := errPrintWorkloadDeployment(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	deploymentState := WorkloadDeploymentDataSourceModel{
		WorkloadID:   getWorkloadId,
		DeploymentID: getDeploymentId,
		Data: WorkloadDeploymentResultsModel{
			ID:           types.Int64Value(deploymentResponse.Data.Id),
			Name:         types.StringValue(deploymentResponse.Data.Name),
			LastEditor:   types.StringValue(deploymentResponse.Data.LastEditor),
			LastModified: types.StringValue(deploymentResponse.Data.LastModified.Format(time.RFC850)),
			CreatedAt:    types.StringValue(deploymentResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		},
	}

	// Set optional fields
	if deploymentResponse.Data.Current != nil {
		deploymentState.Data.Current = types.BoolValue(*deploymentResponse.Data.Current)
	}

	if deploymentResponse.Data.Active != nil {
		deploymentState.Data.Active = types.BoolValue(*deploymentResponse.Data.Active)
	}

	// Handle Strategy configuration
	strategy := deploymentResponse.Data.GetStrategy()
	strategyModel := &DeploymentStrategyModel{
		Type: types.StringValue(strategy.GetType()),
	}

	attrs := strategy.GetAttributes()
	strategyAttrsModel := &DeploymentStrategyAttrsModel{
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
	deploymentState.Data.Strategy = strategyModel

	deploymentState.ID = types.StringValue("Get By Id Workload Deployment")
	diags = resp.State.Set(ctx, &deploymentState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintWorkloadDeployment(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Workload Deployment found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
