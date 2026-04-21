// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

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

func TestClientMailboxRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		method       string
		path         string
		responseBody string
		run          func(t *testing.T, client *Client)
		wantBody     string
	}{
		{
			name:         "CreateMailbox",
			method:       http.MethodPost,
			path:         "/api/organization/org/accounts",
			wantBody:     `"mailboxAddress":"support@example.com"`,
			responseBody: `{"status":{"code":200},"data":{"zuid":"1001","accountId":"2002","mailboxAddress":"support@example.com","roleName":"member","emailAddress":[{"mailId":"support@example.com","isPrimary":true}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.CreateMailbox(context.Background(), CreateMailboxInput{
					Country:             "FR",
					DisplayName:         "Support",
					FirstName:           "Support",
					InitialPassword:     "secret",
					Language:            "fr",
					LastName:            "Team",
					PrimaryEmailAddress: "support@example.com",
					Role:                "member",
					TimeZone:            "Europe/Paris",
				})
				if err != nil {
					t.Fatalf("CreateMailbox returned error: %v", err)
				}
				if got.ZUID != "1001" || got.AccountID != "2002" {
					t.Fatalf("unexpected mailbox response: %#v", got)
				}
			},
		},
		{
			name:         "GetMailbox",
			method:       http.MethodGet,
			path:         "/api/organization/org/accounts/1001",
			responseBody: `{"status":{"code":200},"data":{"zuid":"1001","accountId":"2002","mailboxAddress":"support@example.com","roleName":"member","emailAddress":[{"mailId":"support@example.com","isPrimary":true}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.GetMailbox(context.Background(), "1001")
				if err != nil {
					t.Fatalf("GetMailbox returned error: %v", err)
				}
				if got.MailboxAddress != "support@example.com" {
					t.Fatalf("unexpected mailbox address: %#v", got)
				}
			},
		},
		{
			name:         "UpdateMailboxDisplayName",
			method:       http.MethodPut,
			path:         "/api/organization/org/accounts/2002",
			wantBody:     `"mode":"displaynameemailupdate"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				err := client.UpdateMailboxDisplayName(context.Background(), &Mailbox{
					AccountID:      "2002",
					MailboxAddress: "support@example.com",
					ZUID:           "1001",
				}, "Support Team")
				if err != nil {
					t.Fatalf("UpdateMailboxDisplayName returned error: %v", err)
				}
			},
		},
		{
			name:         "ChangeMailboxRole",
			method:       http.MethodPut,
			path:         "/api/organization/org/accounts",
			wantBody:     `"mode":"changeRole"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.ChangeMailboxRole(context.Background(), "1001", "admin"); err != nil {
					t.Fatalf("ChangeMailboxRole returned error: %v", err)
				}
			},
		},
		{
			name:         "DeleteMailbox",
			method:       http.MethodDelete,
			path:         "/api/organization/org/accounts",
			wantBody:     `"accountList":["1001"]`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteMailbox(context.Background(), "1001"); err != nil {
					t.Fatalf("DeleteMailbox returned error: %v", err)
				}
			},
		},
		{
			name:         "AddMailboxAlias",
			method:       http.MethodPut,
			path:         "/api/organization/org/accounts/1001",
			wantBody:     `"mode":"addEmailAlias"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.AddMailboxAlias(context.Background(), "1001", "sales@example.com"); err != nil {
					t.Fatalf("AddMailboxAlias returned error: %v", err)
				}
			},
		},
		{
			name:         "DeleteMailboxAlias",
			method:       http.MethodPut,
			path:         "/api/organization/org/accounts/1001",
			wantBody:     `"mode":"deleteEmailAlias"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteMailboxAlias(context.Background(), "1001", "sales@example.com"); err != nil {
					t.Fatalf("DeleteMailboxAlias returned error: %v", err)
				}
			},
		},
		{
			name:         "MailboxForwardingMutations",
			method:       http.MethodPut,
			path:         "/api/organization/org/accounts/2002",
			wantBody:     `"mailForward":"sales@example.com"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				mailbox := &Mailbox{AccountID: "2002", ZUID: "1001"}
				for _, call := range []func(context.Context, *Mailbox, string) error{
					client.AddMailboxForward,
					client.EnableMailboxForward,
					client.DisableMailboxForward,
					client.DeleteMailboxForward,
				} {
					if err := call(context.Background(), mailbox, "sales@example.com"); err != nil {
						t.Fatalf("mail forwarding mutation returned error: %v", err)
					}
				}
			},
		},
		{
			name:         "SetDeleteZohoMailCopy",
			method:       http.MethodPut,
			path:         "/api/organization/org/accounts/2002",
			wantBody:     `"mode":"deleteZohoMailCopy"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.SetDeleteZohoMailCopy(context.Background(), &Mailbox{AccountID: "2002", ZUID: "1001"}, true); err != nil {
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
				if len(got) != 1 || got[0].Email != "sales@example.com" || !got[0].DeleteCopy {
					t.Fatalf("unexpected mailbox forwarding payload: %#v", got)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := testClient(t, tc.method, tc.path, tc.wantBody, tc.responseBody)
			tc.run(t, client)
		})
	}
}

func TestClientDomainRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		method       string
		path         string
		responseBody string
		run          func(t *testing.T, client *Client)
		wantBody     string
	}{
		{
			name:         "CreateDomain",
			method:       http.MethodPost,
			path:         "/api/organization/org/domains",
			wantBody:     `"domainName":"example.com"`,
			responseBody: `{"status":{"code":200},"data":{"domainName":"example.com","domainId":"dom-1","verificationStatus":"pending"}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.CreateDomain(context.Background(), "example.com")
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
			path:         "/api/organization/org/domains/example.com",
			responseBody: `{"status":{"code":200},"data":{"domainName":"example.com","domainId":"dom-1","verificationStatus":"verified","dkimDetailList":[{"dkimId":"dk-1","selector":"terraform","publicKey":"pub","dkimStatus":"verified"}]}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.GetDomain(context.Background(), "example.com")
				if err != nil {
					t.Fatalf("GetDomain returned error: %v", err)
				}
				if got.DomainName != "example.com" || len(got.DKIMDetails) != 1 {
					t.Fatalf("unexpected domain payload: %#v", got)
				}
			},
		},
		{
			name:         "DeleteDomain",
			method:       http.MethodDelete,
			path:         "/api/organization/org/domains/example.com",
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.DeleteDomain(context.Background(), "example.com"); err != nil {
					t.Fatalf("DeleteDomain returned error: %v", err)
				}
			},
		},
		{
			name:         "VerifyDomain",
			method:       http.MethodPut,
			path:         "/api/organization/org/domains/example.com",
			wantBody:     `"mode":"verifyDomainByTXT"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.VerifyDomain(context.Background(), "example.com", "txt"); err != nil {
					t.Fatalf("VerifyDomain returned error: %v", err)
				}
			},
		},
		{
			name:         "DomainMutations",
			method:       http.MethodPut,
			path:         "/api/organization/org/domains/example.com",
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				calls := []func(context.Context, string) error{
					client.EnableMailHosting,
					client.VerifySPF,
					client.VerifyMX,
					client.SetPrimaryDomain,
					client.DeleteCatchAll,
					client.EnableSubdomainStripping,
					client.DisableSubdomainStripping,
				}
				for _, call := range calls {
					if err := call(context.Background(), "example.com"); err != nil {
						t.Fatalf("domain mutation returned error: %v", err)
					}
				}
			},
		},
		{
			name:         "DomainAliasMutations",
			method:       http.MethodPut,
			path:         "/api/organization/org/domains/example.com",
			wantBody:     `"aliasDomainName":"alias.example.com"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.AddDomainAlias(context.Background(), "example.com", "alias.example.com"); err != nil {
					t.Fatalf("AddDomainAlias returned error: %v", err)
				}
				if err := client.DeleteDomainAlias(context.Background(), "example.com", "alias.example.com"); err != nil {
					t.Fatalf("DeleteDomainAlias returned error: %v", err)
				}
			},
		},
		{
			name:         "CreateDKIM",
			method:       http.MethodPut,
			path:         "/api/organization/org/domains/example.com",
			wantBody:     `"mode":"generateDkimKey"`,
			responseBody: `{"status":{"code":200},"data":{"dkimId":"dk-1","selector":"terraform","publicKey":"pub","dkimStatus":"verified"}}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				got, err := client.CreateDKIM(context.Background(), CreateDKIMInput{
					DomainName: "example.com",
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
			name:         "DKIMMutations",
			method:       http.MethodPut,
			path:         "/api/organization/org/domains/example.com",
			wantBody:     `"dkimId":"dk-1"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				calls := []func(context.Context, string, string) error{
					client.SetDefaultDKIM,
					client.VerifyDKIM,
					client.DeleteDKIM,
				}
				for _, call := range calls {
					if err := call(context.Background(), "example.com", "dk-1"); err != nil {
						t.Fatalf("dkim mutation returned error: %v", err)
					}
				}
			},
		},
		{
			name:         "SetCatchAll",
			method:       http.MethodPut,
			path:         "/api/organization/org/domains/example.com",
			wantBody:     `"catchAllAddress":"support@example.com"`,
			responseBody: `{"status":{"code":200},"data":null}`,
			run: func(t *testing.T, client *Client) {
				t.Helper()
				if err := client.SetCatchAll(context.Background(), "example.com", "support@example.com"); err != nil {
					t.Fatalf("SetCatchAll returned error: %v", err)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := testClient(t, tc.method, tc.path, tc.wantBody, tc.responseBody)
			tc.run(t, client)
		})
	}
}

func TestClientDoJSONErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       int
		responseBody string
		wantNotFound bool
	}{
		{
			name:         "EmptyErrorBody",
			status:       http.StatusNotFound,
			responseBody: ``,
			wantNotFound: true,
		},
		{
			name:         "EnvelopeError",
			status:       http.StatusBadRequest,
			responseBody: `{"status":{"code":404,"description":"missing","message":"not found"},"data":null}`,
			wantNotFound: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.responseBody)
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
			if got := IsNotFound(err); got != tc.wantNotFound {
				t.Fatalf("unexpected not found classification: got %v want %v", got, tc.wantNotFound)
			}
		})
	}
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

	if got := apiPath("organization", "org", "domains", "example.com"); got != "/api/organization/org/domains/example.com" {
		t.Fatalf("unexpected apiPath: %q", got)
	}

	if got := client.orgPath("domains", "example.com"); got != "/api/organization/org/domains/example.com" {
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
			{MailID: "support@example.com", IsPrimary: true},
			{MailID: "sales@example.com", IsAlias: true},
		},
	}

	mailbox := convertMailbox(rawMailbox)
	if mailbox.MailboxAddress != "support@example.com" || len(mailbox.EmailAddresses) != 2 || len(mailbox.MailForwards) != 1 {
		t.Fatalf("unexpected converted mailbox: %#v", mailbox)
	}

	domain := convertDomain(rawDomain{
		DomainID:           "dom-1",
		DomainName:         "example.com",
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
