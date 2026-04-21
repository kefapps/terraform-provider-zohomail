// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"zohomail": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccProviderSmoke(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderSmokeConfig,
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped unless TF_ACC=1")
	}

	for _, key := range []string{envAccessToken, envDataCenter, envOrganizationID} {
		if os.Getenv(key) == "" {
			t.Skip("acceptance tests require " + key)
		}
	}
}

const testAccProviderSmokeConfig = `
provider "zohomail" {}
`
