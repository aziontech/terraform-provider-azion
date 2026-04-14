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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &DigitalCertificatesDataSource{}
	_ datasource.DataSourceWithConfigure = &DigitalCertificatesDataSource{}
)

func dataSourceAzionDigitalCertificates() datasource.DataSource {
	return &DigitalCertificatesDataSource{}
}

type DigitalCertificatesDataSource struct {
	client *apiClient
}

type DigitalCertificatesDataSourceModel struct {
	ID            types.String              `tfsdk:"id"`
	Counter       types.Int64               `tfsdk:"counter"`
	TotalPages    types.Int64               `tfsdk:"total_pages"`
	Page          types.Int64               `tfsdk:"page"`
	PageSize      types.Int64               `tfsdk:"page_size"`
	Links         *CertificateLinksModel    `tfsdk:"links"`
	SchemaVersion types.Int64               `tfsdk:"schema_version"`
	Results       []CertificatesResultModel `tfsdk:"results"`
}

type CertificateLinksModel struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type CertificatesResultModel struct {
	ID             types.Int64    `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Issuer         types.String   `tfsdk:"issuer"`
	SubjectName    []types.String `tfsdk:"subject_name"`
	Validity       types.String   `tfsdk:"validity"`
	Status         types.String   `tfsdk:"status"`
	StatusDetail   types.String   `tfsdk:"status_detail"`
	Type           types.String   `tfsdk:"certificate_type"`
	Managed        types.Bool     `tfsdk:"managed"`
	Challenge      types.String   `tfsdk:"challenge"`
	Authority      types.String   `tfsdk:"authority"`
	KeyAlgorithm   types.String   `tfsdk:"key_algorithm"`
	Active         types.Bool     `tfsdk:"active"`
	ProductVersion types.String   `tfsdk:"product_version"`
	LastEditor     types.String   `tfsdk:"last_editor"`
	CreatedAt      types.String   `tfsdk:"created_at"`
	LastModified   types.String   `tfsdk:"last_modified"`
	RenewedAt      types.String   `tfsdk:"renewed_at"`
}

func (d *DigitalCertificatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DigitalCertificatesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digital_certificates"
}

func (d *DigitalCertificatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of certificates.",
				Computed:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The current page number.",
				Computed:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The number of items per page.",
				Computed:    true,
			},
			"links": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"previous": schema.StringAttribute{
						Computed: true,
					},
					"next": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "Identifier of the digital certificate.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the digital certificate.",
							Computed:    true,
						},
						"issuer": schema.StringAttribute{
							Description: "Issuer of the digital certificate.",
							Computed:    true,
						},
						"subject_name": schema.ListAttribute{
							Description: "Subject name of the digital certificate.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"validity": schema.StringAttribute{
							Description: "Validity of the digital certificate.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the digital certificate.",
							Computed:    true,
						},
						"status_detail": schema.StringAttribute{
							Description: "Status detail of the digital certificate.",
							Computed:    true,
						},
						"certificate_type": schema.StringAttribute{
							Description: "Type of the digital certificate.",
							Computed:    true,
						},
						"managed": schema.BoolAttribute{
							Description: "Indicates whether the digital certificate is managed.",
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
						"created_at": schema.StringAttribute{
							Description: "Creation timestamp of the certificate.",
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
					},
				},
			},
		},
	}
}

func (d *DigitalCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	certificatesResponse, response, err := d.client.api.DigitalCertificatesCertificatesAPI.ListCertificates(ctx).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			certificatesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCertificateList, *http.Response, error) {
				return d.client.api.DigitalCertificatesCertificatesAPI.ListCertificates(ctx).Execute()
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

	state := populateCertificatesListResults(certificatesResponse)
	state.ID = types.StringValue("Get All Digital Certificates")
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateCertificatesListResults transforms API response data to Terraform state model.
func populateCertificatesListResults(list *azionapi.PaginatedCertificateList) DigitalCertificatesDataSourceModel {
	var previous, next string
	if list.HasPrevious() {
		previous = list.GetPrevious()
	}
	if list.HasNext() {
		next = list.GetNext()
	}

	var results []CertificatesResultModel
	for _, cert := range list.GetResults() {
		var subjectNameList []types.String
		for _, subjectName := range cert.GetSubjectName() {
			subjectNameList = append(subjectNameList, types.StringValue(subjectName))
		}

		var renewedAt string
		if cert.RenewedAt.IsSet() && cert.RenewedAt.Get() != nil {
			renewedAt = (*cert.RenewedAt.Get()).Format(time.RFC3339)
		}

		var createdAt string
		if cert.CreatedAt.IsSet() && cert.CreatedAt.Get() != nil {
			createdAt = (*cert.CreatedAt.Get()).Format(time.RFC3339)
		}

		certInfo := CertificatesResultModel{
			ID:             types.Int64Value(cert.GetId()),
			Name:           types.StringValue(cert.GetName()),
			Issuer:         types.StringValue(cert.GetIssuer()),
			SubjectName:    subjectNameList,
			Validity:       types.StringValue(cert.GetValidity()),
			Status:         types.StringValue(cert.GetStatus()),
			StatusDetail:   types.StringValue(cert.GetStatusDetail()),
			Type:           types.StringValue(cert.GetType()),
			Managed:        types.BoolValue(cert.GetManaged()),
			Challenge:      types.StringValue(cert.GetChallenge()),
			Authority:      types.StringValue(cert.GetAuthority()),
			KeyAlgorithm:   types.StringValue(cert.GetKeyAlgorithm()),
			ProductVersion: types.StringValue(cert.GetProductVersion()),
			LastEditor:     types.StringValue(cert.GetLastEditor()),
			CreatedAt:      types.StringValue(createdAt),
			LastModified:   types.StringValue(cert.GetLastModified().Format(time.RFC3339)),
			RenewedAt:      types.StringValue(renewedAt),
		}

		// Handle optional active field
		if cert.Active != nil {
			certInfo.Active = types.BoolValue(*cert.Active)
		}

		results = append(results, certInfo)
	}

	state := DigitalCertificatesDataSourceModel{
		SchemaVersion: types.Int64Value(1),
		Counter:       types.Int64Value(list.GetCount()),
		TotalPages:    types.Int64Value(list.GetTotalPages()),
		Page:          types.Int64Value(list.GetPage()),
		PageSize:      types.Int64Value(list.GetPageSize()),
		Links: &CertificateLinksModel{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
		Results: results,
	}

	return state
}
