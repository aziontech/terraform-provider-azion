package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-v4-go-sdk/edge"
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

type WorkloadsDataSource struct {
	client *apiClient
}

type WorkloadsDataSourceModel struct {
	Counter  types.Int64        `tfsdk:"counter"`
	Page     types.Int64        `tfsdk:"page"`
	PageSize types.Int64        `tfsdk:"page_size"`
	Results  []WorkloadsResults `tfsdk:"results"`
	ID       types.String       `tfsdk:"id"`
}

type WorkloadsResults struct {
	ID               types.Int64  `tfsdk:"id" json:"id"`
	Name             types.String `tfsdk:"name" json:"name"`
	AlternateDomains types.Set    `tfsdk:"alternate_domains" json:"alternate_domains"`
	IsActive         types.Bool   `tfsdk:"is_active" json:"active"`
	NetworkMap       types.String `tfsdk:"network_map" json:"network_map"`
	LastEditor       types.String `tfsdk:"last_editor" json:"last_editor"`
	LastModified     types.String `tfsdk:"last_modified" json:"last_modified"`
	TLS              *TLSConfig   `tfsdk:"tls" json:"tls"`
	Protocols        *Protocols   `tfsdk:"protocols" json:"protocols"`
	MTLS             *MTLSConfig  `tfsdk:"mtls" json:"mtls"`
	ProductVersion   types.String `tfsdk:"product_version" json:"product_version"`
}

func (d *WorkloadsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *WorkloadsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (d *WorkloadsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Workloads.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number of Workloads.",
				Optional:    true,
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
							Description: "ID of this workload.",
						},
						"edge_application": schema.Int64Attribute{
							Required:    true,
							Description: "Edge Application associated ID.",
						},
						"edge_firewall": schema.Int64Attribute{
							Optional:    true,
							Description: "Edge Firewall associated ID.",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of this workload.",
						},
						"alternate_domains": schema.SetAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "List of alternate domains.",
						},
						"active": schema.BoolAttribute{
							Optional:    true,
							Description: "Indicates if the workload is active.",
						},
						"network_map": schema.StringAttribute{
							Optional:    true,
							Description: "Network map identifier.",
						},
						"last_editor": schema.StringAttribute{
							Computed:    true,
							Description: "Last editor of this workload.",
						},
						"last_modified": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp of the last modification.",
						},
						"tls": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"certificate": schema.Int64Attribute{
									Optional:    true,
									Description: "TLS certificate.",
								},
								"ciphers": schema.StringAttribute{
									Optional:    true,
									Description: "Certificate ciphers.",
								},
								"minimum_version": schema.StringAttribute{
									Optional:    true,
									Description: "Minimum tls version.",
								},
							},
						},
						"protocols": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"http": schema.SingleNestedAttribute{
									Optional: true,
									Attributes: map[string]schema.Attribute{
										"versions": schema.SetAttribute{
											Optional:    true,
											ElementType: types.StringType,
										},
										"http_ports": schema.SetAttribute{
											Optional:    true,
											ElementType: types.Int64Type,
										},
										"https_ports": schema.SetAttribute{
											Optional:    true,
											ElementType: types.Int64Type,
										},
										"quic_ports": schema.SetAttribute{
											Optional:    true,
											ElementType: types.Int64Type,
										},
									},
								},
							},
						},
						"mtls": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"verification": schema.StringAttribute{
									Optional: true,
								},
								"certificate": schema.Int64Attribute{
									Optional: true,
								},
								"crl": schema.SetAttribute{
									Optional:    true,
									ElementType: types.Int64Type,
								},
							},
						},
						"product_version": schema.StringAttribute{
							Computed:    true,
							Description: "Product version information.",
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

	workloadsResponse, response, err := d.client.workloadsApi.WorkloadsAPI.ListWorkloads(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			workloadsResponse, response, err = utils.RetryOn429(func() (*edge.PaginatedResponseListWorkloadList, *http.Response, error) {
				return d.client.workloadsApi.WorkloadsAPI.ListWorkloads(ctx).Page(Page.ValueInt64()).PageSize(PageSize.ValueInt64()).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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

	workloadState := WorkloadsDataSourceModel{
		Counter:  types.Int64Value(utils.PtrToInt64(workloadsResponse.Count)),
		Page:     types.Int64Value(Page.ValueInt64()),
		PageSize: types.Int64Value(PageSize.ValueInt64()),
	}

	for _, resultWorkload := range workloadsResponse.Results {
		var slice []types.String = []types.String{}
		for _, altDomain := range resultWorkload.AlternateDomains {
			slice = append(slice, types.StringValue(altDomain))
		}
		dataObject := WorkloadsResults{
			ID:               types.Int64Value(resultWorkload.GetId()),
			Name:             types.StringValue(resultWorkload.GetName()),
			IsActive:         types.BoolValue(resultWorkload.GetActive()),
			AlternateDomains: utils.SliceStringTypeToSetOrNull(slice),
			NetworkMap:       types.StringValue(resultWorkload.GetNetworkMap()),
		}
		if resultWorkload.Tls != nil {
			if resultWorkload.Tls.GetCertificate() > 0 {
				dataObject.TLS.Certificate = types.Int64Value(resultWorkload.Tls.GetCertificate())
			}
			if resultWorkload.Tls.GetCiphers().String != nil {
				dataObject.TLS.Ciphers = types.StringValue(*resultWorkload.Tls.GetCiphers().String)
			}
			if resultWorkload.Tls.GetMinimumVersion().String != nil {
				dataObject.TLS.MinVersion = types.StringValue(*resultWorkload.Tls.GetMinimumVersion().String)
			}
		}
		if resultWorkload.Mtls != nil {
			var slice []types.Int64
			for _, crl := range resultWorkload.Mtls.Crl {
				slice = append(slice, types.Int64Value(crl))
			}
			dataObject.MTLS.CRL = utils.SliceIntTypeToSet(slice)
			dataObject.MTLS.Certificate = types.Int64Value(resultWorkload.Mtls.GetCertificate())
			dataObject.MTLS.Verification = types.StringValue(*resultWorkload.Mtls.Verification)

		}

		workloadState.Results = append(workloadState.Results, dataObject)
	}

	workloadState.ID = types.StringValue("placeholder")
	diags := resp.State.Set(ctx, &workloadState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
