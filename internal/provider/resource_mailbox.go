// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

var (
	_ resource.Resource                = &mailboxResource{}
	_ resource.ResourceWithConfigure   = &mailboxResource{}
	_ resource.ResourceWithImportState = &mailboxResource{}
)

type mailboxResource struct {
	client *zohomail.Client
}

type mailboxResourceModel struct {
	AccountID           types.String `tfsdk:"account_id"`
	Country             types.String `tfsdk:"country"`
	DisplayName         types.String `tfsdk:"display_name"`
	EmailAddresses      types.Set    `tfsdk:"email_addresses"`
	FirstName           types.String `tfsdk:"first_name"`
	ID                  types.String `tfsdk:"id"`
	InitialPassword     types.String `tfsdk:"initial_password"`
	Language            types.String `tfsdk:"language"`
	LastName            types.String `tfsdk:"last_name"`
	MailboxAddress      types.String `tfsdk:"mailbox_address"`
	MailboxStatus       types.String `tfsdk:"mailbox_status"`
	OneTimePassword     types.Bool   `tfsdk:"one_time_password"`
	PrimaryEmailAddress types.String `tfsdk:"primary_email_address"`
	Role                types.String `tfsdk:"role"`
	TimeZone            types.String `tfsdk:"time_zone"`
}

func NewMailboxResource() resource.Resource {
	return &mailboxResource{}
}

func (r *mailboxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox"
}

func (r *mailboxResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *mailboxResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a Zoho Mail mailbox backed by a real Zoho Mail user account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zoho Mail user identifier (`zuid`).",
			},
			"primary_email_address": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Primary mailbox email address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"initial_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "Initial password used only at mailbox creation time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"first_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Mailbox user first name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Mailbox user last name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Mailbox display name shown in Zoho Mail.",
			},
			"role": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail role name applied to the mailbox user.",
			},
			"country": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail country code for the mailbox user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"language": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail language code for the mailbox user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"time_zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail time zone for the mailbox user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"one_time_password": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the initial password must be changed on first login. Evaluated only when the mailbox is created.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"account_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zoho Mail account identifier for the mailbox.",
			},
			"mailbox_address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mailbox address returned by Zoho Mail.",
			},
			"mailbox_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mailbox status returned by Zoho Mail.",
			},
			"email_addresses": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "All email addresses currently attached to the mailbox, including aliases.",
			},
		},
	}
}

func (r *mailboxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailboxResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.CreateMailbox(ctx, zohomail.CreateMailboxInput{
		Country:             plan.Country.ValueString(),
		DisplayName:         plan.DisplayName.ValueString(),
		FirstName:           plan.FirstName.ValueString(),
		InitialPassword:     plan.InitialPassword.ValueString(),
		Language:            plan.Language.ValueString(),
		LastName:            plan.LastName.ValueString(),
		OneTimePassword:     !plan.OneTimePassword.IsNull() && !plan.OneTimePassword.IsUnknown() && plan.OneTimePassword.ValueBool(),
		PrimaryEmailAddress: plan.PrimaryEmailAddress.ValueString(),
		Role:                plan.Role.ValueString(),
		TimeZone:            plan.TimeZone.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Zoho Mail mailbox", err.Error())
		return
	}

	state, diags := mailboxStateFromRemote(ctx, plan, mailbox)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mailboxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailboxResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mailbox, err := r.client.GetMailbox(ctx, state.ID.ValueString())
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Zoho Mail mailbox", err.Error())
		return
	}

	nextState, diags := mailboxStateFromRemote(ctx, state, mailbox)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *mailboxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mailboxResourceModel
	var state mailboxResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current := &zohomail.Mailbox{
		AccountID:      state.AccountID.ValueString(),
		MailboxAddress: state.MailboxAddress.ValueString(),
		ZUID:           state.ID.ValueString(),
	}

	if plan.DisplayName.ValueString() != state.DisplayName.ValueString() {
		if err := r.client.UpdateMailboxDisplayName(ctx, current, plan.DisplayName.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to update mailbox display name", err.Error())
			return
		}
	}

	if plan.Role.ValueString() != state.Role.ValueString() {
		if err := r.client.ChangeMailboxRole(ctx, state.ID.ValueString(), plan.Role.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to update mailbox role", err.Error())
			return
		}
	}

	remote, err := r.client.GetMailbox(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh mailbox after update", err.Error())
		return
	}

	nextState, diags := mailboxStateFromRemote(ctx, plan, remote)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *mailboxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailboxResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteMailbox(ctx, state.ID.ValueString()); err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Zoho Mail mailbox", err.Error())
	}
}

func (r *mailboxResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func mailboxStateFromRemote(ctx context.Context, current mailboxResourceModel, remote *zohomail.Mailbox) (mailboxResourceModel, diag.Diagnostics) {
	emailAddresses, diags := setValueFromStrings(ctx, remote.EmailAddresses)
	if len(remote.EmailAddresses) == 0 {
		emailAddresses = types.SetNull(types.StringType)
	}

	return mailboxResourceModel{
		AccountID:           types.StringValue(remote.AccountID),
		Country:             stringValueFromRemoteOrCurrent(remote.Country, current.Country),
		DisplayName:         types.StringValue(remote.DisplayName),
		EmailAddresses:      emailAddresses,
		FirstName:           stringValueFromRemoteOrCurrent(remote.FirstName, current.FirstName),
		ID:                  types.StringValue(remote.ZUID),
		Language:            stringValueFromRemoteOrCurrent(remote.Language, current.Language),
		LastName:            stringValueFromRemoteOrCurrent(remote.LastName, current.LastName),
		MailboxAddress:      types.StringValue(remote.MailboxAddress),
		MailboxStatus:       types.StringValue(remote.MailboxStatus),
		OneTimePassword:     current.OneTimePassword,
		PrimaryEmailAddress: stringValueFromRemoteOrCurrent(remote.MailboxAddress, current.PrimaryEmailAddress),
		Role:                types.StringValue(remote.Role),
		TimeZone:            stringValueFromRemoteOrCurrent(remote.TimeZone, current.TimeZone),
	}, diags
}

func stringValueFromRemoteOrCurrent(remote string, current types.String) types.String {
	if remote != "" {
		return types.StringValue(remote)
	}

	return current
}
