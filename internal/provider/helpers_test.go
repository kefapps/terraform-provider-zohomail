// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

func TestSplitID(t *testing.T) {
	t.Parallel()

	parts, err := splitID("mailbox-123:alias@example.com", 2)
	if err != nil {
		t.Fatalf("unexpected splitID error: %v", err)
	}

	if parts[0] != "mailbox-123" || parts[1] != "alias@example.com" {
		t.Fatalf("unexpected split parts: %#v", parts)
	}
}

func TestJoinID(t *testing.T) {
	t.Parallel()

	if got := joinID(" mailbox-123 ", "", " alias@example.com "); got != "mailbox-123:alias@example.com" {
		t.Fatalf("unexpected joined id: %q", got)
	}
}

func TestEnsureForwardAddressesWithinMailboxDomains(t *testing.T) {
	t.Parallel()

	mailbox := &zohomail.Mailbox{
		ZUID:           "123",
		EmailAddresses: []string{"support@example.com", "sales@example.net"},
	}

	if err := ensureForwardAddressesWithinMailboxDomains([]string{"hello@example.com"}, mailbox); err != nil {
		t.Fatalf("expected same-domain forwarding to be allowed: %v", err)
	}

	if err := ensureForwardAddressesWithinMailboxDomains([]string{"other@external.org"}, mailbox); err == nil {
		t.Fatal("expected forwarding to an unrelated domain to be rejected")
	}
}

func TestSetAndSliceConversions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	setValue, diags := setValueFromStrings(ctx, []string{"b@example.com", "a@example.com"})
	if diags.HasError() {
		t.Fatalf("unexpected set conversion diagnostics: %v", diags)
	}

	values, diags := stringSliceFromSet(ctx, setValue)
	if diags.HasError() {
		t.Fatalf("unexpected set slice diagnostics: %v", diags)
	}

	if len(values) != 2 || values[0] != "a@example.com" || values[1] != "b@example.com" {
		t.Fatalf("unexpected set slice values: %#v", values)
	}

	listValue, diags := types.ListValueFrom(ctx, types.StringType, []string{"one", "two"})
	if diags.HasError() {
		t.Fatalf("unexpected list conversion diagnostics: %v", diags)
	}

	listValues, diags := stringSliceFromList(ctx, listValue)
	if diags.HasError() {
		t.Fatalf("unexpected list slice diagnostics: %v", diags)
	}

	if len(listValues) != 2 || listValues[0] != "one" || listValues[1] != "two" {
		t.Fatalf("unexpected list slice values: %#v", listValues)
	}
}

func TestNormalizeOptionalString(t *testing.T) {
	t.Parallel()

	if got := normalizeOptionalString(types.StringValue("  value  ")); got != "value" {
		t.Fatalf("unexpected normalized optional string: %q", got)
	}

	if got := normalizeOptionalString(types.StringNull()); got != "" {
		t.Fatalf("expected null optional string to normalize to empty, got %q", got)
	}
}

func TestConfiguredClientAndAppendDiagnostics(t *testing.T) {
	t.Parallel()

	resp := &resource.ConfigureResponse{}
	if got := configuredClient(resource.ConfigureRequest{}, resp); got != nil {
		t.Fatalf("expected nil client when provider data is absent, got %#v", got)
	}

	if !pathRoot("access_token").Equal(pathRoot("access_token")) {
		t.Fatal("expected pathRoot to be stable")
	}

	badResp := &resource.ConfigureResponse{}
	if got := configuredClient(resource.ConfigureRequest{ProviderData: "bad"}, badResp); got != nil {
		t.Fatalf("expected nil client for invalid provider data, got %#v", got)
	}
	if !badResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for invalid provider data type")
	}

	client := &zohomail.Client{}
	okResp := &resource.ConfigureResponse{}
	if got := configuredClient(resource.ConfigureRequest{ProviderData: client}, okResp); got != client {
		t.Fatalf("expected configuredClient to return the provided client, got %#v", got)
	}

	combined := appendSetDiagnostics(diag.Diagnostics{}, badResp.Diagnostics)
	if len(combined) == 0 {
		t.Fatal("expected appendSetDiagnostics to carry diagnostics forward")
	}
}

func TestMailboxStateFromRemoteUsesRemoteCreateOnlyFields(t *testing.T) {
	t.Parallel()

	state, diags := mailboxStateFromRemote(context.Background(), mailboxResourceModel{
		OneTimePassword: types.BoolNull(),
	}, &zohomail.Mailbox{
		AccountID:      "acc-1",
		Country:        "FR",
		DisplayName:    "Support Team",
		EmailAddresses: []string{"support@example.com", "sales@example.com"},
		FirstName:      "Support",
		Language:       "fr",
		LastName:       "Team",
		MailboxAddress: "support@example.com",
		MailboxStatus:  "active",
		Role:           "member",
		TimeZone:       "Europe/Paris",
		ZUID:           "z-1",
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if got := state.Country.ValueString(); got != "FR" {
		t.Fatalf("unexpected country: %q", got)
	}
	if got := state.FirstName.ValueString(); got != "Support" {
		t.Fatalf("unexpected first_name: %q", got)
	}
	if got := state.Language.ValueString(); got != "fr" {
		t.Fatalf("unexpected language: %q", got)
	}
	if got := state.LastName.ValueString(); got != "Team" {
		t.Fatalf("unexpected last_name: %q", got)
	}
	if got := state.PrimaryEmailAddress.ValueString(); got != "support@example.com" {
		t.Fatalf("unexpected primary_email_address: %q", got)
	}
	if got := state.TimeZone.ValueString(); got != "Europe/Paris" {
		t.Fatalf("unexpected time_zone: %q", got)
	}
	if !state.OneTimePassword.IsNull() {
		t.Fatalf("expected one_time_password to remain unknown after refresh, got %#v", state.OneTimePassword)
	}
}

func TestResolvedCatchAllAddress(t *testing.T) {
	t.Parallel()

	if got := resolvedCatchAllAddress("catchall@example.com", types.StringValue("old@example.com"), false); got != "catchall@example.com" {
		t.Fatalf("expected remote catch-all to win, got %q", got)
	}

	if got := resolvedCatchAllAddress("", types.StringValue("old@example.com"), true); got != "old@example.com" {
		t.Fatalf("expected fallback catch-all to be preserved during create/update refresh, got %q", got)
	}

	if got := resolvedCatchAllAddress("", types.StringValue("old@example.com"), false); got != "" {
		t.Fatalf("expected read refresh to surface missing remote catch-all, got %q", got)
	}
}
