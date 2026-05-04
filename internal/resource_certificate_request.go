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
	_ resource.Resource                = &certificateRequestResource{}
	_ resource.ResourceWithConfigure   = &certificateRequestResource{}
	_ resource.ResourceWithImportState = &certificateRequestResource{}
)

// NewCertificateRequestResource creates a new certificate request resource.
func NewCertificateRequestResource() resource.Resource {
	return &certificateRequestResource{}
}

// certificateRequestResource is the resource implementation.
type certificateRequestResource struct {
	client *apiClient
}

// certificateRequestResourceModel represents the Terraform state model.
type certificateRequestResourceModel struct {
	SchemaVersion types.Int64                     `tfsdk:"schema_version"`
	Results       *certificateRequestResultsModel `tfsdk:"results"`
	ID            types.String                    `tfsdk:"id"`
	LastUpdated   types.String                    `tfsdk:"last_updated"`
}

// certificateRequestResultsModel represents the certificate request data in Terraform state.
type certificateRequestResultsModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	CommonName         types.String `tfsdk:"common_name"`
	AlternativeNames   types.List   `tfsdk:"alternative_names"`
	Issuer             types.String `tfsdk:"issuer"`
	SubjectName        types.List   `tfsdk:"subject_name"`
	Validity           types.String `tfsdk:"validity"`
	Status             types.String `tfsdk:"status"`
	StatusDetail       types.String `tfsdk:"status_detail"`
	Type               types.String `tfsdk:"certificate_type"`
	Managed            types.Bool   `tfsdk:"managed"`
	CSR                types.String `tfsdk:"csr"`
	Challenge          types.String `tfsdk:"challenge"`
	Authority          types.String `tfsdk:"authority"`
	KeyAlgorithm       types.String `tfsdk:"key_algorithm"`
	Active             types.Bool   `tfsdk:"active"`
	ProductVersion     types.String `tfsdk:"product_version"`
	LastEditor         types.String `tfsdk:"last_editor"`
	LastModified       types.String `tfsdk:"last_modified"`
	CreatedAt          types.String `tfsdk:"created_at"`
	RenewedAt          types.String `tfsdk:"renewed_at"`
	CertificateContent types.String `tfsdk:"certificate_content"`
	PrivateKey         types.String `tfsdk:"private_key"`
}

func (r *certificateRequestResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate_request"
}

// Schema for certificate request (Let's Encrypt).
func (r *certificateRequestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a certificate request resource for Let's Encrypt certificates. " +
			"This resource allows you to request SSL/TLS certificates from Let's Encrypt automatically.\n\n" +
			"~> **Note:** This resource only supports creation. Update operations are not available. " +
			"Read and Delete operations use the standard digital certificates endpoint.\n\n" +
			"~> **Note about challenge types:**\n" +
			"Use `dns` challenge for DNS-based validation or `http` challenge for HTTP-based validation. " +
			"The challenge type determines how Let's Encrypt will verify domain ownership.",
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
				Description: "The certificate request details.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Identifier of the certificate.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the certificate.",
						Required:    true,
					},
					"common_name": schema.StringAttribute{
						Description: "Common Name (CN) for the certificate. This is the primary domain name.",
						Required:    true,
					},
					"alternative_names": schema.ListAttribute{
						Description: "Subject Alternative Names (SANs) for the certificate. Additional domain names to include.",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
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
						Description: "Validity of the certificate.",
						Computed:    true,
					},
					"status": schema.StringAttribute{
						Description: "Status of the certificate. Options: `pending`, `challenge_verification`, `active`, `inactive`, `expired`, `failed`.",
						Computed:    true,
					},
					"status_detail": schema.StringAttribute{
						Description: "Status detail of the certificate.",
						Computed:    true,
					},
					"certificate_type": schema.StringAttribute{
						Description: "Type of the certificate.",
						Computed:    true,
					},
					"managed": schema.BoolAttribute{
						Description: "Whether the certificate is managed by Azion.",
						Computed:    true,
					},
					"csr": schema.StringAttribute{
						Description: "Certificate Signing Request (CSR).",
						Computed:    true,
					},
					"challenge": schema.StringAttribute{
						Description: "Challenge type for ACME certificate validation. " +
							"Options: `dns` (Uses DNS to solve the ACME challenge), `http` (Uses HTTP to solve the ACME challenge).",
						Required: true,
					},
					"authority": schema.StringAttribute{
						Description: "Certificate authority. Options: `lets_encrypt`.",
						Required:    true,
					},
					"key_algorithm": schema.StringAttribute{
						Description: "Key algorithm used for the certificate. " +
							"Options: `rsa_2048` (2048-bit RSA), `rsa_4096` (4096-bit RSA), `ecc_384` (384-bit Prime Field Curve).",
						Optional: true,
						Computed: true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the certificate is active.",
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
					"certificate_content": schema.StringAttribute{
						Description: "The content of the certificate (PEM format). This field is populated after the certificate is issued.",
						Computed:    true,
						Sensitive:   true,
					},
					"private_key": schema.StringAttribute{
						Description: "Private key of the certificate (PEM format). This field is populated after the certificate is issued.",
						Computed:    true,
						Sensitive:   true,
					},
				},
			},
		},
	}
}

func (r *certificateRequestResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *certificateRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan certificateRequestResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the certificate request for V4 API.
	certificateRequest := azionapi.NewCertificateRequest(
		plan.Results.Name.ValueString(),
		plan.Results.Challenge.ValueString(),
		plan.Results.Authority.ValueString(),
		plan.Results.CommonName.ValueString(),
	)

	// Set optional fields.
	if !plan.Results.AlternativeNames.IsNull() && !plan.Results.AlternativeNames.IsUnknown() {
		var altNames []string
		diags := plan.Results.AlternativeNames.ElementsAs(ctx, &altNames, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		certificateRequest.SetAlternativeNames(altNames)
	}

	if !plan.Results.KeyAlgorithm.IsNull() && !plan.Results.KeyAlgorithm.IsUnknown() {
		certificateRequest.SetKeyAlgorithm(plan.Results.KeyAlgorithm.ValueString())
	}

	// Call the V4 API - Request Certificate endpoint (Let's Encrypt).
	certificateResponse, response, err := r.client.api.DigitalCertificatesRequestACertificateAPI.RequestCertificate(ctx).CertificateRequest(*certificateRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesRequestACertificateAPI.RequestCertificate(ctx).CertificateRequest(*certificateRequest).Execute()
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
	plan.Results = populateCertificateRequestResultsFromAPI(ctx, cert)
	plan.SchemaVersion = types.Int64Value(1)
	plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state certificateRequestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCertificateRequestID(state.ID, state.Results.ID)
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

	// Populate the state from the API response.
	cert := certificateResponse.GetData()
	state.Results = populateCertificateRequestResultsFromAPI(ctx, cert)
	state.SchemaVersion = types.Int64Value(1)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The Certificate Request API does not have an UPDATE endpoint.
	// Let's Encrypt certificates cannot be updated - they must be recreated.
	resp.Diagnostics.AddError(
		"Update not supported",
		"Certificate requests cannot be updated. To change a certificate, you must destroy and recreate the resource.",
	)
}

func (r *certificateRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certificateRequestResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCertificateRequestID(state.ID, state.Results.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			err.Error(),
		)
		return
	}

	// Call the V4 API to delete the certificate - Use the regular certificates endpoint.
	_, response, err := r.client.api.DigitalCertificatesCertificatesAPI.DeleteCertificate(ctx, certificateID).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificatesAPI.DeleteCertificate(ctx, certificateID).Execute()
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
}

func (r *certificateRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// parseCertificateRequestID extracts the certificate ID from either the string ID or the int64 ID.
func parseCertificateRequestID(stringID types.String, int64ID types.Int64) (int64, error) {
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

// populateCertificateRequestResultsFromAPI transforms API response data to Terraform state model.
func populateCertificateRequestResultsFromAPI(ctx context.Context, cert azionapi.Certificate) *certificateRequestResultsModel {
	// Convert subject names to types.List.
	var subjectNameList types.List
	subjectNames := cert.GetSubjectName()
	if len(subjectNames) > 0 {
		subjectNameList, _ = types.ListValueFrom(ctx, types.StringType, subjectNames)
	} else {
		subjectNameList = types.ListNull(types.StringType)
	}

	var renewedAt string
	if cert.RenewedAt.IsSet() && cert.RenewedAt.Get() != nil {
		renewedAt = (*cert.RenewedAt.Get()).Format(time.RFC3339)
	}

	var createdAt string
	if cert.CreatedAt.IsSet() && cert.CreatedAt.Get() != nil {
		createdAt = (*cert.CreatedAt.Get()).Format(time.RFC3339)
	}

	result := &certificateRequestResultsModel{
		ID:                 types.Int64Value(cert.GetId()),
		Name:               types.StringValue(cert.GetName()),
		Issuer:             types.StringValue(cert.GetIssuer()),
		SubjectName:        subjectNameList,
		Validity:           types.StringValue(cert.GetValidity()),
		Status:             types.StringValue(cert.GetStatus()),
		StatusDetail:       types.StringValue(cert.GetStatusDetail()),
		Type:               types.StringValue(cert.GetType()),
		Managed:            types.BoolValue(cert.GetManaged()),
		CSR:                types.StringValue(cert.GetCsr()),
		Challenge:          types.StringValue(cert.GetChallenge()),
		Authority:          types.StringValue(cert.GetAuthority()),
		KeyAlgorithm:       types.StringValue(cert.GetKeyAlgorithm()),
		ProductVersion:     types.StringValue(cert.GetProductVersion()),
		LastEditor:         types.StringValue(cert.GetLastEditor()),
		LastModified:       types.StringValue(cert.GetLastModified().Format(time.RFC3339)),
		CreatedAt:          types.StringValue(createdAt),
		RenewedAt:          types.StringValue(renewedAt),
		CertificateContent: types.StringValue(cert.GetCertificate()),
		PrivateKey:         types.StringValue(cert.GetPrivateKey()),
	}

	// Handle optional fields.
	if cert.Active != nil {
		result.Active = types.BoolValue(*cert.Active)
	}

	// Alternative names - the Certificate struct doesn't have alternative_names field.
	// Use subject names (excluding the first one which is the common name) as alternative names.
	if len(subjectNames) > 1 {
		altNamesList, _ := types.ListValueFrom(ctx, types.StringType, subjectNames[1:])
		result.AlternativeNames = altNamesList
	} else {
		result.AlternativeNames = types.ListNull(types.StringType)
	}

	// Common name - use the first subject name as common name.
	if len(subjectNames) > 0 {
		result.CommonName = types.StringValue(subjectNames[0])
	} else {
		result.CommonName = types.StringValue("")
	}

	return result
}
