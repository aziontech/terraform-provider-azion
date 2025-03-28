package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aziontech/azionapi-go-sdk/idns"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
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
	client *apiClient
}

type recordResourceModel struct {
	ZoneId        types.String `tfsdk:"zone_id"`
	Record        *recordModel `tfsdk:"record"`
	SchemaVersion types.Int64  `tfsdk:"schema_version"`
	LastUpdated   types.String `tfsdk:"last_updated"`
}

type recordModel struct {
	Id          types.Int64    `tfsdk:"id"`
	AnswersList []types.String `tfsdk:"answers_list"`
	RecordType  types.String   `tfsdk:"record_type"`
	Ttl         types.Int64    `tfsdk:"ttl"`
	Policy      types.String   `tfsdk:"policy"`
	Entry       types.String   `tfsdk:"entry"`
	Weight      types.Int64    `tfsdk:"weight"`
	Description types.String   `tfsdk:"description"`
}

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_record"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"schema_version": schema.Int64Attribute{
				Description: "Schema Version.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
			"zone_id": schema.StringAttribute{
				Description: "The zone identifier to target for the resource.",
				Required:    true,
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
					"record_type": schema.StringAttribute{
						Required:    true,
						Description: "Defines the record type (A, CNAME, MX, NS).",
					},
					"entry": schema.StringAttribute{
						Required:    true,
						Description: "The first part of domain or 'Name'.",
					},
					"answers_list": schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of answers replied by DNS Authoritative to that Record.",
					},
					"policy": schema.StringAttribute{
						Required:    true,
						Description: "Must be 'simple' or 'weighted'.",
					},
					"weight": schema.Int64Attribute{
						Optional:    true,
						Description: "You can only use this field when policy is 'weighted'.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Description: "You can only use this field when policy is 'weighted'.",
					},
					"ttl": schema.Int64Attribute{
						Required:    true,
						Description: "Time-to-live defines max-time for packets life in seconds.",
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

	r.client = req.ProviderData.(*apiClient)
}

func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recordResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	recordTlg32, err := utils.CheckInt64toInt32Security(plan.Record.Ttl.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, plan.Record.Ttl.ValueInt64())
		return
	}

	record := idns.RecordPostOrPut{
		RecordType: idns.PtrString(plan.Record.RecordType.ValueString()),
		Entry:      idns.PtrString(plan.Record.Entry.ValueString()),
		Ttl:        idns.PtrInt32(recordTlg32),
		Policy:     idns.PtrString(plan.Record.Policy.ValueString()),
	}

	recordWeigh32, err := utils.CheckInt64toInt32Security(plan.Record.Weight.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, plan.Record.Weight.ValueInt64())
		return
	}

	if plan.Record.Policy.ValueString() == "weighted" {
		if idns.PtrInt32(recordWeigh32) != nil {
			record.Weight = idns.PtrInt32(recordWeigh32)
		}
		if idns.PtrString(plan.Record.Description.ValueString()) != nil {
			record.Description = idns.PtrString(plan.Record.Description.ValueString())
		}
	} else {
		plan.Record.Weight = types.Int64Value(0)
		plan.Record.Description = types.StringValue("")
		record.Weight = idns.PtrInt32(50)
		record.Description = idns.PtrString("")
	}

	for _, answerList := range plan.Record.AnswersList {
		record.AnswersList = append(record.AnswersList, answerList.ValueString())
	}

	zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	createRecord, httpResponse, err := r.client.idnsApi.RecordsAPI.PostZoneRecord(ctx, int32(zoneId)).RecordPostOrPut(record).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			createRecord, httpResponse, err = utils.RetryOn429(func() (*idns.PostOrPutRecordResponse, *http.Response, error) {
				return r.client.idnsApi.RecordsAPI.PostZoneRecord(ctx, int32(zoneId)).RecordPostOrPut(record).Execute() //nolint
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, _ := errorPrint(httpResponse.StatusCode, err)
			bodyBytes, _ := io.ReadAll(httpResponse.Body)
			resp.Diagnostics.AddError(usrMsg, string(bodyBytes))

			return
		}
	}

	plan.SchemaVersion = types.Int64Value(int64(*createRecord.SchemaVersion))

	var slice []types.String
	for _, answerList := range createRecord.Results.AnswersList {
		slice = append(slice, types.StringValue(answerList))
	}

	plan.ZoneId = types.StringValue(plan.ZoneId.ValueString())

	plan.Record = &recordModel{
		Id:          types.Int64Value(int64(*createRecord.Results.Id)),
		RecordType:  types.StringValue(*createRecord.Results.RecordType),
		Ttl:         types.Int64Value(int64(*createRecord.Results.Ttl)),
		Policy:      types.StringValue(*createRecord.Results.Policy),
		Entry:       types.StringValue(*createRecord.Results.Entry),
		AnswersList: slice,
	}

	if plan.Record.Policy.ValueString() == "weighted" {
		if createRecord.Results.Weight != nil {
			plan.Record.Weight = types.Int64Value(int64(*createRecord.Results.Weight))
		}
		if *createRecord.Results.Description != "" {
			plan.Record.Description = types.StringValue(*createRecord.Results.Description)
		}
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func errorPrint(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "No Records Found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}

func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading Records")

	// Since we must find the target record in a paginated result set from the API,
	// we must use a large page size so we can retrieve (probably) all records.
	// It's just a workaround until a GET /record/{record_id} API endpoint is available.
	// TODO: Change this once a GET /record/{record_id} API endpoint is available.
	largeRecordsPageSize := int64(1000000)

	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueFromCmd := strings.Split(state.ZoneId.ValueString(), "/")
	idZone := utils.AtoiNoError(valueFromCmd[0], resp)
	state.ZoneId = types.StringValue(valueFromCmd[0])
	var idRecord int32
	if len(valueFromCmd) > 1 {
		idRecord = utils.AtoiNoError(valueFromCmd[1], resp)
	}

	recordsResponse, httpResponse, err := r.client.idnsApi.RecordsAPI.
		GetZoneRecords(ctx, idZone).PageSize(largeRecordsPageSize).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResponse.StatusCode == 429 {
			recordsResponse, httpResponse, err = utils.RetryOn429(func() (*idns.GetRecordsResponse, *http.Response, error) {
				return r.client.idnsApi.RecordsAPI.GetZoneRecords(ctx, idZone).PageSize(largeRecordsPageSize).Execute() //nolint
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errorPrint(httpResponse.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	state.SchemaVersion = types.Int64Value(int64(*recordsResponse.SchemaVersion))

	for _, resultRecord := range recordsResponse.Results.Records {
		if types.Int64Value(int64(*resultRecord.RecordId)) != types.Int64Value(int64(idRecord)) {
			continue
		}
		state.Record = &recordModel{
			Id:         types.Int64Value(int64(*resultRecord.RecordId)),
			RecordType: types.StringValue(*resultRecord.RecordType),
			Ttl:        types.Int64Value(int64(*resultRecord.Ttl)),
			Policy:     types.StringValue(*resultRecord.Policy),
			Entry:      types.StringValue(*resultRecord.Entry),
		}
		for _, answer := range resultRecord.AnswersList {
			state.Record.AnswersList = append(state.Record.AnswersList, types.StringValue(answer))
		}

		if *resultRecord.Policy == "weighted" {
			state.Record.Weight = types.Int64Value(int64(*resultRecord.Weight))
			state.Record.Description = types.StringValue(*resultRecord.Description)
		}
	}
	if state.Record == nil {
		resp.Diagnostics.AddError("Record not found", fmt.Sprintf("RecordID %v not found in zoneID %v", idRecord, idZone))
		return
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

	idPlan, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	recordTlg32, err := utils.CheckInt64toInt32Security(plan.Record.Ttl.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, plan.Record.Ttl.ValueInt64())
		return
	}

	recordWeight32, err := utils.CheckInt64toInt32Security(plan.Record.Weight.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, plan.Record.Weight.ValueInt64())
		return
	}

	record := idns.RecordPostOrPut{
		Entry:       idns.PtrString(plan.Record.Entry.ValueString()),
		Policy:      idns.PtrString(plan.Record.Policy.ValueString()),
		RecordType:  idns.PtrString(plan.Record.RecordType.ValueString()),
		Ttl:         idns.PtrInt32(recordTlg32),
		Weight:      idns.PtrInt32(recordWeight32),
		Description: idns.PtrString(plan.Record.Description.ValueString()),
	}

	for _, planAnswerList := range plan.Record.AnswersList {
		record.AnswersList = append(record.AnswersList, planAnswerList.ValueString())
	}

	planID32, err := utils.CheckInt64toInt32Security(idPlan)
	if err != nil {
		utils.ExceedsValidRange(resp, idPlan)
		return
	}

	recordID32, err := utils.CheckInt64toInt32Security(state.Record.Id.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, state.Record.Id.ValueInt64())
		return
	}

	updateRecord, httpResponse, err := r.client.idnsApi.RecordsAPI.
		PutZoneRecord(ctx, planID32, recordID32).
		RecordPostOrPut(record).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			updateRecord, httpResponse, err = utils.RetryOn429(func() (*idns.PostOrPutRecordResponse, *http.Response, error) {
				return r.client.idnsApi.RecordsAPI.PutZoneRecord(ctx, planID32, recordID32).RecordPostOrPut(record).Execute() //nolint
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, _ := errorPrint(httpResponse.StatusCode, err)
			bodyBytes, _ := io.ReadAll(httpResponse.Body)
			resp.Diagnostics.AddError(usrMsg, string(bodyBytes))
			return
		}
	}

	plan.Record.Id = types.Int64Value(int64(idPlan))
	plan.SchemaVersion = types.Int64Value(int64(*updateRecord.SchemaVersion))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	var answerList []types.String
	for _, resultRecord := range updateRecord.Results.AnswersList {
		answerList = append(answerList, types.StringValue(string(resultRecord)))
	}

	plan.Record = &recordModel{
		Id:          types.Int64Value(int64(*updateRecord.Results.Id)),
		RecordType:  types.StringValue(*updateRecord.Results.RecordType),
		Ttl:         types.Int64Value(int64(*updateRecord.Results.Ttl)),
		Policy:      types.StringValue(*updateRecord.Results.Policy),
		Entry:       types.StringValue(*updateRecord.Results.Entry),
		AnswersList: answerList,
	}

	if plan.Record.Policy.ValueString() == "weighted" {
		plan.Record.Weight = types.Int64Value(int64(*updateRecord.Results.Weight))
		plan.Record.Description = types.StringValue(*updateRecord.Results.Description)
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

	stateID, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	stateID32, err := utils.CheckInt64toInt32Security(stateID)
	if err != nil {
		utils.ExceedsValidRange(resp, stateID)
		return
	}

	recordID32, err := utils.CheckInt64toInt32Security(state.Record.Id.ValueInt64())
	if err != nil {
		utils.ExceedsValidRange(resp, state.Record.Id.ValueInt64())
		return
	}

	_, httpResponse, err := r.client.idnsApi.RecordsAPI.DeleteZoneRecord(ctx, stateID32, recordID32).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			_, httpResponse, err = utils.RetryOn429(func() (string, *http.Response, error) {
				return r.client.idnsApi.RecordsAPI.DeleteZoneRecord(ctx, stateID32, recordID32).Execute() //nolint
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close() // <-- Close the body here
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"Error Reading Azion API",
				"Could not read azion API "+err.Error(),
			)
			return
		}
	}
}

func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}
