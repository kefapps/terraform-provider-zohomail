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
	_ resource.Resource                = &domainSubdomainStrippingResource{}
	_ resource.ResourceWithConfigure   = &domainSubdomainStrippingResource{}
	_ resource.ResourceWithImportState = &domainSubdomainStrippingResource{}
)

type domainSubdomainStrippingResource struct {
	client *zohomail.Client
}

type domainSubdomainStrippingResourceModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	ID         types.String `tfsdk:"id"`
}

func NewDomainSubdomainStrippingResource() resource.Resource {
	return &domainSubdomainStrippingResource{}
}

func (r *domainSubdomainStrippingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_subdomain_stripping"
}

func (r *domainSubdomainStrippingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *domainSubdomainStrippingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Enable subdomain stripping for a Zoho Mail domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Subdomain stripping resource identifier. Equal to `domain_name`.",
			},
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether subdomain stripping is enabled in Zoho Mail.",
			},
		},
	}
}

func (r *domainSubdomainStrippingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainSubdomainStrippingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.EnableSubdomainStripping(ctx, plan.DomainName.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to enable Zoho Mail subdomain stripping", err.Error())
		return
	}

	nextState, remove, err := r.refreshState(ctx, domainSubdomainStrippingResourceModel{
		DomainName: plan.DomainName,
		Enabled:    types.BoolValue(true),
		ID:         types.StringValue(plan.DomainName.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh Zoho Mail subdomain stripping state", err.Error())
		return
	}
	if remove {
		resp.Diagnostics.AddError(
			"Unable to confirm Zoho Mail subdomain stripping state",
			"Zoho Mail reported the subdomain stripping resource as absent immediately after enablement.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainSubdomainStrippingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainSubdomainStrippingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, remove, err := r.refreshState(ctx, state)
	if zohomail.IsNotFound(err) || remove {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Zoho Mail subdomain stripping state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainSubdomainStrippingResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	// No-op: the resource only models the enabled state and replacement is sufficient for lifecycle changes.
}

func (r *domainSubdomainStrippingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainSubdomainStrippingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DisableSubdomainStripping(ctx, state.DomainName.ValueString()); err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to disable Zoho Mail subdomain stripping", err.Error())
	}
}

func (r *domainSubdomainStrippingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enabled"), true)...)
}

func (r *domainSubdomainStrippingResource) refreshState(ctx context.Context, current domainSubdomainStrippingResourceModel) (domainSubdomainStrippingResourceModel, bool, error) {
	domain, err := r.client.GetDomain(ctx, current.DomainName.ValueString())
	if err != nil {
		return domainSubdomainStrippingResourceModel{}, false, err
	}

	if domain.SubDomainStrippingSet {
		if !domain.SubDomainStripping {
			return domainSubdomainStrippingResourceModel{}, true, nil
		}

		return domainSubdomainStrippingResourceModel{
			DomainName: current.DomainName,
			Enabled:    types.BoolValue(true),
			ID:         types.StringValue(current.DomainName.ValueString()),
		}, false, nil
	}

	enabled := current.Enabled
	if enabled.IsNull() || enabled.IsUnknown() {
		enabled = types.BoolValue(true)
	}

	return domainSubdomainStrippingResourceModel{
		DomainName: current.DomainName,
		Enabled:    enabled,
		ID:         types.StringValue(current.DomainName.ValueString()),
	}, false, nil
}
