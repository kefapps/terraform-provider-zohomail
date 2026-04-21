// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

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
