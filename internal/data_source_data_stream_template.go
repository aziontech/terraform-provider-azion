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
	_ datasource.DataSource              = &DataStreamTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &DataStreamTemplateDataSource{}
)

func dataSourceAzionDataStreamTemplate() datasource.DataSource {
	return &DataStreamTemplateDataSource{}
}

type DataStreamTemplateDataSource struct {
	client *apiClient
}

type DataStreamTemplateDataSourceModel struct {
	Data DataStreamTemplateResults `tfsdk:"data"`
	ID   types.String              `tfsdk:"id"`
}

type DataStreamTemplateResults struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Active       types.Bool   `tfsdk:"active"`
	DataSet      types.String `tfsdk:"data_set"`
	Custom       types.Bool   `tfsdk:"custom"`
	LastEditor   types.String `tfsdk:"last_editor"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (d *DataStreamTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *DataStreamTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_stream_template"
}

func (d *DataStreamTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data stream template.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed:   true,
				Attributes: dataStreamTemplateDataSourceAttributes(),
			},
		},
	}
}

// dataStreamTemplateDataSourceAttributes is shared by the singular and plural data sources.
func dataStreamTemplateDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Description: "The template identifier.",
			Computed:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name of the template.",
			Computed:    true,
		},
		"active": schema.BoolAttribute{
			Description: "Status of the template.",
			Computed:    true,
		},
		"data_set": schema.StringAttribute{
			Description: "The payload template, holding the record layout with `$variable` placeholders.",
			Computed:    true,
		},
		"custom": schema.BoolAttribute{
			Description: "Whether the template is user-defined. Azion's built-in templates are false.",
			Computed:    true,
		},
		"last_editor": schema.StringAttribute{
			Description: "The last editor of the template.",
			Computed:    true,
		},
		"created_at": schema.StringAttribute{
			Description: "The creation timestamp of the template.",
			Computed:    true,
		},
		"last_modified": schema.StringAttribute{
			Description: "Last modified timestamp of the template.",
			Computed:    true,
		},
	}
}

func (d *DataStreamTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getTemplateID types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getTemplateID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	templateID, err := strconv.ParseInt(getTemplateID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	templateResponse, response, err := d.client.api.DataStreamTemplatesAPI.
		RetrieveTemplate(ctx, templateID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			templateResponse, response, err = utils.RetryOn429(func() (*azionapi.TemplateResponse, *http.Response, error) {
				return d.client.api.DataStreamTemplatesAPI.RetrieveTemplate(ctx, templateID).Execute() //nolint
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
			usrMsg, errMsg := errPrintDataStreamTemplate(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	templateState := DataStreamTemplateDataSourceModel{
		Data: populateDataStreamTemplateDataSourceResults(templateResponse.GetData()),
		ID:   types.StringValue("Get By Id Data Stream Template"),
	}

	diags = resp.State.Set(ctx, &templateState)
	resp.Diagnostics.Append(diags...)
}

// populateDataStreamTemplateDataSourceResults flattens a template into the data source model.
func populateDataStreamTemplateDataSourceResults(template azionapi.Template) DataStreamTemplateResults {
	return DataStreamTemplateResults{
		ID:           types.Int64Value(template.GetId()),
		Name:         types.StringValue(template.GetName()),
		Active:       types.BoolPointerValue(template.Active),
		DataSet:      types.StringValue(template.GetDataSet()),
		Custom:       types.BoolValue(template.GetCustom()),
		LastEditor:   types.StringValue(template.GetLastEditor()),
		CreatedAt:    types.StringValue(template.GetCreatedAt().Format(time.RFC850)),
		LastModified: types.StringValue(template.GetLastModified().Format(time.RFC850)),
	}
}

// errPrintDataStreamTemplate returns user-friendly error messages for data stream template operations.
func errPrintDataStreamTemplate(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Data Stream Template found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
