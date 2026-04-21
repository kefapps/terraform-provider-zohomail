// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

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
