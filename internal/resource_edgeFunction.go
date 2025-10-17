package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
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
	EdgeFunction *edgeFunctionResourceResults `tfsdk:"edge_function"`
	ID           types.String                 `tfsdk:"id"`
	LastUpdated  types.String                 `tfsdk:"last_updated"`
}

type edgeFunctionResourceResults struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	LastEditor           types.String `tfsdk:"last_editor"`
	LastModified         types.String `tfsdk:"last_modified"`
	ProductVersion       types.String `tfsdk:"product_version"`
	Active               types.Bool   `tfsdk:"active"`
	Runtime              types.String `tfsdk:"runtime"`
	ExecutionEnvironment types.String `tfsdk:"execution_environment"`
	Code                 types.String `tfsdk:"code"`
	DefaultArgs          types.String `tfsdk:"default_args"`
	ReferenceCount       types.Int64  `tfsdk:"reference_count"`
	Version              types.String `tfsdk:"version"`
	Vendor               types.String `tfsdk:"vendor"`
}

func (r *edgeFunctionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_function"
}

func (r *edgeFunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "" +
			"~> **Note about default_args**\n" +
			"Parameter `default_args` must be specified with `jsonencode` function\n\n" +
			"~> **Note about Code**\n" +
			"Parameter `code`: For prevent any inconsistent use the function trimspace() - https://developer.hashicorp.com/terraform/language/functions/trimspace\n Can be specified with local_file in - https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file",
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
			"edge_function": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The function identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Required:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the function.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the function.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the function.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the function.",
						Optional:    true,
						Computed:    true,
					},
					"runtime": schema.StringAttribute{
						Description: "Runtime of the function.",
						Optional:    true,
						Computed:    true,
					},
					"execution_environment": schema.StringAttribute{
						Description: "Execution environment of the function.",
						Optional:    true,
						Computed:    true,
					},
					"code": schema.StringAttribute{
						Description: "Code of the function.",
						Required:    true,
					},
					"default_args": schema.StringAttribute{
						Description: "Default arguments of the function as JSON.",
						Optional:    true,
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
					"vendor": schema.StringAttribute{
						Description: "Vendor of the function.",
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

	edgeFunction := sdk.EdgeFunctionsRequest{
		Name: plan.EdgeFunction.Name.ValueString(),
		Code: plan.EdgeFunction.Code.ValueString(),
	}

	// Only include optional fields if they are set
	if !plan.EdgeFunction.Active.IsNull() && !plan.EdgeFunction.Active.IsUnknown() {
		edgeFunction.Active = sdk.PtrBool(plan.EdgeFunction.Active.ValueBool())
	}

	if !plan.EdgeFunction.ExecutionEnvironment.IsNull() && !plan.EdgeFunction.ExecutionEnvironment.IsUnknown() {
		edgeFunction.ExecutionEnvironment = sdk.PtrString(plan.EdgeFunction.ExecutionEnvironment.ValueString())
	}

	if !plan.EdgeFunction.Runtime.IsNull() && !plan.EdgeFunction.Runtime.IsUnknown() {
		edgeFunction.Runtime = sdk.PtrString(plan.EdgeFunction.Runtime.ValueString())
	}

	if !plan.EdgeFunction.DefaultArgs.IsNull() && !plan.EdgeFunction.DefaultArgs.IsUnknown() {
		planJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.DefaultArgs.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
			return
		}
		edgeFunction.DefaultArgs = planJsonArgs
	}

	createEdgeFunction, response, err := r.client.edgeApi.FunctionsAPI.CreateFunction(ctx).EdgeFunctionsRequest(edgeFunction).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createEdgeFunction, response, err = utils.RetryOn429(func() (*sdk.ResponseFunctionsDoc, *http.Response, error) {
				return r.client.edgeApi.FunctionsAPI.CreateFunction(ctx).EdgeFunctionsRequest(edgeFunction).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(createEdgeFunction.Data.DefaultArgs)
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
		ID:                   types.Int64Value(createEdgeFunction.Data.Id),
		Name:                 types.StringValue(createEdgeFunction.Data.Name),
		Code:                 types.StringValue(createEdgeFunction.Data.Code),
		DefaultArgs:          types.StringValue(jsonArgsStr),
		ExecutionEnvironment: types.StringValue(*createEdgeFunction.Data.ExecutionEnvironment),
		Active:               types.BoolValue(*createEdgeFunction.Data.Active),
		LastEditor:           types.StringValue(createEdgeFunction.Data.LastEditor),
		LastModified:         types.StringValue(createEdgeFunction.Data.LastModified.Format(time.RFC850)),
		ProductVersion:       types.StringValue(createEdgeFunction.Data.ProductVersion),
		Version:              types.StringValue(createEdgeFunction.Data.Version),
		Vendor:               types.StringValue(createEdgeFunction.Data.Vendor),
		ReferenceCount:       types.Int64Value(createEdgeFunction.Data.ReferenceCount),
	}

	if createEdgeFunction.Data.Runtime != nil {
		plan.EdgeFunction.Runtime = types.StringValue(*createEdgeFunction.Data.Runtime)
	}

	plan.ID = types.StringValue(strconv.FormatInt(createEdgeFunction.Data.Id, 10))
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
		edgeFunctionId = state.EdgeFunction.ID.ValueInt64()
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

	funcIdStr := strconv.FormatInt(edgeFunctionId, 10)

	getEdgeFunction, response, err := r.client.edgeApi.FunctionsAPI.RetrieveFunction(ctx, funcIdStr).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getEdgeFunction, response, err = utils.RetryOn429(func() (*sdk.ResponseRetrieveFunctionsDoc, *http.Response, error) {
				return r.client.edgeApi.FunctionsAPI.RetrieveFunction(ctx, funcIdStr).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(getEdgeFunction.Data.DefaultArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state.EdgeFunction = &edgeFunctionResourceResults{
		ID:                   types.Int64Value(getEdgeFunction.Data.Id),
		Name:                 types.StringValue(getEdgeFunction.Data.Name),
		Code:                 types.StringValue(getEdgeFunction.Data.Code),
		DefaultArgs:          types.StringValue(jsonArgsStr),
		ExecutionEnvironment: types.StringValue(*getEdgeFunction.Data.ExecutionEnvironment),
		Active:               types.BoolValue(*getEdgeFunction.Data.Active),
		LastEditor:           types.StringValue(getEdgeFunction.Data.LastEditor),
		LastModified:         types.StringValue(getEdgeFunction.Data.LastModified.Format(time.RFC850)),
		ProductVersion:       types.StringValue(getEdgeFunction.Data.ProductVersion),
		Version:              types.StringValue(getEdgeFunction.Data.Version),
		Vendor:               types.StringValue(getEdgeFunction.Data.Vendor),
		ReferenceCount:       types.Int64Value(getEdgeFunction.Data.ReferenceCount),
	}

	if getEdgeFunction.Data.Runtime != nil {
		state.EdgeFunction.Runtime = types.StringValue(*getEdgeFunction.Data.Runtime)
	}
	state.ID = types.StringValue(strconv.FormatInt(getEdgeFunction.Data.Id, 10))

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

	updateEdgeFunctionRequest := sdk.PatchedEdgeFunctionsRequest{
		Name: plan.EdgeFunction.Name.ValueStringPointer(),
		Code: plan.EdgeFunction.Code.ValueStringPointer(),
	}

	// Only include optional fields if they are set
	if !plan.EdgeFunction.Active.IsNull() && !plan.EdgeFunction.Active.IsUnknown() {
		updateEdgeFunctionRequest.Active = plan.EdgeFunction.Active.ValueBoolPointer()
	}

	if !plan.EdgeFunction.ExecutionEnvironment.IsNull() && !plan.EdgeFunction.ExecutionEnvironment.IsUnknown() {
		updateEdgeFunctionRequest.ExecutionEnvironment = plan.EdgeFunction.ExecutionEnvironment.ValueStringPointer()
	}

	if !plan.EdgeFunction.Runtime.IsNull() && !plan.EdgeFunction.Runtime.IsUnknown() {
		updateEdgeFunctionRequest.Runtime = plan.EdgeFunction.Runtime.ValueStringPointer()
	}

	if !plan.EdgeFunction.DefaultArgs.IsNull() && !plan.EdgeFunction.DefaultArgs.IsUnknown() {
		requestJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.DefaultArgs.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
			return
		}
		updateEdgeFunctionRequest.DefaultArgs = requestJsonArgs
	}

	var edgeFunctionId int64
	var err error
	if state.ID.IsNull() {
		edgeFunctionId = state.EdgeFunction.ID.ValueInt64()
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

	funcIdStr := strconv.FormatInt(edgeFunctionId, 10)

	updateEdgeFunction, response, err := r.client.edgeApi.FunctionsAPI.PartialUpdateFunction(ctx, funcIdStr).PatchedEdgeFunctionsRequest(updateEdgeFunctionRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateEdgeFunction, response, err = utils.RetryOn429(func() (*sdk.ResponseFunctionsDoc, *http.Response, error) {
				return r.client.edgeApi.FunctionsAPI.PartialUpdateFunction(ctx, funcIdStr).PatchedEdgeFunctionsRequest(updateEdgeFunctionRequest).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(updateEdgeFunction.Data.DefaultArgs)
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
		ID:                   types.Int64Value(updateEdgeFunction.Data.Id),
		Name:                 types.StringValue(updateEdgeFunction.Data.Name),
		Code:                 types.StringValue(updateEdgeFunction.Data.Code),
		DefaultArgs:          types.StringValue(jsonArgsStr),
		ExecutionEnvironment: types.StringValue(*updateEdgeFunction.Data.ExecutionEnvironment),
		Active:               types.BoolValue(*updateEdgeFunction.Data.Active),
		LastEditor:           types.StringValue(updateEdgeFunction.Data.LastEditor),
		LastModified:         types.StringValue(updateEdgeFunction.Data.LastModified.Format(time.RFC850)),
		ProductVersion:       types.StringValue(updateEdgeFunction.Data.ProductVersion),
		Version:              types.StringValue(updateEdgeFunction.Data.Version),
		Vendor:               types.StringValue(updateEdgeFunction.Data.Vendor),
		ReferenceCount:       types.Int64Value(updateEdgeFunction.Data.ReferenceCount),
	}

	if updateEdgeFunction.Data.Runtime != nil {
		plan.EdgeFunction.Runtime = types.StringValue(*updateEdgeFunction.Data.Runtime)
	}

	plan.ID = types.StringValue(strconv.FormatInt(updateEdgeFunction.Data.Id, 10))
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
		edgeFunctionId = state.EdgeFunction.ID.ValueInt64()
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
	functionIdString := strconv.FormatInt(edgeFunctionId, 10)

	_, response, err := r.client.edgeApi.FunctionsAPI.DestroyFunction(ctx, functionIdString).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.ResponseDeleteFunctionsDoc, *http.Response, error) {
				return r.client.edgeApi.FunctionsAPI.DestroyFunction(ctx, functionIdString).Execute() //nolint
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

func (r *edgeFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
