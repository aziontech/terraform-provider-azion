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
	_ resource.Resource                = &customPageResource{}
	_ resource.ResourceWithConfigure   = &customPageResource{}
	_ resource.ResourceWithImportState = &customPageResource{}
)

func NewCustomPageResource() resource.Resource {
	return &customPageResource{}
}

type customPageResource struct {
	client *apiClient
}

type customPageResourceModel struct {
	CustomPage  *customPageResourceResults `tfsdk:"custom_page"`
	ID          types.String               `tfsdk:"id"`
	LastUpdated types.String               `tfsdk:"last_updated"`
}

type customPageResourceResults struct {
	ID             types.Int64                     `tfsdk:"id"`
	Name           types.String                    `tfsdk:"name"`
	LastEditor     types.String                    `tfsdk:"last_editor"`
	LastModified   types.String                    `tfsdk:"last_modified"`
	CreatedAt      types.String                    `tfsdk:"created_at"`
	Active         types.Bool                      `tfsdk:"active"`
	ProductVersion types.String                    `tfsdk:"product_version"`
	IsVersioned    types.Bool                      `tfsdk:"is_versioned"`
	Version        types.Int64                     `tfsdk:"version"`
	VersionState   types.String                    `tfsdk:"version_state"`
	VersionID      types.String                    `tfsdk:"version_id"`
	Pages          []customPageResourcePageWrapper `tfsdk:"pages"`
}

type customPageResourcePageWrapper struct {
	Entry *customPageResourcePageResults `tfsdk:"entry"`
}

type customPageResourcePageResults struct {
	Code types.String                           `tfsdk:"code"`
	Page customPageResourcePageConnectorResults `tfsdk:"page"`
}

type customPageResourcePageConnectorResults struct {
	Type       types.String                            `tfsdk:"type"`
	Attributes customPageResourcePageAttributesResults `tfsdk:"attributes"`
}

type customPageResourcePageAttributesResults struct {
	Connector        types.Int64  `tfsdk:"connector"`
	TTL              types.Int64  `tfsdk:"ttl"`
	URI              types.String `tfsdk:"uri"`
	CustomStatusCode types.Int64  `tfsdk:"custom_status_code"`
}

func (r *customPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_page"
}

func (r *customPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Custom Page resource.",
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
			"custom_page": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The custom page identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the custom page.",
						Required:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the custom page.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the custom page.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the custom page.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the custom page.",
						Optional:    true,
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the custom page.",
						Computed:    true,
					},
					"is_versioned": schema.BoolAttribute{
						Description: "Whether the custom page is versioned.",
						Computed:    true,
					},
					"version": schema.Int64Attribute{
						Description: "The current version of the custom page.",
						Computed:    true,
					},
					"version_state": schema.StringAttribute{
						Description: "The state of the current custom page version.",
						Computed:    true,
					},
					"version_id": schema.StringAttribute{
						Description: "The identifier of the current custom page version.",
						Computed:    true,
					},
					"pages": schema.ListNestedAttribute{
						Description: "List of pages associated with the custom page.",
						Required:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"entry": schema.SingleNestedAttribute{
									Description: "A single page entry — pairs an HTTP status code with its connector configuration.",
									Required:    true,
									Attributes: map[string]schema.Attribute{
										"code": schema.StringAttribute{
											Description: "HTTP status code for the page.",
											Required:    true,
										},
										"page": schema.SingleNestedAttribute{
											Description: "Page connector configuration.",
											Required:    true,
											Attributes: map[string]schema.Attribute{
												"type": schema.StringAttribute{
													Description: "Type of the page connector.",
													Required:    true,
												},
												"attributes": schema.SingleNestedAttribute{
													Description: "Attributes of the page connector.",
													Required:    true,
													Attributes: map[string]schema.Attribute{
														"connector": schema.Int64Attribute{
															Description: "Connector ID.",
															Required:    true,
														},
														"ttl": schema.Int64Attribute{
															Description: "Time to live for the page.",
															Optional:    true,
															Computed:    true,
														},
														"uri": schema.StringAttribute{
															Description: "URI for the page.",
															Optional:    true,
															Computed:    true,
														},
														"custom_status_code": schema.Int64Attribute{
															Description: "Custom status code for the page.",
															Optional:    true,
															Computed:    true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *customPageResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *customPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customPageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the request.
	customPageRequest := azionapi.CustomPageRequest{
		Name: plan.CustomPage.Name.ValueString(),
	}

	// Set optional active field.
	if !plan.CustomPage.Active.IsNull() && !plan.CustomPage.Active.IsUnknown() {
		customPageRequest.SetActive(plan.CustomPage.Active.ValueBool())
	}

	// Build pages.
	var pages []azionapi.PageRequestBase
	for _, wrapper := range plan.CustomPage.Pages {
		if wrapper.Entry == nil {
			continue
		}
		page := wrapper.Entry
		pageRequest := azionapi.PageRequestBase{
			Code: page.Code.ValueString(),
			Page: azionapi.PageConnectorRequest{
				Type: page.Page.Type.ValueString(),
				Attributes: azionapi.PageConnectorAttributesRequest{
					Connector: page.Page.Attributes.Connector.ValueInt64(),
				},
			},
		}

		// Set optional TTL.
		if !page.Page.Attributes.TTL.IsNull() && !page.Page.Attributes.TTL.IsUnknown() {
			pageRequest.Page.Attributes.SetTtl(page.Page.Attributes.TTL.ValueInt64())
		}

		// Set optional URI.
		if !page.Page.Attributes.URI.IsNull() && !page.Page.Attributes.URI.IsUnknown() {
			pageRequest.Page.Attributes.SetUri(page.Page.Attributes.URI.ValueString())
		}

		// Set optional CustomStatusCode.
		if !page.Page.Attributes.CustomStatusCode.IsNull() && !page.Page.Attributes.CustomStatusCode.IsUnknown() {
			pageRequest.Page.Attributes.SetCustomStatusCode(page.Page.Attributes.CustomStatusCode.ValueInt64())
		}

		pages = append(pages, pageRequest)
	}
	customPageRequest.SetPages(pages)

	createCustomPage, response, err := r.client.api.CustomPagesAPI.CreateCustomPage(ctx).CustomPageRequest(customPageRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createCustomPage, response, err = utils.RetryOn429(func() (*azionapi.CustomPageResponse, *http.Response, error) {
				return r.client.api.CustomPagesAPI.CreateCustomPage(ctx).CustomPageRequest(customPageRequest).Execute() //nolint
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

	// Populate the state from the response.
	plan.CustomPage = &customPageResourceResults{
		ID:             types.Int64Value(createCustomPage.Data.Id),
		Name:           types.StringValue(createCustomPage.Data.Name),
		LastEditor:     types.StringValue(createCustomPage.Data.LastEditor),
		LastModified:   types.StringValue(createCustomPage.Data.LastModified.Format(time.RFC3339)),
		CreatedAt:      types.StringValue(createCustomPage.Data.CreatedAt.Format(time.RFC3339)),
		ProductVersion: types.StringValue(createCustomPage.Data.ProductVersion),
		IsVersioned:    types.BoolValue(createCustomPage.Data.IsVersioned),
		Version:        types.Int64PointerValue(createCustomPage.Data.Version.Get()),
		VersionState:   types.StringPointerValue(createCustomPage.Data.VersionState.Get()),
		VersionID:      types.StringPointerValue(createCustomPage.Data.VersionId.Get()),
	}

	if createCustomPage.Data.Active != nil {
		plan.CustomPage.Active = types.BoolValue(*createCustomPage.Data.Active)
	}

	// Convert pages from response.
	for _, page := range createCustomPage.Data.Pages {
		pageResult := customPageResourcePageResults{
			Code: types.StringValue(page.Code),
			Page: customPageResourcePageConnectorResults{
				Type: types.StringValue(page.Page.Type),
				Attributes: customPageResourcePageAttributesResults{
					Connector: types.Int64Value(page.Page.Attributes.Connector),
				},
			},
		}

		if page.Page.Attributes.Ttl != nil {
			pageResult.Page.Attributes.TTL = types.Int64Value(*page.Page.Attributes.Ttl)
		}

		if page.Page.Attributes.Uri.IsSet() && page.Page.Attributes.Uri.Get() != nil {
			pageResult.Page.Attributes.URI = types.StringValue(*page.Page.Attributes.Uri.Get())
		}

		if page.Page.Attributes.CustomStatusCode.IsSet() && page.Page.Attributes.CustomStatusCode.Get() != nil {
			pageResult.Page.Attributes.CustomStatusCode = types.Int64Value(*page.Page.Attributes.CustomStatusCode.Get())
		}

		plan.CustomPage.Pages = append(plan.CustomPage.Pages, customPageResourcePageWrapper{Entry: &pageResult})
	}

	plan.ID = types.StringValue(strconv.FormatInt(createCustomPage.Data.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customPageResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var customPageId int64
	var err error
	if state.CustomPage != nil {
		customPageId = state.CustomPage.ID.ValueInt64()
	} else {
		customPageId, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Custom Page ID",
			)
			return
		}
	}

	getCustomPage, response, err := r.client.api.CustomPagesAPI.RetrieveCustomPage(ctx, customPageId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getCustomPage, response, err = utils.RetryOn429(func() (*azionapi.CustomPageResponse, *http.Response, error) {
				return r.client.api.CustomPagesAPI.RetrieveCustomPage(ctx, customPageId).Execute() //nolint
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

	state.CustomPage = &customPageResourceResults{
		ID:             types.Int64Value(getCustomPage.Data.Id),
		Name:           types.StringValue(getCustomPage.Data.Name),
		LastEditor:     types.StringValue(getCustomPage.Data.LastEditor),
		LastModified:   types.StringValue(getCustomPage.Data.LastModified.Format(time.RFC3339)),
		CreatedAt:      types.StringValue(getCustomPage.Data.CreatedAt.Format(time.RFC3339)),
		ProductVersion: types.StringValue(getCustomPage.Data.ProductVersion),
		IsVersioned:    types.BoolValue(getCustomPage.Data.IsVersioned),
		Version:        types.Int64PointerValue(getCustomPage.Data.Version.Get()),
		VersionState:   types.StringPointerValue(getCustomPage.Data.VersionState.Get()),
		VersionID:      types.StringPointerValue(getCustomPage.Data.VersionId.Get()),
	}

	if getCustomPage.Data.Active != nil {
		state.CustomPage.Active = types.BoolValue(*getCustomPage.Data.Active)
	}

	// Convert pages from response.
	for _, page := range getCustomPage.Data.Pages {
		pageResult := customPageResourcePageResults{
			Code: types.StringValue(page.Code),
			Page: customPageResourcePageConnectorResults{
				Type: types.StringValue(page.Page.Type),
				Attributes: customPageResourcePageAttributesResults{
					Connector: types.Int64Value(page.Page.Attributes.Connector),
				},
			},
		}

		if page.Page.Attributes.Ttl != nil {
			pageResult.Page.Attributes.TTL = types.Int64Value(*page.Page.Attributes.Ttl)
		}

		if page.Page.Attributes.Uri.IsSet() && page.Page.Attributes.Uri.Get() != nil {
			pageResult.Page.Attributes.URI = types.StringValue(*page.Page.Attributes.Uri.Get())
		}

		if page.Page.Attributes.CustomStatusCode.IsSet() && page.Page.Attributes.CustomStatusCode.Get() != nil {
			pageResult.Page.Attributes.CustomStatusCode = types.Int64Value(*page.Page.Attributes.CustomStatusCode.Get())
		}

		state.CustomPage.Pages = append(state.CustomPage.Pages, customPageResourcePageWrapper{Entry: &pageResult})
	}

	state.ID = types.StringValue(strconv.FormatInt(getCustomPage.Data.Id, 10))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customPageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state customPageResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the request.
	customPageRequest := azionapi.CustomPageRequest{
		Name: plan.CustomPage.Name.ValueString(),
	}

	// Set optional active field.
	if !plan.CustomPage.Active.IsNull() && !plan.CustomPage.Active.IsUnknown() {
		customPageRequest.SetActive(plan.CustomPage.Active.ValueBool())
	}

	// Build pages.
	var pages []azionapi.PageRequestBase
	for _, wrapper := range plan.CustomPage.Pages {
		if wrapper.Entry == nil {
			continue
		}
		page := wrapper.Entry
		pageRequest := azionapi.PageRequestBase{
			Code: page.Code.ValueString(),
			Page: azionapi.PageConnectorRequest{
				Type: page.Page.Type.ValueString(),
				Attributes: azionapi.PageConnectorAttributesRequest{
					Connector: page.Page.Attributes.Connector.ValueInt64(),
				},
			},
		}

		// Set optional TTL.
		if !page.Page.Attributes.TTL.IsNull() && !page.Page.Attributes.TTL.IsUnknown() {
			pageRequest.Page.Attributes.SetTtl(page.Page.Attributes.TTL.ValueInt64())
		}

		// Set optional URI.
		if !page.Page.Attributes.URI.IsNull() && !page.Page.Attributes.URI.IsUnknown() {
			pageRequest.Page.Attributes.SetUri(page.Page.Attributes.URI.ValueString())
		}

		// Set optional CustomStatusCode.
		if !page.Page.Attributes.CustomStatusCode.IsNull() && !page.Page.Attributes.CustomStatusCode.IsUnknown() {
			pageRequest.Page.Attributes.SetCustomStatusCode(page.Page.Attributes.CustomStatusCode.ValueInt64())
		}

		pages = append(pages, pageRequest)
	}
	customPageRequest.SetPages(pages)

	var customPageId int64
	var err error
	if state.ID.IsNull() {
		customPageId = state.CustomPage.ID.ValueInt64()
	} else {
		customPageId, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Custom Page ID",
			)
			return
		}
	}

	// Custom Pages API uses PUT for full update.
	updateCustomPage, response, err := r.client.api.CustomPagesAPI.UpdateCustomPage(ctx, customPageId).CustomPageRequest(customPageRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateCustomPage, response, err = utils.RetryOn429(func() (*azionapi.CustomPageResponse, *http.Response, error) {
				return r.client.api.CustomPagesAPI.UpdateCustomPage(ctx, customPageId).CustomPageRequest(customPageRequest).Execute() //nolint
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

	// Populate the state from the response.
	plan.CustomPage = &customPageResourceResults{
		ID:             types.Int64Value(updateCustomPage.Data.Id),
		Name:           types.StringValue(updateCustomPage.Data.Name),
		LastEditor:     types.StringValue(updateCustomPage.Data.LastEditor),
		LastModified:   types.StringValue(updateCustomPage.Data.LastModified.Format(time.RFC3339)),
		CreatedAt:      types.StringValue(updateCustomPage.Data.CreatedAt.Format(time.RFC3339)),
		ProductVersion: types.StringValue(updateCustomPage.Data.ProductVersion),
		IsVersioned:    types.BoolValue(updateCustomPage.Data.IsVersioned),
		Version:        types.Int64PointerValue(updateCustomPage.Data.Version.Get()),
		VersionState:   types.StringPointerValue(updateCustomPage.Data.VersionState.Get()),
		VersionID:      types.StringPointerValue(updateCustomPage.Data.VersionId.Get()),
	}

	if updateCustomPage.Data.Active != nil {
		plan.CustomPage.Active = types.BoolValue(*updateCustomPage.Data.Active)
	}

	// Convert pages from response.
	for _, page := range updateCustomPage.Data.Pages {
		pageResult := customPageResourcePageResults{
			Code: types.StringValue(page.Code),
			Page: customPageResourcePageConnectorResults{
				Type: types.StringValue(page.Page.Type),
				Attributes: customPageResourcePageAttributesResults{
					Connector: types.Int64Value(page.Page.Attributes.Connector),
				},
			},
		}

		if page.Page.Attributes.Ttl != nil {
			pageResult.Page.Attributes.TTL = types.Int64Value(*page.Page.Attributes.Ttl)
		}

		if page.Page.Attributes.Uri.IsSet() && page.Page.Attributes.Uri.Get() != nil {
			pageResult.Page.Attributes.URI = types.StringValue(*page.Page.Attributes.Uri.Get())
		}

		if page.Page.Attributes.CustomStatusCode.IsSet() && page.Page.Attributes.CustomStatusCode.Get() != nil {
			pageResult.Page.Attributes.CustomStatusCode = types.Int64Value(*page.Page.Attributes.CustomStatusCode.Get())
		}

		plan.CustomPage.Pages = append(plan.CustomPage.Pages, customPageResourcePageWrapper{Entry: &pageResult})
	}

	plan.ID = types.StringValue(strconv.FormatInt(updateCustomPage.Data.Id, 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customPageResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var customPageId int64
	var err error
	if state.CustomPage != nil {
		customPageId = state.CustomPage.ID.ValueInt64()
	} else {
		customPageId, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Custom Page ID",
			)
			return
		}
	}

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.CustomPagesAPI.DeleteCustomPage(ctx, customPageId).Execute() //nolint
	}, 5) // Maximum 5 retries
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
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

func (r *customPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
