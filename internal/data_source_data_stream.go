package provider

import (
	"context"
	"encoding/json"
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
	_ datasource.DataSource              = &DataStreamDataSource{}
	_ datasource.DataSourceWithConfigure = &DataStreamDataSource{}
)

func dataSourceAzionDataStream() datasource.DataSource {
	return &DataStreamDataSource{}
}

type DataStreamDataSource struct {
	client *apiClient
}

type DataStreamDataSourceModel struct {
	Data DataStreamResults `tfsdk:"data"`
	ID   types.String      `tfsdk:"id"`
}

type DataStreamResults struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Active         types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	CreatedAt      types.String `tfsdk:"created_at"`
	LastModified   types.String `tfsdk:"last_modified"`
	ProductVersion types.String `tfsdk:"product_version"`
	Inputs         types.String `tfsdk:"inputs"`
	Transform      types.String `tfsdk:"transform"`
	Outputs        types.String `tfsdk:"outputs"`
}

func (d *DataStreamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DataStreamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_stream"
}

func (d *DataStreamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data stream.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed:   true,
				Attributes: dataStreamDataSourceAttributes(),
			},
		},
	}
}

// dataStreamDataSourceAttributes is shared by the singular and plural data sources.
// The polymorphic inputs, transform and outputs lists are exposed as JSON strings
// so every endpoint and transform type is readable without a per-type schema.
func dataStreamDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Description: "The data stream identifier.",
			Computed:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name of the data stream.",
			Computed:    true,
		},
		"active": schema.BoolAttribute{
			Description: "Status of the data stream.",
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
		"inputs": schema.StringAttribute{
			Description: "Inputs of the data stream as a JSON string. Each entry has a `type` and `attributes.data_source`.",
			Computed:    true,
		},
		"transform": schema.StringAttribute{
			Description: "Transforms of the data stream as a JSON string. Structure varies by `type` (sampling, filter_workloads, render_template).",
			Computed:    true,
		},
		"outputs": schema.StringAttribute{
			Description: "Outputs of the data stream as a JSON string. Structure varies by `type` (standard, kafka, s3, big_query, elasticsearch, splunk, aws_kinesis_firehose, datadog, qradar, azure_monitor, azure_blob_storage).",
			Computed:    true,
		},
	}
}

func (d *DataStreamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getDataStreamID types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getDataStreamID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	streamID, err := strconv.ParseInt(getDataStreamID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	dataStreamResponse, response, err := d.client.api.DataStreamStreamsAPI.
		RetrieveDataStream(ctx, streamID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			dataStreamResponse, response, err = utils.RetryOn429(func() (*azionapi.DataStreamResponse, *http.Response, error) {
				return d.client.api.DataStreamStreamsAPI.RetrieveDataStream(ctx, streamID).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer func(r *http.Response) { _ = r.Body.Close() }(response)
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errPrintDataStream(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer func(r *http.Response) { _ = r.Body.Close() }(response)
	}

	result, err := populateDataStreamDataSourceResults(dataStreamResponse.GetData())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"Failed to populate data stream results",
		)
		return
	}

	dataStreamState := DataStreamDataSourceModel{
		Data: result,
		ID:   types.StringValue("Get By Id Data Stream"),
	}

	diags = resp.State.Set(ctx, &dataStreamState)
	resp.Diagnostics.Append(diags...)
}

// populateDataStreamDataSourceResults flattens a data stream, marshalling the
// polymorphic lists to JSON strings.
func populateDataStreamDataSourceResults(stream azionapi.DataStream) (DataStreamResults, error) {
	result := DataStreamResults{
		ID:             types.Int64Value(stream.GetId()),
		Name:           types.StringValue(stream.GetName()),
		Active:         types.BoolPointerValue(stream.Active),
		LastEditor:     types.StringValue(stream.GetLastEditor()),
		CreatedAt:      types.StringValue(stream.GetCreated().Format(time.RFC850)),
		LastModified:   types.StringValue(stream.GetLastModified().Format(time.RFC850)),
		ProductVersion: types.StringValue(stream.GetProductVersion()),
	}

	inputsJSON, err := json.Marshal(stream.GetInputs())
	if err != nil {
		return result, fmt.Errorf("failed to marshal data stream inputs: %w", err)
	}
	result.Inputs = types.StringValue(string(inputsJSON))

	transformJSON, err := json.Marshal(stream.GetTransform())
	if err != nil {
		return result, fmt.Errorf("failed to marshal data stream transform: %w", err)
	}
	result.Transform = types.StringValue(string(transformJSON))

	outputsJSON, err := json.Marshal(stream.GetOutputs())
	if err != nil {
		return result, fmt.Errorf("failed to marshal data stream outputs: %w", err)
	}
	result.Outputs = types.StringValue(string(outputsJSON))

	return result, nil
}

// errPrintDataStream returns user-friendly error messages for data stream operations.
func errPrintDataStream(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Data Stream found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
