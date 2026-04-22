// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

const (
	envAccessToken    = "ZOHOMAIL_ACCESS_TOKEN"
	envDataCenter     = "ZOHOMAIL_DATA_CENTER"
	envOrganizationID = "ZOHOMAIL_ORGANIZATION_ID"
)

var _ provider.Provider = &zohoMailProvider{}

type zohoMailProvider struct {
	version string
}

type providerModel struct {
	AccessToken    types.String `tfsdk:"access_token"`
	DataCenter     types.String `tfsdk:"data_center"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &zohoMailProvider{
			version: version,
		}
	}
}

func (p *zohoMailProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zohomail"
	resp.Version = p.version
}

func (p *zohoMailProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		MarkdownDescription: "Terraform provider for managing Zoho Mail administration objects.",
		Attributes: map[string]providerschema.Attribute{
			"organization_id": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Zoho Mail organization identifier. Falls back to `ZOHOMAIL_ORGANIZATION_ID`.",
			},
			"access_token": providerschema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Zoho Mail OAuth access token with the admin scopes required by the managed resources. Falls back to `ZOHOMAIL_ACCESS_TOKEN`.",
			},
			"data_center": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("Zoho Mail data center. Supported values: `%s`. Falls back to `ZOHOMAIL_DATA_CENTER`.", strings.Join(zohomail.SupportedDataCenters(), "`, `")),
			},
		},
	}
}

func (p *zohoMailProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessToken := stringValueFromConfig(config.AccessToken, envAccessToken)
	dataCenter := strings.ToLower(stringValueFromConfig(config.DataCenter, envDataCenter))
	organizationID := stringValueFromConfig(config.OrganizationID, envOrganizationID)

	if accessToken == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("access_token"),
			"Missing Zoho Mail access token",
			fmt.Sprintf("Set `access_token` in the provider configuration or define `%s`.", envAccessToken),
		)
	}

	if dataCenter == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("data_center"),
			"Missing Zoho Mail data center",
			fmt.Sprintf("Set `data_center` in the provider configuration or define `%s`.", envDataCenter),
		)
	} else if _, err := zohomail.BaseURLForDataCenter(dataCenter); err != nil {
		resp.Diagnostics.AddAttributeError(
			pathRoot("data_center"),
			"Unsupported Zoho Mail data center",
			err.Error(),
		)
	}

	if organizationID == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("organization_id"),
			"Missing Zoho Mail organization ID",
			fmt.Sprintf("Set `organization_id` in the provider configuration or define `%s`.", envOrganizationID),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := zohomail.NewClient(zohomail.Config{
		AccessToken:    accessToken,
		DataCenter:     dataCenter,
		OrganizationID: organizationID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Zoho Mail client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *zohoMailProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewDomainAliasResource,
		NewDomainCatchAllResource,
		NewDomainDKIMResource,
		NewDomainOnboardingResource,
		NewDomainSubdomainStrippingResource,
		NewMailboxResource,
		NewMailboxAliasResource,
		NewMailboxForwardingResource,
	}
}

func (p *zohoMailProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func stringValueFromConfig(value types.String, envKey string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return strings.TrimSpace(value.ValueString())
	}

	return strings.TrimSpace(os.Getenv(envKey))
}

func sortedStrings(values []string) []string {
	cloned := append([]string(nil), values...)
	slices.Sort(cloned)

	return cloned
}

func appendSetDiagnostics(diags diag.Diagnostics, extra diag.Diagnostics) diag.Diagnostics {
	if len(extra) == 0 {
		return diags
	}

	diags.Append(extra...)
	return diags
}
