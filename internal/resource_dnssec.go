package provider

import (
	"context"
	"encoding/json"
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
	Dnssec        *dnssecModel `tfsdk:"dnssec"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

type dnssecModel struct {
	IsEnabled types.Bool `tfsdk:"is_enabled"`
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
			"dnssec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"is_enabled": schema.BoolAttribute{
						Required:    true,
						Description: "Zone DNSSEC flags for enabled.",
					},
				},
			},
		},
	}
}

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

	zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	dnssecReq := azionapi.NewDNSSECRequest(plan.Dnssec.IsEnabled.ValueBool())

	_, response, err := r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
	if err != nil {
		// Check if the error is due to JSON unmarshaling (unknown field) but HTTP request was successful
		if response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			// HTTP request was successful, proceed to parse response manually
		} else if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
				return r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
			}, 5) // Maximum 5 retries

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

	plan.Dnssec = &dnssecModel{
		IsEnabled: types.BoolValue(dnssecResp.Data.Enabled),
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *dnssecResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnssecResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	zoneId, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	_, response, err := r.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
	if err != nil {
		// Check if the error is due to JSON unmarshaling (unknown field) but HTTP request was successful
		if response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			// HTTP request was successful, proceed to parse response manually
		} else if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
				return r.client.api.DNSDNSSECAPI.RetrieveDnssec(ctx, zoneId).Execute()
			}, 5) // Maximum 5 retries

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

	state.Dnssec = &dnssecModel{
		IsEnabled: types.BoolValue(dnssecResp.Data.Enabled),
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

	zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	dnssecReq := azionapi.NewDNSSECRequest(plan.Dnssec.IsEnabled.ValueBool())

	_, response, err := r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
	if err != nil {
		// Check if the error is due to JSON unmarshaling (unknown field) but HTTP request was successful
		if response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			// HTTP request was successful, proceed to parse response manually
		} else if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
				return r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
			}, 5) // Maximum 5 retries

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

	plan.Dnssec = &dnssecModel{
		IsEnabled: types.BoolValue(dnssecResp.Data.Enabled),
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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

	zoneId, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	dnssecReq := azionapi.NewDNSSECRequest(false)

	_, response, err := r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
	if err != nil {
		// Check if the error is due to JSON unmarshaling (unknown field) but HTTP request was successful
		if response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			// HTTP request was successful, proceed (delete doesn't need response body)
		} else if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DNSSECResponse, *http.Response, error) {
				return r.client.api.DNSDNSSECAPI.UpdateDnssec(ctx, zoneId).DNSSECRequest(*dnssecReq).Execute()
			}, 5) // Maximum 5 retries

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
}

func (r *dnssecResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
