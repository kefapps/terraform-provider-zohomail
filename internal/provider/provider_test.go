// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProviderMetadata(t *testing.T) {
	t.Parallel()

	p := New("test")()
	resp := &provider.MetadataResponse{}

	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	if resp.TypeName != "zohomail" {
		t.Fatalf("expected provider type name zohomail, got %q", resp.TypeName)
	}

	if resp.Version != "test" {
		t.Fatalf("expected provider version test, got %q", resp.Version)
	}
}

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	p := New("test")()
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	if resp.Schema.MarkdownDescription == "" {
		t.Fatal("expected provider schema markdown description to be set")
	}

	if got := len(resp.Schema.Attributes); got != 3 {
		t.Fatalf("expected 3 provider attributes, got %d", got)
	}
}

func TestStringValueFromConfig(t *testing.T) {
	const envKey = "ZOHOMAIL_TEST_FALLBACK"

	t.Setenv(envKey, "env-value")

	if got := stringValueFromConfig(types.StringValue("explicit"), envKey); got != "explicit" {
		t.Fatalf("expected explicit value to win, got %q", got)
	}

	if got := stringValueFromConfig(types.StringNull(), envKey); got != "env-value" {
		t.Fatalf("expected env fallback, got %q", got)
	}
}
