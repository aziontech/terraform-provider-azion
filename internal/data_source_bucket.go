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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &BucketDataSource{}
	_ datasource.DataSourceWithConfigure = &BucketDataSource{}
)

func dataSourceAzionBucket() datasource.DataSource {
	return &BucketDataSource{}
}

type BucketDataSource struct {
	client *apiClient
}

type BucketDataSourceModel struct {
	Name types.String       `tfsdk:"name"`
	Data BucketResultsModel `tfsdk:"data"`
	ID   types.String       `tfsdk:"id"`
}

type BucketResultsModel struct {
	Name            types.String `tfsdk:"name"`
	WorkloadsAccess types.String `tfsdk:"workloads_access"`
	LastEditor      types.String `tfsdk:"last_editor"`
	LastModified    types.String `tfsdk:"last_modified"`
	ProductVersion  types.String `tfsdk:"product_version"`
}

func (d *BucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *BucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (d *BucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the bucket to retrieve.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
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
	}
}

func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var bucketName types.String
	diags := req.Config.GetAttribute(ctx, path.Root("name"), &bucketName)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketResponse, response, err := d.client.api.StorageBucketsAPI.
		RetrieveBucket(ctx, bucketName.ValueString()).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			bucketResponse, response, err = utils.RetryOn429(func() (*azionapi.BucketCreateResponse, *http.Response, error) {
				return d.client.api.StorageBucketsAPI.RetrieveBucket(ctx, bucketName.ValueString()).Execute() //nolint
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
			usrMsg, errMsg := errPrintBucket(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	bucketState := BucketDataSourceModel{
		Name: types.StringValue(bucketName.ValueString()),
		ID:   types.StringValue(bucketName.ValueString()),
		Data: BucketResultsModel{
			Name:            types.StringValue(bucketResponse.Data.Name),
			WorkloadsAccess: types.StringValue(bucketResponse.Data.WorkloadsAccess),
			LastEditor:      types.StringValue(bucketResponse.Data.LastEditor),
			LastModified:    types.StringValue(bucketResponse.Data.LastModified.Format(time.RFC3339)),
			ProductVersion:  types.StringValue(bucketResponse.Data.ProductVersion),
		},
	}

	diags = resp.State.Set(ctx, &bucketState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errPrintBucket(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "Bucket not found"
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
