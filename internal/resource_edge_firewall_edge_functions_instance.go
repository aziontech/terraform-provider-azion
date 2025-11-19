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
	_ resource.Resource                = &edgeFirewallFunctionsInstanceResource{}
	_ resource.ResourceWithConfigure   = &edgeFirewallFunctionsInstanceResource{}
	_ resource.ResourceWithImportState = &edgeFirewallFunctionsInstanceResource{}
)

func NewEdgeFirewallEdgeFunctionsInstanceResource() resource.Resource {
	return &edgeFirewallFunctionsInstanceResource{}
}

type edgeFirewallFunctionsInstanceResource struct {
	client *apiClient
}

type edgeFirewallEdgeFunctionInstanceResourceModel struct {
	State          types.String                                    `tfsdk:"state"`
	Data           edgeFirewallEdgeFunctionInstanceResourceResults `tfsdk:"data"`
	ID             types.String                                    `tfsdk:"id"`
	EdgeFirewallID types.String                                    `tfsdk:"edge_firewall_id"`
	LastUpdated    types.String                                    `tfsdk:"last_updated"`
}

type edgeFirewallEdgeFunctionInstanceResourceResults struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *edgeFirewallFunctionsInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_firewall_edge_functions_instance"
}

func (r *edgeFirewallFunctionsInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"edge_firewall_id": schema.StringAttribute{
				Description: "The edge firewall identifier.",
				Required:    true,
			},
			"state": schema.StringAttribute{
				Description: "State of the edge function instance.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The edge function instance identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Required:    true,
					},
					"args": schema.StringAttribute{
						Description: "JSON arguments of the function.",
						Optional:    true,
					},
					"function": schema.Int64Attribute{
						Description: "The edge function identifier.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the edge function instance is active.",
						Optional:    true,
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the edge firewall edge functions instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the edge firewall edge functions instance.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *edgeFirewallFunctionsInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *edgeFirewallFunctionsInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan edgeFirewallEdgeFunctionInstanceResourceModel
	var edgeFirewallId types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsEdgeApplicationID := req.Config.GetAttribute(ctx, path.Root("edge_firewall_id"), &edgeFirewallId)
	resp.Diagnostics.Append(diagsEdgeApplicationID...)
	if resp.Diagnostics.HasError() {
		return
	}

	var argsStr string
	if plan.Data.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		if plan.Data.Args.ValueString() == "" || plan.Data.Args.IsNull() {
			resp.Diagnostics.AddError("Args",
				"Is not null")
			return
		}
		argsStr = plan.Data.Args.ValueString()
	}

	planJsonArgs, err := utils.UnmarshallJsonArgsFirewall(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"failed to unmarshal json args from plan",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	edgeFunctionInstanceRequest := edgeapi.FirewallFunctionInstanceRequest{
		Name:     plan.Data.Name.ValueString(),
		Function: plan.Data.Function.ValueInt64(),
		Args:     &planJsonArgs,
	}

	edgeFunctionInstancesResponse, response, err := r.client.edgeApi.FirewallsFunctionAPI.
		CreateFirewallFunction(ctx, edgeFirewallId.ValueString()).
		FirewallFunctionInstanceRequest(edgeFunctionInstanceRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseFirewallFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.FirewallsFunctionAPI.
					CreateFirewallFunction(ctx, edgeFirewallId.ValueString()).
					FirewallFunctionInstanceRequest(edgeFunctionInstanceRequest).
					Execute() //nolint
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

	plan.Data = edgeFirewallEdgeFunctionInstanceResourceResults{
		Name:         types.StringValue(edgeFunctionInstancesResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(edgeFunctionInstancesResponse.Data.GetFunction()),
		ID:           types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Data.GetId(), 10)),
		Active:       types.BoolValue(edgeFunctionInstancesResponse.Data.GetActive()),
		LastEditor:   types.StringValue(edgeFunctionInstancesResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(edgeFunctionInstancesResponse.Data.GetLastModified().Format(time.RFC850)),
	}

	plan.State = types.StringValue(edgeFunctionInstancesResponse.GetState())
	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFirewallFunctionsInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state edgeFirewallEdgeFunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var edgeFirewallID string
	var functionsInstancesId string
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		edgeFirewallID = valueFromCmd[0]
		functionsInstancesId = valueFromCmd[1]
	} else {
		edgeFirewallID = state.EdgeFirewallID.ValueString()
		functionsInstancesId = state.Data.ID.ValueString()
	}

	if functionsInstancesId == "" {
		resp.Diagnostics.AddError(
			"Edge Functions Instance id error ",
			"should not be null or empty",
		)
		return
	}

	edgeFunctionInstancesResponse, response, err := r.client.
		edgeApi.FirewallsFunctionAPI.
		RetrieveFirewallFunction(
			ctx, edgeFirewallID, functionsInstancesId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			edgeFunctionInstancesResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseRetrieveFirewallFunctionInstance, *http.Response, error) {
				return r.client.
					edgeApi.FirewallsFunctionAPI.
					RetrieveFirewallFunction(ctx, edgeFirewallID, functionsInstancesId).Execute() //nolint
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
	// For Read operation, we'll set state to "executed" as a default since the retrieve API might not return state
	stateValue := "executed"

	edgeApplicationsEdgeFunctionsInstanceState := edgeFirewallEdgeFunctionInstanceResourceModel{
		EdgeFirewallID: types.StringValue(edgeFirewallID),
		State:          types.StringValue(stateValue),
		ID:             types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Data.GetId(), 10)),
		Data: edgeFirewallEdgeFunctionInstanceResourceResults{
			ID:           types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Data.GetId(), 10)),
			LastEditor:   types.StringValue(edgeFunctionInstancesResponse.Data.GetLastEditor()),
			LastModified: types.StringValue(edgeFunctionInstancesResponse.Data.GetLastModified().Format(time.RFC850)),
			Name:         types.StringValue(edgeFunctionInstancesResponse.Data.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(edgeFunctionInstancesResponse.Data.GetFunction()),
			Active:       types.BoolValue(edgeFunctionInstancesResponse.Data.GetActive()),
		},
	}

	diags = resp.State.Set(ctx, &edgeApplicationsEdgeFunctionsInstanceState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFirewallFunctionsInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan edgeFirewallEdgeFunctionInstanceResourceModel
	var edgeFirewallID types.String
	var functionsInstancesId types.String
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state edgeFirewallEdgeFunctionInstanceResourceModel
	diagsOrigin := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsOrigin...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Data.ID.IsNull() || plan.Data.ID.ValueString() == "" {
		functionsInstancesId = state.Data.ID
	} else {
		functionsInstancesId = plan.Data.ID
	}

	if plan.EdgeFirewallID.IsNull() {
		edgeFirewallID = state.EdgeFirewallID
	} else {
		edgeFirewallID = plan.EdgeFirewallID
	}

	var argsStr string
	if plan.Data.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		if plan.Data.Args.ValueString() == "" || plan.Data.Args.IsNull() {
			resp.Diagnostics.AddError("Args",
				"Is not null")
			return
		}
		argsStr = plan.Data.Args.ValueString()
	}

	requestJsonArgsStr, err := utils.UnmarshallJsonArgsFirewall(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"failed to unmarshal json args from plan",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ApplicationPutInstanceRequest := edgeapi.PatchedFirewallFunctionInstanceRequest{
		Name:     plan.Data.Name.ValueStringPointer(),
		Function: plan.Data.Function.ValueInt64Pointer(),
		Args:     &requestJsonArgsStr,
	}

	edgeFunctionInstancesUpdateResponse, response, err := r.client.edgeApi.FirewallsFunctionAPI.
		PartialUpdateFirewallFunction(ctx, edgeFirewallID.ValueString(), functionsInstancesId.ValueString()).
		PatchedFirewallFunctionInstanceRequest(ApplicationPutInstanceRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			edgeFunctionInstancesUpdateResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseFirewallFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.FirewallsFunctionAPI.
					PartialUpdateFirewallFunction(ctx, edgeFirewallID.ValueString(), functionsInstancesId.ValueString()).
					PatchedFirewallFunctionInstanceRequest(ApplicationPutInstanceRequest).
					Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesUpdateResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Data = edgeFirewallEdgeFunctionInstanceResourceResults{
		Function:     types.Int64Value(edgeFunctionInstancesUpdateResponse.Data.GetFunction()),
		Name:         types.StringValue(edgeFunctionInstancesUpdateResponse.Data.GetName()),
		LastEditor:   types.StringValue(edgeFunctionInstancesUpdateResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(edgeFunctionInstancesUpdateResponse.Data.GetLastModified().Format(time.RFC850)),
		Args:         types.StringValue(jsonArgsStr),
		ID:           types.StringValue(strconv.FormatInt(edgeFunctionInstancesUpdateResponse.Data.GetId(), 10)),
		Active:       types.BoolValue(edgeFunctionInstancesUpdateResponse.Data.GetActive()),
	}

	plan.State = types.StringValue(edgeFunctionInstancesUpdateResponse.GetState())
	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesUpdateResponse.Data.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *edgeFirewallFunctionsInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state edgeFirewallEdgeFunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Data.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Functions Instance id error ",
			"is not null",
		)
		return
	}

	if state.EdgeFirewallID.IsNull() {
		resp.Diagnostics.AddError(
			"Edge Application ID error ",
			"is not null",
		)
		return
	}

	_, response, err := r.client.edgeApi.FirewallsFunctionAPI.
		DestroyFirewallFunction(ctx, state.EdgeFirewallID.ValueString(), state.Data.ID.ValueString()).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgeapi.ResponseDeleteFirewallFunctionInstance, *http.Response, error) {
				return r.client.edgeApi.FirewallsFunctionAPI.
					DestroyFirewallFunction(ctx, state.EdgeFirewallID.ValueString(), state.Data.ID.ValueString()).
					Execute() //nolint
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

func (r *edgeFirewallFunctionsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
