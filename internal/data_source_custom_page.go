package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CustomPageDataSource{}
	_ datasource.DataSourceWithConfigure = &CustomPageDataSource{}
)

func dataSourceAzionCustomPage() datasource.DataSource {
	return &CustomPageDataSource{}
}

type CustomPageDataSource struct {
	client *apiClient
}

type CustomPageDataSourceModel struct {
	Data CustomPageResults `tfsdk:"data"`
	ID   types.String      `tfsdk:"id"`
}

type CustomPageResults struct {
	ID             types.Int64             `tfsdk:"id"`
	Name           types.String            `tfsdk:"name"`
	LastEditor     types.String            `tfsdk:"last_editor"`
	LastModified   types.String            `tfsdk:"last_modified"`
	CreatedAt      types.String            `tfsdk:"created_at"`
	Active         types.Bool              `tfsdk:"active"`
	ProductVersion types.String            `tfsdk:"product_version"`
	IsVersioned    types.Bool              `tfsdk:"is_versioned"`
	Version        types.Int64             `tfsdk:"version"`
	VersionState   types.String            `tfsdk:"version_state"`
	VersionID      types.String            `tfsdk:"version_id"`
	Pages          []CustomPagePageWrapper `tfsdk:"pages"`
}

type CustomPagePageWrapper struct {
	Entry *CustomPagePageResults `tfsdk:"entry"`
}

type CustomPagePageResults struct {
	Code types.String                   `tfsdk:"code"`
	Page CustomPagePageConnectorResults `tfsdk:"page"`
}

type CustomPagePageConnectorResults struct {
	Type       types.String                    `tfsdk:"type"`
	Attributes CustomPagePageAttributesResults `tfsdk:"attributes"`
}

type CustomPagePageAttributesResults struct {
	Connector        types.Int64  `tfsdk:"connector"`
	TTL              types.Int64  `tfsdk:"ttl"`
	URI              types.String `tfsdk:"uri"`
	CustomStatusCode types.Int64  `tfsdk:"custom_status_code"`
}

func (d *CustomPageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *CustomPageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_page"
}

func (d *CustomPageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The custom page identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the custom page.",
						Computed:    true,
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
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"entry": schema.SingleNestedAttribute{
									Description: "A single page entry — pairs an HTTP status code with its connector configuration.",
									Computed:    true,
									Attributes: map[string]schema.Attribute{
										"code": schema.StringAttribute{
											Description: "HTTP status code for the page.",
											Computed:    true,
										},
										"page": schema.SingleNestedAttribute{
											Description: "Page connector configuration.",
											Computed:    true,
											Attributes: map[string]schema.Attribute{
												"type": schema.StringAttribute{
													Description: "Type of the page connector.",
													Computed:    true,
												},
												"attributes": schema.SingleNestedAttribute{
													Description: "Attributes of the page connector.",
													Computed:    true,
													Attributes: map[string]schema.Attribute{
														"connector": schema.Int64Attribute{
															Description: "Connector ID.",
															Computed:    true,
														},
														"ttl": schema.Int64Attribute{
															Description: "Time to live for the page.",
															Computed:    true,
														},
														"uri": schema.StringAttribute{
															Description: "URI for the page.",
															Computed:    true,
														},
														"custom_status_code": schema.Int64Attribute{
															Description: "Custom status code for the page.",
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

func (d *CustomPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getCustomPageId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getCustomPageId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	customPageID, err := strconv.ParseInt(getCustomPageId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	customPageResponse, response, err := d.client.api.CustomPagesAPI.
		RetrieveCustomPage(ctx, customPageID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			customPageResponse, response, err = utils.RetryOn429(func() (*azionapi.CustomPageResponse, *http.Response, error) {
				return d.client.api.CustomPagesAPI.RetrieveCustomPage(ctx, customPageID).Execute() //nolint
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
			usrMsg, errMsg := errPrintCustomPage(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	customPageState := CustomPageDataSourceModel{
		Data: CustomPageResults{
			ID:             types.Int64Value(customPageResponse.Data.Id),
			Name:           types.StringValue(customPageResponse.Data.Name),
			LastEditor:     types.StringValue(customPageResponse.Data.LastEditor),
			LastModified:   types.StringValue(customPageResponse.Data.LastModified.Format(time.RFC3339)),
			CreatedAt:      types.StringValue(customPageResponse.Data.CreatedAt.Format(time.RFC3339)),
			ProductVersion: types.StringValue(customPageResponse.Data.ProductVersion),
			IsVersioned:    types.BoolValue(customPageResponse.Data.IsVersioned),
			Version:        types.Int64PointerValue(customPageResponse.Data.Version.Get()),
			VersionState:   types.StringPointerValue(customPageResponse.Data.VersionState.Get()),
			VersionID:      types.StringPointerValue(customPageResponse.Data.VersionId.Get()),
		},
	}

	// Handle optional active field.
	if customPageResponse.Data.Active != nil {
		customPageState.Data.Active = types.BoolValue(*customPageResponse.Data.Active)
	}

	// Convert pages.
	for _, page := range customPageResponse.Data.Pages {
		pageResult := CustomPagePageResults{
			Code: types.StringValue(page.Code),
			Page: CustomPagePageConnectorResults{
				Type: types.StringValue(page.Page.Type),
				Attributes: CustomPagePageAttributesResults{
					Connector: types.Int64Value(page.Page.Attributes.Connector),
				},
			},
		}

		// Handle optional TTL.
		if page.Page.Attributes.Ttl != nil {
			pageResult.Page.Attributes.TTL = types.Int64Value(*page.Page.Attributes.Ttl)
		}

		// Handle optional URI.
		if page.Page.Attributes.Uri.IsSet() && page.Page.Attributes.Uri.Get() != nil {
			pageResult.Page.Attributes.URI = types.StringValue(*page.Page.Attributes.Uri.Get())
		}

		// Handle optional CustomStatusCode.
		if page.Page.Attributes.CustomStatusCode.IsSet() && page.Page.Attributes.CustomStatusCode.Get() != nil {
			pageResult.Page.Attributes.CustomStatusCode = types.Int64Value(*page.Page.Attributes.CustomStatusCode.Get())
		}

		customPageState.Data.Pages = append(customPageState.Data.Pages, CustomPagePageWrapper{Entry: &pageResult})
	}

	customPageState.ID = types.StringValue("Get By Id Custom Page")
	diags = resp.State.Set(ctx, &customPageState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintCustomPage(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Custom Page found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
