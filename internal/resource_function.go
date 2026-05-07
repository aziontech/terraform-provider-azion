package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
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
	_ resource.Resource                = &functionResource{}
	_ resource.ResourceWithConfigure   = &functionResource{}
	_ resource.ResourceWithImportState = &functionResource{}
)

func NewFunctionResource() resource.Resource {
	return &functionResource{}
}

type functionResource struct {
	client *apiClient
}

type functionResourceModel struct {
	Function    *functionResourceResults `tfsdk:"function"`
	ID          types.String             `tfsdk:"id"`
	LastUpdated types.String             `tfsdk:"last_updated"`
}

type functionResourceResults struct {
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

func (r *functionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (r *functionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"function": schema.SingleNestedAttribute{
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

func (r *functionResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *functionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan functionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	edgeFunction := azionapi.FunctionsRequest{
		Name: plan.Function.Name.ValueString(),
		Code: plan.Function.Code.ValueString(),
	}

	// Only include optional fields if they are set
	if !plan.Function.Active.IsNull() && !plan.Function.Active.IsUnknown() {
		edgeFunction.SetActive(plan.Function.Active.ValueBool())
	}

	if !plan.Function.ExecutionEnvironment.IsNull() && !plan.Function.ExecutionEnvironment.IsUnknown() {
		edgeFunction.SetExecutionEnvironment(plan.Function.ExecutionEnvironment.ValueString())
	}

	if !plan.Function.Runtime.IsNull() && !plan.Function.Runtime.IsUnknown() {
		edgeFunction.SetRuntime(plan.Function.Runtime.ValueString())
	}

	if !plan.Function.DefaultArgs.IsNull() && !plan.Function.DefaultArgs.IsUnknown() {
		planJsonArgs, err := utils.ConvertStringToInterface(plan.Function.DefaultArgs.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
			return
		}
		edgeFunction.SetDefaultArgs(planJsonArgs)
	}

	createFunction, response, err := r.client.api.FunctionsAPI.CreateFunction(ctx).FunctionsRequest(edgeFunction).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createFunction, response, err = utils.RetryOn429(func() (*azionapi.FunctionResponse, *http.Response, error) {
				return r.client.api.FunctionsAPI.CreateFunction(ctx).FunctionsRequest(edgeFunction).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(createFunction.Data.DefaultArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Function = &functionResourceResults{
		ID:                   types.Int64Value(createFunction.Data.Id),
		Name:                 types.StringValue(createFunction.Data.Name),
		Code:                 types.StringValue(createFunction.Data.Code),
		DefaultArgs:          types.StringValue(jsonArgsStr),
		ExecutionEnvironment: types.StringValue(*createFunction.Data.ExecutionEnvironment),
		Active:               types.BoolValue(*createFunction.Data.Active),
		LastEditor:           types.StringValue(createFunction.Data.LastEditor),
		LastModified:         types.StringValue(createFunction.Data.LastModified.Format(time.RFC850)),
		ProductVersion:       types.StringValue(createFunction.Data.ProductVersion),
		Version:              types.StringValue(createFunction.Data.Version),
		Vendor:               types.StringValue(createFunction.Data.Vendor),
		ReferenceCount:       types.Int64Value(createFunction.Data.ReferenceCount),
	}

	if createFunction.Data.Runtime != nil {
		plan.Function.Runtime = types.StringValue(*createFunction.Data.Runtime)
	}

	plan.ID = types.StringValue(strconv.FormatInt(createFunction.Data.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *functionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state functionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var functionId int64
	var err error
	if state.Function != nil {
		functionId = state.Function.ID.ValueInt64()
	} else {
		functionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Function ID",
			)
			return
		}
	}

	getFunction, response, err := r.client.api.FunctionsAPI.RetrieveFunction(ctx, functionId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getFunction, response, err = utils.RetryOn429(func() (*azionapi.FunctionResponse, *http.Response, error) {
				return r.client.api.FunctionsAPI.RetrieveFunction(ctx, functionId).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(getFunction.Data.DefaultArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state.Function = &functionResourceResults{
		ID:                   types.Int64Value(getFunction.Data.Id),
		Name:                 types.StringValue(getFunction.Data.Name),
		Code:                 types.StringValue(getFunction.Data.Code),
		DefaultArgs:          types.StringValue(jsonArgsStr),
		ExecutionEnvironment: types.StringValue(*getFunction.Data.ExecutionEnvironment),
		Active:               types.BoolValue(*getFunction.Data.Active),
		LastEditor:           types.StringValue(getFunction.Data.LastEditor),
		LastModified:         types.StringValue(getFunction.Data.LastModified.Format(time.RFC850)),
		ProductVersion:       types.StringValue(getFunction.Data.ProductVersion),
		Version:              types.StringValue(getFunction.Data.Version),
		Vendor:               types.StringValue(getFunction.Data.Vendor),
		ReferenceCount:       types.Int64Value(getFunction.Data.ReferenceCount),
	}

	if getFunction.Data.Runtime != nil {
		state.Function.Runtime = types.StringValue(*getFunction.Data.Runtime)
	}
	state.ID = types.StringValue(strconv.FormatInt(getFunction.Data.Id, 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *functionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan functionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state functionResourceModel
	diagsFunction := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsFunction...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateFunctionRequest := azionapi.PatchedFunctionsRequest{}

	// Only include optional fields if they are set
	if !plan.Function.Name.IsNull() && !plan.Function.Name.IsUnknown() {
		updateFunctionRequest.SetName(plan.Function.Name.ValueString())
	}

	if !plan.Function.Code.IsNull() && !plan.Function.Code.IsUnknown() {
		updateFunctionRequest.SetCode(plan.Function.Code.ValueString())
	}

	if !plan.Function.Active.IsNull() && !plan.Function.Active.IsUnknown() {
		updateFunctionRequest.SetActive(plan.Function.Active.ValueBool())
	}

	if !plan.Function.ExecutionEnvironment.IsNull() && !plan.Function.ExecutionEnvironment.IsUnknown() {
		updateFunctionRequest.SetExecutionEnvironment(plan.Function.ExecutionEnvironment.ValueString())
	}

	if !plan.Function.Runtime.IsNull() && !plan.Function.Runtime.IsUnknown() {
		updateFunctionRequest.SetRuntime(plan.Function.Runtime.ValueString())
	}

	if !plan.Function.DefaultArgs.IsNull() && !plan.Function.DefaultArgs.IsUnknown() {
		requestJsonArgs, err := utils.ConvertStringToInterface(plan.Function.DefaultArgs.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"err",
			)
			return
		}
		updateFunctionRequest.SetDefaultArgs(requestJsonArgs)
	}

	var functionId int64
	var err error
	if state.ID.IsNull() {
		functionId = state.Function.ID.ValueInt64()
	} else {
		functionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Function ID",
			)
			return
		}
	}

	updateFunction, response, err := r.client.api.FunctionsAPI.PartialUpdateFunction(ctx, functionId).PatchedFunctionsRequest(updateFunctionRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateFunction, response, err = utils.RetryOn429(func() (*azionapi.FunctionResponse, *http.Response, error) {
				return r.client.api.FunctionsAPI.PartialUpdateFunction(ctx, functionId).PatchedFunctionsRequest(updateFunctionRequest).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(updateFunction.Data.DefaultArgs)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Function = &functionResourceResults{
		ID:                   types.Int64Value(updateFunction.Data.Id),
		Name:                 types.StringValue(updateFunction.Data.Name),
		Code:                 types.StringValue(updateFunction.Data.Code),
		DefaultArgs:          types.StringValue(jsonArgsStr),
		ExecutionEnvironment: types.StringValue(*updateFunction.Data.ExecutionEnvironment),
		Active:               types.BoolValue(*updateFunction.Data.Active),
		LastEditor:           types.StringValue(updateFunction.Data.LastEditor),
		LastModified:         types.StringValue(updateFunction.Data.LastModified.Format(time.RFC850)),
		ProductVersion:       types.StringValue(updateFunction.Data.ProductVersion),
		Version:              types.StringValue(updateFunction.Data.Version),
		Vendor:               types.StringValue(updateFunction.Data.Vendor),
		ReferenceCount:       types.Int64Value(updateFunction.Data.ReferenceCount),
	}

	if updateFunction.Data.Runtime != nil {
		plan.Function.Runtime = types.StringValue(*updateFunction.Data.Runtime)
	}

	plan.ID = types.StringValue(strconv.FormatInt(updateFunction.Data.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *functionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state functionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var functionId int64
	var err error
	if state.Function != nil {
		functionId = state.Function.ID.ValueInt64()
	} else {
		functionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Function ID",
			)
			return
		}
	}

	_, response, err := r.client.api.FunctionsAPI.DeleteFunction(ctx, functionId).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.FunctionsAPI.DeleteFunction(ctx, functionId).Execute() //nolint
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

func (r *functionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
