package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &FirewallFunctionsInstanceResource{}
	_ resource.ResourceWithConfigure   = &FirewallFunctionsInstanceResource{}
	_ resource.ResourceWithImportState = &FirewallFunctionsInstanceResource{}
)

func NewFirewallFunctionsInstanceResource() resource.Resource {
	return &FirewallFunctionsInstanceResource{}
}

type FirewallFunctionsInstanceResource struct {
	client *apiClient
}

type FirewallFunctionInstanceResourceModel struct {
	State       types.String                         `tfsdk:"state"`
	Data        FirewallFunctionInstanceResourceData `tfsdk:"data"`
	ID          types.String                         `tfsdk:"id"`
	FirewallID  types.Int64                          `tfsdk:"firewall_id"`
	LastUpdated types.String                         `tfsdk:"last_updated"`
}

type FirewallFunctionInstanceResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *FirewallFunctionsInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_functions_instance"
}

func (r *FirewallFunctionsInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_id": schema.Int64Attribute{
				Description: "The firewall identifier.",
				Required:    true,
			},
			"state": schema.StringAttribute{
				Description: "State of the function instance.",
				Computed:    true,
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
					"name": schema.StringAttribute{
						Description: "Name of the function.",
						Required:    true,
					},
					"args": schema.StringAttribute{
						Description: "JSON arguments of the function.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("{}"),
					},
					"function": schema.Int64Attribute{
						Description: "The function identifier.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the function instance is active.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the firewall function instance.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the firewall function instance.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *FirewallFunctionsInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *FirewallFunctionsInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallFunctionInstanceResourceModel
	var firewallID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsFirewallID := req.Config.GetAttribute(ctx, path.Root("firewall_id"), &firewallID)
	resp.Diagnostics.Append(diagsFirewallID...)
	if resp.Diagnostics.HasError() {
		return
	}

	var argsStr string
	if plan.Data.Args.IsNull() || plan.Data.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		argsStr = plan.Data.Args.ValueString()
		if argsStr == "" {
			argsStr = "{}"
		}
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

	functionInstanceRequest := sdk.FirewallFunctionInstanceRequest{
		Name:     plan.Data.Name.ValueString(),
		Function: plan.Data.Function.ValueInt64(),
		Args:     &planJsonArgs,
		Active:   plan.Data.Active.ValueBoolPointer(),
	}

	functionInstanceResponse, response, err := r.client.api.FirewallsFunctionAPI.
		CreateFirewallFunction(ctx, firewallID.ValueInt64()).
		FirewallFunctionInstanceRequest(functionInstanceRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.client.api.FirewallsFunctionAPI.
					CreateFirewallFunction(ctx, firewallID.ValueInt64()).
					FirewallFunctionInstanceRequest(functionInstanceRequest).
					Execute() //nolint
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

	plan.Data = FirewallFunctionInstanceResourceData{
		Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
		ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
		Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
		LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
	}

	plan.State = types.StringValue(functionInstanceResponse.GetState())
	plan.ID = types.StringValue(strconv.FormatInt(functionInstanceResponse.Data.GetId(), 10))
	plan.FirewallID = firewallID
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *FirewallFunctionsInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallFunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var firewallID int64
	var functionInstanceID int64
	valueFromCmd := strings.Split(state.ID.ValueString(), "/")
	if len(valueFromCmd) > 1 {
		firewallID, _ = strconv.ParseInt(valueFromCmd[0], 10, 64)
		functionInstanceID, _ = strconv.ParseInt(valueFromCmd[1], 10, 64)
	} else {
		firewallID = state.FirewallID.ValueInt64()
		functionInstanceID = state.Data.ID.ValueInt64()
	}

	if functionInstanceID == 0 {
		resp.Diagnostics.AddError(
			"Function Instance id error ",
			"should not be null or empty",
		)
		return
	}

	functionInstanceResponse, response, err := r.client.
		api.FirewallsFunctionAPI.
		RetrieveFirewallFunction(ctx, firewallID, functionInstanceID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.client.
					api.FirewallsFunctionAPI.
					RetrieveFirewallFunction(ctx, firewallID, functionInstanceID).Execute() //nolint
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
	// For Read operation, we'll set state to "executed" as a default since the retrieve API might not return state
	stateValue := "executed"

	readState := FirewallFunctionInstanceResourceModel{
		FirewallID: types.Int64Value(firewallID),
		State:      types.StringValue(stateValue),
		ID:         types.StringValue(strconv.FormatInt(functionInstanceResponse.Data.GetId(), 10)),
		Data: FirewallFunctionInstanceResourceData{
			ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
			LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
			LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
			Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
			Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
		},
	}

	diags = resp.State.Set(ctx, &readState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *FirewallFunctionsInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallFunctionInstanceResourceModel
	var firewallID types.Int64
	var functionInstanceID types.Int64
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallFunctionInstanceResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always use the function instance ID from state (it's a computed field)
	functionInstanceID = state.Data.ID

	// Always use the firewall ID from state (it's required and shouldn't change)
	firewallID = state.FirewallID

	var argsStr string
	if plan.Data.Args.IsNull() || plan.Data.Args.IsUnknown() {
		argsStr = "{}"
	} else {
		argsStr = plan.Data.Args.ValueString()
		if argsStr == "" {
			argsStr = "{}"
		}
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

	patchRequest := sdk.PatchedFirewallFunctionInstanceRequest{
		Name:     plan.Data.Name.ValueStringPointer(),
		Function: plan.Data.Function.ValueInt64Pointer(),
		Args:     &requestJsonArgsStr,
		Active:   plan.Data.Active.ValueBoolPointer(),
	}

	updateResponse, response, err := r.client.api.FirewallsFunctionAPI.
		PartialUpdateFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).
		PatchedFirewallFunctionInstanceRequest(patchRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.client.api.FirewallsFunctionAPI.
					PartialUpdateFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64()).
					PatchedFirewallFunctionInstanceRequest(patchRequest).
					Execute() //nolint
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

	jsonArgsStr, err := utils.ConvertInterfaceToString(updateResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"err",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Data = FirewallFunctionInstanceResourceData{
		Function:     types.Int64Value(updateResponse.Data.GetFunction()),
		Name:         types.StringValue(updateResponse.Data.GetName()),
		LastEditor:   types.StringValue(updateResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(updateResponse.Data.GetLastModified().Format(time.RFC850)),
		Args:         types.StringValue(jsonArgsStr),
		ID:           types.Int64Value(updateResponse.Data.GetId()),
		Active:       types.BoolValue(updateResponse.Data.GetActive()),
	}

	plan.State = types.StringValue(updateResponse.GetState())
	plan.ID = types.StringValue(strconv.FormatInt(updateResponse.Data.GetId(), 10))
	plan.FirewallID = firewallID
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *FirewallFunctionsInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallFunctionInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Data.ID.IsNull() {
		resp.Diagnostics.AddError(
			"Function Instance id error ",
			"is not null",
		)
		return
	}

	if state.FirewallID.IsNull() {
		resp.Diagnostics.AddError(
			"Firewall ID error ",
			"is not null",
		)
		return
	}

	_, response, err := r.client.api.FirewallsFunctionAPI.
		DeleteFirewallFunction(ctx, state.FirewallID.ValueInt64(), state.Data.ID.ValueInt64()).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*sdk.DeleteResponse, *http.Response, error) {
				return r.client.api.FirewallsFunctionAPI.
					DeleteFirewallFunction(ctx, state.FirewallID.ValueInt64(), state.Data.ID.ValueInt64()).
					Execute()
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
}

func (r *FirewallFunctionsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
