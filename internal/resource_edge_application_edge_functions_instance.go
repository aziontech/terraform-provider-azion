package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
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
	ID            types.String                         `tfsdk:"id"`
	ApplicationID types.String                         `tfsdk:"application_id"`
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
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
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
	var edgeApplicationID types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("application_id"), &edgeApplicationID)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
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

	edgeFunctionInstanceRequest := edgeapi.ApplicationFunctionInstanceRequest{
		Name:     plan.EdgeFunction.Name.ValueString(),
		Function: plan.EdgeFunction.EdgeFunctionId.ValueInt64(),
		Args:     planJsonArgs,
		Active:   plan.EdgeFunction.Active.ValueBoolPointer(),
	}

	edgeFunctionInstancesResponse, response, err := r.client.edgeApi.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, edgeApplicationID.ValueString()).ApplicationFunctionInstanceRequest(edgeFunctionInstanceRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseApplicationFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.CreateApplicationFunctionInstance(ctx, edgeApplicationID.ValueString()).ApplicationFunctionInstanceRequest(edgeFunctionInstanceRequest).Execute() //nolint
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

	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Data.GetId(), 10))
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
	var ApplicationID string
	var functionsInstancesId int64
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		ApplicationID = string(utils.AtoiNoError(valueFromCmd[0], resp))
		functionsInstancesId = int64(utils.AtoiNoError(valueFromCmd[1], resp))
	} else {
		ApplicationID = state.ApplicationID.ValueString()
		functionsInstancesId = state.EdgeFunction.ID.ValueInt64()
	}

	if functionsInstancesId == 0 {
		resp.Diagnostics.AddError(
			"Functions Instance id error ",
			"is not null",
		)
		return
	}

	stringFunctionsInstancesId := strconv.FormatInt(functionsInstancesId, 10)

	edgeFunctionInstancesResponse, response, err := r.client.edgeApi.ApplicationsFunctionAPI.
		RetrieveApplicationFunctionInstance(ctx, ApplicationID, stringFunctionsInstancesId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseRetrieveApplicationFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.
					RetrieveApplicationFunctionInstance(ctx, ApplicationID, stringFunctionsInstancesId).Execute() //nolint
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
		ApplicationID: types.StringValue(ApplicationID),
		ID:            types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Data.GetId(), 10)),
		EdgeFunction: &EdgeFunctionInstanceResourceResults{
			ID:             types.Int64Value(edgeFunctionInstancesResponse.Data.GetId()),
			EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Data.GetFunction()),
			Name:           types.StringValue(edgeFunctionInstancesResponse.Data.GetName()),
			Args:           types.StringValue(jsonArgsStr),
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
	var edgeApplicationID types.String
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

	if plan.ApplicationID.IsNull() {
		edgeApplicationID = state.ApplicationID
	} else {
		edgeApplicationID = plan.ApplicationID
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

	ApplicationPutInstanceRequest := edgeapi.PatchedApplicationFunctionInstanceRequest{
		Name:     plan.EdgeFunction.Name.ValueStringPointer(),
		Function: plan.EdgeFunction.EdgeFunctionId.ValueInt64Pointer(),
		Args:     requestJsonArgsStr,
		Active:   plan.EdgeFunction.Active.ValueBoolPointer(),
	}

	functionInstanceIDStr := strconv.FormatInt(functionsInstancesId.ValueInt64(), 10)
	edgeFunctionInstancesUpdateResponse, response, err := r.client.edgeApi.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, edgeApplicationID.ValueString(), functionInstanceIDStr).PatchedApplicationFunctionInstanceRequest(ApplicationPutInstanceRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesUpdateResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseApplicationFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.PartialUpdateApplicationFunctionInstance(ctx, edgeApplicationID.ValueString(), functionInstanceIDStr).PatchedApplicationFunctionInstanceRequest(ApplicationPutInstanceRequest).Execute() //nolint
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

	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesUpdateResponse.Data.GetId(), 10))
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

	functionInstanceID := strconv.FormatInt(state.EdgeFunction.ID.ValueInt64(), 10)
	_, response, err := r.client.edgeApi.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueString(), functionInstanceID).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			// Resource already deleted, consider this a success
			return
		}
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgeapi.ResponseDeleteApplicationFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.ApplicationsFunctionAPI.DeleteApplicationFunctionInstance(ctx, state.ApplicationID.ValueString(), functionInstanceID).Execute() //nolint
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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
