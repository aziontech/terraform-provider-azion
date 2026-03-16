package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &edgeFunctionsInstanceResource{}
	_ resource.ResourceWithConfigure   = &edgeFunctionsInstanceResource{}
	_ resource.ResourceWithImportState = &edgeFunctionsInstanceResource{}
)

func NewEdgeApplicationEdgeFunctionsInstanceResource() resource.Resource {
	return &edgeFunctionsInstanceResource{}
}

type edgeFunctionsInstanceResource struct {
	client *apiClient
}

type EdgeFunctionInstanceResourceModel struct {
	EdgeFunction  *EdgeFunctionInstanceResourceResults `tfsdk:"data"`
	ID            types.Int64                          `tfsdk:"id"`
	ApplicationID types.Int64                          `tfsdk:"application_id"`
	LastUpdated   types.String                         `tfsdk:"last_updated"`
}

type EdgeFunctionInstanceResourceResults struct {
	EdgeFunctionId types.Int64  `tfsdk:"function_id"`
	Name           types.String `tfsdk:"name"`
	Args           types.String `tfsdk:"args"`
	ID             types.Int64  `tfsdk:"id"`
	Active         types.Bool   `tfsdk:"active"`
}

func (r *edgeFunctionsInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_edge_functions_instance"
}

func (r *edgeFunctionsInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"application_id": schema.Int64Attribute{
				Description: "The edge application identifier.",
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
						Description: "The edge function instance identifier.",
						Computed:    true,
					},
					"function_id": schema.Int64Attribute{
						Description: "The edge function identifier.",
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

func (r *edgeFunctionsInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *edgeFunctionsInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EdgeFunctionInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var argsStr string
	if plan.EdgeFunction.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		if plan.EdgeFunction.Args.ValueString() == "" || plan.EdgeFunction.Args.IsNull() {
			resp.Diagnostics.AddError("Args",
				"Is not null")
			return
		}
		argsStr = plan.EdgeFunction.Args.ValueString()
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

	edgeFunctionInstanceRequest := azionapi.FunctionInstanceRequest{
		Name:     plan.EdgeFunction.Name.ValueString(),
		Function: plan.EdgeFunction.EdgeFunctionId.ValueInt64(),
		Args:     planJsonArgs,
		Active:   plan.EdgeFunction.Active.ValueBoolPointer(),
	}

	edgeFunctionInstancesResponse, response, err := r.client.api.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64()).FunctionInstanceRequest(edgeFunctionInstanceRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*azionapi.FunctionInstanceResponse, *http.Response, error) {
				return r.client.api.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64()).FunctionInstanceRequest(edgeFunctionInstanceRequest).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.EdgeFunction = &EdgeFunctionInstanceResourceResults{
		EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Data.GetFunction()),
		Name:           types.StringValue(edgeFunctionInstancesResponse.Data.GetName()),
		Args:           types.StringValue(jsonArgsStr),
		ID:             types.Int64Value(edgeFunctionInstancesResponse.Data.GetId()),
		Active:         types.BoolValue(edgeFunctionInstancesResponse.Data.GetActive()),
	}

	plan.ID = types.Int64Value(edgeFunctionInstancesResponse.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFunctionsInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EdgeFunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var applicationID int64
	var functionsInstancesId int64

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
		functionsInstancesId = instanceID
	} else {
		applicationID = state.ApplicationID.ValueInt64()
		functionsInstancesId = state.ID.ValueInt64()
	}

	if functionsInstancesId == 0 {
		resp.Diagnostics.AddError(
			"Functions Instance id error ",
			"is not null",
		)
		return
	}

	edgeFunctionInstancesResponse, response, err := r.client.edgeApi.ApplicationsFunctionAPI.
		RetrieveApplicationFunctionInstance(ctx, applicationID, functionsInstancesId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*edgeapi.FunctionInstanceResponse, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.
					RetrieveApplicationFunctionInstance(ctx, applicationID, functionsInstancesId).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	edgeApplicationsEdgeFunctionsInstanceState := EdgeFunctionInstanceResourceModel{
		ApplicationID: types.Int64Value(applicationID),
		ID:            types.Int64Value(edgeFunctionInstancesResponse.Data.GetId()),
		EdgeFunction: &EdgeFunctionInstanceResourceResults{
			ID:             types.Int64Value(edgeFunctionInstancesResponse.Data.GetId()),
			EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Data.GetFunction()),
			Name:           types.StringValue(edgeFunctionInstancesResponse.Data.GetName()),
			Args:           types.StringValue(jsonArgsStr),
			Active:         types.BoolValue(edgeFunctionInstancesResponse.Data.GetActive()),
		},
	}

	diags = resp.State.Set(ctx, &edgeApplicationsEdgeFunctionsInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFunctionsInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EdgeFunctionInstanceResourceModel
	var functionsInstancesId types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state EdgeFunctionInstanceResourceModel
	diagsOrigin := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsOrigin...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.EdgeFunction.ID.IsNull() || plan.EdgeFunction.ID.ValueInt64() == 0 {
		functionsInstancesId = state.EdgeFunction.ID
	} else {
		functionsInstancesId = plan.EdgeFunction.ID
	}

	var argsStr string
	if plan.EdgeFunction.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		if plan.EdgeFunction.Args.ValueString() == "" || plan.EdgeFunction.Args.IsNull() {
			resp.Diagnostics.AddError(
				"Args",
				"Is not null")
			return
		}
		argsStr = plan.EdgeFunction.Args.ValueString()
	}

	requestJsonArgsStr, err := utils.UnmarshallJsonArgs(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"error while unmarshalling json args",
		)
	}

	ApplicationPutInstanceRequest := edgeapi.PatchedFunctionInstanceRequest{
		Name:     plan.EdgeFunction.Name.ValueStringPointer(),
		Function: plan.EdgeFunction.EdgeFunctionId.ValueInt64Pointer(),
		Args:     requestJsonArgsStr,
		Active:   plan.EdgeFunction.Active.ValueBoolPointer(),
	}

	edgeFunctionInstancesUpdateResponse, response, err := r.client.edgeApi.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64(), functionsInstancesId.ValueInt64()).PatchedFunctionInstanceRequest(ApplicationPutInstanceRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesUpdateResponse, response, err = utils.RetryOn429(func() (*edgeapi.FunctionInstanceResponse, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, plan.ApplicationID.ValueInt64(), functionsInstancesId.ValueInt64()).PatchedFunctionInstanceRequest(ApplicationPutInstanceRequest).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesUpdateResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"error while reading json args from response",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.EdgeFunction = &EdgeFunctionInstanceResourceResults{
		EdgeFunctionId: types.Int64Value(edgeFunctionInstancesUpdateResponse.Data.GetFunction()),
		Name:           types.StringValue(edgeFunctionInstancesUpdateResponse.Data.GetName()),
		Args:           types.StringValue(jsonArgsStr),
		ID:             types.Int64Value(edgeFunctionInstancesUpdateResponse.Data.GetId()),
		Active:         types.BoolValue(edgeFunctionInstancesUpdateResponse.Data.GetActive()),
	}

	plan.ID = types.Int64Value(edgeFunctionInstancesUpdateResponse.Data.GetId())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFunctionsInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EdgeFunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.EdgeFunction.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Functions Instance id error ",
			"is not null",
		)
		return
	}

	if state.ApplicationID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}

	_, response, err := r.client.edgeApi.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueInt64(), state.EdgeFunction.ID.ValueInt64()).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			// Resource already deleted, consider this a success
			return
		}
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgeapi.DeleteResponse, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueInt64(), state.EdgeFunction.ID.ValueInt64()).Execute() //nolint
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

func (r *edgeFunctionsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	state := EdgeFunctionInstanceResourceModel{
		ApplicationID: types.Int64Value(applicationID),
		ID:            types.Int64Value(instanceID),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
