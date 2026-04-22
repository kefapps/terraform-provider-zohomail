// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

var (
	_ resource.Resource                = &mailboxAliasResource{}
	_ resource.ResourceWithConfigure   = &mailboxAliasResource{}
	_ resource.ResourceWithImportState = &mailboxAliasResource{}
)

type mailboxAliasResource struct {
	client *zohomail.Client
}

type mailboxAliasResourceModel struct {
	EmailAlias types.String `tfsdk:"email_alias"`
	ID         types.String `tfsdk:"id"`
	MailboxID  types.String `tfsdk:"mailbox_id"`
}

func NewMailboxAliasResource() resource.Resource {
	return &mailboxAliasResource{}
}

func (r *mailboxAliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox_alias"
}

func (r *mailboxAliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *mailboxAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a Zoho Mail email alias attached to a mailbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier `<mailbox_id>:<email_alias>`.",
			},
			"mailbox_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail mailbox identifier (`zuid`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email_alias": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Email alias to attach to the mailbox.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *mailboxAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailboxAliasResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.AddMailboxAlias(ctx, plan.MailboxID.ValueString(), plan.EmailAlias.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to add mailbox alias", err.Error())
		return
	}

	plan.ID = types.StringValue(joinID(plan.MailboxID.ValueString(), plan.EmailAlias.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mailboxAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailboxAliasResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.GetMailbox(ctx, state.MailboxID.ValueString())
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read mailbox alias", err.Error())
		return
	}

	alias := state.EmailAlias.ValueString()
	if !slices.Contains(mailbox.EmailAddresses, alias) {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(joinID(state.MailboxID.ValueString(), alias))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mailboxAliasResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	// No-op: both schema attributes require replacement, so Terraform never plans an in-place update.
}

func (r *mailboxAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailboxAliasResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteMailboxAlias(ctx, state.MailboxID.ValueString(), state.EmailAlias.ValueString()); err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete mailbox alias", err.Error())
	}
}

func (r *mailboxAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid mailbox alias import ID", fmt.Sprintf("Expected `<mailbox_id>:<email_alias>`, got %q.", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mailbox_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email_alias"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
