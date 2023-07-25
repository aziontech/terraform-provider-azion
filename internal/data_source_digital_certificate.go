package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CertificateDataSource{}
	_ datasource.DataSourceWithConfigure = &CertificateDataSource{}
)

func dataSourceAzionCertificate() datasource.DataSource {
	return &CertificateDataSource{}
}

type CertificateDataSource struct {
	client *apiClient
}

type CertificateDataSourceModel struct {
	ID            types.String           `tfsdk:"id"`
	SchemaVersion types.Int64            `tfsdk:"schema_version"`
	Result        CertificateResultModel `tfsdk:"Result"`
	CertificateID types.Int64            `tfsdk:"certificateID"`
}

type CertificateResultModel struct {
	ID                 types.Int64    `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	Issuer             types.String   `tfsdk:"issuer"`
	SubjectName        []types.String `tfsdk:"subject_name"`
	Validity           types.String   `tfsdk:"validity"`
	Status             types.String   `tfsdk:"status"`
	CertificateType    types.String   `tfsdk:"certificate_type"`
	Managed            types.Bool     `tfsdk:"managed"`
	CSR                types.String   `tfsdk:"csr"`
	CertificateContent types.String   `tfsdk:"certificate_content"`
}

func (c *CertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c.client = req.ProviderData.(*apiClient)
}

func (c *CertificateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (c *CertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"certificateID": schema.Int64Attribute{
				Description: "Identifier of the certificate.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"Result": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Identifier of the certificate.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the certificate.",
						Required:    true,
					},
					"issuer": schema.StringAttribute{
						Description: "Issuer of the certificate.",
						Optional:    true,
					},
					"subject_name": schema.ListAttribute{
						Description: "Subject name of the certificate.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"validity": schema.StringAttribute{
						Description: "Validity of the certificate.",
						Optional:    true,
					},
					"status": schema.StringAttribute{
						Description: "Status of the certificate.",
						Optional:    true,
					},
					"certificate_type": schema.StringAttribute{
						Description: "Type of the certificate.",
						Optional:    true,
					},
					"managed": schema.BoolAttribute{
						Description: "Whether the certificate is managed.",
						Optional:    true,
					},
					"csr": schema.StringAttribute{
						Description: "Certificate Signing Request (CSR).",
						Optional:    true,
					},
					"certificate_content": schema.StringAttribute{
						Description: "The content of the certificate.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (c *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getCertificateID types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getCertificateID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateResponse, response, err := c.client.digitalCertificatesApi.RetrieveDigitalCertificateByIDApi.GetCertificate(ctx, getCertificateID.ValueInt64()).Execute()
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
	for _, subjectName := range certificateResponse.Result.GetSubjectName() {
		GetSubjectName = append(GetSubjectName, types.StringValue(subjectName))
	}

	certificateState := CertificateDataSourceModel{
		CertificateID: getCertificateID,
		SchemaVersion: types.Int64Value(int64(*certificateResponse.SchemaVersion)),
		Result: CertificateResultModel{
			ID:   types.Int64Value(int64(certificateResponse.Result.GetId())),
			Name: types.StringValue(certificateResponse.Result.GetName()),
			//Issuer:             types.StringValue(certificateResponse.Result.Issuer),
			SubjectName:     GetSubjectName,
			Validity:        types.StringValue(certificateResponse.Result.GetValidity()),
			Status:          types.StringValue(certificateResponse.Result.GetStatus()),
			CertificateType: types.StringValue(certificateResponse.Result.GetCertificateType()),
			Managed:         types.BoolValue(certificateResponse.Result.GetManaged()),
			//CSR:                types.StringValue(certificateResponse.Result.GetC),
			//CertificateContent: types.StringValue(certificateResponse.Result.CertificateContent),
		},
	}
	certificateState.ID = types.StringValue("Get All Digital Certificate")
	diags = resp.State.Set(ctx, &certificateState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
