package provider

import (
	"context"
	"github.com/aziontech/azionapi-go-sdk/digital_certificates"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
	"strconv"
	"time"
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
	SchemaVersion      types.Int64                        `tfsdk:"schema_version"`
	CertificateRequest *digitalCertificateResourceRequest `tfsdk:"certificate_request"`
	CertificateResult  *digitalCertificateResourceResults `tfsdk:"certificate_result"`
	ID                 types.String                       `tfsdk:"id"`
	LastUpdated        types.String                       `tfsdk:"last_updated"`
}

type digitalCertificateResourceRequest struct {
	Name        types.String `tfsdk:"name"`
	Certificate types.String `tfsdk:"certificate"`
	PrivateKey  types.String `tfsdk:"private_key"`
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
	AzionInformation   types.String   `tfsdk:"azion_information"`
}

func (r *digitalCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digital_certificate"
}

func (r *digitalCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"certificate_request": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "Name of the certificate.",
						Required:    true,
					},
					"certificate": schema.StringAttribute{
						Description: "The content of the certificate.",
						Required:    true,
					},
					"private_key": schema.StringAttribute{
						Description: "Private key of the digital certificate.",
						Required:    true,
					},
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
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"certificate_id": schema.Int64Attribute{
						Description: "The function identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the certificate.",
						Computed:    true,
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
						Computed:    true,
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

	certificateRequest := digital_certificates.CreateCertificateRequest{
		Name:        plan.CertificateRequest.Name.ValueString(),
		Certificate: plan.CertificateRequest.Certificate.ValueString(),
		PrivateKey:  plan.CertificateRequest.PrivateKey.ValueString(),
	}

	certificateResponse, response, err := r.client.digitalCertificatesApi.CreateDigitalCertificateApi.CreateCertificate(ctx).CreateCertificateRequest(certificateRequest).Execute()
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
		CertificateContent: types.StringValue(certificateResponse.Results.GetCertificateContent()),
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

	//var edgeFunctionId int64
	//var err error
	//if state.EdgeFunction != nil {
	//	edgeFunctionId = state.EdgeFunction.FunctionID.ValueInt64()
	//} else {
	//	edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
	//	if err != nil {
	//		resp.Diagnostics.AddError(
	//			"Value Conversion error ",
	//			"Could not convert Edge Function ID",
	//		)
	//		return
	//	}
	//}

	//getEdgeFunction, response, err := r.client.edgefunctionsApi.EdgeFunctionsApi.EdgeFunctionsIdGet(ctx, edgeFunctionId).Execute()
	//if err != nil {
	//	bodyBytes, erro := io.ReadAll(response.Body)
	//	if erro != nil {
	//		resp.Diagnostics.AddError(
	//			err.Error(),
	//			"err",
	//		)
	//	}
	//	bodyString := string(bodyBytes)
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		bodyString,
	//	)
	//	return
	//}
	//
	//jsonArgsStr, err := utils.ConvertInterfaceToString(getEdgeFunction.Results.JsonArgs)
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		"err",
	//	)
	//}
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//EdgeFunctionState := EdgeFunctionDataSourceModel{
	//	SchemaVersion: types.Int64Value(int64(*getEdgeFunction.SchemaVersion)),
	//	Results: EdgeFunctionResults{
	//		FunctionID:    types.Int64Value(*getEdgeFunction.Results.Id),
	//		Name:          types.StringValue(*getEdgeFunction.Results.Name),
	//		Language:      types.StringValue(*getEdgeFunction.Results.Language),
	//		Code:          types.StringValue(*getEdgeFunction.Results.Code),
	//		JSONArgs:      types.StringValue(jsonArgsStr),
	//		InitiatorType: types.StringValue(*getEdgeFunction.Results.InitiatorType),
	//		IsActive:      types.BoolValue(*getEdgeFunction.Results.Active),
	//		LastEditor:    types.StringValue(*getEdgeFunction.Results.LastEditor),
	//		Modified:      types.StringValue(*getEdgeFunction.Results.Modified),
	//	},
	//}
	//if getEdgeFunction.Results.ReferenceCount != nil {
	//	EdgeFunctionState.Results.ReferenceCount = types.Int64Value(*getEdgeFunction.Results.ReferenceCount)
	//}
	//if getEdgeFunction.Results.FunctionToRun != nil {
	//	EdgeFunctionState.Results.FunctionToRun = types.StringValue(*getEdgeFunction.Results.FunctionToRun)
	//}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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

	//requestJsonArgs, err := utils.ConvertStringToInterface(plan.EdgeFunction.JSONArgs.ValueString())
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		"err",
	//	)
	//}
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//edgeFunctionId := state.EdgeFunction.FunctionID.ValueInt64()
	//updateEdgeFunctionRequest := edgefunctions.PutEdgeFunctionRequest{
	//	Name:     edgefunctions.PtrString(plan.EdgeFunction.Name.ValueString()),
	//	Code:     edgefunctions.PtrString(plan.EdgeFunction.Code.ValueString()),
	//	Active:   edgefunctions.PtrBool(plan.EdgeFunction.IsActive.ValueBool()),
	//	JsonArgs: requestJsonArgs,
	//}
	//
	//updateEdgeFunction, response, err := r.client.edgefunctionsApi.EdgeFunctionsApi.EdgeFunctionsIdPut(ctx, edgeFunctionId).PutEdgeFunctionRequest(updateEdgeFunctionRequest).Execute()
	//if err != nil {
	//	bodyBytes, erro := io.ReadAll(response.Body)
	//	if erro != nil {
	//		resp.Diagnostics.AddError(
	//			err.Error(),
	//			"err",
	//		)
	//	}
	//	bodyString := string(bodyBytes)
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		bodyString,
	//	)
	//	return
	//}
	//
	//jsonArgsStr, err := utils.ConvertInterfaceToString(updateEdgeFunction.Results.JsonArgs)
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		"err",
	//	)
	//}
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//plan.EdgeFunction = &digitalCertificateResourceResults{
	//	FunctionID:    types.Int64Value(*updateEdgeFunction.Results.Id),
	//	Name:          types.StringValue(*updateEdgeFunction.Results.Name),
	//	Language:      types.StringValue(*updateEdgeFunction.Results.Language),
	//	Code:          types.StringValue(*updateEdgeFunction.Results.Code),
	//	JSONArgs:      types.StringValue(jsonArgsStr),
	//	InitiatorType: types.StringValue(*updateEdgeFunction.Results.InitiatorType),
	//	IsActive:      types.BoolValue(*updateEdgeFunction.Results.Active),
	//	LastEditor:    types.StringValue(*updateEdgeFunction.Results.LastEditor),
	//	Modified:      types.StringValue(*updateEdgeFunction.Results.Modified),
	//}
	//if updateEdgeFunction.Results.ReferenceCount != nil {
	//	plan.EdgeFunction.ReferenceCount = types.Int64Value(*updateEdgeFunction.Results.ReferenceCount)
	//}
	//if updateEdgeFunction.Results.FunctionToRun != nil {
	//	plan.EdgeFunction.FunctionToRun = types.StringValue(*updateEdgeFunction.Results.FunctionToRun)
	//}
	//plan.SchemaVersion = types.Int64Value(int64(*updateEdgeFunction.SchemaVersion))
	//plan.ID = types.StringValue(strconv.FormatInt(*updateEdgeFunction.Results.Id, 10))
	//plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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

	//var edgeFunctionId int64
	//var err error
	//if state.EdgeFunction != nil {
	//	edgeFunctionId = state.EdgeFunction.FunctionID.ValueInt64()
	//} else {
	//	edgeFunctionId, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
	//	if err != nil {
	//		resp.Diagnostics.AddError(
	//			"Value Conversion error ",
	//			"Could not convert Edge Function ID",
	//		)
	//		return
	//	}
	//}
	//response, err := r.client.edgefunctionsApi.EdgeFunctionsApi.EdgeFunctionsIdDelete(ctx, edgeFunctionId).Execute()
	//if err != nil {
	//	bodyBytes, erro := io.ReadAll(response.Body)
	//	if erro != nil {
	//		resp.Diagnostics.AddError(
	//			err.Error(),
	//			"err",
	//		)
	//	}
	//	bodyString := string(bodyBytes)
	//	resp.Diagnostics.AddError(
	//		err.Error(),
	//		bodyString,
	//	)
	//	return
	//}
}

func (r *digitalCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
