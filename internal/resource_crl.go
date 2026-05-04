package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &crlResource{}
	_ resource.ResourceWithConfigure   = &crlResource{}
	_ resource.ResourceWithImportState = &crlResource{}
	_ resource.ResourceWithModifyPlan  = &crlResource{}
)

func NewCrlResource() resource.Resource {
	return &crlResource{}
}

type crlResource struct {
	client *apiClient
}

type crlResourceModel struct {
	Crl         *crlResourceResults `tfsdk:"crl"`
	ID          types.String        `tfsdk:"id"`
	LastUpdated types.String        `tfsdk:"last_updated"`
}

type crlResourceResults struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Active         types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	CreatedAt      types.String `tfsdk:"created_at"`
	LastModified   types.String `tfsdk:"last_modified"`
	ProductVersion types.String `tfsdk:"product_version"`
	Issuer         types.String `tfsdk:"issuer"`
	LastUpdate     types.String `tfsdk:"last_update"`
	NextUpdate     types.String `tfsdk:"next_update"`
	Crl            types.String `tfsdk:"crl"`
}

func (r *crlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crl"
}

func (r *crlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Certificate Revocation List (CRL) resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the certificate revocation list.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"crl": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Identifier of the certificate revocation list.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the certificate revocation list.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Indicates if the certificate revocation list is active. This field cannot be set to false.",
						Optional:    true,
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the certificate revocation list.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "Timestamp of the certificate revocation list creation on the platform.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Timestamp of the last modification made to the certificate content on the platform.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the certificate revocation list.",
						Computed:    true,
					},
					"issuer": schema.StringAttribute{
						Description: "Issuer of the certificate revocation list.",
						Required:    true,
					},
					"last_update": schema.StringAttribute{
						Description: "Timestamp of the last update issued by the certification revocation list issuer.",
						Computed:    true,
					},
					"next_update": schema.StringAttribute{
						Description: "Timestamp of the next scheduled update from the certification revocation list issuer.",
						Computed:    true,
					},
					"crl": schema.StringAttribute{
						Description: "The certificate revocation list content.",
						Required:    true,
					},
				},
			},
		},
	}
}

func (r *crlResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

// ModifyPlan normalizes the CRL content by trimming trailing whitespace to ensure consistency
// between user input and API responses.
func (r *crlResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only normalize on create/update plans
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan crlResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Normalize CRL content if it's set
	if plan.Crl != nil && !plan.Crl.Crl.IsNull() && !plan.Crl.Crl.IsUnknown() {
		normalized := strings.TrimSpace(plan.Crl.Crl.ValueString())
		plan.Crl.Crl = types.StringValue(normalized)
	}

	// Set the modified plan
	diags = resp.Plan.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *crlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan crlResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the request.
	crlRequest := azionapi.NewCertificateRevocationListWithDefaults()
	crlRequest.SetName(plan.Crl.Name.ValueString())
	crlRequest.SetIssuer(plan.Crl.Issuer.ValueString())
	// Normalize CRL content by trimming trailing whitespace to ensure consistency with API response
	crlRequest.SetCrl(strings.TrimSpace(plan.Crl.Crl.ValueString()))
	crlRequest.SetLastModified(time.Now())
	crlRequest.SetLastUpdate(time.Now())
	crlRequest.SetNextUpdate(time.Now())

	// Set optional active field.
	if !plan.Crl.Active.IsNull() && !plan.Crl.Active.IsUnknown() {
		crlRequest.SetActive(plan.Crl.Active.ValueBool())
	}

	createCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
		CreateCertificateRevocationList(ctx).
		CertificateRevocationList(*crlRequest).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			createCrl, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
					CreateCertificateRevocationList(ctx).
					CertificateRevocationList(*crlRequest).
					Execute()
			}, 5)

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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	// Populate the state from the response.
	crlData := createCrl.GetData()
	plan.Crl = populateCrlResourceResults(crlData)
	plan.ID = types.StringValue(strconv.FormatInt(crlData.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *crlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state crlResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var crlID int64
	var err error
	if state.Crl != nil {
		crlID = state.Crl.ID.ValueInt64()
	} else {
		crlID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert CRL ID",
			)
			return
		}
	}

	getCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
		RetrieveCertificateRevocationList(ctx, crlID).
		Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getCrl, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
					RetrieveCertificateRevocationList(ctx, crlID).
					Execute()
			}, 5)

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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	crlData := getCrl.GetData()
	state.Crl = populateCrlResourceResults(crlData)
	state.ID = types.StringValue(strconv.FormatInt(crlData.GetId(), 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *crlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan crlResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state crlResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	var crlID int64
	var err error
	if state.ID.IsNull() {
		crlID = state.Crl.ID.ValueInt64()
	} else {
		crlID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert CRL ID",
			)
			return
		}
	}

	// Build the request using PATCH for partial update.
	patchedCrl := azionapi.NewPatchedCertificateRevocationList()
	patchedCrl.SetName(plan.Crl.Name.ValueString())
	patchedCrl.SetIssuer(plan.Crl.Issuer.ValueString())
	// Normalize CRL content by trimming trailing whitespace to ensure consistency with API response
	patchedCrl.SetCrl(strings.TrimSpace(plan.Crl.Crl.ValueString()))

	// Set optional active field.
	if !plan.Crl.Active.IsNull() && !plan.Crl.Active.IsUnknown() {
		patchedCrl.SetActive(plan.Crl.Active.ValueBool())
	}

	updateCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
		PartialUpdateCertificateRevocationList(ctx, crlID).
		PatchedCertificateRevocationList(*patchedCrl).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			updateCrl, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
					PartialUpdateCertificateRevocationList(ctx, crlID).
					PatchedCertificateRevocationList(*patchedCrl).
					Execute()
			}, 5)

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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	crlData := updateCrl.GetData()
	plan.Crl = populateCrlResourceResults(crlData)
	plan.ID = types.StringValue(strconv.FormatInt(crlData.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *crlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state crlResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var crlID int64
	var err error
	if state.Crl != nil {
		crlID = state.Crl.ID.ValueInt64()
	} else {
		crlID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert CRL ID",
			)
			return
		}
	}

	_, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
		DeleteCertificateRevocationList(ctx, crlID).
		Execute()
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
					DeleteCertificateRevocationList(ctx, crlID).
					Execute()
			}, 5)

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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}
}

func (r *crlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	crlID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			"The ID must be a valid integer",
		)
		return
	}

	// Retrieve the CRL to populate the state.
	getCrl, response, err := r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
		RetrieveCertificateRevocationList(ctx, crlID).
		Execute()
	if err != nil {
		if response != nil {
			if response.StatusCode == 429 {
				getCrl, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
					return r.client.api.DigitalCertificatesCertificateRevocationListsAPI.
						RetrieveCertificateRevocationList(ctx, crlID).
						Execute()
				}, 5)

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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	crlData := getCrl.GetData()
	state := crlResourceModel{
		Crl: populateCrlResourceResults(crlData),
		ID:  types.StringValue(strconv.FormatInt(crlData.GetId(), 10)),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateCrlResourceResults transforms API response data to Terraform resource model.
func populateCrlResourceResults(crl azionapi.CertificateRevocationList) *crlResourceResults {
	var createdAt string
	if crl.CreatedAt.IsSet() && crl.CreatedAt.Get() != nil {
		createdAt = (*crl.CreatedAt.Get()).Format(time.RFC3339)
	}

	result := &crlResourceResults{
		ID:             types.Int64Value(crl.GetId()),
		Name:           types.StringValue(crl.GetName()),
		LastEditor:     types.StringValue(crl.GetLastEditor()),
		CreatedAt:      types.StringValue(createdAt),
		LastModified:   types.StringValue(crl.GetLastModified().Format(time.RFC3339)),
		ProductVersion: types.StringValue(crl.GetProductVersion()),
		Issuer:         types.StringValue(crl.GetIssuer()),
		LastUpdate:     types.StringValue(crl.GetLastUpdate().Format(time.RFC3339)),
		NextUpdate:     types.StringValue(crl.GetNextUpdate().Format(time.RFC3339)),
		// Normalize CRL content by trimming trailing whitespace to ensure consistency
		Crl: types.StringValue(strings.TrimSpace(crl.GetCrl())),
	}

	// Handle optional fields.
	if crl.Active != nil {
		result.Active = types.BoolValue(*crl.Active)
	}

	return result
}
