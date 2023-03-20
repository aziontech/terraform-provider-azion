package provider

import (
	"context"
	"fmt"
	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
)

var (
	_ datasource.DataSource              = &dnsSecDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsSecDataSource{}
)

func dataSourceAzionDNSSec() datasource.DataSource {
	return &dnsSecDataSource{}
}

type dnsSecDataSource struct {
	client *idns.APIClient
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
	d.client = req.ProviderData.(*idns.APIClient)
}

func (d *dnsSecDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dnssec"
}

func (d *dnsSecDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
			},
			"zone_id": schema.Int64Attribute{
				Optional: true,
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
						Description: "Enable description of the DNS.",
					},
					"status": schema.StringAttribute{
						Optional:    true,
						Description: "Domain description of the DNS.",
					},
					"delegation_signer": schema.SingleNestedAttribute{
						Optional:   true,
						Attributes: DnsDelegationSignerDS(),
					},
				},
			},
		},
	}

}
func DnsDelegationSignerDS() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"digesttype": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: digesttypeDS(),
		},
		"algorithmtype": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: algorithmtypeDS(),
		},
		"digest": schema.StringAttribute{
			Optional:    true,
			Description: "Domain description of the DNS.",
		},
		"keytag": schema.Int64Attribute{
			Optional:    true,
			Description: "Domain description of the DNS.",
		},
	}
}
func digesttypeDS() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed: true,
		},
		"slug": schema.StringAttribute{
			Computed: true,
		},
	}
}
func algorithmtypeDS() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed: true,
		},
		"slug": schema.StringAttribute{
			Computed: true,
		},
	}
}

func (d *dnsSecDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, fmt.Sprintf("Reading DNSSEC"))

	var getZoneId types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("zone_id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneId := int32(getZoneId.ValueInt64())

	getDnsSec, response, err := d.client.DNSSECApi.GetZoneDnsSec(ctx, zoneId).Execute()
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
	dnsSecEnabled := *getDnsSec.Results.IsEnabled
	if dnsSecEnabled != false {
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
