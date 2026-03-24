package provider

import (
	"context"
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
	ID           types.Int64  `tfsdk:"id"`
	LastEditor   types.String `tfsdk:"last_editor"`
	LastModified types.String `tfsdk:"last_modified"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	Items        types.Set    `tfsdk:"items"`
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
					"type": schema.StringAttribute{
						Description: "Type of the network list. Can be: asn, countries, or ip_cidr.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the network list.",
						Required:    true,
					},
					"items": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of items in the network list. Contents depend on the type: country codes, IP addresses, or ASN numbers.",
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

	var items []string
	diagsItems := plan.NetworkList.Items.ElementsAs(ctx, &items, false)
	resp.Diagnostics.Append(diagsItems...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListRequest := azionapi.NetworkListRequest{
		Name:  plan.NetworkList.Name.ValueString(),
		Type:  plan.NetworkList.Type.ValueString(),
		Items: items,
	}

	createNetworkListResponse, response, err := r.client.api.NetworkListsAPI.CreateNetworkList(ctx).NetworkListRequest(networkListRequest).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			createNetworkListResponse, response, err = utils.RetryOn429(func() (*azionapi.NetworkListResponse, *http.Response, error) {
				return r.client.api.NetworkListsAPI.CreateNetworkList(ctx).NetworkListRequest(networkListRequest).Execute() //nolint
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
			if response != nil && response.Body != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(
						errReadAll.Error(),
						"error reading response from API",
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
					"API request failed",
				)
			}
			return
		}
	}

	plan.SchemaVersion = types.Int64Value(1)

	data := createNetworkListResponse.GetData()
	var sliceString []types.String
	for _, item := range data.GetItems() {
		sliceString = append(sliceString, types.StringValue(item))
	}

	plan.NetworkList = &NetworkListResourceResults{
		ID:           types.Int64Value(data.GetId()),
		LastEditor:   types.StringValue(data.GetLastEditor()),
		LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
		Type:         types.StringValue(data.GetType()),
		Name:         types.StringValue(data.GetName()),
		Items:        utils.SliceStringTypeToSet(sliceString),
	}

	plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
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

	var networkListId int64
	if state.ID.IsNull() {
		networkListId = state.NetworkList.ID.ValueInt64()
	} else {
		id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid ID format",
				err.Error(),
			)
			return
		}
		networkListId = id
	}

	getNetworkList, response, err := r.client.api.NetworkListsAPI.RetrieveNetworkList(ctx, networkListId).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == 429 {
			getNetworkList, response, err = utils.RetryOn429(func() (*azionapi.NetworkListResponse, *http.Response, error) {
				return r.client.api.NetworkListsAPI.RetrieveNetworkList(ctx, networkListId).Execute() //nolint
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
			if response != nil && response.Body != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(
						errReadAll.Error(),
						"error reading response from API",
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
					"API request failed",
				)
			}
			return
		}
	}

	data := getNetworkList.GetData()
	var sliceString []types.String
	for _, item := range data.GetItems() {
		sliceString = append(sliceString, types.StringValue(item))
	}

	networkListState := NetworkListResourceModel{
		SchemaVersion: types.Int64Value(1),
		NetworkList: &NetworkListResourceResults{
			ID:           types.Int64Value(data.GetId()),
			LastEditor:   types.StringValue(data.GetLastEditor()),
			LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
			Type:         types.StringValue(data.GetType()),
			Name:         types.StringValue(data.GetName()),
			Items:        utils.SliceStringTypeToSet(sliceString),
		},
		ID: types.StringValue(strconv.FormatInt(data.GetId(), 10)),
	}

	diags = resp.State.Set(ctx, &networkListState)
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
	diagsNetworkList := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsNetworkList...)
	if resp.Diagnostics.HasError() {
		return
	}

	var networkListId int64
	if state.ID.IsNull() {
		networkListId = state.NetworkList.ID.ValueInt64()
	} else {
		id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid ID format",
				err.Error(),
			)
			return
		}
		networkListId = id
	}

	var items []string
	diagsItems := plan.NetworkList.Items.ElementsAs(ctx, &items, false)
	resp.Diagnostics.Append(diagsItems...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkListRequest := azionapi.NetworkListRequest{
		Name:  plan.NetworkList.Name.ValueString(),
		Type:  plan.NetworkList.Type.ValueString(),
		Items: items,
	}

	updateNetworkList, response, err := r.client.api.NetworkListsAPI.UpdateNetworkList(ctx, networkListId).NetworkListRequest(networkListRequest).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			updateNetworkList, response, err = utils.RetryOn429(func() (*azionapi.NetworkListResponse, *http.Response, error) {
				return r.client.api.NetworkListsAPI.UpdateNetworkList(ctx, networkListId).NetworkListRequest(networkListRequest).Execute() //nolint
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
			if response != nil && response.Body != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(
						errReadAll.Error(),
						"error reading response from API",
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
					"API request failed",
				)
			}
			return
		}
	}

	plan.SchemaVersion = types.Int64Value(1)

	data := updateNetworkList.GetData()
	var sliceString []types.String
	for _, item := range data.GetItems() {
		sliceString = append(sliceString, types.StringValue(item))
	}

	plan.NetworkList = &NetworkListResourceResults{
		ID:           types.Int64Value(data.GetId()),
		LastEditor:   types.StringValue(data.GetLastEditor()),
		LastModified: types.StringValue(data.GetLastModified().Format(time.RFC3339)),
		Type:         types.StringValue(data.GetType()),
		Name:         types.StringValue(data.GetName()),
		Items:        utils.SliceStringTypeToSet(sliceString),
	}

	plan.ID = types.StringValue(strconv.FormatInt(data.GetId(), 10))
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

	var networkListId int64
	if state.ID.IsNull() {
		networkListId = state.NetworkList.ID.ValueInt64()
	} else {
		id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid ID format",
				err.Error(),
			)
			return
		}
		networkListId = id
	}

	_, response, err := r.client.api.NetworkListsAPI.DeleteNetworkList(ctx, networkListId).Execute() //nolint
	if err != nil {
		if response != nil && response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.NetworkListsAPI.DeleteNetworkList(ctx, networkListId).Execute() //nolint
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
			if response != nil && response.Body != nil {
				bodyBytes, errReadAll := io.ReadAll(response.Body)
				if errReadAll != nil {
					resp.Diagnostics.AddError(
						errReadAll.Error(),
						"error reading response from API",
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
					"API request failed",
				)
			}
			return
		}
	}
}

func (r *networkListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
