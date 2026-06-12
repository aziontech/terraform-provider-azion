package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	_ resource.Resource                = &bucketResource{}
	_ resource.ResourceWithConfigure   = &bucketResource{}
	_ resource.ResourceWithImportState = &bucketResource{}
)

func NewBucketResource() resource.Resource {
	return &bucketResource{}
}

type bucketResource struct {
	client *apiClient
}

type bucketResourceModel struct {
	Bucket      *bucketResourceResults `tfsdk:"bucket"`
	ID          types.String           `tfsdk:"id"`
	LastUpdated types.String           `tfsdk:"last_updated"`
}

type bucketResourceResults struct {
	Name            types.String `tfsdk:"name"`
	WorkloadsAccess types.String `tfsdk:"workloads_access"`
	LastEditor      types.String `tfsdk:"last_editor"`
	LastModified    types.String `tfsdk:"last_modified"`
	ProductVersion  types.String `tfsdk:"product_version"`
}

func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing Azion Storage Buckets.",
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
			"bucket": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "Name of the bucket. This field is immutable and cannot be updated after creation.",
						Required:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"workloads_access": schema.StringAttribute{
						Description: "Access type for workloads: read_only, read_write, or restricted.",
						Required:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the bucket.",
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

func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket := azionapi.NewBucketCreateRequest(
		plan.Bucket.Name.ValueString(),
		plan.Bucket.WorkloadsAccess.ValueString(),
	)

	createBucket, response, err := r.client.api.StorageBucketsAPI.
		CreateBucket(ctx).
		BucketCreateRequest(*bucket).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createBucket, response, err = utils.RetryOn429(func() (*azionapi.BucketCreateResponse, *http.Response, error) {
				return r.client.api.StorageBucketsAPI.
					CreateBucket(ctx).
					BucketCreateRequest(*bucket).
					Execute() //nolint
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

	plan.Bucket = populateBucketResults(createBucket)
	plan.ID = types.StringValue(createBucket.Data.Name)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var bucketName string
	if state.Bucket != nil {
		bucketName = state.Bucket.Name.ValueString()
	} else {
		bucketName = state.ID.ValueString()
	}

	getBucket, response, err := r.client.api.StorageBucketsAPI.
		RetrieveBucket(ctx, bucketName).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getBucket, response, err = utils.RetryOn429(func() (*azionapi.BucketCreateResponse, *http.Response, error) {
				return r.client.api.StorageBucketsAPI.
					RetrieveBucket(ctx, bucketName).
					Execute() //nolint
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

	state.Bucket = populateBucketResults(getBucket)
	state.ID = types.StringValue(getBucket.Data.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bucketResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state bucketResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Bucket.Name.ValueString()
	updateBucketRequest := azionapi.NewPatchedBucketRequest()

	if !plan.Bucket.WorkloadsAccess.IsNull() && !plan.Bucket.WorkloadsAccess.IsUnknown() {
		updateBucketRequest.SetWorkloadsAccess(plan.Bucket.WorkloadsAccess.ValueString())
	}

	updateBucket, response, err := r.client.api.StorageBucketsAPI.
		UpdateBucket(ctx, bucketName).
		PatchedBucketRequest(*updateBucketRequest).
		Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateBucket, response, err = utils.RetryOn429(func() (*azionapi.BucketCreateResponse, *http.Response, error) {
				return r.client.api.StorageBucketsAPI.
					UpdateBucket(ctx, bucketName).
					PatchedBucketRequest(*updateBucketRequest).
					Execute() //nolint
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

	plan.Bucket = populateBucketResults(updateBucket)
	plan.ID = types.StringValue(updateBucket.Data.Name)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Bucket.Name.ValueString()

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.StorageBucketsAPI.
			DeleteBucket(ctx, bucketName).
			Execute() //nolint
	}, 5)
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

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to populate bucket results from API response.
func populateBucketResults(response *azionapi.BucketCreateResponse) *bucketResourceResults {
	result := &bucketResourceResults{
		Name:            types.StringValue(response.Data.Name),
		WorkloadsAccess: types.StringValue(response.Data.WorkloadsAccess),
		LastEditor:      types.StringValue(response.Data.LastEditor),
		LastModified:    types.StringValue(response.Data.LastModified.Format(time.RFC3339)),
		ProductVersion:  types.StringValue(response.Data.ProductVersion),
	}
	return result
}

func errPrintBucketResource(errCode int, err error) (string, string) {
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
	case 409:
		usrMsg = "Conflict - Bucket already exists"
	default:
		usrMsg = err.Error()
	}
	return usrMsg, fmt.Sprintf("%d - %s", errCode, usrMsg)
}
