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
	_ datasource.DataSource              = &DataStreamTemplatesDataSource{}
	_ datasource.DataSourceWithConfigure = &DataStreamTemplatesDataSource{}
)

func dataSourceAzionDataStreamTemplates() datasource.DataSource {
	return &DataStreamTemplatesDataSource{}
}

type DataStreamTemplatesDataSource struct {
	client *apiClient
}

type DataStreamTemplatesDataSourceModel struct {
	Counter types.Int64                 `tfsdk:"counter"`
	Results []DataStreamTemplateResults `tfsdk:"results"`
	ID      types.String                `tfsdk:"id"`
}

func (d *DataStreamTemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DataStreamTemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_stream_templates"
}

func (d *DataStreamTemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of data stream templates.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: dataStreamTemplateDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *DataStreamTemplatesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	templatesResponse, response, err := d.client.api.DataStreamTemplatesAPI.ListTemplates(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			templatesResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedTemplateList, *http.Response, error) {
				return d.client.api.DataStreamTemplatesAPI.ListTemplates(ctx).Execute() //nolint
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
			usrMsg, errMsg := errPrintDataStreamTemplates(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	templatesState := DataStreamTemplatesDataSourceModel{}

	if templatesResponse.Count != nil {
		templatesState.Counter = types.Int64Value(*templatesResponse.Count)
	}

	for _, template := range templatesResponse.GetResults() {
		templatesState.Results = append(templatesState.Results, populateDataStreamTemplateDataSourceResults(template))
	}

	templatesState.ID = types.StringValue("Get All Data Stream Templates")
	diags := resp.State.Set(ctx, &templatesState)
	resp.Diagnostics.Append(diags...)
}

// errPrintDataStreamTemplates returns user-friendly error messages for data stream templates operations.
func errPrintDataStreamTemplates(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Data Stream Templates found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
