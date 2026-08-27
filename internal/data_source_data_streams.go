package provider

import (
	"context"
	"fmt"
	"net/http"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &DataStreamsDataSource{}
	_ datasource.DataSourceWithConfigure = &DataStreamsDataSource{}
)

func dataSourceAzionDataStreams() datasource.DataSource {
	return &DataStreamsDataSource{}
}

type DataStreamsDataSource struct {
	client *apiClient
}

type DataStreamsDataSourceModel struct {
	Counter types.Int64         `tfsdk:"counter"`
	Results []DataStreamResults `tfsdk:"results"`
	ID      types.String        `tfsdk:"id"`
}

func (d *DataStreamsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DataStreamsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_streams"
}

func (d *DataStreamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of data streams.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: dataStreamDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *DataStreamsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	dataStreamsResponse, response, err := d.client.api.DataStreamStreamsAPI.ListDataStreams(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			dataStreamsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedDataStreamList, *http.Response, error) {
				return d.client.api.DataStreamStreamsAPI.ListDataStreams(ctx).Execute() //nolint
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
			usrMsg, errMsg := errPrintDataStreams(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	dataStreamsState := DataStreamsDataSourceModel{}

	if dataStreamsResponse.Count != nil {
		dataStreamsState.Counter = types.Int64Value(*dataStreamsResponse.Count)
	}

	for _, stream := range dataStreamsResponse.GetResults() {
		result, err := populateDataStreamDataSourceResults(stream)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to populate data stream result",
			)
			return
		}
		dataStreamsState.Results = append(dataStreamsState.Results, result)
	}

	dataStreamsState.ID = types.StringValue("Get All Data Streams")
	diags := resp.State.Set(ctx, &dataStreamsState)
	resp.Diagnostics.Append(diags...)
}

// errPrintDataStreams returns user-friendly error messages for data streams operations.
func errPrintDataStreams(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Data Streams found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
