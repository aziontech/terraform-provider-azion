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
	_ datasource.DataSource              = &WorkloadsDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkloadsDataSource{}
)

func dataSourceAzionWorkloads() datasource.DataSource {
	return &WorkloadsDataSource{}
}

type WorkloadsDataSource struct {
	client *apiClient
}

type WorkloadsDataSourceModel struct {
	Counter types.Int64        `tfsdk:"counter"`
	ID      types.String       `tfsdk:"id"`
	Results []WorkloadsResults `tfsdk:"results"`
}

type WorkloadsResults struct {
	ID               types.Int64     `tfsdk:"id"`
	Name             types.String    `tfsdk:"name"`
	AlternateDomains types.List      `tfsdk:"alternate_domains"`
	Active           types.Bool      `tfsdk:"active"`
	NetworkMap       types.String    `tfsdk:"network_map"`
	LastEditor       types.String    `tfsdk:"last_editor"`
	LastModified     types.String    `tfsdk:"last_modified"`
	TLS              TLSConfig       `tfsdk:"tls"`
	Protocols        ProtocolsConfig `tfsdk:"protocols"`
	MTLS             MTLSConfig      `tfsdk:"mtls"`
	Domains          []DomainEntry   `tfsdk:"domains"`
	ProductVersion   types.String    `tfsdk:"product_version"`
}

type TLSConfig struct {
	Certificate    types.Int64  `tfsdk:"certificate"`
	Ciphers        types.String `tfsdk:"ciphers"`
	MinimumVersion types.String `tfsdk:"minimum_version"`
}

type ProtocolsConfig struct {
	HTTP HTTPProtocols `tfsdk:"http"`
}

type HTTPProtocols struct {
	Versions   types.List `tfsdk:"versions"`
	HTTPPorts  types.List `tfsdk:"http_ports"`
	HTTPSPorts types.List `tfsdk:"https_ports"`
	QuicPorts  types.List `tfsdk:"quic_ports"`
}

type MTLSConfig struct {
	Verification types.String `tfsdk:"verification"`
	Certificate  types.Int64  `tfsdk:"certificate"`
	CRL          types.List   `tfsdk:"crl"`
}

type DomainEntry struct {
	Domain      types.String `tfsdk:"domain"`
	AllowAccess types.Bool   `tfsdk:"allow_access"`
}

func (d *WorkloadsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *WorkloadsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workloads"
}

func (d *WorkloadsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"counter": schema.Int64Attribute{
				Computed: true,
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
						"alternate_domains": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "List of alternate domains.",
						},
						"active": schema.BoolAttribute{
							Computed:    true,
							Description: "Indicates if the entry is active.",
						},
						"network_map": schema.StringAttribute{
							Computed:    true,
							Description: "Network map reference.",
						},
						"last_editor": schema.StringAttribute{
							Computed:    true,
							Description: "Last editor of this entry.",
						},
						"last_modified": schema.StringAttribute{
							Computed:    true,
							Description: "Last modified timestamp.",
						},
						"tls": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"certificate": schema.Int64Attribute{
									Computed: true,
								},
								"ciphers": schema.StringAttribute{
									Computed: true,
								},
								"minimum_version": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						"protocols": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"http": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"versions": schema.ListAttribute{
											Computed:    true,
											ElementType: types.StringType,
										},
										"http_ports": schema.ListAttribute{
											Computed:    true,
											ElementType: types.Int64Type,
										},
										"https_ports": schema.ListAttribute{
											Computed:    true,
											ElementType: types.Int64Type,
										},
										"quic_ports": schema.ListAttribute{
											Computed:    true,
											ElementType: types.Int64Type,
										},
									},
								},
							},
						},
						"mtls": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"verification": schema.StringAttribute{
									Computed: true,
								},
								"certificate": schema.Int64Attribute{
									Computed: true,
								},
								"crl": schema.ListAttribute{
									Computed:    true,
									ElementType: types.StringType,
								},
							},
						},
						"domains": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"domain": schema.StringAttribute{
										Computed: true,
									},
									"allow_access": schema.BoolAttribute{
										Computed: true,
									},
								},
							},
						},
						"product_version": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *WorkloadsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var Page types.Int64
	var PageSize types.Int64

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &Page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &PageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	if Page.ValueInt64() == 0 {
		Page = types.Int64Value(1)
	}

	if PageSize.ValueInt64() == 0 {
		PageSize = types.Int64Value(10)
	}

	workloadsResponse, response, err := d.client.edgeApi.WorkloadsAPI.ListWorkloads(ctx).Page(int32(Page.ValueInt64())).PageSize(int32(PageSize.ValueInt64())).Execute() //nolint
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

	var count int64 = 0 // Default value in case Count is nil
	if workloadsResponse.Count != nil {
		count = int64(*workloadsResponse.Count) // Dereference and convert
	}

	counter := types.Int64Value(count)

	workloadState := WorkloadsDataSourceModel{
		Counter: types.Int64Value(counter.ValueInt64()),
	}

	for _, resultWorkload := range workloadsResponse.Results {

		var slice []types.String
		for _, domains := range resultWorkload.AlternateDomains {
			slice = append(slice, types.StringValue(domains))
		}

		var dr = WorkloadsResults{
			ID:               types.Int64Value(int64(resultWorkload.GetId())),
			Name:             types.StringValue(resultWorkload.GetName()),
			AlternateDomains: utils.SliceStringTypeToList(slice),
			Active:           types.BoolValue(resultWorkload.GetActive()),
			NetworkMap:       types.StringValue(resultWorkload.GetNetworkMap()),
			LastEditor:       types.StringValue(resultWorkload.GetLastEditor()),
			LastModified:     types.StringValue(resultWorkload.GetLastModified().Format("2006-01-02T15:04:05Z")),
			ProductVersion:   types.StringValue(resultWorkload.GetProductVersion()),
		}

		if resultWorkload.Tls != nil {
			dr.TLS = TLSConfig{
				Certificate:    types.Int64Value(int64(resultWorkload.Tls.GetCertificate())),
				Ciphers:        types.StringValue(resultWorkload.Tls.GetCiphers()),
				MinimumVersion: types.StringValue(resultWorkload.Tls.GetMinimumVersion()),
			}
		}

		if resultWorkload.Protocols != nil {
			dr.Protocols = ProtocolsConfig{
				HTTP: HTTPProtocols{
					Versions:   types.ListValue(types.StringType, resultWorkload.Protocols.HTTP.Versions),   // Changed to types.List
					HTTPPorts:  types.ListValue(types.StringType, resultWorkload.Protocols.HTTP.HTTPPorts),  // Changed to types.List
					HTTPSPorts: types.ListValue(types.StringType, resultWorkload.Protocols.HTTP.HTTPSPorts), // Changed to types.List
					QuicPorts:  types.ListValue(types.StringType, resultWorkload.Protocols.HTTP.QuicPorts),  // Changed to types.List
				},
			}
		}

		if resultWorkload.MTLS != nil {
			dr.MTLS = MTLSConfig{
				Verification: types.StringValue(resultWorkload.MTLS.GetVerification()),
				Certificate:  types.Int64Value(resultWorkload.MTLS.GetCertificate()),
				CRL:          types.ListValue(types.StringType, resultWorkload.MTLS.CRL), // Changed to types.List
			}
		}

		for _, domain := range resultWorkload.Domains {
			dr.Domains = append(dr.Domains, DomainEntry{
				Domain:      types.StringValue(domain.Domain),
				AllowAccess: types.BoolValue(domain.AllowAccess),
			})
		}

		workloadState.Results = append(workloadState.Results, dr)
	}

	workloadState.ID = types.StringValue("placeholder")
	diags := resp.State.Set(ctx, &workloadState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
