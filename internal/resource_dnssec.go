package provider

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/idns"
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
	ZoneId        types.String `tfsdk:"zone_id"`
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	DnsSec        *dnsSecModel `tfsdk:"dns_sec"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

type dnsSecModel struct {
	IsEnabled types.Bool `tfsdk:"is_enabled"`
	//Status           types.String              `tfsdk:"status"`
	//DelegationSigner *DnsDelegationSignerModel `tfsdk:"delegation_signer"`
}

//type DnsDelegationSignerModel struct {
//	DigestType    *DnsDelegationSignerDigestType `tfsdk:"digesttype"`
//	AlgorithmType *DnsDelegationSignerDigestType `tfsdk:"algorithmtype"`
//	Digest        types.String                   `tfsdk:"digest"`
//	KeyTag        types.Int64                    `tfsdk:"keytag"`
//}
//
//type DnsDelegationSignerDigestType struct {
//	Id   types.Int64  `tfsdk:"id"`
//	Slug types.String `tfsdk:"slug"`
//}

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
					//"status": schema.StringAttribute{
					//	Computed:    true,
					//	Description: "The status of the Zone DNSSEC.",
					//},
					//"delegation_signer": schema.SingleNestedAttribute{
					//	Description: "Zone DNSSEC delegation-signer.",
					//	Computed:    true,
					//	Attributes:  DnsDelegationSigner(),
					//},
				},
			},
		},
	}
}

//func DnsDelegationSigner() map[string]schema.Attribute {
//	return map[string]schema.Attribute{
//		"digesttype": schema.SingleNestedAttribute{
//			Computed:    true,
//			Description: "Digest Type for Zone DNSSEC.",
//			Attributes:  DnsDelegationSignerDigestTypeScheme(),
//		},
//		"algorithmtype": schema.SingleNestedAttribute{
//			Computed:    true,
//			Description: "Digest algorithm use for Zone DNSSEC.",
//			Attributes:  DnsDelegationSignerDigestTypeScheme(),
//		},
//		"digest": schema.StringAttribute{
//			Computed:    true,
//			Description: "Zone DNSSEC digest.",
//		},
//		"keytag": schema.Int64Attribute{
//			Computed:    true,
//			Description: "Key Tag for the Zone DNSSEC.",
//		},
//	}
//}
//func DnsDelegationSignerDigestTypeScheme() map[string]schema.Attribute {
//	return map[string]schema.Attribute{
//		"id": schema.Int64Attribute{
//			Description: "The ID of this digest.",
//			Computed:    true,
//		},
//		"slug": schema.StringAttribute{
//			Description: "The Slug of this digest.",
//			Computed:    true,
//		},
//	}
//}

func (r *dnssecResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*apiClient)
}

func (r *dnssecResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnssecResourceModel
	diags := req.Config.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneId, err := strconv.ParseUint(plan.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}
	dnsSec := idns.DnsSec{
		IsEnabled: idns.PtrBool(plan.DnsSec.IsEnabled.ValueBool()),
	}

	enableDnsSec, response, err := r.client.idnsApi.DNSSECAPI.PutZoneDnsSec(ctx, int32(zoneId)).DnsSec(dnsSec).Execute()
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
	plan.SchemaVersion = types.Int64Value(int64(*enableDnsSec.SchemaVersion))
	plan.DnsSec = &dnsSecModel{
		IsEnabled: types.BoolValue(*enableDnsSec.Results.IsEnabled),
		//Status:    types.StringValue(*enableDnsSec.Results.Status),
		//DelegationSigner: &DnsDelegationSignerModel{
		//	DigestType: &DnsDelegationSignerDigestType{
		//		Id:   types.Int64Value(int64(*enableDnsSec.Results.DelegationSigner.DigestType.Id)),
		//		Slug: types.StringValue(*enableDnsSec.Results.DelegationSigner.DigestType.Slug),
		//	},
		//	AlgorithmType: &DnsDelegationSignerDigestType{
		//		Id:   types.Int64Value(int64(*enableDnsSec.Results.DelegationSigner.AlgorithmType.Id)),
		//		Slug: types.StringValue(*enableDnsSec.Results.DelegationSigner.AlgorithmType.Slug),
		//	},
		//	Digest: types.StringValue(*enableDnsSec.Results.DelegationSigner.Digest),
		//	KeyTag: types.Int64Value(int64(*enableDnsSec.Results.DelegationSigner.KeyTag)),
		//},
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *dnssecResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnssecResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneId, err := strconv.ParseUint(state.ZoneId.ValueString(), 10, 16)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}
	getDnsSec, response, err := r.client.idnsApi.DNSSECAPI.GetZoneDnsSec(ctx, int32(zoneId)).Execute()
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

	state.DnsSec = &dnsSecModel{
		IsEnabled: types.BoolValue(*getDnsSec.Results.IsEnabled),
		//Status:    types.StringValue(*getDnsSec.Results.Status),
		//DelegationSigner: &DnsDelegationSignerModel{
		//	DigestType: &DnsDelegationSignerDigestType{
		//		Id:   types.Int64Value(int64(*getDnsSec.Results.DelegationSigner.DigestType.Id)),
		//		Slug: types.StringValue(*getDnsSec.Results.DelegationSigner.DigestType.Slug),
		//	},
		//	AlgorithmType: &DnsDelegationSignerDigestType{
		//		Id:   types.Int64Value(int64(*getDnsSec.Results.DelegationSigner.AlgorithmType.Id)),
		//		Slug: types.StringValue(*getDnsSec.Results.DelegationSigner.AlgorithmType.Slug),
		//	},
		//	Digest: types.StringValue(*getDnsSec.Results.DelegationSigner.Digest),
		//	KeyTag: types.Int64Value(int64(*getDnsSec.Results.DelegationSigner.KeyTag)),
		//},
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

	idPlan, err := strconv.ParseUint(plan.ZoneId.ValueString(), 10, 16)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	dnsSec := idns.DnsSec{
		IsEnabled: idns.PtrBool(plan.DnsSec.IsEnabled.ValueBool()),
	}

	enableDnsSec, response, err := r.client.idnsApi.DNSSECAPI.PutZoneDnsSec(ctx, int32(idPlan)).DnsSec(dnsSec).Execute()
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
	plan.SchemaVersion = types.Int64Value(int64(*enableDnsSec.SchemaVersion))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.DnsSec = &dnsSecModel{
		IsEnabled: types.BoolValue(*enableDnsSec.Results.IsEnabled),
		//Status:    types.StringValue(*enableDnsSec.Results.Status),
		//DelegationSigner: &DnsDelegationSignerModel{
		//	DigestType: &DnsDelegationSignerDigestType{
		//		Id:   types.Int64Value(int64(*enableDnsSec.Results.DelegationSigner.DigestType.Id)),
		//		Slug: types.StringValue(*enableDnsSec.Results.DelegationSigner.DigestType.Slug),
		//	},
		//	AlgorithmType: &DnsDelegationSignerDigestType{
		//		Id:   types.Int64Value(int64(*enableDnsSec.Results.DelegationSigner.AlgorithmType.Id)),
		//		Slug: types.StringValue(*enableDnsSec.Results.DelegationSigner.AlgorithmType.Slug),
		//	},
		//	Digest: types.StringValue(*enableDnsSec.Results.DelegationSigner.Digest),
		//	KeyTag: types.Int64Value(int64(*enableDnsSec.Results.DelegationSigner.KeyTag)),
		//},
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
	zoneId, err := strconv.ParseUint(state.ZoneId.ValueString(), 10, 16)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}
	dnsSec := idns.DnsSec{
		IsEnabled: idns.PtrBool(false),
	}

	_, response, err := r.client.idnsApi.DNSSECAPI.PutZoneDnsSec(ctx, int32(zoneId)).DnsSec(dnsSec).Execute()
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
}

func (r *dnssecResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
