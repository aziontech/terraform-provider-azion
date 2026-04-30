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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CrlsDataSource{}
	_ datasource.DataSourceWithConfigure = &CrlsDataSource{}
)

func dataSourceAzionCrls() datasource.DataSource {
	return &CrlsDataSource{}
}

type CrlsDataSource struct {
	client *apiClient
}

type CrlsDataSourceModel struct {
	ID            types.String      `tfsdk:"id"`
	Counter       types.Int64       `tfsdk:"counter"`
	TotalPages    types.Int64       `tfsdk:"total_pages"`
	Page          types.Int64       `tfsdk:"page"`
	PageSize      types.Int64       `tfsdk:"page_size"`
	Links         *CrlLinksModel    `tfsdk:"links"`
	SchemaVersion types.Int64       `tfsdk:"schema_version"`
	Results       []CrlsResultModel `tfsdk:"results"`
}

type CrlLinksModel struct {
	Previous types.String `tfsdk:"previous"`
	Next     types.String `tfsdk:"next"`
}

type CrlsResultModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Active         types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	CreatedAt      types.String `tfsdk:"created_at"`
	LastModified   types.String `tfsdk:"last_modified"`
	ProductVersion types.String `tfsdk:"product_version"`
	Issuer         types.String `tfsdk:"issuer"`
	LastUpdate     types.String `tfsdk:"last_update"`
	NextUpdate     types.String `tfsdk:"next_update"`
	Crl            types.String `tfsdk:"crl"`
}

func (d *CrlsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *CrlsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crls"
}

func (d *CrlsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total number of certificate revocation lists.",
				Computed:    true,
			},
			"total_pages": schema.Int64Attribute{
				Description: "The total number of pages.",
				Computed:    true,
			},
			"page": schema.Int64Attribute{
				Description: "The current page number.",
				Computed:    true,
			},
			"page_size": schema.Int64Attribute{
				Description: "The number of items per page.",
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
							Description: "Identifier of the certificate revocation list.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the certificate revocation list.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Indicates if the certificate revocation list is active.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the certificate revocation list.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Timestamp of the certificate revocation list creation on the platform.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Timestamp of the last modification made to the certificate content on the platform.",
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "Product version of the certificate revocation list.",
							Computed:    true,
						},
						"issuer": schema.StringAttribute{
							Description: "Issuer of the certificate revocation list.",
							Computed:    true,
						},
						"last_update": schema.StringAttribute{
							Description: "Timestamp of the last update issued by the certification revocation list issuer.",
							Computed:    true,
						},
						"next_update": schema.StringAttribute{
							Description: "Timestamp of the next scheduled update from the certification revocation list issuer.",
							Computed:    true,
						},
						"crl": schema.StringAttribute{
							Description: "The certificate revocation list content.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *CrlsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	crlsResponse, response, err := d.client.api.DigitalCertificatesCertificateRevocationListsAPI.ListCertificateRevocationLists(ctx).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			crlsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedCertificateRevocationList, *http.Response, error) {
				return d.client.api.DigitalCertificatesCertificateRevocationListsAPI.ListCertificateRevocationLists(ctx).Execute()
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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	state := populateCrlsListResults(crlsResponse)
	state.ID = types.StringValue("Get All Certificate Revocation Lists")
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateCrlsListResults transforms API response data to Terraform state model.
func populateCrlsListResults(list *azionapi.PaginatedCertificateRevocationList) CrlsDataSourceModel {
	var previous, next string
	if list.HasPrevious() {
		previous = list.GetPrevious()
	}
	if list.HasNext() {
		next = list.GetNext()
	}

	var results []CrlsResultModel
	for _, crl := range list.GetResults() {
		var createdAt string
		if crl.CreatedAt.IsSet() && crl.CreatedAt.Get() != nil {
			createdAt = (*crl.CreatedAt.Get()).Format(time.RFC3339)
		}

		crlInfo := CrlsResultModel{
			ID:             types.Int64Value(crl.GetId()),
			Name:           types.StringValue(crl.GetName()),
			LastEditor:     types.StringValue(crl.GetLastEditor()),
			CreatedAt:      types.StringValue(createdAt),
			LastModified:   types.StringValue(crl.GetLastModified().Format(time.RFC3339)),
			ProductVersion: types.StringValue(crl.GetProductVersion()),
			Issuer:         types.StringValue(crl.GetIssuer()),
			LastUpdate:     types.StringValue(crl.GetLastUpdate().Format(time.RFC3339)),
			NextUpdate:     types.StringValue(crl.GetNextUpdate().Format(time.RFC3339)),
			Crl:            types.StringValue(crl.GetCrl()),
		}

		// Handle optional fields
		if crl.Active != nil {
			crlInfo.Active = types.BoolValue(*crl.Active)
		}

		results = append(results, crlInfo)
	}

	state := CrlsDataSourceModel{
		SchemaVersion: types.Int64Value(1),
		Results:       results,
		Links: &CrlLinksModel{
			Previous: types.StringValue(previous),
			Next:     types.StringValue(next),
		},
	}

	if list.HasCount() {
		state.Counter = types.Int64Value(list.GetCount())
	}

	if list.HasTotalPages() {
		state.TotalPages = types.Int64Value(list.GetTotalPages())
	}

	if list.HasPage() {
		state.Page = types.Int64Value(list.GetPage())
	}

	if list.HasPageSize() {
		state.PageSize = types.Int64Value(list.GetPageSize())
	}

	return state
}
