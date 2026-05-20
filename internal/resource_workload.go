package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &workloadResource{}
	_ resource.ResourceWithConfigure   = &workloadResource{}
	_ resource.ResourceWithImportState = &workloadResource{}
)

func NewWorkloadResource() resource.Resource {
	return &workloadResource{}
}

type workloadResource struct {
	client *apiClient
}

type workloadResourceModel struct {
	Workload    *workloadResourceResults `tfsdk:"workload"`
	ID          types.String             `tfsdk:"id"`
	LastUpdated types.String             `tfsdk:"last_updated"`
}

type workloadResourceResults struct {
	ID                        types.Int64               `tfsdk:"id"`
	Name                      types.String              `tfsdk:"name"`
	Active                    types.Bool                `tfsdk:"active"`
	LastEditor                types.String              `tfsdk:"last_editor"`
	LastModified              types.String              `tfsdk:"last_modified"`
	CreatedAt                 types.String              `tfsdk:"created_at"`
	Infrastructure            types.Int64               `tfsdk:"infrastructure"`
	Tls                       *TLSWorkloadResourceModel `tfsdk:"tls"`
	Protocols                 *ProtocolsResourceModel   `tfsdk:"protocols"`
	Mtls                      *MTLSResourceModel        `tfsdk:"mtls"`
	Domains                   types.Set                 `tfsdk:"domains"`
	WorkloadDomainAllowAccess types.Bool                `tfsdk:"workload_domain_allow_access"`
	WorkloadDomain            types.String              `tfsdk:"workload_domain"`
	ProductVersion            types.String              `tfsdk:"product_version"`
}

type TLSWorkloadResourceModel struct {
	Certificate    types.Int64  `tfsdk:"certificate"`
	Ciphers        types.Int64  `tfsdk:"ciphers"`
	MinimumVersion types.String `tfsdk:"minimum_version"`
}

type ProtocolsResourceModel struct {
	Http *HttpProtocolResourceModel `tfsdk:"http"`
}

type HttpProtocolResourceModel struct {
	Versions   types.List `tfsdk:"versions"`
	HttpPorts  types.List `tfsdk:"http_ports"`
	HttpsPorts types.List `tfsdk:"https_ports"`
	QuicPorts  types.List `tfsdk:"quic_ports"`
}

type MTLSResourceModel struct {
	Enabled types.Bool               `tfsdk:"enabled"`
	Config  *MTLSConfigResourceModel `tfsdk:"config"`
}

type MTLSConfigResourceModel struct {
	Certificate  types.Int64  `tfsdk:"certificate"`
	Crl          types.List   `tfsdk:"crl"`
	Verification types.String `tfsdk:"verification"`
}

func (r *workloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (r *workloadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing Azion Workloads.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"workload": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The workload identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the workload.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the workload.",
						Optional:    true,
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
						Optional:    true,
						Computed:    true,
					},
					"tls": schema.SingleNestedAttribute{
						Description: "TLS configuration for the workload.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"certificate": schema.Int64Attribute{
								Description: "Certificate ID for TLS.",
								Optional:    true,
							},
							"ciphers": schema.Int64Attribute{
								Description: "Cipher suite configuration.",
								Optional:    true,
							},
							"minimum_version": schema.StringAttribute{
								Description: "Minimum TLS version.",
								Optional:    true,
							},
						},
					},
					"protocols": schema.SingleNestedAttribute{
						Description: "Protocol configurations for the workload.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"http": schema.SingleNestedAttribute{
								Description: "HTTP protocol configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"versions": schema.ListAttribute{
										ElementType: types.StringType,
										Description: "HTTP versions supported.",
										Optional:    true,
									},
									"http_ports": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "HTTP ports.",
										Optional:    true,
									},
									"https_ports": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "HTTPS ports.",
										Optional:    true,
									},
									"quic_ports": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "QUIC ports.",
										Optional:    true,
									},
								},
							},
						},
					},
					"mtls": schema.SingleNestedAttribute{
						Description: "Mutual TLS configuration for the workload.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether MTLS is enabled.",
								Optional:    true,
							},
							"config": schema.SingleNestedAttribute{
								Description: "MTLS configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"certificate": schema.Int64Attribute{
										Description: "MTLS certificate ID.",
										Optional:    true,
									},
									"crl": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "Certificate Revocation List.",
										Optional:    true,
									},
									"verification": schema.StringAttribute{
										Description: "MTLS verification type: enforce or permissive.",
										Optional:    true,
									},
								},
							},
						},
					},
					"domains": schema.SetAttribute{
						ElementType: types.StringType,
						Description: "Set of domains associated with the workload.",
						Optional:    true,
						Computed:    true,
					},
					"workload_domain_allow_access": schema.BoolAttribute{
						Description: "Whether domain access is allowed.",
						Optional:    true,
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

func (r *workloadResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *workloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workloadResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workload := azionapi.NewWorkloadRequest(plan.Workload.Name.ValueString())

	// Set optional fields
	if !plan.Workload.Active.IsNull() && !plan.Workload.Active.IsUnknown() {
		workload.SetActive(plan.Workload.Active.ValueBool())
	}

	if !plan.Workload.Infrastructure.IsNull() && !plan.Workload.Infrastructure.IsUnknown() {
		workload.SetInfrastructure(plan.Workload.Infrastructure.ValueInt64())
	}

	if !plan.Workload.WorkloadDomainAllowAccess.IsNull() && !plan.Workload.WorkloadDomainAllowAccess.IsUnknown() {
		workload.SetWorkloadDomainAllowAccess(plan.Workload.WorkloadDomainAllowAccess.ValueBool())
	}

	// Handle TLS configuration
	if plan.Workload.Tls != nil {
		tls := azionapi.NewTLSWorkloadRequest()
		if !plan.Workload.Tls.Certificate.IsNull() && !plan.Workload.Tls.Certificate.IsUnknown() {
			tls.SetCertificate(plan.Workload.Tls.Certificate.ValueInt64())
		}
		if !plan.Workload.Tls.Ciphers.IsNull() && !plan.Workload.Tls.Ciphers.IsUnknown() {
			tls.SetCiphers(plan.Workload.Tls.Ciphers.ValueInt64())
		}
		if !plan.Workload.Tls.MinimumVersion.IsNull() && !plan.Workload.Tls.MinimumVersion.IsUnknown() {
			tls.SetMinimumVersion(plan.Workload.Tls.MinimumVersion.ValueString())
		}
		workload.SetTls(*tls)
	}

	// Handle Protocols configuration
	if plan.Workload.Protocols != nil && plan.Workload.Protocols.Http != nil {
		protocols := azionapi.NewProtocolsRequest()
		http := azionapi.NewHttpProtocolRequest()

		if !plan.Workload.Protocols.Http.Versions.IsNull() && !plan.Workload.Protocols.Http.Versions.IsUnknown() {
			var versions []string
			diags := plan.Workload.Protocols.Http.Versions.ElementsAs(ctx, &versions, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetVersions(versions)
		}

		if !plan.Workload.Protocols.Http.HttpPorts.IsNull() && !plan.Workload.Protocols.Http.HttpPorts.IsUnknown() {
			var httpPorts []int64
			diags := plan.Workload.Protocols.Http.HttpPorts.ElementsAs(ctx, &httpPorts, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetHttpPorts(httpPorts)
		}

		if !plan.Workload.Protocols.Http.HttpsPorts.IsNull() && !plan.Workload.Protocols.Http.HttpsPorts.IsUnknown() {
			var httpsPorts []int64
			diags := plan.Workload.Protocols.Http.HttpsPorts.ElementsAs(ctx, &httpsPorts, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetHttpsPorts(httpsPorts)
		}

		if !plan.Workload.Protocols.Http.QuicPorts.IsNull() && !plan.Workload.Protocols.Http.QuicPorts.IsUnknown() {
			var quicPorts []int64
			diags := plan.Workload.Protocols.Http.QuicPorts.ElementsAs(ctx, &quicPorts, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetQuicPorts(quicPorts)
		}

		protocols.SetHttp(*http)
		workload.SetProtocols(*protocols)
	}

	// Handle MTLS configuration
	if plan.Workload.Mtls != nil {
		mtls := azionapi.NewMTLSRequest()
		if !plan.Workload.Mtls.Enabled.IsNull() && !plan.Workload.Mtls.Enabled.IsUnknown() {
			mtls.SetEnabled(plan.Workload.Mtls.Enabled.ValueBool())
		}

		if plan.Workload.Mtls.Config != nil {
			config := azionapi.NewMTLSConfigRequest()
			if !plan.Workload.Mtls.Config.Certificate.IsNull() && !plan.Workload.Mtls.Config.Certificate.IsUnknown() {
				config.SetCertificate(plan.Workload.Mtls.Config.Certificate.ValueInt64())
			}
			if !plan.Workload.Mtls.Config.Crl.IsNull() && !plan.Workload.Mtls.Config.Crl.IsUnknown() {
				var crl []int64
				diags := plan.Workload.Mtls.Config.Crl.ElementsAs(ctx, &crl, false)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				config.SetCrl(crl)
			}
			if !plan.Workload.Mtls.Config.Verification.IsNull() && !plan.Workload.Mtls.Config.Verification.IsUnknown() {
				config.SetVerification(plan.Workload.Mtls.Config.Verification.ValueString())
			}
			mtls.SetConfig(*config)
		}
		workload.SetMtls(*mtls)
	}

	// Handle Domains
	if !plan.Workload.Domains.IsNull() && !plan.Workload.Domains.IsUnknown() {
		var domains []string
		diags := plan.Workload.Domains.ElementsAs(ctx, &domains, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		workload.SetDomains(domains)
	}

	createWorkload, response, err := r.client.api.WorkloadsAPI.CreateWorkload(ctx).WorkloadRequest(*workload).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			createWorkload, response, err = utils.RetryOn429(func() (*azionapi.WorkloadResponse, *http.Response, error) {
				return r.client.api.WorkloadsAPI.CreateWorkload(ctx).WorkloadRequest(*workload).Execute()
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
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Populate the state from the response, preserving plan values for optional nested fields.
	plan.Workload = populateWorkloadResults(ctx, createWorkload, plan.Workload)
	plan.ID = types.StringValue(strconv.FormatInt(createWorkload.Data.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workloadResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var workloadId int64
	var err error
	if state.Workload != nil {
		workloadId = state.Workload.ID.ValueInt64()
	} else {
		workloadId, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Workload ID",
			)
			return
		}
	}

	getWorkload, response, err := r.client.api.WorkloadsAPI.RetrieveWorkload(ctx, workloadId).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getWorkload, response, err = utils.RetryOn429(func() (*azionapi.WorkloadResponse, *http.Response, error) {
				return r.client.api.WorkloadsAPI.RetrieveWorkload(ctx, workloadId).Execute()
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
	}
	if response != nil {
		defer response.Body.Close()
	}

	state.Workload = populateWorkloadResults(ctx, getWorkload, state.Workload)
	state.ID = types.StringValue(strconv.FormatInt(getWorkload.Data.Id, 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workloadResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state workloadResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadId := state.Workload.ID.ValueInt64()
	updateWorkloadRequest := azionapi.NewPatchedWorkloadRequest()

	// Set optional fields
	if !plan.Workload.Name.IsNull() && !plan.Workload.Name.IsUnknown() {
		updateWorkloadRequest.SetName(plan.Workload.Name.ValueString())
	}

	if !plan.Workload.Active.IsNull() && !plan.Workload.Active.IsUnknown() {
		updateWorkloadRequest.SetActive(plan.Workload.Active.ValueBool())
	}

	if !plan.Workload.Infrastructure.IsNull() && !plan.Workload.Infrastructure.IsUnknown() {
		updateWorkloadRequest.SetInfrastructure(plan.Workload.Infrastructure.ValueInt64())
	}

	if !plan.Workload.WorkloadDomainAllowAccess.IsNull() && !plan.Workload.WorkloadDomainAllowAccess.IsUnknown() {
		updateWorkloadRequest.SetWorkloadDomainAllowAccess(plan.Workload.WorkloadDomainAllowAccess.ValueBool())
	}

	// Handle TLS configuration
	if plan.Workload.Tls != nil {
		tls := azionapi.NewTLSWorkloadRequest()
		if !plan.Workload.Tls.Certificate.IsNull() && !plan.Workload.Tls.Certificate.IsUnknown() {
			tls.SetCertificate(plan.Workload.Tls.Certificate.ValueInt64())
		}
		if !plan.Workload.Tls.Ciphers.IsNull() && !plan.Workload.Tls.Ciphers.IsUnknown() {
			tls.SetCiphers(plan.Workload.Tls.Ciphers.ValueInt64())
		}
		if !plan.Workload.Tls.MinimumVersion.IsNull() && !plan.Workload.Tls.MinimumVersion.IsUnknown() {
			tls.SetMinimumVersion(plan.Workload.Tls.MinimumVersion.ValueString())
		}
		updateWorkloadRequest.SetTls(*tls)
	}

	// Handle Protocols configuration
	if plan.Workload.Protocols != nil && plan.Workload.Protocols.Http != nil {
		protocols := azionapi.NewProtocolsRequest()
		http := azionapi.NewHttpProtocolRequest()

		if !plan.Workload.Protocols.Http.Versions.IsNull() && !plan.Workload.Protocols.Http.Versions.IsUnknown() {
			var versions []string
			diags := plan.Workload.Protocols.Http.Versions.ElementsAs(ctx, &versions, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetVersions(versions)
		}

		if !plan.Workload.Protocols.Http.HttpPorts.IsNull() && !plan.Workload.Protocols.Http.HttpPorts.IsUnknown() {
			var httpPorts []int64
			diags := plan.Workload.Protocols.Http.HttpPorts.ElementsAs(ctx, &httpPorts, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetHttpPorts(httpPorts)
		}

		if !plan.Workload.Protocols.Http.HttpsPorts.IsNull() && !plan.Workload.Protocols.Http.HttpsPorts.IsUnknown() {
			var httpsPorts []int64
			diags := plan.Workload.Protocols.Http.HttpsPorts.ElementsAs(ctx, &httpsPorts, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetHttpsPorts(httpsPorts)
		}

		if !plan.Workload.Protocols.Http.QuicPorts.IsNull() && !plan.Workload.Protocols.Http.QuicPorts.IsUnknown() {
			var quicPorts []int64
			diags := plan.Workload.Protocols.Http.QuicPorts.ElementsAs(ctx, &quicPorts, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			http.SetQuicPorts(quicPorts)
		}

		protocols.SetHttp(*http)
		updateWorkloadRequest.SetProtocols(*protocols)
	}

	// Handle MTLS configuration
	if plan.Workload.Mtls != nil {
		mtls := azionapi.NewMTLSRequest()
		if !plan.Workload.Mtls.Enabled.IsNull() && !plan.Workload.Mtls.Enabled.IsUnknown() {
			mtls.SetEnabled(plan.Workload.Mtls.Enabled.ValueBool())
		}

		if plan.Workload.Mtls.Config != nil {
			config := azionapi.NewMTLSConfigRequest()
			if !plan.Workload.Mtls.Config.Certificate.IsNull() && !plan.Workload.Mtls.Config.Certificate.IsUnknown() {
				config.SetCertificate(plan.Workload.Mtls.Config.Certificate.ValueInt64())
			}
			if !plan.Workload.Mtls.Config.Crl.IsNull() && !plan.Workload.Mtls.Config.Crl.IsUnknown() {
				var crl []int64
				diags := plan.Workload.Mtls.Config.Crl.ElementsAs(ctx, &crl, false)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				config.SetCrl(crl)
			}
			if !plan.Workload.Mtls.Config.Verification.IsNull() && !plan.Workload.Mtls.Config.Verification.IsUnknown() {
				config.SetVerification(plan.Workload.Mtls.Config.Verification.ValueString())
			}
			mtls.SetConfig(*config)
		}
		updateWorkloadRequest.SetMtls(*mtls)
	}

	// Handle Domains
	if !plan.Workload.Domains.IsNull() && !plan.Workload.Domains.IsUnknown() {
		var domains []string
		diags := plan.Workload.Domains.ElementsAs(ctx, &domains, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateWorkloadRequest.SetDomains(domains)
	}

	updateWorkload, response, err := r.client.api.WorkloadsAPI.PartialUpdateWorkload(ctx, workloadId).PatchedWorkloadRequest(*updateWorkloadRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			updateWorkload, response, err = utils.RetryOn429(func() (*azionapi.WorkloadResponse, *http.Response, error) {
				return r.client.api.WorkloadsAPI.PartialUpdateWorkload(ctx, workloadId).PatchedWorkloadRequest(*updateWorkloadRequest).Execute()
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
	}
	if response != nil {
		defer response.Body.Close()
	}

	plan.Workload = populateWorkloadResults(ctx, updateWorkload, plan.Workload)
	plan.ID = types.StringValue(strconv.FormatInt(updateWorkload.Data.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workloadResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadId := state.Workload.ID.ValueInt64()

	_, response, err := r.client.api.WorkloadsAPI.DeleteWorkload(ctx, workloadId).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.WorkloadsAPI.DeleteWorkload(ctx, workloadId).Execute()
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
	}
	if response != nil {
		defer response.Body.Close()
	}
}

func (r *workloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to populate workload results from API response.
// plan is used to preserve optional nested field values - if a nested field was null in the plan,
// it stays null in the result to avoid "Provider produced inconsistent result after apply" errors.
// When plan is nil (the post-import Read, where the prior state holds only the ID), every nested
// block the API returned is populated so the imported state mirrors the remote resource.
func populateWorkloadResults(ctx context.Context, response *azionapi.WorkloadResponse, plan *workloadResourceResults) *workloadResourceResults {
	if plan == nil {
		plan = &workloadResourceResults{
			Tls:       &TLSWorkloadResourceModel{},
			Protocols: &ProtocolsResourceModel{Http: &HttpProtocolResourceModel{}},
			Mtls:      &MTLSResourceModel{Config: &MTLSConfigResourceModel{}},
		}
	}

	result := &workloadResourceResults{
		ID:             types.Int64Value(response.Data.Id),
		Name:           types.StringValue(response.Data.Name),
		LastEditor:     types.StringValue(response.Data.LastEditor),
		LastModified:   types.StringValue(response.Data.LastModified.Format(time.RFC850)),
		CreatedAt:      types.StringValue(response.Data.CreatedAt.Format(time.RFC3339)),
		ProductVersion: types.StringValue(response.Data.ProductVersion),
		WorkloadDomain: types.StringValue(response.Data.WorkloadDomain),
	}

	if response.Data.Active != nil {
		result.Active = types.BoolValue(*response.Data.Active)
	}

	if response.Data.Infrastructure != nil {
		result.Infrastructure = types.Int64Value(*response.Data.Infrastructure)
	}

	if response.Data.WorkloadDomainAllowAccess != nil {
		result.WorkloadDomainAllowAccess = types.BoolValue(*response.Data.WorkloadDomainAllowAccess)
	}

	// Handle TLS - only populate from API if it was specified in the plan
	if plan.Tls != nil && response.Data.Tls != nil {
		tlsModel := &TLSWorkloadResourceModel{}
		if response.Data.Tls.Certificate.IsSet() {
			cert := response.Data.Tls.Certificate.Get()
			if cert != nil {
				tlsModel.Certificate = types.Int64Value(*cert)
			}
		}
		if response.Data.Tls.Ciphers != nil {
			tlsModel.Ciphers = types.Int64Value(*response.Data.Tls.Ciphers)
		}
		if response.Data.Tls.MinimumVersion.IsSet() {
			minVer := response.Data.Tls.MinimumVersion.Get()
			if minVer != nil {
				tlsModel.MinimumVersion = types.StringValue(*minVer)
			}
		}
		result.Tls = tlsModel
	}

	// Handle Protocols - only populate from API if it was specified in the plan
	if plan.Protocols != nil && response.Data.Protocols != nil {
		protocolsModel := &ProtocolsResourceModel{}
		// Only populate Http if it was specified in the plan to avoid
		// "Provider produced inconsistent result after apply" when the API
		// echoes back an http object the user didn't configure.
		if plan.Protocols.Http != nil && response.Data.Protocols.Http != nil {
			httpModel := &HttpProtocolResourceModel{}
			if response.Data.Protocols.Http.Versions != nil {
				versionsList, _ := types.ListValueFrom(ctx, types.StringType, response.Data.Protocols.Http.Versions)
				httpModel.Versions = versionsList
			} else {
				httpModel.Versions = types.ListNull(types.StringType)
			}
			if response.Data.Protocols.Http.HttpPorts != nil {
				httpPortsList, _ := types.ListValueFrom(ctx, types.Int64Type, response.Data.Protocols.Http.HttpPorts)
				httpModel.HttpPorts = httpPortsList
			} else {
				httpModel.HttpPorts = types.ListNull(types.Int64Type)
			}
			if response.Data.Protocols.Http.HttpsPorts != nil {
				httpsPortsList, _ := types.ListValueFrom(ctx, types.Int64Type, response.Data.Protocols.Http.HttpsPorts)
				httpModel.HttpsPorts = httpsPortsList
			} else {
				httpModel.HttpsPorts = types.ListNull(types.Int64Type)
			}
			if response.Data.Protocols.Http.QuicPorts != nil {
				quicPortsList, _ := types.ListValueFrom(ctx, types.Int64Type, response.Data.Protocols.Http.QuicPorts)
				httpModel.QuicPorts = quicPortsList
			} else {
				httpModel.QuicPorts = types.ListNull(types.Int64Type)
			}
			protocolsModel.Http = httpModel
		}
		result.Protocols = protocolsModel
	}

	// Handle MTLS - only populate from API if it was specified in the plan
	if plan.Mtls != nil && response.Data.Mtls != nil {
		mtlsModel := &MTLSResourceModel{}
		if response.Data.Mtls.Enabled.IsSet() {
			enabled := response.Data.Mtls.Enabled.Get()
			if enabled != nil {
				mtlsModel.Enabled = types.BoolValue(*enabled)
			}
		}
		// Only populate Config if it was specified in the plan to avoid
		// "Provider produced inconsistent result after apply" when the API
		// echoes back a config object with all-null inner fields.
		if plan.Mtls.Config != nil && response.Data.Mtls.Config.IsSet() {
			config := response.Data.Mtls.Config.Get()
			if config != nil {
				configModel := &MTLSConfigResourceModel{}
				if config.Certificate.IsSet() {
					cert := config.Certificate.Get()
					if cert != nil {
						configModel.Certificate = types.Int64Value(*cert)
					}
				}
				if config.Crl != nil {
					crlList, _ := types.ListValueFrom(ctx, types.Int64Type, config.Crl)
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
		result.Mtls = mtlsModel
	}

	// Handle Domains.
	if response.Data.Domains != nil {
		domainsSet, _ := types.SetValueFrom(ctx, types.StringType, response.Data.Domains)
		result.Domains = domainsSet
	} else {
		result.Domains = types.SetNull(types.StringType)
	}

	return result
}
