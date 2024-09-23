package provider

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/digital_certificates"
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
	_ resource.Resource                = &digitalCertificateResource{}
	_ resource.ResourceWithConfigure   = &digitalCertificateResource{}
	_ resource.ResourceWithImportState = &digitalCertificateResource{}
)

func NewDigitalCertificateResource() resource.Resource {
	return &digitalCertificateResource{}
}

type digitalCertificateResource struct {
	client *apiClient
}

type digitalCertificateResourceModel struct {
	SchemaVersion     types.Int64                        `tfsdk:"schema_version"`
	CertificateResult *digitalCertificateResourceResults `tfsdk:"certificate_result"`
	ID                types.String                       `tfsdk:"id"`
	LastUpdated       types.String                       `tfsdk:"last_updated"`
}

type digitalCertificateResourceResults struct {
	CertificateID      types.Int64    `tfsdk:"certificate_id"`
	Name               types.String   `tfsdk:"name"`
	Issuer             types.String   `tfsdk:"issuer"`
	SubjectName        []types.String `tfsdk:"subject_name"`
	Validity           types.String   `tfsdk:"validity"`
	Status             types.String   `tfsdk:"status"`
	CertificateType    types.String   `tfsdk:"certificate_type"`
	Managed            types.Bool     `tfsdk:"managed"`
	CSR                types.String   `tfsdk:"csr"`
	CertificateContent types.String   `tfsdk:"certificate_content"`
	PrivateKey         types.String   `tfsdk:"private_key"`
	AzionInformation   types.String   `tfsdk:"azion_information"`
}

func (r *digitalCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digital_certificate"
}

func (r *digitalCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "" +
			"~> **Note about private_key and certificate_content:**\n" +
			"Parameter `private_key` and `certificate_content` can be specified with local_file in - https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"certificate_result": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"certificate_id": schema.Int64Attribute{
						Description: "The function identifier.",
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
						Optional:    true,
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
					"certificate_content": schema.StringAttribute{
						Description: "The content of the certificate.",
						Required:    true,
						Sensitive:   true,
					},
					"private_key": schema.StringAttribute{
						Description: "Private key of the digital certificate.",
						Required:    true,
						Sensitive:   true,
					},
					"azion_information": schema.StringAttribute{
						Description: "Information of the digital certificate.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (r *digitalCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *digitalCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan digitalCertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var privateKey types.String
	var certificateContent types.String

	privateKey = plan.CertificateResult.PrivateKey
	certificateContent = plan.CertificateResult.CertificateContent

	certificateRequest := digital_certificates.CreateCertificateRequest{
		Name:        plan.CertificateResult.Name.ValueString(),
		Certificate: certificateContent.ValueString(),
		PrivateKey:  privateKey.ValueString(),
	}

	certificateResponse, response, err := r.client.digitalCertificatesApi.CreateDigitalCertificateApi.CreateCertificate(ctx).CreateCertificateRequest(certificateRequest).Execute() //nolint
	if err != nil {
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

	var GetSubjectName []types.String
	for _, subjectName := range certificateResponse.Results.GetSubjectName() {
		GetSubjectName = append(GetSubjectName, types.StringValue(subjectName))
	}

	plan.CertificateResult = &digitalCertificateResourceResults{
		CertificateID:      types.Int64Value(int64(certificateResponse.Results.GetId())),
		Name:               types.StringValue(certificateResponse.Results.GetName()),
		Issuer:             types.StringValue(certificateResponse.Results.GetIssuer()),
		SubjectName:        GetSubjectName,
		Validity:           types.StringValue(certificateResponse.Results.GetValidity()),
		Status:             types.StringValue(certificateResponse.Results.GetStatus()),
		CertificateType:    types.StringValue(certificateResponse.Results.GetCertificateType()),
		Managed:            types.BoolValue(certificateResponse.Results.GetManaged()),
		CSR:                types.StringValue(certificateResponse.Results.GetCsr()),
		CertificateContent: types.StringValue(certificateContent.ValueString()),
		PrivateKey:         types.StringValue(privateKey.ValueString()),
		AzionInformation:   types.StringValue(certificateResponse.Results.GetAzionInformation()),
	}

	plan.SchemaVersion = types.Int64Value(int64(*certificateResponse.SchemaVersion))
	plan.ID = types.StringValue(strconv.FormatInt(int64(certificateResponse.Results.GetId()), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *digitalCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state digitalCertificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var CertificateID int64
	var err error
	if state.ID.IsNull() {
		CertificateID = state.CertificateResult.CertificateID.ValueInt64()
	} else {
		CertificateID, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert CertificateID to int",
			)
			return
		}
	}

	certificateResponse, response, err := r.client.digitalCertificatesApi.RetrieveDigitalCertificateByIDApi.GetCertificate(ctx, CertificateID).Execute() //nolint
	if err != nil {
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

	var privateKey types.String
	if state.CertificateResult == nil {
		resp.Diagnostics.AddWarning(
			"PrivateKey is controlled by Terraform",
			"You need to put your private key in the state and set a terraform apply for update the state!",
		)
		privateKey = types.StringValue("")
	} else {
		privateKey = types.StringValue(state.CertificateResult.PrivateKey.ValueString())
	}
	var GetSubjectName []types.String
	for _, subjectName := range certificateResponse.Results.GetSubjectName() {
		GetSubjectName = append(GetSubjectName, types.StringValue(subjectName))
	}
	certificateState := &digitalCertificateResourceModel{
		SchemaVersion: types.Int64Value(int64(*certificateResponse.SchemaVersion)),
		CertificateResult: &digitalCertificateResourceResults{
			CertificateID:      types.Int64Value(int64(certificateResponse.Results.GetId())),
			Name:               types.StringValue(certificateResponse.Results.GetName()),
			Issuer:             types.StringValue(certificateResponse.Results.GetIssuer()),
			SubjectName:        GetSubjectName,
			Validity:           types.StringValue(certificateResponse.Results.GetValidity()),
			Status:             types.StringValue(certificateResponse.Results.GetStatus()),
			CertificateType:    types.StringValue(certificateResponse.Results.GetCertificateType()),
			Managed:            types.BoolValue(certificateResponse.Results.GetManaged()),
			CSR:                types.StringValue(certificateResponse.Results.GetCsr()),
			CertificateContent: types.StringValue(certificateResponse.Results.GetCertificateContent()),
			PrivateKey:         privateKey,
			AzionInformation:   types.StringValue(certificateResponse.Results.GetAzionInformation()),
		},
	}

	certificateState.ID = types.StringValue(strconv.FormatInt(int64(certificateResponse.Results.GetId()), 10))

	if state.CertificateResult == nil {
		diags = resp.State.Set(ctx, &certificateState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *digitalCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan digitalCertificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state digitalCertificateResourceModel
	diagsEdgeFunction := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsEdgeFunction...)
	if resp.Diagnostics.HasError() {
		return
	}

	var certificateID int64
	var err error
	if state.CertificateResult.CertificateID.ValueInt64() != 0 {
		certificateID = state.CertificateResult.CertificateID.ValueInt64()
	} else {
		certificateID, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert CertificateID to int",
			)
			return
		}
	}

	var privateKey types.String
	var certificateContent types.String

	privateKey = plan.CertificateResult.PrivateKey
	certificateContent = plan.CertificateResult.CertificateContent

	certificateRequest := digital_certificates.UpdateDigitalCertificateRequest{
		Name:        digital_certificates.PtrString(plan.CertificateResult.Name.ValueString()),
		Certificate: digital_certificates.PtrString(certificateContent.ValueString()),
		PrivateKey:  digital_certificates.PtrString(privateKey.ValueString()),
	}

	certificateID32, err := utils.CheckInt64toInt32Security(certificateID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error before Overflow",
			fmt.Sprintf("n32 %d exceeds int32 limits", certificateID),
		)
		return
	}

	certificateResponse, response, err := r.client.digitalCertificatesApi.
		UpdateDigitalCertificateApi.UpdateDigitalCertificate(ctx, certificateID32).
		UpdateDigitalCertificateRequest(certificateRequest).Execute() //nolint
	if err != nil {
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

	var GetSubjectName []types.String
	for _, subjectName := range certificateResponse.Results.GetSubjectName() {
		GetSubjectName = append(GetSubjectName, types.StringValue(subjectName))
	}

	plan.CertificateResult = &digitalCertificateResourceResults{
		CertificateID:      types.Int64Value(int64(certificateResponse.Results.GetId())),
		Name:               types.StringValue(certificateResponse.Results.GetName()),
		Issuer:             types.StringValue(certificateResponse.Results.GetIssuer()),
		SubjectName:        GetSubjectName,
		Validity:           types.StringValue(certificateResponse.Results.GetValidity()),
		Status:             types.StringValue(certificateResponse.Results.GetStatus()),
		CertificateType:    types.StringValue(certificateResponse.Results.GetCertificateType()),
		Managed:            types.BoolValue(certificateResponse.Results.GetManaged()),
		CSR:                types.StringValue(certificateResponse.Results.GetCsr()),
		CertificateContent: types.StringValue(certificateContent.ValueString()),
		PrivateKey:         types.StringValue(privateKey.ValueString()),
		AzionInformation:   types.StringValue(certificateResponse.Results.GetAzionInformation()),
	}

	plan.SchemaVersion = types.Int64Value(int64(*certificateResponse.SchemaVersion))
	plan.ID = types.StringValue(strconv.FormatInt(int64(certificateResponse.Results.GetId()), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *digitalCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state digitalCertificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var certificateID int64
	var err error
	if state.CertificateResult.CertificateID.ValueInt64() != 0 {
		certificateID = state.CertificateResult.CertificateID.ValueInt64()
	} else {
		certificateID, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert CertificateID to int",
			)
			return
		}
	}

	certificateID32, err := utils.CheckInt64toInt32Security(certificateID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error before Overflow",
			fmt.Sprintf("n32 %d exceeds int32 limits", certificateID),
		)
		return
	}

	response, err := r.client.digitalCertificatesApi.DeleteDigitalCertificateApi.
		RemoveDigitalCertificates(ctx, certificateID32).Execute() //nolint
	if err != nil {
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

func (r *digitalCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
