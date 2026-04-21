// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

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
	_ resource.Resource                = &domainCatchAllResource{}
	_ resource.ResourceWithConfigure   = &domainCatchAllResource{}
	_ resource.ResourceWithImportState = &domainCatchAllResource{}
)

type domainCatchAllResource struct {
	client *zohomail.Client
}

type domainCatchAllResourceModel struct {
	CatchAllAddress types.String `tfsdk:"catch_all_address"`
	DomainName      types.String `tfsdk:"domain_name"`
	ID              types.String `tfsdk:"id"`
}

func NewDomainCatchAllResource() resource.Resource {
	return &domainCatchAllResource{}
}

func (r *domainCatchAllResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_catch_all"
}

func (r *domainCatchAllResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *domainCatchAllResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage the Zoho Mail catch-all address for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Catch-all resource identifier. Equal to `domain_name`.",
			},
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catch_all_address": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Mailbox address that should receive catch-all messages.",
			},
		},
	}
}

func (r *domainCatchAllResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainCatchAllResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetCatchAll(ctx, plan.DomainName.ValueString(), plan.CatchAllAddress.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to set Zoho Mail catch-all address", err.Error())
		return
	}

	nextState, err := r.refreshState(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh Zoho Mail catch-all address", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainCatchAllResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainCatchAllResourceModel

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
		resp.Diagnostics.AddError("Unable to read Zoho Mail catch-all address", err.Error())
		return
	}

	if nextState.CatchAllAddress.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainCatchAllResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainCatchAllResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetCatchAll(ctx, plan.DomainName.ValueString(), plan.CatchAllAddress.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to update Zoho Mail catch-all address", err.Error())
		return
	}

	nextState, err := r.refreshState(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh Zoho Mail catch-all address", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainCatchAllResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainCatchAllResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteCatchAll(ctx, state.DomainName.ValueString()); err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Zoho Mail catch-all address", err.Error())
	}
}

func (r *domainCatchAllResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), req.ID)...)
}

func (r *domainCatchAllResource) refreshState(ctx context.Context, current domainCatchAllResourceModel) (domainCatchAllResourceModel, error) {
	domain, err := r.client.GetDomain(ctx, current.DomainName.ValueString())
	if err != nil {
		return domainCatchAllResourceModel{}, err
	}

	address := domain.CatchAllAddress
	if address == "" {
		address = current.CatchAllAddress.ValueString()
	}

	return domainCatchAllResourceModel{
		CatchAllAddress: types.StringValue(address),
		DomainName:      current.DomainName,
		ID:              types.StringValue(current.DomainName.ValueString()),
	}, nil
}
