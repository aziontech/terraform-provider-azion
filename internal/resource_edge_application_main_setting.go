package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &edgeApplicationResource{}
	_ resource.ResourceWithConfigure   = &edgeApplicationResource{}
	_ resource.ResourceWithImportState = &edgeApplicationResource{}
)

func NewEdgeApplicationMainSettingsResource() resource.Resource {
	return &edgeApplicationResource{}
}

type edgeApplicationResource struct {
	client *apiClient
}

type EdgeApplicationResourceModel struct {
	EdgeApplication *EdgeApplicationResults `tfsdk:"edge_application"`
	ID              types.String            `tfsdk:"id"`
	LastUpdated     types.String            `tfsdk:"last_updated"`
}

type EdgeApplicationResults struct {
	ApplicationID  types.Int64         `tfsdk:"application_id"`
	Name           types.String        `tfsdk:"name"`
	Modules        *ApplicationModules `tfsdk:"modules"`
	Active         types.Bool          `tfsdk:"active"`
	Debug          types.Bool          `tfsdk:"debug"`
	ProductVersion types.String        `tfsdk:"product_version"`
}

func (r *edgeApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_main_setting"
}

func (r *edgeApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"edge_application": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"application_id": schema.Int64Attribute{
						Description: "The Edge Application identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the Edge Application.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Optional:    true,
						Description: "Indicates whether the Edge Application is active.",
					},
					"debug": schema.BoolAttribute{
						Optional:    true,
						Description: "Indicates whether debug rules are enabled for the Edge Application.",
					},
					"product_version": schema.StringAttribute{
						Computed:    true,
						Description: "The product version.",
					},
					"modules": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"edge_cache": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Optional: true},
								},
							},
							"functions": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Optional: true},
								},
							},
							"application_accelerator": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Optional: true},
								},
							},
							"image_processor": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{Optional: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *edgeApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

var mutex sync.Mutex

func (r *edgeApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	mutex.Lock()
	defer mutex.Unlock()

	if resp.Diagnostics.HasError() {
		return
	}

	edgeApplication := sdk.ApplicationRequest{
		Name:   plan.EdgeApplication.Name.ValueString(),
		Active: plan.EdgeApplication.Active.ValueBoolPointer(),
		Debug:  plan.EdgeApplication.Debug.ValueBoolPointer(),
	}

	modsPlan := plan.EdgeApplication.Modules
	modsRequest := transformModuleIntoRequest(modsPlan)

	edgeApplication.Modules = &modsRequest

	createEdgeApplication, response, err := r.client.applicationsApi.
		ApplicationsAPI.CreateApplication(ctx).
		ApplicationRequest(edgeApplication).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			createEdgeApplication, response, err = utils.RetryOn429(func() (*sdk.ResponseApplication, *http.Response, error) {
				return r.client.applicationsApi.
					ApplicationsAPI.CreateApplication(ctx).
					ApplicationRequest(edgeApplication).Execute() //nolint
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

	edgeAppResults := &EdgeApplicationResults{
		ApplicationID:  types.Int64Value(createEdgeApplication.Data.GetId()),
		Name:           types.StringValue(createEdgeApplication.Data.GetName()),
		Active:         types.BoolValue(createEdgeApplication.Data.GetActive()),
		Debug:          types.BoolValue(createEdgeApplication.Data.GetDebug()),
		ProductVersion: types.StringValue(createEdgeApplication.Data.GetProductVersion()),
		Modules:        plan.EdgeApplication.Modules,
	}

	if createEdgeApplication.Data.Modules != nil {
		modulesResp := createEdgeApplication.Data.GetModules()
		modules := ApplicationModules{}
		if modulesResp.EdgeCache != nil {
			modules.Cache = &CacheModule{
				Enabled: types.BoolValue(modulesResp.EdgeCache.GetEnabled()),
			}
		}
		if modulesResp.Functions != nil {
			modules.Functions = &EdgeFunctionModule{
				Enabled: types.BoolValue(modulesResp.Functions.GetEnabled()),
			}
		}
		if modulesResp.ApplicationAccelerator != nil {
			modules.ApplicationAccelerator = &ApplicationAcceleratorModule{
				Enabled: types.BoolValue(modulesResp.ApplicationAccelerator.GetEnabled()),
			}
		}
		if modulesResp.ImageProcessor != nil {
			modules.ImageProcessor = &ImageProcessorModule{
				Enabled: types.BoolValue(modulesResp.ImageProcessor.GetEnabled()),
			}
		}
		edgeAppResults.Modules = &modules
	}

	plan.EdgeApplication = edgeAppResults
	plan.ID = types.StringValue(fmt.Sprintf("%d", createEdgeApplication.Data.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idInt64, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	stateEdgeApplication, response, err := r.client.applicationsApi.
		ApplicationsAPI.
		RetrieveApplication(ctx, idInt64).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
			stateEdgeApplication, response, err = utils.RetryOn429(func() (*sdk.ResponseRetrieveApplication, *http.Response, error) {
				return r.client.applicationsApi.ApplicationsAPI.RetrieveApplication(ctx, idInt64).Execute() //nolint
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

	state.EdgeApplication = &EdgeApplicationResults{
		ApplicationID:  types.Int64Value(stateEdgeApplication.Data.GetId()),
		Name:           types.StringValue(stateEdgeApplication.Data.GetName()),
		Active:         types.BoolValue(stateEdgeApplication.Data.GetActive()),
		Debug:          types.BoolValue(stateEdgeApplication.Data.GetDebug()),
		ProductVersion: types.StringValue(stateEdgeApplication.Data.GetProductVersion()),
	}
	state.ID = types.StringValue(fmt.Sprintf("%d", stateEdgeApplication.Data.GetId()))

	modelPlan := ApplicationModules{}
	if stateEdgeApplication.Data.Modules != nil {
		modelState := stateEdgeApplication.Data.GetModules()
		if modelState.EdgeCache != nil {
			modelPlan.Cache = &CacheModule{
				Enabled: types.BoolValue(modelState.EdgeCache.GetEnabled()),
			}
		}
		if modelState.Functions != nil {
			modelPlan.Functions = &EdgeFunctionModule{
				Enabled: types.BoolValue(modelState.Functions.GetEnabled()),
			}
		}
		if modelState.ApplicationAccelerator != nil {
			modelPlan.ApplicationAccelerator = &ApplicationAcceleratorModule{
				Enabled: types.BoolValue(modelState.ApplicationAccelerator.GetEnabled()),
			}
		}
		if modelState.ImageProcessor != nil {
			modelPlan.ImageProcessor = &ImageProcessorModule{
				Enabled: types.BoolValue(modelState.ImageProcessor.GetEnabled()),
			}
		}
	}
	state.EdgeApplication.Modules = &modelPlan

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	edgeApplication := sdk.ApplicationRequest{
		Name:   plan.EdgeApplication.Name.ValueString(),
		Debug:  plan.EdgeApplication.Debug.ValueBoolPointer(),
		Active: plan.EdgeApplication.Active.ValueBoolPointer(),
	}

	modsPlan := plan.EdgeApplication.Modules
	modsRequest := transformModuleIntoRequest(modsPlan)
	edgeApplication.Modules = &modsRequest

	idInt64, _ := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	updateEdgeApplication, response, err := r.client.applicationsApi.
		ApplicationsAPI.
		UpdateApplication(ctx, idInt64).
		ApplicationRequest(edgeApplication).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			updateEdgeApplication, response, err = utils.RetryOn429(func() (*sdk.ResponseApplication, *http.Response, error) {
				return r.client.applicationsApi.
					ApplicationsAPI.
					UpdateApplication(ctx, idInt64).
					ApplicationRequest(edgeApplication).Execute() //nolint
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

	plan.EdgeApplication = &EdgeApplicationResults{
		ApplicationID:  types.Int64Value(updateEdgeApplication.Data.GetId()),
		Name:           types.StringValue(updateEdgeApplication.Data.GetName()),
		Active:         types.BoolValue(updateEdgeApplication.Data.GetActive()),
		Debug:          types.BoolValue(updateEdgeApplication.Data.GetDebug()),
		ProductVersion: types.StringValue(updateEdgeApplication.Data.GetProductVersion()),
		Modules:        modsPlan,
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d", updateEdgeApplication.Data.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idInt64, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	_, response, err := r.client.applicationsApi.ApplicationsAPI.
		DestroyApplication(ctx, idInt64).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.ResponseDeleteApplication, *http.Response, error) {
				return r.client.applicationsApi.ApplicationsAPI.DestroyApplication(ctx, idInt64).Execute() //nolint
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
}

func (r *edgeApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func transformModuleIntoRequest(modsPlan *ApplicationModules) sdk.ApplicationModulesRequest {
	modsRequest := sdk.ApplicationModulesRequest{}
	if modsPlan != nil {
		cachePlan := modsPlan.Cache
		if cachePlan != nil && !cachePlan.Enabled.IsNull() {
			enabled := cachePlan.Enabled
			cacheReq := sdk.CacheModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetEdgeCache(cacheReq)
		}
		functionsPlan := modsPlan.Functions
		if functionsPlan != nil && !functionsPlan.Enabled.IsNull() {
			enabled := functionsPlan.Enabled
			functionsReq := sdk.EdgeFunctionModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetFunctions(functionsReq)
		}
		applicationAcceleratorPlan := modsPlan.ApplicationAccelerator
		if applicationAcceleratorPlan != nil && !applicationAcceleratorPlan.Enabled.IsNull() {
			enabled := applicationAcceleratorPlan.Enabled
			appAccReq := sdk.ApplicationAcceleratorModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetApplicationAccelerator(appAccReq)
		}
		imageProcessorPlan := modsPlan.ImageProcessor
		if imageProcessorPlan != nil && !imageProcessorPlan.Enabled.IsNull() {
			enabled := imageProcessorPlan.Enabled
			imgProcReq := sdk.ImageProcessorModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetImageProcessor(imgProcReq)
		}
	}

	return modsRequest
}
