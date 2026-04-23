// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package zohomail

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestBaseURLForDataCenter(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"ae": "https://mail.zoho.ae",
		"au": "https://mail.zoho.com.au",
		"ca": "https://mail.zoho.ca",
		"cn": "https://mail.zoho.com.cn",
		"eu": "https://mail.zoho.eu",
		"in": "https://mail.zoho.in",
		"jp": "https://mail.zoho.jp",
		"sa": "https://mail.zoho.sa",
		"us": "https://mail.zoho.com",
	}

	for dataCenter, expected := range tests {
		got, err := BaseURLForDataCenter(dataCenter)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", dataCenter, err)
		}
		if got != expected {
			t.Fatalf("unexpected base url for %s: got %q want %q", dataCenter, got, expected)
		}
	}
}

func TestBaseURLForDataCenterUnsupported(t *testing.T) {
	t.Parallel()

	if _, err := BaseURLForDataCenter("br"); err == nil {
		t.Fatal("expected unsupported data center to fail")
	}
}

func TestSupportedDataCenters(t *testing.T) {
	t.Parallel()

	got := SupportedDataCenters()
	if len(got) != 9 {
		t.Fatalf("unexpected supported data centers: %#v", got)
	}
	if got[0] != "ae" || got[len(got)-1] != "us" {
		t.Fatalf("expected sorted supported data centers, got %#v", got)
	}
}

func TestAPIErrorMessage(t *testing.T) {
	t.Parallel()

	err := (&APIError{
		Description: "missing",
		Message:     "not found",
		StatusCode:  404,
		ZohoCode:    404,
	}).Error()

	if err == "" {
		t.Fatal("expected api error string to be populated")
	}
}

func TestStringValueNil(t *testing.T) {
	t.Parallel()

	if got := stringValue(nil); got != "" {
		t.Fatalf("expected nil string value to normalize to empty string, got %q", got)
	}
}

func TestConvertDomainNilVerificationCodes(t *testing.T) {
	t.Parallel()

	domain := convertDomain(rawDomain{})
	if domain.CNAMEVerificationCode != "" {
		t.Fatalf("expected empty CNAME verification code, got %q", domain.CNAMEVerificationCode)
	}
	if domain.HTMLVerificationCode != "" {
		t.Fatalf("expected empty HTML verification code, got %q", domain.HTMLVerificationCode)
	}
	if domain.TXTVerificationValue != "" {
		t.Fatalf("expected empty TXT verification value, got %q", domain.TXTVerificationValue)
	}
}

func TestConvertDomainTXTVerificationValue(t *testing.T) {
	t.Parallel()

	domain := convertDomain(rawDomain{CNAMEVerificationCode: "zb12345678"})
	if domain.TXTVerificationValue != "zoho-verification=zb12345678.zmverify.zoho.com" {
		t.Fatalf("unexpected TXT verification value, got %q", domain.TXTVerificationValue)
	}
}

func TestConvertDomainSubDomainStrippingTracksPresence(t *testing.T) {
	t.Parallel()

	unknown := convertDomain(rawDomain{})
	if unknown.SubDomainStrippingSet {
		t.Fatal("expected subdomain stripping to remain unset when Zoho omits the field")
	}

	enabled := convertDomain(rawDomain{SubDomainStripping: true})
	if !enabled.SubDomainStrippingSet || !enabled.SubDomainStripping {
		t.Fatalf("expected subdomain stripping to be set and true, got %#v", enabled)
	}
}

func TestConvertDKIMFallsBackToIsVerified(t *testing.T) {
	t.Parallel()

	dkim := convertDKIM(rawDKIM{
		DKIMID:     "dk-1",
		IsVerified: true,
		PublicKey:  "pub",
		Selector:   "selector",
	})
	if dkim.Status != "true" {
		t.Fatalf("expected DKIM status to use isVerified when dkimStatus is absent, got %q", dkim.Status)
	}
}

func TestVerificationPendingErrorHelpers(t *testing.T) {
	t.Parallel()

	err := &VerificationPendingError{
		Timeout: time.Minute,
		Err:     errors.New("verification pending"),
	}

	if !IsVerificationPending(err) {
		t.Fatal("expected IsVerificationPending to match VerificationPendingError")
	}

	if !isRetryableVerificationError(context.DeadlineExceeded) {
		t.Fatal("expected context deadline exceeded to be retryable for verification requests")
	}
}

func TestAPIErrorClassifiers(t *testing.T) {
	t.Parallel()

	licenseErr := &APIError{
		Description: "Internal Server Error",
		Details:     "Maximum user license limit reached",
		StatusCode:  500,
		ZohoCode:    500,
	}
	if !IsMailboxLicenseLimitReached(licenseErr) {
		t.Fatal("expected mailbox license limit classifier to match")
	}

	operationErr := &APIError{
		Description: "Forbidden",
		Details:     "OPERATION_NOT_PERMITTED",
		StatusCode:  403,
		ZohoCode:    403,
	}
	if !IsOperationNotPermitted(operationErr) {
		t.Fatal("expected operation not permitted classifier to match")
	}
}

func TestConvertMailboxPreservesExactNumericIdentifiers(t *testing.T) {
	t.Parallel()

	var raw mailboxResponse
	if err := json.Unmarshal([]byte(`{"accountId":9223372036854775807,"zuid":2011402488612345678}`), &raw); err != nil {
		t.Fatalf("unexpected mailbox json decode error: %v", err)
	}

	mailbox := convertMailbox(raw)
	if mailbox.AccountID != "9223372036854775807" {
		t.Fatalf("unexpected account id: %q", mailbox.AccountID)
	}
	if mailbox.ZUID != "2011402488612345678" {
		t.Fatalf("unexpected zuid: %q", mailbox.ZUID)
	}
}

func TestConvertMailboxFallsBackToRoleField(t *testing.T) {
	t.Parallel()

	var raw mailboxResponse
	if err := json.Unmarshal([]byte(`{"role":"admin"}`), &raw); err != nil {
		t.Fatalf("unexpected mailbox json decode error: %v", err)
	}

	mailbox := convertMailbox(raw)
	if mailbox.Role != "admin" {
		t.Fatalf("unexpected role fallback: %q", mailbox.Role)
	}
}
