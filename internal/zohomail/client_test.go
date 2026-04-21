// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package zohomail

import "testing"

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
