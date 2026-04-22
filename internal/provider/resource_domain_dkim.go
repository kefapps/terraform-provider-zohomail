// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

var (
	_ resource.Resource                = &domainDKIMResource{}
	_ resource.ResourceWithConfigure   = &domainDKIMResource{}
	_ resource.ResourceWithImportState = &domainDKIMResource{}
)

type domainDKIMResource struct {
	client *zohomail.Client
}

type domainDKIMResourceModel struct {
	DomainName      types.String `tfsdk:"domain_name"`
	HashType        types.String `tfsdk:"hash_type"`
	ID              types.String `tfsdk:"id"`
	IsDefault       types.Bool   `tfsdk:"is_default"`
	IsVerified      types.Bool   `tfsdk:"is_verified"`
	MakeDefault     types.Bool   `tfsdk:"make_default"`
	PublicKey       types.String `tfsdk:"public_key"`
	Selector        types.String `tfsdk:"selector"`
	VerifyPublicKey types.Bool   `tfsdk:"verify_public_key"`
}

func NewDomainDKIMResource() resource.Resource {
	return &domainDKIMResource{}
}

func (r *domainDKIMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_dkim"
}

func (r *domainDKIMResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, resp)
}

func (r *domainDKIMResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a Zoho Mail DKIM selector for a domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier `<domain_name>:<dkim_id>`.",
			},
			"domain_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Zoho Mail domain name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"selector": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "DKIM selector name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hash_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Hash type requested from Zoho Mail for the DKIM selector.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"make_default": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to set the DKIM selector as the domain default after creation.",
			},
			"verify_public_key": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to trigger DKIM verification after DNS is ready. Zoho can surface DKIM propagation asynchronously, so this may need a later apply after DNS has settled.",
			},
			"public_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Generated DKIM public key to publish in DNS.",
			},
			"is_default": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the selector is the default DKIM key.",
			},
			"is_verified": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether Zoho Mail reports the selector as verified.",
			},
		},
	}
}

func (r *domainDKIMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainDKIMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dkim, err := r.client.CreateDKIM(ctx, zohomail.CreateDKIMInput{
		DomainName: plan.DomainName.ValueString(),
		HashType:   plan.HashType.ValueString(),
		Selector:   plan.Selector.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Zoho Mail DKIM selector", err.Error())
		return
	}

	if valueBool(plan.MakeDefault) {
		if err := r.client.SetDefaultDKIM(ctx, plan.DomainName.ValueString(), dkim.DKIMID); err != nil {
			resp.Diagnostics.AddError("Unable to set Zoho Mail DKIM selector as default", err.Error())
			return
		}
	}

	if valueBool(plan.VerifyPublicKey) {
		if err := r.client.VerifyDKIM(ctx, plan.DomainName.ValueString(), dkim.DKIMID); err != nil {
			resp.Diagnostics.AddError("Unable to verify Zoho Mail DKIM selector", err.Error())
			return
		}
	}

	nextState, err := r.refreshState(ctx, plan.DomainName.ValueString(), dkim.DKIMID, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh Zoho Mail DKIM selector", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainDKIMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainDKIMResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nextState, err := r.refreshState(ctx, state.DomainName.ValueString(), stateIDPart(state.ID.ValueString(), 1), state)
	if zohomail.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Zoho Mail DKIM selector", err.Error())
		return
	}

	if nextState.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainDKIMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainDKIMResourceModel
	var state domainDKIMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dkimID := stateIDPart(state.ID.ValueString(), 1)

	if valueBool(plan.MakeDefault) && !valueBool(state.MakeDefault) {
		if err := r.client.SetDefaultDKIM(ctx, plan.DomainName.ValueString(), dkimID); err != nil {
			resp.Diagnostics.AddError("Unable to set Zoho Mail DKIM selector as default", err.Error())
			return
		}
	}

	if valueBool(plan.VerifyPublicKey) && !valueBool(state.VerifyPublicKey) {
		if err := r.client.VerifyDKIM(ctx, plan.DomainName.ValueString(), dkimID); err != nil {
			resp.Diagnostics.AddError("Unable to verify Zoho Mail DKIM selector", err.Error())
			return
		}
	}

	nextState, err := r.refreshState(ctx, plan.DomainName.ValueString(), dkimID, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to refresh Zoho Mail DKIM selector", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &nextState)...)
}

func (r *domainDKIMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainDKIMResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dkimID := stateIDPart(state.ID.ValueString(), 1)
	if err := r.client.DeleteDKIM(ctx, state.DomainName.ValueString(), dkimID); err != nil && !zohomail.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Zoho Mail DKIM selector", err.Error())
	}
}

func (r *domainDKIMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainName, dkimID, hashType, err := parseDomainDKIMImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid DKIM import ID", fmt.Sprintf("Expected `<domain_name>:<dkim_id>` or `<domain_name>:<dkim_id>:<hash_type>`, got %q.", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_name"), domainName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), joinID(domainName, dkimID))...)
	if !hashType.IsNull() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hash_type"), hashType)...)
	}
}

func (r *domainDKIMResource) refreshState(ctx context.Context, domainName string, dkimID string, current domainDKIMResourceModel) (domainDKIMResourceModel, error) {
	domain, err := r.client.GetDomain(ctx, domainName)
	if err != nil {
		return domainDKIMResourceModel{}, err
	}

	for _, detail := range domain.DKIMDetails {
		if detail.DKIMID != dkimID {
			continue
		}

		return domainDKIMResourceModel{
			DomainName:      current.DomainName,
			HashType:        current.HashType,
			ID:              types.StringValue(joinID(domainName, detail.DKIMID)),
			IsDefault:       types.BoolValue(detail.IsDefault),
			IsVerified:      types.BoolValue(detail.Status == "true" || detail.Status == "verified"),
			MakeDefault:     current.MakeDefault,
			PublicKey:       types.StringValue(detail.PublicKey),
			Selector:        types.StringValue(detail.Selector),
			VerifyPublicKey: current.VerifyPublicKey,
		}, nil
	}

	return domainDKIMResourceModel{}, nil
}

func stateIDPart(id string, index int) string {
	parts, err := splitID(id, 2)
	if err != nil {
		return ""
	}

	return parts[index]
}

func parseDomainDKIMImportID(raw string) (string, string, types.String, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 && len(parts) != 3 {
		return "", "", types.StringNull(), fmt.Errorf("expected ID in the form <domain_name>:<dkim_id> or <domain_name>:<dkim_id>:<hash_type>")
	}

	for idx, part := range parts {
		parts[idx] = strings.TrimSpace(part)
		if parts[idx] == "" {
			return "", "", types.StringNull(), fmt.Errorf("ID part %d cannot be empty", idx+1)
		}
	}

	hashType := types.StringNull()
	if len(parts) == 3 {
		hashType = types.StringValue(parts[2])
	}

	return parts[0], parts[1], hashType, nil
}
