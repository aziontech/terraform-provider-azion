package provider

import (
	"context"
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
	_ resource.Resource                = &functionInstanceResource{}
	_ resource.ResourceWithConfigure   = &functionInstanceResource{}
	_ resource.ResourceWithImportState = &functionInstanceResource{}
)

func NewApplicationFunctionInstanceResource() resource.Resource {
	return &functionInstanceResource{}
}

type functionInstanceResource struct {
	client *apiClient
}

type FunctionInstanceResourceModel struct {
	Function      *FunctionInstanceResourceResults `tfsdk:"data"`
	ID            types.Int64                      `tfsdk:"id"`
	ApplicationID types.Int64                      `tfsdk:"application_id"`
	LastUpdated   types.String                     `tfsdk:"last_updated"`
}

type FunctionInstanceResourceResults struct {
	FunctionID types.Int64  `tfsdk:"function_id"`
	Name       types.String `tfsdk:"name"`
	Args       types.String `tfsdk:"args"`
	ID         types.Int64  `tfsdk:"id"`
	Active     types.Bool   `tfsdk:"active"`
}

func (r *functionInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_function_instance"
}

func (r *functionInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"application_id": schema.Int64Attribute{
				Description: "The application identifier.",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The function instance identifier.",
						Computed:    true,
					},
					"function_id": schema.Int64Attribute{
						Description: "The function identifier.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Required:    true,
					},
					"args": schema.StringAttribute{
						Description: "JSON arguments of the function.",
						Optional:    true,
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the function instance is active.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *functionInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *functionInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FunctionInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var argsStr string
	if plan.Function.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		if plan.Function.Args.ValueString() == "" || plan.Function.Args.IsNull() {
			resp.Diagnostics.AddError("Args",
				"Is not null")
			return
		}
		argsStr = plan.Function.Args.ValueString()
	}

	planJsonArgs, err := utils.UnmarshallJsonArgs(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	functionInstanceRequest := azionapi.FunctionInstanceRequest{
		Name:     plan.Function.Name.ValueString(),
		Function: plan.Function.FunctionID.ValueInt64(),
		Args:     planJsonArgs,
		Active:   plan.Function.Active.ValueBoolPointer(),
	}

	functionInstanceResponse, response, err := r.client.api.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64()).FunctionInstanceRequest(functionInstanceRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
				return r.client.api.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64()).FunctionInstanceRequest(functionInstanceRequest).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Function = &FunctionInstanceResourceResults{
		FunctionID: types.Int64Value(functionInstanceResponse.Data.GetFunction()),
		Name:       types.StringValue(functionInstanceResponse.Data.GetName()),
		Args:       types.StringValue(jsonArgsStr),
		ID:         types.Int64Value(functionInstanceResponse.Data.GetId()),
		Active:     types.BoolValue(functionInstanceResponse.Data.GetActive()),
	}

	plan.ID = types.Int64Value(functionInstanceResponse.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *functionInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var applicationID int64
	var functionInstanceID int64

	// ID can be either just the instance ID or "applicationID/instanceID" format for import
	idStr := strconv.FormatInt(state.ID.ValueInt64(), 10)
	valueFromCmd := strings.Split(idStr, "/")
	if len(valueFromCmd) > 1 {
		appID, err := strconv.ParseInt(valueFromCmd[0], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid application ID format", err.Error())
			return
		}
		instanceID, err := strconv.ParseInt(valueFromCmd[1], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid instance ID format", err.Error())
			return
		}
		applicationID = appID
		functionInstanceID = instanceID
	} else {
		applicationID = state.ApplicationID.ValueInt64()
		functionInstanceID = state.ID.ValueInt64()
	}

	if functionInstanceID == 0 {
		resp.Diagnostics.AddError(
			"Function Instance id error ",
			"is not null",
		)
		return
	}

	functionInstanceResponse, response, err := r.client.api.ApplicationsFunctionAPI.
		RetrieveApplicationFunctionInstance(ctx, applicationID, functionInstanceID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
				return r.client.api.ApplicationsFunctionAPI.
					RetrieveApplicationFunctionInstance(ctx, applicationID, functionInstanceID).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	functionInstanceState := FunctionInstanceResourceModel{
		ApplicationID: types.Int64Value(applicationID),
		ID:            types.Int64Value(functionInstanceResponse.Data.GetId()),
		Function: &FunctionInstanceResourceResults{
			ID:         types.Int64Value(functionInstanceResponse.Data.GetId()),
			FunctionID: types.Int64Value(functionInstanceResponse.Data.GetFunction()),
			Name:       types.StringValue(functionInstanceResponse.Data.GetName()),
			Args:       types.StringValue(jsonArgsStr),
			Active:     types.BoolValue(functionInstanceResponse.Data.GetActive()),
		},
	}

	diags = resp.State.Set(ctx, &functionInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *functionInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FunctionInstanceResourceModel
	var functionInstanceID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FunctionInstanceResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Function.ID.IsNull() || plan.Function.ID.ValueInt64() == 0 {
		functionInstanceID = state.Function.ID
	} else {
		functionInstanceID = plan.Function.ID
	}

	var argsStr string
	if plan.Function.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		if plan.Function.Args.ValueString() == "" || plan.Function.Args.IsNull() {
			resp.Diagnostics.AddError(
				"Args",
				"Is not null")
			return
		}
		argsStr = plan.Function.Args.ValueString()
	}

	requestJsonArgsStr, err := utils.UnmarshallJsonArgs(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"error while unmarshalling json args",
		)
	}

	patchRequest := azionapi.PatchedFunctionInstanceRequest{
		Name:     plan.Function.Name.ValueStringPointer(),
		Function: plan.Function.FunctionID.ValueInt64Pointer(),
		Args:     requestJsonArgsStr,
		Active:   plan.Function.Active.ValueBoolPointer(),
	}

	functionInstanceUpdateResponse, response, err := r.client.api.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64(), functionInstanceID.ValueInt64()).PatchedFunctionInstanceRequest(patchRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionInstanceUpdateResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
				return r.client.api.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64(), functionInstanceID.ValueInt64()).PatchedFunctionInstanceRequest(patchRequest).Execute() //nolint
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
					"error while reading response body",
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceUpdateResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"error while reading json args from response",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Function = &FunctionInstanceResourceResults{
		FunctionID: types.Int64Value(functionInstanceUpdateResponse.Data.GetFunction()),
		Name:       types.StringValue(functionInstanceUpdateResponse.Data.GetName()),
		Args:       types.StringValue(jsonArgsStr),
		ID:         types.Int64Value(functionInstanceUpdateResponse.Data.GetId()),
		Active:     types.BoolValue(functionInstanceUpdateResponse.Data.GetActive()),
	}

	plan.ID = types.Int64Value(functionInstanceUpdateResponse.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *functionInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Function.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Function Instance id error ",
			"is not null",
		)
		return
	}

	if state.ApplicationID.IsNull() {
		resp.Diagnostics.AddError(
			"Application ID error ",
			"is not null",
		)
		return
	}

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueInt64(), state.Function.ID.ValueInt64()).Execute() //nolint
	}, 5) // Maximum 5 retries
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			// Resource already deleted, consider this a success
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

func (r *functionInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "applicationID/instanceID"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import format",
			"Expected format: applicationID/instanceID",
		)
		return
	}

	applicationID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application ID", err.Error())
		return
	}

	instanceID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid instance ID", err.Error())
		return
	}

	state := FunctionInstanceResourceModel{
		ApplicationID: types.Int64Value(applicationID),
		ID:            types.Int64Value(instanceID),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
