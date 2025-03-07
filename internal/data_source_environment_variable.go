package provider

import (
	"context"
	"io"
	"net/http"

	"github.com/aziontech/azionapi-go-sdk/variables"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &VariableDataSource{}
	_ datasource.DataSourceWithConfigure = &VariableDataSource{}
)

func dataSourceAzionVariable() datasource.DataSource {
	return &VariableDataSource{}
}

type VariableDataSource struct {
	client *apiClient
}

type VariableDataSourceModel struct {
	Result VariableResult `tfsdk:"result"`
	ID     types.String   `tfsdk:"id"`
}

type VariableResult struct {
	Uuid       types.String `tfsdk:"uuid"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
	Secret     types.Bool   `tfsdk:"secret"`
	LastEditor types.String `tfsdk:"last_editor"`
	CreateAt   types.String `tfsdk:"created_at"`
	UpdateAt   types.String `tfsdk:"updated_at"`
}

func (n *VariableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	n.client = req.ProviderData.(*apiClient)
}

func (n *VariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (n *VariableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true},

			"result": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						Description: "UUID of the environment variable.",
						Required:    true,
					},
					"key": schema.StringAttribute{
						Description: "Key of the environment variable.",
						Computed:    true,
					},
					"value": schema.StringAttribute{
						Description: "Value of the environment variable.",
						Computed:    true,
					},
					"secret": schema.BoolAttribute{
						Description: "Whether the variable is a secret or not.",
						Computed:    true,
					},
					"last_editor": schema.StringAttribute{
						Description: "The last user who edited the variable.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Computed:    true,
						Description: "Informs when the variable was created",
					},
					"updated_at": schema.StringAttribute{
						Computed:    true,
						Description: "The last time the variable was updated.",
					},
				},
			},
		},
	}
}

func (n *VariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid types.String

	diagsPhase := req.Config.GetAttribute(ctx, path.Root("result").AtName("uuid"), &uuid)
	resp.Diagnostics.Append(diagsPhase...)
	if resp.Diagnostics.HasError() {
		return
	}

	variableResponse, response, err := n.client.variablesApi.VariablesAPI.ApiVariablesRetrieve(ctx, uuid.ValueString()).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*variables.Variable, *http.Response, error) {
				return n.client.variablesApi.VariablesAPI.ApiVariablesRetrieve(ctx, uuid.ValueString()).Execute() //nolint
			}, 5) // Maximum 5 retries

			if response != nil {
				defer response.Body.Close() // <-- Close the body here
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
	}

	variablesResult := VariableResult{
		Uuid:       types.StringValue(variableResponse.GetUuid()),
		Key:        types.StringValue(variableResponse.GetKey()),
		Value:      types.StringValue(variableResponse.GetValue()),
		Secret:     types.BoolValue(variableResponse.GetSecret()),
		CreateAt:   types.StringValue(variableResponse.GetCreatedAt().String()),
		UpdateAt:   types.StringValue(variableResponse.GetUpdatedAt().String()),
		LastEditor: types.StringValue(variableResponse.GetLastEditor()),
	}
	variablesState := VariableDataSourceModel{
		Result: variablesResult,
		ID:     types.StringValue("Retrieve an environment variable by UUID"),
	}

	diags := resp.State.Set(ctx, &variablesState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
