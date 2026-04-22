// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

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
		PreCheck: func() { testAccDNSPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainAliasSetupConfig(primaryDomain, aliasDomain),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, primaryDomain)
					testAccWaitForDomainVerificationTXT(t, aliasDomain)
				},
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
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("id"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("mail_hosting_enabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("verification_status"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
			},
			{
				ResourceName:            onboardingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"verification_method", "enable_mail_hosting", "verify_spf", "verify_mx", "make_primary"},
			},
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
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

func TestAccDomainOnboarding_verifyMXSlow(t *testing.T) {
	domainName := testAccRandomDomain("onboard-mx")
	onboardingResourceName := "zohomail_domain_onboarding.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccSlowDNSVerificationPreCheck(t, "MX") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, true, false, true, false, false, ""),
			},
			{
				PreConfig: func() {
					testAccWaitForMXRecords(t, domainName, testAccMXRecords())
				},
				Config: testAccOnboardedDomainConfig(domainName, true, false, true, false, true, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("mx_status"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDomainOnboarding_verifySPFSlow(t *testing.T) {
	domainName := testAccRandomDomain("onboard-spf")
	onboardingResourceName := "zohomail_domain_onboarding.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccSlowDNSVerificationPreCheck(t, "SPF") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, true, true, false, false, ""),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainSPF(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, true, true, true, false, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(onboardingResourceName, tfjsonpath.New("spf_status"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDomainDKIM_basicImport(t *testing.T) {
	domainName := testAccRandomDomain("dkim")
	selector := "tf" + testAccRandomDomain("sel")[:8]
	resourceName := "zohomail_domain_dkim.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccDNSPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainDKIMBlock(domainName, selector, false, false)),
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
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainDKIMBlock(domainName, selector, true, false)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_default"), knownvalue.Bool(true)),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"hash_type", "make_default", "verify_public_key"},
			},
		},
	})
}

func TestAccDomainDKIM_verifyPublicKeySlow(t *testing.T) {
	domainName := testAccRandomDomain("dkim-verify")
	selector := "tf" + testAccRandomDomain("sel")[:8]
	resourceName := "zohomail_domain_dkim.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccSlowDNSVerificationPreCheck(t, "DKIM") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainDKIMBlock(domainName, selector, true, false)),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainDKIMTXT(t, domainName, selector)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainDKIMBlock(domainName, selector, true, true)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_default"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_verified"), knownvalue.Bool(true)),
				},
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
		PreCheck: func() {
			testAccAdvancedDomainFeaturePreCheck(t, "Catch-all")
			testAccMultiMailboxPreCheck(t, "Catch-all")
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, ""),
			},
			{
				PreConfig: func() {
					testAccRequireCatchAllCapability(t, domainName)
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainCatchAllBlock(supportEmail, helloEmail, domainName, supportEmail)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("catch_all_address"), knownvalue.StringExact(supportEmail)),
				},
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainCatchAllBlock(supportEmail, helloEmail, domainName, helloEmail)),
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
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					if err := testAccZohoClient(t).DeleteCatchAll(context.Background(), domainName); err != nil {
						t.Fatalf("delete catch-all remotely: %v", err)
					}
				},
				Config: testAccOnboardedDomainConfig(domainName, false, false, true, false, false, testAccDomainCatchAllBlock(supportEmail, helloEmail, domainName, helloEmail)),
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

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccAdvancedDomainFeaturePreCheck(t, "Subdomain stripping") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ExternalProviders:        testAccExternalProvidersCloudflare,
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
				ImportStateVerify: true,
			},
			{
				Config: testAccOnboardedDomainConfig(domainName, false, false, false, false, false, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroy),
					},
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

%[4]s

resource "zohomail_domain" "primary" {
  domain_name = %[2]q
}

resource "cloudflare_dns_record" "primary_verification" {
  zone_id = local.zone_id
  name    = %[2]q
  type    = "TXT"
  content = zohomail_domain.primary.txt_verification_value
  ttl     = %[5]d
}

resource "cloudflare_dns_record" "alias_verification" {
  zone_id = local.zone_id
  name    = %[3]q
  type    = "TXT"
  content = zohomail_domain.alias.txt_verification_value
  ttl     = %[5]d
}

%[6]s

resource "zohomail_domain_onboarding" "primary" {
  depends_on          = [%[7]s]
  domain_name         = zohomail_domain.primary.domain_name
  verification_method = "txt"
  enable_mail_hosting = true
  verify_spf          = false
  verify_mx           = false
  make_primary        = false
}

resource "zohomail_domain_onboarding" "alias" {
  depends_on          = [%[8]s]
  domain_name         = zohomail_domain.alias.domain_name
  verification_method = "txt"
  enable_mail_hosting = false
  verify_spf          = false
  verify_mx           = false
  make_primary        = false
}

resource "zohomail_domain" "alias" {
  domain_name = %[3]q
}

resource "zohomail_domain_alias" "test" {
  depends_on     = [zohomail_domain_onboarding.primary, zohomail_domain_onboarding.alias]
  primary_domain = zohomail_domain.primary.domain_name
  alias_domain   = zohomail_domain.alias.domain_name
}
`, testAccProvidersConfig(true), primaryDomain, aliasDomain, testAccCloudflareZoneDataConfig(), testAccDefaultDNSVerificationTTL, testAccDualDomainMXBlocks(primaryDomain, aliasDomain), testAccDualDomainDependsOn("primary_verification", "primary_mx"), testAccDualDomainDependsOn("alias_verification", "alias_mx"))
}

func testAccDomainAliasSetupConfig(primaryDomain string, aliasDomain string) string {
	return fmt.Sprintf(`
%[1]s

%[4]s

resource "zohomail_domain" "primary" {
  domain_name = %[2]q
}

resource "cloudflare_dns_record" "primary_verification" {
  zone_id = local.zone_id
  name    = %[2]q
  type    = "TXT"
  content = zohomail_domain.primary.txt_verification_value
  ttl     = %[5]d
}

resource "zohomail_domain" "alias" {
  domain_name = %[3]q
}

resource "cloudflare_dns_record" "alias_verification" {
  zone_id = local.zone_id
  name    = %[3]q
  type    = "TXT"
  content = zohomail_domain.alias.txt_verification_value
  ttl     = %[5]d
}

%[6]s
`, testAccProvidersConfig(true), primaryDomain, aliasDomain, testAccCloudflareZoneDataConfig(), testAccDefaultDNSVerificationTTL, testAccDualDomainMXBlocks(primaryDomain, aliasDomain))
}

func testAccDualDomainMXBlocks(primaryDomain string, aliasDomain string) string {
	blocks := make([]string, 0, len(testAccMXRecords())*2)
	for idx, record := range testAccMXRecords() {
		blocks = append(blocks, fmt.Sprintf(`
resource "cloudflare_dns_record" "primary_mx_%d" {
  zone_id  = local.zone_id
  name     = %q
  type     = "MX"
  content  = %q
  priority = %d
  ttl      = %d
}
`, idx, primaryDomain, record.Host, record.Priority, testAccDefaultDNSVerificationTTL))
		blocks = append(blocks, fmt.Sprintf(`
resource "cloudflare_dns_record" "alias_mx_%d" {
  zone_id  = local.zone_id
  name     = %q
  type     = "MX"
  content  = %q
  priority = %d
  ttl      = %d
}
`, idx, aliasDomain, record.Host, record.Priority, testAccDefaultDNSVerificationTTL))
	}

	return strings.Join(blocks, "\n")
}

func testAccDualDomainDependsOn(verification string, mxPrefix string) string {
	dependsOn := []string{fmt.Sprintf("cloudflare_dns_record.%s", verification)}
	for idx := range testAccMXRecords() {
		dependsOn = append(dependsOn, fmt.Sprintf("cloudflare_dns_record.%s_%d", mxPrefix, idx))
	}

	return strings.Join(dependsOn, ", ")
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
  content = zohomail_domain.test.txt_verification_value
  ttl     = %d
}
`, domainName, domainName, testAccDefaultDNSVerificationTTL),
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
	dependsOn := []string{"cloudflare_dns_record.verification"}
	if includeSPF {
		dependsOn = append(dependsOn, "cloudflare_dns_record.spf")
	}
	if includeMX {
		for idx := range testAccMXRecords() {
			dependsOn = append(dependsOn, fmt.Sprintf("cloudflare_dns_record.mx_%d", idx))
		}
	}

	return fmt.Sprintf(`
%[1]s

resource "zohomail_domain_onboarding" "test" {
  depends_on          = [%[6]s]
  domain_name         = zohomail_domain.test.domain_name
  verification_method = "txt"
  enable_mail_hosting = %[2]t
  verify_spf          = %[3]t
  verify_mx           = %[4]t
  make_primary        = false
}

%[5]s
`, testAccDomainDNSSetupConfig(domainName, includeMX, includeSPF), enableMailHosting, verifySPF, verifyMX, extra, strings.Join(dependsOn, ", "))
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
