package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &WafsDataSource{}
	_ datasource.DataSourceWithConfigure = &WafsDataSource{}
)

func dataSourceAzionWafs() datasource.DataSource {
	return &WafsDataSource{}
}

type WafsDataSource struct {
	client *apiClient
}

type WafsDataSourceModel struct {
	ID         types.String       `tfsdk:"id"`
	Counter    types.Int64        `tfsdk:"counter"`
	TotalPages types.Int64        `tfsdk:"total_pages"`
	Page       types.Int64        `tfsdk:"page"`
	PageSize   types.Int64        `tfsdk:"page_size"`
	Links      *WafsResponseLinks `tfsdk:"links"`
	Results    []WafListItemModel `tfsdk:"results"`
}

type WafsResponseLinks struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type WafListItemModel struct {
	ID             types.Int64             `tfsdk:"id"`
	Name           types.String            `tfsdk:"name"`
	Active         types.Bool              `tfsdk:"active"`
	LastEditor     types.String            `tfsdk:"last_editor"`
	LastModified   types.String            `tfsdk:"last_modified"`
	ProductVersion types.String            `tfsdk:"product_version"`
	EngineSettings *WafEngineSettingsModel `tfsdk:"engine_settings"`
}

func (o *WafsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wafs"
}

func (o *WafsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of WAFs.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The page size number.",
				Optional:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"links": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"previous": schema.StringAttribute{
						Computed: true,
					},
					"next": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the WAF.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the WAF.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the WAF is active.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the WAF.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp.",
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "Product version of the WAF.",
							Computed:    true,
						},
						"engine_settings": schema.SingleNestedAttribute{
							Description: "Engine settings for the WAF.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"engine_version": schema.StringAttribute{
									Description: "Engine version for the WAF.",
									Computed:    true,
								},
								"type": schema.StringAttribute{
									Description: "Type of the WAF engine.",
									Computed:    true,
								},
								"attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the WAF engine settings.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"rulesets": schema.ListAttribute{
											Description: "List of ruleset IDs.",
											Computed:    true,
											ElementType: types.Int64Type,
										},
										"thresholds": schema.ListNestedAttribute{
											Description: "Threshold configurations for the WAF.",
											Computed:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"threshold": schema.SingleNestedAttribute{
														Description: "A single threshold configuration.",
														Computed:    true,
														Attributes: map[string]schema.Attribute{
															"threat": schema.StringAttribute{
																Description: "The threat type for the threshold.",
																Computed:    true,
															},
															"sensitivity": schema.StringAttribute{
																Description: "The sensitivity level for the threshold.",
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
		},
	}
}

func (o *WafsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var page, pageSize types.Int64

	diagsPage := req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diagsPage...)
	if resp.Diagnostics.HasError() {
		return
	}

	diagsPageSize := req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
	resp.Diagnostics.Append(diagsPageSize...)
	if resp.Diagnostics.HasError() {
		return
	}

	if page.IsNull() || page.IsUnknown() || page.ValueInt64() == 0 {
		page = types.Int64Value(1)
	}
	if pageSize.IsNull() || pageSize.IsUnknown() || pageSize.ValueInt64() == 0 {
		pageSize = types.Int64Value(10)
	}

	listResponse, response, err := o.client.api.WAFsAPI.ListWafs(ctx).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedWAFList, *http.Response, error) {
				return o.client.api.WAFsAPI.ListWafs(ctx).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
			}, 5)

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

	if response != nil {
		defer response.Body.Close()
	}

	// Transform links.
	var previous, next string
	if listResponse.HasPrevious() {
		previous = listResponse.GetPrevious()
	}
	if listResponse.HasNext() {
		next = listResponse.GetNext()
	}

	// Transform results.
	var results []WafListItemModel
	for _, waf := range listResponse.GetResults() {
		results = append(results, transformWAFToListItemModel(waf))
	}

	state := WafsDataSourceModel{
		ID:         types.StringValue("Get All WAFs"),
		Results:    results,
		TotalPages: types.Int64Value(listResponse.GetTotalPages()),
		Page:       page,
		PageSize:   pageSize,
		Counter:    types.Int64Value(listResponse.GetCount()),
		Links: &WafsResponseLinks{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// transformWAFToListItemModel transforms an SDK WAF to a Terraform list item model.
func transformWAFToListItemModel(waf azionapi.WAF) WafListItemModel {
	result := WafListItemModel{
		ID:           types.Int64Value(waf.GetId()),
		Name:         types.StringValue(waf.GetName()),
		LastEditor:   types.StringValue(waf.GetLastEditor()),
		LastModified: types.StringValue(waf.GetLastModified().Format(time.RFC3339)),
	}

	// Optional active.
	if waf.HasActive() {
		result.Active = types.BoolValue(waf.GetActive())
	} else {
		result.Active = types.BoolNull()
	}

	// Optional product_version.
	if waf.HasProductVersion() {
		result.ProductVersion = types.StringValue(waf.GetProductVersion())
	} else {
		result.ProductVersion = types.StringNull()
	}

	// Optional engine_settings.
	if waf.HasEngineSettings() {
		engineSettings := waf.GetEngineSettings()
		result.EngineSettings = transformWAFEngineSettingsToModel(engineSettings)
	} else {
		result.EngineSettings = nil
	}

	return result
}
