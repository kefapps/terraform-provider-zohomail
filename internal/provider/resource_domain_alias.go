// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

var (
	_ resource.Resource                = &domainAliasResource{}
	_ resource.ResourceWithConfigure   = &domainAliasResource{}
	_ resource.ResourceWithImportState = &domainAliasResource{}
)

type domainAliasResource struct {
	client *zohomail.Client
}

type domainAliasResourceModel struct {
	AliasDomain   types.String `tfsdk:"alias_domain"`
	ID            types.String `tfsdk:"id"`
	PrimaryDomain types.String `tfsdk:"primary_domain"`
}

func NewDomainAliasResource() resource.Resource {
	return &domainAliasResource{}
}

func (r *domainAliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_alias"
}

func (r *domainAliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *domainAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a Zoho Mail alias domain relationship.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier `<primary_domain>:<alias_domain>`.",
			},
			"primary_domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Primary Zoho Mail domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias_domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Alias domain name to attach to the primary domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *domainAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainAliasResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.AddDomainAlias(ctx, plan.PrimaryDomain.ValueString(), plan.AliasDomain.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to create Zoho Mail domain alias", err.Error())
		return
	}

	plan.ID = types.StringValue(joinID(plan.PrimaryDomain.ValueString(), plan.AliasDomain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainAliasResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.AliasDomain.ValueString())
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Zoho Mail domain alias", err.Error())
		return
	}

	if !domain.IsDomainAlias {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(joinID(state.PrimaryDomain.ValueString(), state.AliasDomain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainAliasResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (r *domainAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainAliasResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDomainAlias(ctx, state.PrimaryDomain.ValueString(), state.AliasDomain.ValueString()); err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Zoho Mail domain alias", err.Error())
	}
}

func (r *domainAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain alias import ID", fmt.Sprintf("Expected `<primary_domain>:<alias_domain>`, got %q.", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("primary_domain"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("alias_domain"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
