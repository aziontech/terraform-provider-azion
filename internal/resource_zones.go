package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

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
	_ resource.Resource                = &zoneResource{}
	_ resource.ResourceWithConfigure   = &zoneResource{}
	_ resource.ResourceWithImportState = &zoneResource{}
)

func NewZoneResource() resource.Resource {
	return &zoneResource{}
}

type zoneResource struct {
	client *apiClient
}

type zoneResourceModel struct {
	ID          types.String      `tfsdk:"id"`
	Data        *zoneResourceData `tfsdk:"data"`
	LastUpdated types.String      `tfsdk:"last_updated"`
}

type zoneResourceData struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Domain         types.String `tfsdk:"domain"`
	IsActive       types.Bool   `tfsdk:"is_active"`
	Nameservers    types.List   `tfsdk:"nameservers"`
	ProductVersion types.String `tfsdk:"product_version"`
}

func (r *zoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_zone"
}

func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the zone. Must provide only one of zone_id, name.",
					},
					"domain": schema.StringAttribute{
						Required:    true,
						Description: "Domain name attributed by Azion to this configuration.",
					},
					"is_active": schema.BoolAttribute{
						Required:    true,
						Description: "Status of the zone.",
					},
					"nameservers": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"product_version": schema.StringAttribute{
						Computed:    true,
						Description: "Product version of the zone.",
					},
				},
			},
		},
	}
}

func (r *zoneResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*apiClient)
}

func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := dnsapi.ZoneRequest{
		Name:   plan.Data.Name.ValueString(),
		Domain: plan.Data.Domain.ValueString(),
		Active: plan.Data.IsActive.ValueBool(),
	}

	createZone, response, err := r.client.idnsApi.DNSZonesAPI.CreateDnsZone(ctx).ZoneRequest(zone).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			createZone, response, err = utils.RetryOn429(func() (*dnsapi.ResponseZone, *http.Response, error) {
				return r.client.idnsApi.DNSZonesAPI.CreateDnsZone(ctx).ZoneRequest(zone).Execute() //nolint
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
					"No response from API",
				)
			}
			return
		}
	}

	plan.ID = types.StringValue(strconv.FormatInt(createZone.Data.Id, 10))
	zoneData := createZone.Data

	var slice []types.String
	for _, ns := range zoneData.Nameservers {
		slice = append(slice, types.StringValue(ns))
	}

	plan.Data = &zoneResourceData{
		ID:             types.Int64Value(zoneData.Id),
		Name:           types.StringValue(zoneData.Name),
		Domain:         types.StringValue(zoneData.Domain),
		IsActive:       types.BoolValue(zoneData.Active),
		Nameservers:    utils.SliceStringTypeToList(slice),
		ProductVersion: types.StringValue(zoneData.ProductVersion),
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := state.ID.ValueString()

	order, response, err := r.client.idnsApi.DNSZonesAPI.RetrieveDnsZone(ctx, zoneID).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			order, response, err = utils.RetryOn429(func() (*dnsapi.ResponseRetrieveZone, *http.Response, error) {
				return r.client.idnsApi.DNSZonesAPI.RetrieveDnsZone(ctx, zoneID).Execute() //nolint
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

	var slice []types.String
	for _, Nameservers := range order.Data.Nameservers {
		slice = append(slice, types.StringValue(Nameservers))
	}
	state.Data = &zoneResourceData{
		ID:             types.Int64Value(order.Data.Id),
		Name:           types.StringValue(order.Data.Name),
		Domain:         types.StringValue(order.Data.Domain),
		IsActive:       types.BoolValue(order.Data.Active),
		Nameservers:    utils.SliceStringTypeToList(slice),
		ProductVersion: types.StringValue(order.Data.ProductVersion),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *zoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	zoneID := plan.ID.ValueString()

	zoneReq := dnsapi.PatchedUpdateZoneRequest{
		Name:   plan.Data.Name.ValueStringPointer(),
		Active: plan.Data.IsActive.ValueBoolPointer(),
	}

	updateZone, response, err := r.client.idnsApi.DNSZonesAPI.
		PartialUpdateDnsZone(ctx, zoneID).PatchedUpdateZoneRequest(zoneReq).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateZone, response, err = utils.RetryOn429(func() (*dnsapi.ResponseZone, *http.Response, error) {
				return r.client.idnsApi.DNSZonesAPI.PartialUpdateDnsZone(ctx, zoneID).PatchedUpdateZoneRequest(zoneReq).Execute() //nolint
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

	plan.ID = types.StringValue(strconv.FormatInt(updateZone.Data.Id, 10))
	zoneData := updateZone.Data

	var slice []types.String
	for _, ns := range zoneData.Nameservers {
		slice = append(slice, types.StringValue(ns))
	}

	plan.Data = &zoneResourceData{
		ID:             types.Int64Value(zoneData.Id),
		Name:           types.StringValue(zoneData.Name),
		Domain:         types.StringValue(zoneData.Domain),
		IsActive:       types.BoolValue(zoneData.Active),
		Nameservers:    utils.SliceStringTypeToList(slice),
		ProductVersion: types.StringValue(zoneData.ProductVersion),
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *zoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := state.ID.ValueString()

	_, response, err := r.client.idnsApi.DNSZonesAPI.DeleteDnsZone(ctx, zoneID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*dnsapi.ResponseDeleteZone, *http.Response, error) {
				return r.client.idnsApi.DNSZonesAPI.DeleteDnsZone(ctx, zoneID).Execute() //nolint
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
			resp.Diagnostics.AddError(
				"Error Reading Azion API",
				"Could not read azion API "+err.Error(),
			)
			return
		}
	}
}

func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
