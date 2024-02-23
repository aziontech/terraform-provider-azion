package provider

import (
	"context"
	"github.com/aziontech/azionapi-go-sdk/variables"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &environmentVariableResource{}
	_ resource.ResourceWithConfigure   = &environmentVariableResource{}
	_ resource.ResourceWithImportState = &environmentVariableResource{}
)

func EnvironmentVariableResource() resource.Resource {
	return &environmentVariableResource{}
}

type environmentVariableResource struct {
	client *apiClient
}

type EnvironmentVariableResourceModel struct {
	EnvironmentVariable *EnvironmentVariableResourceResults `tfsdk:"result"`
	ID                  types.String                        `tfsdk:"id"`
	LastUpdated         types.String                        `tfsdk:"last_updated"`
}

type EnvironmentVariableResourceResults struct {
	Uuid       types.String `tfsdk:"uuid"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
	Secret     types.Bool   `tfsdk:"secret"`
	LastEditor types.String `tfsdk:"last_editor"`
	CreateAt   types.String `tfsdk:"created_at"`
	UpdateAt   types.String `tfsdk:"updated_at"`
}

func (r *environmentVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (r *environmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"result": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						Description: "UUID of the environment variable.",
						Computed:    true,
					},
					"key": schema.StringAttribute{
						Description: "Key of the environment variable.",
						Required:    true,
					},
					"value": schema.StringAttribute{
						Description: "Value of the environment variable.",
						Required:    true,
						//Sensitive:   true,
					},
					"secret": schema.BoolAttribute{
						Description: "Whether the variable is a secret or not.",
						Computed:    true,
						Optional:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last user who edited the variable.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Computed:    true,
						Description: "Informs when the variable was created",
					},
					"updated_at": schema.StringAttribute{
						Computed:    true,
						Description: "The last time the variable was updated.",
					},
				},
			},
		},
	}
}

func (r *environmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *environmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentVariableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.EnvironmentVariable.Secret.ValueBool() {
		resp.Diagnostics.AddWarning(
			"Secret Advice",
			"If you set secret to true, value its controlled by the terraform provider. You can't change it in the terraform provider.",
		)
	}

	environmentVariableRequest := variables.VariableCreate{
		Key:    plan.EnvironmentVariable.Key.ValueString(),
		Value:  plan.EnvironmentVariable.Value.ValueString(),
		Secret: plan.EnvironmentVariable.Secret.ValueBoolPointer(),
	}

	environmentVariableResponse, response, err := r.client.variablesApi.VariablesAPI.ApiVariablesCreate(ctx).VariableCreate(environmentVariableRequest).Execute()
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

	if environmentVariableResponse.Secret {
		plan.EnvironmentVariable = &EnvironmentVariableResourceResults{
			Uuid:       types.StringValue(environmentVariableResponse.GetUuid()),
			Key:        types.StringValue(environmentVariableResponse.GetKey()),
			Value:      types.StringValue(plan.EnvironmentVariable.Value.ValueString()),
			Secret:     types.BoolValue(environmentVariableResponse.GetSecret()),
			LastEditor: types.StringValue(environmentVariableResponse.GetLastEditor()),
			CreateAt:   types.StringValue(environmentVariableResponse.GetCreatedAt().String()),
			UpdateAt:   types.StringValue(environmentVariableResponse.GetUpdatedAt().String()),
		}
	} else {
		plan.EnvironmentVariable = &EnvironmentVariableResourceResults{
			Uuid:       types.StringValue(environmentVariableResponse.GetUuid()),
			Key:        types.StringValue(environmentVariableResponse.GetKey()),
			Value:      types.StringValue(environmentVariableResponse.GetValue()),
			Secret:     types.BoolValue(environmentVariableResponse.GetSecret()),
			LastEditor: types.StringValue(environmentVariableResponse.GetLastEditor()),
			CreateAt:   types.StringValue(environmentVariableResponse.GetCreatedAt().String()),
			UpdateAt:   types.StringValue(environmentVariableResponse.GetUpdatedAt().String()),
		}
	}

	plan.ID = types.StringValue(environmentVariableResponse.GetUuid())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *environmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var uuid string
	if state.ID.IsNull() {
		uuid = state.EnvironmentVariable.Uuid.ValueString()
	} else {
		uuid = state.ID.ValueString()
	}

	getEnvironmentVariable, response, err := r.client.variablesApi.VariablesAPI.ApiVariablesRetrieve(ctx, uuid).Execute()
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

	if state.EnvironmentVariable != nil {
		EnvironmentVariableState := EnvironmentVariableResourceModel{
			EnvironmentVariable: &EnvironmentVariableResourceResults{
				Uuid:       types.StringValue(getEnvironmentVariable.GetUuid()),
				Key:        types.StringValue(getEnvironmentVariable.GetKey()),
				Value:      types.StringValue(state.EnvironmentVariable.Value.ValueString()),
				Secret:     types.BoolValue(getEnvironmentVariable.GetSecret()),
				CreateAt:   types.StringValue(getEnvironmentVariable.GetCreatedAt().String()),
				UpdateAt:   types.StringValue(getEnvironmentVariable.GetUpdatedAt().String()),
				LastEditor: types.StringValue(getEnvironmentVariable.GetLastEditor()),
			},
			LastUpdated: types.StringValue(state.LastUpdated.ValueString()),
			ID:          types.StringValue(uuid),
		}
		diags = resp.State.Set(ctx, &EnvironmentVariableState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		EnvironmentVariableState := EnvironmentVariableResourceModel{
			EnvironmentVariable: &EnvironmentVariableResourceResults{
				Uuid:       types.StringValue(getEnvironmentVariable.GetUuid()),
				Key:        types.StringValue(getEnvironmentVariable.GetKey()),
				Value:      types.StringValue(getEnvironmentVariable.GetValue()),
				Secret:     types.BoolValue(getEnvironmentVariable.GetSecret()),
				CreateAt:   types.StringValue(getEnvironmentVariable.GetCreatedAt().String()),
				UpdateAt:   types.StringValue(getEnvironmentVariable.GetUpdatedAt().String()),
				LastEditor: types.StringValue(getEnvironmentVariable.GetLastEditor()),
			},
			ID: types.StringValue(uuid),
		}
		diags = resp.State.Set(ctx, &EnvironmentVariableState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *environmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentVariableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state EnvironmentVariableResourceModel
	diagsNetworkList := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsNetworkList...)
	if resp.Diagnostics.HasError() {
		return
	}

	var uuid string
	if state.ID.IsNull() {
		uuid = state.EnvironmentVariable.Uuid.ValueString()
	} else {
		uuid = state.ID.ValueString()
	}

	if plan.EnvironmentVariable.Secret.ValueBool() {
		resp.Diagnostics.AddWarning(
			"Secret Advice",
			"If you set secret to true, value its controlled by the terraform provider. You can't change it in the terraform provider.",
		)
	}

	environmentVariableRequest := variables.VariableCreate{
		Key:    plan.EnvironmentVariable.Key.ValueString(),
		Value:  plan.EnvironmentVariable.Value.ValueString(),
		Secret: plan.EnvironmentVariable.Secret.ValueBoolPointer(),
	}

	environmentVariableResponse, response, err := r.client.variablesApi.VariablesAPI.ApiVariablesUpdate(ctx, uuid).VariableCreate(environmentVariableRequest).Execute()
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

	if environmentVariableResponse.Secret {
		plan.EnvironmentVariable = &EnvironmentVariableResourceResults{
			Uuid:       types.StringValue(environmentVariableResponse.GetUuid()),
			Key:        types.StringValue(environmentVariableResponse.GetKey()),
			Value:      types.StringValue(plan.EnvironmentVariable.Value.ValueString()),
			Secret:     types.BoolValue(environmentVariableResponse.GetSecret()),
			LastEditor: types.StringValue(environmentVariableResponse.GetLastEditor()),
			CreateAt:   types.StringValue(environmentVariableResponse.GetCreatedAt().String()),
			UpdateAt:   types.StringValue(environmentVariableResponse.GetUpdatedAt().String()),
		}
	} else {
		plan.EnvironmentVariable = &EnvironmentVariableResourceResults{
			Uuid:       types.StringValue(environmentVariableResponse.GetUuid()),
			Key:        types.StringValue(environmentVariableResponse.GetKey()),
			Value:      types.StringValue(environmentVariableResponse.GetValue()),
			Secret:     types.BoolValue(environmentVariableResponse.GetSecret()),
			LastEditor: types.StringValue(environmentVariableResponse.GetLastEditor()),
			CreateAt:   types.StringValue(environmentVariableResponse.GetCreatedAt().String()),
			UpdateAt:   types.StringValue(environmentVariableResponse.GetUpdatedAt().String()),
		}
	}

	plan.ID = types.StringValue(environmentVariableResponse.GetUuid())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *environmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var uuid string
	if state.ID.IsNull() {
		uuid = state.EnvironmentVariable.Uuid.ValueString()
	} else {
		uuid = state.ID.ValueString()
	}

	response, err := r.client.variablesApi.VariablesAPI.ApiVariablesDestroy(ctx, uuid).Execute()
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

func (r *environmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
