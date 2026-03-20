package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	Results []WorkloadsResults `tfsdk:"results"`
	ID      types.String       `tfsdk:"id"`
}

type WorkloadsResults struct {
	ID                        types.Int64        `tfsdk:"id"`
	Name                      types.String       `tfsdk:"name"`
	Active                    types.Bool         `tfsdk:"active"`
	LastEditor                types.String       `tfsdk:"last_editor"`
	LastModified              types.String       `tfsdk:"last_modified"`
	Infrastructure            types.Int64        `tfsdk:"infrastructure"`
	Tls                       *TLSWorkloadsModel `tfsdk:"tls"`
	Protocols                 *ProtocolsModel    `tfsdk:"protocols"`
	Mtls                      *MTLSModel         `tfsdk:"mtls"`
	Domains                   types.List         `tfsdk:"domains"`
	WorkloadDomainAllowAccess types.Bool         `tfsdk:"workload_domain_allow_access"`
	WorkloadDomain            types.String       `tfsdk:"workload_domain"`
	ProductVersion            types.String       `tfsdk:"product_version"`
}

type TLSWorkloadsModel struct {
	Certificate    types.Int64  `tfsdk:"certificate"`
	Ciphers        types.Int64  `tfsdk:"ciphers"`
	MinimumVersion types.String `tfsdk:"minimum_version"`
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
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of workloads.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The workload identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the workload.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Status of the workload.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "The last editor of the workload.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the workload.",
							Computed:    true,
						},
						"infrastructure": schema.Int64Attribute{
							Description: "Infrastructure type: 1 for Production (All Locations), 2 for Staging.",
							Computed:    true,
						},
						"tls": schema.SingleNestedAttribute{
							Description: "TLS configuration for the workload.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"certificate": schema.Int64Attribute{
									Description: "Certificate ID for TLS.",
									Computed:    true,
								},
								"ciphers": schema.Int64Attribute{
									Description: "Cipher suite configuration.",
									Computed:    true,
								},
								"minimum_version": schema.StringAttribute{
									Description: "Minimum TLS version.",
									Computed:    true,
								},
							},
						},
						"protocols": schema.SingleNestedAttribute{
							Description: "Protocol configurations for the workload.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"http": schema.SingleNestedAttribute{
									Description: "HTTP protocol configuration.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"versions": schema.ListAttribute{
											ElementType: types.StringType,
											Description: "HTTP versions supported.",
											Computed:    true,
										},
										"http_ports": schema.ListAttribute{
											ElementType: types.Int64Type,
											Description: "HTTP ports.",
											Computed:    true,
										},
										"https_ports": schema.ListAttribute{
											ElementType: types.Int64Type,
											Description: "HTTPS ports.",
											Computed:    true,
										},
										"quic_ports": schema.ListAttribute{
											ElementType: types.Int64Type,
											Description: "QUIC ports.",
											Computed:    true,
										},
									},
								},
							},
						},
						"mtls": schema.SingleNestedAttribute{
							Description: "Mutual TLS configuration for the workload.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									Description: "Whether MTLS is enabled.",
									Computed:    true,
								},
								"config": schema.SingleNestedAttribute{
									Description: "MTLS configuration.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"certificate": schema.Int64Attribute{
											Description: "MTLS certificate ID.",
											Computed:    true,
										},
										"crl": schema.ListAttribute{
											ElementType: types.Int64Type,
											Description: "Certificate Revocation List.",
											Computed:    true,
										},
										"verification": schema.StringAttribute{
											Description: "MTLS verification type: enforce or permissive.",
											Computed:    true,
										},
									},
								},
							},
						},
						"domains": schema.ListAttribute{
							ElementType: types.StringType,
							Description: "List of domains associated with the workload.",
							Computed:    true,
						},
						"workload_domain_allow_access": schema.BoolAttribute{
							Description: "Whether domain access is allowed.",
							Computed:    true,
						},
						"workload_domain": schema.StringAttribute{
							Description: "The workload domain.",
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "Product version of the workload.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *WorkloadsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	workloadsResponse, response, err := d.client.api.WorkloadsAPI.ListWorkloads(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			workloadsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedWorkloadList, *http.Response, error) {
				return d.client.api.WorkloadsAPI.ListWorkloads(ctx).Execute() //nolint
			}, 5) // Maximum 5 retries

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
			usrMsg, errMsg := errPrintWorkloads(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	workloadsState := WorkloadsDataSourceModel{}

	if workloadsResponse.Count != nil {
		workloadsState.Counter = types.Int64Value(*workloadsResponse.Count)
	}

	for _, resultWorkload := range workloadsResponse.GetResults() {
		result := WorkloadsResults{
			ID:             types.Int64Value(resultWorkload.Id),
			Name:           types.StringValue(resultWorkload.Name),
			LastEditor:     types.StringValue(resultWorkload.LastEditor),
			LastModified:   types.StringValue(resultWorkload.LastModified.Format(time.RFC850)),
			ProductVersion: types.StringValue(resultWorkload.ProductVersion),
			WorkloadDomain: types.StringValue(resultWorkload.WorkloadDomain),
		}

		// Set optional fields
		if resultWorkload.Active != nil {
			result.Active = types.BoolValue(*resultWorkload.Active)
		}

		if resultWorkload.Infrastructure != nil {
			result.Infrastructure = types.Int64Value(*resultWorkload.Infrastructure)
		}

		if resultWorkload.WorkloadDomainAllowAccess != nil {
			result.WorkloadDomainAllowAccess = types.BoolValue(*resultWorkload.WorkloadDomainAllowAccess)
		}

		// Handle TLS configuration
		if resultWorkload.Tls != nil {
			tlsModel := &TLSWorkloadsModel{}
			if resultWorkload.Tls.Certificate.IsSet() {
				cert := resultWorkload.Tls.Certificate.Get()
				if cert != nil {
					tlsModel.Certificate = types.Int64Value(*cert)
				}
			}
			if resultWorkload.Tls.Ciphers != nil {
				tlsModel.Ciphers = types.Int64Value(*resultWorkload.Tls.Ciphers)
			}
			if resultWorkload.Tls.MinimumVersion.IsSet() {
				minVer := resultWorkload.Tls.MinimumVersion.Get()
				if minVer != nil {
					tlsModel.MinimumVersion = types.StringValue(string(*minVer))
				}
			}
			result.Tls = tlsModel
		}

		// Handle Protocols configuration
		if resultWorkload.Protocols != nil && resultWorkload.Protocols.Http != nil {
			httpProto := resultWorkload.Protocols.Http
			httpModel := &HttpProtocolModel{}

			if httpProto.Versions != nil {
				versionsList, diags := types.ListValueFrom(ctx, types.StringType, httpProto.Versions)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				httpModel.Versions = versionsList
			}

			if httpProto.HttpPorts != nil {
				httpPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.HttpPorts)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				httpModel.HttpPorts = httpPortsList
			}

			if httpProto.HttpsPorts != nil {
				httpsPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.HttpsPorts)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				httpModel.HttpsPorts = httpsPortsList
			}

			if httpProto.QuicPorts != nil {
				quicPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.QuicPorts)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				httpModel.QuicPorts = quicPortsList
			}

			result.Protocols = &ProtocolsModel{Http: httpModel}
		}

		// Handle MTLS configuration
		if resultWorkload.Mtls != nil {
			mtlsModel := &MTLSModel{}
			if resultWorkload.Mtls.Enabled.IsSet() {
				enabled := resultWorkload.Mtls.Enabled.Get()
				if enabled != nil {
					mtlsModel.Enabled = types.BoolValue(*enabled)
				}
			}

			if resultWorkload.Mtls.Config.IsSet() {
				config := resultWorkload.Mtls.Config.Get()
				if config != nil {
					configModel := &MTLSConfigModel{}
					if config.Certificate.IsSet() {
						cert := config.Certificate.Get()
						if cert != nil {
							configModel.Certificate = types.Int64Value(*cert)
						}
					}
					if config.Crl != nil {
						crlList, diags := types.ListValueFrom(ctx, types.Int64Type, config.Crl)
						resp.Diagnostics.Append(diags...)
						if resp.Diagnostics.HasError() {
							return
						}
						configModel.Crl = crlList
					}
					if config.Verification.IsSet() {
						verif := config.Verification.Get()
						if verif != nil {
							configModel.Verification = types.StringValue(*verif)
						}
					}
					mtlsModel.Config = configModel
				}
			}
			result.Mtls = mtlsModel
		}

		// Handle Domains
		if resultWorkload.Domains != nil {
			domainsList, diags := types.ListValueFrom(ctx, types.StringType, resultWorkload.Domains)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			result.Domains = domainsList
		}

		workloadsState.Results = append(workloadsState.Results, result)
	}

	workloadsState.ID = types.StringValue("Get All Workloads")
	diags := resp.State.Set(ctx, &workloadsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintWorkloads(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Workloads found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
