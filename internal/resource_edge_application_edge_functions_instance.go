package provider

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgeapplications"
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
	SchemaVersion types.Int64                          `tfsdk:"schema_version"`
	EdgeFunction  *EdgeFunctionInstanceResourceResults `tfsdk:"results"`
	ID            types.String                         `tfsdk:"id"`
	ApplicationID types.Int64                          `tfsdk:"edge_application_id"`
	LastUpdated   types.String                         `tfsdk:"last_updated"`
}

type EdgeFunctionInstanceResourceResults struct {
	EdgeFunctionId types.Int64  `tfsdk:"edge_function_id"`
	Name           types.String `tfsdk:"name"`
	Args           types.String `tfsdk:"args"`
	ID             types.Int64  `tfsdk:"id"`
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
			"edge_application_id": schema.Int64Attribute{
				Description: "The edge application identifier.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The edge function instance identifier.",
						Computed:    true,
					},
					"edge_function_id": schema.Int64Attribute{
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
	var edgeApplicationID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &edgeApplicationID)
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

	planJsonArgs, err := utils.ConvertStringToInterface(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFunctionInstanceRequest := edgeapplications.ApplicationCreateInstanceRequest{
		Name:           plan.EdgeFunction.Name.ValueString(),
		EdgeFunctionId: plan.EdgeFunction.EdgeFunctionId.ValueInt64(),
		Args:           planJsonArgs,
	}

	edgeFunctionInstancesResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesPost(ctx, edgeApplicationID.ValueInt64()).ApplicationCreateInstanceRequest(edgeFunctionInstanceRequest).Execute() //nolint
	if err != nil {
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Results.GetArgs())
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
		EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Results.GetEdgeFunctionId()),
		Name:           types.StringValue(edgeFunctionInstancesResponse.Results.GetName()),
		Args:           types.StringValue(jsonArgsStr),
		ID:             types.Int64Value(edgeFunctionInstancesResponse.Results.GetId()),
	}

	plan.SchemaVersion = types.Int64Value(*edgeFunctionInstancesResponse.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Results.GetId(), 10))
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
	var ApplicationID int64
	var functionsInstancesId int64
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		ApplicationID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		functionsInstancesId = int64(utils.AtoiNoError(valueFromCmd[1], resp))
	} else {
		ApplicationID = state.ApplicationID.ValueInt64()
		functionsInstancesId = state.EdgeFunction.ID.ValueInt64()
	}

	if functionsInstancesId == 0 {
		resp.Diagnostics.AddError(
			"Edge Functions Instance id error ",
			"is not null",
		)
		return
	}

	edgeFunctionInstancesResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesFunctionsInstancesIdGet(ctx, ApplicationID, functionsInstancesId).Execute() //nolint
	if err != nil {
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Results.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	edgeApplicationsEdgeFunctionsInstanceState := EdgeFunctionInstanceResourceModel{
		ApplicationID: types.Int64Value(ApplicationID),
		SchemaVersion: types.Int64Value(edgeFunctionInstancesResponse.SchemaVersion),
		ID:            types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Results.GetId(), 10)),
		EdgeFunction: &EdgeFunctionInstanceResourceResults{
			ID:             types.Int64Value(edgeFunctionInstancesResponse.Results.GetId()),
			EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Results.GetEdgeFunctionId()),
			Name:           types.StringValue(edgeFunctionInstancesResponse.Results.GetName()),
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
	var edgeApplicationID types.Int64
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
			resp.Diagnostics.AddError("Args",
				"Is not null")
			return
		}
		argsStr = plan.EdgeFunction.Args.ValueString()
	}

	requestJsonArgsStr, err := utils.ConvertStringToInterface(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}

	ApplicationPutInstanceRequest := edgeapplications.ApplicationPutInstanceRequest{
		Name:           plan.EdgeFunction.Name.ValueString(),
		EdgeFunctionId: plan.EdgeFunction.EdgeFunctionId.ValueInt64(),
		Args:           requestJsonArgsStr,
	}

	edgeFunctionInstancesUpdateResponse, response, err := r.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesFunctionsInstancesIdPut(ctx, edgeApplicationID.String(), functionsInstancesId.String()).ApplicationPutInstanceRequest(ApplicationPutInstanceRequest).Execute() //nolint
	if err != nil {
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesUpdateResponse.Results.GetArgs())
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
		EdgeFunctionId: types.Int64Value(edgeFunctionInstancesUpdateResponse.Results.GetEdgeFunctionId()),
		Name:           types.StringValue(edgeFunctionInstancesUpdateResponse.Results.GetName()),
		Args:           types.StringValue(jsonArgsStr),
		ID:             types.Int64Value(edgeFunctionInstancesUpdateResponse.Results.GetId()),
	}

	plan.SchemaVersion = types.Int64Value(*edgeFunctionInstancesUpdateResponse.SchemaVersion)
	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesUpdateResponse.Results.GetId(), 10))
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

	response, err := r.client.edgeApplicationsApi.EdgeApplicationsEdgeFunctionsInstancesAPI.EdgeApplicationsEdgeApplicationIdFunctionsInstancesFunctionsInstancesIdDelete(ctx, state.ApplicationID.String(), state.EdgeFunction.ID.String()).Execute() //nolint
	if err != nil {
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

func (r *edgeFunctionsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
