// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

func TestProviderRegistrations(t *testing.T) {
	t.Parallel()

	p := &zohoMailProvider{}

	if got := len(p.Resources(context.Background())); got != 9 {
		t.Fatalf("expected 9 resources, got %d", got)
	}

	if got := len(p.DataSources(context.Background())); got != 0 {
		t.Fatalf("expected no data sources, got %d", got)
	}
}

func TestMailboxResourceSchemaOneTimePasswordRequiresReplace(t *testing.T) {
	t.Parallel()

	r := NewMailboxResource()
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["one_time_password"].(resourceschema.BoolAttribute)
	if !ok {
		t.Fatalf("expected one_time_password to be a BoolAttribute, got %T", resp.Schema.Attributes["one_time_password"])
	}

	if len(attr.PlanModifiers) == 0 {
		t.Fatal("expected one_time_password to require replacement")
	}
}

func TestValidatedVerificationMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		want     string
		wantErr  bool
	}{
		{input: "txt", want: "txt"},
		{input: " CNAME ", want: "cname"},
		{input: "html", want: "html"},
		{input: "tx", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got, err := validatedVerificationMethod(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected validation error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error for %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("unexpected normalized verification method: got %q want %q", got, tc.want)
			}
		})
	}
}
