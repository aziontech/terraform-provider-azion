package provider

import (
	"context"
	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strconv"
	"time"
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
	client *idns.APIClient
}

type zoneResourceModel struct {
	IDplan        types.String `tfsdk:"idplan"`
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	zone          zoneModel    `tfsdk:"zone"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

type zoneModel struct {
	Id          types.Int64  `tfsdk:"id"`
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
	resp.TypeName = req.ProviderTypeName + "_order"
}

func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"idplan": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"zone": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Name description of the DNS.",
					},
					"domain": schema.StringAttribute{
						Required:    true,
						Description: "Domain description of the DNS.",
					},
					"is_active": schema.BoolAttribute{
						Required:    true,
						Description: "Enable description of the DNS.",
					},
					"retry": schema.Int64Attribute{
						Optional: true,
					},
					"nxttl": schema.Int64Attribute{
						Optional: true,
					},
					"soattl": schema.Int64Attribute{
						Optional: true,
					},
					"refresh": schema.Int64Attribute{
						Optional: true,
					},
					"expiry": schema.Int64Attribute{
						Optional: true,
					},
					"nameservers": schema.ListAttribute{
						Optional:    true,
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

	r.client = req.ProviderData.(*idns.APIClient)
}

func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := idns.Zone{
		Name:     idns.PtrString(plan.zone.Name.ValueString()),
		Domain:   idns.PtrString(plan.zone.Domain.ValueString()),
		IsActive: idns.PtrBool(plan.zone.IsActive.ValueBool()),
	}

	createZone, _, err := r.client.ZonesApi.PostZone(ctx).Zone(zone).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating order",
			"Could not create order, unexpected error: "+err.Error(),
		)
		return
	}

	plan.IDplan = types.StringValue(strconv.Itoa(int(*createZone.Results[0].Id)))
	plan.SchemaVersion = types.Int64Value(int64(*createZone.SchemaVersion))
	for _, resultZone := range createZone.Results {
		plan.zone = zoneModel{
			Domain:   types.StringValue(*resultZone.Domain),
			IsActive: types.BoolValue(*resultZone.IsActive),
			Name:     types.StringValue(*resultZone.Name),
			Id:       types.Int64Value(int64(*resultZone.Id)),
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
	// Get current state
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed order value from HashiCups
	order, _, err := r.client.ZonesApi.GetZone(ctx, int32(state.zone.Id.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading HashiCups Order",
			"Could not read HashiCups order ID "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.zone = zoneModel{
		Domain:   types.StringValue(*order.Results.Domain),
		IsActive: types.BoolValue(*order.Results.IsActive),
		Name:     types.StringValue(*order.Results.Name),
		Id:       types.Int64Value(int64(*order.Results.Id)),
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *zoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := zoneModel{
		Id: plan.zone.Id,
	}

	createZone, _, err := r.client.ZonesApi.PutZone(ctx, int32(zone.Id.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating order",
			"Could not create order, unexpected error: "+err.Error(),
		)
		return
	}

	plan.IDplan = types.StringValue(strconv.Itoa(int(*createZone.Results[0].Id)))
	plan.SchemaVersion = types.Int64Value(int64(*createZone.SchemaVersion))
	for _, resultZone := range createZone.Results {
		plan.zone = zoneModel{
			Domain:   types.StringValue(*resultZone.Domain),
			IsActive: types.BoolValue(*resultZone.IsActive),
			Name:     types.StringValue(*resultZone.Name),
			Id:       types.Int64Value(int64(*resultZone.Id)),
		}
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *zoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed order value from HashiCups
	_, _, err := r.client.ZonesApi.DeleteZone(ctx, int32(state.zone.Id.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading HashiCups Order",
			"Could not read HashiCups order ID "+err.Error(),
		)
		return
	}
}

func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
