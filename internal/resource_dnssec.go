package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/idns"
	dnsapi "github.com/aziontech/azionapi-v4-go-sdk-dev/dns-api"
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
	_ resource.Resource                = &dnssecResource{}
	_ resource.ResourceWithConfigure   = &dnssecResource{}
	_ resource.ResourceWithImportState = &dnssecResource{}
)

func NewDnssecResource() resource.Resource {
	return &dnssecResource{}
}

type dnssecResource struct {
	client *apiClient
}

type dnssecResourceModel struct {
	ZoneId      types.String `tfsdk:"zone_id"`
	DnsSec      *dnsSecModel `tfsdk:"dns_sec"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type dnsSecModel struct {
	Enabled          types.Bool                `tfsdk:"is_enabled"`
	Status           types.String              `tfsdk:"status"`
	DelegationSigner *DnsDelegationSignerModel `tfsdk:"delegation_signer"`
}

type DnsDelegationSignerModel struct {
	AlgorithmType *DnsDelegationSignerDigestType `tfsdk:"algorithm_type"`
	Digest        types.String                   `tfsdk:"digest"`
	DigestType    *DnsDelegationSignerDigestType `tfsdk:"digest_type"`
	KeyTag        types.Int64                    `tfsdk:"key_tag"`
}

type DnsDelegationSignerDigestType struct {
	Id   types.Int64  `tfsdk:"id"`
	Slug types.String `tfsdk:"slug"`
}

func (r *dnssecResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_dnssec"
}

func (r *dnssecResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				Required:    true,
				Description: "The zone identifier to target for the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
			},
			"dns_sec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"is_enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Zone DNSSEC flags for enabled.",
					},
					"status": schema.StringAttribute{
						Computed:    true,
						Description: "The status of the Zone DNSSEC.",
					},
					"delegation_signer": schema.SingleNestedAttribute{
						Description: "Zone DNSSEC delegation-signer.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"digest_type": schema.SingleNestedAttribute{
								Computed:    true,
								Description: "Digest Type for Zone DNSSEC.",
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Description: "The ID of this digest.",
										Computed:    true,
									},
									"slug": schema.StringAttribute{
										Description: "The Slug of this digest.",
										Computed:    true,
									},
								},
							},
							"algorithm_type": schema.SingleNestedAttribute{
								Computed:    true,
								Description: "Digest algorithm used for Zone DNSSEC.",
								Attributes: map[string]schema.Attribute{
									"id": schema.Int64Attribute{
										Description: "The ID of this digest.",
										Computed:    true,
									},
									"slug": schema.StringAttribute{
										Description: "The Slug of this digest.",
										Computed:    true,
									},
								},
							},
							"digest": schema.StringAttribute{
								Computed:    true,
								Description: "Zone DNSSEC digest.",
							},
							"key_tag": schema.Int64Attribute{
								Computed:    true,
								Description: "Key Tag for the Zone DNSSEC.",
							},
						},
					},
				},
			},
		},
	}
}

func (r *dnssecResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnssecResourceModel
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idPlan, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	dnsSec := dnsapi.PatchedDNSSECRequest{
		Enabled: idns.PtrBool(plan.DnsSec.Enabled.ValueBool()),
	}

	enableDnsSec, response, err := r.client.idnsApi.DNSDNSSECAPI.
		PartialUpdateDnssec(ctx, idPlan).PatchedDNSSECRequest(dnsSec).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			enableDnsSec, response, err = utils.RetryOn429(func() (*dnsapi.ResponseDNSSEC, *http.Response, error) {
				return r.client.idnsApi.DNSDNSSECAPI.PartialUpdateDnssec(ctx, idPlan).PatchedDNSSECRequest(dnsSec).Execute() //nolint
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

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	delegationSigner := enableDnsSec.Data.GetDelegationSigner()

	plan.DnsSec = &dnsSecModel{
		Enabled: types.BoolValue(enableDnsSec.Data.GetEnabled()),
		Status:  types.StringValue(enableDnsSec.Data.GetStatus()),
		DelegationSigner: &DnsDelegationSignerModel{
			DigestType: &DnsDelegationSignerDigestType{
				Id:   types.Int64Value(int64(delegationSigner.DigestType.GetId())),
				Slug: types.StringValue(delegationSigner.DigestType.GetSlug()),
			},
			AlgorithmType: &DnsDelegationSignerDigestType{
				Id:   types.Int64Value(int64(delegationSigner.AlgorithmType.GetId())),
				Slug: types.StringValue(delegationSigner.AlgorithmType.GetSlug()),
			},
			Digest: types.StringValue(delegationSigner.GetDigest()),
			KeyTag: types.Int64Value(int64(delegationSigner.GetKeyTag())),
		},
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *dnssecResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*apiClient)
}

func (r *dnssecResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnssecResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneID, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	getDnsSec, response, err := r.client.idnsApi.DNSDNSSECAPI.
		RetrieveDnssec(ctx, zoneID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getDnsSec, response, err = utils.RetryOn429(func() (*dnsapi.ResponseRetrieveDNSSEC, *http.Response, error) {
				return r.client.idnsApi.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneID).Execute() //nolint
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

	delegationSigner := getDnsSec.Data.GetDelegationSigner()

	state.DnsSec = &dnsSecModel{
		Enabled: types.BoolValue(getDnsSec.Data.GetEnabled()),
		Status:  types.StringValue(getDnsSec.Data.GetStatus()),
		DelegationSigner: &DnsDelegationSignerModel{
			DigestType: &DnsDelegationSignerDigestType{
				Id:   types.Int64Value(int64(delegationSigner.DigestType.GetId())),
				Slug: types.StringValue(delegationSigner.DigestType.GetSlug()),
			},
			AlgorithmType: &DnsDelegationSignerDigestType{
				Id:   types.Int64Value(int64(delegationSigner.AlgorithmType.GetId())),
				Slug: types.StringValue(delegationSigner.AlgorithmType.GetSlug()),
			},
			Digest: types.StringValue(delegationSigner.GetDigest()),
			KeyTag: types.Int64Value(int64(delegationSigner.GetKeyTag())),
		},
	}
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *dnssecResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnssecResourceModel
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idPlan, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	dnsSec := dnsapi.PatchedDNSSECRequest{
		Enabled: idns.PtrBool(plan.DnsSec.Enabled.ValueBool()),
	}

	enableDnsSec, response, err := r.client.idnsApi.DNSDNSSECAPI.
		PartialUpdateDnssec(ctx, idPlan).PatchedDNSSECRequest(dnsSec).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			enableDnsSec, response, err = utils.RetryOn429(func() (*dnsapi.ResponseDNSSEC, *http.Response, error) {
				return r.client.idnsApi.DNSDNSSECAPI.PartialUpdateDnssec(ctx, idPlan).PatchedDNSSECRequest(dnsSec).Execute() //nolint
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

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	delegationSigner := enableDnsSec.Data.GetDelegationSigner()

	plan.DnsSec = &dnsSecModel{
		Enabled: types.BoolValue(enableDnsSec.Data.GetEnabled()),
		Status:  types.StringValue(enableDnsSec.Data.GetStatus()),
		DelegationSigner: &DnsDelegationSignerModel{
			DigestType: &DnsDelegationSignerDigestType{
				Id:   types.Int64Value(int64(delegationSigner.DigestType.GetId())),
				Slug: types.StringValue(delegationSigner.DigestType.GetSlug()),
			},
			AlgorithmType: &DnsDelegationSignerDigestType{
				Id:   types.Int64Value(int64(delegationSigner.AlgorithmType.GetId())),
				Slug: types.StringValue(delegationSigner.AlgorithmType.GetSlug()),
			},
			Digest: types.StringValue(delegationSigner.GetDigest()),
			KeyTag: types.Int64Value(int64(delegationSigner.GetKeyTag())),
		},
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *dnssecResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnssecResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idPlan, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	dnsSec := dnsapi.PatchedDNSSECRequest{
		Enabled: idns.PtrBool(false),
	}

	_, response, err := r.client.idnsApi.DNSDNSSECAPI.
		PartialUpdateDnssec(ctx, idPlan).PatchedDNSSECRequest(dnsSec).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*dnsapi.ResponseDNSSEC, *http.Response, error) {
				return r.client.idnsApi.DNSDNSSECAPI.PartialUpdateDnssec(ctx, idPlan).PatchedDNSSECRequest(dnsSec).Execute() //nolint
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

	resp.State.RemoveResource(ctx)
}

func (r *dnssecResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
