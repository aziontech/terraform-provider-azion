package provider

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &recordResource{}
	_ resource.ResourceWithConfigure   = &recordResource{}
	_ resource.ResourceWithImportState = &recordResource{}
)

func NewRecordResource() resource.Resource {
	return &recordResource{}
}

type recordResource struct {
	client *idns.APIClient
}

type recordResourceModel struct {
	// ID            types.String `tfsdk:"id"`
	Record        *recordModel `tfsdk:"record"`
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	LastUpdated   types.String `tfsdk:"last_updated"`
	// ZoneId        types.Int64  `tfsdk:"zone_id"`
	ZoneId types.String `tfsdk:"zone_id"`
}

type recordModel struct {
	Id          types.Int64    `tfsdk:"id"`
	AnswersList []types.String `tfsdk:"answers_list"`
	RecordType  types.String   `tfsdk:"record_type"`
	Ttl         types.Int64    `tfsdk:"ttl"`
	Policy      types.String   `tfsdk:"policy"`
	Entry       types.String   `tfsdk:"entry"`
	// Description types.String   `tfsdk:"description"`
}

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_records"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// "id": schema.StringAttribute{
			// 	Description: "Numeric identifier.",
			// 	Computed:    true,
			// 	PlanModifiers: []planmodifier.String{
			// 		stringplanmodifier.UseStateForUnknown(),
			// 	},
			// },
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
			},
			// "zone_id": schema.StringAttribute{
			"zone_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"record": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed: true,
					},
					"entry": schema.StringAttribute{
						Required: true,
					},
					// "description": schema.StringAttribute{
					// 	Optional: true,
					// },
					"answers_list": schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
					},
					"policy": schema.StringAttribute{
						Required: true,
					},
					"record_type": schema.StringAttribute{
						Required: true,
					},
					"ttl": schema.Int64Attribute{
						Required: true,
					},
				},
			},
		},
	}
}

func (r *recordResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*idns.APIClient)
}

func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan recordResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	record := idns.RecordPostOrPut{
		RecordType: idns.PtrString(plan.Record.RecordType.ValueString()),
		Entry:      idns.PtrString(plan.Record.Entry.ValueString()),
		// Description: idns.PtrString(plan.Results.Description.ValueString()),
		Ttl: idns.PtrInt32(int32(plan.Record.Ttl.ValueInt64())),
	}

	for _, answerList := range plan.Record.AnswersList {
		record.AnswersList = append(record.AnswersList, answerList.ValueString())
	}

	zoneId, err := strconv.Atoi(plan.ZoneId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}
	createRecord, response, err := r.client.RecordsApi.PostZoneRecord(ctx, int32(zoneId)).RecordPostOrPut(record).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
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

	// plan.ID = types.StringValue(strconv.Itoa(int(*createRecord.Results.Id)))
	plan.SchemaVersion = types.Int64Value(int64(*createRecord.SchemaVersion))

	var slice []types.String
	for _, answerList := range createRecord.Results.AnswersList {
		slice = append(slice, types.StringValue(answerList))
	}

	plan.ZoneId = types.StringValue(plan.ZoneId.ValueString())

	plan.Record = &recordModel{
		Id:         types.Int64Value(int64(*createRecord.Results.Id)),
		RecordType: types.StringValue(*createRecord.Results.RecordType),
		Ttl:        types.Int64Value(int64(*createRecord.Results.Ttl)),
		Policy:     types.StringValue(*createRecord.Results.Policy),
		Entry:      types.StringValue(*createRecord.Results.Entry),
		// Description: types.StringValue(*createRecord.Results.Description),
		AnswersList: slice,
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *recordResource) AtoiNoError(strToConv types.String) int64 {
	intReturn, err := strconv.ParseInt(strToConv.String(), 10, 64)
	if err != nil {
		fmt.Println("Error converting data: ", err)
	}
	return intReturn
}

func (r *recordResource) errorPrint(resp *resource.ReadResponse, errCode int) {
	var usrMsg string
	switch errCode {
	case 404:
		usrMsg = "No Records Found"
	case 401:
		usrMsg = "Unauthorized Token"
	default:
		usrMsg = "Cannot read Azion response"
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	resp.Diagnostics.AddError(
		usrMsg,
		errMsg,
	)
}

func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading Records")

	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// idState := state.ZoneId.ValueString()
	idState, err := strconv.Atoi(state.ZoneId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	// idState, _ := strconv.Atoi(path.Root("zone_id").String())

	recordsResponse, httpResp, err := r.client.RecordsApi.GetZoneRecords(ctx, int32(idState)).Execute()
	if err != nil {
		r.errorPrint(resp, httpResp.StatusCode)
		return
	}

	state.SchemaVersion = types.Int64Value(int64(*recordsResponse.SchemaVersion))

	// state.ZoneId = types.Int64Value(int64(int32(idState)))

	for _, resultRecord := range recordsResponse.Results.Records {
		state.Record = &recordModel{
			Id:         types.Int64Value(int64(*resultRecord.RecordId)),
			RecordType: types.StringValue(*resultRecord.RecordType),
			Ttl:        types.Int64Value(int64(*resultRecord.Ttl)),
			Policy:     types.StringValue(*resultRecord.Policy),
			Entry:      types.StringValue(*resultRecord.Entry),
			// Description: types.StringValue(*resultRecord.Description),
		}
		for _, answer := range resultRecord.AnswersList {
			state.Record.AnswersList = append(state.Record.AnswersList, types.StringValue(answer))
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *recordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recordResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state recordResourceModel
	diags2 := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	// idPlan := plan.ZoneId.ValueInt64()

	idPlan, err := strconv.Atoi(plan.ZoneId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	record := idns.RecordPostOrPut{
		// Id:         idns.PtrInt32(int32(state.Results.Id.ValueInt64())),
		Entry:      idns.PtrString(plan.Record.Entry.ValueString()),
		Policy:     idns.PtrString(plan.Record.Policy.ValueString()),
		RecordType: idns.PtrString(plan.Record.RecordType.ValueString()),
		Ttl:        idns.PtrInt32(int32(plan.Record.Ttl.ValueInt64())),
	}

	for _, planAnswerList := range plan.Record.AnswersList {
		record.AnswersList = append(record.AnswersList, planAnswerList.ValueString())
	}

	updateRecord, response, err := r.client.RecordsApi.PutZoneRecord(ctx, int32(idPlan), int32(state.Record.Id.ValueInt64())).RecordPostOrPut(record).Execute()
	if err != nil {
		bodyBytes, erro := io.ReadAll(response.Body)
		if erro != nil {
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

	plan.Record.Id = types.Int64Value(int64(idPlan))
	plan.SchemaVersion = types.Int64Value(int64(*updateRecord.SchemaVersion))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	var answerList []types.String
	for _, resultRecord := range updateRecord.Results.AnswersList {
		answerList = append(answerList, types.StringValue(string(resultRecord)))
	}

	plan.Record = &recordModel{
		Id:         types.Int64Value(int64(*updateRecord.Results.Id)),
		RecordType: types.StringValue(*updateRecord.Results.RecordType),
		Ttl:        types.Int64Value(int64(*updateRecord.Results.Ttl)),
		Policy:     types.StringValue(*updateRecord.Results.Policy),
		Entry:      types.StringValue(*updateRecord.Results.Entry),
		// Description: types.StringValue(*updateRecord.Results.Description),
		AnswersList: answerList,
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *recordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idState, err := strconv.Atoi(state.ZoneId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not conversion ID",
		)
		return
	}

	_, _, err = r.client.RecordsApi.DeleteZoneRecord(ctx, int32(idState), int32(state.Record.Id.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Azion API",
			"Could not read azion API "+err.Error(),
		)
		return
	}
}

func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
