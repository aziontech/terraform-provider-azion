package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &dataStreamTemplateResource{}
	_ resource.ResourceWithConfigure   = &dataStreamTemplateResource{}
	_ resource.ResourceWithImportState = &dataStreamTemplateResource{}
)

func NewDataStreamTemplateResource() resource.Resource {
	return &dataStreamTemplateResource{}
}

type dataStreamTemplateResource struct {
	client *apiClient
}

// Main resource model.
type dataStreamTemplateResourceModel struct {
	Template      *dataStreamTemplateResourceResults `tfsdk:"template"`
	ID            types.String                       `tfsdk:"id"`
	LastUpdated   types.String                       `tfsdk:"last_updated"`
	SchemaVersion types.Int64                        `tfsdk:"schema_version"`
}

// Template results - the template body plus computed metadata.
type dataStreamTemplateResourceResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Active       types.Bool   `tfsdk:"active"`
	DataSet      types.String `tfsdk:"data_set"`
	Custom       types.Bool   `tfsdk:"custom"`
	LastEditor   types.String `tfsdk:"last_editor"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *dataStreamTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_stream_template"
}

func (r *dataStreamTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Data Stream template. A template defines the payload shape (`data_set`) that the `render_template` transform of a data stream applies to each record.",
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
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"template": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The template identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the template.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the template.",
						Optional:    true,
						Computed:    true,
					},
					"data_set": schema.StringAttribute{
						Description: "The payload template. A string holding the record layout with `$variable` placeholders. The API stores the content verbatim but strips leading and trailing whitespace.",
						Required:    true,
					},
					"custom": schema.BoolAttribute{
						Description: "Whether the template is user-defined. Always true for templates managed by Terraform; Azion's built-in templates are false.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the template.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the template.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the template.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *dataStreamTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *dataStreamTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dataStreamTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Template == nil {
		resp.Diagnostics.AddError("Invalid configuration", "template block is required")
		return
	}

	templateReq := azionapi.NewTemplateRequest(
		plan.Template.Name.ValueString(),
		plan.Template.DataSet.ValueString(),
	)
	if !plan.Template.Active.IsNull() && !plan.Template.Active.IsUnknown() {
		templateReq.SetActive(plan.Template.Active.ValueBool())
	}

	createTemplate, response, err := r.client.api.DataStreamTemplatesAPI.CreateTemplate(ctx).TemplateRequest(*templateReq).Execute() //nolint
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			createTemplate, response, err = utils.RetryOn429(func() (*azionapi.TemplateResponse, *http.Response, error) {
				return r.client.api.DataStreamTemplatesAPI.CreateTemplate(ctx).TemplateRequest(*templateReq).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	populateDataStreamTemplateFromResponse(plan.Template, createTemplate.GetData())
	plan.ID = types.StringValue(strconv.FormatInt(plan.Template.ID.ValueInt64(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dataStreamTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dataStreamTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var templateID int64
	var err error
	if state.Template != nil && !state.Template.ID.IsNull() {
		templateID = state.Template.ID.ValueInt64()
	} else {
		templateID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Data Stream Template ID",
			)
			return
		}
	}

	getTemplate, response, err := r.client.api.DataStreamTemplatesAPI.RetrieveTemplate(ctx, templateID).Execute() //nolint
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getTemplate, response, err = utils.RetryOn429(func() (*azionapi.TemplateResponse, *http.Response, error) {
				return r.client.api.DataStreamTemplatesAPI.RetrieveTemplate(ctx, templateID).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	populateDataStreamTemplateFromResponse(state.Template, getTemplate.GetData())
	state.ID = types.StringValue(strconv.FormatInt(state.Template.ID.ValueInt64(), 10))
	state.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dataStreamTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dataStreamTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dataStreamTemplateResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := state.Template.ID.ValueInt64()

	templateReq := azionapi.NewPatchedTemplateRequest()
	templateReq.SetName(plan.Template.Name.ValueString())
	templateReq.SetDataSet(plan.Template.DataSet.ValueString())
	if !plan.Template.Active.IsNull() && !plan.Template.Active.IsUnknown() {
		templateReq.SetActive(plan.Template.Active.ValueBool())
	}

	updateTemplate, response, err := r.client.api.DataStreamTemplatesAPI.PartialUpdateTemplate(ctx, templateID).PatchedTemplateRequest(*templateReq).Execute() //nolint
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			updateTemplate, response, err = utils.RetryOn429(func() (*azionapi.TemplateResponse, *http.Response, error) {
				return r.client.api.DataStreamTemplatesAPI.PartialUpdateTemplate(ctx, templateID).PatchedTemplateRequest(*templateReq).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	populateDataStreamTemplateFromResponse(plan.Template, updateTemplate.GetData())
	plan.ID = types.StringValue(strconv.FormatInt(plan.Template.ID.ValueInt64(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dataStreamTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dataStreamTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID := state.Template.ID.ValueInt64()

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.DataStreamTemplatesAPI.DeleteTemplate(ctx, templateID).Execute()
	}, 5)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
		addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)
		return
	}
}

func (r *dataStreamTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	templateID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse data stream template ID: %s", req.ID),
		)
		return
	}

	getTemplate, response, err := r.client.api.DataStreamTemplatesAPI.RetrieveTemplate(ctx, templateID).Execute() //nolint
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getTemplate, response, err = utils.RetryOn429(func() (*azionapi.TemplateResponse, *http.Response, error) {
				return r.client.api.DataStreamTemplatesAPI.RetrieveTemplate(ctx, templateID).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamTemplateAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	state := &dataStreamTemplateResourceModel{
		Template: &dataStreamTemplateResourceResults{},
	}
	populateDataStreamTemplateFromResponse(state.Template, getTemplate.GetData())
	state.ID = types.StringValue(strconv.FormatInt(templateID, 10))
	state.SchemaVersion = types.Int64Value(0)

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// populateDataStreamTemplateFromResponse refreshes the model with the API response.
// Every field is a plain scalar and is taken verbatim, except `data_set` - see
// preferConfiguredDataSet.
func populateDataStreamTemplateFromResponse(model *dataStreamTemplateResourceResults, template azionapi.Template) {
	if model == nil {
		return
	}

	model.ID = types.Int64Value(template.GetId())
	model.Name = types.StringValue(template.GetName())
	model.Active = types.BoolPointerValue(template.Active)
	model.DataSet = preferConfiguredDataSet(model.DataSet, template.GetDataSet())
	model.Custom = types.BoolValue(template.GetCustom())
	model.LastEditor = types.StringValue(template.GetLastEditor())
	model.CreatedAt = types.StringValue(template.GetCreatedAt().Format(time.RFC850))
	model.LastModified = types.StringValue(template.GetLastModified().Format(time.RFC850))
}

// preferConfiguredDataSet keeps the configured `data_set` when the API's echo
// differs from it only by surrounding whitespace.
//
// The API strips leading and trailing whitespace from `data_set`, so a heredoc
// (`<<-EOT`, which always ends in a newline) comes back one byte shorter than it
// went out. `data_set` is a Required attribute, and Terraform demands that such
// an attribute survive apply byte-for-byte, so storing the trimmed echo fails
// the "Provider produced inconsistent result after apply" check - and once past
// that, would drift on every subsequent plan.
//
// A genuine server-side rewrite still differs after trimming, so it is stored
// and surfaces as real drift. On import there is no prior value, so the API's
// value is used as-is.
func preferConfiguredDataSet(prior types.String, apiValue string) types.String {
	if !prior.IsNull() && !prior.IsUnknown() &&
		strings.TrimSpace(prior.ValueString()) == strings.TrimSpace(apiValue) {
		return prior
	}
	return types.StringValue(apiValue)
}

// addDataStreamTemplateAPIError adds an appropriate error to diagnostics based on the API response.
func addDataStreamTemplateAPIError(diagnostics *diag.Diagnostics, err error, response *http.Response) {
	if response == nil {
		diagnostics.AddError(err.Error(), "No response received")
		return
	}

	if response.StatusCode == http.StatusTooManyRequests {
		diagnostics.AddError(err.Error(), "API request rate limited")
		return
	}

	bodyBytes, errReadAll := io.ReadAll(response.Body)
	if errReadAll != nil {
		diagnostics.AddError(errReadAll.Error(), "err")
		return
	}
	diagnostics.AddError(err.Error(), string(bodyBytes))
}
