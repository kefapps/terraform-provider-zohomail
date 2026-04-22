// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: Apache-2.0

package zohomail

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testDomainExample    = "example.com"
	testDomainPath       = "/api/organization/org/domains/example.com"
	testMailboxAccountID = "2002"
	testMailboxPath      = "/api/organization/org/accounts/1001"
	testOrgAccountsPath  = "/api/organization/org/accounts"
	testSupportEmail     = "support@example.com"
	testSalesEmail       = "sales@example.com"
)

type requestCase struct {
	method           string
	name             string
	path             string
	responseBody     string
	run              func(*testing.T, *Client)
	wantBodyContains string
}

func TestClientMailboxAccountRequests(t *testing.T) {
	t.Parallel()

	runRequestCases(t, []requestCase{
		{
			name:             "CreateMailbox",
			method:           http.MethodPost,
			path:             testOrgAccountsPath,
			wantBodyContains: `"primaryEmailAddress":"support@example.com"`,
			responseBody:     `{"status":{"code":200},"data":{"zuid":1001,"accountId":2002,"mailboxAddress":"support@example.com","roleName":"member","emailAddress":[{"mailId":"support@example.com","isPrimary":true}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				got, err := client.CreateMailbox(context.Background(), CreateMailboxInput{
					Country:             "in",
					DisplayName:         "Support",
					FirstName:           "Support",
					InitialPassword:     "secret",
					Language:            "En",
					LastName:            "Team",
					PrimaryEmailAddress: testSupportEmail,
					Role:                "member",
					TimeZone:            "Asia/Kolkata",
				})
				if err != nil {
					t.Fatalf("CreateMailbox returned error: %v", err)
				}
				if got.ZUID != "1001" || got.AccountID != testMailboxAccountID {
					t.Fatalf("unexpected mailbox response: %#v", got)
				}
			},
		},
		{
			name:         "GetMailbox",
			method:       http.MethodGet,
			path:         testMailboxPath,
			responseBody: `{"status":{"code":200},"data":{"zuid":1001,"accountId":2002,"mailboxAddress":"support@example.com","roleName":"member","emailAddress":[{"mailId":"support@example.com","isPrimary":true}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				got, err := client.GetMailbox(context.Background(), "1001")
				if err != nil {
					t.Fatalf("GetMailbox returned error: %v", err)
				}
				if got.MailboxAddress != testSupportEmail {
					t.Fatalf("unexpected mailbox address: %#v", got)
				}
			},
		},
		{
			name:             "UpdateMailboxDisplayName",
			method:           http.MethodPut,
			path:             "/api/organization/org/accounts/2002",
			wantBodyContains: `"mode":"displaynameemailupdate"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				err := client.UpdateMailboxDisplayName(context.Background(), &Mailbox{
					AccountID:      testMailboxAccountID,
					MailboxAddress: testSupportEmail,
					ZUID:           "1001",
				}, "Support Team")
				if err != nil {
					t.Fatalf("UpdateMailboxDisplayName returned error: %v", err)
				}
			},
		},
		{
			name:             "ChangeMailboxRole",
			method:           http.MethodPut,
			path:             testOrgAccountsPath,
			wantBodyContains: `"mode":"changeRole"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				if err := client.ChangeMailboxRole(context.Background(), "1001", "admin"); err != nil {
					t.Fatalf("ChangeMailboxRole returned error: %v", err)
				}
			},
		},
		{
			name:             "DeleteMailbox",
			method:           http.MethodDelete,
			path:             testOrgAccountsPath,
			wantBodyContains: `"accountList":["1001"]`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				if err := client.DeleteMailbox(context.Background(), "1001"); err != nil {
					t.Fatalf("DeleteMailbox returned error: %v", err)
				}
			},
		},
	})
}

func TestClientMailboxAliasRequests(t *testing.T) {
	t.Parallel()

	runRequestCases(t, []requestCase{
		{
			name:             "AddMailboxAlias",
			method:           http.MethodPut,
			path:             testMailboxPath,
			wantBodyContains: `"mode":"addEmailAlias"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.AddMailboxAlias(context.Background(), "1001", testSalesEmail); err != nil {
					t.Fatalf("AddMailboxAlias returned error: %v", err)
				}
			},
		},
		{
			name:             "DeleteMailboxAlias",
			method:           http.MethodPut,
			path:             testMailboxPath,
			wantBodyContains: `"mode":"deleteEmailAlias"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteMailboxAlias(context.Background(), "1001", testSalesEmail); err != nil {
					t.Fatalf("DeleteMailboxAlias returned error: %v", err)
				}
			},
		},
	})
}

func TestClientMailboxForwardingRequests(t *testing.T) {
	t.Parallel()

	runRequestCases(t, []requestCase{
		{
			name:             "AddMailboxForward",
			method:           http.MethodPut,
			path:             "/api/organization/org/accounts/2002",
			wantBodyContains: `"mode":"addMailForward"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.AddMailboxForward(context.Background(), &Mailbox{AccountID: testMailboxAccountID, ZUID: "1001"}, testSalesEmail); err != nil {
					t.Fatalf("AddMailboxForward returned error: %v", err)
				}
			},
		},
		{
			name:             "EnableMailboxForward",
			method:           http.MethodPut,
			path:             "/api/organization/org/accounts/2002",
			wantBodyContains: `"mode":"enableMailForward"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.EnableMailboxForward(context.Background(), &Mailbox{AccountID: testMailboxAccountID, ZUID: "1001"}, testSalesEmail); err != nil {
					t.Fatalf("EnableMailboxForward returned error: %v", err)
				}
			},
		},
		{
			name:             "DisableMailboxForward",
			method:           http.MethodPut,
			path:             "/api/organization/org/accounts/2002",
			wantBodyContains: `"mode":"disableMailForward"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DisableMailboxForward(context.Background(), &Mailbox{AccountID: testMailboxAccountID, ZUID: "1001"}, testSalesEmail); err != nil {
					t.Fatalf("DisableMailboxForward returned error: %v", err)
				}
			},
		},
		{
			name:             "DeleteMailboxForward",
			method:           http.MethodPut,
			path:             "/api/organization/org/accounts/2002",
			wantBodyContains: `"mode":"deleteMailForward"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteMailboxForward(context.Background(), &Mailbox{AccountID: testMailboxAccountID, ZUID: "1001"}, testSalesEmail); err != nil {
					t.Fatalf("DeleteMailboxForward returned error: %v", err)
				}
			},
		},
		{
			name:             "SetDeleteZohoMailCopy",
			method:           http.MethodPut,
			path:             "/api/organization/org/accounts/2002",
			wantBodyContains: `"mode":"deleteZohoMailCopy"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.SetDeleteZohoMailCopy(context.Background(), &Mailbox{AccountID: testMailboxAccountID, ZUID: "1001"}, true); err != nil {
					t.Fatalf("SetDeleteZohoMailCopy returned error: %v", err)
				}
			},
		},
		{
			name:         "GetMailboxForwarding",
			method:       http.MethodGet,
			path:         "/api/accounts/2002",
			responseBody: `{"status":{"code":200},"data":{"mailForward":[{"mailForward":"sales@example.com","deleteCopy":true,"status":"verified"}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				got, err := client.GetMailboxForwarding(context.Background(), "2002")
				if err != nil {
					t.Fatalf("GetMailboxForwarding returned error: %v", err)
				}
				if len(got) != 1 || got[0].Email != testSalesEmail || !got[0].DeleteCopy {
					t.Fatalf("unexpected mailbox forwarding payload: %#v", got)
				}
			},
		},
	})
}

func TestClientDomainRequests(t *testing.T) {
	t.Parallel()

	runRequestCases(t, []requestCase{
		{
			name:             "CreateDomain",
			method:           http.MethodPost,
			path:             "/api/organization/org/domains",
			wantBodyContains: `"domainName":"example.com"`,
			responseBody:     `{"status":{"code":200},"data":{"domainName":"example.com","domainId":"dom-1","verificationStatus":"pending"}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.CreateDomain(context.Background(), testDomainExample)
				if err != nil {
					t.Fatalf("CreateDomain returned error: %v", err)
				}
				if got.DomainID != "dom-1" {
					t.Fatalf("unexpected domain response: %#v", got)
				}
			},
		},
		{
			name:         "GetDomain",
			method:       http.MethodGet,
			path:         testDomainPath,
			responseBody: `{"status":{"code":200},"data":{"domainName":"example.com","domainId":"dom-1","verificationStatus":"verified","dkimDetailList":[{"dkimId":"dk-1","selector":"terraform","publicKey":"pub","dkimStatus":"verified"}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.GetDomain(context.Background(), testDomainExample)
				if err != nil {
					t.Fatalf("GetDomain returned error: %v", err)
				}
				if got.DomainName != testDomainExample || len(got.DKIMDetails) != 1 {
					t.Fatalf("unexpected domain payload: %#v", got)
				}
			},
		},
		{
			name:         "DeleteDomain",
			method:       http.MethodDelete,
			path:         testDomainPath,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteDomain(context.Background(), testDomainExample); err != nil {
					t.Fatalf("DeleteDomain returned error: %v", err)
				}
			},
		},
		{
			name:             "VerifyDomain",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"verifyDomainByTXT"`,
			responseBody:     `{"status":{"code":200},"data":{"status":true}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.VerifyDomain(context.Background(), testDomainExample, "txt"); err != nil {
					t.Fatalf("VerifyDomain returned error: %v", err)
				}
			},
		},
	})
}

func TestClientDomainMutationRequests(t *testing.T) {
	t.Parallel()

	runRequestCases(t, []requestCase{
		{
			name:             "EnableMailHosting",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"enableMailHosting"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.EnableMailHosting(context.Background(), testDomainExample); err != nil {
					t.Fatalf("EnableMailHosting returned error: %v", err)
				}
			},
		},
		{
			name:             "DisableMailHosting",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"disableMailHosting"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()

				if err := client.DisableMailHosting(context.Background(), testDomainExample); err != nil {
					t.Fatalf("DisableMailHosting returned error: %v", err)
				}
			},
		},
		{
			name:             "VerifySPF",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"VerifySpfRecord"`,
			responseBody:     `{"status":{"code":200},"data":{"spfstatus":true}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.VerifySPF(context.Background(), testDomainExample); err != nil {
					t.Fatalf("VerifySPF returned error: %v", err)
				}
			},
		},
		{
			name:             "VerifyMX",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"verifyMxRecord"`,
			responseBody:     `{"status":{"code":200},"data":{"mxstatus":true}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.VerifyMX(context.Background(), testDomainExample); err != nil {
					t.Fatalf("VerifyMX returned error: %v", err)
				}
			},
		},
		{
			name:             "SetPrimaryDomain",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"setAsPrimaryDomain"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.SetPrimaryDomain(context.Background(), testDomainExample); err != nil {
					t.Fatalf("SetPrimaryDomain returned error: %v", err)
				}
			},
		},
		{
			name:             "SetCatchAll",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"catchAllAddress":"support@example.com"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.SetCatchAll(context.Background(), testDomainExample, testSupportEmail); err != nil {
					t.Fatalf("SetCatchAll returned error: %v", err)
				}
			},
		},
		{
			name:             "DeleteCatchAll",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"deleteCatchAllAddress"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteCatchAll(context.Background(), testDomainExample); err != nil {
					t.Fatalf("DeleteCatchAll returned error: %v", err)
				}
			},
		},
		{
			name:             "EnableSubdomainStripping",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"enableSubDomainStripping"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.EnableSubdomainStripping(context.Background(), testDomainExample); err != nil {
					t.Fatalf("EnableSubdomainStripping returned error: %v", err)
				}
			},
		},
		{
			name:             "DisableSubdomainStripping",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"disableSubDomainStripping"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DisableSubdomainStripping(context.Background(), testDomainExample); err != nil {
					t.Fatalf("DisableSubdomainStripping returned error: %v", err)
				}
			},
		},
	})
}

func TestClientDomainAliasAndDKIMRequests(t *testing.T) {
	t.Parallel()

	runRequestCases(t, []requestCase{
		{
			name:             "AddDomainAlias",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"domainAlias":"alias.example.com"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.AddDomainAlias(context.Background(), testDomainExample, "alias.example.com"); err != nil {
					t.Fatalf("AddDomainAlias returned error: %v", err)
				}
			},
		},
		{
			name:             "DeleteDomainAlias",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"removeDomainAsAlias"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteDomainAlias(context.Background(), testDomainExample, "alias.example.com"); err != nil {
					t.Fatalf("DeleteDomainAlias returned error: %v", err)
				}
			},
		},
		{
			name:             "CreateDKIM",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"hashType":"sha256"`,
			responseBody:     `{"status":{"code":200},"data":{"dkimId":"dk-1","selector":"terraform","publicKey":"pub","dkimStatus":"verified"}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.CreateDKIM(context.Background(), CreateDKIMInput{
					DomainName: testDomainExample,
					HashType:   "sha256",
					Selector:   "terraform",
				})
				if err != nil {
					t.Fatalf("CreateDKIM returned error: %v", err)
				}
				if got.DKIMID != "dk-1" || got.PublicKey != "pub" {
					t.Fatalf("unexpected dkim payload: %#v", got)
				}
			},
		},
		{
			name:             "SetDefaultDKIM",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"makeDkimDefault"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.SetDefaultDKIM(context.Background(), testDomainExample, "dk-1"); err != nil {
					t.Fatalf("SetDefaultDKIM returned error: %v", err)
				}
			},
		},
		{
			name:             "VerifyDKIM",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"verifyDkimKey"`,
			responseBody:     `{"status":{"code":200},"data":{"dkimstatus":true}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.VerifyDKIM(context.Background(), testDomainExample, "dk-1"); err != nil {
					t.Fatalf("VerifyDKIM returned error: %v", err)
				}
			},
		},
		{
			name:             "DeleteDKIM",
			method:           http.MethodPut,
			path:             testDomainPath,
			wantBodyContains: `"mode":"deleteDkimDetail"`,
			responseBody:     `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteDKIM(context.Background(), testDomainExample, "dk-1"); err != nil {
					t.Fatalf("DeleteDKIM returned error: %v", err)
				}
			},
		},
	})
}

func TestClientVerificationFailures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		responseBody string
		run          func(*Client) error
	}{
		{
			name:         "VerifyDomain",
			responseBody: `{"status":{"code":200},"data":{"status":false,"message":"Verification failed due to host not found","error":"TXT_RECORD_HOST_UNKNOWN"}}`,
			run: func(client *Client) error {
				return client.VerifyDomain(context.Background(), testDomainExample, "txt")
			},
		},
		{
			name:         "VerifySPF",
			responseBody: `{"status":{"code":200},"data":{"spfstatus":false}}`,
			run: func(client *Client) error {
				return client.VerifySPF(context.Background(), testDomainExample)
			},
		},
		{
			name:         "VerifyMX",
			responseBody: `{"status":{"code":200},"data":{"mxstatus":false}}`,
			run: func(client *Client) error {
				return client.VerifyMX(context.Background(), testDomainExample)
			},
		},
		{
			name:         "VerifyDKIM",
			responseBody: `{"status":{"code":200},"data":{"dkimstatus":false}}`,
			run: func(client *Client) error {
				return client.VerifyDKIM(context.Background(), testDomainExample, "dk-1")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := testClient(t, http.MethodPut, testDomainPath, ``, tc.responseBody)
			if err := tc.run(client); err == nil {
				t.Fatalf("%s expected verification failure, got nil", tc.name)
			}
		})
	}
}

func TestClientVerifyDomainAlreadyVerifiedIsSuccess(t *testing.T) {
	t.Parallel()

	client := testClient(t, http.MethodPut, testDomainPath, ``, `{"status":{"code":400,"description":"Invalid Input"},"data":{"moreInfo":"Domain already verified","error":"DOMAIN_ALREADY_VERIFIED"}}`)
	if err := client.VerifyDomain(context.Background(), testDomainExample, "txt"); err != nil {
		t.Fatalf("VerifyDomain should treat already verified as success, got %v", err)
	}
}

func TestClientEnableMailHostingAlreadyEnabledIsSuccess(t *testing.T) {
	t.Parallel()

	client := testClient(t, http.MethodPut, testDomainPath, `"mode":"enableMailHosting"`, `{"status":{"code":400,"description":"Invalid Input"},"data":{"moreInfo":"MailHosting is already enabled for the domain","error":"MAILHOSTING_ALREADY_ENABLED"}}`)
	if err := client.EnableMailHosting(context.Background(), testDomainExample); err != nil {
		t.Fatalf("EnableMailHosting should treat already enabled as success, got %v", err)
	}
}

func TestClientVerificationAPIErrorsAreNotRetried(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(*Client) error
	}{
		{
			name: "VerifyDomain",
			run: func(client *Client) error {
				return client.VerifyDomain(context.Background(), testDomainExample, "txt")
			},
		},
		{
			name: "VerifySPF",
			run: func(client *Client) error {
				return client.VerifySPF(context.Background(), testDomainExample)
			},
		},
		{
			name: "VerifyMX",
			run: func(client *Client) error {
				return client.VerifyMX(context.Background(), testDomainExample)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"status":{"code":401,"description":"Unauthorized"},"data":{"moreInfo":"Invalid OAuth token","error":"INVALID_OAUTHTOKEN"}}`)
			}))
			defer server.Close()

			client := &Client{
				accessToken:    "token",
				baseURL:        server.URL,
				httpClient:     server.Client(),
				organizationID: "org",
			}

			err := tc.run(client)
			if err == nil {
				t.Fatal("expected verification API error, got nil")
			}
			if attempts != 1 {
				t.Fatalf("expected a single verification attempt, got %d", attempts)
			}
		})
	}
}

func TestClientDoJSONErrorHandling(t *testing.T) {
	t.Parallel()

	runDoJSONErrorCase(t, http.StatusNotFound, ``, true)
	runDoJSONErrorCase(t, http.StatusBadRequest, `{"status":{"code":404,"description":"missing","message":"not found"},"data":null}`, true)
	runDoJSONErrorCase(t, http.StatusBadRequest, `{"status":{"code":400,"description":"Invalid Input"},"data":{"moreInfo":"Verification failed due to host not found","error":"TXT_RECORD_HOST_UNKNOWN"}}`, false)
}

func TestClientHelpersAndConversions(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		AccessToken:    "token",
		DataCenter:     "eu",
		OrganizationID: "org",
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if client.OrganizationID() != "org" {
		t.Fatalf("unexpected organization id: %q", client.OrganizationID())
	}

	if got := apiPath("organization", "org", "domains", testDomainExample); got != testDomainPath {
		t.Fatalf("unexpected apiPath: %q", got)
	}

	if got := client.orgPath("domains", testDomainExample); got != testDomainPath {
		t.Fatalf("unexpected orgPath: %q", got)
	}

	rawMailbox := mailboxResponse{
		AccountID:      "acc-1",
		DisplayName:    "Support",
		MailBoxAddress: "",
		MailForward:    json.RawMessage(`[{"mailForward":"sales@example.com","deleteCopy":true,"status":"verified"}]`),
		Role:           "member",
		ZUID:           "z-1",
		EmailAddress: []emailAddress{
			{MailID: testSupportEmail, IsPrimary: true},
			{MailID: testSalesEmail, IsAlias: true},
		},
	}

	mailbox := convertMailbox(rawMailbox)
	if mailbox.MailboxAddress != testSupportEmail || len(mailbox.EmailAddresses) != 2 || len(mailbox.MailForwards) != 1 {
		t.Fatalf("unexpected converted mailbox: %#v", mailbox)
	}

	domain := convertDomain(rawDomain{
		DomainID:           "dom-1",
		DomainName:         testDomainExample,
		VerificationStatus: "verified",
		DKIMDetailList: []rawDKIM{
			{DKIMID: "dk-1", Selector: "terraform", PublicKey: "pub", DKIMStatus: "verified"},
		},
	})

	if domain.DomainID != "dom-1" || len(domain.DKIMDetails) != 1 {
		t.Fatalf("unexpected converted domain: %#v", domain)
	}

	if stringValue(true) != "true" || statusString(nil) != "" || !boolValue("true") || boolValue("false") {
		t.Fatal("unexpected helper conversion behaviour")
	}

	apiErr := &APIError{StatusCode: http.StatusNotFound}
	if !IsNotFound(apiErr) {
		t.Fatal("expected api error to be classified as not found")
	}
}

func runRequestCases(t *testing.T, cases []requestCase) {
	t.Helper()

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := testClient(t, tc.method, tc.path, tc.wantBodyContains, tc.responseBody)
			tc.run(t, client)
		})
	}
}

func runDoJSONErrorCase(t *testing.T, status int, responseBody string, wantNotFound bool) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, responseBody)
	}))
	defer server.Close()

	client := &Client{
		accessToken:    "token",
		baseURL:        server.URL,
		httpClient:     server.Client(),
		organizationID: "org",
	}

	err := client.doJSON(context.Background(), http.MethodGet, "/api/example", nil, nil)
	if err == nil {
		t.Fatal("expected doJSON to return an error")
	}
	if got := IsNotFound(err); got != wantNotFound {
		t.Fatalf("unexpected not found classification: got %v want %v", got, wantNotFound)
	}
	if strings.Contains(responseBody, "TXT_RECORD_HOST_UNKNOWN") && !strings.Contains(err.Error(), "TXT_RECORD_HOST_UNKNOWN") {
		t.Fatalf("expected doJSON error to include API error details, got %q", err)
	}
}

func testClient(t *testing.T, wantMethod, wantPath, wantBodyContains, responseBody string) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Fatalf("unexpected method: got %s want %s", r.Method, wantMethod)
		}
		if r.URL.Path != wantPath {
			t.Fatalf("unexpected path: got %s want %s", r.URL.Path, wantPath)
		}
		if got := r.Header.Get("Authorization"); got != "Zoho-oauthtoken token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if wantBodyContains != "" && !strings.Contains(string(body), wantBodyContains) {
			t.Fatalf("request body %q does not contain %q", string(body), wantBodyContains)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, responseBody)
	}))

	t.Cleanup(server.Close)

	return &Client{
		accessToken:    "token",
		baseURL:        server.URL,
		httpClient:     server.Client(),
		organizationID: "org",
	}
}
