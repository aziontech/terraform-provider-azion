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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &zoneResource{}
	_ resource.ResourceWithConfigure   = &zoneResource{}
	_ resource.ResourceWithImportState = &zoneResource{}
)

func NewZoneResource() resource.Resource {
	return &zoneResource{}
}

type zoneResource struct {
	client *apiClient
}

type zoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Zone        *zoneModel   `tfsdk:"zone"`
}

type zoneModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Domain         types.String `tfsdk:"domain"`
	Active         types.Bool   `tfsdk:"active"`
	Nameservers    types.List   `tfsdk:"nameservers"`
	ProductVersion types.String `tfsdk:"product_version"`
}

func (r *zoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intelligent_dns_zone"
}

func (r *zoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric identifier of the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"zone": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "The zone identifier.",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the zone.",
					},
					"domain": schema.StringAttribute{
						Required:    true,
						Description: "Domain name attributed by Azion to this configuration.",
					},
					"active": schema.BoolAttribute{
						Required:    true,
						Description: "Status of the zone.",
					},
					"nameservers": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "List of nameservers for the zone.",
					},
					"product_version": schema.StringAttribute{
						Computed:    true,
						Description: "Product version of the zone.",
					},
				},
			},
		},
	}
}

func (r *zoneResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*apiClient)
}

func (r *zoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneRequest := azionapi.NewZoneRequest(
		plan.Zone.Name.ValueString(),
		plan.Zone.Domain.ValueString(),
		plan.Zone.Active.ValueBool(),
	)

	zoneResponse, response, err := r.client.api.DNSZonesAPI.CreateDnsZone(ctx).
		ZoneRequest(*zoneRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
				return r.client.api.DNSZonesAPI.CreateDnsZone(ctx).
					ZoneRequest(*zoneRequest).Execute()
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
				resp.Diagnostics.AddError(errReadAll.Error(), "err")
				return
			}
			bodyString := string(bodyBytes)
			usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, fmt.Sprintf("%s\nDetails: %s", errMsg, bodyString))
			return
		}
	}

	zoneData := zoneResponse.GetData()

	// Convert nameservers to Terraform List
	var nameserversList types.List
	if zoneData.GetNameservers() != nil {
		nsSlice := make([]string, len(zoneData.GetNameservers()))
		for i, ns := range zoneData.GetNameservers() {
			nsSlice[i] = ns
		}
		nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		nameserversList = types.ListNull(types.StringType)
	}

	plan.ID = types.StringValue(strconv.FormatInt(zoneData.GetId(), 10))
	plan.Zone = &zoneModel{
		ID:             types.Int64Value(zoneData.GetId()),
		Name:           types.StringValue(zoneData.GetName()),
		Domain:         types.StringValue(zoneData.GetDomain()),
		Active:         types.BoolValue(zoneData.GetActive()),
		Nameservers:    nameserversList,
		ProductVersion: types.StringValue(zoneData.GetProductVersion()),
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *zoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneId, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	zoneResponse, response, err := r.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
				return r.client.api.DNSZonesAPI.RetrieveDnsZone(ctx, zoneId).Execute()
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
			usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	zoneData := zoneResponse.GetData()

	// Convert nameservers to Terraform List
	var nameserversList types.List
	if zoneData.GetNameservers() != nil {
		nsSlice := make([]string, len(zoneData.GetNameservers()))
		for i, ns := range zoneData.GetNameservers() {
			nsSlice[i] = ns
		}
		nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		nameserversList = types.ListNull(types.StringType)
	}

	state.Zone = &zoneModel{
		ID:             types.Int64Value(zoneData.GetId()),
		Name:           types.StringValue(zoneData.GetName()),
		Domain:         types.StringValue(zoneData.GetDomain()),
		Active:         types.BoolValue(zoneData.GetActive()),
		Nameservers:    nameserversList,
		ProductVersion: types.StringValue(zoneData.GetProductVersion()),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *zoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan zoneResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneId, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	updateRequest := azionapi.NewUpdateZoneRequest(
		plan.Zone.Name.ValueString(),
		plan.Zone.Active.ValueBool(),
	)

	zoneResponse, response, err := r.client.api.DNSZonesAPI.UpdateDnsZone(ctx, zoneId).
		UpdateZoneRequest(*updateRequest).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			zoneResponse, response, err = utils.RetryOn429(func() (*azionapi.ZoneResponse, *http.Response, error) {
				return r.client.api.DNSZonesAPI.UpdateDnsZone(ctx, zoneId).
					UpdateZoneRequest(*updateRequest).Execute()
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
			usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}

	zoneData := zoneResponse.GetData()

	// Convert nameservers to Terraform List
	var nameserversList types.List
	if zoneData.GetNameservers() != nil {
		nsSlice := make([]string, len(zoneData.GetNameservers()))
		for i, ns := range zoneData.GetNameservers() {
			nsSlice[i] = ns
		}
		nameserversList, diags = types.ListValueFrom(ctx, types.StringType, nsSlice)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		nameserversList = types.ListNull(types.StringType)
	}

	plan.ID = types.StringValue(strconv.FormatInt(zoneData.GetId(), 10))
	plan.Zone = &zoneModel{
		ID:             types.Int64Value(zoneData.GetId()),
		Name:           types.StringValue(zoneData.GetName()),
		Domain:         types.StringValue(zoneData.GetDomain()),
		Active:         types.BoolValue(zoneData.GetActive()),
		Nameservers:    nameserversList,
		ProductVersion: types.StringValue(zoneData.GetProductVersion()),
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *zoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state zoneResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneId, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert ID",
		)
		return
	}

	_, response, err := r.client.api.DNSZonesAPI.DeleteDnsZone(ctx, zoneId).Execute()
	if err != nil {
		if response.StatusCode == 429 {
			_, response, err = utils.RetryOn429(func() (*azionapi.DeleteResponse, *http.Response, error) {
				return r.client.api.DNSZonesAPI.DeleteDnsZone(ctx, zoneId).Execute()
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
			usrMsg, errMsg := errPrintZoneResource(response.StatusCode, err)
			resp.Diagnostics.AddError(usrMsg, errMsg)
			return
		}
	}
}

func (r *zoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func errPrintZoneResource(errCode int, err error) (string, string) {
	var usrMsg string
	switch errCode {
	case 400:
		usrMsg = "Bad Request"
	case 401:
		usrMsg = "Unauthorized Token"
	case 404:
		usrMsg = "Zone not found"
	case 409:
		usrMsg = "Conflict - Zone already exists"
	default:
		usrMsg = err.Error()
	}

	errMsg := fmt.Sprintf("%d - %s", errCode, usrMsg)
	return usrMsg, errMsg
}
