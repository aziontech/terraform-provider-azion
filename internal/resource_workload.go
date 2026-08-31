package provider

import (
	"context"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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

// Azion API defaults for workload fields.
//
// The configuration is the desired state: a field listed here is reset to its
// default on every apply, so a change made in Azion Console is undone rather than
// silently adopted.
//
// Fields NOT listed are deliberately left without a default. They stay
// Optional + Computed, so an omitted one resolves to whatever the API reports
// instead of being forced to a value — sent and tracked only when declared.
//
// Every optional nested block still needs a default even when its contents are
// not enforced. A Computed block without one is unknown in the plan whenever the
// configuration omits it, and the nested objects here are read into pointer
// fields that cannot hold unknown values, so a missing default breaks Create.
// Inside such a default, a null leaf means "let the API decide": the leaf is
// Computed, so the framework turns that null into an unknown and the response
// fills it.
const (
	// 1 is Production (All Locations); 2 is Staging.
	defaultWorkloadInfrastructure = int64(1)

	defaultWorkloadDomainAllowAccess = true
	defaultWorkloadTLSMinimumVersion = "tls_1_3"
	defaultWorkloadMTLSEnabled       = false
)

var (
	defaultWorkloadHTTPPorts  = []int64{80}
	defaultWorkloadHTTPSPorts = []int64{443}
)

// Attribute types for the nested objects, needed to build object defaults whose
// types must match the schema exactly.
var (
	workloadTLSAttrTypes = map[string]attr.Type{
		"certificate":     types.Int64Type,
		"ciphers":         types.Int64Type,
		"minimum_version": types.StringType,
	}

	workloadHTTPAttrTypes = map[string]attr.Type{
		"versions":    types.ListType{ElemType: types.StringType},
		"http_ports":  types.ListType{ElemType: types.Int64Type},
		"https_ports": types.ListType{ElemType: types.Int64Type},
		"quic_ports":  types.ListType{ElemType: types.Int64Type},
	}

	workloadProtocolsAttrTypes = map[string]attr.Type{
		"http": types.ObjectType{AttrTypes: workloadHTTPAttrTypes},
	}

	workloadMTLSConfigAttrTypes = map[string]attr.Type{
		"certificate":  types.Int64Type,
		"crl":          types.ListType{ElemType: types.Int64Type},
		"verification": types.StringType,
	}

	workloadMTLSAttrTypes = map[string]attr.Type{
		"enabled": types.BoolType,
		"config":  types.ObjectType{AttrTypes: workloadMTLSConfigAttrTypes},
	}
)

var (
	// certificate and ciphers stay null: they are not enforced, so the API
	// supplies them.
	workloadTLSDefault = types.ObjectValueMust(workloadTLSAttrTypes, map[string]attr.Value{
		"certificate":     types.Int64Null(),
		"ciphers":         types.Int64Null(),
		"minimum_version": types.StringValue(defaultWorkloadTLSMinimumVersion),
	})

	// versions and quic_ports stay null: the API decides them, and quic_ports is
	// only meaningful when http3 is among the versions.
	workloadHTTPDefault = types.ObjectValueMust(workloadHTTPAttrTypes, map[string]attr.Value{
		"versions":    types.ListNull(types.StringType),
		"http_ports":  int64ListValue(defaultWorkloadHTTPPorts),
		"https_ports": int64ListValue(defaultWorkloadHTTPSPorts),
		"quic_ports":  types.ListNull(types.Int64Type),
	})

	workloadProtocolsDefault = types.ObjectValueMust(workloadProtocolsAttrTypes, map[string]attr.Value{
		"http": workloadHTTPDefault,
	})

	workloadMTLSDefault = types.ObjectValueMust(workloadMTLSAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(defaultWorkloadMTLSEnabled),
		"config":  types.ObjectNull(workloadMTLSConfigAttrTypes),
	})
)

func int64ListValue(values []int64) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.Int64Value(value))
	}

	return types.ListValueMust(types.Int64Type, elements)
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
						Default:     int64default.StaticInt64(defaultWorkloadInfrastructure),
					},
					"tls": schema.SingleNestedAttribute{
						Description: "TLS configuration for the workload.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(workloadTLSDefault),
						Attributes: map[string]schema.Attribute{
							"certificate": schema.Int64Attribute{
								Description: "Certificate ID for TLS. Supplied by the API when not declared.",
								Optional:    true,
								Computed:    true,
							},
							"ciphers": schema.Int64Attribute{
								Description: "Cipher suite configuration. Supplied by the API when not declared.",
								Optional:    true,
								Computed:    true,
							},
							"minimum_version": schema.StringAttribute{
								Description: "Minimum TLS version: tls_1_0, tls_1_1, tls_1_2 or tls_1_3.",
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString(defaultWorkloadTLSMinimumVersion),
							},
						},
					},
					"protocols": schema.SingleNestedAttribute{
						Description: "Protocol configurations for the workload.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(workloadProtocolsDefault),
						Attributes: map[string]schema.Attribute{
							"http": schema.SingleNestedAttribute{
								Description: "HTTP protocol configuration.",
								Optional:    true,
								Computed:    true,
								Default:     objectdefault.StaticValue(workloadHTTPDefault),
								Attributes: map[string]schema.Attribute{
									"versions": schema.ListAttribute{
										ElementType: types.StringType,
										Description: "HTTP versions supported. Supplied by the API when not declared.",
										Optional:    true,
										Computed:    true,
									},
									"http_ports": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "HTTP ports.",
										Optional:    true,
										Computed:    true,
										Default:     listdefault.StaticValue(int64ListValue(defaultWorkloadHTTPPorts)),
									},
									"https_ports": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "HTTPS ports.",
										Optional:    true,
										Computed:    true,
										Default:     listdefault.StaticValue(int64ListValue(defaultWorkloadHTTPSPorts)),
									},
									"quic_ports": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "QUIC ports. When omitted, the value is determined by the API (QUIC is only required when http3 is present in versions).",
										Optional:    true,
										Computed:    true,
									},
								},
							},
						},
					},
					"mtls": schema.SingleNestedAttribute{
						Description: "Mutual TLS configuration for the workload.",
						Optional:    true,
						Computed:    true,
						Default:     objectdefault.StaticValue(workloadMTLSDefault),
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether MTLS is enabled.",
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(defaultWorkloadMTLSEnabled),
							},
							"config": schema.SingleNestedAttribute{
								Description: "MTLS configuration. Declare it when enabling MTLS; it stays null while MTLS is disabled.",
								Optional:    true,
								Computed:    true,
								Default:     objectdefault.StaticValue(types.ObjectNull(workloadMTLSConfigAttrTypes)),
								Attributes: map[string]schema.Attribute{
									"certificate": schema.Int64Attribute{
										Description: "MTLS certificate ID.",
										Optional:    true,
										Computed:    true,
									},
									"crl": schema.ListAttribute{
										ElementType: types.Int64Type,
										Description: "Certificate Revocation List.",
										Optional:    true,
										Computed:    true,
									},
									"verification": schema.StringAttribute{
										Description: "MTLS verification type: enforce or permissive.",
										Optional:    true,
										Computed:    true,
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
						Default:     booldefault.StaticBool(defaultWorkloadDomainAllowAccess),
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
		if response != nil && response.StatusCode == 429 {
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Populate the state from the response, preserving plan values for optional nested fields.
	plan.Workload = populateWorkloadResults(ctx, createWorkload)
	plan.ID = types.StringValue(strconv.FormatInt(createWorkload.Data.GetId(), 10))
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
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	state.Workload = populateWorkloadResults(ctx, getWorkload)
	state.ID = types.StringValue(strconv.FormatInt(getWorkload.Data.GetId(), 10))

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

	// PATCH, deliberately, unlike the other main-settings resources in this
	// provider. A full WorkloadRequest carries `bindings`, which this resource
	// does not model — those belong to azion_workload_deployment. A PUT would
	// send none and wipe the workload's binding to its deployment.
	//
	// Enforcement is unaffected: every field this resource models carries a
	// schema default, so the plan holds a concrete value and the request below
	// always sends it. PATCH only spares what the provider does not model.
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
		if response != nil && response.StatusCode == 429 {
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
			resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	plan.Workload = populateWorkloadResults(ctx, updateWorkload)
	plan.ID = types.StringValue(strconv.FormatInt(updateWorkload.Data.GetId(), 10))
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

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.WorkloadsAPI.DeleteWorkload(ctx, workloadId).Execute()
	}, 5)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError(err.Error(), utils.ReadAPIErrorBody(response))
		return
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
// populateWorkloadResults mirrors the API response into the resource model
// without filtering. State has to match the remote workload for Terraform to
// plan a revert when something was changed out-of-band; unconfigured fields do
// not drift perpetually because every optional attribute is Computed, either
// with a schema default or resolved from this response.
func populateWorkloadResults(ctx context.Context, response *azionapi.WorkloadResponse) *workloadResourceResults {
	result := &workloadResourceResults{
		ID:             types.Int64Value(response.Data.GetId()),
		Name:           types.StringValue(response.Data.Name),
		Active:         types.BoolPointerValue(response.Data.Active),
		LastEditor:     types.StringValue(response.Data.GetLastEditor()),
		LastModified:   types.StringValue(response.Data.GetLastModified().Format(time.RFC850)),
		CreatedAt:      types.StringValue(response.Data.GetCreatedAt().Format(time.RFC3339)),
		ProductVersion: types.StringValue(response.Data.GetProductVersion()),
		WorkloadDomain: types.StringValue(response.Data.GetWorkloadDomain()),

		Infrastructure:            types.Int64PointerValue(response.Data.Infrastructure),
		WorkloadDomainAllowAccess: types.BoolPointerValue(response.Data.WorkloadDomainAllowAccess),
	}

	if response.Data.Tls != nil {
		result.Tls = &TLSWorkloadResourceModel{
			Certificate:    types.Int64PointerValue(response.Data.Tls.Certificate.Get()),
			Ciphers:        types.Int64PointerValue(response.Data.Tls.Ciphers),
			MinimumVersion: types.StringPointerValue(response.Data.Tls.MinimumVersion.Get()),
		}
	}

	if response.Data.Protocols != nil {
		result.Protocols = &ProtocolsResourceModel{}
		if response.Data.Protocols.Http != nil {
			http := response.Data.Protocols.Http
			result.Protocols.Http = &HttpProtocolResourceModel{
				Versions:   stringListOrNull(ctx, http.Versions),
				HttpPorts:  int64ListOrNull(ctx, http.HttpPorts),
				HttpsPorts: int64ListOrNull(ctx, http.HttpsPorts),
				QuicPorts:  int64ListOrNull(ctx, http.QuicPorts),
			}
		}
	}

	if response.Data.Mtls != nil {
		result.Mtls = &MTLSResourceModel{
			Enabled: types.BoolPointerValue(response.Data.Mtls.Enabled.Get()),
		}

		if config := response.Data.Mtls.Config.Get(); config != nil {
			configModel := &MTLSConfigResourceModel{
				Certificate:  types.Int64PointerValue(config.Certificate.Get()),
				Crl:          int64ListOrNull(ctx, config.Crl),
				Verification: types.StringPointerValue(config.Verification.Get()),
			}

			// The API echoes an all-null config for a workload that has no mTLS
			// configured. That object carries nothing, so treat it as absent:
			// writing it to state would contradict the null the plan holds while
			// mtls is disabled, failing the apply with "Provider produced
			// inconsistent result after apply".
			if !configModel.isEmpty() {
				result.Mtls.Config = configModel
			}
		}
	}

	if response.Data.Domains != nil {
		domains, _ := types.SetValueFrom(ctx, types.StringType, response.Data.Domains)
		result.Domains = domains
	} else {
		result.Domains = types.SetNull(types.StringType)
	}

	return result
}

// isEmpty reports whether the config carries no values at all, which is how the
// API represents "no mTLS configuration" rather than omitting the object.
func (m *MTLSConfigResourceModel) isEmpty() bool {
	if !m.Certificate.IsNull() || !m.Verification.IsNull() {
		return false
	}

	return m.Crl.IsNull() || len(m.Crl.Elements()) == 0
}

func stringListOrNull(ctx context.Context, values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}

	list, _ := types.ListValueFrom(ctx, types.StringType, values)

	return list
}

func int64ListOrNull(ctx context.Context, values []int64) types.List {
	if values == nil {
		return types.ListNull(types.Int64Type)
	}

	list, _ := types.ListValueFrom(ctx, types.Int64Type, values)

	return list
}
