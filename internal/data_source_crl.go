package provider

import (
	"context"
	"io"
	"net/http"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &CrlDataSource{}
	_ datasource.DataSourceWithConfigure = &CrlDataSource{}
)

func dataSourceAzionCrl() datasource.DataSource {
	return &CrlDataSource{}
}

type CrlDataSource struct {
	client *apiClient
}

type CrlDataSourceModel struct {
	ID            types.String     `tfsdk:"id"`
	SchemaVersion types.Int64      `tfsdk:"schema_version"`
	Results       *CrlResultsModel `tfsdk:"results"`
	CrlID         types.Int64      `tfsdk:"crl_id"`
}

type CrlResultsModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Active         types.Bool   `tfsdk:"active"`
	LastEditor     types.String `tfsdk:"last_editor"`
	CreatedAt      types.String `tfsdk:"created_at"`
	LastModified   types.String `tfsdk:"last_modified"`
	ProductVersion types.String `tfsdk:"product_version"`
	Issuer         types.String `tfsdk:"issuer"`
	LastUpdate     types.String `tfsdk:"last_update"`
	NextUpdate     types.String `tfsdk:"next_update"`
	Crl            types.String `tfsdk:"crl"`
}

func (c *CrlDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c.client = req.ProviderData.(*apiClient)
}

func (c *CrlDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crl"
}

func (c *CrlDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the data source.",
				Computed:    true,
			},
			"crl_id": schema.Int64Attribute{
				Description: "Identifier of the certificate revocation list.",
				Required:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"results": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Identifier of the certificate revocation list.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the certificate revocation list.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Indicates if the certificate revocation list is active.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "Last editor of the certificate revocation list.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "Timestamp of the certificate revocation list creation on the platform.",
						Computed:    true,
					},
					"last_modified": schema.StringAttribute{
						Description: "Timestamp of the last modification made to the certificate content on the platform.",
						Computed:    true,
					},
					"product_version": schema.StringAttribute{
						Description: "Product version of the certificate revocation list.",
						Computed:    true,
					},
					"issuer": schema.StringAttribute{
						Description: "Issuer of the certificate revocation list.",
						Computed:    true,
					},
					"last_update": schema.StringAttribute{
						Description: "Timestamp of the last update issued by the certification revocation list issuer.",
						Computed:    true,
					},
					"next_update": schema.StringAttribute{
						Description: "Timestamp of the next scheduled update from the certification revocation list issuer.",
						Computed:    true,
					},
					"crl": schema.StringAttribute{
						Description: "The certificate revocation list content.",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (c *CrlDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var getCrlID types.Int64
	diags := req.Config.GetAttribute(ctx, path.Root("crl_id"), &getCrlID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	crlResponse, response, err := c.client.api.DigitalCertificatesCertificateRevocationListsAPI.RetrieveCertificateRevocationList(ctx, getCrlID.ValueInt64()).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			crlResponse, response, err = utils.RetryOn429(func() (*azionapi.CertificateRevocationListResponse, *http.Response, error) {
				return c.client.api.DigitalCertificatesCertificateRevocationListsAPI.RetrieveCertificateRevocationList(ctx, getCrlID.ValueInt64()).Execute()
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
	} else {
		if response != nil {
			defer response.Body.Close()
		}
	}

	// Populate the results from the API response
	crlState := populateCrlResults(crlResponse.GetData(), getCrlID)
	crlState.ID = types.StringValue("Get By ID Certificate Revocation List")
	diags = resp.State.Set(ctx, &crlState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// populateCrlResults transforms API response data to Terraform state model.
func populateCrlResults(crl azionapi.CertificateRevocationList, crlID types.Int64) CrlDataSourceModel {
	var createdAt string
	if crl.CreatedAt.IsSet() && crl.CreatedAt.Get() != nil {
		createdAt = (*crl.CreatedAt.Get()).Format(time.RFC3339)
	}

	result := CrlDataSourceModel{
		CrlID:         crlID,
		SchemaVersion: types.Int64Value(1),
		Results: &CrlResultsModel{
			ID:             types.Int64Value(crl.GetId()),
			Name:           types.StringValue(crl.GetName()),
			LastEditor:     types.StringValue(crl.GetLastEditor()),
			CreatedAt:      types.StringValue(createdAt),
			LastModified:   types.StringValue(crl.GetLastModified().Format(time.RFC3339)),
			ProductVersion: types.StringValue(crl.GetProductVersion()),
			Issuer:         types.StringValue(crl.GetIssuer()),
			LastUpdate:     types.StringValue(crl.GetLastUpdate().Format(time.RFC3339)),
			NextUpdate:     types.StringValue(crl.GetNextUpdate().Format(time.RFC3339)),
			Crl:            types.StringValue(crl.GetCrl()),
		},
	}

	// Handle optional fields
	if crl.Active != nil {
		result.Results.Active = types.BoolValue(*crl.Active)
	}

	return result
}
