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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
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
	State          types.String        `tfsdk:"state"`
	VersionID      types.String        `tfsdk:"version_id"`
}

// Azion API defaults for application fields.
//
// The configuration is the desired state: a field left out of the configuration
// is reset to the value below on every apply, so a module toggled in Azion
// Console is undone rather than silently adopted.
//
// Every optional attribute needs a default. A Computed attribute without one is
// unknown in the plan whenever the configuration omits it, and the nested
// objects here are read into pointer fields that cannot hold unknown values, so
// a missing default breaks Create instead of merely weakening enforcement.
const (
	defaultApplicationActive           = true
	defaultApplicationDebug            = false
	defaultModuleCacheEnabled          = true
	defaultModuleFunctionsEnabled      = true
	defaultModuleAcceleratorEnabled    = false
	defaultModuleImageProcessorEnabled = false
)

// Attribute types for the nested objects, needed to build object defaults whose
// types must match the schema exactly.
var (
	moduleToggleAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
	}

	applicationModulesAttrTypes = map[string]attr.Type{
		"cache":                   types.ObjectType{AttrTypes: moduleToggleAttrTypes},
		"functions":               types.ObjectType{AttrTypes: moduleToggleAttrTypes},
		"application_accelerator": types.ObjectType{AttrTypes: moduleToggleAttrTypes},
		"image_processor":         types.ObjectType{AttrTypes: moduleToggleAttrTypes},
	}
)

var (
	applicationModulesDefault = types.ObjectValueMust(applicationModulesAttrTypes, map[string]attr.Value{
		"cache":                   moduleToggleDefault(defaultModuleCacheEnabled),
		"functions":               moduleToggleDefault(defaultModuleFunctionsEnabled),
		"application_accelerator": moduleToggleDefault(defaultModuleAcceleratorEnabled),
		"image_processor":         moduleToggleDefault(defaultModuleImageProcessorEnabled),
	})
)

func moduleToggleDefault(enabled bool) types.Object {
	return types.ObjectValueMust(moduleToggleAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(enabled),
	})
}

// moduleToggleAttribute builds the schema for one module, all of which are a
// single `enabled` flag.
func moduleToggleAttribute(description string, enabled bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		Default:     objectdefault.StaticValue(moduleToggleDefault(enabled)),
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(enabled),
			},
		},
	}
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
						Default:     booldefault.StaticBool(defaultApplicationActive),
						Description: "Indicates whether the Application is active.",
					},
					"debug": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(defaultApplicationDebug),
						Description: "Indicates whether debug rules are enabled for the Application.",
					},
					"product_version": schema.StringAttribute{
						Computed:    true,
						Description: "The product version.",
					},
					"state": schema.StringAttribute{
						Computed:    true,
						Description: "The state of the current application version.",
					},
					"version_id": schema.StringAttribute{
						Computed:    true,
						Description: "The identifier of the current application version.",
					},
					"modules": schema.SingleNestedAttribute{
						Description: "Modules enabled for the Application. Omitting a module resets it to its default on every apply.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(applicationModulesDefault),
						Attributes: map[string]schema.Attribute{
							"cache":                   moduleToggleAttribute("Cache module.", defaultModuleCacheEnabled),
							"functions":               moduleToggleAttribute("Functions module.", defaultModuleFunctionsEnabled),
							"application_accelerator": moduleToggleAttribute("Application Accelerator module.", defaultModuleAcceleratorEnabled),
							"image_processor":         moduleToggleAttribute("Image Processor module.", defaultModuleImageProcessorEnabled),
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

	plan.Application = transformApplicationResponseToModel(&createApplication.Data)
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

	state.Application = transformApplicationResponseToModel(&stateApplication.Data)
	state.ID = types.StringValue(fmt.Sprintf("%d", stateApplication.Data.GetId()))

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

	plan.Application = transformApplicationResponseToModel(&updateApplication.Data)

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
	_, response, err := utils.RetryOn429Delete(func() (*sdk.DeleteResponse, *http.Response, error) {
		return r.client.api.ApplicationsAPI.DeleteApplication(ctx, idInt64).Execute() //nolint
	}, 5) // Maximum 5 retries
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
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

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func transformModuleIntoRequest(modsPlan *ApplicationModules) sdk.ApplicationModulesRequest {
	modsRequest := sdk.ApplicationModulesRequest{}
	if modsPlan != nil {
		cachePlan := modsPlan.Cache
		if cachePlan != nil && isSet(cachePlan.Enabled) {
			enabled := cachePlan.Enabled
			cacheReq := sdk.CacheModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetCache(cacheReq)
		}

		functionsPlan := modsPlan.Functions
		if functionsPlan != nil && isSet(functionsPlan.Enabled) {
			enabled := functionsPlan.Enabled
			functionsReq := sdk.FunctionModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetFunctions(functionsReq)
		}

		applicationAcceleratorPlan := modsPlan.ApplicationAccelerator
		if applicationAcceleratorPlan != nil && isSet(applicationAcceleratorPlan.Enabled) {
			enabled := applicationAcceleratorPlan.Enabled
			appAccReq := sdk.ApplicationAcceleratorModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetApplicationAccelerator(appAccReq)
		}

		imageProcessorPlan := modsPlan.ImageProcessor
		if imageProcessorPlan != nil && isSet(imageProcessorPlan.Enabled) {
			enabled := imageProcessorPlan.Enabled
			imgProcReq := sdk.ImageProcessorModuleRequest{
				Enabled: enabled.ValueBoolPointer(),
			}
			modsRequest.SetImageProcessor(imgProcReq)
		}
	}

	return modsRequest
}

// transformApplicationResponseToModel mirrors the API response into the resource
// model without filtering. State has to match the remote application for
// Terraform to plan a revert when a module was toggled outside Terraform;
// unconfigured fields do not drift perpetually because every optional attribute
// is Computed with a schema default.
func transformApplicationResponseToModel(data *sdk.Application) *ApplicationResults {
	if data == nil {
		return nil
	}

	results := &ApplicationResults{
		ApplicationID:  types.Int64Value(data.GetId()),
		Name:           types.StringValue(data.GetName()),
		Active:         types.BoolValue(data.GetActive()),
		Debug:          types.BoolValue(data.GetDebug()),
		ProductVersion: types.StringValue(data.GetProductVersion()),
		State:          types.StringPointerValue(data.State.Get()),
		VersionID:      types.StringPointerValue(data.VersionId.Get()),
	}

	if data.Modules == nil {
		return results
	}

	modules := data.GetModules()
	results.Modules = &ApplicationModules{}

	if modules.Cache != nil {
		results.Modules.Cache = &CacheModule{Enabled: types.BoolValue(modules.Cache.GetEnabled())}
	}
	if modules.Functions != nil {
		results.Modules.Functions = &FunctionModule{Enabled: types.BoolValue(modules.Functions.GetEnabled())}
	}
	if modules.ApplicationAccelerator != nil {
		results.Modules.ApplicationAccelerator = &ApplicationAcceleratorModule{Enabled: types.BoolValue(modules.ApplicationAccelerator.GetEnabled())}
	}
	if modules.ImageProcessor != nil {
		results.Modules.ImageProcessor = &ImageProcessorModule{Enabled: types.BoolValue(modules.ImageProcessor.GetEnabled())}
	}

	return results
}
