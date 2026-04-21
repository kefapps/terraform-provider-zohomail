// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"slices"

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
	_ resource.Resource                = &mailboxForwardingResource{}
	_ resource.ResourceWithConfigure   = &mailboxForwardingResource{}
	_ resource.ResourceWithImportState = &mailboxForwardingResource{}
)

type mailboxForwardingResource struct {
	client *zohomail.Client
}

type mailboxForwardingResourceModel struct {
	AccountID          types.String `tfsdk:"account_id"`
	DeleteZohoMailCopy types.Bool   `tfsdk:"delete_zoho_mail_copy"`
	ID                 types.String `tfsdk:"id"`
	MailboxID          types.String `tfsdk:"mailbox_id"`
	TargetAddresses    types.Set    `tfsdk:"target_addresses"`
}

func NewMailboxForwardingResource() resource.Resource {
	return &mailboxForwardingResource{}
}

func (r *mailboxForwardingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox_forwarding"
}

func (r *mailboxForwardingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *mailboxForwardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage internal Zoho Mail forwarding rules for a mailbox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Forwarding resource identifier. Equal to `mailbox_id`.",
			},
			"mailbox_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail mailbox identifier (`zuid`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_addresses": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Forwarding target email addresses. V1 constrains them to domains already attached to the mailbox.",
			},
			"delete_zoho_mail_copy": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether Zoho Mail should delete the original mailbox copy after forwarding.",
			},
			"account_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zoho Mail account identifier backing the mailbox.",
			},
		},
	}
}

func (r *mailboxForwardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailboxForwardingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, diags := r.applyForwarding(ctx, mailboxForwardingResourceModel{}, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *mailboxForwardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailboxForwardingResourceModel

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
		resp.Diagnostics.AddError("Unable to read mailbox forwarding", err.Error())
		return
	}

	forwards, err := r.client.GetMailboxForwarding(ctx, mailbox.AccountID)
	if err != nil {
		if errors.Is(err, zohomail.ErrForwardingStateUnavailable) {
			state.AccountID = types.StringValue(mailbox.AccountID)
			resp.Diagnostics.AddWarning(
				"Zoho Mail forwarding state unavailable",
				"Zoho Mail did not return forwarding details for this mailbox account. Keeping the last known Terraform state for `target_addresses` and `delete_zoho_mail_copy`.",
			)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}

		resp.Diagnostics.AddError("Unable to refresh mailbox forwarding", err.Error())
		return
	}

	state.AccountID = types.StringValue(mailbox.AccountID)
	state.ID = types.StringValue(state.MailboxID.ValueString())

	targets := make([]string, 0, len(forwards))
	deleteCopy := false
	for _, forward := range forwards {
		targets = append(targets, forward.Email)
		deleteCopy = forward.DeleteCopy
	}

	setValue, diags := setValueFromStrings(ctx, targets)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.TargetAddresses = setValue
	state.DeleteZohoMailCopy = types.BoolValue(deleteCopy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mailboxForwardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mailboxForwardingResourceModel
	var state mailboxForwardingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, diags := r.applyForwarding(ctx, state, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *mailboxForwardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailboxForwardingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.GetMailbox(ctx, state.MailboxID.ValueString())
	if zohomail.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to load mailbox before deleting forwarding", err.Error())
		return
	}

	targets, diags := stringSliceFromSet(ctx, state.TargetAddresses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, target := range targets {
		if err := r.client.DisableMailboxForward(ctx, mailbox, target); err != nil && !zohomail.IsNotFound(err) {
			resp.Diagnostics.AddError("Unable to disable mailbox forwarding", err.Error())
			return
		}
		if err := r.client.DeleteMailboxForward(ctx, mailbox, target); err != nil && !zohomail.IsNotFound(err) {
			resp.Diagnostics.AddError("Unable to delete mailbox forwarding", err.Error())
			return
		}
	}
}

func (r *mailboxForwardingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mailbox_id"), req.ID)...)
}

func (r *mailboxForwardingResource) applyForwarding(ctx context.Context, state mailboxForwardingResourceModel, plan mailboxForwardingResourceModel) (mailboxForwardingResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	targets, targetDiags := stringSliceFromSet(ctx, plan.TargetAddresses)
	diags.Append(targetDiags...)
	if diags.HasError() {
		return mailboxForwardingResourceModel{}, diags
	}

	if len(targets) == 0 {
		diags.AddError("Missing forwarding targets", "At least one `target_addresses` entry is required.")
		return mailboxForwardingResourceModel{}, diags
	}

	if len(targets) > 3 {
		diags.AddError("Too many forwarding targets", "Zoho Mail supports at most three forwarding targets per mailbox.")
		return mailboxForwardingResourceModel{}, diags
	}

	mailbox, err := r.client.GetMailbox(ctx, plan.MailboxID.ValueString())
	if err != nil {
		diags.AddError("Unable to load mailbox before configuring forwarding", err.Error())
		return mailboxForwardingResourceModel{}, diags
	}

	if err := ensureForwardAddressesWithinMailboxDomains(targets, mailbox); err != nil {
		diags.AddError("Unsupported forwarding target", err.Error())
		return mailboxForwardingResourceModel{}, diags
	}

	currentTargets, currentDiags := stringSliceFromSet(ctx, state.TargetAddresses)
	diags.Append(currentDiags...)
	if diags.HasError() {
		return mailboxForwardingResourceModel{}, diags
	}

	for _, address := range difference(currentTargets, targets) {
		if err := r.client.DisableMailboxForward(ctx, mailbox, address); err != nil && !zohomail.IsNotFound(err) {
			diags.AddError("Unable to disable mailbox forwarding target", err.Error())
			return mailboxForwardingResourceModel{}, diags
		}
		if err := r.client.DeleteMailboxForward(ctx, mailbox, address); err != nil && !zohomail.IsNotFound(err) {
			diags.AddError("Unable to delete mailbox forwarding target", err.Error())
			return mailboxForwardingResourceModel{}, diags
		}
	}

	for _, address := range difference(targets, currentTargets) {
		if err := r.client.AddMailboxForward(ctx, mailbox, address); err != nil {
			diags.AddError("Unable to add mailbox forwarding target", err.Error())
			return mailboxForwardingResourceModel{}, diags
		}
		if err := r.client.EnableMailboxForward(ctx, mailbox, address); err != nil {
			diags.AddError("Unable to enable mailbox forwarding target", err.Error())
			return mailboxForwardingResourceModel{}, diags
		}
	}

	deleteCopy := !plan.DeleteZohoMailCopy.IsNull() && !plan.DeleteZohoMailCopy.IsUnknown() && plan.DeleteZohoMailCopy.ValueBool()
	if err := r.client.SetDeleteZohoMailCopy(ctx, mailbox, deleteCopy); err != nil {
		diags.AddError("Unable to update delete_zoho_mail_copy", err.Error())
		return mailboxForwardingResourceModel{}, diags
	}

	nextState := mailboxForwardingResourceModel{
		AccountID:          types.StringValue(mailbox.AccountID),
		DeleteZohoMailCopy: types.BoolValue(deleteCopy),
		ID:                 types.StringValue(plan.MailboxID.ValueString()),
		MailboxID:          plan.MailboxID,
	}

	remoteForwards, err := r.client.GetMailboxForwarding(ctx, mailbox.AccountID)
	if err == nil {
		remoteTargets := make([]string, 0, len(remoteForwards))
		for _, forward := range remoteForwards {
			remoteTargets = append(remoteTargets, forward.Email)
			nextState.DeleteZohoMailCopy = types.BoolValue(forward.DeleteCopy)
		}

		setValue, setDiags := setValueFromStrings(ctx, remoteTargets)
		diags.Append(setDiags...)
		nextState.TargetAddresses = setValue
		return nextState, diags
	}

	if !errors.Is(err, zohomail.ErrForwardingStateUnavailable) {
		diags.AddError("Unable to refresh mailbox forwarding", err.Error())
		return mailboxForwardingResourceModel{}, diags
	}

	setValue, setDiags := setValueFromStrings(ctx, targets)
	diags.Append(setDiags...)
	nextState.TargetAddresses = setValue
	return nextState, diags
}

func difference(left []string, right []string) []string {
	result := make([]string, 0, len(left))
	for _, item := range left {
		if !slices.Contains(right, item) {
			result = append(result, item)
		}
	}

	return result
}
