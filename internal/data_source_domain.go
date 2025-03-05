package provider

import (
	"context"
	"io"

	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &DomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainsDataSource{}
)

func dataSourceAzionDomain() datasource.DataSource {
	return &DomainDataSource{}
}

type DomainDataSource struct {
	client *apiClient
}

type DomainDataSourceModel struct {
	SchemaVersion types.Int64   `tfsdk:"schema_version"`
	Results       DomainResults `tfsdk:"results"`
	ID            types.String  `tfsdk:"id"`
}

type DomainResults struct {
	DomainId             types.Int64  `tfsdk:"domain_id"`
	Name                 types.String `tfsdk:"name"`
	Cnames               types.List   `tfsdk:"cnames"`
	CnameAccessOnly      types.Bool   `tfsdk:"cname_access_only"`
	IsActive             types.Bool   `tfsdk:"is_active"`
	EdgeApplicationId    types.Int64  `tfsdk:"edge_application_id"`
	DigitalCertificateId types.Int64  `tfsdk:"digital_certificate_id"`
	DomainName           types.String `tfsdk:"domain_name"`
	Environment          types.String `tfsdk:"environment"`
}

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DomainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"domain_id": schema.Int64Attribute{
						Description: "The domain identifier to target for the resource.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Name of this entry.",
					},
					"cnames": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "List of domains to use as URLs for your files.",
					},
					"cname_access_only": schema.BoolAttribute{
						Computed:    true,
						Description: "Allow access to your URL only via provided CNAMEs.",
					},
					"is_active": schema.BoolAttribute{
						Computed:    true,
						Description: "Status of the domain.",
					},
					"edge_application_id": schema.Int64Attribute{
						Computed:    true,
						Description: "Edge Application associated ID.",
					},
					"digital_certificate_id": schema.Int64Attribute{
						Computed:    true,
						Description: "Digital Certificate associated ID.",
					},
					"domain_name": schema.StringAttribute{
						Computed:    true,
						Description: "Domain name attributed by Azion to this configuration.",
					},
					"environment": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
	}
}

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getDomainId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getDomainId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainResponse, response, err := d.client.domainsApi.DomainsAPI.GetDomain(ctx, getDomainId.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			resp.Diagnostics.AddWarning(
				"Too many requests",
				"Terraform provider will wait some time before attempting this request again. Please wait.",
			)
			err := utils.SleepAfter429(response)
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
				)
				return
			}
			domainResponse, _, err = d.client.domainsApi.DomainsAPI.GetDomain(ctx, getDomainId.ValueString()).Execute() //nolint
			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"err",
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

	var slice []types.String
	for _, Cnames := range domainResponse.Results.Cnames {
		slice = append(slice, types.StringValue(Cnames))
	}

	domainState := DomainDataSourceModel{
		SchemaVersion: types.Int64Value(domainResponse.SchemaVersion),
		Results: DomainResults{
			DomainId:          types.Int64Value(domainResponse.Results.GetId()),
			Name:              types.StringValue(domainResponse.Results.GetName()),
			CnameAccessOnly:   types.BoolValue(domainResponse.Results.GetCnameAccessOnly()),
			IsActive:          types.BoolValue(domainResponse.Results.GetIsActive()),
			EdgeApplicationId: types.Int64Value(domainResponse.Results.GetEdgeApplicationId()),
			DomainName:        types.StringValue(domainResponse.Results.GetDomainName()),
			Cnames:            utils.SliceStringTypeToList(slice),
		},
	}
	if domainResponse.Results.Environment != nil {
		domainState.Results.Environment = types.StringValue(*domainResponse.Results.Environment)
	}
	if domainResponse.Results.DigitalCertificateId != nil {
		domainState.Results.DigitalCertificateId = types.Int64Value(domainResponse.Results.GetDigitalCertificateId())
	}

	domainState.ID = types.StringValue("Get By ID Domain")
	diags = resp.State.Set(ctx, &domainState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
