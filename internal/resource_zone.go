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
	"io"
	"strconv"
	"terraform-provider-azion/internal/utils"
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
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the order.",
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
						Computed: true,
					},
					"nxttl": schema.Int64Attribute{
						Computed: true,
					},
					"soattl": schema.Int64Attribute{
						Computed: true,
					},
					"refresh": schema.Int64Attribute{
						Computed: true,
					},
					"expiry": schema.Int64Attribute{
						Computed: true,
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
		Name:     idns.PtrString(plan.Zone.Name.ValueString()),
		Domain:   idns.PtrString(plan.Zone.Domain.ValueString()),
		IsActive: idns.PtrBool(plan.Zone.IsActive.ValueBool()),
	}

	createZone, response, err := r.client.ZonesApi.PostZone(ctx).Zone(zone).Execute()
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
	idPlan, err := strconv.Atoi(state.ID.ValueString())
	order, response, err := r.client.ZonesApi.GetZone(ctx, int32(idPlan)).Execute()
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
	idPlan, err := strconv.Atoi(plan.ID.ValueString())
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

	updateZone, response, err := r.client.ZonesApi.PutZone(ctx, int32(idPlan)).Zone(zone).Execute()
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

	_, _, err := r.client.ZonesApi.DeleteZone(ctx, int32(state.Zone.ID.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Azion API",
			"Could not read azion API "+err.Error(),
		)
		return
	}
}

func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
