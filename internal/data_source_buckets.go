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
	_ datasource.DataSource              = &BucketsDataSource{}
	_ datasource.DataSourceWithConfigure = &BucketsDataSource{}
)

func dataSourceAzionBuckets() datasource.DataSource {
	return &BucketsDataSource{}
}

type BucketsDataSource struct {
	client *apiClient
}

type BucketsDataSourceModel struct {
	Counter    types.Int64           `tfsdk:"counter"`
	TotalPages types.Int64           `tfsdk:"total_pages"`
	Page       types.Int64           `tfsdk:"page"`
	PageSize   types.Int64           `tfsdk:"page_size"`
	Results    []BucketsResultsModel `tfsdk:"results"`
	ID         types.String          `tfsdk:"id"`
}

type BucketsResultsModel struct {
	Name            types.String `tfsdk:"name"`
	WorkloadsAccess types.String `tfsdk:"workloads_access"`
	LastEditor      types.String `tfsdk:"last_editor"`
	LastModified    types.String `tfsdk:"last_modified"`
	ProductVersion  types.String `tfsdk:"product_version"`
}

func (d *BucketsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *BucketsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buckets"
}

func (d *BucketsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of buckets.",
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
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the bucket.",
							Computed:    true,
						},
						"workloads_access": schema.StringAttribute{
							Description: "Access type for workloads: read_only, read_write, or restricted.",
							Computed:    true,
						},
						"last_editor": schema.StringAttribute{
							Description: "Last editor of the bucket.",
							Computed:    true,
						},
						"last_modified": schema.StringAttribute{
							Description: "Last modified timestamp of the bucket.",
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "Product version of the bucket.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *BucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	bucketsResponse, response, err := d.client.api.StorageBucketsAPI.
		ListBuckets(ctx).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			bucketsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedBucketList, *http.Response, error) {
				return d.client.api.StorageBucketsAPI.ListBuckets(ctx).Execute() //nolint
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
			usrMsg, errMsg := errPrintBuckets(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	bucketsState := BucketsDataSourceModel{
		ID: types.StringValue("buckets"),
	}

	if bucketsResponse.Count != nil {
		bucketsState.Counter = types.Int64Value(*bucketsResponse.Count)
	}

	if bucketsResponse.TotalPages != nil {
		bucketsState.TotalPages = types.Int64Value(*bucketsResponse.TotalPages)
	}

	if bucketsResponse.Page != nil {
		bucketsState.Page = types.Int64Value(*bucketsResponse.Page)
	}

	if bucketsResponse.PageSize != nil {
		bucketsState.PageSize = types.Int64Value(*bucketsResponse.PageSize)
	}

	if bucketsResponse.Results != nil {
		results := make([]BucketsResultsModel, len(bucketsResponse.Results))
		for i, bucket := range bucketsResponse.Results {
			results[i] = BucketsResultsModel{
				Name:            types.StringValue(bucket.Name),
				WorkloadsAccess: types.StringValue(bucket.WorkloadsAccess),
				LastEditor:      types.StringValue(bucket.LastEditor),
				LastModified:    types.StringValue(bucket.LastModified.Format(time.RFC3339)),
				ProductVersion:  types.StringValue(bucket.ProductVersion),
			}
		}
		bucketsState.Results = results
	}

	diags := resp.State.Set(ctx, &bucketsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintBuckets(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "Buckets not found"
	case 403:
		usrMsg = "Forbidden"
	case 405:
		usrMsg = "Method Not Allowed"
	case 406:
		usrMsg = "Not Acceptable"
	default:
		usrMsg = err.Error()
	}
	return usrMsg, fmt.Sprintf("%d - %s", errCode, usrMsg)
}
