package provider

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &WafRuleSetDataSource{}
	_ datasource.DataSourceWithConfigure = &WafRuleSetDataSource{}
)

func dataSourceAzionWafRuleSet() datasource.DataSource {
	return &WafRuleSetDataSource{}
}

type WafRuleSetDataSource struct {
	client *apiClient
}

type WafRuleSetDataSourceModel struct {
	SchemaVersion types.Int64        `tfsdk:"schema_version"`
	ID            types.String       `tfsdk:"id"`
	Results       *WafRuleSetResults `tfsdk:"result"`
}

type WafRuleSetResults struct {
	ID                             types.Int64    `tfsdk:"waf_id"`
	Name                           types.String   `tfsdk:"name"`
	Mode                           types.String   `tfsdk:"mode"`
	Active                         types.Bool     `tfsdk:"active"`
	SQLInjection                   types.Bool     `tfsdk:"sql_injection"`
	SQLInjectionSensitivity        types.String   `tfsdk:"sql_injection_sensitivity"`
	RemoteFileInclusion            types.Bool     `tfsdk:"remote_file_inclusion"`
	RemoteFileInclusionSensitivity types.String   `tfsdk:"remote_file_inclusion_sensitivity"`
	DirectoryTraversal             types.Bool     `tfsdk:"directory_traversal"`
	DirectoryTraversalSensitivity  types.String   `tfsdk:"directory_traversal_sensitivity"`
	CrossSiteScripting             types.Bool     `tfsdk:"cross_site_scripting"`
	CrossSiteScriptingSensitivity  types.String   `tfsdk:"cross_site_scripting_sensitivity"`
	EvadingTricks                  types.Bool     `tfsdk:"evading_tricks"`
	EvadingTricksSensitivity       types.String   `tfsdk:"evading_tricks_sensitivity"`
	FileUpload                     types.Bool     `tfsdk:"file_upload"`
	FileUploadSensitivity          types.String   `tfsdk:"file_upload_sensitivity"`
	UnwantedAccess                 types.Bool     `tfsdk:"unwanted_access"`
	UnwantedAccessSensitivity      types.String   `tfsdk:"unwanted_access_sensitivity"`
	IdentifiedAttack               types.Bool     `tfsdk:"identified_attack"`
	IdentifiedAttackSensitivity    types.String   `tfsdk:"identified_attack_sensitivity"`
	BypassAddresses                []types.String `tfsdk:"bypass_addresses"`
}

func (o *WafRuleSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	o.client = req.ProviderData.(*apiClient)
}

func (o *WafRuleSetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_set"
}

func (o *WafRuleSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the data source.",
				Computed:    true,
			},
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"result": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"waf_id": schema.Int64Attribute{
						Description: "The WAF identifier.",
						Required:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the WAF configuration.",
						Computed:    true,
					},
					"mode": schema.StringAttribute{
						Description: "WAF mode (e.g., counting).",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the WAF is active.",
						Computed:    true,
					},
					"sql_injection": schema.BoolAttribute{
						Description: "Enable SQL injection protection.",
						Computed:    true,
					},
					"sql_injection_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for SQL injection protection.",
						Computed:    true,
					},
					"remote_file_inclusion": schema.BoolAttribute{
						Description: "Enable remote file inclusion protection.",
						Computed:    true,
					},
					"remote_file_inclusion_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for remote file inclusion protection.",
						Computed:    true,
					},
					"directory_traversal": schema.BoolAttribute{
						Description: "Enable directory traversal protection.",
						Computed:    true,
					},
					"directory_traversal_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for directory traversal protection.",
						Computed:    true,
					},
					"cross_site_scripting": schema.BoolAttribute{
						Description: "Enable cross-site scripting protection.",
						Computed:    true,
					},
					"cross_site_scripting_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for cross-site scripting protection.",
						Computed:    true,
					},
					"evading_tricks": schema.BoolAttribute{
						Description: "Enable evading tricks protection.",
						Computed:    true,
					},
					"evading_tricks_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for evading tricks protection.",
						Computed:    true,
					},
					"file_upload": schema.BoolAttribute{
						Description: "Enable file upload protection.",
						Computed:    true,
					},
					"file_upload_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for file upload protection.",
						Computed:    true,
					},
					"unwanted_access": schema.BoolAttribute{
						Description: "Enable protection against unwanted access.",
						Computed:    true,
					},
					"unwanted_access_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for protection against unwanted access.",
						Computed:    true,
					},
					"identified_attack": schema.BoolAttribute{
						Description: "Enable protection against identified attacks.",
						Computed:    true,
					},
					"identified_attack_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for protection against identified attacks.",
						Computed:    true,
					},
					"bypass_addresses": schema.ListAttribute{
						Description: "List of bypass addresses.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (o *WafRuleSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var wafRuleSetID types.Int64

	diagsPhase := req.Config.GetAttribute(ctx, path.Root("result").AtName("waf_id"), &wafRuleSetID)
	resp.Diagnostics.Append(diagsPhase...)
	if resp.Diagnostics.HasError() {
		return
	}

	wafResponse, response, err := o.client.wafApi.WAFAPI.GetWAFRuleset(ctx, wafRuleSetID.ValueInt64()).Execute() //nolint
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

	var sliceAddresses []types.String
	for _, Addresses := range wafResponse.Results.GetBypassAddresses() {
		sliceAddresses = append(sliceAddresses, types.StringValue(Addresses))
	}
	wafResults := &WafRuleSetResults{
		ID:                             types.Int64Value(wafResponse.Results.GetId()),
		Name:                           types.StringValue(wafResponse.Results.GetName()),
		Mode:                           types.StringValue(wafResponse.Results.GetMode()),
		Active:                         types.BoolValue(wafResponse.Results.GetActive()),
		BypassAddresses:                sliceAddresses,
		SQLInjection:                   types.BoolValue(wafResponse.Results.GetSqlInjection()),
		SQLInjectionSensitivity:        types.StringValue(string(wafResponse.Results.GetSqlInjectionSensitivity())),
		RemoteFileInclusion:            types.BoolValue(wafResponse.Results.GetRemoteFileInclusion()),
		RemoteFileInclusionSensitivity: types.StringValue(string(wafResponse.Results.GetRemoteFileInclusionSensitivity())),
		DirectoryTraversal:             types.BoolValue(wafResponse.Results.GetDirectoryTraversal()),
		DirectoryTraversalSensitivity:  types.StringValue(string(wafResponse.Results.GetDirectoryTraversalSensitivity())),
		CrossSiteScripting:             types.BoolValue(wafResponse.Results.GetCrossSiteScripting()),
		CrossSiteScriptingSensitivity:  types.StringValue(string(wafResponse.Results.GetCrossSiteScriptingSensitivity())),
		EvadingTricks:                  types.BoolValue(wafResponse.Results.GetEvadingTricks()),
		EvadingTricksSensitivity:       types.StringValue(string(wafResponse.Results.GetEvadingTricksSensitivity())),
		FileUpload:                     types.BoolValue(wafResponse.Results.GetFileUpload()),
		FileUploadSensitivity:          types.StringValue(string(wafResponse.Results.GetFileUploadSensitivity())),
		UnwantedAccess:                 types.BoolValue(wafResponse.Results.GetUnwantedAccess()),
		UnwantedAccessSensitivity:      types.StringValue(string(wafResponse.Results.GetUnwantedAccessSensitivity())),
		IdentifiedAttack:               types.BoolValue(wafResponse.Results.GetIdentifiedAttack()),
		IdentifiedAttackSensitivity:    types.StringValue(string(wafResponse.Results.GetIdentifiedAttackSensitivity())),
	}

	wafRuleSetState := WafRuleSetDataSourceModel{
		SchemaVersion: types.Int64Value(int64(wafResponse.GetSchemaVersion())),
		Results:       wafResults,
	}

	wafRuleSetState.ID = types.StringValue("Get By ID Waf Rule Set")
	diags := resp.State.Set(ctx, &wafRuleSetState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
