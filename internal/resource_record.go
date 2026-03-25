package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	azionapi "github.com/aziontech/azionapi-v4-go-sdk-dev/azion-api"
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
	ZoneId      types.String `tfsdk:"zone_id"`
	Record      *recordModel `tfsdk:"record"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type recordModel struct {
	Id          types.Int64    `tfsdk:"id"`
	Rdata       []types.String `tfsdk:"rdata"`
	Type        types.String   `tfsdk:"type"`
	Ttl         types.Int64    `tfsdk:"ttl"`
	Policy      types.String   `tfsdk:"policy"`
	Name        types.String   `tfsdk:"name"`
	Weight      types.Int64    `tfsdk:"weight"`
	Description types.String   `tfsdk:"description"`
}

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_record"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
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
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Defines the record type (A, AAAA, ANAME, CNAME, MX, NS, PTR, SRV, TXT, CAA, DS).",
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the DNS record.",
					},
					"rdata": schema.ListAttribute{
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
						Description: "Description of the record.",
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

	zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	// Build the record request.
	recordReq := azionapi.NewRecordRequest(
		plan.Record.Name.ValueString(),
		plan.Record.Type.ValueString(),
		buildRdataList(plan.Record.Rdata),
	)

	// Set TTL.
	recordReq.SetTtl(plan.Record.Ttl.ValueInt64())

	// Set policy.
	recordReq.SetPolicy(plan.Record.Policy.ValueString())

	// Set weight and description for weighted policy.
	if plan.Record.Policy.ValueString() == "weighted" {
		if !plan.Record.Weight.IsNull() && !plan.Record.Weight.IsUnknown() {
			recordReq.SetWeight(plan.Record.Weight.ValueInt64())
		}
		if !plan.Record.Description.IsNull() && !plan.Record.Description.IsUnknown() {
			recordReq.SetDescription(plan.Record.Description.ValueString())
		}
	}

	// Execute create request.
	createRecord, httpResponse, err := r.client.api.DNSRecordsAPI.CreateDnsRecord(ctx, zoneId).
		RecordRequest(*recordReq).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			createRecord, httpResponse, err = utils.RetryOn429(func() (*azionapi.RecordResponse, *http.Response, error) {
				return r.client.api.DNSRecordsAPI.CreateDnsRecord(ctx, zoneId).
					RecordRequest(*recordReq).Execute()
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errorPrintRecord(httpResponse.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}

	// Update plan with response.
	plan.ZoneId = types.StringValue(plan.ZoneId.ValueString())
	plan.Record = populateRecordModel(createRecord.GetData(), plan.Record.Policy.ValueString())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading Record")

	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse zone_id and record_id from state.
	// Format: "zone_id/record_id" for import, or just "zone_id" for existing state.
	valueFromCmd := strings.Split(state.ZoneId.ValueString(), "/")
	zoneId, err := strconv.ParseInt(valueFromCmd[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	var recordId int64
	if len(valueFromCmd) > 1 {
		recordId, err = strconv.ParseInt(valueFromCmd[1], 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Value Conversion error ",
				"Could not convert Record ID",
			)
			return
		}
	} else if state.Record != nil && !state.Record.Id.IsNull() {
		recordId = state.Record.Id.ValueInt64()
	}

	// Retrieve the record.
	recordResponse, httpResponse, err := r.client.api.DNSRecordsAPI.RetrieveDnsRecord(ctx, recordId, zoneId).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResponse.StatusCode == 429 {
			recordResponse, httpResponse, err = utils.RetryOn429(func() (*azionapi.RecordResponse, *http.Response, error) {
				return r.client.api.DNSRecordsAPI.RetrieveDnsRecord(ctx, recordId, zoneId).Execute()
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errorPrintRecord(httpResponse.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}

	// Update state with response.
	state.ZoneId = types.StringValue(valueFromCmd[0])
	state.Record = populateRecordModel(recordResponse.GetData(), "")

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

	zoneId, err := strconv.ParseInt(plan.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	recordId := state.Record.Id.ValueInt64()

	// Build the record request.
	recordReq := azionapi.NewRecordRequest(
		plan.Record.Name.ValueString(),
		plan.Record.Type.ValueString(),
		buildRdataList(plan.Record.Rdata),
	)

	// Set TTL.
	recordReq.SetTtl(plan.Record.Ttl.ValueInt64())

	// Set policy.
	recordReq.SetPolicy(plan.Record.Policy.ValueString())

	// Set weight and description for weighted policy.
	if plan.Record.Policy.ValueString() == "weighted" {
		if !plan.Record.Weight.IsNull() && !plan.Record.Weight.IsUnknown() {
			recordReq.SetWeight(plan.Record.Weight.ValueInt64())
		}
		if !plan.Record.Description.IsNull() && !plan.Record.Description.IsUnknown() {
			recordReq.SetDescription(plan.Record.Description.ValueString())
		}
	}

	// Execute update request.
	updateRecord, httpResponse, err := r.client.api.DNSRecordsAPI.UpdateDnsRecord(ctx, recordId, zoneId).
		RecordRequest(*recordReq).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			updateRecord, httpResponse, err = utils.RetryOn429(func() (*azionapi.RecordResponse, *http.Response, error) {
				return r.client.api.DNSRecordsAPI.UpdateDnsRecord(ctx, recordId, zoneId).
					RecordRequest(*recordReq).Execute()
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errorPrintRecord(httpResponse.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}

	// Update plan with response.
	plan.ZoneId = types.StringValue(plan.ZoneId.ValueString())
	plan.Record = populateRecordModel(updateRecord.GetData(), plan.Record.Policy.ValueString())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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

	zoneId, err := strconv.ParseInt(state.ZoneId.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert Zone ID",
		)
		return
	}

	recordId := state.Record.Id.ValueInt64()

	// Execute delete request.
	_, httpResponse, err := r.client.api.DNSRecordsAPI.DeleteDnsRecord(ctx, recordId, zoneId).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			_, httpResponse, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.DNSRecordsAPI.DeleteDnsRecord(ctx, recordId, zoneId).Execute()
			}, 5) // Maximum 5 retries

			if httpResponse != nil {
				defer httpResponse.Body.Close()
			}

			if err != nil {
				resp.Diagnostics.AddError(
					err.Error(),
					"API request failed after too many retries",
				)
				return
			}
		} else {
			usrMsg, errMsg := errorPrintRecord(httpResponse.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	if httpResponse != nil {
		defer httpResponse.Body.Close()
	}
}

func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}

// buildRdataList converts a slice of types.String to a slice of string.
func buildRdataList(rdata []types.String) []string {
	result := make([]string, len(rdata))
	for i, d := range rdata {
		result[i] = d.ValueString()
	}
	return result
}

// populateRecordModel populates a recordModel from an SDK Record.
func populateRecordModel(record azionapi.Record, policyOverride string) *recordModel {
	model := &recordModel{
		Id:   types.Int64Value(record.GetId()),
		Name: types.StringValue(record.GetName()),
		Type: types.StringValue(record.GetType()),
	}

	// Set TTL.
	if record.HasTtl() {
		model.Ttl = types.Int64Value(record.GetTtl())
	}

	// Set policy.
	if policyOverride != "" {
		model.Policy = types.StringValue(policyOverride)
	} else if record.HasPolicy() {
		model.Policy = types.StringValue(record.GetPolicy())
	}

	// Set weight and description for weighted policy.
	if model.Policy.ValueString() == "weighted" {
		if record.HasWeight() {
			model.Weight = types.Int64Value(record.GetWeight())
		} else {
			model.Weight = types.Int64Null()
		}
		if record.HasDescription() {
			model.Description = types.StringValue(record.GetDescription())
		} else {
			model.Description = types.StringNull()
		}
	} else {
		model.Weight = types.Int64Null()
		model.Description = types.StringNull()
	}

	// Set rdata.
	rdata := record.GetRdata()
	model.Rdata = make([]types.String, len(rdata))
	for i, d := range rdata {
		model.Rdata[i] = types.StringValue(d)
	}

	return model
}

// errorPrintRecord returns user-friendly error messages for record operations.
func errorPrintRecord(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "Record Not Found"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
