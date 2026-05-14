package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &applicationResource{}
	_ resource.ResourceWithConfigure   = &applicationResource{}
	_ resource.ResourceWithImportState = &applicationResource{}
)

func NewApplicationMainSettingsResource() resource.Resource {
	return &applicationResource{}
}

type applicationResource struct {
	client *apiClient
}

type ApplicationResourceModel struct {
	Application *ApplicationResults `tfsdk:"application"`
	ID          types.String        `tfsdk:"id"`
	LastUpdated types.String        `tfsdk:"last_updated"`
}

type ApplicationResults struct {
	ApplicationID  types.Int64         `tfsdk:"application_id"`
	Name           types.String        `tfsdk:"name"`
	Modules        *ApplicationModules `tfsdk:"modules"`
	Active         types.Bool          `tfsdk:"active"`
	Debug          types.Bool          `tfsdk:"debug"`
	ProductVersion types.String        `tfsdk:"product_version"`
}

func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_main_setting"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"application": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"application_id": schema.Int64Attribute{
						Description: "The Application identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "The name of the Application.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
						Description: "Indicates whether the Application is active.",
					},
					"debug": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Indicates whether debug rules are enabled for the Application.",
					},
					"product_version": schema.StringAttribute{
						Computed:    true,
						Description: "The product version.",
					},
					"modules": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"cache": schema.SingleNestedAttribute{
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

func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

var mutex sync.Mutex

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	mutex.Lock()
	defer mutex.Unlock()

	if resp.Diagnostics.HasError() {
		return
	}

	application := sdk.ApplicationRequest{
		Name:   plan.Application.Name.ValueString(),
		Active: plan.Application.Active.ValueBoolPointer(),
		Debug:  plan.Application.Debug.ValueBoolPointer(),
	}

	modsPlan := plan.Application.Modules
	modsRequest := transformModuleIntoRequest(modsPlan)

	application.Modules = &modsRequest

	createApplication, response, err := r.client.api.
		ApplicationsAPI.CreateApplication(ctx).
		ApplicationRequest(application).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			createApplication, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
				return r.client.api.
					ApplicationsAPI.CreateApplication(ctx).
					ApplicationRequest(application).Execute() //nolint
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

	appResults := &ApplicationResults{
		ApplicationID:  types.Int64Value(createApplication.Data.GetId()),
		Name:           types.StringValue(createApplication.Data.GetName()),
		Active:         types.BoolValue(createApplication.Data.GetActive()),
		Debug:          types.BoolValue(createApplication.Data.GetDebug()),
		ProductVersion: types.StringValue(createApplication.Data.GetProductVersion()),
		Modules:        plan.Application.Modules,
	}

	// Only update modules from API response if the plan had modules specified
	// This prevents Terraform from seeing an inconsistency when modules was null in plan
	if plan.Application.Modules != nil && createApplication.Data.Modules != nil {
		modulesResp := createApplication.Data.GetModules()
		modules := ApplicationModules{}

		// Only populate modules that were specified in the plan
		if plan.Application.Modules.Cache != nil && modulesResp.Cache != nil {
			modules.Cache = &CacheModule{
				Enabled: types.BoolValue(modulesResp.Cache.GetEnabled()),
			}
		}
		if plan.Application.Modules.Functions != nil && modulesResp.Functions != nil {
			modules.Functions = &FunctionModule{
				Enabled: types.BoolValue(modulesResp.Functions.GetEnabled()),
			}
		}
		if plan.Application.Modules.ApplicationAccelerator != nil && modulesResp.ApplicationAccelerator != nil {
			modules.ApplicationAccelerator = &ApplicationAcceleratorModule{
				Enabled: types.BoolValue(modulesResp.ApplicationAccelerator.GetEnabled()),
			}
		}
		if plan.Application.Modules.ImageProcessor != nil && modulesResp.ImageProcessor != nil {
			modules.ImageProcessor = &ImageProcessorModule{
				Enabled: types.BoolValue(modulesResp.ImageProcessor.GetEnabled()),
			}
		}
		appResults.Modules = &modules
	}

	plan.Application = appResults
	plan.ID = types.StringValue(fmt.Sprintf("%d", createApplication.Data.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idInt64, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	stateApplication, response, err := r.client.api.
		ApplicationsAPI.
		RetrieveApplication(ctx, idInt64).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
			stateApplication, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
				return r.client.api.ApplicationsAPI.RetrieveApplication(ctx, idInt64).Execute() //nolint
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

	// Preserve the prior state's Modules shape so unconfigured submodules
	// aren't introduced into state by the API response, which would cause
	// perpetual drift on subsequent plans.
	var previousModules *ApplicationModules
	if state.Application != nil {
		previousModules = state.Application.Modules
	}

	state.Application = &ApplicationResults{
		ApplicationID:  types.Int64Value(stateApplication.Data.GetId()),
		Name:           types.StringValue(stateApplication.Data.GetName()),
		Active:         types.BoolValue(stateApplication.Data.GetActive()),
		Debug:          types.BoolValue(stateApplication.Data.GetDebug()),
		ProductVersion: types.StringValue(stateApplication.Data.GetProductVersion()),
	}
	state.ID = types.StringValue(fmt.Sprintf("%d", stateApplication.Data.GetId()))

	if previousModules != nil && stateApplication.Data.Modules != nil {
		modelState := stateApplication.Data.GetModules()
		modelPlan := ApplicationModules{}
		if previousModules.Cache != nil && modelState.Cache != nil {
			modelPlan.Cache = &CacheModule{
				Enabled: types.BoolValue(modelState.Cache.GetEnabled()),
			}
		}
		if previousModules.Functions != nil && modelState.Functions != nil {
			modelPlan.Functions = &FunctionModule{
				Enabled: types.BoolValue(modelState.Functions.GetEnabled()),
			}
		}
		if previousModules.ApplicationAccelerator != nil && modelState.ApplicationAccelerator != nil {
			modelPlan.ApplicationAccelerator = &ApplicationAcceleratorModule{
				Enabled: types.BoolValue(modelState.ApplicationAccelerator.GetEnabled()),
			}
		}
		if previousModules.ImageProcessor != nil && modelState.ImageProcessor != nil {
			modelPlan.ImageProcessor = &ImageProcessorModule{
				Enabled: types.BoolValue(modelState.ImageProcessor.GetEnabled()),
			}
		}
		state.Application.Modules = &modelPlan
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	application := sdk.ApplicationRequest{
		Name:   plan.Application.Name.ValueString(),
		Debug:  plan.Application.Debug.ValueBoolPointer(),
		Active: plan.Application.Active.ValueBoolPointer(),
	}

	modsPlan := plan.Application.Modules
	modsRequest := transformModuleIntoRequest(modsPlan)
	application.Modules = &modsRequest

	idInt64, _ := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	updateApplication, response, err := r.client.api.
		ApplicationsAPI.
		UpdateApplication(ctx, idInt64).
		ApplicationRequest(application).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			updateApplication, response, err = utils.RetryOn429(func() (*sdk.ApplicationResponse, *http.Response, error) {
				return r.client.api.
					ApplicationsAPI.
					UpdateApplication(ctx, idInt64).
					ApplicationRequest(application).Execute() //nolint
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

	plan.Application = &ApplicationResults{
		ApplicationID:  types.Int64Value(updateApplication.Data.GetId()),
		Name:           types.StringValue(updateApplication.Data.GetName()),
		Active:         types.BoolValue(updateApplication.Data.GetActive()),
		Debug:          types.BoolValue(updateApplication.Data.GetDebug()),
		ProductVersion: types.StringValue(updateApplication.Data.GetProductVersion()),
		Modules:        modsPlan,
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d", updateApplication.Data.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idInt64, _ := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	_, response, err := r.client.api.ApplicationsAPI.
		DeleteApplication(ctx, idInt64).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.DeleteResponse, *http.Response, error) {
				return r.client.api.ApplicationsAPI.DeleteApplication(ctx, idInt64).Execute() //nolint
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

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
			modsRequest.SetCache(cacheReq)
		}

		functionsPlan := modsPlan.Functions
		if functionsPlan != nil && !functionsPlan.Enabled.IsNull() {
			enabled := functionsPlan.Enabled
			functionsReq := sdk.FunctionModuleRequest{
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
