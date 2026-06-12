package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// Main resource model.
type connectorResourceModel struct {
	Connector     *connectorResourceResults `tfsdk:"connector"`
	ID            types.String              `tfsdk:"id"`
	LastUpdated   types.String              `tfsdk:"last_updated"`
	SchemaVersion types.Int64               `tfsdk:"schema_version"`
}

// Connector results - all fields including type-specific attributes.
type connectorResourceResults struct {
	ID             types.Int64             `tfsdk:"id"`
	Name           types.String            `tfsdk:"name"`
	LastEditor     types.String            `tfsdk:"last_editor"`
	LastModified   types.String            `tfsdk:"last_modified"`
	CreatedAt      types.String            `tfsdk:"created_at"`
	ProductVersion types.String            `tfsdk:"product_version"`
	Active         types.Bool              `tfsdk:"active"`
	Type           types.String            `tfsdk:"type"`
	IsVersioned    types.Bool              `tfsdk:"is_versioned"`
	Version        types.Int64             `tfsdk:"version"`
	VersionState   types.String            `tfsdk:"version_state"`
	VersionID      types.String            `tfsdk:"version_id"`
	StorageAttrs   *StorageAttributesModel `tfsdk:"storage_attributes"`
	HTTPAttrs      *HTTPAttributesModel    `tfsdk:"http_attributes"`
}

// Storage connector attributes.
type StorageAttributesModel struct {
	Bucket types.String `tfsdk:"bucket"`
	Prefix types.String `tfsdk:"prefix"`
}

// HTTP connector attributes.
type HTTPAttributesModel struct {
	Addresses         []AddressWrapperModel       `tfsdk:"addresses"`
	ConnectionOptions *HTTPConnectionOptionsModel `tfsdk:"connection_options"`
	Modules           *HTTPModulesModel           `tfsdk:"modules"`
}

// AddressWrapperModel wraps a single endpoint under an `endpoint` label.
type AddressWrapperModel struct {
	Endpoint *AddressModel `tfsdk:"endpoint"`
}

// Address model for HTTP connectors.
type AddressModel struct {
	Address   types.String         `tfsdk:"address"`
	Active    types.Bool           `tfsdk:"active"`
	HTTPPort  types.Int64          `tfsdk:"http_port"`
	HTTPSPort types.Int64          `tfsdk:"https_port"`
	Modules   *AddressModulesModel `tfsdk:"modules"`
}

// Address modules.
type AddressModulesModel struct {
	LoadBalancer *AddressLoadBalancerModel `tfsdk:"load_balancer"`
}

// Address load balancer module - uses server_role and weight.
type AddressLoadBalancerModel struct {
	ServerRole types.String `tfsdk:"server_role"`
	Weight     types.Int64  `tfsdk:"weight"`
}

// HTTP connection options.
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

// HTTP modules.
type HTTPModulesModel struct {
	LoadBalancer *LoadBalancerModuleModel `tfsdk:"load_balancer"`
	OriginShield *OriginShieldModuleModel `tfsdk:"origin_shield"`
}

// Load balancer module.
type LoadBalancerModuleModel struct {
	Enabled types.Bool               `tfsdk:"enabled"`
	Config  *LoadBalancerConfigModel `tfsdk:"config"`
}

// Load balancer config.
type LoadBalancerConfigModel struct {
	Method            types.String `tfsdk:"method"`
	MaxRetries        types.Int64  `tfsdk:"max_retries"`
	ConnectionTimeout types.Int64  `tfsdk:"connection_timeout"`
	ReadWriteTimeout  types.Int64  `tfsdk:"read_write_timeout"`
}

// Origin shield module.
type OriginShieldModuleModel struct {
	Enabled types.Bool               `tfsdk:"enabled"`
	Config  *OriginShieldConfigModel `tfsdk:"config"`
}

// Origin shield config.
type OriginShieldConfigModel struct {
	OriginIPAcl *OriginIPAclModel `tfsdk:"origin_ip_acl"`
	Hmac        *HMACConfigModel  `tfsdk:"hmac"`
}

// Origin IP ACL configuration for origin shield.
type OriginIPAclModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

// HMAC configuration.
type HMACConfigModel struct {
	Enabled types.Bool           `tfsdk:"enabled"`
	Config  *AWS4HMACConfigModel `tfsdk:"config"`
}

// AWS4 HMAC configuration.
type AWS4HMACConfigModel struct {
	Type       types.String             `tfsdk:"type"`
	Attributes *AWS4HMACAttributesModel `tfsdk:"attributes"`
}

// AWS4 HMAC attributes.
type AWS4HMACAttributesModel struct {
	Region    types.String `tfsdk:"region"`
	Service   types.String `tfsdk:"service"`
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
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
						Optional:    true,
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the connector (http or storage).",
						Required:    true,
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
								Description: "List of origin endpoints.",
								Required:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"endpoint": schema.SingleNestedAttribute{
											Description: "A single origin endpoint configuration.",
											Required:    true,
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
								},
							},
							"connection_options": schema.SingleNestedAttribute{
								Description: "HTTP connection options.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"dns_resolution": schema.StringAttribute{
										Description: "DNS resolution strategy.",
										Optional:    true,
									},
									"following_redirect": schema.BoolAttribute{
										Description: "Whether to follow redirects.",
										Optional:    true,
									},
									"host": schema.StringAttribute{
										Description: "Host header value. Use ${host} to pass through the original host.",
										Optional:    true,
									},
									"http_version_policy": schema.StringAttribute{
										Description: "HTTP version policy.",
										Optional:    true,
									},
									"path_prefix": schema.StringAttribute{
										Description: "Path prefix for requests.",
										Optional:    true,
									},
									"real_ip_header": schema.StringAttribute{
										Description: "Header for real IP.",
										Optional:    true,
									},
									"real_port_header": schema.StringAttribute{
										Description: "Header for real port.",
										Optional:    true,
									},
									"transport_policy": schema.StringAttribute{
										Description: "Transport policy.",
										Optional:    true,
									},
								},
							},
							"modules": schema.SingleNestedAttribute{
								Description: "HTTP modules configuration.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"load_balancer": schema.SingleNestedAttribute{
										Description: "Load balancer module.",
										Optional:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Description: "Whether load balancer is enabled.",
												Optional:    true,
											},
											"config": schema.SingleNestedAttribute{
												Description: "Load balancer configuration.",
												Optional:    true,
												Attributes: map[string]schema.Attribute{
													"method": schema.StringAttribute{
														Description: "Load balancing method (round_robin, least_conn, ip_hash).",
														Optional:    true,
													},
													"max_retries": schema.Int64Attribute{
														Description: "Maximum number of retry attempts on connection failure.",
														Optional:    true,
													},
													"connection_timeout": schema.Int64Attribute{
														Description: "Maximum time (in seconds) to wait for a connection to be established.",
														Optional:    true,
													},
													"read_write_timeout": schema.Int64Attribute{
														Description: "Maximum time (in seconds) to wait for data read/write after connection.",
														Optional:    true,
													},
												},
											},
										},
									},
									"origin_shield": schema.SingleNestedAttribute{
										Description: "Origin shield module.",
										Optional:    true,
										Attributes: map[string]schema.Attribute{
											"enabled": schema.BoolAttribute{
												Description: "Whether origin shield is enabled.",
												Optional:    true,
											},
											"config": schema.SingleNestedAttribute{
												Description: "Origin shield configuration.",
												Optional:    true,
												Attributes: map[string]schema.Attribute{
													"origin_ip_acl": schema.SingleNestedAttribute{
														Description: "Origin IP ACL configuration.",
														Optional:    true,
														Attributes: map[string]schema.Attribute{
															"enabled": schema.BoolAttribute{
																Description: "Whether the origin IP ACL is enabled.",
																Optional:    true,
															},
														},
													},
													"hmac": schema.SingleNestedAttribute{
														Description: "HMAC configuration for origin shield.",
														Optional:    true,
														Attributes: map[string]schema.Attribute{
															"enabled": schema.BoolAttribute{
																Description: "Whether HMAC is enabled.",
																Optional:    true,
															},
															"config": schema.SingleNestedAttribute{
																Description: "AWS4 HMAC configuration.",
																Optional:    true,
																Attributes: map[string]schema.Attribute{
																	"type": schema.StringAttribute{
																		Description: "HMAC type (e.g., aws4_hmac_sha256).",
																		Optional:    true,
																	},
																	"attributes": schema.SingleNestedAttribute{
																		Description: "AWS4 HMAC attributes.",
																		Optional:    true,
																		Attributes: map[string]schema.Attribute{
																			"region": schema.StringAttribute{
																				Description: "AWS region.",
																				Optional:    true,
																			},
																			"service": schema.StringAttribute{
																				Description: "AWS service name.",
																				Optional:    true,
																			},
																			"access_key": schema.StringAttribute{
																				Description: "AWS access key.",
																				Optional:    true,
																				Sensitive:   true,
																			},
																			"secret_key": schema.StringAttribute{
																				Description: "AWS secret key.",
																				Optional:    true,
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
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response != nil && response.StatusCode == http.StatusTooManyRequests {
				createConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
					return r.client.api.ConnectorsAPI.CreateConnector(ctx).ConnectorRequest(connectorReq).Execute()
				}, 5)
				if response != nil {
					defer response.Body.Close()
				}
				if err != nil {
					resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
					return
				}
			} else {
				addConnectorAPIError(&resp.Diagnostics, err, response, "create")
				return
			}
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
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response != nil && response.StatusCode == http.StatusTooManyRequests {
				createConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
					return r.client.api.ConnectorsAPI.CreateConnector(ctx).ConnectorRequest(connectorReq).Execute()
				}, 5)
				if response != nil {
					defer response.Body.Close()
				}
				if err != nil {
					resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
					return
				}
			} else {
				addConnectorAPIError(&resp.Diagnostics, err, response, "create")
				return
			}
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
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
				return r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addConnectorAPIError(&resp.Diagnostics, err, response, "read after create")
			return
		}
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
	case *azionapi.ConnectorStorage:
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
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
				return r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addConnectorAPIError(&resp.Diagnostics, err, response, "read")
			return
		}
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
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response != nil && response.StatusCode == http.StatusTooManyRequests {
				updateConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
					return r.client.api.ConnectorsAPI.PartialUpdateConnector(ctx, connectorId).PatchedConnectorRequest(connectorReq).Execute()
				}, 5)
				if response != nil {
					defer response.Body.Close()
				}
				if err != nil {
					resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
					return
				}
			} else {
				addConnectorAPIError(&resp.Diagnostics, err, response, "update")
				return
			}
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
		if response != nil {
			defer response.Body.Close()
		}
		if err != nil {
			if response != nil && response.StatusCode == http.StatusTooManyRequests {
				updateConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
					return r.client.api.ConnectorsAPI.PartialUpdateConnector(ctx, connectorId).PatchedConnectorRequest(connectorReq).Execute()
				}, 5)
				if response != nil {
					defer response.Body.Close()
				}
				if err != nil {
					resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
					return
				}
			} else {
				addConnectorAPIError(&resp.Diagnostics, err, response, "update")
				return
			}
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

	_, response, err := utils.RetryOn429Delete(func() (*azionapi.DeleteResponse, *http.Response, error) {
		return r.client.api.ConnectorsAPI.DeleteConnector(ctx, connectorId).Execute()
	}, 5)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return
		}
		addConnectorAPIError(&resp.Diagnostics, err, response, "delete")
		return
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
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		if response != nil && response.StatusCode == http.StatusTooManyRequests {
			getConnector, response, err = utils.RetryOn429(func() (*azionapi.ConnectorResponse, *http.Response, error) {
				return r.client.api.ConnectorsAPI.RetrieveConnector(ctx, connectorId).Execute()
			}, 5)
			if response != nil {
				defer response.Body.Close()
			}
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "API request failed after too many retries")
				return
			}
		} else {
			addConnectorAPIError(&resp.Diagnostics, err, response, "import")
			return
		}
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

	req := azionapi.NewConnectorStorageRequest(
		connector.Name.ValueString(),
		connector.Type.ValueString(),
		attrs,
	)

	if !connector.Active.IsNull() && !connector.Active.IsUnknown() {
		req.SetActive(connector.Active.ValueBool())
	}

	return azionapi.ConnectorStorageRequestAsConnectorRequest(req), nil
}

func (r *connectorResource) buildHTTPConnectorRequest(_ context.Context, connector *connectorResourceResults) (azionapi.ConnectorRequest, error) {
	if connector.HTTPAttrs == nil {
		return azionapi.ConnectorRequest{}, fmt.Errorf("http_attributes is required for http type connectors")
	}

	addresses := buildAddressRequests(connector.HTTPAttrs.Addresses)
	attrs := azionapi.NewConnectorHTTPAttributesRequest(addresses)

	if connector.HTTPAttrs.ConnectionOptions != nil {
		attrs.SetConnectionOptions(*buildConnectionOptionsRequest(connector.HTTPAttrs.ConnectionOptions))
	}

	if connector.HTTPAttrs.Modules != nil {
		attrs.SetModules(*buildHTTPModulesRequest(connector.HTTPAttrs.Modules))
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

func buildAddressRequests(addrs []AddressWrapperModel) []azionapi.AddressRequest {
	var addresses []azionapi.AddressRequest
	for _, wrapper := range addrs {
		if wrapper.Endpoint == nil {
			continue
		}
		addr := wrapper.Endpoint
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
	return addresses
}

func buildConnectionOptionsRequest(co *HTTPConnectionOptionsModel) *azionapi.HTTPConnectionOptionsRequest {
	connOpts := azionapi.NewHTTPConnectionOptionsRequest()
	if !co.DNSResolution.IsNull() && !co.DNSResolution.IsUnknown() {
		connOpts.SetDnsResolution(co.DNSResolution.ValueString())
	}
	if !co.FollowingRedirect.IsNull() && !co.FollowingRedirect.IsUnknown() {
		connOpts.SetFollowingRedirect(co.FollowingRedirect.ValueBool())
	}
	if !co.Host.IsNull() && !co.Host.IsUnknown() {
		connOpts.SetHost(co.Host.ValueString())
	}
	if !co.HTTPVersionPolicy.IsNull() && !co.HTTPVersionPolicy.IsUnknown() {
		connOpts.SetHttpVersionPolicy(co.HTTPVersionPolicy.ValueString())
	}
	if !co.PathPrefix.IsNull() && !co.PathPrefix.IsUnknown() {
		connOpts.SetPathPrefix(co.PathPrefix.ValueString())
	}
	if !co.RealIPHeader.IsNull() && !co.RealIPHeader.IsUnknown() {
		connOpts.SetRealIpHeader(co.RealIPHeader.ValueString())
	}
	if !co.RealPortHeader.IsNull() && !co.RealPortHeader.IsUnknown() {
		connOpts.SetRealPortHeader(co.RealPortHeader.ValueString())
	}
	if !co.TransportPolicy.IsNull() && !co.TransportPolicy.IsUnknown() {
		connOpts.SetTransportPolicy(co.TransportPolicy.ValueString())
	}
	return connOpts
}

func buildHTTPModulesRequest(m *HTTPModulesModel) *azionapi.HTTPModulesRequest {
	modules := azionapi.NewHTTPModulesRequest()

	if m.LoadBalancer != nil {
		lb := azionapi.NewLoadBalancerModuleRequest()
		if !m.LoadBalancer.Enabled.IsNull() && !m.LoadBalancer.Enabled.IsUnknown() {
			lb.SetEnabled(m.LoadBalancer.Enabled.ValueBool())
		}
		if m.LoadBalancer.Config != nil {
			lbConfig := azionapi.NewLoadBalancerModuleConfigRequest()
			if !m.LoadBalancer.Config.Method.IsNull() && !m.LoadBalancer.Config.Method.IsUnknown() {
				lbConfig.SetMethod(m.LoadBalancer.Config.Method.ValueString())
			}
			if !m.LoadBalancer.Config.MaxRetries.IsNull() && !m.LoadBalancer.Config.MaxRetries.IsUnknown() {
				lbConfig.SetMaxRetries(m.LoadBalancer.Config.MaxRetries.ValueInt64())
			}
			if !m.LoadBalancer.Config.ConnectionTimeout.IsNull() && !m.LoadBalancer.Config.ConnectionTimeout.IsUnknown() {
				lbConfig.SetConnectionTimeout(m.LoadBalancer.Config.ConnectionTimeout.ValueInt64())
			}
			if !m.LoadBalancer.Config.ReadWriteTimeout.IsNull() && !m.LoadBalancer.Config.ReadWriteTimeout.IsUnknown() {
				lbConfig.SetReadWriteTimeout(m.LoadBalancer.Config.ReadWriteTimeout.ValueInt64())
			}
			lb.SetConfig(*lbConfig)
		}
		modules.SetLoadBalancer(*lb)
	}

	if m.OriginShield != nil {
		os := azionapi.NewOriginShieldModuleRequest()
		if !m.OriginShield.Enabled.IsNull() && !m.OriginShield.Enabled.IsUnknown() {
			os.SetEnabled(m.OriginShield.Enabled.ValueBool())
		}
		if m.OriginShield.Config != nil {
			osConfig := azionapi.NewOriginShieldConfigRequest()
			if m.OriginShield.Config.OriginIPAcl != nil {
				ipAcl := azionapi.NewOriginIPACLRequest()
				if !m.OriginShield.Config.OriginIPAcl.Enabled.IsNull() && !m.OriginShield.Config.OriginIPAcl.Enabled.IsUnknown() {
					ipAcl.SetEnabled(m.OriginShield.Config.OriginIPAcl.Enabled.ValueBool())
				}
				osConfig.SetOriginIpAcl(*ipAcl)
			}
			if m.OriginShield.Config.Hmac != nil {
				hmacEnabled := false
				if !m.OriginShield.Config.Hmac.Enabled.IsNull() && !m.OriginShield.Config.Hmac.Enabled.IsUnknown() {
					hmacEnabled = m.OriginShield.Config.Hmac.Enabled.ValueBool()
				}
				hmacReq := azionapi.NewHMACRequest(hmacEnabled)
				if m.OriginShield.Config.Hmac.Config != nil && m.OriginShield.Config.Hmac.Config.Attributes != nil {
					region := ""
					if !m.OriginShield.Config.Hmac.Config.Attributes.Region.IsNull() && !m.OriginShield.Config.Hmac.Config.Attributes.Region.IsUnknown() {
						region = m.OriginShield.Config.Hmac.Config.Attributes.Region.ValueString()
					}
					accessKey := ""
					if !m.OriginShield.Config.Hmac.Config.Attributes.AccessKey.IsNull() && !m.OriginShield.Config.Hmac.Config.Attributes.AccessKey.IsUnknown() {
						accessKey = m.OriginShield.Config.Hmac.Config.Attributes.AccessKey.ValueString()
					}
					secretKey := ""
					if !m.OriginShield.Config.Hmac.Config.Attributes.SecretKey.IsNull() && !m.OriginShield.Config.Hmac.Config.Attributes.SecretKey.IsUnknown() {
						secretKey = m.OriginShield.Config.Hmac.Config.Attributes.SecretKey.ValueString()
					}
					hmacAttrs := azionapi.NewAWS4HMACAttributesRequest(region, accessKey, secretKey)
					if !m.OriginShield.Config.Hmac.Config.Attributes.Service.IsNull() && !m.OriginShield.Config.Hmac.Config.Attributes.Service.IsUnknown() {
						hmacAttrs.SetService(m.OriginShield.Config.Hmac.Config.Attributes.Service.ValueString())
					}
					aws4Hmac := azionapi.NewAWS4HMACRequest(*hmacAttrs)
					if !m.OriginShield.Config.Hmac.Config.Type.IsNull() && !m.OriginShield.Config.Hmac.Config.Type.IsUnknown() {
						aws4Hmac.SetType(m.OriginShield.Config.Hmac.Config.Type.ValueString())
					}
					hmacReq.SetConfig(*aws4Hmac)
				}
				osConfig.SetHmac(*hmacReq)
			}
			os.SetConfig(*osConfig)
		}
		modules.SetOriginShield(*os)
	}

	return modules
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

	req := azionapi.NewPatchedConnectorStorageRequest(connector.Type.ValueString())
	req.SetName(connector.Name.ValueString())
	req.SetAttributes(attrs)

	if !connector.Active.IsNull() && !connector.Active.IsUnknown() {
		req.SetActive(connector.Active.ValueBool())
	}

	return azionapi.PatchedConnectorStorageRequestAsPatchedConnectorRequest(req), nil
}

func (r *connectorResource) buildHTTPPatchedConnectorRequest(_ context.Context, connector *connectorResourceResults) (azionapi.PatchedConnectorRequest, error) {
	if connector.HTTPAttrs == nil {
		return azionapi.PatchedConnectorRequest{}, fmt.Errorf("http_attributes is required for http type connectors")
	}

	attrs := azionapi.ConnectorHTTPAttributesRequest{}
	attrs.SetAddresses(buildAddressRequests(connector.HTTPAttrs.Addresses))

	if connector.HTTPAttrs.ConnectionOptions != nil {
		attrs.SetConnectionOptions(*buildConnectionOptionsRequest(connector.HTTPAttrs.ConnectionOptions))
	}

	if connector.HTTPAttrs.Modules != nil {
		attrs.SetModules(*buildHTTPModulesRequest(connector.HTTPAttrs.Modules))
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
	case *azionapi.ConnectorStorage:
		// Storage connector.
		model.ID = types.Int64Value(c.Id)
		model.Name = types.StringValue(c.Name)
		model.LastEditor = types.StringValue(c.LastEditor)
		model.LastModified = types.StringValue(c.LastModified.Format(time.RFC850))
		model.CreatedAt = types.StringValue(c.CreatedAt.Format(time.RFC850))
		model.ProductVersion = types.StringValue(c.ProductVersion)
		model.Type = types.StringValue(c.Type)
		model.Active = types.BoolPointerValue(c.Active)
		model.IsVersioned = types.BoolValue(c.IsVersioned)
		model.Version = types.Int64PointerValue(c.Version.Get())
		model.VersionState = types.StringPointerValue(c.VersionState.Get())
		model.VersionID = types.StringPointerValue(c.VersionId.Get())

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
		// Snapshot the prior http_attributes shape so unconfigured nested blocks
		// aren't introduced into state from API echoes, which would cause perpetual
		// drift on subsequent plans.
		priorHTTPAttrs := model.HTTPAttrs

		model.ID = types.Int64Value(c.Id)
		model.Name = types.StringValue(c.Name)
		model.LastEditor = types.StringValue(c.LastEditor)
		model.LastModified = types.StringValue(c.LastModified.Format(time.RFC850))
		model.CreatedAt = types.StringValue(c.CreatedAt.Format(time.RFC850))
		model.ProductVersion = types.StringValue(c.ProductVersion)
		model.Type = types.StringValue(c.Type)
		model.Active = types.BoolPointerValue(c.Active)
		model.IsVersioned = types.BoolValue(c.IsVersioned)
		model.Version = types.Int64PointerValue(c.Version.Get())
		model.VersionState = types.StringPointerValue(c.VersionState.Get())
		model.VersionID = types.StringPointerValue(c.VersionId.Get())

		httpAttrs := &HTTPAttributesModel{
			Addresses: populateAddresses(c.Attributes.Addresses),
		}

		if shouldPopulate(priorHTTPAttrs, func(p *HTTPAttributesModel) bool { return p.ConnectionOptions != nil }) && c.Attributes.ConnectionOptions != nil {
			var priorCO *HTTPConnectionOptionsModel
			if priorHTTPAttrs != nil {
				priorCO = priorHTTPAttrs.ConnectionOptions
			}
			httpAttrs.ConnectionOptions = populateConnectionOptions(priorCO, c.Attributes.ConnectionOptions)
		}

		if shouldPopulate(priorHTTPAttrs, func(p *HTTPAttributesModel) bool { return p.Modules != nil }) && c.Attributes.Modules != nil {
			var priorModules *HTTPModulesModel
			if priorHTTPAttrs != nil {
				priorModules = priorHTTPAttrs.Modules
			}
			httpAttrs.Modules = populateHTTPModules(priorModules, c.Attributes.Modules)
		}

		model.HTTPAttrs = httpAttrs
		// Clear storage attributes
		model.StorageAttrs = nil
	}
}

// shouldPopulate returns true when prior is nil (e.g. fresh import) or when
// the predicate over the prior shape returns true. This gates state population
// so that fields the user never configured don't get introduced from API echoes.
func shouldPopulate[T any](prior *T, pred func(*T) bool) bool {
	if prior == nil {
		return true
	}
	return pred(prior)
}

func populateAddresses(in []azionapi.Address) []AddressWrapperModel {
	var out []AddressWrapperModel
	for _, addr := range in {
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
		out = append(out, AddressWrapperModel{
			Endpoint: &addrModel,
		})
	}
	return out
}

func populateConnectionOptions(prior *HTTPConnectionOptionsModel, co *azionapi.HTTPConnectionOptions) *HTTPConnectionOptionsModel {
	out := &HTTPConnectionOptionsModel{}
	// Seed with prior values so unconfigured leaves stay null and don't drift
	// from API echoes (e.g. http_version_policy="http1_1", path_prefix="").
	if prior != nil {
		*out = *prior
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.DNSResolution.IsNull() }) && co.DnsResolution != nil {
		out.DNSResolution = types.StringValue(*co.DnsResolution)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.FollowingRedirect.IsNull() }) && co.FollowingRedirect != nil {
		out.FollowingRedirect = types.BoolValue(*co.FollowingRedirect)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.Host.IsNull() }) && co.Host != nil {
		out.Host = types.StringValue(*co.Host)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.HTTPVersionPolicy.IsNull() }) && co.HttpVersionPolicy != nil {
		out.HTTPVersionPolicy = types.StringValue(*co.HttpVersionPolicy)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.PathPrefix.IsNull() }) && co.PathPrefix != nil {
		out.PathPrefix = types.StringValue(*co.PathPrefix)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.RealIPHeader.IsNull() }) && co.RealIpHeader != nil {
		out.RealIPHeader = types.StringValue(*co.RealIpHeader)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.RealPortHeader.IsNull() }) && co.RealPortHeader != nil {
		out.RealPortHeader = types.StringValue(*co.RealPortHeader)
	}
	if shouldPopulate(prior, func(p *HTTPConnectionOptionsModel) bool { return !p.TransportPolicy.IsNull() }) && co.TransportPolicy != nil {
		out.TransportPolicy = types.StringValue(*co.TransportPolicy)
	}
	return out
}

func populateHTTPModules(prior *HTTPModulesModel, m *azionapi.HTTPModules) *HTTPModulesModel {
	out := &HTTPModulesModel{}

	if shouldPopulate(prior, func(p *HTTPModulesModel) bool { return p.LoadBalancer != nil }) && m.LoadBalancer != nil {
		var priorLB *LoadBalancerModuleModel
		if prior != nil {
			priorLB = prior.LoadBalancer
		}
		lb := &LoadBalancerModuleModel{}
		if priorLB != nil {
			lb.Enabled = priorLB.Enabled
		}
		if shouldPopulate(priorLB, func(p *LoadBalancerModuleModel) bool { return !p.Enabled.IsNull() }) && m.LoadBalancer.Enabled != nil {
			lb.Enabled = types.BoolValue(*m.LoadBalancer.Enabled)
		}
		if shouldPopulate(priorLB, func(p *LoadBalancerModuleModel) bool { return p.Config != nil }) && m.LoadBalancer.Config.IsSet() {
			if lbConfig := m.LoadBalancer.Config.Get(); lbConfig != nil {
				var priorLBConfig *LoadBalancerConfigModel
				if priorLB != nil {
					priorLBConfig = priorLB.Config
				}
				cfg := &LoadBalancerConfigModel{}
				if priorLBConfig != nil {
					*cfg = *priorLBConfig
				}
				if shouldPopulate(priorLBConfig, func(p *LoadBalancerConfigModel) bool { return !p.Method.IsNull() }) && lbConfig.Method != nil {
					cfg.Method = types.StringValue(*lbConfig.Method)
				}
				if shouldPopulate(priorLBConfig, func(p *LoadBalancerConfigModel) bool { return !p.MaxRetries.IsNull() }) && lbConfig.MaxRetries != nil {
					cfg.MaxRetries = types.Int64Value(*lbConfig.MaxRetries)
				}
				if shouldPopulate(priorLBConfig, func(p *LoadBalancerConfigModel) bool { return !p.ConnectionTimeout.IsNull() }) && lbConfig.ConnectionTimeout != nil {
					cfg.ConnectionTimeout = types.Int64Value(*lbConfig.ConnectionTimeout)
				}
				if shouldPopulate(priorLBConfig, func(p *LoadBalancerConfigModel) bool { return !p.ReadWriteTimeout.IsNull() }) && lbConfig.ReadWriteTimeout != nil {
					cfg.ReadWriteTimeout = types.Int64Value(*lbConfig.ReadWriteTimeout)
				}
				lb.Config = cfg
			}
		}
		out.LoadBalancer = lb
	}

	if shouldPopulate(prior, func(p *HTTPModulesModel) bool { return p.OriginShield != nil }) && m.OriginShield != nil {
		var priorOS *OriginShieldModuleModel
		if prior != nil {
			priorOS = prior.OriginShield
		}
		os := &OriginShieldModuleModel{}
		if priorOS != nil {
			os.Enabled = priorOS.Enabled
		}
		if shouldPopulate(priorOS, func(p *OriginShieldModuleModel) bool { return !p.Enabled.IsNull() }) && m.OriginShield.Enabled != nil {
			os.Enabled = types.BoolValue(*m.OriginShield.Enabled)
		}
		if shouldPopulate(priorOS, func(p *OriginShieldModuleModel) bool { return p.Config != nil }) && m.OriginShield.Config.IsSet() {
			if osConfig := m.OriginShield.Config.Get(); osConfig != nil {
				var priorOSConfig *OriginShieldConfigModel
				if priorOS != nil {
					priorOSConfig = priorOS.Config
				}
				os.Config = &OriginShieldConfigModel{}

				if shouldPopulate(priorOSConfig, func(p *OriginShieldConfigModel) bool { return p.OriginIPAcl != nil }) && osConfig.OriginIpAcl != nil {
					var priorIPAcl *OriginIPAclModel
					if priorOSConfig != nil {
						priorIPAcl = priorOSConfig.OriginIPAcl
					}
					ipAcl := &OriginIPAclModel{}
					if priorIPAcl != nil {
						ipAcl.Enabled = priorIPAcl.Enabled
					}
					if shouldPopulate(priorIPAcl, func(p *OriginIPAclModel) bool { return !p.Enabled.IsNull() }) && osConfig.OriginIpAcl.Enabled != nil {
						ipAcl.Enabled = types.BoolValue(*osConfig.OriginIpAcl.Enabled)
					}
					os.Config.OriginIPAcl = ipAcl
				}

				if shouldPopulate(priorOSConfig, func(p *OriginShieldConfigModel) bool { return p.Hmac != nil }) && osConfig.Hmac != nil {
					var priorHmac *HMACConfigModel
					if priorOSConfig != nil {
						priorHmac = priorOSConfig.Hmac
					}
					hmac := &HMACConfigModel{}
					if priorHmac != nil {
						hmac.Enabled = priorHmac.Enabled
					}
					if shouldPopulate(priorHmac, func(p *HMACConfigModel) bool { return !p.Enabled.IsNull() }) {
						hmac.Enabled = types.BoolValue(osConfig.Hmac.Enabled)
					}
					if shouldPopulate(priorHmac, func(p *HMACConfigModel) bool { return p.Config != nil }) && osConfig.Hmac.Config.IsSet() {
						if aws4 := osConfig.Hmac.Config.Get(); aws4 != nil {
							var priorAws4 *AWS4HMACConfigModel
							if priorHmac != nil {
								priorAws4 = priorHmac.Config
							}
							hmac.Config = &AWS4HMACConfigModel{}
							if priorAws4 != nil {
								hmac.Config.Type = priorAws4.Type
							}
							if shouldPopulate(priorAws4, func(p *AWS4HMACConfigModel) bool { return !p.Type.IsNull() }) && aws4.Type != nil {
								hmac.Config.Type = types.StringValue(*aws4.Type)
							}
							if shouldPopulate(priorAws4, func(p *AWS4HMACConfigModel) bool { return p.Attributes != nil }) {
								var priorAttrs *AWS4HMACAttributesModel
								if priorAws4 != nil {
									priorAttrs = priorAws4.Attributes
								}
								attrs := &AWS4HMACAttributesModel{}
								if priorAttrs != nil {
									*attrs = *priorAttrs
								}
								if shouldPopulate(priorAttrs, func(p *AWS4HMACAttributesModel) bool { return !p.Region.IsNull() }) {
									attrs.Region = types.StringValue(aws4.Attributes.Region)
								}
								if shouldPopulate(priorAttrs, func(p *AWS4HMACAttributesModel) bool { return !p.AccessKey.IsNull() }) {
									attrs.AccessKey = types.StringValue(aws4.Attributes.AccessKey)
								}
								if shouldPopulate(priorAttrs, func(p *AWS4HMACAttributesModel) bool { return !p.SecretKey.IsNull() }) {
									attrs.SecretKey = types.StringValue(aws4.Attributes.SecretKey)
								}
								if shouldPopulate(priorAttrs, func(p *AWS4HMACAttributesModel) bool { return !p.Service.IsNull() }) && aws4.Attributes.Service != nil {
									attrs.Service = types.StringValue(*aws4.Attributes.Service)
								}
								hmac.Config.Attributes = attrs
							}
						}
					}
					os.Config.Hmac = hmac
				}
			}
		}
		out.OriginShield = os
	}

	return out
}

// addConnectorAPIError adds an appropriate error to diagnostics based on the API response.
func addConnectorAPIError(diagnostics *diag.Diagnostics, err error, response *http.Response, _ string) {
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
