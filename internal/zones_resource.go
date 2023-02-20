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

func NewzoneResource() resource.Resource {
	return &zoneResource{}
}

type zoneResource struct {
	client *idns.APIClient
}

type zoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	zones       []zoneModel  `tfsdk:"zones"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type zoneModel struct {
	Name     types.String `tfsdk:"name"`
	Domain   types.String `tfsdk:"domain"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

// Metadata returns the data source type name.
func (r *zoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_order"
}

// Schema defines the schema for the resource.
func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"zones": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"domain": schema.StringAttribute{
							Required: true,
						},
						"is_active": schema.BoolAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *zoneResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*idns.APIClient)
}

// Create creates the resource and sets the initial Terraform state.
func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	zoneRequest := r.client.ZonesApi.PostZone(ctx).Zone()

	var items []*idns.Zone
	for _, item := range plan.zones {
		items = append(items, idns.Zone{
			Name:     string(item.Name.ValueString()),
			Domain:   item.Domain.ValueString(),
			IsActive: item.IsActive.ValueBool(),
		})
	}

	// Create new order
	order, err := r.client.CreateOrder(items)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating order",
			"Could not create order, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.Itoa(order.ID))
	for itemIndex, item := range order.Items {
		plan.Items[itemIndex] = orderItemModel{
			Coffee: orderItemCoffeeModel{
				ID:          types.Int64Value(int64(item.Coffee.ID)),
				Name:        types.StringValue(item.Coffee.Name),
				Teaser:      types.StringValue(item.Coffee.Teaser),
				Description: types.StringValue(item.Coffee.Description),
				Price:       types.Float64Value(item.Coffee.Price),
				Image:       types.StringValue(item.Coffee.Image),
			},
			Quantity: types.Int64Value(int64(item.Quantity)),
		}
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed order value from HashiCups
	order, err := r.client.GetOrder(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading HashiCups Order",
			"Could not read HashiCups order ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.Items = []orderItemModel{}
	for _, item := range order.Items {
		state.Items = append(state.Items, orderItemModel{
			Coffee: orderItemCoffeeModel{
				ID:          types.Int64Value(int64(item.Coffee.ID)),
				Name:        types.StringValue(item.Coffee.Name),
				Teaser:      types.StringValue(item.Coffee.Teaser),
				Description: types.StringValue(item.Coffee.Description),
				Price:       types.Float64Value(item.Coffee.Price),
				Image:       types.StringValue(item.Coffee.Image),
			},
			Quantity: types.Int64Value(int64(item.Quantity)),
		})
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
	// Retrieve values from plan
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var hashicupsItems []hashicups.OrderItem
	for _, item := range plan.Items {
		hashicupsItems = append(hashicupsItems, hashicups.OrderItem{
			Coffee: hashicups.Coffee{
				ID: int(item.Coffee.ID.ValueInt64()),
			},
			Quantity: int(item.Quantity.ValueInt64()),
		})
	}

	// Update existing order
	_, err := r.client.UpdateOrder(plan.ID.ValueString(), hashicupsItems)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating HashiCups Order",
			"Could not update order, unexpected error: "+err.Error(),
		)
		return
	}

	// Fetch updated items from GetOrder as UpdateOrder items are not
	// populated.
	order, err := r.client.GetOrder(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading HashiCups Order",
			"Could not read HashiCups order ID "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update resource state with updated items and timestamp
	plan.Items = []orderItemModel{}
	for _, item := range order.Items {
		plan.Items = append(plan.Items, orderItemModel{
			Coffee: orderItemCoffeeModel{
				ID:          types.Int64Value(int64(item.Coffee.ID)),
				Name:        types.StringValue(item.Coffee.Name),
				Teaser:      types.StringValue(item.Coffee.Teaser),
				Description: types.StringValue(item.Coffee.Description),
				Price:       types.Float64Value(item.Coffee.Price),
				Image:       types.StringValue(item.Coffee.Image),
			},
			Quantity: types.Int64Value(int64(item.Quantity)),
		})
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
	// Retrieve values from state
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	err := r.client.DeleteOrder(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting HashiCups Order",
			"Could not delete order, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
