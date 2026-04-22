// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const (
	testAccMailboxCountry         = "in"
	testAccMailboxInitialPassword = "KefjboTfacc!20260422"
	testAccMailboxLanguage        = "En"
	testAccMailboxTimeZone        = "Asia/Kolkata"
)

func TestAccMailbox_basicImportUpdateReplace(t *testing.T) {
	domainName := testAccRandomDomain("mailbox")
	primaryEmail := testAccRandomEmail("support", domainName)
	resourceName := "zohomail_mailbox.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccMailboxLifecyclePreCheck(t, "Mailbox") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders:        testAccExternalProvidersCloudflare,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccMailboxConfig(domainName, primaryEmail, "Support", "member", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("account_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("primary_email_address"), knownvalue.StringExact(primaryEmail)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("display_name"), knownvalue.StringExact("Support")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("role"), knownvalue.StringExact("member")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("email_addresses"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact(primaryEmail),
					})),
				},
			},
			{
				Config: testAccMailboxConfig(domainName, primaryEmail, "Support Team", "admin", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("display_name"), knownvalue.StringExact("Support Team")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("role"), knownvalue.StringExact("admin")),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initial_password", "one_time_password"},
			},
			{
				Config: testAccMailboxConfig(domainName, primaryEmail, "Support Team", "admin", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("one_time_password"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("display_name"), knownvalue.StringExact("Support Team")),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initial_password", "one_time_password"},
			},
		},
	})
}

func TestAccMailboxAlias_basicImportDrift(t *testing.T) {
	domainName := testAccRandomDomain("alias")
	mailboxEmail := testAccRandomEmail("support", domainName)
	aliasEmail := testAccRandomEmail("sales", domainName)
	aliasResourceName := "zohomail_mailbox_alias.test"
	mailboxResourceName := "zohomail_mailbox.test"

	var mailboxID string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccMailboxLifecyclePreCheck(t, "Mailbox alias") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders:        testAccExternalProvidersCloudflare,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccMailboxAliasConfig(domainName, mailboxEmail, aliasEmail),
				ConfigStateChecks: []statecheck.StateCheck{
					testAccCaptureStringValue(mailboxResourceName, tfjsonpath.New("id"), &mailboxID),
					statecheck.ExpectKnownValue(aliasResourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(aliasResourceName, tfjsonpath.New("email_alias"), knownvalue.StringExact(aliasEmail)),
					statecheck.ExpectKnownValue(aliasResourceName, tfjsonpath.New("mailbox_id"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      aliasResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig: func() {
					if mailboxID == "" {
						t.Fatal("mailbox ID was not captured before alias drift step")
					}

					client := testAccZohoClient(t)
					if err := client.DeleteMailboxAlias(context.Background(), mailboxID, aliasEmail); err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
						t.Fatalf("delete mailbox alias remotely: %v", err)
					}
				},
				Config: testAccMailboxAliasConfig(domainName, mailboxEmail, aliasEmail),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aliasResourceName, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(aliasResourceName, tfjsonpath.New("email_alias"), knownvalue.StringExact(aliasEmail)),
				},
			},
		},
	})
}

func TestAccMailboxForwarding_basicImportUpdate(t *testing.T) {
	domainName := testAccRandomDomain("forward")
	sourceEmail := testAccRandomEmail("support", domainName)
	salesEmail := testAccRandomEmail("sales", domainName)
	helloEmail := testAccRandomEmail("hello", domainName)
	resourceName := "zohomail_mailbox_forwarding.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccMultiMailboxPreCheck(t, "Mailbox forwarding") },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders:        testAccExternalProvidersCloudflare,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config: testAccMailboxForwardingConfig(domainName, sourceEmail, salesEmail, helloEmail, []string{salesEmail, helloEmail}, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("account_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("delete_zoho_mail_copy"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("target_addresses"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact(helloEmail),
						knownvalue.StringExact(salesEmail),
					})),
				},
			},
			{
				Config: testAccMailboxForwardingConfig(domainName, sourceEmail, salesEmail, helloEmail, []string{salesEmail}, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("delete_zoho_mail_copy"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("target_addresses"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact(salesEmail),
					})),
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

func TestAccMailboxForwarding_rejectExternalDomains(t *testing.T) {
	domainName := testAccRandomDomain("forwarderr")
	sourceEmail := testAccRandomEmail("support", domainName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccMailboxLifecyclePreCheck(t, "Mailbox forwarding") },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders:        testAccExternalProvidersCloudflare,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDNSSetupConfig(domainName, false, false),
			},
			{
				PreConfig: func() {
					testAccWaitForDomainVerificationTXT(t, domainName)
				},
				Config:      testAccMailboxForwardingConfig(domainName, sourceEmail, "", "", []string{"outside@example.net"}, false),
				ExpectError: regexp.MustCompile(`Unsupported forwarding target`),
			},
		},
	})
}

func TestTestAccMailboxEmailExpression(t *testing.T) {
	t.Parallel()

	got := testAccMailboxEmailExpression("support-abcd@example.com")
	want := `"support-abcd@${zohomail_domain.test.domain_name}"`
	if got != want {
		t.Fatalf("unexpected mailbox email expression: got %s want %s", got, want)
	}
}

func testAccMailboxConfig(domainName string, primaryEmail string, displayName string, role string, oneTimePassword bool) string {
	return testAccMailboxDomainConfig(domainName, fmt.Sprintf(`
resource "zohomail_mailbox" "test" {
  depends_on            = [zohomail_domain_onboarding.test]
  primary_email_address = %[1]s
  initial_password      = %[5]q
  first_name            = "Support"
  last_name             = "Team"
  display_name          = %[2]q
  role                  = %[3]q
  country               = %[6]q
  language              = %[7]q
  time_zone             = %[8]q
  one_time_password     = %[4]t
}
`, testAccMailboxEmailExpression(primaryEmail), displayName, role, oneTimePassword, testAccMailboxInitialPassword, testAccMailboxCountry, testAccMailboxLanguage, testAccMailboxTimeZone))
}

func testAccMailboxAliasConfig(domainName string, mailboxEmail string, aliasEmail string) string {
	return testAccMailboxDomainConfig(domainName, fmt.Sprintf(`
resource "zohomail_mailbox" "test" {
  depends_on            = [zohomail_domain_onboarding.test]
  primary_email_address = %[1]s
  initial_password      = %[3]q
  first_name            = "Support"
  last_name             = "Team"
  display_name          = "Support"
  role                  = "member"
  country               = %[4]q
  language              = %[5]q
  time_zone             = %[6]q
}

resource "zohomail_mailbox_alias" "test" {
  mailbox_id  = zohomail_mailbox.test.id
  email_alias = %[2]q
}
`, testAccMailboxEmailExpression(mailboxEmail), aliasEmail, testAccMailboxInitialPassword, testAccMailboxCountry, testAccMailboxLanguage, testAccMailboxTimeZone))
}

func testAccMailboxForwardingConfig(domainName string, sourceEmail string, salesEmail string, helloEmail string, targets []string, deleteCopy bool) string {
	mailboxes := []string{
		testAccMailboxResourceBlock("support", sourceEmail, "Support"),
	}
	if salesEmail != "" {
		mailboxes = append(mailboxes, testAccMailboxResourceBlock("sales", salesEmail, "Sales"))
	}
	if helloEmail != "" {
		mailboxes = append(mailboxes, testAccMailboxResourceBlock("hello", helloEmail, "Hello"))
	}

	return testAccMailboxDomainConfig(domainName, fmt.Sprintf(`
%[1]s

resource "zohomail_mailbox_forwarding" "test" {
  mailbox_id             = zohomail_mailbox.support.id
  target_addresses       = %[2]s
  delete_zoho_mail_copy  = %[3]t
}
`, strings.Join(mailboxes, "\n"), testAccHCLStringList(targets), deleteCopy))
}

func testAccMailboxResourceBlock(name string, email string, displayName string) string {
	return fmt.Sprintf(`
resource "zohomail_mailbox" %q {
  depends_on            = [zohomail_domain_onboarding.test]
  primary_email_address = %s
  initial_password      = %q
  first_name            = %q
  last_name             = "Team"
  display_name          = %q
  role                  = "member"
  country               = %q
  language              = %q
  time_zone             = %q
}
`, name, testAccMailboxEmailExpression(email), testAccMailboxInitialPassword, displayName, displayName, testAccMailboxCountry, testAccMailboxLanguage, testAccMailboxTimeZone)
}

func testAccMailboxEmailExpression(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return fmt.Sprintf("%q", email)
	}

	return fmt.Sprintf("%q", parts[0]+"@${zohomail_domain.test.domain_name}")
}

func testAccMailboxDomainConfig(domainName string, extra string) string {
	return testAccOnboardedDomainConfig(domainName, false, false, true, false, false, extra)
}

func testAccHCLStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return fmt.Sprintf("[%s]", strings.Join(quoted, ", "))
}
