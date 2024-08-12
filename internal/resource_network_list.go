package provider

import (
	"context"
	"github.com/aziontech/azionapi-go-sdk/networklist"
	"io"
	"strconv"
	"time"

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
	_ resource.Resource                = &networkListResource{}
	_ resource.ResourceWithConfigure   = &networkListResource{}
	_ resource.ResourceWithImportState = &networkListResource{}
)

func NetworkListResource() resource.Resource {
	return &networkListResource{}
}

type networkListResource struct {
	client *apiClient
}

type NetworkListResourceModel struct {
	SchemaVersion types.Int64                 `tfsdk:"schema_version"`
	NetworkList   *NetworkListResourceResults `tfsdk:"results"`
	ID            types.String                `tfsdk:"id"`
	LastUpdated   types.String                `tfsdk:"last_updated"`
}

type NetworkListResourceResults struct {
	ID             types.Int64  `tfsdk:"id"`
	LastEditor     types.String `tfsdk:"last_editor"`
	LastModified   types.String `tfsdk:"last_modified"`
	ListType       types.String `tfsdk:"list_type"`
	Name           types.String `tfsdk:"name"`
	ItemsValuesStr types.Set    `tfsdk:"items_values_str"`
	ItemsValuesInt types.List   `tfsdk:"items_values_int"`
}

func (r *networkListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_list"
}

func (r *networkListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed:    true,
						Description: "Identification of this entry.",
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the network list.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the network list.",
						Computed:    true,
					},
					"list_type": schema.StringAttribute{
						Description: "Type of the network list.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the network list.",
						Required:    true,
					},
					"items_values_str": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of countries in the network list.",
					},
					"items_values_int": schema.ListAttribute{
						Optional:    true,
						ElementType: types.Int64Type,
						Description: "List of countries in the network list.",
					},
				},
			},
		},
	}
}

func (r *networkListResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *networkListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsInt := req.Plan.SetAttribute(ctx, path.Root("results").AtName("items_values_int"), types.ListNull(types.Int64Type))
	resp.Diagnostics.Append(diagsInt...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListRequest := networklist.CreateNetworkListsRequest{
		Name:     plan.NetworkList.Name.ValueStringPointer(),
		ListType: plan.NetworkList.ListType.ValueStringPointer(),
	}

	requestItemsValue := plan.NetworkList.ItemsValuesStr.ElementsAs(ctx, &networkListRequest.ItemsValues, false)
	resp.Diagnostics.Append(requestItemsValue...)
	if resp.Diagnostics.HasError() {
		return
	}

	createNetworkListResponse, response, err := r.client.networkListApi.DefaultAPI.NetworkListsPost(ctx).CreateNetworkListsRequest(networkListRequest).Execute()
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
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
	plan.SchemaVersion = types.Int64Value(3)
	var sliceString []types.String
	for _, itemsValuesStr := range createNetworkListResponse.Results.GetItemsValues() {
		sliceString = append(sliceString, types.StringValue(itemsValuesStr))
	}
	plan.NetworkList = &NetworkListResourceResults{
		ID:             types.Int64Value(createNetworkListResponse.Results.GetId()),
		LastEditor:     types.StringValue(createNetworkListResponse.Results.GetLastEditor()),
		LastModified:   types.StringValue(createNetworkListResponse.Results.GetLastModified()),
		ListType:       types.StringValue(createNetworkListResponse.Results.GetListType()),
		Name:           types.StringValue(createNetworkListResponse.Results.GetName()),
		ItemsValuesStr: utils.SliceStringTypeToSet(sliceString),
		ItemsValuesInt: types.ListNull(types.Int64Type),
	}

	plan.ID = types.StringValue(strconv.FormatInt(createNetworkListResponse.Results.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *networkListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var networkListId string
	if state.ID.IsNull() {
		networkListId = strconv.Itoa(int(state.NetworkList.ID.ValueInt64()))
	} else {
		networkListId = state.ID.ValueString()
	}

	getNetworkList, response, err := r.client.networkListApi.DefaultAPI.NetworkListsUuidGet(ctx, networkListId).Execute()
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
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

	var sliceString []types.String
	for _, itemsValuesStr := range getNetworkList.GetResults().NetworkListUuidResponseEntryString.GetItemsValues() {
		sliceString = append(sliceString, types.StringValue(itemsValuesStr))
	}
	var sliceInt []types.Int64
	for _, itemsValuesInt := range getNetworkList.GetResults().NetworkListUuidResponseEntryInt.GetItemsValues() {
		sliceInt = append(sliceInt, types.Int64Value(int64(itemsValuesInt)))
	}

	if len(sliceString) != 0 {
		networkListsState := NetworkListResourceModel{
			SchemaVersion: types.Int64Value(getNetworkList.GetSchemaVersion()),
			NetworkList: &NetworkListResourceResults{
				LastEditor:     types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryString.GetLastEditor()),
				LastModified:   types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryString.GetLastModified()),
				ListType:       types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryString.GetListType()),
				Name:           types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryString.GetName()),
				ItemsValuesStr: utils.SliceStringTypeToSet(sliceString),
				ItemsValuesInt: types.ListNull(types.Int64Type),
			},
			ID: types.StringValue(networkListId),
		}
		diags = resp.State.Set(ctx, &networkListsState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		networkListsState := NetworkListResourceModel{
			SchemaVersion: types.Int64Value(getNetworkList.GetSchemaVersion()),
			NetworkList: &NetworkListResourceResults{
				LastEditor:     types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryInt.GetLastEditor()),
				LastModified:   types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryInt.GetLastModified()),
				ListType:       types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryInt.GetListType()),
				Name:           types.StringValue(getNetworkList.GetResults().NetworkListUuidResponseEntryInt.GetName()),
				ItemsValuesStr: types.SetValueMust(types.StringType, nil),
				ItemsValuesInt: utils.SliceIntTypeToList(sliceInt),
			},
			ID: types.StringValue(networkListId),
		}
		diags = resp.State.Set(ctx, &networkListsState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *networkListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NetworkListResourceModel
	diagsNetworkList := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsNetworkList...)
	if resp.Diagnostics.HasError() {
		return
	}

	var networkListId string
	if state.ID.IsNull() {
		networkListId = strconv.Itoa(int(state.NetworkList.ID.ValueInt64()))
	} else {
		networkListId = state.ID.ValueString()
	}

	networkListRequest := networklist.CreateNetworkListsRequest{
		Name:     plan.NetworkList.Name.ValueStringPointer(),
		ListType: plan.NetworkList.ListType.ValueStringPointer(),
	}

	requestItemsValue := plan.NetworkList.ItemsValuesStr.ElementsAs(ctx, &networkListRequest.ItemsValues, false)
	resp.Diagnostics.Append(requestItemsValue...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateNetworkList, response, err := r.client.networkListApi.DefaultAPI.NetworkListsUuidPut(ctx, networkListId).CreateNetworkListsRequest(networkListRequest).Execute()
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
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

	plan.SchemaVersion = types.Int64Value(3)
	var sliceString []types.String
	for _, itemsValuesStr := range updateNetworkList.Results.GetItemsValues() {
		sliceString = append(sliceString, types.StringValue(itemsValuesStr))
	}
	plan.NetworkList = &NetworkListResourceResults{
		ID:             types.Int64Value(updateNetworkList.Results.GetId()),
		LastEditor:     types.StringValue(updateNetworkList.Results.GetLastEditor()),
		LastModified:   types.StringValue(updateNetworkList.Results.GetLastModified()),
		ListType:       types.StringValue(updateNetworkList.Results.GetListType()),
		Name:           types.StringValue(updateNetworkList.Results.GetName()),
		ItemsValuesStr: utils.SliceStringTypeToSet(sliceString),
		ItemsValuesInt: types.ListNull(types.Int64Type),
	}

	plan.ID = types.StringValue(strconv.FormatInt(updateNetworkList.Results.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *networkListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var networkListId string
	if state.ID.IsNull() {
		networkListId = strconv.Itoa(int(state.NetworkList.ID.ValueInt64()))
	} else {
		networkListId = state.ID.ValueString()
	}

	response, err := r.client.networkListApi.DefaultAPI.NetworkListsUuidDelete(ctx, networkListId).Execute()
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
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

func (r *networkListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
