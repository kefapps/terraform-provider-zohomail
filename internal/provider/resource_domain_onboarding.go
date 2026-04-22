// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

var (
	_ resource.Resource                = &domainOnboardingResource{}
	_ resource.ResourceWithConfigure   = &domainOnboardingResource{}
	_ resource.ResourceWithImportState = &domainOnboardingResource{}
)

type domainOnboardingResource struct {
	client *zohomail.Client
}

type domainOnboardingResourceModel struct {
	DomainName         types.String `tfsdk:"domain_name"`
	EnableMailHosting  types.Bool   `tfsdk:"enable_mail_hosting"`
	ID                 types.String `tfsdk:"id"`
	IsPrimary          types.Bool   `tfsdk:"is_primary"`
	MailHostingEnabled types.Bool   `tfsdk:"mail_hosting_enabled"`
	MXStatus           types.String `tfsdk:"mx_status"`
	MakePrimary        types.Bool   `tfsdk:"make_primary"`
	SPFStatus          types.String `tfsdk:"spf_status"`
	VerificationMethod types.String `tfsdk:"verification_method"`
	VerificationStatus types.String `tfsdk:"verification_status"`
	VerifyMX           types.Bool   `tfsdk:"verify_mx"`
	VerifySPF          types.Bool   `tfsdk:"verify_spf"`
}

func NewDomainOnboardingResource() resource.Resource {
	return &domainOnboardingResource{}
}

func (r *domainOnboardingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_onboarding"
}

func (r *domainOnboardingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *domainOnboardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Trigger Zoho Mail domain verification and post-verification onboarding actions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Onboarding identifier. Equal to `domain_name`.",
			},
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail domain name to verify and onboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"verification_method": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Verification method to trigger in Zoho Mail. Accepted values: `txt`, `cname`, `html`.",
			},
			"enable_mail_hosting": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Enable Zoho Mail hosting after domain verification.",
			},
			"verify_spf": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Ask Zoho Mail to verify the SPF DNS record. Zoho can surface SPF propagation asynchronously, so this flag may need a later apply after DNS has settled.",
			},
			"verify_mx": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Ask Zoho Mail to verify MX records. Zoho can surface MX propagation asynchronously, so this flag may need a later apply after DNS has settled.",
			},
			"make_primary": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Mark the domain as the primary Zoho Mail domain after verification.",
			},
			"verification_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Domain verification status reported by Zoho Mail.",
			},
			"mail_hosting_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether Zoho Mail hosting is enabled for the domain.",
			},
			"spf_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SPF verification status reported by Zoho Mail.",
			},
			"mx_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "MX verification status reported by Zoho Mail.",
			},
			"is_primary": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the domain is primary after onboarding.",
			},
		},
	}
}

func (r *domainOnboardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainOnboardingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState := r.applyOnboarding(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainOnboardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainOnboardingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.DomainName.ValueString())
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain onboarding state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, domainOnboardingStateFromRemote(state, domain))...)
}

func (r *domainOnboardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainOnboardingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState := r.applyOnboarding(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainOnboardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *domainOnboardingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), req.ID)...)
}

func (r *domainOnboardingResource) applyOnboarding(ctx context.Context, plan domainOnboardingResourceModel, diags *diag.Diagnostics) domainOnboardingResourceModel {
	method, err := validatedVerificationMethod(plan.VerificationMethod.ValueString())
	if err != nil {
		diags.AddError("Invalid Zoho Mail verification method", err.Error())
		return domainOnboardingResourceModel{}
	}

	domain, err := r.client.GetDomain(ctx, plan.DomainName.ValueString())
	if err != nil {
		diags.AddError("Unable to load Zoho Mail domain before onboarding", err.Error())
		return domainOnboardingResourceModel{}
	}

	if !domainVerificationComplete(domain) {
		if err := r.client.VerifyDomain(ctx, plan.DomainName.ValueString(), method); err != nil {
			diags.AddError("Unable to verify Zoho Mail domain", err.Error())
			return domainOnboardingResourceModel{}
		}
	}

	if valueBool(plan.EnableMailHosting) {
		if err := r.client.EnableMailHosting(ctx, plan.DomainName.ValueString()); err != nil {
			diags.AddError("Unable to enable Zoho Mail hosting", err.Error())
			return domainOnboardingResourceModel{}
		}
	}

	if valueBool(plan.VerifySPF) {
		if err := r.client.VerifySPF(ctx, plan.DomainName.ValueString()); err != nil {
			diags.AddError("Unable to verify Zoho Mail SPF record", err.Error())
			return domainOnboardingResourceModel{}
		}
	}

	if valueBool(plan.VerifyMX) {
		if err := r.client.VerifyMX(ctx, plan.DomainName.ValueString()); err != nil {
			diags.AddError("Unable to verify Zoho Mail MX records", err.Error())
			return domainOnboardingResourceModel{}
		}
	}

	if valueBool(plan.MakePrimary) {
		if err := r.client.SetPrimaryDomain(ctx, plan.DomainName.ValueString()); err != nil {
			diags.AddError("Unable to make Zoho Mail domain primary", err.Error())
			return domainOnboardingResourceModel{}
		}
	}

	domain, err = r.client.GetDomain(ctx, plan.DomainName.ValueString())
	if err != nil {
		diags.AddError("Unable to refresh Zoho Mail domain onboarding", err.Error())
		return domainOnboardingResourceModel{}
	}

	return domainOnboardingStateFromRemote(plan, domain)
}

func domainOnboardingStateFromRemote(current domainOnboardingResourceModel, remote *zohomail.Domain) domainOnboardingResourceModel {
	return domainOnboardingResourceModel{
		DomainName:         current.DomainName,
		EnableMailHosting:  current.EnableMailHosting,
		ID:                 types.StringValue(remote.DomainName),
		IsPrimary:          types.BoolValue(remote.IsPrimary),
		MailHostingEnabled: types.BoolValue(remote.MailHostingEnabled),
		MXStatus:           types.StringValue(remote.MXStatus),
		MakePrimary:        current.MakePrimary,
		SPFStatus:          types.StringValue(remote.SPFStatus),
		VerificationMethod: current.VerificationMethod,
		VerificationStatus: types.StringValue(remote.VerificationStatus),
		VerifyMX:           current.VerifyMX,
		VerifySPF:          current.VerifySPF,
	}
}

func valueBool(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}

func validatedVerificationMethod(method string) (string, error) {
	method = strings.ToLower(strings.TrimSpace(method))

	switch method {
	case "txt", "cname", "html":
		return method, nil
	default:
		return "", fmt.Errorf("unsupported verification_method %q; expected txt, cname, or html", method)
	}
}

func domainVerificationComplete(domain *zohomail.Domain) bool {
	status := strings.ToLower(strings.TrimSpace(domain.VerificationStatus))
	return status == "true" || status == "verified" || status == "success"
}
