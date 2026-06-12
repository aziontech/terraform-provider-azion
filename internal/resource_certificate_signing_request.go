package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	_ resource.Resource                = &certificateSigningRequestResource{}
	_ resource.ResourceWithConfigure   = &certificateSigningRequestResource{}
	_ resource.ResourceWithImportState = &certificateSigningRequestResource{}
)

// NewCertificateSigningRequestResource creates a new CSR resource.
func NewCertificateSigningRequestResource() resource.Resource {
	return &certificateSigningRequestResource{}
}

// certificateSigningRequestResource is the resource implementation.
type certificateSigningRequestResource struct {
	client *apiClient
}

// certificateSigningRequestResourceModel represents the Terraform state model.
type certificateSigningRequestResourceModel struct {
	SchemaVersion types.Int64                            `tfsdk:"schema_version"`
	Results       *certificateSigningRequestResultsModel `tfsdk:"results"`
	ID            types.String                           `tfsdk:"id"`
	LastUpdated   types.String                           `tfsdk:"last_updated"`
}

// certificateSigningRequestResultsModel represents the CSR data in Terraform state.
type certificateSigningRequestResultsModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	CommonName        types.String `tfsdk:"common_name"`
	Country           types.String `tfsdk:"country"`
	State             types.String `tfsdk:"state"`
	Locality          types.String `tfsdk:"locality"`
	Organization      types.String `tfsdk:"organization"`
	OrganizationUnity types.String `tfsdk:"organization_unity"`
	Email             types.String `tfsdk:"email"`
	AlternativeNames  types.List   `tfsdk:"alternative_names"`
	Type              types.String `tfsdk:"certificate_type"`
	KeyAlgorithm      types.String `tfsdk:"key_algorithm"`
	Active            types.Bool   `tfsdk:"active"`
	Certificate       types.String `tfsdk:"certificate"`
	PrivateKey        types.String `tfsdk:"private_key"`
	Issuer            types.String `tfsdk:"issuer"`
	SubjectName       types.List   `tfsdk:"subject_name"`
	Validity          types.String `tfsdk:"validity"`
	Managed           types.Bool   `tfsdk:"managed"`
	Status            types.String `tfsdk:"status"`
	StatusDetail      types.String `tfsdk:"status_detail"`
	CSR               types.String `tfsdk:"csr"`
	Challenge         types.String `tfsdk:"challenge"`
	Authority         types.String `tfsdk:"authority"`
	ProductVersion    types.String `tfsdk:"product_version"`
	LastEditor        types.String `tfsdk:"last_editor"`
	LastModified      types.String `tfsdk:"last_modified"`
	CreatedAt         types.String `tfsdk:"created_at"`
	RenewedAt         types.String `tfsdk:"renewed_at"`
}

func (r *certificateSigningRequestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate_signing_request"
}

// Schema for certificate signing request.
func (r *certificateSigningRequestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a certificate signing request (CSR) resource. This resource allows you to create certificate signing requests.\n\n" +
			"~> **Note:** This resource supports Create, Read, and Delete operations. The CSR API only provides a POST endpoint for creation. " +
			"Read and Delete operations use the standard digital certificates endpoint. Update operations are not supported - any changes will require resource recreation.\n\n" +
			"~> **Note about private_key and certificate:**\n" +
			"Parameters `private_key` and `certificate` are sensitive and can be specified using `local_file` from the [local provider](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema version of the resource.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Description: "The certificate signing request details.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Identifier of the certificate.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the certificate signing request.",
						Required:    true,
					},
					"common_name": schema.StringAttribute{
						Description: "Common Name (CN) for the certificate.",
						Required:    true,
					},
					"country": schema.StringAttribute{
						Description: "Country code (C) for the certificate subject.",
						Required:    true,
					},
					"state": schema.StringAttribute{
						Description: "State or province name (ST) for the certificate subject.",
						Required:    true,
					},
					"locality": schema.StringAttribute{
						Description: "Locality or city name (L) for the certificate subject.",
						Required:    true,
					},
					"organization": schema.StringAttribute{
						Description: "Organization name (O) for the certificate subject.",
						Required:    true,
					},
					"organization_unity": schema.StringAttribute{
						Description: "Organizational unit name (OU) for the certificate subject.",
						Required:    true,
					},
					"email": schema.StringAttribute{
						Description: "Email address for the certificate subject.",
						Required:    true,
					},
					"alternative_names": schema.ListAttribute{
						Description: "Subject Alternative Names (SANs) for the certificate.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"certificate_type": schema.StringAttribute{
						Description: "Type of the certificate. The value can't be changed after the certificate creation. " +
							"Options: `edge_certificate` (Edge Certificate), `trusted_ca_certificate` (Trusted CA Certificate).",
						Optional: true,
						Computed: true,
					},
					"key_algorithm": schema.StringAttribute{
						Description: "Key algorithm for the certificate. Options: " +
							"`rsa_2048` (2048-bit RSA), `rsa_4096` (4096-bit RSA), `ecc_384` (384-bit Prime Field Curve).",
						Optional: true,
						Computed: true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the certificate is active.",
						Optional:    true,
						Computed:    true,
					},
					"certificate": schema.StringAttribute{
						Description: "The certificate content (PEM format).",
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
					},
					"private_key": schema.StringAttribute{
						Description: "Private key for the certificate (PEM format).",
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
					},
					"issuer": schema.StringAttribute{
						Description: "Issuer of the certificate.",
						Computed:    true,
					},
					"subject_name": schema.ListAttribute{
						Description: "Subject name of the certificate.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"validity": schema.StringAttribute{
						Description: "Validity period of the certificate.",
						Computed:    true,
					},
					"managed": schema.BoolAttribute{
						Description: "Whether the certificate is managed.",
						Computed:    true,
					},
					"status": schema.StringAttribute{
						Description: "Status of the certificate. Options: `pending`, `challenge_verification`, `active`, `inactive`, `expired`, `failed`.",
						Computed:    true,
					},
					"status_detail": schema.StringAttribute{
						Description: "Detailed status information.",
						Computed:    true,
					},
					"csr": schema.StringAttribute{
						Description: "The Certificate Signing Request content.",
						Computed:    true,
					},
					"challenge": schema.StringAttribute{
						Description: "Challenge type for ACME certificate. Options: `dns` (DNS challenge), `http` (HTTP challenge).",
						Computed:    true,
					},
					"authority": schema.StringAttribute{
						Description: "Certificate authority. Options: `lets_encrypt`.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the certificate.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the certificate.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the certificate.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "Creation timestamp of the certificate.",
						Computed:    true,
					},
					"renewed_at": schema.StringAttribute{
						Description: "Renewal timestamp of the certificate.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *certificateSigningRequestResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *certificateSigningRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan certificateSigningRequestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the CSR request for V4 API.
	csrRequest := azionapi.CertificateSigningRequest{
		Name:              plan.Results.Name.ValueString(),
		CommonName:        plan.Results.CommonName.ValueString(),
		Country:           plan.Results.Country.ValueString(),
		State:             plan.Results.State.ValueString(),
		Locality:          plan.Results.Locality.ValueString(),
		Organization:      plan.Results.Organization.ValueString(),
		OrganizationUnity: plan.Results.OrganizationUnity.ValueString(),
		Email:             plan.Results.Email.ValueString(),
	}

	// Set optional fields.
	if !plan.Results.AlternativeNames.IsNull() && !plan.Results.AlternativeNames.IsUnknown() {
		var altNames []string
		diags := plan.Results.AlternativeNames.ElementsAs(ctx, &altNames, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		csrRequest.SetAlternativeNames(altNames)
	}

	if !plan.Results.Type.IsNull() && !plan.Results.Type.IsUnknown() {
		csrRequest.SetType(plan.Results.Type.ValueString())
	}

	if !plan.Results.KeyAlgorithm.IsNull() && !plan.Results.KeyAlgorithm.IsUnknown() {
		csrRequest.SetKeyAlgorithm(plan.Results.KeyAlgorithm.ValueString())
	}

	if !plan.Results.Active.IsNull() && !plan.Results.Active.IsUnknown() {
		csrRequest.SetActive(plan.Results.Active.ValueBool())
	}

	if !plan.Results.Certificate.IsNull() && !plan.Results.Certificate.IsUnknown() {
		csrRequest.SetCertificate(plan.Results.Certificate.ValueString())
	}

	if !plan.Results.PrivateKey.IsNull() && !plan.Results.PrivateKey.IsUnknown() {
		csrRequest.SetPrivateKey(plan.Results.PrivateKey.ValueString())
	}

	// Call the V4 API.
	certificateResponse, response, err := r.client.api.DigitalCertificatesCertificateSigningRequestsAPI.CreateCertificateSigningRequest(ctx).CertificateSigningRequest(csrRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificateSigningRequestsAPI.CreateCertificateSigningRequest(ctx).CertificateSigningRequest(csrRequest).Execute()
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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	// Populate the state from the API response.
	cert := certificateResponse.GetData()
	plan.Results = populateCSRResultsFromAPI(ctx, cert, plan.Results)
	plan.SchemaVersion = types.Int64Value(1)
	plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateSigningRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state certificateSigningRequestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCSRID(state.ID, state.Results.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			err.Error(),
		)
		return
	}

	// Call the V4 API - Use the regular certificates endpoint to read.
	certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, certificateID).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, certificateID).Execute()
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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	// Populate the state from the API response while preserving input fields.
	cert := certificateResponse.GetData()
	state.Results = populateCSRResultsFromAPI(ctx, cert, state.Results)
	state.SchemaVersion = types.Int64Value(1)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateSigningRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The CSR API does not have an UPDATE endpoint, so we just return the current state.
	var plan certificateSigningRequestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Return the plan unchanged.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateSigningRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certificateSigningRequestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCSRID(state.ID, state.Results.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			err.Error(),
		)
		return
	}

	// Call the V4 API to delete the certificate - Use the regular certificates endpoint.
	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.DigitalCertificatesCertificatesAPI.DeleteCertificate(ctx, certificateID).Execute()
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

func (r *certificateSigningRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// parseCSRID extracts the certificate ID from either the string ID or the int64 ID.
func parseCSRID(stringID types.String, int64ID types.Int64) (int64, error) {
	if !stringID.IsNull() && !stringID.IsUnknown() {
		var id int64
		_, err := fmt.Sscanf(stringID.ValueString(), "%d", &id)
		if err != nil {
			return 0, fmt.Errorf("could not parse certificate ID: %w", err)
		}
		return id, nil
	}
	if !int64ID.IsNull() && !int64ID.IsUnknown() {
		return int64ID.ValueInt64(), nil
	}
	return 0, fmt.Errorf("no valid certificate ID found in state")
}

// populateCSRResultsFromAPI transforms API response data to Terraform state model.
// It preserves the original input values for fields that are not returned by the API.
func populateCSRResultsFromAPI(ctx context.Context, cert azionapi.Certificate, original *certificateSigningRequestResultsModel) *certificateSigningRequestResultsModel {
	// Convert subject names to types.List.
	var subjectNameList []types.String
	for _, name := range cert.GetSubjectName() {
		subjectNameList = append(subjectNameList, types.StringValue(name))
	}
	subjectName, _ := types.ListValueFrom(ctx, types.StringType, subjectNameList)

	// Handle timestamps using Ok methods for nullable fields.
	var createdAt string
	if createdAtTime, ok := cert.GetCreatedAtOk(); ok && createdAtTime != nil {
		createdAt = createdAtTime.Format(time.RFC3339)
	}

	var lastModified string
	if lastModifiedTime, ok := cert.GetLastModifiedOk(); ok && lastModifiedTime != nil {
		lastModified = lastModifiedTime.Format(time.RFC3339)
	}

	var renewedAt string
	if renewedAtTime, ok := cert.GetRenewedAtOk(); ok && renewedAtTime != nil {
		renewedAt = renewedAtTime.Format(time.RFC3339)
	}

	// Handle active field - use API value if available, otherwise preserve original.
	var active bool
	if cert.HasActive() {
		active = cert.GetActive()
	} else if !original.Active.IsNull() && !original.Active.IsUnknown() {
		active = original.Active.ValueBool()
	}

	// Handle alternative names - preserve original since API doesn't return them.
	var alternativeNames types.List
	if !original.AlternativeNames.IsNull() && !original.AlternativeNames.IsUnknown() {
		alternativeNames = original.AlternativeNames
	} else {
		alternativeNames = types.ListNull(types.StringType)
	}

	result := &certificateSigningRequestResultsModel{
		ID:             types.Int64Value(cert.GetId()),
		Name:           types.StringValue(cert.GetName()),
		Issuer:         types.StringValue(cert.GetIssuer()),
		SubjectName:    subjectName,
		Validity:       types.StringValue(cert.GetValidity()),
		Status:         types.StringValue(cert.GetStatus()),
		StatusDetail:   types.StringValue(cert.GetStatusDetail()),
		Type:           types.StringValue(cert.GetType()),
		Managed:        types.BoolValue(cert.GetManaged()),
		CSR:            types.StringValue(cert.GetCsr()),
		Challenge:      types.StringValue(cert.GetChallenge()),
		Authority:      types.StringValue(cert.GetAuthority()),
		KeyAlgorithm:   types.StringValue(cert.GetKeyAlgorithm()),
		Active:         types.BoolValue(active),
		ProductVersion: types.StringValue(cert.GetProductVersion()),
		LastEditor:     types.StringValue(cert.GetLastEditor()),
		LastModified:   types.StringValue(lastModified),
		CreatedAt:      types.StringValue(createdAt),
		RenewedAt:      types.StringValue(renewedAt),
		// Handle certificate and private_key - use null if unknown, otherwise preserve original
		Certificate: handleOptionalString(original.Certificate),
		PrivateKey:  handleOptionalString(original.PrivateKey),
		// Preserve original input values for CSR fields that are not returned by the API
		CommonName:        original.CommonName,
		Country:           original.Country,
		State:             original.State,
		Locality:          original.Locality,
		Organization:      original.Organization,
		OrganizationUnity: original.OrganizationUnity,
		Email:             original.Email,
		AlternativeNames:  alternativeNames,
	}

	return result
}

// handleOptionalString handles optional string fields that may be unknown.
// If the value is unknown, it returns a null string; otherwise, it preserves the original value.
func handleOptionalString(value types.String) types.String {
	if value.IsUnknown() {
		return types.StringNull()
	}
	return value
}
