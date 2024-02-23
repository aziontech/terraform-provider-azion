package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgefunctions"
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
	_ resource.Resource                = &edgeFunctionResource{}
	_ resource.ResourceWithConfigure   = &edgeFunctionResource{}
	_ resource.ResourceWithImportState = &edgeFunctionResource{}
)

func NewEdgeFunctionResource() resource.Resource {
	return &edgeFunctionResource{}
}

type edgeFunctionResource struct {
	client *apiClient
}

type edgeFunctionResourceModel struct {
	SchemaVersion types.Int64                  `tfsdk:"schema_version"`
	EdgeFunction  *edgeFunctionResourceResults `tfsdk:"edge_function"`
	ID            types.String                 `tfsdk:"id"`
	LastUpdated   types.String                 `tfsdk:"last_updated"`
}

type edgeFunctionResourceResults struct {
	FunctionID     types.Int64  `tfsdk:"function_id"`
	Name           types.String `tfsdk:"name"`
	Language       types.String `tfsdk:"language"`
	Code           types.String `tfsdk:"code"`
	JSONArgs       types.String `tfsdk:"json_args"`
	FunctionToRun  types.String `tfsdk:"function_to_run"`
	InitiatorType  types.String `tfsdk:"initiator_type"`
	IsActive       types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	Modified       types.String `tfsdk:"modified"`
	ReferenceCount types.Int64  `tfsdk:"reference_count"`
	Version        types.String `tfsdk:"version"`
}

func (r *edgeFunctionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_function"
}

func (r *edgeFunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "" +
			"~> **Note about Json_Args**\n" +
			"Parameter `json_args` must be specified with `jsonencode` function\n\n" +
			"~> **Note about Code**\n" +
			"Parameter `code`: For prevent any inconsistent use the function trimspace() - https://developer.hashicorp.com/terraform/language/functions/trimspace\n Can be specified with local_file in - https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"edge_function": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"function_id": schema.Int64Attribute{
						Description: "The function identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Required:    true,
					},
					"language": schema.StringAttribute{
						Description: "Language of the function.",
						Required:    true,
					},
					"code": schema.StringAttribute{
						Description: "Path Code of the function.",
						Required:    true,
					},
					"json_args": schema.StringAttribute{
						Required:    true,
						Description: "JSON arguments of the function.",
					},
					"function_to_run": schema.StringAttribute{
						Description: "The function to run.",
						Computed:    true,
					},
					"initiator_type": schema.StringAttribute{
						Description: "Initiator type of the function.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the function.",
						Required:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the function.",
						Computed:    true,
					},
					"modified": schema.StringAttribute{
						Description: "Last modified timestamp of the function.",
						Computed:    true,
					},
					"reference_count": schema.Int64Attribute{
						Description: "The reference count of the function.",
						Computed:    true,
					},
					"version": schema.StringAttribute{
						Description: "Version of the function.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *edgeFunctionResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *edgeFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan edgeFunctionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.JSONArgs.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFunction := edgefunctions.CreateEdgeFunctionRequest{
		Name:          edgefunctions.PtrString(plan.EdgeFunction.Name.ValueString()),
		Language:      edgefunctions.PtrString(plan.EdgeFunction.Language.ValueString()),
		Code:          edgefunctions.PtrString(plan.EdgeFunction.Code.ValueString()),
		Active:        edgefunctions.PtrBool(plan.EdgeFunction.IsActive.ValueBool()),
		InitiatorType: edgefunctions.PtrString(plan.EdgeFunction.InitiatorType.ValueString()),
		JsonArgs:      planJsonArgs,
	}

	createEdgeFunction, response, err := r.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsPost(ctx).CreateEdgeFunctionRequest(edgeFunction).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(createEdgeFunction.Results.JsonArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.EdgeFunction = &edgeFunctionResourceResults{
		FunctionID:    types.Int64Value(*createEdgeFunction.Results.Id),
		Name:          types.StringValue(*createEdgeFunction.Results.Name),
		Language:      types.StringValue(*createEdgeFunction.Results.Language),
		Code:          types.StringValue(*createEdgeFunction.Results.Code),
		JSONArgs:      types.StringValue(jsonArgsStr),
		InitiatorType: types.StringValue(*createEdgeFunction.Results.InitiatorType),
		IsActive:      types.BoolValue(*createEdgeFunction.Results.Active),
		LastEditor:    types.StringValue(*createEdgeFunction.Results.LastEditor),
		Modified:      types.StringValue(*createEdgeFunction.Results.Modified),
	}
	if createEdgeFunction.Results.ReferenceCount != nil {
		plan.EdgeFunction.ReferenceCount = types.Int64Value(*createEdgeFunction.Results.ReferenceCount)
	}
	if createEdgeFunction.Results.FunctionToRun != nil {
		plan.EdgeFunction.FunctionToRun = types.StringValue(*createEdgeFunction.Results.FunctionToRun)
	}
	plan.SchemaVersion = types.Int64Value(int64(*createEdgeFunction.SchemaVersion))
	plan.ID = types.StringValue(strconv.FormatInt(*createEdgeFunction.Results.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state edgeFunctionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var edgeFunctionId int64
	var err error
	if state.EdgeFunction != nil {
		edgeFunctionId = state.EdgeFunction.FunctionID.ValueInt64()
	} else {
		edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Edge Function ID",
			)
			return
		}
	}

	getEdgeFunction, response, err := r.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsIdGet(ctx, edgeFunctionId).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(getEdgeFunction.Results.JsonArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	EdgeFunctionState := EdgeFunctionDataSourceModel{
		SchemaVersion: types.Int64Value(int64(*getEdgeFunction.SchemaVersion)),
		Results: EdgeFunctionResults{
			FunctionID:    types.Int64Value(*getEdgeFunction.Results.Id),
			Name:          types.StringValue(*getEdgeFunction.Results.Name),
			Language:      types.StringValue(*getEdgeFunction.Results.Language),
			Code:          types.StringValue(*getEdgeFunction.Results.Code),
			JSONArgs:      types.StringValue(jsonArgsStr),
			InitiatorType: types.StringValue(*getEdgeFunction.Results.InitiatorType),
			IsActive:      types.BoolValue(*getEdgeFunction.Results.Active),
			LastEditor:    types.StringValue(*getEdgeFunction.Results.LastEditor),
			Modified:      types.StringValue(*getEdgeFunction.Results.Modified),
		},
	}
	if getEdgeFunction.Results.ReferenceCount != nil {
		EdgeFunctionState.Results.ReferenceCount = types.Int64Value(*getEdgeFunction.Results.ReferenceCount)
	}
	if getEdgeFunction.Results.FunctionToRun != nil {
		EdgeFunctionState.Results.FunctionToRun = types.StringValue(*getEdgeFunction.Results.FunctionToRun)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan edgeFunctionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state edgeFunctionResourceModel
	diagsEdgeFunction := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsEdgeFunction...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.JSONArgs.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	updateEdgeFunctionRequest := edgefunctions.PutEdgeFunctionRequest{
		Name:          edgefunctions.PtrString(plan.EdgeFunction.Name.ValueString()),
		Code:          edgefunctions.PtrString(plan.EdgeFunction.Code.ValueString()),
		Active:        edgefunctions.PtrBool(plan.EdgeFunction.IsActive.ValueBool()),
		InitiatorType: edgefunctions.PtrString(plan.EdgeFunction.InitiatorType.ValueString()),
		JsonArgs:      requestJsonArgs,
	}
	var edgeFunctionId int64
	if state.ID.IsNull() {
		edgeFunctionId = state.EdgeFunction.FunctionID.ValueInt64()
	} else {
		edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert edgeFunctionId to int",
			)
			return
		}
	}

	updateEdgeFunction, response, err := r.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsIdPut(ctx, edgeFunctionId).PutEdgeFunctionRequest(updateEdgeFunctionRequest).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(updateEdgeFunction.Results.JsonArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.EdgeFunction = &edgeFunctionResourceResults{
		FunctionID:    types.Int64Value(*updateEdgeFunction.Results.Id),
		Name:          types.StringValue(*updateEdgeFunction.Results.Name),
		Language:      types.StringValue(*updateEdgeFunction.Results.Language),
		Code:          types.StringValue(*updateEdgeFunction.Results.Code),
		JSONArgs:      types.StringValue(jsonArgsStr),
		InitiatorType: types.StringValue(*updateEdgeFunction.Results.InitiatorType),
		IsActive:      types.BoolValue(*updateEdgeFunction.Results.Active),
		LastEditor:    types.StringValue(*updateEdgeFunction.Results.LastEditor),
		Modified:      types.StringValue(*updateEdgeFunction.Results.Modified),
	}
	if updateEdgeFunction.Results.ReferenceCount != nil {
		plan.EdgeFunction.ReferenceCount = types.Int64Value(*updateEdgeFunction.Results.ReferenceCount)
	}
	if updateEdgeFunction.Results.FunctionToRun != nil {
		plan.EdgeFunction.FunctionToRun = types.StringValue(*updateEdgeFunction.Results.FunctionToRun)
	}
	plan.SchemaVersion = types.Int64Value(int64(*updateEdgeFunction.SchemaVersion))
	plan.ID = types.StringValue(strconv.FormatInt(*updateEdgeFunction.Results.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state edgeFunctionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var edgeFunctionId int64
	var err error
	if state.EdgeFunction != nil {
		edgeFunctionId = state.EdgeFunction.FunctionID.ValueInt64()
	} else {
		edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Edge Function ID",
			)
			return
		}
	}
	response, err := r.client.edgefunctionsApi.EdgeFunctionsAPI.EdgeFunctionsIdDelete(ctx, edgeFunctionId).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

func (r *edgeFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
