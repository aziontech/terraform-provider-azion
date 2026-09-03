package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &dataStreamResource{}
	_ resource.ResourceWithConfigure   = &dataStreamResource{}
	_ resource.ResourceWithImportState = &dataStreamResource{}
)

func NewDataStreamResource() resource.Resource {
	return &dataStreamResource{}
}

type dataStreamResource struct {
	client *apiClient
}

// Main resource model.
type dataStreamResourceModel struct {
	DataStream    *dataStreamResourceResults `tfsdk:"data_stream"`
	ID            types.String               `tfsdk:"id"`
	LastUpdated   types.String               `tfsdk:"last_updated"`
	SchemaVersion types.Int64                `tfsdk:"schema_version"`
}

// Data stream results - the stream body plus computed metadata.
type dataStreamResourceResults struct {
	ID             types.Int64                `tfsdk:"id"`
	Name           types.String               `tfsdk:"name"`
	Active         types.Bool                 `tfsdk:"active"`
	LastEditor     types.String               `tfsdk:"last_editor"`
	CreatedAt      types.String               `tfsdk:"created_at"`
	LastModified   types.String               `tfsdk:"last_modified"`
	ProductVersion types.String               `tfsdk:"product_version"`
	Inputs         []DataStreamInputModel     `tfsdk:"inputs"`
	Transform      []DataStreamTransformModel `tfsdk:"transform"`
	Outputs        []DataStreamOutputModel    `tfsdk:"outputs"`
}

// DataStreamInputModel is a single input of the stream. Only `raw_logs` exists today.
type DataStreamInputModel struct {
	Type       types.String                    `tfsdk:"type"`
	Attributes *DataStreamInputAttributesModel `tfsdk:"attributes"`
}

// DataStreamInputAttributesModel holds the data source selected for an input.
type DataStreamInputAttributesModel struct {
	DataSource types.String `tfsdk:"data_source"`
}

// DataStreamTransformModel is a polymorphic transform step. Exactly one of the
// `*_attributes` blocks must be set, matching `type`.
type DataStreamTransformModel struct {
	Type                      types.String                              `tfsdk:"type"`
	SamplingAttributes        *DataStreamSamplingAttributesModel        `tfsdk:"sampling_attributes"`
	FilterWorkloadsAttributes *DataStreamFilterWorkloadsAttributesModel `tfsdk:"filter_workloads_attributes"`
	RenderTemplateAttributes  *DataStreamRenderTemplateAttributesModel  `tfsdk:"render_template_attributes"`
}

// DataStreamSamplingAttributesModel configures the `sampling` transform.
type DataStreamSamplingAttributesModel struct {
	Rate types.Int64 `tfsdk:"rate"`
}

// DataStreamFilterWorkloadsAttributesModel configures the `filter_workloads` transform.
type DataStreamFilterWorkloadsAttributesModel struct {
	Workloads []types.Int64 `tfsdk:"workloads"`
}

// DataStreamRenderTemplateAttributesModel configures the `render_template` transform.
type DataStreamRenderTemplateAttributesModel struct {
	Template types.Int64 `tfsdk:"template"`
}

// DataStreamOutputModel is a polymorphic output (endpoint) of the stream. Exactly
// one of the `*_attributes` blocks must be set, matching `type`.
type DataStreamOutputModel struct {
	Type                       types.String                             `tfsdk:"type"`
	StandardAttributes         *DataStreamStandardOutputModel           `tfsdk:"standard_attributes"`
	KafkaAttributes            *DataStreamKafkaOutputModel              `tfsdk:"kafka_attributes"`
	S3Attributes               *DataStreamS3OutputModel                 `tfsdk:"s3_attributes"`
	BigQueryAttributes         *DataStreamBigQueryOutputModel           `tfsdk:"big_query_attributes"`
	ElasticsearchAttributes    *DataStreamURLAPIKeyOutputModel          `tfsdk:"elasticsearch_attributes"`
	SplunkAttributes           *DataStreamURLAPIKeyOutputModel          `tfsdk:"splunk_attributes"`
	AWSKinesisFirehoseAttrs    *DataStreamAWSKinesisFirehoseOutputModel `tfsdk:"aws_kinesis_firehose_attributes"`
	DatadogAttributes          *DataStreamURLAPIKeyOutputModel          `tfsdk:"datadog_attributes"`
	QRadarAttributes           *DataStreamQRadarOutputModel             `tfsdk:"qradar_attributes"`
	AzureMonitorAttributes     *DataStreamAzureMonitorOutputModel       `tfsdk:"azure_monitor_attributes"`
	AzureBlobStorageAttributes *DataStreamAzureBlobStorageOutputModel   `tfsdk:"azure_blob_storage_attributes"`
}

// DataStreamStandardOutputModel configures the `standard` (HTTP POST) endpoint.
type DataStreamStandardOutputModel struct {
	URL              types.String            `tfsdk:"url"`
	Headers          map[string]types.String `tfsdk:"headers"`
	LogLineSeparator types.String            `tfsdk:"log_line_separator"`
	PayloadFormat    types.String            `tfsdk:"payload_format"`
	MaxSize          types.Int64             `tfsdk:"max_size"`
}

// DataStreamKafkaOutputModel configures the `kafka` endpoint.
type DataStreamKafkaOutputModel struct {
	BootstrapServers types.String `tfsdk:"bootstrap_servers"`
	KafkaTopic       types.String `tfsdk:"kafka_topic"`
	UseTLS           types.Bool   `tfsdk:"use_tls"`
}

// DataStreamS3OutputModel configures the `s3` endpoint.
type DataStreamS3OutputModel struct {
	AccessKey       types.String `tfsdk:"access_key"`
	SecretKey       types.String `tfsdk:"secret_key"`
	Region          types.String `tfsdk:"region"`
	ObjectKeyPrefix types.String `tfsdk:"object_key_prefix"`
	BucketName      types.String `tfsdk:"bucket_name"`
	ContentType     types.String `tfsdk:"content_type"`
	HostURL         types.String `tfsdk:"host_url"`
}

// DataStreamBigQueryOutputModel configures the `big_query` endpoint.
type DataStreamBigQueryOutputModel struct {
	DatasetID         types.String `tfsdk:"dataset_id"`
	ProjectID         types.String `tfsdk:"project_id"`
	TableID           types.String `tfsdk:"table_id"`
	ServiceAccountKey types.String `tfsdk:"service_account_key"`
}

// DataStreamURLAPIKeyOutputModel configures the `elasticsearch`, `splunk` and
// `datadog` endpoints, which all take a URL plus an API key.
type DataStreamURLAPIKeyOutputModel struct {
	URL    types.String `tfsdk:"url"`
	APIKey types.String `tfsdk:"api_key"`
}

// DataStreamAWSKinesisFirehoseOutputModel configures the `aws_kinesis_firehose` endpoint.
type DataStreamAWSKinesisFirehoseOutputModel struct {
	AccessKey  types.String `tfsdk:"access_key"`
	StreamName types.String `tfsdk:"stream_name"`
	Region     types.String `tfsdk:"region"`
	SecretKey  types.String `tfsdk:"secret_key"`
}

// DataStreamQRadarOutputModel configures the `qradar` endpoint.
type DataStreamQRadarOutputModel struct {
	URL types.String `tfsdk:"url"`
}

// DataStreamAzureMonitorOutputModel configures the `azure_monitor` endpoint.
type DataStreamAzureMonitorOutputModel struct {
	LogType            types.String `tfsdk:"log_type"`
	SharedKey          types.String `tfsdk:"shared_key"`
	TimeGeneratedField types.String `tfsdk:"time_generated_field"`
	WorkspaceID        types.String `tfsdk:"workspace_id"`
}

// DataStreamAzureBlobStorageOutputModel configures the `azure_blob_storage` endpoint.
type DataStreamAzureBlobStorageOutputModel struct {
	StorageAccount types.String `tfsdk:"storage_account"`
	ContainerName  types.String `tfsdk:"container_name"`
	BlobSasToken   types.String `tfsdk:"blob_sas_token"`
}

func (r *dataStreamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_stream"
}

func (r *dataStreamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Data Stream. A stream pipes logs from one or more inputs through a transform pipeline into one or more outputs (endpoints).",
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
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"data_stream": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The data stream identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the data stream.",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the data stream.",
						Optional:    true,
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the data stream.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the data stream.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the data stream.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the data stream.",
						Computed:    true,
					},
					"inputs": schema.ListNestedAttribute{
						Description: "Input feeding the stream. The API stores a single input per stream: it silently keeps only one " +
							"of a longer list, so more than one entry is rejected here rather than losing them on apply.",
						Required: true,
						Validators: []validator.List{
							listvalidator.SizeBetween(1, 1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Type of the input. Supported value: `raw_logs`.",
									Required:    true,
								},
								"attributes": schema.SingleNestedAttribute{
									Description: "Attributes of the input.",
									Required:    true,
									Attributes: map[string]schema.Attribute{
										"data_source": schema.StringAttribute{
											Description: "Source of the logs. One of `workloads`, `waf`, `functions_console`, `activity_history` — " +
												"the slugs served by the API's `GET /data_stream/data_sources` endpoint. The API silently " +
												"rewrites an unrecognized slug instead of rejecting it, so it is validated here.",
											Required: true,
											Validators: []validator.String{
												stringvalidator.OneOf("workloads", "waf", "functions_console", "activity_history"),
											},
										},
									},
								},
							},
						},
					},
					"transform": schema.ListNestedAttribute{
						Description: "Transforms applied to the records. Exactly one `*_attributes` block must be set per entry, matching `type`. " +
							"At least one entry is required: the API rejects an empty list. It also requires a `render_template` entry, " +
							"plus either a `sampling` or a `filter_workloads` entry. The API normalizes the pipeline order, so the order " +
							"the entries are listed in here does not change how they are applied.",
						Required: true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Type of the transform. One of `sampling`, `filter_workloads`, `render_template`.",
									Required:    true,
								},
								"sampling_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `sampling` transform.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"rate": schema.Int64Attribute{
											Description: "Percentage of records to keep.",
											Required:    true,
										},
									},
								},
								"filter_workloads_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `filter_workloads` transform.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"workloads": schema.ListAttribute{
											Description: "Identifiers of the workloads whose logs are kept.",
											Required:    true,
											ElementType: types.Int64Type,
										},
									},
								},
								"render_template_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `render_template` transform.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"template": schema.Int64Attribute{
											Description: "Identifier of the data stream template used to render the payload.",
											Required:    true,
										},
									},
								},
							},
						},
					},
					"outputs": schema.ListNestedAttribute{
						Description: "Endpoint the records are delivered to. Exactly one `*_attributes` block must be set, matching `type`. " +
							"The API stores a single output per stream: given a longer list it keeps only the last entry (and answers 500 " +
							"when the entries have differing types), so more than one is rejected here rather than losing them on apply.",
						Required: true,
						Validators: []validator.List{
							listvalidator.SizeBetween(1, 1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"type": schema.StringAttribute{
									Description: "Type of the endpoint. One of `standard`, `kafka`, `s3`, `big_query`, `elasticsearch`, `splunk`, `aws_kinesis_firehose`, `datadog`, `qradar`, `azure_monitor`, `azure_blob_storage`.",
									Required:    true,
								},
								"standard_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `standard` (HTTP/HTTPS POST) endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"url": schema.StringAttribute{
											Description: "Destination URL.",
											Required:    true,
										},
										"headers": schema.MapAttribute{
											Description: "Headers sent with each request.",
											Required:    true,
											ElementType: types.StringType,
											Sensitive:   true,
										},
										"log_line_separator": schema.StringAttribute{
											Description: "Separator inserted between records in the payload. The API trims surrounding " +
												"whitespace from this value, so a newline separator has to be written as the two-character " +
												"escape `\\n` rather than a literal newline — an all-whitespace value would be stored as empty.",
											Optional: true,
											Computed: true,
											Validators: []validator.String{
												stringvalidator.RegexMatches(
													regexp.MustCompile(`(?s)^(\S(.*\S)?)?$`),
													"must not start or end with whitespace: the API trims it, so the stored value would "+
														"differ from the configuration. Write a newline separator as \"\\n\" (backslash n).",
												),
											},
										},
										"payload_format": schema.StringAttribute{
											Description: "Format of the payload, for example `$dataset`.",
											Optional:    true,
											Computed:    true,
										},
										"max_size": schema.Int64Attribute{
											Description: "Maximum payload size, in bytes.",
											Optional:    true,
											Computed:    true,
										},
									},
								},
								"kafka_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `kafka` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"bootstrap_servers": schema.StringAttribute{
											Description: "Comma-separated list of Kafka bootstrap servers.",
											Required:    true,
										},
										"kafka_topic": schema.StringAttribute{
											Description: "Kafka topic the records are published to.",
											Required:    true,
										},
										"use_tls": schema.BoolAttribute{
											Description: "Whether TLS is used to connect to the brokers.",
											Required:    true,
										},
									},
								},
								"s3_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `s3` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"access_key": schema.StringAttribute{
											Description: "Access key of the S3-compatible service.",
											Required:    true,
											Sensitive:   true,
										},
										"secret_key": schema.StringAttribute{
											Description: "Secret key of the S3-compatible service.",
											Required:    true,
											Sensitive:   true,
										},
										"region": schema.StringAttribute{
											Description: "Region of the bucket.",
											Required:    true,
										},
										"object_key_prefix": schema.StringAttribute{
											Description: "Prefix prepended to the object keys.",
											Optional:    true,
										},
										"bucket_name": schema.StringAttribute{
											Description: "Name of the bucket.",
											Required:    true,
										},
										"content_type": schema.StringAttribute{
											Description: "Content type of the uploaded objects. One of `plain/text`, `application/gzip`.",
											Required:    true,
										},
										"host_url": schema.StringAttribute{
											Description: "Host URL of the S3-compatible service.",
											Required:    true,
										},
									},
								},
								"big_query_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `big_query` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"dataset_id": schema.StringAttribute{
											Description: "BigQuery dataset identifier.",
											Required:    true,
										},
										"project_id": schema.StringAttribute{
											Description: "Google Cloud project identifier.",
											Required:    true,
										},
										"table_id": schema.StringAttribute{
											Description: "BigQuery table identifier.",
											Required:    true,
										},
										"service_account_key": schema.StringAttribute{
											Description: "Service account key, as a JSON string.",
											Required:    true,
											Sensitive:   true,
										},
									},
								},
								"elasticsearch_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `elasticsearch` endpoint.",
									Optional:    true,
									Attributes:  dataStreamURLAPIKeyAttributes("Elasticsearch"),
								},
								"splunk_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `splunk` endpoint.",
									Optional:    true,
									Attributes:  dataStreamURLAPIKeyAttributes("Splunk"),
								},
								"aws_kinesis_firehose_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `aws_kinesis_firehose` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"access_key": schema.StringAttribute{
											Description: "AWS access key.",
											Required:    true,
											Sensitive:   true,
										},
										"stream_name": schema.StringAttribute{
											Description: "Name of the Kinesis Data Firehose delivery stream.",
											Required:    true,
										},
										"region": schema.StringAttribute{
											Description: "AWS region of the delivery stream.",
											Required:    true,
										},
										"secret_key": schema.StringAttribute{
											Description: "AWS secret key.",
											Required:    true,
											Sensitive:   true,
										},
									},
								},
								"datadog_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `datadog` endpoint.",
									Optional:    true,
									Attributes:  dataStreamURLAPIKeyAttributes("Datadog"),
								},
								"qradar_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `qradar` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"url": schema.StringAttribute{
											Description: "IBM QRadar HTTP receiver URL.",
											Required:    true,
										},
									},
								},
								"azure_monitor_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `azure_monitor` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"log_type": schema.StringAttribute{
											Description: "Record type of the data being submitted.",
											Required:    true,
										},
										"shared_key": schema.StringAttribute{
											Description: "Primary or secondary key of the workspace.",
											Required:    true,
											Sensitive:   true,
										},
										"time_generated_field": schema.StringAttribute{
											Description: "Name of the field used as the event timestamp.",
											Optional:    true,
										},
										"workspace_id": schema.StringAttribute{
											Description: "Log Analytics workspace identifier.",
											Required:    true,
										},
									},
								},
								"azure_blob_storage_attributes": schema.SingleNestedAttribute{
									Description: "Attributes for the `azure_blob_storage` endpoint.",
									Optional:    true,
									Attributes: map[string]schema.Attribute{
										"storage_account": schema.StringAttribute{
											Description: "Name of the storage account.",
											Required:    true,
										},
										"container_name": schema.StringAttribute{
											Description: "Name of the blob container.",
											Required:    true,
										},
										"blob_sas_token": schema.StringAttribute{
											Description: "Shared access signature token of the container.",
											Required:    true,
											Sensitive:   true,
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

// dataStreamURLAPIKeyAttributes builds the schema shared by the endpoints that
// take only a URL and an API key.
func dataStreamURLAPIKeyAttributes(vendor string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Description: fmt.Sprintf("%s endpoint URL.", vendor),
			Required:    true,
		},
		"api_key": schema.StringAttribute{
			Description: fmt.Sprintf("%s API key.", vendor),
			Required:    true,
			Sensitive:   true,
		},
	}
}

func (r *dataStreamResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *dataStreamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dataStreamResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataStreamReq, err := buildDataStreamRequest(plan.DataStream)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "Failed to build data stream request")
		return
	}

	createDataStream, response, err := r.client.api.DataStreamStreamsAPI.CreateDataStream(ctx).DataStreamRequest(dataStreamReq).Execute() //nolint
	if response != nil {
		defer func(r *http.Response) { _ = r.Body.Close() }(response)
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			createDataStream, response, err = utils.RetryOn429(func() (*azionapi.DataStreamResponse, *http.Response, error) {
				return r.client.api.DataStreamStreamsAPI.CreateDataStream(ctx).DataStreamRequest(dataStreamReq).Execute()
			}, 5)
			if response != nil {
				defer func(r *http.Response) { _ = r.Body.Close() }(response)
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	populateDataStreamFromResponse(plan.DataStream, createDataStream.GetData())
	plan.ID = types.StringValue(strconv.FormatInt(plan.DataStream.ID.ValueInt64(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dataStreamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dataStreamResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var streamID int64
	var err error
	if state.DataStream != nil && !state.DataStream.ID.IsNull() {
		streamID = state.DataStream.ID.ValueInt64()
	} else {
		streamID, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Data Stream ID",
			)
			return
		}
	}

	getDataStream, response, err := r.client.api.DataStreamStreamsAPI.RetrieveDataStream(ctx, streamID).Execute() //nolint
	if response != nil {
		defer func(r *http.Response) { _ = r.Body.Close() }(response)
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getDataStream, response, err = utils.RetryOn429(func() (*azionapi.DataStreamResponse, *http.Response, error) {
				return r.client.api.DataStreamStreamsAPI.RetrieveDataStream(ctx, streamID).Execute()
			}, 5)
			if response != nil {
				defer func(r *http.Response) { _ = r.Body.Close() }(response)
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	populateDataStreamFromResponse(state.DataStream, getDataStream.GetData())
	state.ID = types.StringValue(strconv.FormatInt(state.DataStream.ID.ValueInt64(), 10))
	state.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dataStreamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dataStreamResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dataStreamResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	streamID := state.DataStream.ID.ValueInt64()

	// PUT is used instead of PATCH because PatchedDataStreamRequest cannot carry
	// `outputs`, so a partial update could never change the endpoints.
	dataStreamReq, err := buildDataStreamRequest(plan.DataStream)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "Failed to build data stream update request")
		return
	}

	updateDataStream, response, err := r.client.api.DataStreamStreamsAPI.UpdateDataStream(ctx, streamID).DataStreamRequest(dataStreamReq).Execute() //nolint
	if response != nil {
		defer func(r *http.Response) { _ = r.Body.Close() }(response)
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			updateDataStream, response, err = utils.RetryOn429(func() (*azionapi.DataStreamResponse, *http.Response, error) {
				return r.client.api.DataStreamStreamsAPI.UpdateDataStream(ctx, streamID).DataStreamRequest(dataStreamReq).Execute()
			}, 5)
			if response != nil {
				defer func(r *http.Response) { _ = r.Body.Close() }(response)
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	populateDataStreamFromResponse(plan.DataStream, updateDataStream.GetData())
	plan.ID = types.StringValue(strconv.FormatInt(plan.DataStream.ID.ValueInt64(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dataStreamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dataStreamResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	streamID := state.DataStream.ID.ValueInt64()

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.DataStreamStreamsAPI.DeleteDataStream(ctx, streamID).Execute()
	}, 5)
	if response != nil {
		defer func(r *http.Response) { _ = r.Body.Close() }(response)
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
		addDataStreamAPIError(&resp.Diagnostics, err, response)
		return
	}
}

func (r *dataStreamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	streamID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse data stream ID: %s", req.ID),
		)
		return
	}

	getDataStream, response, err := r.client.api.DataStreamStreamsAPI.RetrieveDataStream(ctx, streamID).Execute() //nolint
	if response != nil {
		defer func(r *http.Response) { _ = r.Body.Close() }(response)
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getDataStream, response, err = utils.RetryOn429(func() (*azionapi.DataStreamResponse, *http.Response, error) {
				return r.client.api.DataStreamStreamsAPI.RetrieveDataStream(ctx, streamID).Execute()
			}, 5)
			if response != nil {
				defer func(r *http.Response) { _ = r.Body.Close() }(response)
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addDataStreamAPIError(&resp.Diagnostics, err, response)
			return
		}
	}

	state := &dataStreamResourceModel{
		DataStream: &dataStreamResourceResults{},
	}
	populateDataStreamFromResponse(state.DataStream, getDataStream.GetData())
	state.ID = types.StringValue(strconv.FormatInt(streamID, 10))
	state.SchemaVersion = types.Int64Value(0)

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// buildDataStreamRequest converts the Terraform plan into the SDK request body.
func buildDataStreamRequest(stream *dataStreamResourceResults) (azionapi.DataStreamRequest, error) {
	if stream == nil {
		return azionapi.DataStreamRequest{}, fmt.Errorf("data_stream block is required")
	}

	inputs, err := buildDataStreamInputRequests(stream.Inputs)
	if err != nil {
		return azionapi.DataStreamRequest{}, err
	}

	transforms, err := buildDataStreamTransformRequests(stream.Transform)
	if err != nil {
		return azionapi.DataStreamRequest{}, err
	}

	outputs, err := buildDataStreamOutputRequests(stream.Outputs)
	if err != nil {
		return azionapi.DataStreamRequest{}, err
	}

	streamReq := azionapi.NewDataStreamRequest(stream.Name.ValueString(), inputs, transforms, outputs)
	if !stream.Active.IsNull() && !stream.Active.IsUnknown() {
		streamReq.SetActive(stream.Active.ValueBool())
	}

	return *streamReq, nil
}

func buildDataStreamInputRequests(inputs []DataStreamInputModel) ([]azionapi.InputInputDataSourceAttributesRequest, error) {
	// Start from an empty slice so an unset list marshals as `[]` rather than `null`.
	out := []azionapi.InputInputDataSourceAttributesRequest{}
	for i, input := range inputs {
		if input.Attributes == nil {
			return nil, fmt.Errorf("inputs[%d]: attributes is required", i)
		}
		attrs := azionapi.NewInputDataSourceRequest(input.Attributes.DataSource.ValueString())
		out = append(out, *azionapi.NewInputInputDataSourceAttributesRequest(input.Type.ValueString(), *attrs))
	}
	return out, nil
}

func buildDataStreamTransformRequests(transforms []DataStreamTransformModel) ([]azionapi.TransformRequest, error) {
	out := []azionapi.TransformRequest{}
	for i, transform := range transforms {
		transformType := transform.Type.ValueString()
		switch transformType {
		case "sampling":
			if transform.SamplingAttributes == nil {
				return nil, fmt.Errorf("transform[%d]: sampling_attributes is required when type is 'sampling'", i)
			}
			attrs := azionapi.NewTransformSamplingRequest(transform.SamplingAttributes.Rate.ValueInt64())
			item := azionapi.NewTransformTransformSamplingAttributesRequest(transformType, *attrs)
			out = append(out, azionapi.TransformTransformSamplingAttributesRequestAsTransformRequest(item))

		case "filter_workloads":
			if transform.FilterWorkloadsAttributes == nil {
				return nil, fmt.Errorf("transform[%d]: filter_workloads_attributes is required when type is 'filter_workloads'", i)
			}
			workloads := []int64{}
			for _, workload := range transform.FilterWorkloadsAttributes.Workloads {
				workloads = append(workloads, workload.ValueInt64())
			}
			attrs := azionapi.NewTransformFilterWorkloadsRequest(workloads)
			item := azionapi.NewTransformTransformFilterWorkloadsAttributesRequest(transformType, *attrs)
			out = append(out, azionapi.TransformTransformFilterWorkloadsAttributesRequestAsTransformRequest(item))

		case "render_template":
			if transform.RenderTemplateAttributes == nil {
				return nil, fmt.Errorf("transform[%d]: render_template_attributes is required when type is 'render_template'", i)
			}
			attrs := azionapi.NewTransformRenderTemplateRequest(transform.RenderTemplateAttributes.Template.ValueInt64())
			item := azionapi.NewTransformTransformRenderTemplateAttributesRequest(transformType, *attrs)
			out = append(out, azionapi.TransformTransformRenderTemplateAttributesRequestAsTransformRequest(item))

		default:
			return nil, fmt.Errorf("transform[%d]: unsupported transform type %q. Supported types are: sampling, filter_workloads, render_template", i, transformType)
		}
	}
	return out, nil
}

func buildDataStreamOutputRequests(outputs []DataStreamOutputModel) ([]azionapi.OutputRequest, error) {
	out := []azionapi.OutputRequest{}
	for i, output := range outputs {
		outputType := output.Type.ValueString()
		request, err := buildDataStreamOutputRequest(i, outputType, output)
		if err != nil {
			return nil, err
		}
		out = append(out, request)
	}
	return out, nil
}

// buildDataStreamOutputRequest builds one entry of the outputs list. The SDK models
// each endpoint as a `{type, attributes}` variant of the OutputRequest `oneOf`, so the
// discriminator is set on the variant rather than on a shared wrapper.
//
//nolint:gocyclo // One branch per endpoint type; splitting it only adds indirection.
func buildDataStreamOutputRequest(index int, outputType string, output DataStreamOutputModel) (azionapi.OutputRequest, error) {
	switch outputType {
	case "standard":
		if output.StandardAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "standard_attributes")
		}
		attrs := output.StandardAttributes
		headers := map[string]string{}
		for key, value := range attrs.Headers {
			headers[key] = value.ValueString()
		}
		endpoint := azionapi.NewHttpPostEndpointRequest(attrs.URL.ValueString(), headers)
		if !attrs.LogLineSeparator.IsNull() && !attrs.LogLineSeparator.IsUnknown() {
			endpoint.SetLogLineSeparator(attrs.LogLineSeparator.ValueString())
		}
		if !attrs.PayloadFormat.IsNull() && !attrs.PayloadFormat.IsUnknown() {
			endpoint.SetPayloadFormat(attrs.PayloadFormat.ValueString())
		}
		if !attrs.MaxSize.IsNull() && !attrs.MaxSize.IsUnknown() {
			endpoint.SetMaxSize(attrs.MaxSize.ValueInt64())
		}
		item := azionapi.NewHttpPostEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.HttpPostEndpointAttributesRequestAsOutputRequest(item), nil

	case "kafka":
		if output.KafkaAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "kafka_attributes")
		}
		attrs := output.KafkaAttributes
		endpoint := azionapi.NewKafkaEndpointRequest(
			attrs.BootstrapServers.ValueString(),
			attrs.KafkaTopic.ValueString(),
			attrs.UseTLS.ValueBool(),
		)
		item := azionapi.NewKafkaEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.KafkaEndpointAttributesRequestAsOutputRequest(item), nil

	case "s3":
		if output.S3Attributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "s3_attributes")
		}
		attrs := output.S3Attributes
		endpoint := azionapi.NewS3EndpointRequest(
			attrs.AccessKey.ValueString(),
			attrs.SecretKey.ValueString(),
			attrs.Region.ValueString(),
			attrs.BucketName.ValueString(),
			attrs.ContentType.ValueString(),
			attrs.HostURL.ValueString(),
		)
		if !attrs.ObjectKeyPrefix.IsNull() && !attrs.ObjectKeyPrefix.IsUnknown() {
			endpoint.SetObjectKeyPrefix(attrs.ObjectKeyPrefix.ValueString())
		}
		item := azionapi.NewS3EndpointAttributesRequest(outputType, *endpoint)
		return azionapi.S3EndpointAttributesRequestAsOutputRequest(item), nil

	case "big_query":
		if output.BigQueryAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "big_query_attributes")
		}
		attrs := output.BigQueryAttributes
		endpoint := azionapi.NewBigQueryEndpointRequest(
			attrs.DatasetID.ValueString(),
			attrs.ProjectID.ValueString(),
			attrs.TableID.ValueString(),
			attrs.ServiceAccountKey.ValueString(),
		)
		item := azionapi.NewBigQueryEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.BigQueryEndpointAttributesRequestAsOutputRequest(item), nil

	case "elasticsearch":
		if output.ElasticsearchAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "elasticsearch_attributes")
		}
		attrs := output.ElasticsearchAttributes
		endpoint := azionapi.NewElasticsearchEndpointRequest(attrs.URL.ValueString(), attrs.APIKey.ValueString())
		item := azionapi.NewElasticsearchEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.ElasticsearchEndpointAttributesRequestAsOutputRequest(item), nil

	case "splunk":
		if output.SplunkAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "splunk_attributes")
		}
		attrs := output.SplunkAttributes
		endpoint := azionapi.NewSplunkEndpointRequest(attrs.URL.ValueString(), attrs.APIKey.ValueString())
		item := azionapi.NewSplunkEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.SplunkEndpointAttributesRequestAsOutputRequest(item), nil

	case "aws_kinesis_firehose":
		if output.AWSKinesisFirehoseAttrs == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "aws_kinesis_firehose_attributes")
		}
		attrs := output.AWSKinesisFirehoseAttrs
		endpoint := azionapi.NewAWSKinesisFirehoseEndpointRequest(
			attrs.AccessKey.ValueString(),
			attrs.StreamName.ValueString(),
			attrs.Region.ValueString(),
			attrs.SecretKey.ValueString(),
		)
		item := azionapi.NewAWSKinesisFirehoseEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.AWSKinesisFirehoseEndpointAttributesRequestAsOutputRequest(item), nil

	case "datadog":
		if output.DatadogAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "datadog_attributes")
		}
		attrs := output.DatadogAttributes
		endpoint := azionapi.NewDatadogEndpointRequest(attrs.URL.ValueString(), attrs.APIKey.ValueString())
		item := azionapi.NewDatadogEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.DatadogEndpointAttributesRequestAsOutputRequest(item), nil

	case "qradar":
		if output.QRadarAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "qradar_attributes")
		}
		endpoint := azionapi.NewQRadarEndpointRequest(output.QRadarAttributes.URL.ValueString())
		item := azionapi.NewQRadarEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.QRadarEndpointAttributesRequestAsOutputRequest(item), nil

	case "azure_monitor":
		if output.AzureMonitorAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "azure_monitor_attributes")
		}
		attrs := output.AzureMonitorAttributes
		endpoint := azionapi.NewAzureMonitorEndpointRequest(
			attrs.LogType.ValueString(),
			attrs.SharedKey.ValueString(),
			attrs.WorkspaceID.ValueString(),
		)
		if !attrs.TimeGeneratedField.IsNull() && !attrs.TimeGeneratedField.IsUnknown() {
			endpoint.SetTimeGeneratedField(attrs.TimeGeneratedField.ValueString())
		}
		item := azionapi.NewAzureMonitorEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.AzureMonitorEndpointAttributesRequestAsOutputRequest(item), nil

	case "azure_blob_storage":
		if output.AzureBlobStorageAttributes == nil {
			return azionapi.OutputRequest{}, missingOutputAttrsErr(index, outputType, "azure_blob_storage_attributes")
		}
		attrs := output.AzureBlobStorageAttributes
		endpoint := azionapi.NewAzureBlobStorageEndpointRequest(
			attrs.StorageAccount.ValueString(),
			attrs.ContainerName.ValueString(),
			attrs.BlobSasToken.ValueString(),
		)
		item := azionapi.NewAzureBlobStorageEndpointAttributesRequest(outputType, *endpoint)
		return azionapi.AzureBlobStorageEndpointAttributesRequestAsOutputRequest(item), nil

	default:
		return azionapi.OutputRequest{}, fmt.Errorf(
			"outputs[%d]: unsupported endpoint type %q. Supported types are: standard, kafka, s3, big_query, elasticsearch, splunk, aws_kinesis_firehose, datadog, qradar, azure_monitor, azure_blob_storage",
			index, outputType)
	}
}

func missingOutputAttrsErr(index int, outputType, attrName string) error {
	return fmt.Errorf("outputs[%d]: %s is required when type is %q", index, attrName, outputType)
}

// populateDataStreamFromResponse refreshes the model with the API response.
// Secrets and optional endpoint fields are seeded from the prior configuration so
// masked or defaulted echoes don't show up as perpetual drift.
func populateDataStreamFromResponse(model *dataStreamResourceResults, stream azionapi.DataStream) {
	if model == nil {
		return
	}

	priorOutputs := model.Outputs
	priorTransforms := model.Transform

	model.ID = types.Int64Value(stream.GetId())
	model.Name = types.StringValue(stream.GetName())
	model.Active = types.BoolPointerValue(stream.Active)
	model.LastEditor = types.StringValue(stream.GetLastEditor())
	model.CreatedAt = types.StringValue(stream.GetCreated().Format(time.RFC850))
	model.LastModified = types.StringValue(stream.GetLastModified().Format(time.RFC850))
	model.ProductVersion = types.StringValue(stream.GetProductVersion())
	model.Inputs = populateDataStreamInputs(stream.GetInputs())
	model.Transform = populateDataStreamTransforms(priorTransforms, stream.GetTransform())
	model.Outputs = populateDataStreamOutputs(priorOutputs, stream.GetOutputs())
}

func populateDataStreamInputs(inputs []azionapi.InputInputDataSourceAttributes) []DataStreamInputModel {
	var out []DataStreamInputModel
	for _, input := range inputs {
		out = append(out, DataStreamInputModel{
			Type: types.StringValue(input.GetType()),
			Attributes: &DataStreamInputAttributesModel{
				DataSource: types.StringValue(input.Attributes.GetDataSource()),
			},
		})
	}
	return out
}

// populateDataStreamTransforms maps the response transforms back onto the model.
// The API stores the pipeline in its own canonical order regardless of the order it
// was submitted in, so entries are re-ordered to follow the configured list whenever
// both hold the same types. Without that, any config not already written in the
// API's order would fail with "Provider produced inconsistent result after apply".
func populateDataStreamTransforms(prior []DataStreamTransformModel, transforms []azionapi.Transform) []DataStreamTransformModel {
	var out []DataStreamTransformModel
	for i := range transforms {
		actual := transforms[i].GetActualInstance()
		if actual == nil {
			continue
		}
		switch t := actual.(type) {
		case *azionapi.TransformTransformSamplingAttributes:
			out = append(out, DataStreamTransformModel{
				Type: types.StringValue(t.GetType()),
				SamplingAttributes: &DataStreamSamplingAttributesModel{
					Rate: types.Int64Value(t.Attributes.GetRate()),
				},
			})
		case *azionapi.TransformTransformFilterWorkloadsAttributes:
			var workloads []types.Int64
			for _, workload := range t.Attributes.GetWorkloads() {
				workloads = append(workloads, types.Int64Value(workload))
			}
			out = append(out, DataStreamTransformModel{
				Type: types.StringValue(t.GetType()),
				FilterWorkloadsAttributes: &DataStreamFilterWorkloadsAttributesModel{
					Workloads: workloads,
				},
			})
		case *azionapi.TransformTransformRenderTemplateAttributes:
			out = append(out, DataStreamTransformModel{
				Type: types.StringValue(t.GetType()),
				RenderTemplateAttributes: &DataStreamRenderTemplateAttributesModel{
					Template: types.Int64Value(t.Attributes.GetTemplate()),
				},
			})
		}
	}
	return orderTransformsLikePrior(prior, out)
}

// orderTransformsLikePrior re-orders actual to match prior's sequence of transform
// types. It bails out and returns the API's order untouched when the two lists don't
// hold the same types, so a genuine remote change still shows up as drift.
func orderTransformsLikePrior(prior, actual []DataStreamTransformModel) []DataStreamTransformModel {
	if len(prior) == 0 || len(prior) != len(actual) {
		return actual
	}

	remaining := make([]DataStreamTransformModel, len(actual))
	copy(remaining, actual)

	ordered := make([]DataStreamTransformModel, 0, len(actual))
	for _, want := range prior {
		match := -1
		for i := range remaining {
			if remaining[i].Type.Equal(want.Type) {
				match = i
				break
			}
		}
		if match < 0 {
			return actual
		}
		ordered = append(ordered, remaining[match])
		remaining = append(remaining[:match], remaining[match+1:]...)
	}
	return ordered
}

// dataStreamOutputEndpoint unwraps a response output into its endpoint type and
// attributes. Each variant of the Output `oneOf` carries its own discriminator, so
// there is no shared accessor to read them from.
func dataStreamOutputEndpoint(output azionapi.Output) (string, interface{}) {
	switch e := output.GetActualInstance().(type) {
	case *azionapi.HttpPostEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.KafkaEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.S3EndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.BigQueryEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.ElasticsearchEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.SplunkEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.AWSKinesisFirehoseEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.DatadogEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.QRadarEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.AzureMonitorEndpointAttributes:
		return e.GetType(), &e.Attributes
	case *azionapi.AzureBlobStorageEndpointAttributes:
		return e.GetType(), &e.Attributes
	}
	return "", nil
}

//nolint:gocyclo // One branch per endpoint type; splitting it only adds indirection.
func populateDataStreamOutputs(prior []DataStreamOutputModel, outputs []azionapi.Output) []DataStreamOutputModel {
	var out []DataStreamOutputModel
	for i := range outputs {
		outputType, actual := dataStreamOutputEndpoint(outputs[i])

		// Prior state is matched positionally; a type change at the same index is
		// treated as a fresh entry so stale secrets aren't carried over.
		var priorOutput *DataStreamOutputModel
		if i < len(prior) && prior[i].Type.ValueString() == outputType {
			priorOutput = &prior[i]
		}

		model := DataStreamOutputModel{Type: types.StringValue(outputType)}

		if actual == nil {
			out = append(out, model)
			continue
		}

		switch e := actual.(type) {
		case *azionapi.HttpPostEndpoint:
			var priorAttrs *DataStreamStandardOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.StandardAttributes
			}
			attrs := &DataStreamStandardOutputModel{URL: types.StringValue(e.GetUrl())}
			if priorAttrs != nil {
				// Headers usually come back redacted, so the configured map wins.
				attrs.Headers = priorAttrs.Headers
			} else {
				attrs.Headers = map[string]types.String{}
				for key, value := range e.GetHeaders() {
					attrs.Headers[key] = types.StringValue(value)
				}
			}
			if shouldPopulate(priorAttrs, func(p *DataStreamStandardOutputModel) bool { return !p.LogLineSeparator.IsNull() }) {
				attrs.LogLineSeparator = types.StringPointerValue(e.LogLineSeparator)
			} else {
				attrs.LogLineSeparator = priorAttrs.LogLineSeparator
			}
			if shouldPopulate(priorAttrs, func(p *DataStreamStandardOutputModel) bool { return !p.PayloadFormat.IsNull() }) {
				attrs.PayloadFormat = types.StringPointerValue(e.PayloadFormat)
			} else {
				attrs.PayloadFormat = priorAttrs.PayloadFormat
			}
			if shouldPopulate(priorAttrs, func(p *DataStreamStandardOutputModel) bool { return !p.MaxSize.IsNull() }) {
				attrs.MaxSize = types.Int64PointerValue(e.MaxSize.Get())
			} else {
				attrs.MaxSize = priorAttrs.MaxSize
			}
			model.StandardAttributes = attrs

		case *azionapi.KafkaEndpoint:
			model.KafkaAttributes = &DataStreamKafkaOutputModel{
				BootstrapServers: types.StringValue(e.GetBootstrapServers()),
				KafkaTopic:       types.StringValue(e.GetKafkaTopic()),
				UseTLS:           types.BoolValue(e.GetUseTls()),
			}

		case *azionapi.S3Endpoint:
			var priorAttrs *DataStreamS3OutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.S3Attributes
			}
			attrs := &DataStreamS3OutputModel{
				Region:      types.StringValue(e.GetRegion()),
				BucketName:  types.StringValue(e.GetBucketName()),
				ContentType: types.StringValue(e.GetContentType()),
				HostURL:     types.StringValue(e.GetHostUrl()),
				AccessKey:   preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.AccessKey }, e.GetAccessKey()),
				SecretKey:   preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.SecretKey }, e.GetSecretKey()),
			}
			if shouldPopulate(priorAttrs, func(p *DataStreamS3OutputModel) bool { return !p.ObjectKeyPrefix.IsNull() }) {
				attrs.ObjectKeyPrefix = types.StringPointerValue(e.ObjectKeyPrefix.Get())
			} else {
				attrs.ObjectKeyPrefix = priorAttrs.ObjectKeyPrefix
			}
			model.S3Attributes = attrs

		case *azionapi.BigQueryEndpoint:
			var priorAttrs *DataStreamBigQueryOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.BigQueryAttributes
			}
			model.BigQueryAttributes = &DataStreamBigQueryOutputModel{
				DatasetID:         types.StringValue(e.GetDatasetId()),
				ProjectID:         types.StringValue(e.GetProjectId()),
				TableID:           types.StringValue(e.GetTableId()),
				ServiceAccountKey: preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.ServiceAccountKey }, e.GetServiceAccountKey()),
			}

		case *azionapi.ElasticsearchEndpoint:
			var priorAttrs *DataStreamURLAPIKeyOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.ElasticsearchAttributes
			}
			model.ElasticsearchAttributes = populateURLAPIKeyOutput(priorAttrs, e.GetUrl(), e.GetApiKey())

		case *azionapi.SplunkEndpoint:
			var priorAttrs *DataStreamURLAPIKeyOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.SplunkAttributes
			}
			model.SplunkAttributes = populateURLAPIKeyOutput(priorAttrs, e.GetUrl(), e.GetApiKey())

		case *azionapi.DatadogEndpoint:
			var priorAttrs *DataStreamURLAPIKeyOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.DatadogAttributes
			}
			model.DatadogAttributes = populateURLAPIKeyOutput(priorAttrs, e.GetUrl(), e.GetApiKey())

		case *azionapi.AWSKinesisFirehoseEndpoint:
			var priorAttrs *DataStreamAWSKinesisFirehoseOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.AWSKinesisFirehoseAttrs
			}
			model.AWSKinesisFirehoseAttrs = &DataStreamAWSKinesisFirehoseOutputModel{
				StreamName: types.StringValue(e.GetStreamName()),
				Region:     types.StringValue(e.GetRegion()),
				AccessKey:  preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.AccessKey }, e.GetAccessKey()),
				SecretKey:  preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.SecretKey }, e.GetSecretKey()),
			}

		case *azionapi.QRadarEndpoint:
			model.QRadarAttributes = &DataStreamQRadarOutputModel{
				URL: types.StringValue(e.GetUrl()),
			}

		case *azionapi.AzureMonitorEndpoint:
			var priorAttrs *DataStreamAzureMonitorOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.AzureMonitorAttributes
			}
			attrs := &DataStreamAzureMonitorOutputModel{
				LogType:     types.StringValue(e.GetLogType()),
				WorkspaceID: types.StringValue(e.GetWorkspaceId()),
				SharedKey:   preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.SharedKey }, e.GetSharedKey()),
			}
			if shouldPopulate(priorAttrs, func(p *DataStreamAzureMonitorOutputModel) bool { return !p.TimeGeneratedField.IsNull() }) {
				attrs.TimeGeneratedField = types.StringPointerValue(e.TimeGeneratedField.Get())
			} else {
				attrs.TimeGeneratedField = priorAttrs.TimeGeneratedField
			}
			model.AzureMonitorAttributes = attrs

		case *azionapi.AzureBlobStorageEndpoint:
			var priorAttrs *DataStreamAzureBlobStorageOutputModel
			if priorOutput != nil {
				priorAttrs = priorOutput.AzureBlobStorageAttributes
			}
			model.AzureBlobStorageAttributes = &DataStreamAzureBlobStorageOutputModel{
				StorageAccount: types.StringValue(e.GetStorageAccount()),
				ContainerName:  types.StringValue(e.GetContainerName()),
				BlobSasToken:   preferPriorSecret(priorAttrs != nil, func() types.String { return priorAttrs.BlobSasToken }, e.GetBlobSasToken()),
			}
		}

		out = append(out, model)
	}
	return out
}

func populateURLAPIKeyOutput(prior *DataStreamURLAPIKeyOutputModel, url, apiKey string) *DataStreamURLAPIKeyOutputModel {
	return &DataStreamURLAPIKeyOutputModel{
		URL:    types.StringValue(url),
		APIKey: preferPriorSecret(prior != nil, func() types.String { return prior.APIKey }, apiKey),
	}
}

// preferPriorSecret keeps the configured value for write-only fields the API
// returns masked. It falls back to the API value on import, where no prior exists.
func preferPriorSecret(hasPrior bool, prior func() types.String, apiValue string) types.String {
	if hasPrior {
		if value := prior(); !value.IsNull() && !value.IsUnknown() {
			return value
		}
	}
	return types.StringValue(apiValue)
}

// addDataStreamAPIError adds an appropriate error to diagnostics based on the API response.
func addDataStreamAPIError(diagnostics *diag.Diagnostics, err error, response *http.Response) {
	if response == nil {
		diagnostics.AddError(err.Error(), "No response received")
		return
	}

	if response.StatusCode == http.StatusTooManyRequests {
		diagnostics.AddError(err.Error(), "API request rate limited")
		return
	}

	bodyBytes, errReadAll := io.ReadAll(response.Body)
	if errReadAll != nil {
		diagnostics.AddError(errReadAll.Error(), "err")
		return
	}
	diagnostics.AddError(err.Error(), string(bodyBytes))
}
