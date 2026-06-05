package provider

import (
	"context"
	"encoding/json"
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
	_ datasource.DataSource              = &ConnectorsDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectorsDataSource{}
)

func dataSourceAzionConnectors() datasource.DataSource {
	return &ConnectorsDataSource{}
}

type ConnectorsDataSource struct {
	client *apiClient
}

type ConnectorsDataSourceModel struct {
	Counter types.Int64         `tfsdk:"counter"`
	Results []ConnectorsResults `tfsdk:"results"`
	ID      types.String        `tfsdk:"id"`
}

type ConnectorsResults struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	LastEditor     types.String `tfsdk:"last_editor"`
	LastModified   types.String `tfsdk:"last_modified"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ProductVersion types.String `tfsdk:"product_version"`
	Active         types.Bool   `tfsdk:"active"`
	Type           types.String `tfsdk:"type"`
	IsVersioned    types.Bool   `tfsdk:"is_versioned"`
	Version        types.Int64  `tfsdk:"version"`
	VersionState   types.String `tfsdk:"version_state"`
	VersionID      types.String `tfsdk:"version_id"`
	Attributes     types.String `tfsdk:"attributes"`
}

func (d *ConnectorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*apiClient)
}

func (d *ConnectorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connectors"
}

func (d *ConnectorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"counter": schema.Int64Attribute{
				Description: "The total count of connectors.",
				Computed:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
							Description: "Type of the connector (http, storage).",
							Computed:    true,
						},
						"is_versioned": schema.BoolAttribute{
							Description: "Whether the connector is versioned.",
							Computed:    true,
						},
						"version": schema.Int64Attribute{
							Description: "The current version of the connector.",
							Computed:    true,
						},
						"version_state": schema.StringAttribute{
							Description: "The state of the current connector version.",
							Computed:    true,
						},
						"version_id": schema.StringAttribute{
							Description: "The identifier of the current connector version.",
							Computed:    true,
						},
						"attributes": schema.StringAttribute{
							Description: "Attributes of the connector as JSON string. Structure varies by type.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ConnectorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	connectorsResponse, response, err := d.client.api.ConnectorsAPI.ListConnectors(ctx).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			connectorsResponse, response, err = utils.RetryOn429(func() (*azionapi.PaginatedConnectorList, *http.Response, error) {
				return d.client.api.ConnectorsAPI.ListConnectors(ctx).Execute() //nolint
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
			usrMsg, errMsg := errPrintConnectors(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if response != nil {
		defer response.Body.Close()
	}

	connectorsState := ConnectorsDataSourceModel{}

	if connectorsResponse.Count != nil {
		connectorsState.Counter = types.Int64Value(*connectorsResponse.Count)
	}

	for _, connector := range connectorsResponse.GetResults() {
		result, err := populateConnectorsResults(connector)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to populate connector result",
			)
			return
		}
		connectorsState.Results = append(connectorsState.Results, result)
	}

	connectorsState.ID = types.StringValue("Get All Connectors")
	diags := resp.State.Set(ctx, &connectorsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func populateConnectorsResults(connector azionapi.Connector) (ConnectorsResults, error) {
	result := ConnectorsResults{}

	// Get the actual connector instance.
	actualConnector := connector.GetActualInstance()
	if actualConnector == nil {
		return result, fmt.Errorf("no connector data found")
	}

	// Handle different connector types.
	switch c := actualConnector.(type) {
	case *azionapi.ConnectorStorage:
		// Storage connector.
		result = ConnectorsResults{
			ID:             types.Int64Value(c.Id),
			Name:           types.StringValue(c.Name),
			LastEditor:     types.StringValue(c.LastEditor),
			LastModified:   types.StringValue(c.LastModified.Format(time.RFC850)),
			CreatedAt:      types.StringValue(c.CreatedAt.Format(time.RFC850)),
			ProductVersion: types.StringValue(c.ProductVersion),
			Type:           types.StringValue(c.Type),
			Active:         types.BoolPointerValue(c.Active),
			IsVersioned:    types.BoolValue(c.IsVersioned),
			Version:        types.Int64PointerValue(c.Version.Get()),
			VersionState:   types.StringPointerValue(c.VersionState.Get()),
			VersionID:      types.StringPointerValue(c.VersionId.Get()),
		}

		// Marshal attributes to JSON string.
		attrsJSON, err := json.Marshal(c.Attributes)
		if err != nil {
			return result, fmt.Errorf("failed to marshal storage attributes: %w", err)
		}
		result.Attributes = types.StringValue(string(attrsJSON))

	case *azionapi.ConnectorHTTP:
		// HTTP connector.
		result = ConnectorsResults{
			ID:             types.Int64Value(c.Id),
			Name:           types.StringValue(c.Name),
			LastEditor:     types.StringValue(c.LastEditor),
			LastModified:   types.StringValue(c.LastModified.Format(time.RFC850)),
			CreatedAt:      types.StringValue(c.CreatedAt.Format(time.RFC850)),
			ProductVersion: types.StringValue(c.ProductVersion),
			Type:           types.StringValue(c.Type),
			Active:         types.BoolPointerValue(c.Active),
			IsVersioned:    types.BoolValue(c.IsVersioned),
			Version:        types.Int64PointerValue(c.Version.Get()),
			VersionState:   types.StringPointerValue(c.VersionState.Get()),
			VersionID:      types.StringPointerValue(c.VersionId.Get()),
		}

		// Marshal attributes to JSON string.
		attrsJSON, err := json.Marshal(c.Attributes)
		if err != nil {
			return result, fmt.Errorf("failed to marshal HTTP attributes: %w", err)
		}
		result.Attributes = types.StringValue(string(attrsJSON))
	}

	return result, nil
}

// errPrintConnectors returns user-friendly error messages for connectors operations.
func errPrintConnectors(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Connectors found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
