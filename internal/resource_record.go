package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	dnsapi "github.com/aziontech/azionapi-v4-go-sdk-dev/dns-api"
	"github.com/aziontech/terraform-provider-azion/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	ID          types.String `tfsdk:"id"`
	ZoneId      types.String `tfsdk:"zone_id"`
	Record      *recordModel `tfsdk:"record"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type recordModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Name        types.String `tfsdk:"name"`
	Ttl         types.Int64  `tfsdk:"ttl"`
	Type        types.String `tfsdk:"type"`
	Rdata       types.List   `tfsdk:"rdata"`
	Policy      types.String `tfsdk:"policy"`
	Weight      types.Int64  `tfsdk:"weight"`
}

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_record"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
					"type": schema.StringAttribute{
						Required:    true,
						Description: "DNS record type (A, AAAA, CNAME, MX, NS, etc).",
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Record name.",
					},
					"rdata": schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "Record data values (answers).",
					},
					"policy": schema.StringAttribute{
						Required:    true,
						Description: "Must be 'simple' or 'weighted'.",
					},
					"weight": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "You can only use this field when policy is 'weighted'.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
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

	policyStr := plan.Record.Policy.ValueString()
	record := dnsapi.RecordRequest{
		Type:   plan.Record.Type.ValueString(),
		Name:   plan.Record.Name.ValueString(),
		Ttl:    plan.Record.Ttl.ValueInt64Pointer(),
		Policy: &policyStr,
	}

	if plan.Record.Policy.ValueString() == "weighted" {
		if !plan.Record.Weight.IsNull() && !plan.Record.Weight.IsUnknown() {
			weight := plan.Record.Weight.ValueInt64()
			record.Weight = &weight
		}
		if !plan.Record.Description.IsNull() && !plan.Record.Description.IsUnknown() {
			desc := plan.Record.Description.ValueString()
			record.Description = &desc
		}
	} else {
		plan.Record.Weight = types.Int64Value(0)
		plan.Record.Description = types.StringValue("")
		weight := int64(50)
		record.Weight = &weight
		desc := ""
		record.Description = &desc
	}

	var rdataValues []string
	if !plan.Record.Rdata.IsNull() && !plan.Record.Rdata.IsUnknown() {
		diags = plan.Record.Rdata.ElementsAs(ctx, &rdataValues, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	record.Rdata = rdataValues

	zoneId := plan.ZoneId.ValueString()

	createRecord, httpResponse, err := r.client.idnsApi.DNSRecordsAPI.CreateDnsRecord(ctx, zoneId).RecordRequest(record).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			createRecord, httpResponse, err = utils.RetryOn429(func() (*dnsapi.ResponseRecord, *http.Response, error) {
				return r.client.idnsApi.DNSRecordsAPI.CreateDnsRecord(ctx, zoneId).RecordRequest(record).Execute() //nolint
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

	var slice []types.String
	for _, answerList := range createRecord.Data.Rdata {
		slice = append(slice, types.StringValue(answerList))
	}
	originalTtl := plan.Record.Ttl

	plan.ZoneId = types.StringValue(plan.ZoneId.ValueString())
	plan.ID = types.StringValue(strconv.FormatInt(int64(createRecord.Data.GetId()), 10))

	plan.Record = &recordModel{
		Id:     types.Int64Value(int64(createRecord.Data.GetId())),
		Type:   types.StringValue(createRecord.Data.GetType()),
		Policy: types.StringValue(createRecord.Data.GetPolicy()),
		Name:   types.StringValue(createRecord.Data.GetName()),
		Rdata:  utils.SliceStringTypeToList(slice),
	}

	if plan.Record.Policy.ValueString() == "weighted" {
		if createRecord.Data.Weight != nil {
			plan.Record.Weight = types.Int64Value(int64(*createRecord.Data.Weight))
		}
		if createRecord.Data.Description != nil && *createRecord.Data.Description != "" {
			plan.Record.Description = types.StringValue(*createRecord.Data.Description)
		}
	}

	if createRecord.Data.Ttl != nil {
		plan.Record.Ttl = types.Int64Value(createRecord.Data.GetTtl())
	} else {
		plan.Record.Ttl = originalTtl
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
	var state recordResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneId := state.ZoneId.ValueString()
	recordIDStr := state.ID.ValueString()

	recordResp, httpResponse, err := r.client.idnsApi.DNSRecordsAPI.
		RetrieveDnsRecord(ctx, zoneId, recordIDStr).Execute() //nolint
	if err != nil {
		if httpResponse != nil && httpResponse.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResponse != nil && httpResponse.StatusCode == 429 {
			recordResp, httpResponse, err = utils.RetryOn429(func() (*dnsapi.ResponseRetrieveRecord, *http.Response, error) {
				return r.client.idnsApi.DNSRecordsAPI.RetrieveDnsRecord(ctx, zoneId, recordIDStr).Execute() //nolint
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

	var rdataSlice []types.String
	for _, r := range recordResp.Data.Rdata {
		rdataSlice = append(rdataSlice, types.StringValue(r))
	}

	state.ZoneId = types.StringValue(zoneId)
	state.ID = types.StringValue(strconv.FormatInt(int64(recordResp.Data.GetId()), 10))
	state.Record = &recordModel{
		Id:     types.Int64Value(int64(recordResp.Data.GetId())),
		Type:   types.StringValue(recordResp.Data.GetType()),
		Name:   types.StringValue(recordResp.Data.GetName()),
		Rdata:  utils.SliceStringTypeToList(rdataSlice),
		Policy: types.StringValue(recordResp.Data.GetPolicy()),
	}

	if recordResp.Data.Weight != nil {
		state.Record.Weight = types.Int64Value(int64(*recordResp.Data.Weight))
	} else {
		state.Record.Weight = types.Int64Null()
	}

	if recordResp.Data.Description != nil {
		state.Record.Description = types.StringValue(*recordResp.Data.Description)
	} else {
		state.Record.Description = types.StringNull()
	}

	if recordResp.Data.Ttl != nil {
		state.Record.Ttl = types.Int64Value(recordResp.Data.GetTtl())
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

	var zoneId string
	if plan.ZoneId.IsNull() {
		zoneId = state.ZoneId.ValueString()
	} else {
		zoneId = plan.ZoneId.ValueString()
	}

	recordID := state.Record.Id.ValueInt64()
	recordIDStr := strconv.FormatInt(recordID, 10)

	policyStr := plan.Record.Policy.ValueString()
	record := dnsapi.RecordRequest{
		Type:   plan.Record.Type.ValueString(),
		Name:   plan.Record.Name.ValueString(),
		Ttl:    plan.Record.Ttl.ValueInt64Pointer(),
		Policy: &policyStr,
	}

	if plan.Record.Policy.ValueString() == "weighted" {
		if !plan.Record.Weight.IsNull() && !plan.Record.Weight.IsUnknown() {
			weight := plan.Record.Weight.ValueInt64()
			record.Weight = &weight
		}
		if !plan.Record.Description.IsNull() && !plan.Record.Description.IsUnknown() {
			desc := plan.Record.Description.ValueString()
			record.Description = &desc
		}
	}

	var rdataValues []string
	if !plan.Record.Rdata.IsNull() && !plan.Record.Rdata.IsUnknown() {
		diags = plan.Record.Rdata.ElementsAs(ctx, &rdataValues, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	record.Rdata = rdataValues

	updateRecord, httpResponse, err := r.client.idnsApi.DNSRecordsAPI.
		UpdateDnsRecord(ctx, zoneId, recordIDStr).
		RecordRequest(record).Execute() //nolint
	if err != nil {
		if httpResponse.StatusCode == 429 {
			updateRecord, httpResponse, err = utils.RetryOn429(func() (*dnsapi.ResponseRecord, *http.Response, error) {
				return r.client.idnsApi.DNSRecordsAPI.UpdateDnsRecord(ctx, zoneId, recordIDStr).RecordRequest(record).Execute() //nolint
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

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.ID = types.StringValue(strconv.FormatInt(int64(updateRecord.Data.GetId()), 10))
	plan.Record = &recordModel{
		Id:     types.Int64Value(int64(updateRecord.Data.GetId())),
		Type:   types.StringValue(updateRecord.Data.GetType()),
		Policy: types.StringValue(updateRecord.Data.GetPolicy()),
		Name:   types.StringValue(updateRecord.Data.GetName()),
		Rdata: utils.SliceStringTypeToList(func() []types.String {
			var s []types.String
			for _, v := range updateRecord.Data.Rdata {
				s = append(s, types.StringValue(v))
			}
			return s
		}()),
	}

	if updateRecord.Data.Weight != nil {
		plan.Record.Weight = types.Int64Value(int64(*updateRecord.Data.Weight))
	} else {
		plan.Record.Weight = types.Int64Null()
	}

	if updateRecord.Data.Description != nil {
		plan.Record.Description = types.StringValue(*updateRecord.Data.Description)
	} else {
		plan.Record.Description = types.StringNull()
	}

	if updateRecord.Data.Ttl != nil {
		plan.Record.Ttl = types.Int64Value(updateRecord.Data.GetTtl())
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

	zoneId := state.ZoneId.ValueString()
	recordID := state.Record.Id.ValueInt64()
	recordIDStr := strconv.FormatInt(recordID, 10)

	_, err := utils.RetryOn429Delete(func() (*http.Response, error) {
		_, httpResp, err := r.client.idnsApi.DNSRecordsAPI.DeleteDnsRecord(ctx, zoneId, recordIDStr).Execute() //nolint
		return httpResp, err
	}, 5)
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
