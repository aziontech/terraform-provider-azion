package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dnssecDataSource{}
	_ datasource.DataSourceWithConfigure = &dnssecDataSource{}
)

func dataSourceAzionDNSSec() datasource.DataSource {
	return &dnssecDataSource{}
}

type dnssecDataSource struct {
	client *apiClient
}

type dnssecDataSourceModel struct {
	ID               types.String             `tfsdk:"id"`
	ZoneId           types.Int64              `tfsdk:"zone_id"`
	SchemaVersion    types.Int64              `tfsdk:"schema_version"`
	Dnssec           *dnssecDSModel           `tfsdk:"dnssec"`
	DelegationSigner *DelegationSignerDSModel `tfsdk:"delegation_signer"`
}

type dnssecDSModel struct {
	IsEnabled types.Bool   `tfsdk:"is_enabled"`
	Status    types.String `tfsdk:"status"`
}

type DelegationSignerDSModel struct {
	AlgorithmType *AlgTypeDS   `tfsdk:"algorithm_type"`
	Digest        types.String `tfsdk:"digest"`
	DigestType    *AlgTypeDS   `tfsdk:"digest_type"`
	KeyTag        types.Int64  `tfsdk:"key_tag"`
}

type AlgTypeDS struct {
	Id   types.Int64  `tfsdk:"id"`
	Slug types.String `tfsdk:"slug"`
}

// dnssecResponse is a custom response type to handle the API's additional "state" field
// that's not present in the SDK's DNSSECResponse model.
type dnssecResponse struct {
	State string     `json:"state"`
	Data  dnssecData `json:"data"`
}

type dnssecData struct {
	Enabled          bool                  `json:"enabled"`
	Status           string                `json:"status"`
	DelegationSigner *delegationSignerData `json:"delegation_signer,omitempty"`
}

type delegationSignerData struct {
	AlgorithmType algTypeData `json:"algorithm_type"`
	Digest        string      `json:"digest"`
	DigestType    algTypeData `json:"digest_type"`
	KeyTag        int64       `json:"key_tag"`
}

type algTypeData struct {
	Id   int64  `json:"id"`
	Slug string `json:"slug"`
}

func (d *dnssecDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *dnssecDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_dnssec"
}

func (d *dnssecDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"dnssec": schema.SingleNestedAttribute{
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
				},
			},
			"delegation_signer": schema.SingleNestedAttribute{
				Description: "Zone DNSSEC delegation signer.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"algorithm_type": schema.SingleNestedAttribute{
						Description: "Algorithm type for Zone DNSSEC.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"id": schema.Int64Attribute{
								Description: "The ID of this algorithm type.",
								Optional:    true,
							},
							"slug": schema.StringAttribute{
								Description: "The slug of this algorithm type.",
								Optional:    true,
							},
						},
					},
					"digest": schema.StringAttribute{
						Optional:    true,
						Description: "Zone DNSSEC digest.",
					},
					"digest_type": schema.SingleNestedAttribute{
						Description: "Digest type for Zone DNSSEC.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"id": schema.Int64Attribute{
								Description: "The ID of this digest type.",
								Optional:    true,
							},
							"slug": schema.StringAttribute{
								Description: "The slug of this digest type.",
								Optional:    true,
							},
						},
					},
					"key_tag": schema.Int64Attribute{
						Optional:    true,
						Description: "Key Tag for the Zone DNSSEC.",
					},
				},
			},
		},
	}
}

func (d *dnssecDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getZoneId types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("zone_id"), &getZoneId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneId := getZoneId.ValueInt64()

	_, response, err := d.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
	if err != nil {
		// Check if the error is due to JSON unmarshaling (unknown field) but HTTP request was successful
		if response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			// HTTP request was successful, proceed to parse response manually
		} else if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
				return d.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				// Check again if it's a successful response after retry
				if response == nil || response.StatusCode < 200 || response.StatusCode >= 300 {
					resp.Diagnostics.AddError(
						err.Error(),
						"API request failed after too many retries",
					)
					return
				}
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

	// Parse response manually to handle the "state" field
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"Failed to read response body",
		)
		return
	}

	var dnssecResp dnssecResponse
	if err := json.Unmarshal(bodyBytes, &dnssecResp); err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"Failed to parse response JSON",
		)
		return
	}

	dnssecState := &dnssecDataSourceModel{
		ZoneId: getZoneId,
		ID:     types.StringValue("Get DNSSEC"),
		Dnssec: &dnssecDSModel{
			IsEnabled: types.BoolValue(dnssecResp.Data.Enabled),
			Status:    types.StringValue(dnssecResp.Data.Status),
		},
	}

	if dnssecResp.Data.DelegationSigner != nil {
		dnssecState.DelegationSigner = &DelegationSignerDSModel{
			AlgorithmType: &AlgTypeDS{
				Id:   types.Int64Value(dnssecResp.Data.DelegationSigner.AlgorithmType.Id),
				Slug: types.StringValue(dnssecResp.Data.DelegationSigner.AlgorithmType.Slug),
			},
			Digest: types.StringValue(dnssecResp.Data.DelegationSigner.Digest),
			DigestType: &AlgTypeDS{
				Id:   types.Int64Value(dnssecResp.Data.DelegationSigner.DigestType.Id),
				Slug: types.StringValue(dnssecResp.Data.DelegationSigner.DigestType.Slug),
			},
			KeyTag: types.Int64Value(dnssecResp.Data.DelegationSigner.KeyTag),
		}
	}

	diags = resp.State.Set(ctx, &dnssecState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
