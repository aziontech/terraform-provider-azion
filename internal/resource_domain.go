package provider

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aziontech/azionapi-go-sdk/domains"
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
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithConfigure   = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *apiClient
}

type DomainResourceModel struct {
	SchemaVersion types.Int64            `tfsdk:"schema_version"`
	Domain        *DomainResourceResults `tfsdk:"domain"`
	ID            types.String           `tfsdk:"id"`
	LastUpdated   types.String           `tfsdk:"last_updated"`
}

type DomainResourceResults struct {
	ID                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Cnames               types.Set    `tfsdk:"cnames"`
	CnameAccessOnly      types.Bool   `tfsdk:"cname_access_only"`
	IsActive             types.Bool   `tfsdk:"is_active"`
	EdgeApplicationId    types.Int64  `tfsdk:"edge_application_id"`
	DigitalCertificateId types.Int64  `tfsdk:"digital_certificate_id"`
	DomainName           types.String `tfsdk:"domain_name"`
	Environment          types.String `tfsdk:"environment"`
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema_version": schema.Int64Attribute{
				Computed: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"domain": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Computed:    true,
						Description: "Identification of this entry.",
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Name of this entry.",
					},
					"cnames": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of domains to use as URLs for your files.",
					},
					"cname_access_only": schema.BoolAttribute{
						Required:    true,
						Description: "Allow access to your URL only via provided CNAMEs.",
					},
					"is_active": schema.BoolAttribute{
						Required:    true,
						Description: "Make access to your URL only via provided CNAMEs.",
					},
					"edge_application_id": schema.Int64Attribute{
						Required:    true,
						Description: "Edge Application associated ID.",
					},
					"digital_certificate_id": schema.Int64Attribute{
						Optional:    true,
						Description: "Digital Certificate associated ID.",
					},
					"domain_name": schema.StringAttribute{
						Computed:    true,
						Description: "Domain name attributed by Azion to this configuration.",
					},
					"environment": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*apiClient)
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	isActive := plan.Domain.IsActive.ValueBool()
	cnameAccessOnly := plan.Domain.CnameAccessOnly.ValueBool()

	domain := domains.CreateDomainRequest{
		EdgeApplicationId: plan.Domain.EdgeApplicationId.ValueInt64(),
		IsActive:          &isActive,
		CnameAccessOnly:   &cnameAccessOnly,
		Name:              plan.Domain.Name.ValueString(),
	}

	requestCnames := plan.Domain.Cnames.ElementsAs(ctx, &domain.Cnames, false)
	resp.Diagnostics.Append(requestCnames...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Domain.DigitalCertificateId.ValueInt64() > 0 {
		domain.DigitalCertificateId = domains.PtrString(plan.Domain.DigitalCertificateId.String())
	}

	createDomain, response, err := r.client.domainsApi.DomainsAPI.CreateDomain(ctx).CreateDomainRequest(domain).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			createDomain, response, err = utils.RetryOn429(func() (*domains.DomainResponseWithResult, *http.Response, error) {
				return r.client.domainsApi.DomainsAPI.CreateDomain(ctx).CreateDomainRequest(domain).Execute() //nolint
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

	plan.SchemaVersion = types.Int64Value(createDomain.SchemaVersion)
	var slice []types.String = []types.String{}
	for _, Cnames := range createDomain.Results.Cnames {
		slice = append(slice, types.StringValue(Cnames))
	}
	plan.Domain = &DomainResourceResults{
		ID:                types.Int64Value(createDomain.Results.GetId()),
		Name:              types.StringValue(createDomain.Results.GetName()),
		CnameAccessOnly:   types.BoolValue(createDomain.Results.GetCnameAccessOnly()),
		IsActive:          types.BoolValue(createDomain.Results.GetIsActive()),
		EdgeApplicationId: types.Int64Value(createDomain.Results.GetEdgeApplicationId()),
		DomainName:        types.StringValue(createDomain.Results.GetDomainName()),
		Cnames:            utils.SliceStringTypeToSetOrNull(slice),
	}

	if createDomain.Results.Environment != nil {
		plan.Domain.Environment = types.StringValue(*createDomain.Results.Environment)
	}
	if createDomain.Results.DigitalCertificateId != nil {
		plan.Domain.DigitalCertificateId = types.Int64Value(*createDomain.Results.DigitalCertificateId)
	}

	plan.ID = types.StringValue("Create Domain")
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var domainId string
	if state.Domain != nil {
		domainId = strconv.Itoa(int(state.Domain.ID.ValueInt64()))
	} else {
		domainId = state.ID.ValueString()
	}

	getDomain, response, err := r.client.domainsApi.DomainsAPI.
		GetDomain(ctx, domainId).Execute() //nolint
	if err != nil {
		if response.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		if response.StatusCode == 429 {
			getDomain, response, err = utils.RetryOn429(func() (*domains.DomainResponseWithResult, *http.Response, error) {
				return r.client.domainsApi.DomainsAPI.GetDomain(ctx, domainId).Execute() //nolint
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

	var slice []types.String
	for _, Cnames := range getDomain.Results.Cnames {
		slice = append(slice, types.StringValue(Cnames))
	}
	state.Domain = &DomainResourceResults{
		ID:                types.Int64Value(getDomain.Results.GetId()),
		Name:              types.StringValue(getDomain.Results.GetName()),
		CnameAccessOnly:   types.BoolValue(getDomain.Results.GetCnameAccessOnly()),
		IsActive:          types.BoolValue(getDomain.Results.GetIsActive()),
		EdgeApplicationId: types.Int64Value(getDomain.Results.GetEdgeApplicationId()),
		DomainName:        types.StringValue(getDomain.Results.GetDomainName()),
		Cnames:            utils.SliceStringTypeToSet(slice),
	}
	if getDomain.Results.Environment != nil {
		state.Domain.Environment = types.StringValue(*getDomain.Results.Environment)
	}
	if getDomain.Results.DigitalCertificateId != nil {
		state.Domain.DigitalCertificateId = types.Int64Value(*getDomain.Results.DigitalCertificateId)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainResourceModel
	diagsDomain := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsDomain...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainId := strconv.Itoa(int(state.Domain.ID.ValueInt64()))
	updateDomainRequest := domains.UpdateDomainRequest{
		EdgeApplicationId: domains.PtrInt64(plan.Domain.EdgeApplicationId.ValueInt64()),
		IsActive:          domains.PtrBool(plan.Domain.IsActive.ValueBool()),
		CnameAccessOnly:   domains.PtrBool(plan.Domain.CnameAccessOnly.ValueBool()),
		Name:              domains.PtrString(plan.Domain.Name.ValueString()),
	}
	if plan.Domain.DigitalCertificateId.ValueInt64() > 0 {
		updateDomainRequest.DigitalCertificateId = domains.PtrString(plan.Domain.DigitalCertificateId.String())
	}
	requestCnames := plan.Domain.Cnames.ElementsAs(ctx, &updateDomainRequest.Cnames, false)
	resp.Diagnostics.Append(requestCnames...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateDomain, response, err := r.client.domainsApi.DomainsAPI.UpdateDomain(ctx, domainId).UpdateDomainRequest(updateDomainRequest).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			updateDomain, response, err = utils.RetryOn429(func() (*domains.DomainResponseWithResult, *http.Response, error) {
				return r.client.domainsApi.DomainsAPI.UpdateDomain(ctx, domainId).UpdateDomainRequest(updateDomainRequest).Execute() //nolint
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

	plan.SchemaVersion = types.Int64Value(updateDomain.SchemaVersion)
	var slice []types.String = []types.String{}
	for _, Cnames := range updateDomain.Results.Cnames {
		slice = append(slice, types.StringValue(Cnames))
	}
	plan.Domain = &DomainResourceResults{
		ID:                types.Int64Value(updateDomain.Results.GetId()),
		Name:              types.StringValue(updateDomain.Results.GetName()),
		CnameAccessOnly:   types.BoolValue(updateDomain.Results.GetCnameAccessOnly()),
		IsActive:          types.BoolValue(updateDomain.Results.GetIsActive()),
		EdgeApplicationId: types.Int64Value(updateDomain.Results.GetEdgeApplicationId()),
		DomainName:        types.StringValue(updateDomain.Results.GetDomainName()),
		Cnames:            utils.SliceStringTypeToSetOrNull(slice),
	}

	if updateDomain.Results.Environment != nil {
		plan.Domain.Environment = types.StringValue(*updateDomain.Results.Environment)
	}
	if updateDomain.Results.DigitalCertificateId != nil {
		plan.Domain.DigitalCertificateId = types.Int64Value(*updateDomain.Results.DigitalCertificateId)
	}
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	domainId := strconv.Itoa(int(state.Domain.ID.ValueInt64()))
	response, err := r.client.domainsApi.DomainsAPI.DelDomain(ctx, domainId).Execute() //nolint
	if err != nil {
		if response.StatusCode == 429 {
			response, err = utils.RetryOn429Delete(func() (*http.Response, error) {
				return r.client.domainsApi.DomainsAPI.DelDomain(ctx, domainId).Execute() //nolint
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
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
