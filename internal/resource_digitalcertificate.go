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
	_ resource.Resource                = &certificateResource{}
	_ resource.ResourceWithConfigure   = &certificateResource{}
	_ resource.ResourceWithImportState = &certificateResource{}
)

// NewCertificateResource creates a new certificate resource.
func NewCertificateResource() resource.Resource {
	return &certificateResource{}
}

// certificateResource is the resource implementation.
type certificateResource struct {
	client *apiClient
}

// certificateResourceModel represents the Terraform state model.
type certificateResourceModel struct {
	SchemaVersion types.Int64              `tfsdk:"schema_version"`
	Results       *certificateResultsModel `tfsdk:"results"`
	ID            types.String             `tfsdk:"id"`
	LastUpdated   types.String             `tfsdk:"last_updated"`
}

// certificateResultsModel represents the certificate data in Terraform state.
type certificateResultsModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
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
	RenewedAt          types.String `tfsdk:"renewed_at"`
	CertificateContent types.String `tfsdk:"certificate_content"`
	PrivateKey         types.String `tfsdk:"private_key"`
}

// Helper function to create NullableString from pointer.
func newNullableString(s *string) azionapi.NullableString {
	return *azionapi.NewNullableString(s)
}

func (r *certificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digital_certificate"
}

// Schema for digital certificate.
func (r *certificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a digital certificate resource. This resource allows you to create, update, and delete digital certificates.\n\n" +
			"~> **Note about private_key and certificate_content:**\n" +
			"Parameters `private_key` and `certificate_content` are sensitive and can be specified using `local_file` from the [local provider](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file).",
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
				Description: "The certificate details.",
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
						Description: "Status of the certificate.",
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
						Description: "Whether the certificate is managed.",
						Computed:    true,
					},
					"csr": schema.StringAttribute{
						Description: "Certificate Signing Request (CSR).",
						Computed:    true,
					},
					"challenge": schema.StringAttribute{
						Description: "Challenge type for the certificate.",
						Computed:    true,
					},
					"authority": schema.StringAttribute{
						Description: "Certificate authority.",
						Computed:    true,
					},
					"key_algorithm": schema.StringAttribute{
						Description: "Key algorithm used for the certificate.",
						Computed:    true,
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
					"renewed_at": schema.StringAttribute{
						Description: "Renewal timestamp of the certificate.",
						Computed:    true,
					},
					"certificate_content": schema.StringAttribute{
						Description: "The content of the certificate (PEM format).",
						Required:    true,
						Sensitive:   true,
					},
					"private_key": schema.StringAttribute{
						Description: "Private key of the digital certificate (PEM format).",
						Required:    true,
						Sensitive:   true,
					},
				},
			},
		},
	}
}

func (r *certificateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *certificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan certificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the certificate request for V4 API.
	certificateRequest := azionapi.Certificate{
		Name:        plan.Results.Name.ValueString(),
		Certificate: newNullableString(plan.Results.CertificateContent.ValueStringPointer()),
		PrivateKey:  newNullableString(plan.Results.PrivateKey.ValueStringPointer()),
	}

	// Call the V4 API.
	certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.CreateCertificate(ctx).Certificate(certificateRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificatesAPI.CreateCertificate(ctx).Certificate(certificateRequest).Execute()
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
	plan.Results = populateCertificateResultsFromAPI(ctx, cert, plan.Results.CertificateContent.ValueString(), plan.Results.PrivateKey.ValueString())
	plan.SchemaVersion = types.Int64Value(1)
	plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state certificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCertificateID(state.ID, state.Results.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			err.Error(),
		)
		return
	}

	// Call the V4 API.
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

	// Preserve the private key and certificate content from state since API doesn't return them.
	privateKey := state.Results.PrivateKey.ValueString()
	certificateContent := state.Results.CertificateContent.ValueString()

	// Populate the state from the API response.
	cert := certificateResponse.GetData()
	state.Results = populateCertificateResultsFromAPI(ctx, cert, certificateContent, privateKey)
	state.SchemaVersion = types.Int64Value(1)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan certificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state certificateResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCertificateID(state.ID, state.Results.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			err.Error(),
		)
		return
	}

	// Build the certificate request for V4 API.
	certificateRequest := azionapi.Certificate{
		Name:        plan.Results.Name.ValueString(),
		Certificate: newNullableString(plan.Results.CertificateContent.ValueStringPointer()),
		PrivateKey:  newNullableString(plan.Results.PrivateKey.ValueStringPointer()),
	}

	// Call the V4 API (using PUT for full update).
	certificateResponse, response, err := r.client.api.DigitalCertificatesCertificatesAPI.UpdateCertificate(ctx, certificateID).Certificate(certificateRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
				return r.client.api.DigitalCertificatesCertificatesAPI.UpdateCertificate(ctx, certificateID).Certificate(certificateRequest).Execute()
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
	plan.Results = populateCertificateResultsFromAPI(ctx, cert, plan.Results.CertificateContent.ValueString(), plan.Results.PrivateKey.ValueString())
	plan.SchemaVersion = types.Int64Value(1)
	plan.ID = types.StringValue(fmt.Sprintf("%d", cert.GetId()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the certificate ID from state.
	certificateID, err := parseCertificateID(state.ID, state.Results.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error",
			err.Error(),
		)
		return
	}

	// Call the V4 API to delete the certificate.
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

func (r *certificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// parseCertificateID extracts the certificate ID from either the string ID or the int64 ID.
func parseCertificateID(stringID types.String, int64ID types.Int64) (int64, error) {
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

// populateCertificateResultsFromAPI transforms API response data to Terraform state model.
func populateCertificateResultsFromAPI(ctx context.Context, cert azionapi.Certificate, certificateContent, privateKey string) *certificateResultsModel {
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

	result := &certificateResultsModel{
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
		RenewedAt:          types.StringValue(renewedAt),
		CertificateContent: types.StringValue(certificateContent),
		PrivateKey:         types.StringValue(privateKey),
	}

	// Handle optional fields.
	if cert.Active != nil {
		result.Active = types.BoolValue(*cert.Active)
	}

	return result
}
