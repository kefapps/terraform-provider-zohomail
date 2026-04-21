// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccDomain_basicImport(t *testing.T) {
	domainName := testAccRandomDomain("domain")
	resourceName := "zohomail_domain.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccDomainPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainConfig(domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("domain_name"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("cname_verification_code"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("html_verification_code"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateKind:   resource.ImportBlockWithID,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDomainAlias_basicImport(t *testing.T) {
	primaryDomain := testAccRandomDomain("primary")
	aliasDomain := testAccRandomDomain("alias")
	resourceName := "zohomail_domain_alias.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccDomainPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainAliasConfig(primaryDomain, aliasDomain),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.StringExact(primaryDomain+":"+aliasDomain)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("primary_domain"), knownvalue.StringExact(primaryDomain)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("alias_domain"), knownvalue.StringExact(aliasDomain)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateKind:   resource.ImportBlockWithID,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccDomainOnboarding_importAndStateOnlyDelete(t *testing.T) {
	domainName := testAccRandomDomain("onboard")
	onboardingResourceName := "zohomail_domain_onboarding.test"
	domainResourceName := "zohomail_domain.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccDNSPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, true, true),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
					testAccWaitForDomainSPF(t, domainName)
					testAccWaitForMXRecords(t, domainName, testAccMXRecords())
				},
				Config: testAccOnboardedDomainConfig(domainName, true, true, true, true, false, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("id"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("mail_hosting_enabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("verification_status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("spf_status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("mx_status"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            onboardingResourceName,
				ImportState:             true,
				ImportStateKind:         resource.ImportBlockWithID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"verification_method", "enable_mail_hosting", "verify_spf", "verify_mx", "make_primary"},
			},
			{
				Config: testAccDomainDNSSetupConfig(domainName, true, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(onboardingResourceName, plancheck.ResourceActionDestroy),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(domainResourceName, tfjsonpath.New("mail_hosting_enabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(domainResourceName, tfjsonpath.New("verification_status"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDomainDKIM_verifyImport(t *testing.T) {
	domainName := testAccRandomDomain("dkim")
	selector := "tf" + testAccRandomDomain("sel")[:8]
	resourceName := "zohomail_domain_dkim.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccDNSPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, testAccDomainDKIMBlock(domainName, selector, false, false)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("selector"), knownvalue.StringExact(selector)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("public_key"), knownvalue.NotNull()),
				},
			},
			{
				PreConfig: func() {
					testAccWaitForDomainDKIMTXT(t, domainName, selector)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, testAccDomainDKIMBlock(domainName, selector, true, true)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_default"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_verified"), knownvalue.Bool(true)),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateKind:         resource.ImportBlockWithID,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"hash_type", "make_default", "verify_public_key"},
			},
		},
	})
}

func TestAccDomainCatchAll_basicImportDrift(t *testing.T) {
	domainName := testAccRandomDomain("catchall")
	supportEmail := testAccRandomEmail("support", domainName)
	helloEmail := testAccRandomEmail("hello", domainName)
	resourceName := "zohomail_domain_catch_all.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccDNSPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, true, true),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
					testAccWaitForDomainSPF(t, domainName)
					testAccWaitForMXRecords(t, domainName, testAccMXRecords())
				},
				Config: testAccOnboardedDomainConfig(domainName, true, true, true, true, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, true, true, true, true, false, testAccDomainCatchAllBlock(supportEmail, helloEmail, domainName, supportEmail)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("catch_all_address"), knownvalue.StringExact(supportEmail)),
				},
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, true, true, true, true, false, testAccDomainCatchAllBlock(supportEmail, helloEmail, domainName, helloEmail)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("catch_all_address"), knownvalue.StringExact(helloEmail)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateKind:   resource.ImportBlockWithID,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					if err := testAccZohoClient(t).DeleteCatchAll(context.Background(), domainName); err != nil {
						t.Fatalf("delete catch-all remotely: %v", err)
					}
				},
				Config: testAccOnboardedDomainConfig(domainName, true, true, true, true, false, testAccDomainCatchAllBlock(supportEmail, helloEmail, domainName, helloEmail)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("catch_all_address"), knownvalue.StringExact(helloEmail)),
				},
			},
		},
	})
}

func TestAccDomainSubdomainStripping_basicImportDelete(t *testing.T) {
	domainName := testAccRandomDomain("substrip")
	resourceName := "zohomail_domain_subdomain_stripping.test"
	domainResourceName := "zohomail_domain.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccDNSPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, testAccDomainSubdomainStrippingBlock(domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("enabled"), knownvalue.Bool(true)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateKind:   resource.ImportBlockWithID,
				ImportStateVerify: true,
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroy),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(domainResourceName, tfjsonpath.New("subdomain_stripping_enabled"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

func testAccDomainConfig(domainName string) string {
	return fmt.Sprintf(`
%[1]s

resource "zohomail_domain" "test" {
  domain_name = %[2]q
}
`, testAccProvidersConfig(false), domainName)
}

func testAccDomainAliasConfig(primaryDomain string, aliasDomain string) string {
	return fmt.Sprintf(`
%[1]s

resource "zohomail_domain" "primary" {
  domain_name = %[2]q
}

resource "zohomail_domain" "alias" {
  domain_name = %[3]q
}

resource "zohomail_domain_alias" "test" {
  primary_domain = zohomail_domain.primary.domain_name
  alias_domain   = zohomail_domain.alias.domain_name
}
`, testAccProvidersConfig(false), primaryDomain, aliasDomain)
}

func testAccDomainDNSSetupConfig(domainName string, includeMX bool, includeSPF bool) string {
	parts := []string{
		testAccProvidersConfig(true),
		testAccCloudflareZoneDataConfig(),
		fmt.Sprintf(`
resource "zohomail_domain" "test" {
  domain_name = %q
}

resource "cloudflare_dns_record" "verification" {
  zone_id = local.zone_id
  name    = %q
  type    = "TXT"
  content = "zoho-verification=${zohomail_domain.test.cname_verification_code}.%s"
  ttl     = %d
}
`, domainName, domainName, testAccDNSVerificationTarget(), testAccDefaultDNSVerificationTTL),
	}

	if includeSPF {
		parts = append(parts, fmt.Sprintf(`
resource "cloudflare_dns_record" "spf" {
  zone_id = local.zone_id
  name    = %q
  type    = "TXT"
  content = %q
  ttl     = %d
}
`, domainName, testAccSPFValue(), testAccDefaultDNSVerificationTTL))
	}

	if includeMX {
		for idx, record := range testAccMXRecords() {
			parts = append(parts, fmt.Sprintf(`
resource "cloudflare_dns_record" "mx_%d" {
  zone_id  = local.zone_id
  name     = %q
  type     = "MX"
  content  = %q
  priority = %d
  ttl      = %d
}
`, idx, domainName, record.Host, record.Priority, testAccDefaultDNSVerificationTTL))
		}
	}

	return strings.Join(parts, "\n")
}

func testAccOnboardedDomainConfig(domainName string, includeMX bool, includeSPF bool, enableMailHosting bool, verifySPF bool, verifyMX bool, extra string) string {
	return fmt.Sprintf(`
%[1]s

resource "zohomail_domain_onboarding" "test" {
  domain_name         = zohomail_domain.test.domain_name
  verification_method = "txt"
  enable_mail_hosting = %[2]t
  verify_spf          = %[3]t
  verify_mx           = %[4]t
  make_primary        = false
}

%[5]s
`, testAccDomainDNSSetupConfig(domainName, includeMX, includeSPF), enableMailHosting, verifySPF, verifyMX, extra)
}

func testAccDomainDKIMBlock(domainName string, selector string, makeDefault bool, verifyPublicKey bool) string {
	return fmt.Sprintf(`
resource "zohomail_domain_dkim" "test" {
  domain_name       = zohomail_domain.test.domain_name
  selector          = %[1]q
  hash_type         = "sha256"
  make_default      = %[2]t
  verify_public_key = %[3]t
}

resource "cloudflare_dns_record" "dkim" {
  zone_id = local.zone_id
  name    = "%[1]s._domainkey.%[4]s"
  type    = "TXT"
  content = zohomail_domain_dkim.test.public_key
  ttl     = %[5]d
}
`, selector, makeDefault, verifyPublicKey, domainName, testAccDefaultDNSVerificationTTL)
}

func testAccDomainCatchAllBlock(supportEmail string, helloEmail string, domainName string, catchAllAddress string) string {
	return fmt.Sprintf(`
%[1]s

%[2]s

resource "zohomail_domain_catch_all" "test" {
  domain_name       = %q
  catch_all_address = %q
}
`, testAccMailboxResourceBlock("support", supportEmail, "Support"), testAccMailboxResourceBlock("hello", helloEmail, "Hello"), domainName, catchAllAddress)
}

func testAccDomainSubdomainStrippingBlock(domainName string) string {
	return fmt.Sprintf(`
resource "zohomail_domain_subdomain_stripping" "test" {
  domain_name = %q
}
`, domainName)
}
