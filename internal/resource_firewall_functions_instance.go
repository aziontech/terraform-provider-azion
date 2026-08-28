package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdk "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	State       types.String                          `tfsdk:"state"`
	Data        *FirewallFunctionInstanceResourceData `tfsdk:"data"`
	ID          types.String                          `tfsdk:"id"`
	FirewallID  types.Int64                           `tfsdk:"firewall_id"`
	LastUpdated types.String                          `tfsdk:"last_updated"`
}

type FirewallFunctionInstanceResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Args         types.String `tfsdk:"args"`
	Function     types.Int64  `tfsdk:"function"`
	Active       types.Bool   `tfsdk:"active"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	CreatedAt    types.String `tfsdk:"created_at"`
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
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the firewall function instance.",
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
	if plan.Data == nil {
		resp.Diagnostics.AddError(
			"Missing function instance data",
			"The data attribute is required to create a firewall function instance.",
		)
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

	functionInstanceResponse, response, err := r.createFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceRequest)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if isHTTPStatus(response, http.StatusTooManyRequests) {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.createFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceRequest)
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
			addFirewallFunctionAPIError(&resp.Diagnostics, err, response)
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

	plan.Data = &FirewallFunctionInstanceResourceData{
		Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
		Args:         types.StringValue(jsonArgsStr),
		Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
		ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
		Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
		LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
		CreatedAt:    types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
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
	if strings.Contains(state.ID.ValueString(), "/") {
		var err error
		firewallID, functionInstanceID, err = parseFirewallFunctionImportID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid import ID", err.Error())
			return
		}
	} else {
		firewallID = state.FirewallID.ValueInt64()
		if state.Data != nil {
			functionInstanceID = state.Data.ID.ValueInt64()
		}
	}

	if firewallID == 0 {
		resp.Diagnostics.AddError(
			"Firewall ID error",
			"should not be null or empty",
		)
		return
	}

	if functionInstanceID == 0 {
		resp.Diagnostics.AddError(
			"Function Instance id error ",
			"should not be null or empty",
		)
		return
	}

	functionInstanceResponse, response, err := r.retrieveFirewallFunction(ctx, firewallID, functionInstanceID)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.retrieveFirewallFunction(ctx, firewallID, functionInstanceID)
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
			addFirewallFunctionAPIError(&resp.Diagnostics, err, response)
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
		Data: &FirewallFunctionInstanceResourceData{
			ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
			LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
			LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
			Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
			Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
			CreatedAt:    types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		},
	}

	diags = resp.State.Set(ctx, &readState)
	resp.Diagnostics.Append(diags...)
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
	if plan.Data == nil {
		resp.Diagnostics.AddError(
			"Missing function instance data",
			"The data attribute is required to update a firewall function instance.",
		)
		return
	}

	var state FirewallFunctionInstanceResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always use the function instance ID from state (it's a computed field)
	if state.Data != nil {
		functionInstanceID = state.Data.ID
	}

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

	updateResponse, response, err := r.updateFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64(), patchRequest)
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			updateResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.updateFirewallFunction(ctx, firewallID.ValueInt64(), functionInstanceID.ValueInt64(), patchRequest)
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
			addFirewallFunctionAPIError(&resp.Diagnostics, err, response)
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

	plan.Data = &FirewallFunctionInstanceResourceData{
		Function:     types.Int64Value(updateResponse.Data.GetFunction()),
		Name:         types.StringValue(updateResponse.Data.GetName()),
		LastEditor:   types.StringValue(updateResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(updateResponse.Data.GetLastModified().Format(time.RFC850)),
		Args:         types.StringValue(jsonArgsStr),
		ID:           types.Int64Value(updateResponse.Data.GetId()),
		Active:       types.BoolValue(updateResponse.Data.GetActive()),
		CreatedAt:    types.StringValue(updateResponse.Data.GetCreatedAt().Format(time.RFC3339)),
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

	if state.Data == nil || state.Data.ID.IsNull() {
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

	_, response, err := utils.RetryOn429Delete(func() (*sdk.DeleteResponse, *http.Response, error) {
		return r.client.api.FirewallsFunctionAPI.
			DeleteFirewallFunction(ctx, state.FirewallID.ValueInt64(), state.Data.ID.ValueInt64()).
			Execute()
	}, 5) // Maximum 5 retries
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
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

func (r *FirewallFunctionsInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	firewallID, functionInstanceID, err := parseFirewallFunctionImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import format",
			err.Error(),
		)
		return
	}

	functionInstanceResponse, response, err := r.retrieveFirewallFunction(ctx, firewallID, functionInstanceID)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			functionInstanceResponse, response, err = utils.RetryOn429(func() (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
				return r.retrieveFirewallFunction(ctx, firewallID, functionInstanceID)
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addFirewallFunctionAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	jsonArgsStr, err := utils.ConvertInterfaceToString(functionInstanceResponse.Data.GetArgs())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "err")
		return
	}

	state := FirewallFunctionInstanceResourceModel{
		FirewallID: types.Int64Value(firewallID),
		State:      types.StringValue(functionInstanceResponse.GetState()),
		ID:         types.StringValue(req.ID),
		Data: &FirewallFunctionInstanceResourceData{
			ID:           types.Int64Value(functionInstanceResponse.Data.GetId()),
			LastEditor:   types.StringValue(functionInstanceResponse.Data.GetLastEditor()),
			LastModified: types.StringValue(functionInstanceResponse.Data.GetLastModified().Format(time.RFC850)),
			Name:         types.StringValue(functionInstanceResponse.Data.GetName()),
			Args:         types.StringValue(jsonArgsStr),
			Function:     types.Int64Value(functionInstanceResponse.Data.GetFunction()),
			Active:       types.BoolValue(functionInstanceResponse.Data.GetActive()),
			CreatedAt:    types.StringValue(functionInstanceResponse.Data.GetCreatedAt().Format(time.RFC3339)),
		},
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func parseFirewallFunctionImportID(id string) (int64, int64, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected format: firewallID/functionInstanceID")
	}

	firewallID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid firewall ID %q: %w", parts[0], err)
	}
	if firewallID == 0 {
		return 0, 0, fmt.Errorf("firewall ID must not be zero")
	}

	functionInstanceID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid function instance ID %q: %w", parts[1], err)
	}
	if functionInstanceID == 0 {
		return 0, 0, fmt.Errorf("function instance ID must not be zero")
	}

	return firewallID, functionInstanceID, nil
}

func (r *FirewallFunctionsInstanceResource) createFirewallFunction(ctx context.Context, firewallID int64, request sdk.FirewallFunctionInstanceRequest) (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
	functionInstanceResponse, response, err := r.client.api.FirewallsFunctionAPI.
		CreateFirewallFunction(ctx, firewallID).
		FirewallFunctionInstanceRequest(request).
		Execute() //nolint
	if err == nil {
		closeResponseBody(response)
	}
	return functionInstanceResponse, response, err
}

func (r *FirewallFunctionsInstanceResource) retrieveFirewallFunction(ctx context.Context, firewallID int64, functionInstanceID int64) (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
	functionInstanceResponse, response, err := r.client.api.FirewallsFunctionAPI.
		RetrieveFirewallFunction(ctx, firewallID, functionInstanceID).Execute() //nolint
	if err == nil {
		closeResponseBody(response)
	}
	return functionInstanceResponse, response, err
}

func (r *FirewallFunctionsInstanceResource) updateFirewallFunction(ctx context.Context, firewallID int64, functionInstanceID int64, request sdk.PatchedFirewallFunctionInstanceRequest) (*sdk.FirewallFunctionInstanceResponse, *http.Response, error) {
	functionInstanceResponse, response, err := r.client.api.FirewallsFunctionAPI.
		PartialUpdateFirewallFunction(ctx, firewallID, functionInstanceID).
		PatchedFirewallFunctionInstanceRequest(request).
		Execute() //nolint
	if err == nil {
		closeResponseBody(response)
	}
	return functionInstanceResponse, response, err
}

func closeResponseBody(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}

func isHTTPStatus(response *http.Response, status int) bool {
	return response != nil && response.StatusCode == status
}

func addFirewallFunctionAPIError(diagnostics *diag.Diagnostics, err error, response *http.Response) {
	if response != nil && response.Body != nil {
		defer response.Body.Close()
		bodyBytes, errReadAll := io.ReadAll(response.Body)
		if errReadAll != nil {
			diagnostics.AddError(errReadAll.Error(), "err")
		}
		diagnostics.AddError(err.Error(), string(bodyBytes))
		return
	}
	diagnostics.AddError(err.Error(), "failed firewall function instance API request")
}
