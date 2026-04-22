// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

func pathRoot(name string) path.Path {
	return path.Root(name)
}

func configuredClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *zohomail.Client {
	if req.ProviderData == nil {
		return nil
	}

	client, ok := req.ProviderData.(*zohomail.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *zohomail.Client, got %T.", req.ProviderData),
		)
		return nil
	}

	return client
}

func setValueFromStrings(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, types.StringType, sortedStrings(values))
}

func stringSliceFromSet(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var values []string
	diags := value.ElementsAs(ctx, &values, false)
	return sortedStrings(values), diags
}

func stringSliceFromList(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	var values []string
	diags := value.ElementsAs(ctx, &values, false)
	return values, diags
}

func joinID(parts ...string) string {
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		trimmed = append(trimmed, strings.TrimSpace(part))
	}

	return strings.Join(trimmed, ":")
}

func splitID(raw string, expectedParts int) ([]string, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != expectedParts {
		return nil, fmt.Errorf("expected ID in the form %s", strings.Repeat("<part>:", expectedParts-1)+"<part>")
	}

	for idx, part := range parts {
		parts[idx] = strings.TrimSpace(part)
		if parts[idx] == "" {
			return nil, fmt.Errorf("ID part %d cannot be empty", idx+1)
		}
	}

	return parts, nil
}

func parseEmailAddress(value string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	if err == nil {
		return strings.ToLower(address.Address), nil
	}

	raw := strings.ToLower(strings.TrimSpace(value))
	if raw == "" || !strings.Contains(raw, "@") {
		return "", fmt.Errorf("invalid email address %q", value)
	}

	return raw, nil
}

func mailboxForwardDomains(mailbox *zohomail.Mailbox) map[string]struct{} {
	result := map[string]struct{}{}
	for _, address := range mailbox.EmailAddresses {
		parts := strings.Split(strings.ToLower(strings.TrimSpace(address)), "@")
		if len(parts) != 2 || parts[1] == "" {
			continue
		}
		result[parts[1]] = struct{}{}
	}

	return result
}

func ensureForwardAddressesWithinMailboxDomains(targets []string, mailbox *zohomail.Mailbox) error {
	allowedDomains := mailboxForwardDomains(mailbox)
	if len(allowedDomains) == 0 {
		return fmt.Errorf("mailbox %s does not expose any managed domains", mailbox.ZUID)
	}

	for _, target := range targets {
		normalized, err := parseEmailAddress(target)
		if err != nil {
			return err
		}

		parts := strings.Split(normalized, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid forwarding target %q", target)
		}

		if _, ok := allowedDomains[parts[1]]; !ok {
			return fmt.Errorf("forwarding target %q is outside the mailbox managed domains", target)
		}
	}

	return nil
}

func normalizeOptionalString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}

	return strings.TrimSpace(value.ValueString())
}
