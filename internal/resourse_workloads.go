package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-v4-go-sdk/edge"
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

type WorkloadResourceModel struct {
	Workload    *WorkloadData `tfsdk:"workload"`
	ID          types.String  `tfsdk:"id"`
	LastUpdated types.String  `tfsdk:"last_updated"`
}

type WorkloadData struct {
	ID               types.Int64  `tfsdk:"id"`
	EdgeApplication  types.Int64  `tfsdk:"edge_application"`
	EdgeFirewall     types.Int64  `tfsdk:"edge_firewall"`
	Name             types.String `tfsdk:"name"`
	AlternateDomains types.Set    `tfsdk:"alternate_domains"`
	IsActive         types.Bool   `tfsdk:"is_active"`
	NetworkMap       types.String `tfsdk:"network_map"`
	LastEditor       types.String `tfsdk:"last_editor"`
	LastModified     types.String `tfsdk:"last_modified"`
	TLS              *TLSConfig   `tfsdk:"tls"`
	Protocols        *Protocols   `tfsdk:"protocols"`
	MTLS             *MTLSConfig  `tfsdk:"mtls"`
	ProductVersion   types.String `tfsdk:"product_version"`
}

type TLSConfig struct {
	Certificate types.Int64  `tfsdk:"certificate"`
	Ciphers     types.String `tfsdk:"ciphers"`
	MinVersion  types.String `tfsdk:"minimum_version"`
}

type Protocols struct {
	HTTP *HTTPProtocols `tfsdk:"http"`
}

type HTTPProtocols struct {
	Versions   types.Set `tfsdk:"versions"`
	HTTPPorts  types.Set `tfsdk:"http_ports"`
	HTTPSPorts types.Set `tfsdk:"https_ports"`
	QuicPorts  types.Set `tfsdk:"quic_ports"`
}

type MTLSConfig struct {
	Verification types.String `tfsdk:"verification"`
	Certificate  types.Int64  `tfsdk:"certificate"`
	CRL          types.Set    `tfsdk:"crl"`
}

func (r *workloadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload"
}

func (r *workloadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
						Computed:    true,
						Description: "ID of this workload.",
					},
					"edge_application": schema.Int64Attribute{
						Optional:    true,
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
					"is_active": schema.BoolAttribute{
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

func (r *workloadResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *workloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WorkloadResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	isActive := plan.Workload.IsActive.ValueBool()

	workload := edge.WorkloadRequest{
		EdgeApplication: plan.Workload.EdgeApplication.ValueInt64(),
		Active:          &isActive,
		Name:            plan.Workload.Name.ValueString(),
	}

	alternateDomains := plan.Workload.AlternateDomains.ElementsAs(ctx, &workload.AlternateDomains, false)
	resp.Diagnostics.Append(alternateDomains...)
	if resp.Diagnostics.HasError() {
		return
	}

	tlsObject := edge.TLS{}
	if plan.Workload.TLS != nil {
		if plan.Workload.TLS.Certificate.ValueInt64() > 0 {
			tlsObject.SetCertificate(plan.Workload.TLS.Certificate.ValueInt64())
		}
		if plan.Workload.TLS.Ciphers.ValueString() != "" {
			cipher := edge.TLSCiphers{
				String: plan.Workload.TLS.Ciphers.ValueStringPointer(),
			}
			tlsObject.SetCiphers(cipher)
		}
		if plan.Workload.TLS.MinVersion.ValueString() != "" {
			minVer := edge.TLSMinimumVersion{
				String: plan.Workload.TLS.MinVersion.ValueStringPointer(),
			}
			tlsObject.SetMinimumVersion(minVer)
		}
	}
	workload.SetTls(edge.TLSRequest(tlsObject))

	mtlsObject := edge.MTLSRequest{}
	if plan.Workload.MTLS != nil {
		crl := plan.Workload.MTLS.CRL.ElementsAs(ctx, &mtlsObject.Crl, false)
		resp.Diagnostics.Append(crl...)
		if resp.Diagnostics.HasError() {
			return
		}
		if plan.Workload.MTLS.Verification.ValueString() != "" {
			mtlsObject.SetVerification(plan.Workload.MTLS.Verification.ValueString())
		}
		if plan.Workload.MTLS.Certificate.ValueInt64() > 0 {
			mtlsObject.SetCertificate(plan.Workload.MTLS.Certificate.ValueInt64())
		}
	}

	workload.SetMtls(mtlsObject)

	if plan.Workload.EdgeFirewall.ValueInt64() > 0 {
		workload.EdgeFirewall.Set(plan.Workload.EdgeFirewall.ValueInt64Pointer())
	}

	httpObject := &edge.HttpProtocolRequest{
		Versions:   []string{},
		HttpPorts:  []int64{},
		HttpsPorts: []int64{},
		QuicPorts:  []int64{},
	}
	httpProtocols := edge.ProtocolsRequest{
		Http: httpObject,
	}
	if plan.Workload.Protocols != nil {
		if plan.Workload.Protocols != nil {
			versions := plan.Workload.Protocols.HTTP.Versions.ElementsAs(ctx, &httpProtocols.Http.Versions, false)
			resp.Diagnostics.Append(versions...)
			if resp.Diagnostics.HasError() {
				return
			}

			httpPorts := plan.Workload.Protocols.HTTP.HTTPPorts.ElementsAs(ctx, &httpProtocols.Http.HttpPorts, false)
			resp.Diagnostics.Append(httpPorts...)
			if resp.Diagnostics.HasError() {
				return
			}

			httpsPorts := plan.Workload.Protocols.HTTP.HTTPSPorts.ElementsAs(ctx, &httpProtocols.Http.HttpsPorts, false)
			resp.Diagnostics.Append(httpsPorts...)
			if resp.Diagnostics.HasError() {
				return
			}

			quicPorts := plan.Workload.Protocols.HTTP.QuicPorts.ElementsAs(ctx, &httpProtocols.Http.QuicPorts, false)
			resp.Diagnostics.Append(quicPorts...)
			if resp.Diagnostics.HasError() {
				return
			}
			workload.SetProtocols(httpProtocols)
		}
	}

	createWorkload, response, err := r.client.workloadsApi.WorkloadsAPI.CreateWorkload(ctx).WorkloadRequest(workload).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			createWorkload, response, err = utils.RetryOn429(func() (*edge.ResponseWorkload, *http.Response, error) {
				return r.client.workloadsApi.WorkloadsAPI.CreateWorkload(ctx).WorkloadRequest(workload).Execute() //nolint
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
			if response != nil {
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
			} else {
				resp.Diagnostics.AddError(
					err.Error(),
					"Response body is nill",
				)
			}

			return
		}
	}

	var slice []types.String = []types.String{}
	for _, altDomain := range createWorkload.Data.AlternateDomains {
		slice = append(slice, types.StringValue(altDomain))
	}
	dataObject := WorkloadData{
		ID:               types.Int64Value(createWorkload.Data.GetId()),
		Name:             types.StringValue(createWorkload.Data.GetName()),
		IsActive:         types.BoolValue(createWorkload.Data.GetActive()),
		AlternateDomains: utils.SliceStringTypeToSetOrNull(slice),
		NetworkMap:       types.StringValue(createWorkload.Data.GetNetworkMap()),
		EdgeApplication:  types.Int64Value(workload.EdgeApplication),
		ProductVersion:   types.StringValue(createWorkload.Data.ProductVersion),
	}

	if plan.Workload.TLS != nil {
		dataObject.TLS = plan.Workload.TLS
	}
	if plan.Workload.MTLS != nil {
		dataObject.MTLS = plan.Workload.MTLS
	}

	if plan.Workload.Protocols != nil {
		dataObject.Protocols = plan.Workload.Protocols
	}

	plan.Workload = &dataObject

	plan.ID = types.StringValue("Create Workload")
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WorkloadResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var workloadId string
	if state.Workload != nil {
		workloadId = strconv.Itoa(int(state.Workload.ID.ValueInt64()))
	} else {
		workloadId = state.ID.ValueString()
	}

	getWorkload, response, err := r.client.workloadsApi.WorkloadsAPI.
		RetrieveWorkload(ctx, workloadId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getWorkload, response, err = utils.RetryOn429(func() (*edge.ResponseRetrieveWorkload, *http.Response, error) {
				return r.client.workloadsApi.WorkloadsAPI.RetrieveWorkload(ctx, workloadId).Execute() //nolint
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
	for _, altDomain := range getWorkload.Data.AlternateDomains {
		slice = append(slice, types.StringValue(altDomain))
	}
	dataObject := WorkloadData{
		ID:               types.Int64Value(getWorkload.Data.GetId()),
		Name:             types.StringValue(getWorkload.Data.GetName()),
		IsActive:         types.BoolValue(getWorkload.Data.GetActive()),
		AlternateDomains: utils.SliceStringTypeToSetOrNull(slice),
		NetworkMap:       types.StringValue(getWorkload.Data.GetNetworkMap()),
		ProductVersion:   types.StringValue(getWorkload.Data.ProductVersion),
	}
	if state.Workload.TLS != nil {
		dataObject.TLS = state.Workload.TLS
	}
	if state.Workload.MTLS != nil {
		dataObject.MTLS = state.Workload.MTLS
	}

	state.Workload = &dataObject

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WorkloadResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WorkloadResourceModel
	diagsWorkload := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsWorkload...)
	if resp.Diagnostics.HasError() {
		return
	}

	workloadId := strconv.Itoa(int(state.Workload.ID.ValueInt64()))
	isActive := plan.Workload.IsActive.ValueBool()

	updateWorkloadRequest := edge.PatchedWorkloadRequest{
		Active: &isActive,
		Name:   plan.Workload.Name.ValueStringPointer(),
	}

	alternateDomains := plan.Workload.AlternateDomains.ElementsAs(ctx, &updateWorkloadRequest.AlternateDomains, false)
	resp.Diagnostics.Append(alternateDomains...)
	if resp.Diagnostics.HasError() {
		return
	}

	tlsObject := edge.TLS{}
	if plan.Workload.TLS != nil {
		if plan.Workload.TLS.Certificate.ValueInt64() > 0 {
			tlsObject.SetCertificate(plan.Workload.TLS.Certificate.ValueInt64())
		}
		if plan.Workload.TLS.Ciphers.ValueString() != "" {
			cipher := edge.TLSCiphers{
				String: plan.Workload.TLS.Ciphers.ValueStringPointer(),
			}
			tlsObject.SetCiphers(cipher)
		}
		if plan.Workload.TLS.MinVersion.ValueString() != "" {
			minVer := edge.TLSMinimumVersion{
				String: plan.Workload.TLS.MinVersion.ValueStringPointer(),
			}
			tlsObject.SetMinimumVersion(minVer)
		}
	}
	updateWorkloadRequest.SetTls(edge.TLSRequest(tlsObject))

	mtlsObject := edge.MTLSRequest{}

	if plan.Workload.MTLS != nil {
		crl := plan.Workload.MTLS.CRL.ElementsAs(ctx, &mtlsObject.Crl, false)
		resp.Diagnostics.Append(crl...)
		if resp.Diagnostics.HasError() {
			return
		}
		if plan.Workload.MTLS.Verification.ValueString() != "" {
			mtlsObject.SetVerification(plan.Workload.MTLS.Verification.ValueString())
		}
		if plan.Workload.MTLS.Certificate.ValueInt64() > 0 {
			mtlsObject.SetCertificate(plan.Workload.MTLS.Certificate.ValueInt64())
		}
	}
	updateWorkloadRequest.SetMtls(mtlsObject)

	httpObject := &edge.HttpProtocolRequest{
		Versions:   []string{},
		HttpPorts:  []int64{},
		HttpsPorts: []int64{},
		QuicPorts:  []int64{},
	}
	httpProtocols := edge.ProtocolsRequest{
		Http: httpObject,
	}
	if plan.Workload.Protocols != nil {
		if plan.Workload.Protocols != nil {
			versions := plan.Workload.Protocols.HTTP.Versions.ElementsAs(ctx, &httpProtocols.Http.Versions, false)
			resp.Diagnostics.Append(versions...)
			if resp.Diagnostics.HasError() {
				return
			}

			httpPorts := plan.Workload.Protocols.HTTP.HTTPPorts.ElementsAs(ctx, &httpProtocols.Http.HttpPorts, false)
			resp.Diagnostics.Append(httpPorts...)
			if resp.Diagnostics.HasError() {
				return
			}

			httpsPorts := plan.Workload.Protocols.HTTP.HTTPSPorts.ElementsAs(ctx, &httpProtocols.Http.HttpsPorts, false)
			resp.Diagnostics.Append(httpsPorts...)
			if resp.Diagnostics.HasError() {
				return
			}

			quicPorts := plan.Workload.Protocols.HTTP.QuicPorts.ElementsAs(ctx, &httpProtocols.Http.QuicPorts, false)
			resp.Diagnostics.Append(quicPorts...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateWorkloadRequest.SetProtocols(httpProtocols)
		}
	}

	updateWorkload, response, err := r.client.workloadsApi.WorkloadsAPI.PartialUpdateWorkload(ctx, workloadId).PatchedWorkloadRequest(updateWorkloadRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateWorkload, response, err = utils.RetryOn429(func() (*edge.ResponseWorkload, *http.Response, error) {
				return r.client.workloadsApi.WorkloadsAPI.PartialUpdateWorkload(ctx, workloadId).PatchedWorkloadRequest(updateWorkloadRequest).Execute() //nolint
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
	for _, altDomain := range updateWorkload.Data.AlternateDomains {
		slice = append(slice, types.StringValue(altDomain))
	}
	dataObject := WorkloadData{
		ID:               types.Int64Value(updateWorkload.Data.GetId()),
		Name:             types.StringValue(updateWorkload.Data.GetName()),
		IsActive:         types.BoolValue(updateWorkload.Data.GetActive()),
		AlternateDomains: utils.SliceStringTypeToSetOrNull(slice),
		NetworkMap:       types.StringValue(updateWorkload.Data.GetNetworkMap()),
		ProductVersion:   types.StringValue(updateWorkload.Data.ProductVersion),
	}

	if plan.Workload.EdgeApplication.ValueInt64() > 0 {
		dataObject.EdgeApplication = plan.Workload.EdgeApplication
	}
	if plan.Workload.EdgeFirewall.ValueInt64() > 0 {
		dataObject.EdgeFirewall = plan.Workload.EdgeFirewall
	}

	if plan.Workload.TLS != nil {
		dataObject.TLS = plan.Workload.TLS
	}
	if plan.Workload.MTLS != nil {
		dataObject.MTLS = plan.Workload.MTLS
	}
	if plan.Workload.Protocols != nil {
		dataObject.Protocols = plan.Workload.Protocols
	}

	plan.Workload = &dataObject

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *workloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WorkloadResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	workloadId := strconv.Itoa(int(state.Workload.ID.ValueInt64()))
	_, response, err := r.client.workloadsApi.WorkloadsAPI.DestroyWorkload(ctx, workloadId).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edge.ResponseDeleteWorkload, *http.Response, error) {
				return r.client.workloadsApi.WorkloadsAPI.DestroyWorkload(ctx, workloadId).Execute() //nolint
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
}

func (r *workloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
