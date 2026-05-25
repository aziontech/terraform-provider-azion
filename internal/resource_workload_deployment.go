package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workloadDeploymentResource{}
	_ resource.ResourceWithConfigure   = &workloadDeploymentResource{}
	_ resource.ResourceWithImportState = &workloadDeploymentResource{}
)

func NewWorkloadDeploymentResource() resource.Resource {
	return &workloadDeploymentResource{}
}

type workloadDeploymentResource struct {
	client *apiClient
}

type WorkloadDeploymentResourceModel struct {
	Deployment  *WorkloadDeploymentResourceResults `tfsdk:"deployment"`
	ID          types.String                       `tfsdk:"id"`
	WorkloadID  types.Int64                        `tfsdk:"workload_id"`
	LastUpdated types.String                       `tfsdk:"last_updated"`
}

type WorkloadDeploymentResourceResults struct {
	ID           types.Int64                      `tfsdk:"id"`
	Name         types.String                     `tfsdk:"name"`
	Current      types.Bool                       `tfsdk:"current"`
	Active       types.Bool                       `tfsdk:"active"`
	Strategy     *DeploymentStrategyResourceModel `tfsdk:"strategy"`
	LastEditor   types.String                     `tfsdk:"last_editor"`
	LastModified types.String                     `tfsdk:"last_modified"`
	CreatedAt    types.String                     `tfsdk:"created_at"`
}

type DeploymentStrategyResourceModel struct {
	Type       types.String                          `tfsdk:"type"`
	Attributes *DeploymentStrategyAttrsResourceModel `tfsdk:"attributes"`
}

type DeploymentStrategyAttrsResourceModel struct {
	Application types.Int64 `tfsdk:"application"`
	Firewall    types.Int64 `tfsdk:"firewall"`
	CustomPage  types.Int64 `tfsdk:"custom_page"`
}

func (r *workloadDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_deployment"
}

func (r *workloadDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing Azion Workload Deployments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the resource (workloadID/deploymentID format).",
			},
			"workload_id": schema.Int64Attribute{
				Description: "The workload identifier.",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"deployment": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The deployment configuration.",
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The deployment identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the deployment.",
						Required:    true,
					},
					"current": schema.BoolAttribute{
						Description: "Whether this is the current deployment.",
						Optional:    true,
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the deployment.",
						Optional:    true,
						Computed:    true,
					},
					"strategy": schema.SingleNestedAttribute{
						Description: "Deployment strategy configuration.",
						Required:    true,
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Type of deployment strategy.",
								Required:    true,
							},
							"attributes": schema.SingleNestedAttribute{
								Description: "Strategy attributes.",
								Required:    true,
								Attributes: map[string]schema.Attribute{
									"application": schema.Int64Attribute{
										Description: "Application ID for the deployment.",
										Required:    true,
									},
									"firewall": schema.Int64Attribute{
										Description: "Firewall ID for the deployment.",
										Optional:    true,
									},
									"custom_page": schema.Int64Attribute{
										Description: "Custom page ID for the deployment.",
										Optional:    true,
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
						Description: "Creation timestamp of the deployment.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *workloadDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *workloadDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkloadDeploymentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the strategy request
	strategyAttrs := azionapi.NewDefaultDeploymentStrategyAttrsRequest(plan.Deployment.Strategy.Attributes.Application.ValueInt64())

	// Handle optional firewall field
	if !plan.Deployment.Strategy.Attributes.Firewall.IsNull() && !plan.Deployment.Strategy.Attributes.Firewall.IsUnknown() {
		strategyAttrs.SetFirewall(plan.Deployment.Strategy.Attributes.Firewall.ValueInt64())
	}

	// Handle optional custom_page field
	if !plan.Deployment.Strategy.Attributes.CustomPage.IsNull() && !plan.Deployment.Strategy.Attributes.CustomPage.IsUnknown() {
		strategyAttrs.SetCustomPage(plan.Deployment.Strategy.Attributes.CustomPage.ValueInt64())
	}

	strategy := azionapi.NewDeploymentStrategyDefaultDeploymentStrategyRequest(
		plan.Deployment.Strategy.Type.ValueString(),
		*strategyAttrs,
	)

	// Build the deployment request
	deploymentRequest := azionapi.NewWorkloadDeploymentRequest(
		plan.Deployment.Name.ValueString(),
		*strategy,
	)

	// Set optional fields
	if !plan.Deployment.Current.IsNull() && !plan.Deployment.Current.IsUnknown() {
		deploymentRequest.SetCurrent(plan.Deployment.Current.ValueBool())
	}

	if !plan.Deployment.Active.IsNull() && !plan.Deployment.Active.IsUnknown() {
		deploymentRequest.SetActive(plan.Deployment.Active.ValueBool())
	}

	// Create the deployment
	createDeployment, response, err := r.client.api.WorkloadDeploymentsAPI.
		CreateWorkloadDeployment(ctx, plan.WorkloadID.ValueInt64()).
		WorkloadDeploymentRequest(*deploymentRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			createDeployment, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
				return r.client.api.WorkloadDeploymentsAPI.
					CreateWorkloadDeployment(ctx, plan.WorkloadID.ValueInt64()).
					WorkloadDeploymentRequest(*deploymentRequest).Execute()
			}, 5)

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
	if response != nil {
		defer response.Body.Close()
	}

	// Populate the state from the response
	plan.Deployment = populateDeploymentResults(createDeployment)
	plan.ID = types.StringValue(fmt.Sprintf("%d/%d", plan.WorkloadID.ValueInt64(), createDeployment.Data.Id))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkloadDeploymentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var workloadID int64
	var deploymentID int64

	// ID can be either just the deployment ID or "workloadID/deploymentID" format for import
	idStr := state.ID.ValueString()
	parts := strings.Split(idStr, "/")
	if len(parts) == 2 {
		wID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid workload ID format", err.Error())
			return
		}
		dID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid deployment ID format", err.Error())
			return
		}
		workloadID = wID
		deploymentID = dID
	} else {
		workloadID = state.WorkloadID.ValueInt64()
		if state.Deployment != nil {
			deploymentID = state.Deployment.ID.ValueInt64()
		}
	}

	if deploymentID == 0 {
		resp.Diagnostics.AddError(
			"Deployment ID error",
			"Deployment ID cannot be zero",
		)
		return
	}

	deploymentResponse, response, err := r.client.api.WorkloadDeploymentsAPI.
		RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			deploymentResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
				return r.client.api.WorkloadDeploymentsAPI.
					RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
			}, 5)

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
	if response != nil {
		defer response.Body.Close()
	}

	state.Deployment = populateDeploymentResults(deploymentResponse)
	state.WorkloadID = types.Int64Value(workloadID)
	state.ID = types.StringValue(fmt.Sprintf("%d/%d", workloadID, deploymentResponse.Data.Id))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkloadDeploymentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkloadDeploymentResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deploymentID := state.Deployment.ID.ValueInt64()
	if plan.Deployment != nil && !plan.Deployment.ID.IsNull() && plan.Deployment.ID.ValueInt64() != 0 {
		deploymentID = plan.Deployment.ID.ValueInt64()
	}

	// Build the patched request
	patchedRequest := azionapi.NewPatchedWorkloadDeploymentRequest()

	if !plan.Deployment.Name.IsNull() && !plan.Deployment.Name.IsUnknown() {
		patchedRequest.SetName(plan.Deployment.Name.ValueString())
	}

	if !plan.Deployment.Current.IsNull() && !plan.Deployment.Current.IsUnknown() {
		patchedRequest.SetCurrent(plan.Deployment.Current.ValueBool())
	}

	if !plan.Deployment.Active.IsNull() && !plan.Deployment.Active.IsUnknown() {
		patchedRequest.SetActive(plan.Deployment.Active.ValueBool())
	}

	// Build strategy if provided
	if plan.Deployment.Strategy != nil {
		strategyAttrs := azionapi.NewDefaultDeploymentStrategyAttrsRequest(
			plan.Deployment.Strategy.Attributes.Application.ValueInt64(),
		)

		// On PATCH, an omitted field is preserved by the API. To let users clear
		// a previously-set firewall by removing it from their config, send an
		// explicit null when the planned value is null.
		switch {
		case plan.Deployment.Strategy.Attributes.Firewall.IsUnknown():
			// Unknown values are not yet resolved; skip.
		case plan.Deployment.Strategy.Attributes.Firewall.IsNull():
			strategyAttrs.SetFirewallNil()
		default:
			strategyAttrs.SetFirewall(plan.Deployment.Strategy.Attributes.Firewall.ValueInt64())
		}

		switch {
		case plan.Deployment.Strategy.Attributes.CustomPage.IsUnknown():
		case plan.Deployment.Strategy.Attributes.CustomPage.IsNull():
			strategyAttrs.SetCustomPageNil()
		default:
			strategyAttrs.SetCustomPage(plan.Deployment.Strategy.Attributes.CustomPage.ValueInt64())
		}

		strategy := azionapi.NewDeploymentStrategyDefaultDeploymentStrategyRequest(
			plan.Deployment.Strategy.Type.ValueString(),
			*strategyAttrs,
		)
		patchedRequest.SetStrategy(*strategy)
	}

	updateResponse, response, err := r.client.api.WorkloadDeploymentsAPI.
		PartialUpdateWorkloadDeployment(ctx, deploymentID, plan.WorkloadID.ValueInt64()).
		PatchedWorkloadDeploymentRequest(*patchedRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			updateResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
				return r.client.api.WorkloadDeploymentsAPI.
					PartialUpdateWorkloadDeployment(ctx, deploymentID, plan.WorkloadID.ValueInt64()).
					PatchedWorkloadDeploymentRequest(*patchedRequest).Execute()
			}, 5)

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
	if response != nil {
		defer response.Body.Close()
	}

	plan.Deployment = populateDeploymentResults(updateResponse)
	plan.ID = types.StringValue(fmt.Sprintf("%d/%d", plan.WorkloadID.ValueInt64(), updateResponse.Data.Id))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkloadDeploymentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Deployment == nil || state.Deployment.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Deployment ID error",
			"Deployment ID is required for deletion",
		)
		return
	}

	if state.WorkloadID.IsNull() {
		resp.Diagnostics.AddError(
			"Workload ID error",
			"Workload ID is required for deletion",
		)
		return
	}

	_, response, err := r.client.api.WorkloadDeploymentsAPI.
		DeleteWorkloadDeployment(ctx, state.Deployment.ID.ValueInt64(), state.WorkloadID.ValueInt64()).Execute()
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			// Resource already deleted, consider this a success
			return
		}
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.WorkloadDeploymentsAPI.
					DeleteWorkloadDeployment(ctx, state.Deployment.ID.ValueInt64(), state.WorkloadID.ValueInt64()).Execute()
			}, 5)

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
	if response != nil {
		defer response.Body.Close()
	}
}

func (r *workloadDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "workloadID/deploymentID"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Expected format: workloadID/deploymentID",
		)
		return
	}

	workloadID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid workload ID format",
			err.Error(),
		)
		return
	}

	deploymentID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid deployment ID format",
			err.Error(),
		)
		return
	}

	// Read the deployment to populate state
	deploymentResponse, response, err := r.client.api.WorkloadDeploymentsAPI.
		RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			deploymentResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadDeploymentResponse, *http.Response, error) {
				return r.client.api.WorkloadDeploymentsAPI.
					RetrieveWorkloadDeployment(ctx, deploymentID, workloadID).Execute()
			}, 5)

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
	if response != nil {
		defer response.Body.Close()
	}

	state := WorkloadDeploymentResourceModel{
		WorkloadID: types.Int64Value(workloadID),
		ID:         types.StringValue(req.ID),
		Deployment: populateDeploymentResults(deploymentResponse),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateDeploymentResults populates the deployment results model from the API response.
func populateDeploymentResults(response *azionapi.WorkloadDeploymentResponse) *WorkloadDeploymentResourceResults {
	result := &WorkloadDeploymentResourceResults{
		ID:           types.Int64Value(response.Data.Id),
		Name:         types.StringValue(response.Data.Name),
		LastEditor:   types.StringValue(response.Data.LastEditor),
		LastModified: types.StringValue(response.Data.LastModified.Format(time.RFC850)),
		CreatedAt:    types.StringValue(response.Data.CreatedAt.Format(time.RFC3339)),
	}

	// Set optional fields
	if response.Data.Current != nil {
		result.Current = types.BoolValue(*response.Data.Current)
	}

	if response.Data.Active != nil {
		result.Active = types.BoolValue(*response.Data.Active)
	}

	// Handle Strategy configuration
	strategy := response.Data.GetStrategy()
	strategyModel := &DeploymentStrategyResourceModel{
		Type: types.StringValue(strategy.GetType()),
	}

	attrs := strategy.GetAttributes()
	strategyAttrsModel := &DeploymentStrategyAttrsResourceModel{
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

	return result
}
