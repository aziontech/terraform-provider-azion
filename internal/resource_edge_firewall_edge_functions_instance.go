package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aziontech/azionapi-go-sdk/edgefunctionsinstance_edgefirewall"
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
	SchemaVersion  types.Int64                                     `tfsdk:"schema_version"`
	EdgeFunction   edgeFirewallEdgeFunctionInstanceResourceResults `tfsdk:"results"`
	ID             types.String                                    `tfsdk:"id"`
	EdgeFirewallID types.Int64                                     `tfsdk:"edge_firewall_id"`
	LastUpdated    types.String                                    `tfsdk:"last_updated"`
}

type edgeFirewallEdgeFunctionInstanceResourceResults struct {
	EdgeFunctionId types.Int64  `tfsdk:"edge_function_id"`
	Name           types.String `tfsdk:"name"`
	Args           types.String `tfsdk:"args"`
	ID             types.Int64  `tfsdk:"id"`
	LastEditor     types.String `tfsdk:"last_editor"`
	LastModified   types.String `tfsdk:"last_modified"`
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
			"edge_firewall_id": schema.Int64Attribute{
				Description: "The edge firewall identifier.",
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
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the edge firewall edge functions instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the edge firewall edge functions instance.",
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

func (r *edgeFirewallFunctionsInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *edgeFirewallFunctionsInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan edgeFirewallEdgeFunctionInstanceResourceModel
	var edgeFirewallId types.Int64
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

	if plan.EdgeFunction.Args.ValueString() == "" || plan.EdgeFunction.Args.IsNull() {
		resp.Diagnostics.AddError("Args", "Is not null")
		return
	}
	argsStr := "{}"
	if !plan.EdgeFunction.Args.IsUnknown() {
		argsStr = plan.EdgeFunction.Args.ValueString()
	}

	planJsonArgs, err := utils.ConvertStringToInterface(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
		return
	}

	edgeFunctionInstanceRequest := edgefunctionsinstance_edgefirewall.CreateEdgeFunctionsInstancesRequest{
		Name:         plan.EdgeFunction.Name.ValueStringPointer(),
		EdgeFunction: plan.EdgeFunction.EdgeFunctionId.ValueInt64Pointer(),
		JsonArgs:     planJsonArgs,
	}

	edgeFunctionInstancesResponse, response, err := r.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
		EdgeFirewallEdgeFirewallIdFunctionsInstancesPost(ctx, edgeFirewallId.ValueInt64()).
		CreateEdgeFunctionsInstancesRequest(edgeFunctionInstanceRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgefunctionsinstance_edgefirewall.EdgeFunctionsInstanceResponse, *http.Response, error) {
				return r.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
					EdgeFirewallEdgeFirewallIdFunctionsInstancesPost(ctx, edgeFirewallId.ValueInt64()).
					CreateEdgeFunctionsInstancesRequest(edgeFunctionInstanceRequest).
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Results.GetJsonArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.EdgeFunction = edgeFirewallEdgeFunctionInstanceResourceResults{
		EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Results.GetEdgeFunction()),
		Name:           types.StringValue(edgeFunctionInstancesResponse.Results.GetName()),
		Args:           types.StringValue(jsonArgsStr),
		ID:             types.Int64Value(edgeFunctionInstancesResponse.Results.GetId()),
		LastEditor:     types.StringValue(edgeFunctionInstancesResponse.Results.GetLastEditor()),
		LastModified:   types.StringValue(edgeFunctionInstancesResponse.Results.GetLastModified()),
	}

	plan.SchemaVersion = types.Int64Value(int64(edgeFunctionInstancesResponse.GetSchemaVersion()))
	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Results.GetId(), 10))
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
	var edgeFirewallID int64
	var functionsInstancesId int64
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		edgeFirewallID = int64(utils.AtoiNoError(valueFromCmd[0], resp))
		functionsInstancesId = int64(utils.AtoiNoError(valueFromCmd[1], resp))
	} else {
		edgeFirewallID = state.EdgeFirewallID.ValueInt64()
		functionsInstancesId = state.EdgeFunction.ID.ValueInt64()
	}

	if functionsInstancesId == 0 {
		resp.Diagnostics.AddError(
			"Edge Functions Instance id error ",
			"is not null",
		)
		return
	}

	edgeFunctionInstancesResponse, response, err := r.client.
		edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
		EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdGet(
			ctx, edgeFirewallID, functionsInstancesId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgefunctionsinstance_edgefirewall.EdgeFunctionsInstanceResponse, *http.Response, error) {
				return r.client.
					edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
					EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdGet(ctx, edgeFirewallID, functionsInstancesId).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesResponse.Results.GetJsonArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	edgeApplicationsEdgeFunctionsInstanceState := edgeFirewallEdgeFunctionInstanceResourceModel{
		EdgeFirewallID: types.Int64Value(edgeFirewallID),
		SchemaVersion:  types.Int64Value(int64(edgeFunctionInstancesResponse.GetSchemaVersion())),
		ID:             types.StringValue(strconv.FormatInt(edgeFunctionInstancesResponse.Results.GetId(), 10)),
		EdgeFunction: edgeFirewallEdgeFunctionInstanceResourceResults{
			ID:             types.Int64Value(edgeFunctionInstancesResponse.Results.GetId()),
			LastEditor:     types.StringValue(edgeFunctionInstancesResponse.Results.GetLastEditor()),
			LastModified:   types.StringValue(edgeFunctionInstancesResponse.Results.GetLastModified()),
			EdgeFunctionId: types.Int64Value(edgeFunctionInstancesResponse.Results.GetEdgeFunction()),
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

func (r *edgeFirewallFunctionsInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan edgeFirewallEdgeFunctionInstanceResourceModel
	var edgeFirewallID types.Int64
	var functionsInstancesId types.Int64
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

	if plan.EdgeFunction.ID.IsNull() || plan.EdgeFunction.ID.ValueInt64() == 0 {
		functionsInstancesId = state.EdgeFunction.ID
	} else {
		functionsInstancesId = plan.EdgeFunction.ID
	}

	if plan.EdgeFirewallID.IsNull() {
		edgeFirewallID = state.EdgeFirewallID
	} else {
		edgeFirewallID = plan.EdgeFirewallID
	}

	if plan.EdgeFunction.Args.ValueString() == "" || plan.EdgeFunction.Args.IsNull() {
		resp.Diagnostics.AddError("Args", "Is not null")
		return
	}

	var argsStr string
	argsStr = "{}"
	if !plan.EdgeFunction.Args.IsUnknown() {
		argsStr = plan.EdgeFunction.Args.ValueString()
	}

	requestJsonArgsStr, err := utils.ConvertStringToInterface(argsStr)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
		return
	}

	ApplicationPutInstanceRequest := edgefunctionsinstance_edgefirewall.CreateEdgeFunctionsInstancesRequest{
		Name:         plan.EdgeFunction.Name.ValueStringPointer(),
		EdgeFunction: plan.EdgeFunction.EdgeFunctionId.ValueInt64Pointer(),
		JsonArgs:     requestJsonArgsStr,
	}

	edgeFunctionInstancesUpdateResponse, response, err := r.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
		EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdPut(ctx, edgeFirewallID.ValueInt64(), functionsInstancesId.ValueInt64()).
		Body(ApplicationPutInstanceRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgefunctionsinstance_edgefirewall.EdgeFunctionsInstanceResponse, *http.Response, error) {
				return r.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
					EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdPut(ctx, edgeFirewallID.ValueInt64(), functionsInstancesId.ValueInt64()).
					Body(ApplicationPutInstanceRequest).Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(edgeFunctionInstancesUpdateResponse.Results.GetJsonArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.EdgeFunction = edgeFirewallEdgeFunctionInstanceResourceResults{
		EdgeFunctionId: types.Int64Value(edgeFunctionInstancesUpdateResponse.Results.GetEdgeFunction()),
		Name:           types.StringValue(edgeFunctionInstancesUpdateResponse.Results.GetName()),
		LastEditor:     types.StringValue(edgeFunctionInstancesUpdateResponse.Results.GetLastEditor()),
		LastModified:   types.StringValue(edgeFunctionInstancesUpdateResponse.Results.GetLastModified()),
		Args:           types.StringValue(jsonArgsStr),
		ID:             types.Int64Value(edgeFunctionInstancesUpdateResponse.Results.GetId()),
	}

	plan.SchemaVersion = types.Int64Value(int64(edgeFunctionInstancesUpdateResponse.GetSchemaVersion()))
	plan.ID = types.StringValue(strconv.FormatInt(edgeFunctionInstancesUpdateResponse.Results.GetId(), 10))
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

	if state.EdgeFunction.ID.IsNull() {
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

	response, err := r.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
		EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdDelete(ctx, state.EdgeFirewallID.ValueInt64(), state.EdgeFunction.ID.ValueInt64()).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
				return r.client.edgefunctionsinstanceEdgefirewallApi.DefaultAPI.
					EdgeFirewallEdgeFirewallIdFunctionsInstancesEdgeFunctionInstanceIdDelete(ctx, state.EdgeFirewallID.ValueInt64(), state.EdgeFunction.ID.ValueInt64()).
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
