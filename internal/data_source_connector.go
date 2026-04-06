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
	_ datasource.DataSource              = &ConnectorDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectorDataSource{}
)

func dataSourceAzionConnector() datasource.DataSource {
	return &ConnectorDataSource{}
}

type ConnectorDataSource struct {
	client *apiClient
}

type ConnectorDataSourceModel struct {
	Data ConnectorResults `tfsdk:"data"`
	ID   types.String     `tfsdk:"id"`
}

type ConnectorResults struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	LastEditor     types.String `tfsdk:"last_editor"`
	LastModified   types.String `tfsdk:"last_modified"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ProductVersion types.String `tfsdk:"product_version"`
	Active         types.Bool   `tfsdk:"active"`
	Type           types.String `tfsdk:"type"`
	Attributes     types.String `tfsdk:"attributes"`
}

func (d *ConnectorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *ConnectorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector"
}

func (d *ConnectorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Required:    true,
			},
			"data": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The connector identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the connector.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the connector.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the connector.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the connector.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the connector.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the connector.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the connector (http, storage, live_ingest).",
						Computed:    true,
					},
					"attributes": schema.StringAttribute{
						Description: "Attributes of the connector as JSON string. Structure varies by type: storage has bucket and prefix; http has addresses, connection_options, and modules.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *ConnectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getConnectorId types.String
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &getConnectorId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorID, err := strconv.ParseInt(getConnectorId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	connectorResponse, response, err := d.client.api.ConnectorsAPI.
		RetrieveConnector(ctx, connectorID).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			connectorResponse, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
				return d.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorID).Execute() //nolint
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
			usrMsg, errMsg := errPrintConnector(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	connectorState, err := populateConnectorResults(ctx, connectorResponse.GetData())
	if err != nil {
		resp.Diagnostics.AddError(
			err.Error(),
			"Failed to populate connector results",
		)
		return
	}

	connectorState.ID = types.StringValue("Get By Id Connector")
	diags = resp.State.Set(ctx, &connectorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func populateConnectorResults(_ context.Context, connector azionapi.Connector) (ConnectorDataSourceModel, error) {
	connectorState := ConnectorDataSourceModel{}

	// Get the actual connector instance.
	actualConnector := connector.GetActualInstance()
	if actualConnector == nil {
		return connectorState, fmt.Errorf("no connector data found")
	}

	// Handle different connector types.
	switch c := actualConnector.(type) {
	case *azionapi.ConnectorBase:
		// Storage connector.
		connectorState.Data = ConnectorResults{
			ID:             types.Int64Value(c.Id),
			Name:           types.StringValue(c.Name),
			LastEditor:     types.StringValue(c.LastEditor),
			LastModified:   types.StringValue(c.LastModified.Format(time.RFC850)),
			CreatedAt:      types.StringValue(c.CreatedAt.Format(time.RFC850)),
			ProductVersion: types.StringValue(c.ProductVersion),
			Type:           types.StringValue(c.Type),
			Active:         types.BoolPointerValue(c.Active),
		}

		// Marshal attributes to JSON string.
		attrsJSON, err := json.Marshal(c.Attributes)
		if err != nil {
			return connectorState, fmt.Errorf("failed to marshal storage attributes: %w", err)
		}
		connectorState.Data.Attributes = types.StringValue(string(attrsJSON))

	case *azionapi.ConnectorHTTP:
		// HTTP connector.
		connectorState.Data = ConnectorResults{
			ID:             types.Int64Value(c.Id),
			Name:           types.StringValue(c.Name),
			LastEditor:     types.StringValue(c.LastEditor),
			LastModified:   types.StringValue(c.LastModified.Format(time.RFC850)),
			CreatedAt:      types.StringValue(c.CreatedAt.Format(time.RFC850)),
			ProductVersion: types.StringValue(c.ProductVersion),
			Type:           types.StringValue(c.Type),
			Active:         types.BoolPointerValue(c.Active),
		}

		// Marshal attributes to JSON string.
		attrsJSON, err := json.Marshal(c.Attributes)
		if err != nil {
			return connectorState, fmt.Errorf("failed to marshal HTTP attributes: %w", err)
		}
		connectorState.Data.Attributes = types.StringValue(string(attrsJSON))
	}

	return connectorState, nil
}

// errPrintConnector returns user-friendly error messages for connector operations.
func errPrintConnector(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Connector found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
