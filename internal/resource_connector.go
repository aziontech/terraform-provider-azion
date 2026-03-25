package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &connectorResource{}
	_ resource.ResourceWithConfigure   = &connectorResource{}
	_ resource.ResourceWithImportState = &connectorResource{}
)

func NewConnectorResource() resource.Resource {
	return &connectorResource{}
}

type connectorResource struct {
	client *apiClient
}

// Main resource model
type connectorResourceModel struct {
	Connector     *connectorResourceResults `tfsdk:"connector"`
	ID            types.String              `tfsdk:"id"`
	LastUpdated   types.String              `tfsdk:"last_updated"`
	SchemaVersion types.Int64               `tfsdk:"schema_version"`
}

// Connector results - all fields including type-specific attributes
type connectorResourceResults struct {
	ID             types.Int64             `tfsdk:"id"`
	Name           types.String            `tfsdk:"name"`
	LastEditor     types.String            `tfsdk:"last_editor"`
	LastModified   types.String            `tfsdk:"last_modified"`
	ProductVersion types.String            `tfsdk:"product_version"`
	Active         types.Bool              `tfsdk:"active"`
	Type           types.String            `tfsdk:"type"`
	StorageAttrs   *StorageAttributesModel `tfsdk:"storage_attributes"`
	HTTPAttrs      *HTTPAttributesModel    `tfsdk:"http_attributes"`
}

// Storage connector attributes
type StorageAttributesModel struct {
	Bucket types.String `tfsdk:"bucket"`
	Prefix types.String `tfsdk:"prefix"`
}

// HTTP connector attributes
type HTTPAttributesModel struct {
	Addresses         []AddressModel `tfsdk:"addresses"`
	ConnectionOptions types.Object   `tfsdk:"connection_options"`
	Modules           types.Object   `tfsdk:"modules"`
}

// Address model for HTTP connectors
type AddressModel struct {
	Address   types.String         `tfsdk:"address"`
	Active    types.Bool           `tfsdk:"active"`
	HTTPPort  types.Int64          `tfsdk:"http_port"`
	HTTPSPort types.Int64          `tfsdk:"https_port"`
	Modules   *AddressModulesModel `tfsdk:"modules"`
}

// Address modules
type AddressModulesModel struct {
	LoadBalancer *AddressLoadBalancerModel `tfsdk:"load_balancer"`
}

// Address load balancer module - uses server_role and weight
type AddressLoadBalancerModel struct {
	ServerRole types.String `tfsdk:"server_role"`
	Weight     types.Int64  `tfsdk:"weight"`
}

// HTTP connection options
type HTTPConnectionOptionsModel struct {
	DNSResolution     types.String `tfsdk:"dns_resolution"`
	FollowingRedirect types.Bool   `tfsdk:"following_redirect"`
	Host              types.String `tfsdk:"host"`
	HTTPVersionPolicy types.String `tfsdk:"http_version_policy"`
	PathPrefix        types.String `tfsdk:"path_prefix"`
	RealIPHeader      types.String `tfsdk:"real_ip_header"`
	RealPortHeader    types.String `tfsdk:"real_port_header"`
	TransportPolicy   types.String `tfsdk:"transport_policy"`
}

// HTTP modules
type HTTPModulesModel struct {
	LoadBalancer types.Object `tfsdk:"load_balancer"`
	OriginShield types.Object `tfsdk:"origin_shield"`
}

// Load balancer module
type LoadBalancerModuleModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

// Origin shield module
type OriginShieldModuleModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func (r *connectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector"
}

func (r *connectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a connector resource. Connectors are polymorphic and support different types (http, storage).",
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
			"connector": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The connector identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the connector.",
						Required:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last editor of the connector.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Last modified timestamp of the connector.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the connector.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Status of the connector.",
						Optional:    true,
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the connector (http or storage).",
						Required:    true,
					},
					"storage_attributes": schema.SingleNestedAttribute{
						Description: "Attributes for storage type connectors. Required when type is 'storage'.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"bucket": schema.StringAttribute{
								Description: "The name of the bucket.",
								Required:    true,
							},
							"prefix": schema.StringAttribute{
								Description: "The prefix path within the bucket.",
								Optional:    true,
							},
						},
					},
					"http_attributes": schema.SingleNestedAttribute{
						Description: "Attributes for HTTP type connectors. Required when type is 'http'.",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"addresses": schema.ListNestedAttribute{
								Description: "List of origin addresses.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"address": schema.StringAttribute{
											Description: "The origin address (IP or hostname).",
											Required:    true,
										},
										"active": schema.BoolAttribute{
											Description: "Whether the address is active.",
											Optional:    true,
											Computed:    true,
										},
										"http_port": schema.Int64Attribute{
											Description: "HTTP port number.",
											Optional:    true,
											Computed:    true,
										},
										"https_port": schema.Int64Attribute{
											Description: "HTTPS port number.",
											Optional:    true,
											Computed:    true,
										},
										"modules": schema.SingleNestedAttribute{
											Description: "Address-level modules.",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"load_balancer": schema.SingleNestedAttribute{
													Description: "Load balancer module at address level.",
													Optional:    true,
													Attributes: map[string]schema.Attribute{
														"server_role": schema.StringAttribute{
															Description: "Role of the address in load balancing (primary or backup).",
															Optional:    true,
														},
														"weight": schema.Int64Attribute{
															Description: "Weight used in load balancing strategy.",
															Optional:    true,
														},
													},
												},
											},
										},
									},
								},
							},
							"connection_options": schema.SingleNestedAttribute{
								Description: "HTTP connection options. API provides defaults if not specified.",
								Optional:    true,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"dns_resolution": schema.StringAttribute{
										Description: "DNS resolution strategy.",
										Optional:    true,
										Computed:    true,
									},
									"following_redirect": schema.BoolAttribute{
										Description: "Whether to follow redirects.",
										Optional:    true,
										Computed:    true,
									},
									"host": schema.StringAttribute{
										Description: "Host header value. Use ${host} to pass through the original host.",
										Optional:    true,
										Computed:    true,
									},
									"http_version_policy": schema.StringAttribute{
										Description: "HTTP version policy.",
										Optional:    true,
										Computed:    true,
									},
									"path_prefix": schema.StringAttribute{
										Description: "Path prefix for requests.",
										Optional:    true,
										Computed:    true,
									},
									"real_ip_header": schema.StringAttribute{
										Description: "Header for real IP.",
										Optional:    true,
										Computed:    true,
									},
									"real_port_header": schema.StringAttribute{
										Description: "Header for real port.",
										Optional:    true,
										Computed:    true,
									},
									"transport_policy": schema.StringAttribute{
										Description: "Transport policy.",
										Optional:    true,
										Computed:    true,
									},
								},
							},
							"modules": schema.SingleNestedAttribute{
								Description: "HTTP modules configuration. API provides defaults if not specified.",
								Optional:    true,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"load_balancer": schema.SingleNestedAttribute{
										Description: "Load balancer module.",
										Optional:    true,
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Description: "Whether load balancer is enabled.",
												Optional:    true,
												Computed:    true,
											},
										},
									},
									"origin_shield": schema.SingleNestedAttribute{
										Description: "Origin shield module.",
										Optional:    true,
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Description: "Whether origin shield is enabled.",
												Optional:    true,
												Computed:    true,
											},
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

func (r *connectorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *connectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorType := plan.Connector.Type.ValueString()
	var connectorId int64

	// Build the appropriate request based on connector type.
	switch connectorType {
	case "storage":
		connectorReq, err := buildStorageConnectorRequest(plan.Connector)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to build storage connector request",
			)
			return
		}
		createConnector, response, err := r.client.api.ConnectorsAPI.CreateConnector(ctx).ConnectorRequest(connectorReq).Execute() //nolint
		if err != nil {
			addConnectorAPIError(&resp.Diagnostics, err, response, "create")
			return
		}
		if response != nil {
			defer response.Body.Close()
		}
		connectorId = getConnectorId(createConnector.GetData())

	case "http":
		connectorReq, err := r.buildHTTPConnectorRequest(ctx, plan.Connector)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to build HTTP connector request",
			)
			return
		}
		createConnector, response, err := r.client.api.ConnectorsAPI.CreateConnector(ctx).ConnectorRequest(connectorReq).Execute() //nolint
		if err != nil {
			addConnectorAPIError(&resp.Diagnostics, err, response, "create")
			return
		}
		if response != nil {
			defer response.Body.Close()
		}
		connectorId = getConnectorId(createConnector.GetData())

	default:
		resp.Diagnostics.AddError(
			"Invalid connector type",
			fmt.Sprintf("Unsupported connector type: %s. Supported types are: storage, http", connectorType),
		)
		return
	}

	// Read the connector back to ensure we have the complete state with all API defaults.
	getConnector, response, err := r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute() //nolint
	if err != nil {
		addConnectorAPIError(&resp.Diagnostics, err, response, "read after create")
		return
	}
	if response != nil {
		defer response.Body.Close()
	}

	r.populateConnectorFromResponse(ctx, plan.Connector, getConnector.GetData())
	plan.ID = types.StringValue(strconv.FormatInt(plan.Connector.ID.ValueInt64(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// getConnectorId extracts the ID from a polymorphic Connector response.
func getConnectorId(connector azionapi.Connector) int64 {
	actualConnector := connector.GetActualInstance()
	if actualConnector == nil {
		return 0
	}

	switch c := actualConnector.(type) {
	case *azionapi.ConnectorBase:
		return c.Id
	case *azionapi.ConnectorHTTP:
		return c.Id
	default:
		return 0
	}
}

func (r *connectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var connectorId int64
	var err error
	if state.Connector != nil {
		connectorId = state.Connector.ID.ValueInt64()
	} else {
		connectorId, err = strconv.ParseInt(state.ID.ValueString(), 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Connector ID",
			)
			return
		}
	}

	getConnector, response, err := r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		addConnectorAPIError(&resp.Diagnostics, err, response, "read")
		return
	}

	if response != nil {
		defer response.Body.Close()
	}

	r.populateConnectorFromResponse(ctx, state.Connector, getConnector.GetData())
	state.ID = types.StringValue(strconv.FormatInt(state.Connector.ID.ValueInt64(), 10))
	state.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *connectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state connectorResourceModel
	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorId := state.Connector.ID.ValueInt64()
	connectorType := plan.Connector.Type.ValueString()

	// Build and send the appropriate update request based on connector type.
	switch connectorType {
	case "storage":
		connectorReq, err := buildStoragePatchedConnectorRequest(plan.Connector)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to build storage connector update request",
			)
			return
		}
		updateConnector, response, err := r.client.api.ConnectorsAPI.PartialUpdateConnector(ctx, connectorId).PatchedConnectorRequest(connectorReq).Execute() //nolint
		if err != nil {
			addConnectorAPIError(&resp.Diagnostics, err, response, "update")
			return
		}
		if response != nil {
			defer response.Body.Close()
		}
		r.populateConnectorFromResponse(ctx, plan.Connector, updateConnector.GetData())

	case "http":
		connectorReq, err := r.buildHTTPPatchedConnectorRequest(ctx, plan.Connector)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
				"Failed to build HTTP connector update request",
			)
			return
		}
		updateConnector, response, err := r.client.api.ConnectorsAPI.PartialUpdateConnector(ctx, connectorId).PatchedConnectorRequest(connectorReq).Execute() //nolint
		if err != nil {
			addConnectorAPIError(&resp.Diagnostics, err, response, "update")
			return
		}
		if response != nil {
			defer response.Body.Close()
		}
		r.populateConnectorFromResponse(ctx, plan.Connector, updateConnector.GetData())

	default:
		resp.Diagnostics.AddError(
			"Invalid connector type",
			fmt.Sprintf("Unsupported connector type: %s. Supported types are: storage, http", connectorType),
		)
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(plan.Connector.ID.ValueInt64(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.SchemaVersion = types.Int64Value(0)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *connectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectorId := state.Connector.ID.ValueInt64()

	_, response, err := r.client.api.ConnectorsAPI.DeleteConnector(ctx, connectorId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			return
		}
		addConnectorAPIError(&resp.Diagnostics, err, response, "delete")
		return
	}

	if response != nil {
		defer response.Body.Close()
	}
}

func (r *connectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	connectorId, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse connector ID: %s", req.ID),
		)
		return
	}

	// First, get the connector from API to determine its type
	getConnector, response, err := r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute()
	if err != nil {
		addConnectorAPIError(&resp.Diagnostics, err, response, "import")
		return
	}
	if response != nil {
		defer response.Body.Close()
	}

	// Create the model and populate it from response
	state := &connectorResourceModel{
		Connector: &connectorResourceResults{},
	}
	r.populateConnectorFromResponse(ctx, state.Connector, getConnector.GetData())
	state.ID = types.StringValue(strconv.FormatInt(connectorId, 10))
	state.SchemaVersion = types.Int64Value(0)

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Helper functions for building requests.

func buildStorageConnectorRequest(connector *connectorResourceResults) (azionapi.ConnectorRequest, error) {
	if connector.StorageAttrs == nil {
		return azionapi.ConnectorRequest{}, fmt.Errorf("storage_attributes is required for storage type connectors")
	}

	attrs := azionapi.ConnectorStorageAttributesRequest{
		Bucket: connector.StorageAttrs.Bucket.ValueString(),
	}

	if !connector.StorageAttrs.Prefix.IsNull() && !connector.StorageAttrs.Prefix.IsUnknown() {
		attrs.SetPrefix(connector.StorageAttrs.Prefix.ValueString())
	}

	req := azionapi.NewConnectorRequestBase(
		connector.Name.ValueString(),
		connector.Type.ValueString(),
		attrs,
	)

	if !connector.Active.IsNull() && !connector.Active.IsUnknown() {
		req.SetActive(connector.Active.ValueBool())
	}

	return azionapi.ConnectorRequestBaseAsConnectorRequest(req), nil
}

func (r *connectorResource) buildHTTPConnectorRequest(ctx context.Context, connector *connectorResourceResults) (azionapi.ConnectorRequest, error) {
	if connector.HTTPAttrs == nil {
		return azionapi.ConnectorRequest{}, fmt.Errorf("http_attributes is required for http type connectors")
	}

	// Build addresses
	var addresses []azionapi.AddressRequest
	for _, addr := range connector.HTTPAttrs.Addresses {
		address := azionapi.NewAddressRequest(addr.Address.ValueString())
		if !addr.Active.IsNull() && !addr.Active.IsUnknown() {
			address.SetActive(addr.Active.ValueBool())
		}
		if !addr.HTTPPort.IsNull() && !addr.HTTPPort.IsUnknown() {
			address.SetHttpPort(addr.HTTPPort.ValueInt64())
		}
		if !addr.HTTPSPort.IsNull() && !addr.HTTPSPort.IsUnknown() {
			address.SetHttpsPort(addr.HTTPSPort.ValueInt64())
		}
		if addr.Modules != nil && addr.Modules.LoadBalancer != nil {
			lb := azionapi.NewAddressLoadBalancerModuleRequest()
			if !addr.Modules.LoadBalancer.ServerRole.IsNull() && !addr.Modules.LoadBalancer.ServerRole.IsUnknown() {
				lb.SetServerRole(addr.Modules.LoadBalancer.ServerRole.ValueString())
			}
			if !addr.Modules.LoadBalancer.Weight.IsNull() && !addr.Modules.LoadBalancer.Weight.IsUnknown() {
				lb.SetWeight(addr.Modules.LoadBalancer.Weight.ValueInt64())
			}
			addrModules := azionapi.NewAddressModulesRequest()
			addrModules.SetLoadBalancer(*lb)
			address.SetModules(*addrModules)
		}
		addresses = append(addresses, *address)
	}

	attrs := azionapi.NewConnectorHTTPAttributesRequest(addresses)

	// Build connection options if provided
	if !connector.HTTPAttrs.ConnectionOptions.IsNull() && !connector.HTTPAttrs.ConnectionOptions.IsUnknown() {
		var connOptsModel HTTPConnectionOptionsModel
		diags := connector.HTTPAttrs.ConnectionOptions.As(ctx, &connOptsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return azionapi.ConnectorRequest{}, fmt.Errorf("failed to parse connection_options")
		}

		connOpts := azionapi.NewHTTPConnectionOptionsRequest()
		if !connOptsModel.DNSResolution.IsNull() && !connOptsModel.DNSResolution.IsUnknown() {
			connOpts.SetDnsResolution(connOptsModel.DNSResolution.ValueString())
		}
		if !connOptsModel.FollowingRedirect.IsNull() && !connOptsModel.FollowingRedirect.IsUnknown() {
			connOpts.SetFollowingRedirect(connOptsModel.FollowingRedirect.ValueBool())
		}
		if !connOptsModel.Host.IsNull() && !connOptsModel.Host.IsUnknown() {
			connOpts.SetHost(connOptsModel.Host.ValueString())
		}
		if !connOptsModel.HTTPVersionPolicy.IsNull() && !connOptsModel.HTTPVersionPolicy.IsUnknown() {
			connOpts.SetHttpVersionPolicy(connOptsModel.HTTPVersionPolicy.ValueString())
		}
		if !connOptsModel.PathPrefix.IsNull() && !connOptsModel.PathPrefix.IsUnknown() {
			connOpts.SetPathPrefix(connOptsModel.PathPrefix.ValueString())
		}
		if !connOptsModel.RealIPHeader.IsNull() && !connOptsModel.RealIPHeader.IsUnknown() {
			connOpts.SetRealIpHeader(connOptsModel.RealIPHeader.ValueString())
		}
		if !connOptsModel.RealPortHeader.IsNull() && !connOptsModel.RealPortHeader.IsUnknown() {
			connOpts.SetRealPortHeader(connOptsModel.RealPortHeader.ValueString())
		}
		if !connOptsModel.TransportPolicy.IsNull() && !connOptsModel.TransportPolicy.IsUnknown() {
			connOpts.SetTransportPolicy(connOptsModel.TransportPolicy.ValueString())
		}
		attrs.SetConnectionOptions(*connOpts)
	}

	// Build modules if provided
	if !connector.HTTPAttrs.Modules.IsNull() && !connector.HTTPAttrs.Modules.IsUnknown() {
		var modulesModel HTTPModulesModel
		diags := connector.HTTPAttrs.Modules.As(ctx, &modulesModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return azionapi.ConnectorRequest{}, fmt.Errorf("failed to parse modules")
		}

		modules := azionapi.NewHTTPModulesRequest()

		if !modulesModel.LoadBalancer.IsNull() && !modulesModel.LoadBalancer.IsUnknown() {
			var lbModel LoadBalancerModuleModel
			if diag := modulesModel.LoadBalancer.As(ctx, &lbModel, basetypes.ObjectAsOptions{}); diag.HasError() {
				return azionapi.ConnectorRequest{}, fmt.Errorf("failed to parse load_balancer module")
			}
			lb := azionapi.NewLoadBalancerModuleRequest()
			if !lbModel.Enabled.IsNull() && !lbModel.Enabled.IsUnknown() {
				lb.SetEnabled(lbModel.Enabled.ValueBool())
			}
			modules.SetLoadBalancer(*lb)
		}

		if !modulesModel.OriginShield.IsNull() && !modulesModel.OriginShield.IsUnknown() {
			var osModel OriginShieldModuleModel
			if diag := modulesModel.OriginShield.As(ctx, &osModel, basetypes.ObjectAsOptions{}); diag.HasError() {
				return azionapi.ConnectorRequest{}, fmt.Errorf("failed to parse origin_shield module")
			}
			os := azionapi.NewOriginShieldModuleRequest()
			if !osModel.Enabled.IsNull() && !osModel.Enabled.IsUnknown() {
				os.SetEnabled(osModel.Enabled.ValueBool())
			}
			modules.SetOriginShield(*os)
		}
		attrs.SetModules(*modules)
	}

	req := azionapi.NewConnectorHTTPRequest(
		connector.Name.ValueString(),
		connector.Type.ValueString(),
		*attrs,
	)

	if !connector.Active.IsNull() && !connector.Active.IsUnknown() {
		req.SetActive(connector.Active.ValueBool())
	}

	return azionapi.ConnectorHTTPRequestAsConnectorRequest(req), nil
}

func buildStoragePatchedConnectorRequest(connector *connectorResourceResults) (azionapi.PatchedConnectorRequest, error) {
	if connector.StorageAttrs == nil {
		return azionapi.PatchedConnectorRequest{}, fmt.Errorf("storage_attributes is required for storage type connectors")
	}

	attrs := azionapi.ConnectorStorageAttributesRequest{
		Bucket: connector.StorageAttrs.Bucket.ValueString(),
	}

	if !connector.StorageAttrs.Prefix.IsNull() && !connector.StorageAttrs.Prefix.IsUnknown() {
		attrs.SetPrefix(connector.StorageAttrs.Prefix.ValueString())
	}

	req := azionapi.NewPatchedConnectorRequestBase(connector.Type.ValueString())
	req.SetName(connector.Name.ValueString())
	req.SetAttributes(attrs)

	if !connector.Active.IsNull() && !connector.Active.IsUnknown() {
		req.SetActive(connector.Active.ValueBool())
	}

	return azionapi.PatchedConnectorRequestBaseAsPatchedConnectorRequest(req), nil
}

func (r *connectorResource) buildHTTPPatchedConnectorRequest(ctx context.Context, connector *connectorResourceResults) (azionapi.PatchedConnectorRequest, error) {
	if connector.HTTPAttrs == nil {
		return azionapi.PatchedConnectorRequest{}, fmt.Errorf("http_attributes is required for http type connectors")
	}

	// Build addresses
	var addresses []azionapi.AddressRequest
	for _, addr := range connector.HTTPAttrs.Addresses {
		address := azionapi.NewAddressRequest(addr.Address.ValueString())
		if !addr.Active.IsNull() && !addr.Active.IsUnknown() {
			address.SetActive(addr.Active.ValueBool())
		}
		if !addr.HTTPPort.IsNull() && !addr.HTTPPort.IsUnknown() {
			address.SetHttpPort(addr.HTTPPort.ValueInt64())
		}
		if !addr.HTTPSPort.IsNull() && !addr.HTTPSPort.IsUnknown() {
			address.SetHttpsPort(addr.HTTPSPort.ValueInt64())
		}
		addresses = append(addresses, *address)
	}

	attrs := azionapi.ConnectorHTTPAttributesRequest{}
	attrs.SetAddresses(addresses)

	// Build connection options if provided
	if !connector.HTTPAttrs.ConnectionOptions.IsNull() && !connector.HTTPAttrs.ConnectionOptions.IsUnknown() {
		var connOptsModel HTTPConnectionOptionsModel
		diags := connector.HTTPAttrs.ConnectionOptions.As(ctx, &connOptsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return azionapi.PatchedConnectorRequest{}, fmt.Errorf("failed to parse connection_options")
		}

		connOpts := azionapi.NewHTTPConnectionOptionsRequest()
		if !connOptsModel.DNSResolution.IsNull() && !connOptsModel.DNSResolution.IsUnknown() {
			connOpts.SetDnsResolution(connOptsModel.DNSResolution.ValueString())
		}
		if !connOptsModel.FollowingRedirect.IsNull() && !connOptsModel.FollowingRedirect.IsUnknown() {
			connOpts.SetFollowingRedirect(connOptsModel.FollowingRedirect.ValueBool())
		}
		if !connOptsModel.Host.IsNull() && !connOptsModel.Host.IsUnknown() {
			connOpts.SetHost(connOptsModel.Host.ValueString())
		}
		if !connOptsModel.HTTPVersionPolicy.IsNull() && !connOptsModel.HTTPVersionPolicy.IsUnknown() {
			connOpts.SetHttpVersionPolicy(connOptsModel.HTTPVersionPolicy.ValueString())
		}
		if !connOptsModel.PathPrefix.IsNull() && !connOptsModel.PathPrefix.IsUnknown() {
			connOpts.SetPathPrefix(connOptsModel.PathPrefix.ValueString())
		}
		if !connOptsModel.RealIPHeader.IsNull() && !connOptsModel.RealIPHeader.IsUnknown() {
			connOpts.SetRealIpHeader(connOptsModel.RealIPHeader.ValueString())
		}
		if !connOptsModel.RealPortHeader.IsNull() && !connOptsModel.RealPortHeader.IsUnknown() {
			connOpts.SetRealPortHeader(connOptsModel.RealPortHeader.ValueString())
		}
		if !connOptsModel.TransportPolicy.IsNull() && !connOptsModel.TransportPolicy.IsUnknown() {
			connOpts.SetTransportPolicy(connOptsModel.TransportPolicy.ValueString())
		}
		attrs.SetConnectionOptions(*connOpts)
	}

	// Build modules if provided
	if !connector.HTTPAttrs.Modules.IsNull() && !connector.HTTPAttrs.Modules.IsUnknown() {
		var modulesModel HTTPModulesModel
		diags := connector.HTTPAttrs.Modules.As(ctx, &modulesModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return azionapi.PatchedConnectorRequest{}, fmt.Errorf("failed to parse modules")
		}

		modules := azionapi.NewHTTPModulesRequest()

		if !modulesModel.LoadBalancer.IsNull() && !modulesModel.LoadBalancer.IsUnknown() {
			var lbModel LoadBalancerModuleModel
			if diag := modulesModel.LoadBalancer.As(ctx, &lbModel, basetypes.ObjectAsOptions{}); diag.HasError() {
				return azionapi.PatchedConnectorRequest{}, fmt.Errorf("failed to parse load_balancer module")
			}
			lb := azionapi.NewLoadBalancerModuleRequest()
			if !lbModel.Enabled.IsNull() && !lbModel.Enabled.IsUnknown() {
				lb.SetEnabled(lbModel.Enabled.ValueBool())
			}
			modules.SetLoadBalancer(*lb)
		}

		if !modulesModel.OriginShield.IsNull() && !modulesModel.OriginShield.IsUnknown() {
			var osModel OriginShieldModuleModel
			if diag := modulesModel.OriginShield.As(ctx, &osModel, basetypes.ObjectAsOptions{}); diag.HasError() {
				return azionapi.PatchedConnectorRequest{}, fmt.Errorf("failed to parse origin_shield module")
			}
			os := azionapi.NewOriginShieldModuleRequest()
			if !osModel.Enabled.IsNull() && !osModel.Enabled.IsUnknown() {
				os.SetEnabled(osModel.Enabled.ValueBool())
			}
			modules.SetOriginShield(*os)
		}
		attrs.SetModules(*modules)
	}

	req := azionapi.NewPatchedConnectorHTTPRequest(connector.Type.ValueString())
	req.SetName(connector.Name.ValueString())
	req.SetAttributes(attrs)

	if !connector.Active.IsNull() && !connector.Active.IsUnknown() {
		req.SetActive(connector.Active.ValueBool())
	}

	return azionapi.PatchedConnectorHTTPRequestAsPatchedConnectorRequest(req), nil
}

func (r *connectorResource) populateConnectorFromResponse(ctx context.Context, model *connectorResourceResults, connector azionapi.Connector) {
	actualConnector := connector.GetActualInstance()
	if actualConnector == nil {
		return
	}

	switch c := actualConnector.(type) {
	case *azionapi.ConnectorBase:
		// Storage connector.
		model.ID = types.Int64Value(c.Id)
		model.Name = types.StringValue(c.Name)
		model.LastEditor = types.StringValue(c.LastEditor)
		model.LastModified = types.StringValue(c.LastModified.Format(time.RFC850))
		model.ProductVersion = types.StringValue(c.ProductVersion)
		model.Type = types.StringValue(c.Type)
		model.Active = types.BoolPointerValue(c.Active)

		// Populate storage attributes
		model.StorageAttrs = &StorageAttributesModel{
			Bucket: types.StringValue(c.Attributes.Bucket),
		}
		if c.Attributes.Prefix != nil {
			model.StorageAttrs.Prefix = types.StringValue(*c.Attributes.Prefix)
		}
		// Clear other attribute types
		model.HTTPAttrs = nil

	case *azionapi.ConnectorHTTP:
		// HTTP connector.
		model.ID = types.Int64Value(c.Id)
		model.Name = types.StringValue(c.Name)
		model.LastEditor = types.StringValue(c.LastEditor)
		model.LastModified = types.StringValue(c.LastModified.Format(time.RFC850))
		model.ProductVersion = types.StringValue(c.ProductVersion)
		model.Type = types.StringValue(c.Type)
		model.Active = types.BoolPointerValue(c.Active)

		// Populate HTTP attributes
		httpAttrs := &HTTPAttributesModel{}

		// Addresses
		for _, addr := range c.Attributes.Addresses {
			addrModel := AddressModel{
				Address: types.StringValue(addr.Address),
			}
			if addr.Active != nil {
				addrModel.Active = types.BoolValue(*addr.Active)
			}
			if addr.HttpPort != nil {
				addrModel.HTTPPort = types.Int64Value(*addr.HttpPort)
			}
			if addr.HttpsPort != nil {
				addrModel.HTTPSPort = types.Int64Value(*addr.HttpsPort)
			}
			if addr.Modules.IsSet() {
				modules := addr.Modules.Get()
				if modules != nil && modules.LoadBalancer != nil {
					addrModel.Modules = &AddressModulesModel{
						LoadBalancer: &AddressLoadBalancerModel{},
					}
					if modules.LoadBalancer.ServerRole != nil {
						addrModel.Modules.LoadBalancer.ServerRole = types.StringValue(*modules.LoadBalancer.ServerRole)
					}
					if modules.LoadBalancer.Weight != nil {
						addrModel.Modules.LoadBalancer.Weight = types.Int64Value(*modules.LoadBalancer.Weight)
					}
				}
			}
			httpAttrs.Addresses = append(httpAttrs.Addresses, addrModel)
		}

		// Connection options - always populate from API response
		if c.Attributes.ConnectionOptions != nil {
			co := c.Attributes.ConnectionOptions
			connOptsModel := HTTPConnectionOptionsModel{}
			if co.DnsResolution != nil {
				connOptsModel.DNSResolution = types.StringValue(*co.DnsResolution)
			}
			if co.FollowingRedirect != nil {
				connOptsModel.FollowingRedirect = types.BoolValue(*co.FollowingRedirect)
			}
			if co.Host != nil {
				connOptsModel.Host = types.StringValue(*co.Host)
			}
			if co.HttpVersionPolicy != nil {
				connOptsModel.HTTPVersionPolicy = types.StringValue(*co.HttpVersionPolicy)
			}
			if co.PathPrefix != nil {
				connOptsModel.PathPrefix = types.StringValue(*co.PathPrefix)
			}
			if co.RealIpHeader != nil {
				connOptsModel.RealIPHeader = types.StringValue(*co.RealIpHeader)
			}
			if co.RealPortHeader != nil {
				connOptsModel.RealPortHeader = types.StringValue(*co.RealPortHeader)
			}
			if co.TransportPolicy != nil {
				connOptsModel.TransportPolicy = types.StringValue(*co.TransportPolicy)
			}
			connOptsValue, diags := types.ObjectValueFrom(ctx, HTTPConnectionOptionsModel{}.attrTypes(), connOptsModel)
			if !diags.HasError() {
				httpAttrs.ConnectionOptions = connOptsValue
			}
		}

		// Modules - always populate from API response
		if c.Attributes.Modules != nil {
			modulesModel := HTTPModulesModel{}

			if c.Attributes.Modules.LoadBalancer != nil {
				lbModel := LoadBalancerModuleModel{}
				if c.Attributes.Modules.LoadBalancer.Enabled != nil {
					lbModel.Enabled = types.BoolValue(*c.Attributes.Modules.LoadBalancer.Enabled)
				}
				lbValue, diags := types.ObjectValueFrom(ctx, LoadBalancerModuleModel{}.attrTypes(), lbModel)
				if !diags.HasError() {
					modulesModel.LoadBalancer = lbValue
				}
			}

			if c.Attributes.Modules.OriginShield != nil {
				osModel := OriginShieldModuleModel{}
				if c.Attributes.Modules.OriginShield.Enabled != nil {
					osModel.Enabled = types.BoolValue(*c.Attributes.Modules.OriginShield.Enabled)
				}
				osValue, diags := types.ObjectValueFrom(ctx, OriginShieldModuleModel{}.attrTypes(), osModel)
				if !diags.HasError() {
					modulesModel.OriginShield = osValue
				}
			}

			modulesValue, diags := types.ObjectValueFrom(ctx, HTTPModulesModel{}.attrTypes(), modulesModel)
			if !diags.HasError() {
				httpAttrs.Modules = modulesValue
			}
		}

		model.HTTPAttrs = httpAttrs
		// Clear storage attributes
		model.StorageAttrs = nil
	}
}

// attrTypes returns the attribute types for HTTPConnectionOptionsModel
func (m HTTPConnectionOptionsModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"dns_resolution":      types.StringType,
		"following_redirect":  types.BoolType,
		"host":                types.StringType,
		"http_version_policy": types.StringType,
		"path_prefix":         types.StringType,
		"real_ip_header":      types.StringType,
		"real_port_header":    types.StringType,
		"transport_policy":    types.StringType,
	}
}

// attrTypes returns the attribute types for HTTPModulesModel
func (m HTTPModulesModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"load_balancer": types.ObjectType{AttrTypes: LoadBalancerModuleModel{}.attrTypes()},
		"origin_shield": types.ObjectType{AttrTypes: OriginShieldModuleModel{}.attrTypes()},
	}
}

// attrTypes returns the attribute types for LoadBalancerModuleModel
func (m LoadBalancerModuleModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
	}
}

// attrTypes returns the attribute types for OriginShieldModuleModel
func (m OriginShieldModuleModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
	}
}

// addConnectorAPIError adds an appropriate error to diagnostics based on the API response.
func addConnectorAPIError(diagnostics *diag.Diagnostics, err error, response *http.Response, operation string) {
	if response == nil {
		diagnostics.AddError(err.Error(), "No response received")
		return
	}

	if response.StatusCode == 429 {
		// Rate limiting.
		diagnostics.AddError(
			err.Error(),
			"API request rate limited",
		)
		return
	}

	bodyBytes, errReadAll := io.ReadAll(response.Body)
	if errReadAll != nil {
		diagnostics.AddError(errReadAll.Error(), "err")
		return
	}
	bodyString := string(bodyBytes)
	diagnostics.AddError(err.Error(), bodyString)
}
