// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/kefapps/terraform-provider-zohomail/internal/zohomail"
)

const (
	testAccEnvDNSBaseDomain  = "ZOHOMAIL_TEST_DNS_BASE_DOMAIN"
	testAccEnvAdvancedDomain = "ZOHOMAIL_TEST_ENABLE_ADVANCED_DOMAIN_FEATURES"
	testAccEnvMailboxFlow    = "ZOHOMAIL_TEST_ENABLE_MAILBOX_LIFECYCLE"
	testAccEnvDNSProvider    = "ZOHOMAIL_TEST_DNS_PROVIDER"
	testAccEnvDNSResolver    = "ZOHOMAIL_TEST_DNS_RESOLVER"
	testAccEnvMultiMailbox   = "ZOHOMAIL_TEST_ENABLE_MULTI_MAILBOX"
	testAccEnvSlowDNSVerify  = "ZOHOMAIL_TEST_ENABLE_SLOW_DNS_VERIFICATION"
	testAccEnvDNSTimeout     = "ZOHOMAIL_TEST_DNS_TIMEOUT"
	testAccEnvDNSZoneName    = "ZOHOMAIL_TEST_DNS_ZONE_NAME"
	testAccEnvMX10           = "ZOHOMAIL_TEST_DNS_MX_10"
	testAccEnvMX20           = "ZOHOMAIL_TEST_DNS_MX_20"
	testAccEnvMX50           = "ZOHOMAIL_TEST_DNS_MX_50"
	testAccEnvSPFValue       = "ZOHOMAIL_TEST_DNS_SPF_VALUE"

	testAccDNSProviderCloudflare     = "cloudflare"
	testAccDefaultCloudflareResolver = "1.1.1.1:53"
	testAccDefaultDNSVerificationTTL = 60
	testAccDefaultDNSTimeout         = 5 * time.Minute
	testAccDefaultSPFValue           = "v=spf1 include:zohomail.com -all"
	testAccDefaultMX10               = "mx.zoho.com"
	testAccDefaultMX20               = "mx2.zoho.com"
	testAccDefaultMX50               = "mx3.zoho.com"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"zohomail": providerserver.NewProtocol6WithError(New("test")()),
}

var testAccExternalProvidersCloudflare = map[string]resource.ExternalProvider{
	"cloudflare": {
		Source:            "registry.terraform.io/cloudflare/cloudflare",
		VersionConstraint: "~> 5.0",
	},
}

type testAccMXRecord struct {
	Host     string
	Priority int
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped unless TF_ACC=1")
	}

	for _, key := range []string{envAccessToken, envDataCenter, envOrganizationID} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			t.Skip("acceptance tests require " + key)
		}
	}
}

func testAccDomainPreCheck(t *testing.T) {
	t.Helper()

	testAccPreCheck(t)

	if strings.TrimSpace(os.Getenv(testAccEnvDNSBaseDomain)) == "" {
		t.Skip("acceptance tests that create Zoho Mail domains require " + testAccEnvDNSBaseDomain)
	}
}

func testAccDNSPreCheck(t *testing.T) {
	t.Helper()

	testAccDomainPreCheck(t)

	if strings.ToLower(strings.TrimSpace(os.Getenv(testAccEnvDNSProvider))) != testAccDNSProviderCloudflare {
		t.Skip("DNS-backed acceptance tests require " + testAccEnvDNSProvider + "=cloudflare")
	}

	for _, key := range []string{testAccEnvDNSZoneName, "CLOUDFLARE_API_TOKEN"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			t.Skip("DNS-backed acceptance tests require " + key)
		}
	}
}

func testAccSlowDNSVerificationPreCheck(t *testing.T, verificationKind string) {
	t.Helper()

	testAccDNSPreCheck(t)

	if testAccBooleanEnv(testAccEnvSlowDNSVerify) {
		return
	}

	t.Skipf(
		"%s acceptance verification requires %s=1 because Zoho can take hours to surface MX, SPF, or DKIM DNS changes",
		verificationKind,
		testAccEnvSlowDNSVerify,
	)
}

func testAccMultiMailboxPreCheck(t *testing.T, scenario string) {
	t.Helper()

	testAccDNSPreCheck(t)

	if testAccBooleanEnv(testAccEnvMultiMailbox) {
		return
	}

	t.Skipf(
		"%s acceptance tests require %s=1 because they need multiple Zoho Mail mailbox licenses on the test tenant",
		scenario,
		testAccEnvMultiMailbox,
	)
}

func testAccMailboxLifecyclePreCheck(t *testing.T, scenario string) {
	t.Helper()

	testAccDNSPreCheck(t)

	if testAccBooleanEnv(testAccEnvMailboxFlow) {
		return
	}

	t.Skipf(
		"%s acceptance tests require %s=1 because they need at least one spare Zoho Mail mailbox license on the test tenant",
		scenario,
		testAccEnvMailboxFlow,
	)
}

func testAccAdvancedDomainFeaturePreCheck(t *testing.T, feature string) {
	t.Helper()

	testAccDNSPreCheck(t)

	if testAccBooleanEnv(testAccEnvAdvancedDomain) {
		return
	}

	t.Skipf(
		"%s acceptance tests require %s=1 because the feature depends on Zoho Mail plan capabilities that are not guaranteed on every tenant",
		feature,
		testAccEnvAdvancedDomain,
	)
}

func testAccBooleanEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func testAccZohoClient(t *testing.T) *zohomail.Client {
	t.Helper()

	client, err := zohomail.NewClient(zohomail.Config{
		AccessToken:    strings.TrimSpace(os.Getenv(envAccessToken)),
		DataCenter:     strings.TrimSpace(os.Getenv(envDataCenter)),
		OrganizationID: strings.TrimSpace(os.Getenv(envOrganizationID)),
	})
	if err != nil {
		t.Fatalf("create Zoho Mail acceptance client: %v", err)
	}

	return client
}

func testAccRequireMailboxCapacity(t *testing.T, scenario string, domainName string, mailboxCount int) {
	t.Helper()

	client := testAccZohoClient(t)
	ctx := context.Background()
	probes := make([]*zohomail.Mailbox, 0, mailboxCount)

	defer func() {
		for _, mailbox := range probes {
			if err := client.DeleteMailbox(ctx, mailbox.ZUID); err != nil && !zohomail.IsNotFound(err) {
				t.Fatalf("delete probe mailbox %s: %v", mailbox.MailboxAddress, err)
			}
		}
	}()

	for idx := 0; idx < mailboxCount; idx++ {
		mailbox, err := client.CreateMailbox(ctx, zohomail.CreateMailboxInput{
			Country:             testAccMailboxCountry,
			DisplayName:         fmt.Sprintf("%s Probe %d", scenario, idx+1),
			FirstName:           "Probe",
			InitialPassword:     testAccMailboxInitialPassword,
			Language:            testAccMailboxLanguage,
			LastName:            "Mailbox",
			OneTimePassword:     false,
			PrimaryEmailAddress: testAccRandomEmail(fmt.Sprintf("probe%d", idx+1), domainName),
			Role:                "member",
			TimeZone:            testAccMailboxTimeZone,
		})
		if err != nil {
			if zohomail.IsMailboxLicenseLimitReached(err) {
				t.Skipf("%s acceptance tests skipped: tenant has no spare Zoho Mail mailbox licenses", scenario)
			}
			t.Fatalf("probe mailbox capacity for %s: %v", scenario, err)
		}

		probes = append(probes, mailbox)
	}
}

func testAccRequireCatchAllCapability(t *testing.T, domainName string) {
	t.Helper()

	client := testAccZohoClient(t)
	ctx := context.Background()

	mailbox, err := client.CreateMailbox(ctx, zohomail.CreateMailboxInput{
		Country:             testAccMailboxCountry,
		DisplayName:         "Catch-all Probe",
		FirstName:           "Catch",
		InitialPassword:     testAccMailboxInitialPassword,
		Language:            testAccMailboxLanguage,
		LastName:            "All",
		OneTimePassword:     false,
		PrimaryEmailAddress: testAccRandomEmail("catchallprobe", domainName),
		Role:                "member",
		TimeZone:            testAccMailboxTimeZone,
	})
	if err != nil {
		if zohomail.IsMailboxLicenseLimitReached(err) {
			t.Skip("Catch-all acceptance tests skipped: tenant has no spare Zoho Mail mailbox licenses")
		}
		t.Fatalf("create catch-all probe mailbox: %v", err)
	}

	defer func() {
		if err := client.DeleteCatchAll(ctx, domainName); err != nil && !zohomail.IsNotFound(err) {
			t.Fatalf("delete catch-all probe address: %v", err)
		}
		if err := client.DeleteMailbox(ctx, mailbox.ZUID); err != nil && !zohomail.IsNotFound(err) {
			t.Fatalf("delete catch-all probe mailbox %s: %v", mailbox.MailboxAddress, err)
		}
	}()

	if err := client.SetCatchAll(ctx, domainName, mailbox.MailboxAddress); err != nil {
		if zohomail.IsOperationNotPermitted(err) {
			t.Skip("Catch-all acceptance tests skipped: tenant plan does not permit catch-all configuration")
		}
		t.Fatalf("probe catch-all capability: %v", err)
	}
}

func testAccProvidersConfig(includeCloudflare bool) string {
	if includeCloudflare {
		return `
provider "cloudflare" {}
provider "zohomail" {}
`
	}

	return `
provider "zohomail" {}
`
}

func testAccCloudflareZoneDataConfig() string {
	return fmt.Sprintf(`
data "cloudflare_zones" "selected" {
  name = %q
}

locals {
  zone_id = one(data.cloudflare_zones.selected.result).id
}
`, testAccDNSZoneName())
}

func testAccRandomDomain(prefix string) string {
	label := fmt.Sprintf("%s-%s", prefix, strings.ToLower(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)))
	return label + "." + testAccBaseDomain()
}

func testAccRandomEmail(localPart string, domainName string) string {
	label := fmt.Sprintf("%s-%s", localPart, strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)))
	return label + "@" + domainName
}

func testAccBaseDomain() string {
	return strings.TrimSpace(os.Getenv(testAccEnvDNSBaseDomain))
}

func testAccDNSZoneName() string {
	return strings.TrimSpace(os.Getenv(testAccEnvDNSZoneName))
}

func testAccSPFValue() string {
	value := strings.TrimSpace(os.Getenv(testAccEnvSPFValue))
	if value == "" {
		return testAccDefaultSPFValue
	}

	return value
}

func testAccMXRecords() []testAccMXRecord {
	return []testAccMXRecord{
		{Host: envOrDefault(testAccEnvMX10, testAccDefaultMX10), Priority: 10},
		{Host: envOrDefault(testAccEnvMX20, testAccDefaultMX20), Priority: 20},
		{Host: envOrDefault(testAccEnvMX50, testAccDefaultMX50), Priority: 50},
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func testAccDNSTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(testAccEnvDNSTimeout))
	if raw == "" {
		return testAccDefaultDNSTimeout
	}

	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return testAccDefaultDNSTimeout
	}

	return duration
}

func testAccResolver() *net.Resolver {
	target := strings.TrimSpace(os.Getenv(testAccEnvDNSResolver))
	if target == "" {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(testAccEnvDNSProvider)), testAccDNSProviderCloudflare) {
			target = testAccDefaultCloudflareResolver
		} else {
			return net.DefaultResolver
		}
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network string, _ string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 5 * time.Second}
			return dialer.DialContext(ctx, network, target)
		},
	}
}

func testAccWaitForTXTRecord(t *testing.T, fqdn string, want string) {
	t.Helper()

	testAccWaitForDNS(t, "TXT "+fqdn, func(ctx context.Context) (bool, error) {
		values, err := testAccResolver().LookupTXT(ctx, fqdn)
		if err != nil {
			return false, nil
		}

		for _, value := range values {
			if strings.TrimSpace(value) == want {
				return true, nil
			}
		}

		return false, nil
	})
}

func testAccWaitForDomainVerificationTXT(t *testing.T, domainName string) {
	t.Helper()

	domain, err := testAccZohoClient(t).GetDomain(context.Background(), domainName)
	if err != nil {
		t.Fatalf("load domain verification code for %s: %v", domainName, err)
	}
	if domain.CNAMEVerificationCode == "" {
		t.Fatalf("domain %s does not expose a CNAME verification code", domainName)
	}
	if domain.TXTVerificationValue == "" {
		t.Fatalf("domain %s does not expose a TXT verification value", domainName)
	}

	testAccWaitForTXTRecord(t, domainName, domain.TXTVerificationValue)
}

func testAccWaitForDomainSPF(t *testing.T, domainName string) {
	t.Helper()

	testAccWaitForTXTRecord(t, domainName, testAccSPFValue())
}

func testAccWaitForMXRecords(t *testing.T, fqdn string, want []testAccMXRecord) {
	t.Helper()

	testAccWaitForDNS(t, "MX "+fqdn, func(ctx context.Context) (bool, error) {
		values, err := testAccResolver().LookupMX(ctx, fqdn)
		if err != nil {
			return false, nil
		}

		found := map[string]int{}
		for _, value := range values {
			found[strings.TrimSuffix(strings.ToLower(value.Host), ".")] = int(value.Pref)
		}

		for _, expected := range want {
			if found[strings.ToLower(expected.Host)] != expected.Priority {
				return false, nil
			}
		}

		return true, nil
	})
}

func testAccWaitForDomainDKIMTXT(t *testing.T, domainName string, selector string) {
	t.Helper()

	domain, err := testAccZohoClient(t).GetDomain(context.Background(), domainName)
	if err != nil {
		t.Fatalf("load DKIM details for %s: %v", domainName, err)
	}

	for _, detail := range domain.DKIMDetails {
		if detail.Selector != selector {
			continue
		}
		if detail.PublicKey == "" {
			t.Fatalf("DKIM selector %s on %s does not expose a public key", selector, domainName)
		}

		testAccWaitForTXTRecord(t, selector+"._domainkey."+domainName, detail.PublicKey)
		return
	}

	t.Fatalf("DKIM selector %s not found on %s", selector, domainName)
}

func testAccWaitForDNS(t *testing.T, description string, check func(context.Context) (bool, error)) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testAccDNSTimeout())
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		ok, err := check(ctx)
		if err != nil {
			t.Fatalf("wait for %s: %v", description, err)
		}
		if ok {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for %s to propagate", description)
		case <-ticker.C:
		}
	}
}

type captureStringValueCheck struct {
	attributePath   tfjsonpath.Path
	destination     *string
	resourceAddress string
}

func (c captureStringValueCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	if req.State == nil || req.State.Values == nil || req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("state unavailable while capturing %s", c.resourceAddress)
		return
	}

	for _, resource := range req.State.Values.RootModule.Resources {
		if resource.Address != c.resourceAddress {
			continue
		}

		value, err := tfjsonpath.Traverse(resource.AttributeValues, c.attributePath)
		if err != nil {
			resp.Error = err
			return
		}

		text, ok := value.(string)
		if !ok {
			resp.Error = fmt.Errorf("attribute %s.%s is %T, expected string", c.resourceAddress, c.attributePath.String(), value)
			return
		}

		*c.destination = text
		return
	}

	resp.Error = fmt.Errorf("resource %s not found in state", c.resourceAddress)
}

func testAccCaptureStringValue(resourceAddress string, attributePath tfjsonpath.Path, destination *string) statecheck.StateCheck {
	return captureStringValueCheck{
		attributePath:   attributePath,
		destination:     destination,
		resourceAddress: resourceAddress,
	}
}
