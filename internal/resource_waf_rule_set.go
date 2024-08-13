package provider

import (
	"context"
	waf "github.com/aziontech/azionapi-go-sdk/waf"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"io"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &wafRuleSetResource{}
	_ resource.ResourceWithConfigure   = &wafRuleSetResource{}
	_ resource.ResourceWithImportState = &wafRuleSetResource{}
)

func WafRuleSetResource() resource.Resource {
	return &wafRuleSetResource{}
}

type wafRuleSetResource struct {
	client *apiClient
}

type WafRuleSetResourceModel struct {
	WafRuleSet  *WafRuleSetResourceResults `tfsdk:"result"`
	ID          types.String               `tfsdk:"id"`
	LastUpdated types.String               `tfsdk:"last_updated"`
}

type WafRuleSetResourceResults struct {
	ID                             types.Int64  `tfsdk:"waf_id"`
	Name                           types.String `tfsdk:"name"`
	Mode                           types.String `tfsdk:"mode"`
	Active                         types.Bool   `tfsdk:"active"`
	SQLInjection                   types.Bool   `tfsdk:"sql_injection"`
	SQLInjectionSensitivity        types.String `tfsdk:"sql_injection_sensitivity"`
	RemoteFileInclusion            types.Bool   `tfsdk:"remote_file_inclusion"`
	RemoteFileInclusionSensitivity types.String `tfsdk:"remote_file_inclusion_sensitivity"`
	DirectoryTraversal             types.Bool   `tfsdk:"directory_traversal"`
	DirectoryTraversalSensitivity  types.String `tfsdk:"directory_traversal_sensitivity"`
	CrossSiteScripting             types.Bool   `tfsdk:"cross_site_scripting"`
	CrossSiteScriptingSensitivity  types.String `tfsdk:"cross_site_scripting_sensitivity"`
	EvadingTricks                  types.Bool   `tfsdk:"evading_tricks"`
	EvadingTricksSensitivity       types.String `tfsdk:"evading_tricks_sensitivity"`
	FileUpload                     types.Bool   `tfsdk:"file_upload"`
	FileUploadSensitivity          types.String `tfsdk:"file_upload_sensitivity"`
	UnwantedAccess                 types.Bool   `tfsdk:"unwanted_access"`
	UnwantedAccessSensitivity      types.String `tfsdk:"unwanted_access_sensitivity"`
	IdentifiedAttack               types.Bool   `tfsdk:"identified_attack"`
	IdentifiedAttackSensitivity    types.String `tfsdk:"identified_attack_sensitivity"`
	BypassAddresses                types.Set    `tfsdk:"bypass_addresses"`
}

func (r *wafRuleSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_waf_rule_set"
}

func (r *wafRuleSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"result": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"waf_id": schema.Int64Attribute{
						Description: "The WAF identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Name of the WAF configuration.",
						Required:    true,
					},
					"mode": schema.StringAttribute{
						Description: "WAF mode (e.g., counting).",
						Required:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Whether the WAF is active.",
						Required:    true,
					},
					"sql_injection": schema.BoolAttribute{
						Description: "Enable SQL injection protection.",
						Required:    true,
					},
					"sql_injection_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for SQL injection protection.",
						Required:    true,
					},
					"remote_file_inclusion": schema.BoolAttribute{
						Description: "Enable remote file inclusion protection.",
						Required:    true,
					},
					"remote_file_inclusion_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for remote file inclusion protection.",
						Required:    true,
					},
					"directory_traversal": schema.BoolAttribute{
						Description: "Enable directory traversal protection.",
						Required:    true,
					},
					"directory_traversal_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for directory traversal protection.",
						Required:    true,
					},
					"cross_site_scripting": schema.BoolAttribute{
						Description: "Enable cross-site scripting protection.",
						Required:    true,
					},
					"cross_site_scripting_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for cross-site scripting protection.",
						Required:    true,
					},
					"evading_tricks": schema.BoolAttribute{
						Description: "Enable evading tricks protection.",
						Required:    true,
					},
					"evading_tricks_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for evading tricks protection.",
						Required:    true,
					},
					"file_upload": schema.BoolAttribute{
						Description: "Enable file upload protection.",
						Required:    true,
					},
					"file_upload_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for file upload protection.",
						Required:    true,
					},
					"unwanted_access": schema.BoolAttribute{
						Description: "Enable protection against unwanted access.",
						Required:    true,
					},
					"unwanted_access_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for protection against unwanted access.",
						Required:    true,
					},
					"identified_attack": schema.BoolAttribute{
						Description: "Enable protection against identified attacks.",
						Required:    true,
					},
					"identified_attack_sensitivity": schema.StringAttribute{
						Description: "Sensitivity level for protection against identified attacks.",
						Required:    true,
					},
					"bypass_addresses": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of bypass addresses.",
					},
				},
			},
		},
	}
}

func (r *wafRuleSetResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *wafRuleSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WafRuleSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	wafRulesetRequest := waf.CreateNewWAFRulesetRequest{
		Name:                           plan.WafRuleSet.Name.ValueString(),
		Mode:                           plan.WafRuleSet.Mode.ValueString(),
		Active:                         plan.WafRuleSet.Active.ValueBool(),
		SqlInjection:                   plan.WafRuleSet.SQLInjection.ValueBool(),
		SqlInjectionSensitivity:        waf.WAFSensitivityChoices(plan.WafRuleSet.SQLInjectionSensitivity.ValueString()),
		RemoteFileInclusion:            plan.WafRuleSet.RemoteFileInclusion.ValueBool(),
		RemoteFileInclusionSensitivity: waf.WAFSensitivityChoices(plan.WafRuleSet.RemoteFileInclusionSensitivity.ValueString()),
		DirectoryTraversal:             plan.WafRuleSet.DirectoryTraversal.ValueBool(),
		DirectoryTraversalSensitivity:  waf.WAFSensitivityChoices(plan.WafRuleSet.DirectoryTraversalSensitivity.ValueString()),
		CrossSiteScripting:             plan.WafRuleSet.CrossSiteScripting.ValueBool(),
		CrossSiteScriptingSensitivity:  waf.WAFSensitivityChoices(plan.WafRuleSet.CrossSiteScriptingSensitivity.ValueString()),
		EvadingTricks:                  plan.WafRuleSet.EvadingTricks.ValueBool(),
		EvadingTricksSensitivity:       waf.WAFSensitivityChoices(plan.WafRuleSet.EvadingTricksSensitivity.ValueString()),
		FileUpload:                     plan.WafRuleSet.FileUpload.ValueBool(),
		FileUploadSensitivity:          waf.WAFSensitivityChoices(plan.WafRuleSet.FileUploadSensitivity.ValueString()),
		UnwantedAccess:                 plan.WafRuleSet.UnwantedAccess.ValueBool(),
		UnwantedAccessSensitivity:      waf.WAFSensitivityChoices(plan.WafRuleSet.UnwantedAccessSensitivity.ValueString()),
		IdentifiedAttack:               plan.WafRuleSet.IdentifiedAttack.ValueBool(),
		IdentifiedAttackSensitivity:    waf.WAFSensitivityChoices(plan.WafRuleSet.IdentifiedAttackSensitivity.ValueString()),
	}

	requestAddresses := plan.WafRuleSet.BypassAddresses.ElementsAs(ctx, &wafRulesetRequest.BypassAddresses, false)
	resp.Diagnostics.Append(requestAddresses...)
	if resp.Diagnostics.HasError() {
		return
	}

	wafRuleSetResponse, response, err := r.client.wafApi.WAFAPI.CreateNewWAFRuleset(ctx).CreateNewWAFRulesetRequest(wafRulesetRequest).Execute()
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
	defer response.Body.Close()

	var sliceAddresses []types.String
	for _, Addresses := range wafRuleSetResponse.BypassAddresses {
		sliceAddresses = append(sliceAddresses, types.StringValue(Addresses))
	}
	plan.WafRuleSet = &WafRuleSetResourceResults{
		ID:                             types.Int64Value(wafRuleSetResponse.GetId()),
		Name:                           types.StringValue(wafRuleSetResponse.GetName()),
		Mode:                           types.StringValue(wafRuleSetResponse.GetMode()),
		Active:                         types.BoolValue(wafRuleSetResponse.GetActive()),
		BypassAddresses:                utils.SliceStringTypeToSetOrNull(sliceAddresses),
		SQLInjection:                   types.BoolValue(wafRuleSetResponse.GetSqlInjection()),
		SQLInjectionSensitivity:        types.StringValue(string(wafRuleSetResponse.GetSqlInjectionSensitivity())),
		RemoteFileInclusion:            types.BoolValue(wafRuleSetResponse.GetRemoteFileInclusion()),
		RemoteFileInclusionSensitivity: types.StringValue(string(wafRuleSetResponse.GetRemoteFileInclusionSensitivity())),
		DirectoryTraversal:             types.BoolValue(wafRuleSetResponse.GetDirectoryTraversal()),
		DirectoryTraversalSensitivity:  types.StringValue(string(wafRuleSetResponse.GetDirectoryTraversalSensitivity())),
		CrossSiteScripting:             types.BoolValue(wafRuleSetResponse.GetCrossSiteScripting()),
		CrossSiteScriptingSensitivity:  types.StringValue(string(wafRuleSetResponse.GetCrossSiteScriptingSensitivity())),
		EvadingTricks:                  types.BoolValue(wafRuleSetResponse.GetEvadingTricks()),
		EvadingTricksSensitivity:       types.StringValue(string(wafRuleSetResponse.GetEvadingTricksSensitivity())),
		FileUpload:                     types.BoolValue(wafRuleSetResponse.GetFileUpload()),
		FileUploadSensitivity:          types.StringValue(string(wafRuleSetResponse.GetFileUploadSensitivity())),
		UnwantedAccess:                 types.BoolValue(wafRuleSetResponse.GetUnwantedAccess()),
		UnwantedAccessSensitivity:      types.StringValue(string(wafRuleSetResponse.GetUnwantedAccessSensitivity())),
		IdentifiedAttack:               types.BoolValue(wafRuleSetResponse.GetIdentifiedAttack()),
		IdentifiedAttackSensitivity:    types.StringValue(string(wafRuleSetResponse.GetIdentifiedAttackSensitivity())),
	}

	plan.ID = types.StringValue(strconv.FormatInt(wafRuleSetResponse.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafRuleSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WafRuleSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var wafRuleSetID int64
	var err error
	if state.ID.IsNull() {
		wafRuleSetID = state.WafRuleSet.ID.ValueInt64()
	} else {
		wafRuleSetID, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WafRuleSet ID",
			)
			return
		}
	}

	wafResponse, response, err := r.client.wafApi.WAFAPI.GetWAFRuleset(ctx, wafRuleSetID).Execute()
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
	defer response.Body.Close()

	var sliceAddresses []types.String
	for _, Addresses := range wafResponse.Results.GetBypassAddresses() {
		sliceAddresses = append(sliceAddresses, types.StringValue(Addresses))
	}

	WafRuleSetState := WafRuleSetResourceModel{
		WafRuleSet: &WafRuleSetResourceResults{
			ID:                             types.Int64Value(wafResponse.Results.GetId()),
			Name:                           types.StringValue(wafResponse.Results.GetName()),
			Mode:                           types.StringValue(wafResponse.Results.GetMode()),
			Active:                         types.BoolValue(wafResponse.Results.GetActive()),
			BypassAddresses:                utils.SliceStringTypeToSetOrNull(sliceAddresses),
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
		},
		LastUpdated: types.StringValue(state.LastUpdated.ValueString()),
		ID:          types.StringValue(strconv.FormatInt(wafRuleSetID, 10)),
	}
	diags = resp.State.Set(ctx, &WafRuleSetState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafRuleSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WafRuleSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WafRuleSetResourceModel
	diagsNetworkList := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsNetworkList...)
	if resp.Diagnostics.HasError() {
		return
	}

	var wafRuleSetID int64
	var err error
	if state.ID.IsNull() {
		wafRuleSetID = state.WafRuleSet.ID.ValueInt64()
	} else {
		wafRuleSetID, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WafRuleSet ID",
			)
			return
		}
	}

	SqlInjectionSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.SQLInjectionSensitivity.ValueString())
	RemoteFileInclusionSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.RemoteFileInclusionSensitivity.ValueString())
	DirectoryTraversalSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.DirectoryTraversalSensitivity.ValueString())
	CrossSiteScriptingSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.CrossSiteScriptingSensitivity.ValueString())
	EvadingTricksSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.EvadingTricksSensitivity.ValueString())
	FileUploadSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.FileUploadSensitivity.ValueString())
	UnwantedAccessSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.UnwantedAccessSensitivity.ValueString())
	IdentifiedAttackSensitivity, _ := waf.NewWAFSensitivityChoicesFromValue(plan.WafRuleSet.IdentifiedAttackSensitivity.ValueString())

	wafRuleSetRequest := waf.SingleWAF{
		Name:                           plan.WafRuleSet.Name.ValueStringPointer(),
		Active:                         plan.WafRuleSet.Active.ValueBoolPointer(),
		SqlInjection:                   plan.WafRuleSet.SQLInjection.ValueBoolPointer(),
		SqlInjectionSensitivity:        SqlInjectionSensitivity,
		RemoteFileInclusion:            plan.WafRuleSet.RemoteFileInclusion.ValueBoolPointer(),
		RemoteFileInclusionSensitivity: RemoteFileInclusionSensitivity,
		DirectoryTraversal:             plan.WafRuleSet.DirectoryTraversal.ValueBoolPointer(),
		DirectoryTraversalSensitivity:  DirectoryTraversalSensitivity,
		CrossSiteScripting:             plan.WafRuleSet.CrossSiteScripting.ValueBoolPointer(),
		CrossSiteScriptingSensitivity:  CrossSiteScriptingSensitivity,
		EvadingTricks:                  plan.WafRuleSet.EvadingTricks.ValueBoolPointer(),
		EvadingTricksSensitivity:       EvadingTricksSensitivity,
		FileUpload:                     plan.WafRuleSet.FileUpload.ValueBoolPointer(),
		FileUploadSensitivity:          FileUploadSensitivity,
		UnwantedAccess:                 plan.WafRuleSet.UnwantedAccess.ValueBoolPointer(),
		UnwantedAccessSensitivity:      UnwantedAccessSensitivity,
		IdentifiedAttack:               plan.WafRuleSet.IdentifiedAttack.ValueBoolPointer(),
		IdentifiedAttackSensitivity:    IdentifiedAttackSensitivity,
	}

	requestAddresses := plan.WafRuleSet.BypassAddresses.ElementsAs(ctx, &wafRuleSetRequest.BypassAddresses, false)
	resp.Diagnostics.Append(requestAddresses...)
	if resp.Diagnostics.HasError() {
		return
	}

	wafRuleSetResponse, response, err := r.client.wafApi.WAFAPI.UpdateWAFRuleset(ctx, strconv.FormatInt(wafRuleSetID, 10)).SingleWAF(wafRuleSetRequest).Execute()
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
	defer response.Body.Close()

	var sliceAddresses []types.String
	for _, Addresses := range wafRuleSetResponse.GetBypassAddresses() {
		sliceAddresses = append(sliceAddresses, types.StringValue(Addresses))
	}
	plan.WafRuleSet = &WafRuleSetResourceResults{
		ID:                             types.Int64Value(wafRuleSetResponse.GetId()),
		Name:                           types.StringValue(wafRuleSetResponse.GetName()),
		Mode:                           types.StringValue(wafRuleSetResponse.GetMode()),
		Active:                         types.BoolValue(wafRuleSetResponse.GetActive()),
		BypassAddresses:                utils.SliceStringTypeToSetOrNull(sliceAddresses),
		SQLInjection:                   types.BoolValue(wafRuleSetResponse.GetSqlInjection()),
		SQLInjectionSensitivity:        types.StringValue(string(wafRuleSetResponse.GetSqlInjectionSensitivity())),
		RemoteFileInclusion:            types.BoolValue(wafRuleSetResponse.GetRemoteFileInclusion()),
		RemoteFileInclusionSensitivity: types.StringValue(string(wafRuleSetResponse.GetRemoteFileInclusionSensitivity())),
		DirectoryTraversal:             types.BoolValue(wafRuleSetResponse.GetDirectoryTraversal()),
		DirectoryTraversalSensitivity:  types.StringValue(string(wafRuleSetResponse.GetDirectoryTraversalSensitivity())),
		CrossSiteScripting:             types.BoolValue(wafRuleSetResponse.GetCrossSiteScripting()),
		CrossSiteScriptingSensitivity:  types.StringValue(string(wafRuleSetResponse.GetCrossSiteScriptingSensitivity())),
		EvadingTricks:                  types.BoolValue(wafRuleSetResponse.GetEvadingTricks()),
		EvadingTricksSensitivity:       types.StringValue(string(wafRuleSetResponse.GetEvadingTricksSensitivity())),
		FileUpload:                     types.BoolValue(wafRuleSetResponse.GetFileUpload()),
		FileUploadSensitivity:          types.StringValue(string(wafRuleSetResponse.GetFileUploadSensitivity())),
		UnwantedAccess:                 types.BoolValue(wafRuleSetResponse.GetUnwantedAccess()),
		UnwantedAccessSensitivity:      types.StringValue(string(wafRuleSetResponse.GetUnwantedAccessSensitivity())),
		IdentifiedAttack:               types.BoolValue(wafRuleSetResponse.GetIdentifiedAttack()),
		IdentifiedAttackSensitivity:    types.StringValue(string(wafRuleSetResponse.GetIdentifiedAttackSensitivity())),
	}

	plan.ID = types.StringValue(strconv.FormatInt(wafRuleSetResponse.GetId(), 10))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *wafRuleSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WafRuleSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var wafRuleSetID int64
	var err error
	if state.ID.IsNull() {
		wafRuleSetID = state.WafRuleSet.ID.ValueInt64()
	} else {
		wafRuleSetID, err = strconv.ParseInt(state.ID.ValueString(), 10, 32)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert WafRuleSet ID",
			)
			return
		}
	}

	response, err := r.client.wafApi.WAFAPI.DeleteWAFRuleset(ctx, strconv.FormatInt(wafRuleSetID, 10)).Execute()
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
	defer response.Body.Close()
}

func (r *wafRuleSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
