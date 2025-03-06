package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aziontech/terraform-provider-azion/internal/utils"

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
	ID            types.String `tfsdk:"id"`
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	Zone          *zoneModel   `tfsdk:"zone"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

type zoneModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Domain      types.String `tfsdk:"domain"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	Retry       types.Int64  `tfsdk:"retry"`
	NxTtl       types.Int64  `tfsdk:"nxttl"`
	SoaTtl      types.Int64  `tfsdk:"soattl"`
	Refresh     types.Int64  `tfsdk:"refresh"`
	Expiry      types.Int64  `tfsdk:"expiry"`
	Nameservers types.List   `tfsdk:"nameservers"`
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
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
			},
			"zone": schema.SingleNestedAttribute{
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
					"retry": schema.Int64Attribute{
						Computed:    true,
						Description: "The rate at which a secondary server will retry to refresh the primary zone file if the initial refresh failed.",
					},
					"nxttl": schema.Int64Attribute{
						Computed:    true,
						Description: "In the event that requesting the domain results in a non-existent query (NXDOMAIN), this is the amount of time that is respected by the recursor to return the NXDOMAIN response.",
					},
					"soattl": schema.Int64Attribute{
						Computed:    true,
						Description: "The interval at which the SOA record itself is refreshed.",
					},
					"refresh": schema.Int64Attribute{
						Computed:    true,
						Description: "The interval at which secondary servers (secondary DNS) are set to refresh the primary zone file from the primary server.",
					},
					"expiry": schema.Int64Attribute{
						Computed:    true,
						Description: "If Refresh and Retry fail repeatedly, this is the time period after which the primary should be considered gone and no longer authoritative for the given zone.",
					},
					"nameservers": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
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

	zone := idns.Zone{
		Name:     idns.PtrString(plan.Zone.Name.ValueString()),
		Domain:   idns.PtrString(plan.Zone.Domain.ValueString()),
		IsActive: idns.PtrBool(plan.Zone.IsActive.ValueBool()),
	}

	createZone, response, err := r.client.idnsApi.ZonesAPI.PostZone(ctx).Zone(zone).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createZone, response, err = utils.RetryOn429(func() (*idns.PostOrPutZoneResponse, *http.Response, error) {
				return r.client.idnsApi.ZonesAPI.PostZone(ctx).Zone(zone).Execute() //nolint
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

	plan.ID = types.StringValue(strconv.Itoa(int(*createZone.Results[0].Id)))
	plan.SchemaVersion = types.Int64Value(int64(*createZone.SchemaVersion))
	for _, resultZone := range createZone.Results {
		var slice []types.String
		for _, Nameservers := range resultZone.Nameservers {
			slice = append(slice, types.StringValue(Nameservers))
		}
		plan.Zone = &zoneModel{
			ID:          types.Int64Value(int64(*resultZone.Id)),
			Name:        types.StringValue(*resultZone.Name),
			Domain:      types.StringValue(*resultZone.Domain),
			IsActive:    types.BoolValue(*resultZone.IsActive),
			NxTtl:       types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.NxTtl))),
			Retry:       types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.Retry))),
			Refresh:     types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.Refresh))),
			Expiry:      types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.Expiry))),
			SoaTtl:      types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.SoaTtl))),
			Nameservers: utils.SliceStringTypeToList(slice),
		}
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

	idPlan, err := strconv.ParseInt(state.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	order, response, err := r.client.idnsApi.ZonesAPI.GetZone(ctx, int32(idPlan)).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			order, response, err = utils.RetryOn429(func() (*idns.GetZoneResponse, *http.Response, error) {
				return r.client.idnsApi.ZonesAPI.GetZone(ctx, int32(idPlan)).Execute() //nolint
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
	for _, Nameservers := range order.Results.Nameservers {
		slice = append(slice, types.StringValue(Nameservers))
	}
	state.Zone = &zoneModel{
		ID:          types.Int64Value(int64(*order.Results.Id)),
		Name:        types.StringValue(*order.Results.Name),
		Domain:      types.StringValue(*order.Results.Domain),
		IsActive:    types.BoolValue(*order.Results.IsActive),
		NxTtl:       types.Int64Value(int64(*idns.NullableInt32.Get(order.Results.NxTtl))),
		Retry:       types.Int64Value(int64(*idns.NullableInt32.Get(order.Results.Retry))),
		Refresh:     types.Int64Value(int64(*idns.NullableInt32.Get(order.Results.Refresh))),
		Expiry:      types.Int64Value(int64(*idns.NullableInt32.Get(order.Results.Expiry))),
		SoaTtl:      types.Int64Value(int64(*idns.NullableInt32.Get(order.Results.SoaTtl))),
		Nameservers: utils.SliceStringTypeToList(slice),
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
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idPlan, err := strconv.ParseInt(plan.ID.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}
	zone := idns.Zone{
		Name:     idns.PtrString(plan.Zone.Name.ValueString()),
		Domain:   idns.PtrString(plan.Zone.Domain.ValueString()),
		IsActive: idns.PtrBool(plan.Zone.IsActive.ValueBool()),
	}

	updateZone, response, err := r.client.idnsApi.ZonesAPI.
		PutZone(ctx, int32(idPlan)).Zone(zone).Execute() //nolint #nosec G701
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*idns.PostOrPutZoneResponse, *http.Response, error) {
				return r.client.idnsApi.ZonesAPI.PutZone(ctx, int32(idPlan)).Zone(zone).Execute() //nolint #nosec G701
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

	plan.ID = types.StringValue(strconv.Itoa(int(*updateZone.Results[0].Id)))
	plan.SchemaVersion = types.Int64Value(int64(*updateZone.SchemaVersion))
	for _, resultZone := range updateZone.Results {
		var slice []types.String
		for _, Nameservers := range resultZone.Nameservers {
			slice = append(slice, types.StringValue(Nameservers))
		}
		plan.Zone = &zoneModel{
			ID:          types.Int64Value(int64(*resultZone.Id)),
			Name:        types.StringValue(*resultZone.Name),
			Domain:      types.StringValue(*resultZone.Domain),
			IsActive:    types.BoolValue(*resultZone.IsActive),
			NxTtl:       types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.NxTtl))),
			Retry:       types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.Retry))),
			Refresh:     types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.Refresh))),
			Expiry:      types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.Expiry))),
			SoaTtl:      types.Int64Value(int64(*idns.NullableInt32.Get(resultZone.SoaTtl))),
			Nameservers: utils.SliceStringTypeToList(slice),
		}
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

	zoneID, err := utils.CheckInt64toInt32Security(state.Zone.ID.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, zoneID)
		return
	}

	_, response, err := r.client.idnsApi.ZonesAPI.DeleteZone(ctx, zoneID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (string, *http.Response, error) {
				return r.client.idnsApi.ZonesAPI.DeleteZone(ctx, zoneID).Execute() //nolint
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
