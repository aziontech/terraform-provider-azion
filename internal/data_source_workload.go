package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &WorkloadDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkloadDataSource{}
)

func dataSourceAzionWorkload() datasource.DataSource {
	return &WorkloadDataSource{}
}

type WorkloadDataSource struct {
	client *apiClient
}

type WorkloadDataSourceModel struct {
	Data WorkloadResults `tfsdk:"data"`
	ID   types.String    `tfsdk:"id"`
}

type WorkloadResults struct {
	ID                        types.Int64       `tfsdk:"id"`
	Name                      types.String      `tfsdk:"name"`
	Active                    types.Bool        `tfsdk:"active"`
	LastEditor                types.String      `tfsdk:"last_editor"`
	LastModified              types.String      `tfsdk:"last_modified"`
	CreatedAt                 types.String      `tfsdk:"created_at"`
	Infrastructure            types.Int64       `tfsdk:"infrastructure"`
	Tls                       *TLSWorkloadModel `tfsdk:"tls"`
	Protocols                 *ProtocolsModel   `tfsdk:"protocols"`
	Mtls                      *MTLSModel        `tfsdk:"mtls"`
	Domains                   types.List        `tfsdk:"domains"`
	WorkloadDomainAllowAccess types.Bool        `tfsdk:"workload_domain_allow_access"`
	WorkloadDomain            types.String      `tfsdk:"workload_domain"`
	ProductVersion            types.String      `tfsdk:"product_version"`
}

type TLSWorkloadModel struct {
	Certificate    types.Int64  `tfsdk:"certificate"`
	Ciphers        types.Int64  `tfsdk:"ciphers"`
	MinimumVersion types.String `tfsdk:"minimum_version"`
}

type ProtocolsModel struct {
	Http *HttpProtocolModel `tfsdk:"http"`
}

type HttpProtocolModel struct {
	Versions   types.List `tfsdk:"versions"`
	HttpPorts  types.List `tfsdk:"http_ports"`
	HttpsPorts types.List `tfsdk:"https_ports"`
	QuicPorts  types.List `tfsdk:"quic_ports"`
}

type MTLSModel struct {
	Enabled types.Bool       `tfsdk:"enabled"`
	Config  *MTLSConfigModel `tfsdk:"config"`
}

type MTLSConfigModel struct {
	Certificate  types.Int64  `tfsdk:"certificate"`
	Crl          types.List   `tfsdk:"crl"`
	Verification types.String `tfsdk:"verification"`
}

func (d *WorkloadDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *WorkloadDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (d *WorkloadDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
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
					"created_at": schema.StringAttribute{
						Description: "Creation timestamp of the workload.",
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
	}
}

func (d *WorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getWorkloadId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getWorkloadId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadID, err := strconv.ParseInt(getWorkloadId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	workloadResponse, response, err := d.client.api.WorkloadsAPI.
		RetrieveWorkload(ctx, workloadID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			workloadResponse, response, err = utils.RetryOn429(func() (*azionapi.WorkloadResponse, *http.Response, error) {
				return d.client.api.WorkloadsAPI.RetrieveWorkload(ctx, workloadID).Execute() //nolint
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
			usrMsg, errMsg := errPrintWorkload(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	workloadState := WorkloadDataSourceModel{
		Data: WorkloadResults{
			ID:             types.Int64Value(workloadResponse.Data.Id),
			Name:           types.StringValue(workloadResponse.Data.Name),
			LastEditor:     types.StringValue(workloadResponse.Data.LastEditor),
			LastModified:   types.StringValue(workloadResponse.Data.LastModified.Format(time.RFC850)),
			CreatedAt:      types.StringValue(workloadResponse.Data.CreatedAt.Format(time.RFC3339)),
			ProductVersion: types.StringValue(workloadResponse.Data.ProductVersion),
			WorkloadDomain: types.StringValue(workloadResponse.Data.WorkloadDomain),
		},
	}

	// Set optional fields
	if workloadResponse.Data.Active != nil {
		workloadState.Data.Active = types.BoolValue(*workloadResponse.Data.Active)
	}

	if workloadResponse.Data.Infrastructure != nil {
		workloadState.Data.Infrastructure = types.Int64Value(*workloadResponse.Data.Infrastructure)
	}

	if workloadResponse.Data.WorkloadDomainAllowAccess != nil {
		workloadState.Data.WorkloadDomainAllowAccess = types.BoolValue(*workloadResponse.Data.WorkloadDomainAllowAccess)
	}

	// Handle TLS configuration
	if workloadResponse.Data.Tls != nil {
		tlsModel := &TLSWorkloadModel{}
		if workloadResponse.Data.Tls.Certificate.IsSet() {
			cert := workloadResponse.Data.Tls.Certificate.Get()
			if cert != nil {
				tlsModel.Certificate = types.Int64Value(*cert)
			}
		}
		if workloadResponse.Data.Tls.Ciphers != nil {
			tlsModel.Ciphers = types.Int64Value(*workloadResponse.Data.Tls.Ciphers)
		}
		if workloadResponse.Data.Tls.MinimumVersion.IsSet() {
			minVer := workloadResponse.Data.Tls.MinimumVersion.Get()
			if minVer != nil {
				tlsModel.MinimumVersion = types.StringValue(string(*minVer))
			}
		}
		workloadState.Data.Tls = tlsModel
	}

	// Handle Protocols configuration
	if workloadResponse.Data.Protocols != nil && workloadResponse.Data.Protocols.Http != nil {
		httpProto := workloadResponse.Data.Protocols.Http
		httpModel := &HttpProtocolModel{}

		if httpProto.Versions != nil {
			versionsList, diags := types.ListValueFrom(ctx, types.StringType, httpProto.Versions)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			httpModel.Versions = versionsList
		} else {
			httpModel.Versions = types.ListNull(types.StringType)
		}

		if httpProto.HttpPorts != nil {
			httpPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.HttpPorts)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			httpModel.HttpPorts = httpPortsList
		} else {
			httpModel.HttpPorts = types.ListNull(types.Int64Type)
		}

		if httpProto.HttpsPorts != nil {
			httpsPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.HttpsPorts)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			httpModel.HttpsPorts = httpsPortsList
		} else {
			httpModel.HttpsPorts = types.ListNull(types.Int64Type)
		}

		if httpProto.QuicPorts != nil {
			quicPortsList, diags := types.ListValueFrom(ctx, types.Int64Type, httpProto.QuicPorts)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			httpModel.QuicPorts = quicPortsList
		} else {
			httpModel.QuicPorts = types.ListNull(types.Int64Type)
		}

		workloadState.Data.Protocols = &ProtocolsModel{Http: httpModel}
	}

	// Handle MTLS configuration
	if workloadResponse.Data.Mtls != nil {
		mtlsModel := &MTLSModel{}
		if workloadResponse.Data.Mtls.Enabled.IsSet() {
			enabled := workloadResponse.Data.Mtls.Enabled.Get()
			if enabled != nil {
				mtlsModel.Enabled = types.BoolValue(*enabled)
			}
		}

		if workloadResponse.Data.Mtls.Config.IsSet() {
			config := workloadResponse.Data.Mtls.Config.Get()
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
				} else {
					configModel.Crl = types.ListNull(types.Int64Type)
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
		workloadState.Data.Mtls = mtlsModel
	}

	// Handle Domains
	if workloadResponse.Data.Domains != nil {
		domainsList, diags := types.ListValueFrom(ctx, types.StringType, workloadResponse.Data.Domains)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		workloadState.Data.Domains = domainsList
	} else {
		workloadState.Data.Domains = types.ListNull(types.StringType)
	}

	workloadState.ID = types.StringValue("Get By Id Workload")
	diags = resp.State.Set(ctx, &workloadState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintWorkload(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Workload found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
