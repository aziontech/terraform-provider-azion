package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	edgeapi "github.com/aziontech/azionapi-v4-go-sdk-dev/edge-api"
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
	Data        *NetworkListResourceData `tfsdk:"data"`
	ID          types.String             `tfsdk:"id"`
	LastUpdated types.String             `tfsdk:"last_updated"`
}

type NetworkListResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	Items        types.List   `tfsdk:"items"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	Active       types.Bool   `tfsdk:"active"`
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
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed:    true,
						Description: "ID of the network list.",
					},
					"name": schema.StringAttribute{
						Description: "Name of the network list.",
						Required:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the network list.",
						Required:    true,
					},
					"items": schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of items in the network list.",
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the network list.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the network list.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the network list is active.",
						Computed:    true,
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

	// Extract items from the plan
	var items []string
	diags = plan.Data.Items.ElementsAs(ctx, &items, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListRequest := edgeapi.NetworkListDetailRequest{
		Name:  plan.Data.Name.ValueString(),
		Type:  plan.Data.Type.ValueString(),
		Items: items,
	}

	createNetworkListResponse, response, err := r.client.edgeApi.NetworkListsAPI.CreateNetworkList(ctx).NetworkListDetailRequest(networkListRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createNetworkListResponse, response, err = utils.RetryOn429(func() (*edgeapi.ResponseNetworkListDetail, *http.Response, error) {
				return r.client.edgeApi.NetworkListsAPI.CreateNetworkList(ctx).NetworkListDetailRequest(networkListRequest).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close()
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

	var responseItems []types.String
	for _, item := range createNetworkListResponse.Data.GetItems() {
		responseItems = append(responseItems, types.StringValue(item))
	}

	plan.Data = &NetworkListResourceData{
		ID:           types.Int64Value(createNetworkListResponse.Data.GetId()),
		Name:         types.StringValue(createNetworkListResponse.Data.GetName()),
		Type:         types.StringValue(createNetworkListResponse.Data.GetType()),
		Items:        utils.SliceStringTypeToList(responseItems),
		LastEditor:   types.StringValue(createNetworkListResponse.Data.GetLastEditor()),
		LastModified: types.StringValue(createNetworkListResponse.Data.GetLastModified().Format(time.RFC3339)),
		Active:       types.BoolValue(createNetworkListResponse.Data.GetActive()),
	}

	plan.ID = types.StringValue(strconv.FormatInt(createNetworkListResponse.Data.GetId(), 10))
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

	networkListId := state.ID.ValueString()

	getNetworkList, response, err := r.client.edgeApi.NetworkListsAPI.
		RetrieveNetworkList(ctx, networkListId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getNetworkList, response, err = utils.RetryOn429(func() (*edgeapi.ResponseRetrieveNetworkListDetail, *http.Response, error) {
				return r.client.edgeApi.NetworkListsAPI.RetrieveNetworkList(ctx, networkListId).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close()
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

	var responseItems []types.String
	if getNetworkList.Data.GetItems() != nil {
		for _, item := range getNetworkList.Data.GetItems() {
			responseItems = append(responseItems, types.StringValue(item))
		}
	}

	networkListData := &NetworkListResourceData{
		ID:           types.Int64Value(getNetworkList.Data.GetId()),
		Name:         types.StringValue(getNetworkList.Data.GetName()),
		Type:         types.StringValue(getNetworkList.Data.GetType()),
		Items:        utils.SliceStringTypeToList(responseItems),
		LastEditor:   types.StringValue(getNetworkList.Data.GetLastEditor()),
		LastModified: types.StringValue(getNetworkList.Data.GetLastModified().Format(time.RFC3339)),
		Active:       types.BoolValue(getNetworkList.Data.GetActive()),
	}

	state.Data = networkListData

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListId := state.ID.ValueString()

	// Extract items from the plan
	var items []string
	diags = plan.Data.Items.ElementsAs(ctx, &items, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListRequest := edgeapi.PatchedNetworkListDetailRequest{
		Name:  plan.Data.Name.ValueStringPointer(),
		Type:  plan.Data.Type.ValueStringPointer(),
		Items: items,
	}

	updateNetworkList, response, err := r.client.edgeApi.NetworkListsAPI.PartialUpdateNetworkList(ctx, networkListId).PatchedNetworkListDetailRequest(networkListRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateNetworkList, response, err = utils.RetryOn429(func() (*edgeapi.ResponseNetworkListDetail, *http.Response, error) {
				return r.client.edgeApi.NetworkListsAPI.PartialUpdateNetworkList(ctx, networkListId).PatchedNetworkListDetailRequest(networkListRequest).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close()
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

	var responseItems []types.String
	for _, item := range updateNetworkList.Data.GetItems() {
		responseItems = append(responseItems, types.StringValue(item))
	}

	plan.Data = &NetworkListResourceData{
		ID:           types.Int64Value(updateNetworkList.Data.GetId()),
		Name:         types.StringValue(updateNetworkList.Data.GetName()),
		Type:         types.StringValue(updateNetworkList.Data.GetType()),
		Items:        utils.SliceStringTypeToList(responseItems),
		LastEditor:   types.StringValue(updateNetworkList.Data.GetLastEditor()),
		LastModified: types.StringValue(updateNetworkList.Data.GetLastModified().Format(time.RFC3339)),
		Active:       types.BoolValue(updateNetworkList.Data.GetActive()),
	}

	plan.ID = types.StringValue(strconv.FormatInt(updateNetworkList.Data.GetId(), 10))
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

	networkListId := state.ID.ValueString()

	_, response, err := r.client.edgeApi.NetworkListsAPI.DestroyNetworkList(ctx, networkListId).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*edgeapi.ResponseDeleteNetworkListDetail, *http.Response, error) {
				return r.client.edgeApi.NetworkListsAPI.DestroyNetworkList(ctx, networkListId).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close()
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
}

func (r *networkListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
