package provider

import (
	"context"
	"io"

	"github.com/aziontech/azionapi-go-sdk/domains"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &DomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &DomainsDataSource{}
)

func dataSourceAzionDomains() datasource.DataSource {
	return &DomainsDataSource{}
}

type DomainsDataSource struct {
	client *apiClient
}

type DomainsDataSourceModel struct {
	SchemaVersion types.Int64              `tfsdk:"schema_version"`
	Counter       types.Int64              `tfsdk:"counter"`
	TotalPages    types.Int64              `tfsdk:"total_pages"`
	Links         *GetDomainsResponseLinks `tfsdk:"links"`
	Results       []DomainsResults         `tfsdk:"results"`
	ID            types.String             `tfsdk:"id"`
}

type GetDomainsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type DomainsResults struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Cnames               types.List   `tfsdk:"cnames"`
	CnameAccessOnly      types.Bool   `tfsdk:"cname_access_only"`
	IsActive             types.Bool   `tfsdk:"is_active"`
	EdgeApplicationId    types.Int64  `tfsdk:"edge_application_id"`
	DigitalCertificateId types.Int64  `tfsdk:"digital_certificate_id"`
	DomainName           types.String `tfsdk:"domain_name"`
	Environment          types.String `tfsdk:"environment"`
}

func (d *DomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DomainsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *DomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
		},
	}
}

func (d *DomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	domainsResponse, response, err := d.client.domainsApi.DomainsApi.GetDomains(ctx).Execute()
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

	domainState := DomainsDataSourceModel{
		SchemaVersion: types.Int64Value(domainsResponse.SchemaVersion),
		Counter:       types.Int64Value(domainsResponse.Count),
		TotalPages:    types.Int64Value(domainsResponse.TotalPages),
		Links: &GetDomainsResponseLinks{
			Previous: types.StringValue(domainsResponse.Links.GetPrevious()),
			Next:     types.StringValue(domainsResponse.Links.GetNext()),
		},
	}

	for _, resultDomain := range domainsResponse.Results {
		var slice []types.String
		for _, Cnames := range resultDomain.Cnames {
			slice = append(slice, types.StringValue(Cnames))
		}
		var dr = DomainsResults{
			ID:                types.Int64Value(resultDomain.Id),
			Name:              types.StringValue(resultDomain.Name),
			CnameAccessOnly:   types.BoolValue(resultDomain.CnameAccessOnly),
			IsActive:          types.BoolValue(resultDomain.IsActive),
			EdgeApplicationId: types.Int64Value(resultDomain.EdgeApplicationId),
			DomainName:        types.StringValue(resultDomain.DomainName),
			Cnames:            utils.SliceStringTypeToList(slice),
		}
		if resultDomain.Environment != nil {
			dr.Environment = types.StringValue(*resultDomain.Environment)
		}
		if resultDomain.DigitalCertificateId.Get() != nil {
			dr.DigitalCertificateId = types.Int64Value(*domains.NullableInt64.Get(resultDomain.DigitalCertificateId))
		}
		domainState.Results = append(domainState.Results, dr)
	}

	domainState.ID = types.StringValue("placeholder")
	diags := resp.State.Set(ctx, &domainState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
