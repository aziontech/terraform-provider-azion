package provider

import (
	"context"
	"io"
	"net/http"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CacheSettingsDataSource{}
	_ datasource.DataSourceWithConfigure = &CacheSettingsDataSource{}
)

func dataSourceAzionEdgeApplicationCacheSettings() datasource.DataSource {
	return &CacheSettingsDataSource{}
}

type CacheSettingsDataSource struct {
	client *apiClient
}

type CacheSettingsDataSourceModel struct {
	ApplicationID types.Int64         `tfsdk:"edge_application_id"`
	Counter       types.Int64         `tfsdk:"counter"`
	Page          types.Int64         `tfsdk:"page"`
	PageSize      types.Int64         `tfsdk:"page_size"`
	TotalPages    types.Int64         `tfsdk:"total_pages"`
	Links         *LinksModel         `tfsdk:"links"`
	Results       []CacheSettingModel `tfsdk:"results"`
	ID            types.Int64         `tfsdk:"id"`
}

func (d *CacheSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *CacheSettingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_application_cache_settings"
}

func (d *CacheSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"edge_application_id": schema.Int64Attribute{
				Description: "Numeric identifier of the Edge Application.",
				Required:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of Cache Settings.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The page number of Cache Settings.",
				Optional:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The Page Size number of Cache Settings.",
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
							Description: "The cache setting identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the cache setting.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation timestamp of the cache setting.",
							Computed:    true,
						},
						"browser_cache": schema.SingleNestedAttribute{
							Description: "Browser cache settings.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"behavior": schema.StringAttribute{
									Description: "Browser cache behavior: override, honor, no-cache.",
									Computed:    true,
								},
								"max_age": schema.Int64Attribute{
									Description: "Maximum TTL for browser cache.",
									Computed:    true,
								},
							},
						},
						"modules": schema.SingleNestedAttribute{
							Description: "Cache settings modules.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"cache": schema.SingleNestedAttribute{
									Description: "Edge cache module settings.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"behavior": schema.StringAttribute{
											Description: "Cache behavior: honor, override.",
											Computed:    true,
										},
										"max_age": schema.Int64Attribute{
											Description: "Maximum TTL for edge cache.",
											Computed:    true,
										},
										"stale_cache": schema.SingleNestedAttribute{
											Description: "Stale cache settings.",
											Computed:    true,
											Attributes: map[string]schema.Attribute{
												"enabled": schema.BoolAttribute{
													Computed: true,
												},
											},
										},
										"large_file_cache": schema.SingleNestedAttribute{
											Description: "Large file cache settings.",
											Computed:    true,
											Attributes: map[string]schema.Attribute{
												"enabled": schema.BoolAttribute{
													Computed: true,
												},
												"offset": schema.Int64Attribute{
													Computed: true,
												},
											},
										},
										"tiered_cache": schema.SingleNestedAttribute{
											Description: "Tiered cache settings.",
											Computed:    true,
											Attributes: map[string]schema.Attribute{
												"topology": schema.StringAttribute{
													Description: "Tiered cache topology.",
													Computed:    true,
												},
												"enabled": schema.BoolAttribute{
													Computed: true,
												},
											},
										},
									},
								},
								"application_accelerator": schema.SingleNestedAttribute{
									Description: "Application accelerator module settings.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"cache_vary_by_method": schema.ListAttribute{
											ElementType: types.StringType,
											Computed:    true,
										},
										"cache_vary_by_querystring": schema.SingleNestedAttribute{
											Computed: true,
											Attributes: map[string]schema.Attribute{
												"behavior": schema.StringAttribute{Computed: true},
												"fields": schema.ListAttribute{
													ElementType: types.StringType,
													Computed:    true,
												},
												"sort_enabled": schema.BoolAttribute{Computed: true},
											},
										},
										"cache_vary_by_cookies": schema.SingleNestedAttribute{
											Computed: true,
											Attributes: map[string]schema.Attribute{
												"behavior": schema.StringAttribute{Computed: true},
												"cookie_names": schema.ListAttribute{
													ElementType: types.StringType,
													Computed:    true,
												},
											},
										},
										"cache_vary_by_devices": schema.SingleNestedAttribute{
											Computed: true,
											Attributes: map[string]schema.Attribute{
												"behavior": schema.StringAttribute{Computed: true},
												"device_group": schema.ListAttribute{
													ElementType: types.Int64Type,
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
	}
}

func (d *CacheSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var applicationID types.Int64
	var page types.Int64
	var pageSize types.Int64

	diags := req.Config.GetAttribute(ctx, path.Root("edge_application_id"), &applicationID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get page and page_size from config
	diags = req.Config.GetAttribute(ctx, path.Root("page"), &page)
	resp.Diagnostics.Append(diags...)

	diags = req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSize)
	resp.Diagnostics.Append(diags...)

	// Set defaults
	if page.IsNull() || page.IsUnknown() {
		page = types.Int64Value(1)
	}
	if pageSize.IsNull() || pageSize.IsUnknown() {
		pageSize = types.Int64Value(10)
	}

	listResponse, response, err := d.client.api.ApplicationsCacheSettingsAPI.ListCacheSettings(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			listResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCacheSettingList, *http.Response, error) {
				return d.client.api.ApplicationsCacheSettingsAPI.ListCacheSettings(ctx, applicationID.ValueInt64()).Page(page.ValueInt64()).PageSize(pageSize.ValueInt64()).Execute()
			}, 5)

			if response != nil {
				defer response.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			bodyBytes, errReadAll := io.ReadAll(response.Body)
			if errReadAll != nil {
				resp.Diagnostics.AddError(errReadAll.Error(), "err")
			}
			bodyString := string(bodyBytes)
			resp.Diagnostics.AddError(err.Error(), bodyString)
			if response != nil {
				response.Body.Close()
			}
			return
		}
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Build state
	state := CacheSettingsDataSourceModel{
		ApplicationID: applicationID,
		Page:          page,
		PageSize:      pageSize,
	}

	if listResponse.HasCount() {
		state.Counter = types.Int64Value(listResponse.GetCount())
	}
	if listResponse.HasTotalPages() {
		state.TotalPages = types.Int64Value(listResponse.GetTotalPages())
	}

	// Links
	var previous, next string
	if listResponse.HasPrevious() {
		prev := listResponse.GetPrevious()
		if prev != "" {
			previous = prev
		}
	}
	if listResponse.HasNext() {
		n := listResponse.GetNext()
		if n != "" {
			next = n
		}
	}
	state.Links = &LinksModel{
		Previous: types.StringValue(previous),
		Next:     types.StringValue(next),
	}

	// Results
	if listResponse.HasResults() {
		for _, cs := range listResponse.GetResults() {
			state.Results = append(state.Results, *transformCacheSettingToModel(&cs))
		}
	}

	state.ID = types.Int64Value(0)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
