package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CertificateDataSource{}
	_ datasource.DataSourceWithConfigure = &CertificateDataSource{}
)

func dataSourceAzionDigitalCertificate() datasource.DataSource {
	return &CertificateDataSource{}
}

type CertificateDataSource struct {
	client *apiClient
}

type CertificateDataSourceModel struct {
	ID            types.String             `tfsdk:"id"`
	SchemaVersion types.Int64              `tfsdk:"schema_version"`
	Results       *CertificateResultsModel `tfsdk:"results"`
	CertificateID types.Int64              `tfsdk:"certificate_id"`
}

type CertificateResultsModel struct {
	ID                 types.Int64    `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	Issuer             types.String   `tfsdk:"issuer"`
	SubjectName        []types.String `tfsdk:"subject_name"`
	Validity           types.String   `tfsdk:"validity"`
	Status             types.String   `tfsdk:"status"`
	StatusDetail       types.String   `tfsdk:"status_detail"`
	Type               types.String   `tfsdk:"certificate_type"`
	Managed            types.Bool     `tfsdk:"managed"`
	CSR                types.String   `tfsdk:"csr"`
	Challenge          types.String   `tfsdk:"challenge"`
	Authority          types.String   `tfsdk:"authority"`
	KeyAlgorithm       types.String   `tfsdk:"key_algorithm"`
	Active             types.Bool     `tfsdk:"active"`
	ProductVersion     types.String   `tfsdk:"product_version"`
	LastEditor         types.String   `tfsdk:"last_editor"`
	LastModified       types.String   `tfsdk:"last_modified"`
	RenewedAt          types.String   `tfsdk:"renewed_at"`
	CertificateContent types.String   `tfsdk:"certificate_content"`
	PrivateKey         types.String   `tfsdk:"private_key"`
}

func (c *CertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c.client = req.ProviderData.(*apiClient)
}

func (c *CertificateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digital_certificate"
}

func (c *CertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"certificate_id": schema.Int64Attribute{
				Description: "Identifier of the certificate.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Identifier of the certificate.",
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
						Description: "The content of the certificate.",
						Computed:    true,
					},
					"private_key": schema.StringAttribute{
						Description: "The private key of the certificate.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (c *CertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getCertificateID types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("certificate_id"), &getCertificateID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateResponse, response, err := c.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, getCertificateID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			certificateResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateResponse, *http.Response, error) {
				return c.client.api.DigitalCertificatesCertificatesAPI.RetrieveCertificate(ctx, getCertificateID.ValueInt64()).Execute()
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

	// Populate the results from the API response
	certificateState := populateCertificateResults(ctx, certificateResponse.GetData(), getCertificateID)
	certificateState.ID = types.StringValue("Get By ID Digital Certificate")
	diags = resp.State.Set(ctx, &certificateState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateCertificateResults transforms API response data to Terraform state model.
func populateCertificateResults(ctx context.Context, cert azionapi.Certificate, certificateID types.Int64) CertificateDataSourceModel {
	var subjectNameList []types.String
	for _, subjectName := range cert.GetSubjectName() {
		subjectNameList = append(subjectNameList, types.StringValue(subjectName))
	}

	var renewedAt string
	if cert.RenewedAt.IsSet() && cert.RenewedAt.Get() != nil {
		renewedAt = (*cert.RenewedAt.Get()).Format(time.RFC3339)
	}

	result := CertificateDataSourceModel{
		CertificateID: certificateID,
		SchemaVersion: types.Int64Value(1),
		Results: &CertificateResultsModel{
			ID:             types.Int64Value(cert.GetId()),
			Name:           types.StringValue(cert.GetName()),
			Issuer:         types.StringValue(cert.GetIssuer()),
			SubjectName:    subjectNameList,
			Validity:       types.StringValue(cert.GetValidity()),
			Status:         types.StringValue(cert.GetStatus()),
			StatusDetail:   types.StringValue(cert.GetStatusDetail()),
			Type:           types.StringValue(cert.GetType()),
			Managed:        types.BoolValue(cert.GetManaged()),
			CSR:            types.StringValue(cert.GetCsr()),
			Challenge:      types.StringValue(cert.GetChallenge()),
			Authority:      types.StringValue(cert.GetAuthority()),
			KeyAlgorithm:   types.StringValue(cert.GetKeyAlgorithm()),
			ProductVersion: types.StringValue(cert.GetProductVersion()),
			LastEditor:     types.StringValue(cert.GetLastEditor()),
			LastModified:   types.StringValue(cert.GetLastModified().Format(time.RFC3339)),
			RenewedAt:      types.StringValue(renewedAt),
		},
	}

	// Handle optional fields
	if cert.Active != nil {
		result.Results.Active = types.BoolValue(*cert.Active)
	}

	if cert.HasCertificate() {
		result.Results.CertificateContent = types.StringValue(cert.GetCertificate())
	}

	if cert.HasPrivateKey() {
		result.Results.PrivateKey = types.StringValue(cert.GetPrivateKey())
	}

	return result
}
