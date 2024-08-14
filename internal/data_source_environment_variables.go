package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &VariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &VariablesDataSource{}
)

func dataSourceAzionVariables() datasource.DataSource {
	return &VariablesDataSource{}
}

type VariablesDataSource struct {
	client *apiClient
}

type VariablesDataSourceModel struct {
	Results []VariablesResults `tfsdk:"results"`
	ID      types.String       `tfsdk:"id"`
}

type VariablesResults struct {
	Uuid       types.String `tfsdk:"uuid"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
	Secret     types.Bool   `tfsdk:"secret"`
	LastEditor types.String `tfsdk:"last_editor"`
	CreateAt   types.String `tfsdk:"created_at"`
	UpdateAt   types.String `tfsdk:"updated_at"`
}

func (n *VariablesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	n.client = req.ProviderData.(*apiClient)
}

func (n *VariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variables"
}

func (n *VariablesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Optional:    true},

			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
		},
	}
}

func (n *VariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	variablesResponse, response, err := n.client.variablesApi.VariablesAPI.ApiVariablesList(ctx).Execute() //nolint
	if err != nil {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				err.Error(),
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

	var variablesList []VariablesResults
	for _, variable := range variablesResponse {
		variablesResults := VariablesResults{
			Uuid:       types.StringValue(variable.GetUuid()),
			Key:        types.StringValue(variable.GetKey()),
			Value:      types.StringValue(variable.GetValue()),
			Secret:     types.BoolValue(variable.GetSecret()),
			CreateAt:   types.StringValue(variable.GetCreatedAt().String()),
			UpdateAt:   types.StringValue(variable.GetUpdatedAt().String()),
			LastEditor: types.StringValue(variable.GetLastEditor()),
		}
		variablesList = append(variablesList, variablesResults)
	}

	variablesState := VariablesDataSourceModel{
		Results: variablesList,
		ID:      types.StringValue("List all environment variables"),
	}

	diags := resp.State.Set(ctx, &variablesState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
