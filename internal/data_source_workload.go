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

type WorkloadDataSource struct {
	client *apiClient
}

type WorkloadDataSourceModel struct {
	Data WorkloadResults `tfsdk:"data"`
	ID   types.String    `tfsdk:"id"`
}

type WorkloadResults struct {
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
				Description: "Identifier of the data source.",
				Optional:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
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
	}
}

func (d *WorkloadDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getWorkloadId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getWorkloadId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadResponse, response, err := d.client.workloadsApi.WorkloadsAPI.RetrieveWorkload(ctx, getWorkloadId.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			workloadResponse, response, err = utils.RetryOn429(func() (*edge.ResponseRetrieveWorkload, *http.Response, error) {
				return d.client.workloadsApi.WorkloadsAPI.RetrieveWorkload(ctx, getWorkloadId.ValueString()).Execute() //nolint
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

	var slice []types.String = []types.String{}
	for _, altDomain := range workloadResponse.Data.AlternateDomains {
		slice = append(slice, types.StringValue(altDomain))
	}

	workloadState := WorkloadDataSourceModel{
		Data: WorkloadResults{
			ID:               types.Int64Value(workloadResponse.Data.GetId()),
			Name:             types.StringValue(workloadResponse.Data.GetName()),
			IsActive:         types.BoolValue(workloadResponse.Data.GetActive()),
			AlternateDomains: utils.SliceStringTypeToSetOrNull(slice),
			NetworkMap:       types.StringValue(workloadResponse.Data.GetNetworkMap()),
		},
	}

	if workloadResponse.Data.Tls != nil {
		if workloadResponse.Data.Tls.GetCertificate() > 0 {
			workloadState.Data.TLS.Certificate = types.Int64Value(workloadResponse.Data.Tls.GetCertificate())
		}
		if workloadResponse.Data.Tls.GetCiphers().String != nil {
			workloadState.Data.TLS.Ciphers = types.StringValue(*workloadResponse.Data.Tls.GetCiphers().String)
		}
		if workloadResponse.Data.Tls.GetMinimumVersion().String != nil {
			workloadState.Data.TLS.MinVersion = types.StringValue(*workloadResponse.Data.Tls.GetMinimumVersion().String)
		}
	}
	if workloadResponse.Data.Mtls != nil {
		var slice []types.Int64
		for _, crl := range workloadResponse.Data.Mtls.Crl {
			slice = append(slice, types.Int64Value(crl))
		}
		workloadState.Data.MTLS.CRL = utils.SliceIntTypeToSet(slice)
		workloadState.Data.MTLS.Certificate = types.Int64Value(workloadResponse.Data.Mtls.GetCertificate())
		workloadState.Data.MTLS.Verification = types.StringValue(*workloadResponse.Data.Mtls.Verification)
	}

	workloadState.ID = types.StringValue("Get By ID Workload")
	diags = resp.State.Set(ctx, &workloadState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
