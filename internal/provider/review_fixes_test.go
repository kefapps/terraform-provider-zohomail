// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

func TestDomainDKIMHashTypeRequiresReplaceOnUnknownOrChangedState(t *testing.T) {
	t.Parallel()

	modifier := stringplanmodifier.RequiresReplace()
	req := planmodifier.StringRequest{
		Plan: tfsdk.Plan{
			Raw: tftypes.NewValue(tftypes.String, "planned"),
		},
		PlanValue: types.StringValue("sha256"),
		State: tfsdk.State{
			Raw: tftypes.NewValue(tftypes.String, "current"),
		},
		StateValue: types.StringNull(),
	}

	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	modifier.PlanModifyString(context.Background(), req, resp)
	if !resp.RequiresReplace {
		t.Fatal("expected imported DKIM state without hash_type to require replacement")
	}

	req.StateValue = types.StringValue("sha256")
	resp = &planmodifier.StringResponse{PlanValue: req.PlanValue}
	modifier.PlanModifyString(context.Background(), req, resp)
	if resp.RequiresReplace {
		t.Fatal("expected matching hash_type to avoid replacement")
	}

	req.StateValue = types.StringValue("sha1")
	resp = &planmodifier.StringResponse{PlanValue: req.PlanValue}
	modifier.PlanModifyString(context.Background(), req, resp)
	if !resp.RequiresReplace {
		t.Fatal("expected known hash_type changes to require replacement")
	}
}

func TestDomainDKIMImportStateSupportsOptionalHashType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := &domainDKIMResource{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	resp := &resource.ImportStateResponse{
		State: mustState(t, schemaResp.Schema, domainDKIMResourceModel{
			DomainName:      types.StringNull(),
			HashType:        types.StringNull(),
			ID:              types.StringNull(),
			IsDefault:       types.BoolNull(),
			IsVerified:      types.BoolNull(),
			MakeDefault:     types.BoolNull(),
			PublicKey:       types.StringNull(),
			Selector:        types.StringNull(),
			VerifyPublicKey: types.BoolNull(),
		}),
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "example.com:dk-1:sha256"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected import diagnostics: %v", resp.Diagnostics)
	}

	var domainName types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("domain_name"), &domainName)...)
	var id types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
	var hashType types.String
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("hash_type"), &hashType)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected state diagnostics: %v", resp.Diagnostics)
	}

	if got := domainName.ValueString(); got != "example.com" {
		t.Fatalf("unexpected imported domain_name: %q", got)
	}
	if got := id.ValueString(); got != "example.com:dk-1" {
		t.Fatalf("unexpected imported id: %q", got)
	}
	if got := hashType.ValueString(); got != "sha256" {
		t.Fatalf("unexpected imported hash_type: %q", got)
	}
}

func TestDomainResourceUpdatePreservesReadDiagnostics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := &domainResource{client: testZohoClient(t, func(req *http.Request) (*http.Response, error) {
		t.Helper()

		if req.Method != http.MethodGet {
			t.Fatalf("unexpected method: got %s want %s", req.Method, http.MethodGet)
		}
		if req.URL.Path != "/api/organization/org/domains/example.com" {
			t.Fatalf("unexpected path: got %s", req.URL.Path)
		}

		return &http.Response{
			Status:     "500 Internal Server Error",
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"status":{"code":500,"description":"boom","message":"read failed"},"data":null}`,
			)),
		}, nil
	})}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	current := testDomainResourceModel("example.com")
	req := resource.UpdateRequest{
		Plan:  mustPlan(t, schemaResp.Schema, current),
		State: mustState(t, schemaResp.Schema, current),
	}
	resp := &resource.UpdateResponse{
		State: mustState(t, schemaResp.Schema, current),
	}

	r.Update(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update to preserve read diagnostics")
	}
}

func testDomainResourceModel(domainName string) domainResourceModel {
	return domainResourceModel{
		CNAMEVerificationCode: types.StringNull(),
		CatchAllAddress:       types.StringNull(),
		DomainID:              types.StringNull(),
		DomainName:            types.StringValue(domainName),
		DKIMStatus:            types.StringNull(),
		HTMLVerificationCode:  types.StringNull(),
		ID:                    types.StringValue(domainName),
		IsDomainAlias:         types.BoolNull(),
		IsPrimary:             types.BoolNull(),
		MailHostingEnabled:    types.BoolNull(),
		MXStatus:              types.StringNull(),
		SPFStatus:             types.StringNull(),
		SubDomainStripping:    types.BoolNull(),
		VerificationStatus:    types.StringNull(),
	}
}

func mustPlan(t *testing.T, schema resourceschema.Schema, val any) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), val); diags.HasError() {
		t.Fatalf("unexpected plan diagnostics: %v", diags)
	}

	return plan
}

func mustState(t *testing.T, schema resourceschema.Schema, val any) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), val); diags.HasError() {
		t.Fatalf("unexpected state diagnostics: %v", diags)
	}

	return state
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testZohoClient(t *testing.T, transport roundTripFunc) *zohomail.Client {
	t.Helper()

	client, err := zohomail.NewClient(zohomail.Config{
		AccessToken:    "token",
		DataCenter:     "us",
		HTTPClient:     &http.Client{Transport: transport},
		OrganizationID: "org",
	})
	if err != nil {
		t.Fatalf("unexpected client creation error: %v", err)
	}

	return client
}
