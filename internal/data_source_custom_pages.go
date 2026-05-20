package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CustomPagesDataSource{}
	_ datasource.DataSourceWithConfigure = &CustomPagesDataSource{}
)

func dataSourceAzionCustomPages() datasource.DataSource {
	return &CustomPagesDataSource{}
}

type CustomPagesDataSource struct {
	client *apiClient
}

type CustomPagesDataSourceModel struct {
	Counter types.Int64          `tfsdk:"counter"`
	Results []CustomPagesResults `tfsdk:"results"`
	ID      types.String         `tfsdk:"id"`
}

type CustomPagesResults struct {
	ID             types.Int64              `tfsdk:"id"`
	Name           types.String             `tfsdk:"name"`
	LastEditor     types.String             `tfsdk:"last_editor"`
	LastModified   types.String             `tfsdk:"last_modified"`
	CreatedAt      types.String             `tfsdk:"created_at"`
	Active         types.Bool               `tfsdk:"active"`
	ProductVersion types.String             `tfsdk:"product_version"`
	Pages          []CustomPagesPageWrapper `tfsdk:"pages"`
}

type CustomPagesPageWrapper struct {
	Entry *CustomPagesPageResults `tfsdk:"entry"`
}

type CustomPagesPageResults struct {
	Code types.String                    `tfsdk:"code"`
	Page CustomPagesPageConnectorResults `tfsdk:"page"`
}

type CustomPagesPageConnectorResults struct {
	Type       types.String                     `tfsdk:"type"`
	Attributes CustomPagesPageAttributesResults `tfsdk:"attributes"`
}

type CustomPagesPageAttributesResults struct {
	Connector        types.Int64  `tfsdk:"connector"`
	TTL              types.Int64  `tfsdk:"ttl"`
	URI              types.String `tfsdk:"uri"`
	CustomStatusCode types.Int64  `tfsdk:"custom_status_code"`
}

func (d *CustomPagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *CustomPagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_pages"
}

func (d *CustomPagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of custom pages.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
		},
	}
}

func (d *CustomPagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	customPagesResponse, response, err := d.client.api.CustomPagesAPI.ListCustomPages(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			customPagesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCustomPageList, *http.Response, error) {
				return d.client.api.CustomPagesAPI.ListCustomPages(ctx).Execute() //nolint
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
			usrMsg, errMsg := errPrintCustomPages(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	customPagesState := CustomPagesDataSourceModel{
		Counter: types.Int64Value(*customPagesResponse.Count),
	}

	for _, resultCustomPage := range customPagesResponse.GetResults() {
		result := CustomPagesResults{
			ID:             types.Int64Value(resultCustomPage.Id),
			Name:           types.StringValue(resultCustomPage.Name),
			LastEditor:     types.StringValue(resultCustomPage.LastEditor),
			LastModified:   types.StringValue(resultCustomPage.LastModified.Format(time.RFC3339)),
			CreatedAt:      types.StringValue(resultCustomPage.CreatedAt.Format(time.RFC3339)),
			ProductVersion: types.StringValue(resultCustomPage.ProductVersion),
		}

		// Handle optional active field.
		if resultCustomPage.Active != nil {
			result.Active = types.BoolValue(*resultCustomPage.Active)
		}

		// Convert pages.
		for _, page := range resultCustomPage.Pages {
			pageResult := CustomPagesPageResults{
				Code: types.StringValue(page.Code),
				Page: CustomPagesPageConnectorResults{
					Type: types.StringValue(page.Page.Type),
					Attributes: CustomPagesPageAttributesResults{
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

			result.Pages = append(result.Pages, CustomPagesPageWrapper{Entry: &pageResult})
		}

		customPagesState.Results = append(customPagesState.Results, result)
	}

	customPagesState.ID = types.StringValue("Get All Custom Pages")
	diags := resp.State.Set(ctx, &customPagesState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintCustomPages(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Custom Pages found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
