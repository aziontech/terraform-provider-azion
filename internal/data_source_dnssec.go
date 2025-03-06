package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dnsSecDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsSecDataSource{}
)

func dataSourceAzionDNSSec() datasource.DataSource {
	return &dnsSecDataSource{}
}

type dnsSecDataSource struct {
	client *apiClient
}

type dnsSecDataSourceModel struct {
	ID            types.String   `tfsdk:"id"`
	ZoneId        types.Int64    `tfsdk:"zone_id"`
	SchemaVersion types.Int64    `tfsdk:"schema_version"`
	DnsSec        *dnsSecDSModel `tfsdk:"dns_sec"`
}

type dnsSecDSModel struct {
	IsEnabled        types.Bool                  `tfsdk:"is_enabled"`
	Status           types.String                `tfsdk:"status"`
	DelegationSigner *DnsDelegationSignerDSModel `tfsdk:"delegation_signer"`
}
type DnsDelegationSignerDSModel struct {
	DigestType    *DigestTypeDS    `tfsdk:"digesttype"`
	AlgorithmType *AlgorithmTypeDS `tfsdk:"algorithmtype"`
	Digest        types.String     `tfsdk:"digest"`
	KeyTag        types.Int64      `tfsdk:"keytag"`
}

type DigestTypeDS struct {
	Id   types.Int64  `tfsdk:"id"`
	Slug types.String `tfsdk:"slug"`
}

type AlgorithmTypeDS struct {
	Id   types.Int64  `tfsdk:"id"`
	Slug types.String `tfsdk:"slug"`
}

func (d *dnsSecDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *dnsSecDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_dnssec"
}

func (d *dnsSecDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
			},
			"zone_id": schema.Int64Attribute{
				Description: "The zone identifier to target for the resource.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Optional:    true,
			},
			"dns_sec": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"is_enabled": schema.BoolAttribute{
						Optional:    true,
						Description: "Zone DNSSEC flags for enabled.",
					},
					"status": schema.StringAttribute{
						Optional:    true,
						Description: "The status of the Zone DNSSEC.",
					},
					"delegation_signer": schema.SingleNestedAttribute{
						Description: "Zone DNSSEC delegation-signer.",
						Optional:    true,
						Attributes:  DnsDelegationSignerDS(),
					},
				},
			},
		},
	}
}
func DnsDelegationSignerDS() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"digesttype": schema.SingleNestedAttribute{
			Description: "Digest Type for Zone DNSSEC.",
			Computed:    true,
			Attributes:  digesttypeDS(),
		},
		"algorithmtype": schema.SingleNestedAttribute{
			Description: "Digest algorithm use for Zone DNSSEC.",
			Computed:    true,
			Attributes:  algorithmtypeDS(),
		},
		"digest": schema.StringAttribute{
			Optional:    true,
			Description: "Zone DNSSEC digest.",
		},
		"keytag": schema.Int64Attribute{
			Optional:    true,
			Description: "Key Tag for the Zone DNSSEC.",
		},
	}
}
func digesttypeDS() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Description: "The ID of this digest.",
			Computed:    true,
		},
		"slug": schema.StringAttribute{
			Description: "The Slug of this digest.",
			Computed:    true,
		},
	}
}
func algorithmtypeDS() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Description: "The ID of this algorithm.",
			Computed:    true,
		},
		"slug": schema.StringAttribute{
			Description: "The Slug of this algorithm.",
			Computed:    true,
		},
	}
}

func (d *dnsSecDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getZoneId types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("zone_id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID32, err := utils.CheckInt64toInt32Security(getZoneId.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, getZoneId.ValueInt64())
		return
	}

	getDnsSec, response, err := d.client.idnsApi.DNSSECAPI.GetZoneDnsSec(ctx, zoneID32).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*idns.GetOrPatchDnsSecResponse, *http.Response, error) {
				return d.client.idnsApi.DNSSECAPI.GetZoneDnsSec(ctx, zoneID32).Execute() //nolint
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

	if getDnsSec.Results.DelegationSigner != nil {
		dnsSecState := &dnsSecDataSourceModel{
			SchemaVersion: types.Int64Value(int64(*getDnsSec.SchemaVersion)),
			ZoneId:        getZoneId,
			DnsSec: &dnsSecDSModel{
				IsEnabled: types.BoolValue(*getDnsSec.Results.IsEnabled),
				Status:    types.StringValue(*getDnsSec.Results.Status),
				DelegationSigner: &DnsDelegationSignerDSModel{
					DigestType: &DigestTypeDS{
						Id:   types.Int64Value(int64(*getDnsSec.Results.DelegationSigner.DigestType.Id)),
						Slug: types.StringValue(*getDnsSec.Results.DelegationSigner.DigestType.Slug),
					},
					AlgorithmType: &AlgorithmTypeDS{
						Id:   types.Int64Value(int64(*getDnsSec.Results.DelegationSigner.AlgorithmType.Id)),
						Slug: types.StringValue(*getDnsSec.Results.DelegationSigner.AlgorithmType.Slug),
					},
					Digest: types.StringValue(*getDnsSec.Results.DelegationSigner.Digest),
					KeyTag: types.Int64Value(int64(*getDnsSec.Results.DelegationSigner.KeyTag)),
				},
			},
		}
		dnsSecState.ID = types.StringValue("Get DNSSEC")
		diags = resp.State.Set(ctx, &dnsSecState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		dnsSecState := &dnsSecDataSourceModel{
			SchemaVersion: types.Int64Value(int64(*getDnsSec.SchemaVersion)),
			ZoneId:        getZoneId,
			DnsSec: &dnsSecDSModel{
				IsEnabled: types.BoolValue(*getDnsSec.Results.IsEnabled),
				Status:    types.StringValue(*getDnsSec.Results.Status),
			},
		}
		dnsSecState.ID = types.StringValue("Get DNSSEC")
		diags = resp.State.Set(ctx, &dnsSecState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}
