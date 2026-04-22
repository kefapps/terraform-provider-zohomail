// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

var (
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithConfigure   = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

type domainResource struct {
	client *zohomail.Client
}

type domainResourceModel struct {
	CNAMEVerificationCode types.String `tfsdk:"cname_verification_code"`
	CatchAllAddress       types.String `tfsdk:"catch_all_address"`
	DomainID              types.String `tfsdk:"domain_id"`
	DomainName            types.String `tfsdk:"domain_name"`
	DKIMStatus            types.String `tfsdk:"dkim_status"`
	HTMLVerificationCode  types.String `tfsdk:"html_verification_code"`
	ID                    types.String `tfsdk:"id"`
	IsDomainAlias         types.Bool   `tfsdk:"is_domain_alias"`
	IsPrimary             types.Bool   `tfsdk:"is_primary"`
	MailHostingEnabled    types.Bool   `tfsdk:"mail_hosting_enabled"`
	MXStatus              types.String `tfsdk:"mx_status"`
	SPFStatus             types.String `tfsdk:"spf_status"`
	SubDomainStripping    types.Bool   `tfsdk:"subdomain_stripping_enabled"`
	TXTVerificationValue  types.String `tfsdk:"txt_verification_value"`
	VerificationStatus    types.String `tfsdk:"verification_status"`
}

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the Zoho Mail domain object and expose verification and hosting state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Domain resource identifier. Equal to `domain_name`.",
			},
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Domain name managed in Zoho Mail.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zoho Mail domain identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"html_verification_code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "HTML verification code returned by Zoho Mail.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cname_verification_code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "CNAME verification code returned by Zoho Mail.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"txt_verification_value": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "TXT verification value to publish at the domain apex when using Zoho Mail TXT verification.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"verification_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Domain verification status returned by Zoho Mail.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mail_hosting_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether Zoho Mail hosting is enabled for the domain.",
			},
			"mx_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "MX verification status returned by Zoho Mail.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"spf_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SPF verification status returned by Zoho Mail.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dkim_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Aggregated DKIM status inferred from the default DKIM selector.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_primary": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain is the primary Zoho Mail domain.",
			},
			"is_domain_alias": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain is configured as a domain alias in Zoho Mail.",
			},
			"catch_all_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Catch-all address reported by Zoho Mail, when available.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subdomain_stripping_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether subdomain stripping is enabled for the domain.",
			},
		},
	}
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.CreateDomain(ctx, plan.DomainName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Zoho Mail domain", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, domainStateFromRemote(plan, domain))...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, err := r.refreshState(ctx, state)
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Zoho Mail domain", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, nextState)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, err := r.refreshState(ctx, plan)
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Zoho Mail domain", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, nextState)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDomain(ctx, state.DomainName.ValueString())
	if zohomail.IsNotFound(err) {
		return
	}
	if zohomail.IsDisableMailHostingRequired(err) {
		if disableErr := r.client.DisableMailHosting(ctx, state.DomainName.ValueString()); disableErr != nil {
			resp.Diagnostics.AddError("Unable to disable Zoho Mail hosting before domain deletion", disableErr.Error())
			return
		}

		err = r.client.DeleteDomain(ctx, state.DomainName.ValueString())
	}
	if err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Zoho Mail domain", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), req.ID)...)
}

func domainStateFromRemote(current domainResourceModel, remote *zohomail.Domain) domainResourceModel {
	subdomainStripping := types.BoolNull()
	if remote.SubDomainStrippingSet {
		subdomainStripping = types.BoolValue(remote.SubDomainStripping)
	}

	return domainResourceModel{
		CNAMEVerificationCode: types.StringValue(remote.CNAMEVerificationCode),
		CatchAllAddress:       types.StringValue(remote.CatchAllAddress),
		DomainID:              types.StringValue(remote.DomainID),
		DomainName:            types.StringValue(remote.DomainName),
		DKIMStatus:            types.StringValue(defaultDKIMStatus(remote)),
		HTMLVerificationCode:  types.StringValue(remote.HTMLVerificationCode),
		ID:                    types.StringValue(remote.DomainName),
		IsDomainAlias:         types.BoolValue(remote.IsDomainAlias),
		IsPrimary:             types.BoolValue(remote.IsPrimary),
		MailHostingEnabled:    types.BoolValue(remote.MailHostingEnabled),
		MXStatus:              types.StringValue(remote.MXStatus),
		SPFStatus:             types.StringValue(remote.SPFStatus),
		SubDomainStripping:    subdomainStripping,
		TXTVerificationValue:  types.StringValue(remote.TXTVerificationValue),
		VerificationStatus:    types.StringValue(remote.VerificationStatus),
	}
}

func defaultDKIMStatus(domain *zohomail.Domain) string {
	for _, detail := range domain.DKIMDetails {
		if detail.IsDefault {
			return detail.Status
		}
	}

	if len(domain.DKIMDetails) > 0 {
		return domain.DKIMDetails[0].Status
	}

	return ""
}

func (r *domainResource) refreshState(ctx context.Context, current domainResourceModel) (domainResourceModel, error) {
	domain, err := r.client.GetDomain(ctx, current.DomainName.ValueString())
	if err != nil {
		return domainResourceModel{}, err
	}

	return domainStateFromRemote(current, domain), nil
}
