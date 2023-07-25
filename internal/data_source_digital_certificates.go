package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &DigitalCertificatesDataSource{}
	_ datasource.DataSourceWithConfigure = &DigitalCertificatesDataSource{}
)

func dataSourceAzionDigitalCertificate() datasource.DataSource {
	return &DigitalCertificatesDataSource{}
}

type DigitalCertificatesDataSource struct {
	client *apiClient
}

type DigitalCertificatesDataSourceModel struct {
	ID            types.String                        `tfsdk:"id"`
	Counter       types.Int64                         `tfsdk:"counter"`
	TotalPages    types.Int64                         `tfsdk:"total_pages"`
	Links         *GetDigitalCertificateResponseLinks `tfsdk:"links"`
	SchemaVersion types.Int64                         `tfsdk:"schema_version"`
	Results       []CertificateInfo                   `tfsdk:"results"`
}

type GetDigitalCertificateResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type CertificateInfo struct {
	ID              types.Int64    `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	Issuer          types.String   `tfsdk:"issuer"`
	SubjectName     []types.String `tfsdk:"subject_name"`
	Validity        types.String   `tfsdk:"validity"`
	Status          types.String   `tfsdk:"status"`
	CertificateType types.String   `tfsdk:"certificate_type"`
	Managed         types.Bool     `tfsdk:"managed"`
}

func (d *DigitalCertificatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DigitalCertificatesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digital_certificate"
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
				Description: "The total number of edge function instances.",
				Computed:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
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
						"certificate_type": schema.StringAttribute{
							Description: "Type of the digital certificate.",
							Computed:    true,
						},
						"managed": schema.BoolAttribute{
							Description: "Indicates whether the digital certificate is managed.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DigitalCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	digitalCertificatesResponse, response, err := d.client.digitalCertificatesApi.RetrieveDigitalCertificateListApi.ListDigitalCertificates(ctx).Execute()
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

	var previous, next string
	if digitalCertificatesResponse.Links.Previous.Get() != nil {
		previous = *digitalCertificatesResponse.Links.Previous.Get()
	}
	if digitalCertificatesResponse.Links.Next.Get() != nil {
		next = *digitalCertificatesResponse.Links.Next.Get()
	}

	digitalCertificateState := DigitalCertificatesDataSourceModel{
		SchemaVersion: types.Int64Value(int64(digitalCertificatesResponse.GetSchemaVersion())),
		TotalPages:    types.Int64Value(int64(*digitalCertificatesResponse.TotalPages)),
		Counter:       types.Int64Value(int64(*digitalCertificatesResponse.Count)),
		Links: &GetDigitalCertificateResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	var results []CertificateInfo
	for _, cert := range digitalCertificatesResponse.Results {
		var GetSubjectName []types.String
		for _, subjectName := range cert.GetSubjectName() {
			GetSubjectName = append(GetSubjectName, types.StringValue(subjectName))
		}
		certificateInfo := CertificateInfo{
			ID:   types.Int64Value(int64(cert.GetId())),
			Name: types.StringValue(cert.GetName()),
			//Issuer:          types.StringValue(cert2.),
			SubjectName:     GetSubjectName,
			Validity:        types.StringValue(cert.Validity),
			Status:          types.StringValue(cert.Status),
			CertificateType: types.StringValue(cert.CertificateType),
			Managed:         types.BoolValue(cert.Managed),
		}

		results = append(results, certificateInfo)
	}

	digitalCertificatesState := DigitalCertificatesDataSourceModel{
		SchemaVersion: types.Int64Value(digitalCertificatesResponse.SchemaVersion),
		Results:       results,
	}

	diags := resp.State.Set(ctx, &digitalCertificatesState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
