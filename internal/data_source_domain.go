package provider

import (
	"context"

	"github.com/aziontech/azionapi-go-sdk/domains"
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

type GetDomainResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type DomainResults struct {
	ID                   types.Int64    `tfsdk:"id"`
	Name                 types.String   `tfsdk:"name"`
	Cnames               []types.String `tfsdk:"cnames"`
	CnameAccessOnly      types.Bool     `tfsdk:"cname_access_only"`
	IsActive             types.Bool     `tfsdk:"is_active"`
	EdgeApplicationId    types.Int64    `tfsdk:"edge_application_id"`
	DigitalCertificateId types.Int64    `tfsdk:"digital_certificate_id"`
	DomainName           types.String   `tfsdk:"domain_name"`
	Environment          types.String   `tfsdk:"environment"`
}

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DomainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"counter": schema.Int64Attribute{
				Computed: true,
			},
			"total_pages": schema.Int64Attribute{
				Computed: true,
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
							Computed:    true,
							Description: "Identification of this entry.",
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
							Description: "Make access to your URL only via provided CNAMEs.",
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

	domainResponse, _, err := d.client.domainsApi.DomainsApi.GetDomain(ctx, getDomainId.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Token has expired",
			err.Error(),
		)
		return
	}

	domainState := DomainDataSourceModel{
		SchemaVersion: types.Int64Value(int64(domainResponse.SchemaVersion)),
		Results: DomainResults{
			ID:                   types.Int64Value(int64(domainResponse.Results.Id)),
			Name:                 types.StringValue(domainResponse.Results.Name),
			CnameAccessOnly:      types.BoolValue(*domainResponse.Results.CnameAccessOnly),
			IsActive:             types.BoolValue(*domainResponse.Results.IsActive),
			EdgeApplicationId:    types.Int64Value(int64(*domainResponse.Results.EdgeApplicationId)),
			DigitalCertificateId: types.Int64Value(int64(*domains.NullableInt64.Get(domainResponse.Results.DigitalCertificateId))),
			DomainName:           types.StringValue(*domainResponse.Results.DomainName),
			Environment:          types.StringValue(*domainResponse.Results.Environment),
		},
	}

	domainState.ID = types.StringValue("placeholder")
	diags = resp.State.Set(ctx, &domainState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
