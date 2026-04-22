// Copyright (c) 2026 Kefjbo
// SPDX-License-Identifier: MPL-2.0

package zohomail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
)

var (
	ErrForwardingStateUnavailable = errors.New("zoho mail forwarding state unavailable from account response")
)

var dataCenterBaseURLs = map[string]string{
	"ae": "https://mail.zoho.ae",
	"au": "https://mail.zoho.com.au",
	"ca": "https://mail.zoho.ca",
	"cn": "https://mail.zoho.com.cn",
	"eu": "https://mail.zoho.eu",
	"in": "https://mail.zoho.in",
	"jp": "https://mail.zoho.jp",
	"sa": "https://mail.zoho.sa",
	"us": "https://mail.zoho.com",
}

type Config struct {
	AccessToken    string
	DataCenter     string
	HTTPClient     *http.Client
	OrganizationID string
}

type Client struct {
	accessToken    string
	baseURL        string
	httpClient     *http.Client
	organizationID string
}

type APIError struct {
	Description string
	Details     string
	Message     string
	StatusCode  int
	ZohoCode    int
}

func (e *APIError) Error() string {
	switch {
	case e.Message != "" && e.Description != "" && e.Details != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s: %s (%s)", e.StatusCode, e.ZohoCode, e.Description, e.Message, e.Details)
	case e.Message != "" && e.Description != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s: %s", e.StatusCode, e.ZohoCode, e.Description, e.Message)
	case e.Description != "" && e.Details != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s (%s)", e.StatusCode, e.ZohoCode, e.Description, e.Details)
	case e.Description != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s", e.StatusCode, e.ZohoCode, e.Description)
	case e.Message != "" && e.Details != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s (%s)", e.StatusCode, e.ZohoCode, e.Message, e.Details)
	case e.Message != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s", e.StatusCode, e.ZohoCode, e.Message)
	case e.Details != "":
		return fmt.Sprintf("zoho mail api error (%d/%d): %s", e.StatusCode, e.ZohoCode, e.Details)
	default:
		return fmt.Sprintf("zoho mail api error (%d/%d)", e.StatusCode, e.ZohoCode)
	}
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound || apiErr.ZohoCode == http.StatusNotFound
	}

	return false
}

func IsDisableMailHostingRequired(err error) bool {
	return isDisableMailHostingRequiredError(err)
}

func BaseURLForDataCenter(dataCenter string) (string, error) {
	baseURL, ok := dataCenterBaseURLs[strings.ToLower(strings.TrimSpace(dataCenter))]
	if !ok {
		return "", fmt.Errorf("unsupported zoho mail data_center %q", dataCenter)
	}

	return baseURL, nil
}

func SupportedDataCenters() []string {
	keys := make([]string, 0, len(dataCenterBaseURLs))
	for key := range dataCenterBaseURLs {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}

func NewClient(cfg Config) (*Client, error) {
	baseURL, err := BaseURLForDataCenter(cfg.DataCenter)
	if err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		accessToken:    cfg.AccessToken,
		baseURL:        strings.TrimRight(baseURL, "/"),
		httpClient:     httpClient,
		organizationID: cfg.OrganizationID,
	}, nil
}

func (c *Client) OrganizationID() string {
	return c.organizationID
}

type apiEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Status apiStatus       `json:"status"`
}

type apiStatus struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Message     string `json:"message"`
}

type mailboxResponse struct {
	AccountID       any             `json:"accountId"`
	Country         string          `json:"country"`
	DisplayName     string          `json:"displayName"`
	EmailAddress    []emailAddress  `json:"emailAddress"`
	FirstName       string          `json:"firstName"`
	Language        string          `json:"language"`
	LastName        string          `json:"lastName"`
	MailBoxAddress  string          `json:"mailboxAddress"`
	MailBoxStatus   string          `json:"mailboxStatus"`
	MailForward     json.RawMessage `json:"mailForward"`
	Role            string          `json:"roleName"`
	SendMailDetails json.RawMessage `json:"sendMailDetails"`
	TimeZone        string          `json:"timeZone"`
	ZUID            any             `json:"zuid"`
}

type emailAddress struct {
	IsAlias     bool   `json:"isAlias"`
	IsConfirmed bool   `json:"isConfirmed"`
	IsPrimary   bool   `json:"isPrimary"`
	MailID      string `json:"mailId"`
}

type mailForward struct {
	DeleteCopy  bool   `json:"deleteCopy"`
	MailForward string `json:"mailForward"`
	Status      any    `json:"status"`
}

type Mailbox struct {
	AccountID      string
	Country        string
	DisplayName    string
	EmailAddresses []string
	FirstName      string
	Language       string
	LastName       string
	MailboxAddress string
	MailboxStatus  string
	MailForwards   []MailForward
	Role           string
	TimeZone       string
	ZUID           string
}

type MailForward struct {
	DeleteCopy bool
	Email      string
	Status     string
}

type CreateMailboxInput struct {
	Country             string
	DisplayName         string
	FirstName           string
	InitialPassword     string
	Language            string
	LastName            string
	OneTimePassword     bool
	PrimaryEmailAddress string
	Role                string
	TimeZone            string
}

type rawDomain struct {
	CNAMEVerificationCode any       `json:"CNAMEVerificationCode"`
	CatchAllAddress       string    `json:"catchAllAddress"`
	DomainID              string    `json:"domainId"`
	DomainName            string    `json:"domainName"`
	DKIMDetailList        []rawDKIM `json:"dkimDetailList"`
	HTMLVerificationCode  any       `json:"HTMLVerificationCode"`
	IsDomainAlias         bool      `json:"isDomainAlias"`
	IsPrimary             bool      `json:"primary"`
	MailHostingEnabled    bool      `json:"mailHostingEnabled"`
	MXStatus              any       `json:"mxstatus"`
	SPFStatus             any       `json:"spfstatus"`
	SubDomainStripping    any       `json:"subDomainStripping"`
	VerificationStatus    any       `json:"verificationStatus"`
}

type rawDKIM struct {
	DomainName string `json:"domainName"`
	DKIMID     string `json:"dkimId"`
	DKIMStatus any    `json:"dkimStatus"`
	DKIMValue  string `json:"dkimValue"`
	IsDefault  bool   `json:"isDefault"`
	PublicKey  string `json:"publicKey"`
	Selector   string `json:"selector"`
}

type verificationResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Status  bool   `json:"status"`
}

type mxVerificationResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	MXStatus bool   `json:"mxstatus"`
}

type spfVerificationResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	SPFStatus bool   `json:"spfstatus"`
}

type dkimVerificationResponse struct {
	DKIMStatus bool   `json:"dkimstatus"`
	Error      string `json:"error"`
	Message    string `json:"message"`
}

type Domain struct {
	CNAMEVerificationCode string
	CatchAllAddress       string
	DKIMDetails           []DKIMDetail
	DomainID              string
	DomainName            string
	HTMLVerificationCode  string
	IsDomainAlias         bool
	IsPrimary             bool
	MailHostingEnabled    bool
	MXStatus              string
	SPFStatus             string
	SubDomainStripping    bool
	TXTVerificationValue  string
	VerificationStatus    string
}

type DKIMDetail struct {
	DKIMID    string
	IsDefault bool
	PublicKey string
	Selector  string
	Status    string
}

type CreateDKIMInput struct {
	DomainName string
	HashType   string
	Selector   string
}

func (c *Client) CreateMailbox(ctx context.Context, input CreateMailboxInput) (*Mailbox, error) {
	payload := map[string]any{
		"country":             input.Country,
		"displayName":         input.DisplayName,
		"firstName":           input.FirstName,
		"language":            input.Language,
		"lastName":            input.LastName,
		"oneTimePassword":     input.OneTimePassword,
		"password":            input.InitialPassword,
		"primaryEmailAddress": input.PrimaryEmailAddress,
		"role":                input.Role,
		"timeZone":            input.TimeZone,
	}

	var response mailboxResponse
	if err := c.doJSON(ctx, http.MethodPost, c.orgPath("accounts"), payload, &response); err != nil {
		return nil, err
	}

	return convertMailbox(response), nil
}

func (c *Client) GetMailbox(ctx context.Context, zuid string) (*Mailbox, error) {
	var response mailboxResponse
	if err := c.doJSON(ctx, http.MethodGet, c.orgPath("accounts", zuid), nil, &response); err != nil {
		return nil, err
	}

	return convertMailbox(response), nil
}

func (c *Client) UpdateMailboxDisplayName(ctx context.Context, mailbox *Mailbox, displayName string) error {
	payload := map[string]any{
		"displayName":  displayName,
		"emailAddress": mailbox.MailboxAddress,
		"mode":         "displaynameemailupdate",
		"zuid":         mailbox.ZUID,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", mailbox.AccountID), payload, nil)
}

func (c *Client) ChangeMailboxRole(ctx context.Context, zuid string, roleName string) error {
	payload := map[string]any{
		"mode":     "changeRole",
		"roleName": roleName,
		"userList": []string{zuid},
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts"), payload, nil)
}

func (c *Client) DeleteMailbox(ctx context.Context, zuid string) error {
	payload := map[string]any{
		"accountList": []string{zuid},
	}

	return c.doJSON(ctx, http.MethodDelete, c.orgPath("accounts"), payload, nil)
}

func (c *Client) AddMailboxAlias(ctx context.Context, zuid string, alias string) error {
	payload := map[string]any{
		"emailAlias": []string{alias},
		"mode":       "addEmailAlias",
		"zuid":       zuid,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", zuid), payload, nil)
}

func (c *Client) DeleteMailboxAlias(ctx context.Context, zuid string, alias string) error {
	payload := map[string]any{
		"emailAlias": []string{alias},
		"mode":       "deleteEmailAlias",
		"zuid":       zuid,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", zuid), payload, nil)
}

func (c *Client) AddMailboxForward(ctx context.Context, mailbox *Mailbox, address string) error {
	payload := map[string]any{
		"mailForward": address,
		"mode":        "addMailForward",
		"zuid":        mailbox.ZUID,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", mailbox.AccountID), payload, nil)
}

func (c *Client) EnableMailboxForward(ctx context.Context, mailbox *Mailbox, address string) error {
	payload := map[string]any{
		"mailForward": address,
		"mode":        "enableMailForward",
		"zuid":        mailbox.ZUID,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", mailbox.AccountID), payload, nil)
}

func (c *Client) DisableMailboxForward(ctx context.Context, mailbox *Mailbox, address string) error {
	payload := map[string]any{
		"mailForward": address,
		"mode":        "disableMailForward",
		"zuid":        mailbox.ZUID,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", mailbox.AccountID), payload, nil)
}

func (c *Client) DeleteMailboxForward(ctx context.Context, mailbox *Mailbox, address string) error {
	payload := map[string]any{
		"mailForward": address,
		"mode":        "deleteMailForward",
		"zuid":        mailbox.ZUID,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", mailbox.AccountID), payload, nil)
}

func (c *Client) SetDeleteZohoMailCopy(ctx context.Context, mailbox *Mailbox, deleteCopy bool) error {
	payload := map[string]any{
		"deleteCopy": deleteCopy,
		"mode":       "deleteZohoMailCopy",
		"zuid":       mailbox.ZUID,
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("accounts", mailbox.AccountID), payload, nil)
}

func (c *Client) GetMailboxForwarding(ctx context.Context, accountID string) ([]MailForward, error) {
	var response mailboxResponse
	if err := c.doJSON(ctx, http.MethodGet, apiPath("accounts", accountID), nil, &response); err != nil {
		return nil, err
	}

	if len(bytes.TrimSpace(response.MailForward)) == 0 || bytes.Equal(bytes.TrimSpace(response.MailForward), []byte("null")) {
		return nil, ErrForwardingStateUnavailable
	}

	var forwards []mailForward
	if err := json.Unmarshal(response.MailForward, &forwards); err != nil {
		return nil, fmt.Errorf("decode mail forwarding state: %w", err)
	}

	result := make([]MailForward, 0, len(forwards))
	for _, forward := range forwards {
		if strings.TrimSpace(forward.MailForward) == "" {
			continue
		}

		result = append(result, MailForward{
			DeleteCopy: forward.DeleteCopy,
			Email:      strings.TrimSpace(forward.MailForward),
			Status:     statusString(forward.Status),
		})
	}

	return result, nil
}

func (c *Client) CreateDomain(ctx context.Context, domainName string) (*Domain, error) {
	payload := map[string]any{
		"domainName": domainName,
	}

	var response rawDomain
	if err := c.doJSON(ctx, http.MethodPost, c.orgPath("domains"), payload, &response); err != nil {
		return nil, err
	}

	domain := convertDomain(response)
	return &domain, nil
}

func (c *Client) GetDomain(ctx context.Context, domainName string) (*Domain, error) {
	var response rawDomain
	if err := c.doJSON(ctx, http.MethodGet, c.orgPath("domains", domainName), nil, &response); err != nil {
		return nil, err
	}

	domain := convertDomain(response)
	return &domain, nil
}

func (c *Client) DeleteDomain(ctx context.Context, domainName string) error {
	return c.doJSON(ctx, http.MethodDelete, c.orgPath("domains", domainName), nil, nil)
}

func (c *Client) VerifyDomain(ctx context.Context, domainName string, method string) error {
	modeByMethod := map[string]string{
		"cname": "verifyDomainByCName",
		"html":  "verifyDomainByHTML",
		"txt":   "verifyDomainByTXT",
	}

	mode, ok := modeByMethod[strings.ToLower(strings.TrimSpace(method))]
	if !ok {
		return fmt.Errorf("unsupported domain verification method %q", method)
	}

	return c.retryVerification(ctx, func(ctx context.Context) error {
		var response verificationResponse
		if err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": mode}, &response); err != nil {
			return err
		}

		if !response.Status {
			return verificationFailure("domain", response.Message, response.Error)
		}

		return nil
	})
}

func (c *Client) EnableMailHosting(ctx context.Context, domainName string) error {
	err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "enableMailHosting"}, nil)
	if isAlreadyEnabledError(err) {
		return nil
	}

	return err
}

func (c *Client) DisableMailHosting(ctx context.Context, domainName string) error {
	err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "disableMailHosting"}, nil)
	if isAlreadyDisabledError(err) {
		return nil
	}

	return err
}

func (c *Client) VerifySPF(ctx context.Context, domainName string) error {
	return c.retryVerification(ctx, func(ctx context.Context) error {
		var response spfVerificationResponse
		if err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "VerifySpfRecord"}, &response); err != nil {
			return err
		}

		if !response.SPFStatus {
			return verificationFailure("SPF", response.Message, response.Error)
		}

		return nil
	})
}

func (c *Client) VerifyMX(ctx context.Context, domainName string) error {
	return c.retryVerification(ctx, func(ctx context.Context) error {
		var response mxVerificationResponse
		if err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "verifyMxRecord"}, &response); err != nil {
			return err
		}

		if !response.MXStatus {
			return verificationFailure("MX", response.Message, response.Error)
		}

		return nil
	})
}

func (c *Client) SetPrimaryDomain(ctx context.Context, domainName string) error {
	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "setAsPrimaryDomain"}, nil)
}

func (c *Client) AddDomainAlias(ctx context.Context, primaryDomain string, aliasDomain string) error {
	payload := map[string]any{
		"domainAlias": aliasDomain,
		"mode":        "makeDomainAsAlias",
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", primaryDomain), payload, nil)
}

func (c *Client) DeleteDomainAlias(ctx context.Context, primaryDomain string, aliasDomain string) error {
	payload := map[string]any{
		"domainAlias": aliasDomain,
		"mode":        "removeDomainAsAlias",
	}

	err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", primaryDomain), payload, nil)
	if err == nil {
		return nil
	}

	fallback := map[string]any{
		"domainAlias": aliasDomain,
		"mode":        "removeDomainAlias",
	}

	if fallbackErr := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", primaryDomain), fallback, nil); fallbackErr == nil {
		return nil
	}

	return err
}

func (c *Client) CreateDKIM(ctx context.Context, input CreateDKIMInput) (*DKIMDetail, error) {
	payload := map[string]any{
		"mode":     "addDkimDetail",
		"selector": input.Selector,
	}

	var response rawDKIM
	if err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", input.DomainName), payload, &response); err != nil {
		return nil, err
	}

	dkim := convertDKIM(response)
	return &dkim, nil
}

func (c *Client) SetDefaultDKIM(ctx context.Context, domainName string, dkimID string) error {
	payload := map[string]any{
		"dkimId": dkimID,
		"mode":   "makeDkimDefault",
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), payload, nil)
}

func (c *Client) VerifyDKIM(ctx context.Context, domainName string, dkimID string) error {
	payload := map[string]any{
		"dkimId": dkimID,
		"mode":   "verifyDkimKey",
	}

	var response dkimVerificationResponse
	if err := c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), payload, &response); err != nil {
		return err
	}

	if !response.DKIMStatus {
		return verificationFailure("DKIM", response.Message, response.Error)
	}

	return nil
}

func (c *Client) DeleteDKIM(ctx context.Context, domainName string, dkimID string) error {
	payload := map[string]any{
		"dkimId": dkimID,
		"mode":   "deleteDkimDetail",
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), payload, nil)
}

func verificationFailure(kind string, message string, errorCode string) error {
	message = strings.TrimSpace(message)
	errorCode = strings.TrimSpace(errorCode)

	switch {
	case message != "" && errorCode != "":
		return fmt.Errorf("zoho mail %s verification failed: %s (%s)", kind, message, errorCode)
	case message != "":
		return fmt.Errorf("zoho mail %s verification failed: %s", kind, message)
	case errorCode != "":
		return fmt.Errorf("zoho mail %s verification failed: %s", kind, errorCode)
	default:
		return fmt.Errorf("zoho mail %s verification failed", kind)
	}
}

func (c *Client) retryVerification(ctx context.Context, fn func(context.Context) error) error {
	const (
		retryDelay   = 5 * time.Second
		retryTimeout = 2 * time.Minute
	)

	retryCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	var lastErr error

	for {
		lastErr = fn(retryCtx)
		if lastErr == nil {
			return nil
		}
		if isAlreadyVerifiedError(lastErr) {
			return nil
		}

		select {
		case <-retryCtx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("zoho mail verification still pending after %s: %w", retryTimeout, lastErr)
		case <-time.After(retryDelay):
		}
	}
}

func (c *Client) SetCatchAll(ctx context.Context, domainName string, address string) error {
	payload := map[string]any{
		"catchAllAddress": address,
		"mode":            "addCatchAllAddress",
	}

	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), payload, nil)
}

func (c *Client) DeleteCatchAll(ctx context.Context, domainName string) error {
	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "deleteCatchAllAddress"}, nil)
}

func (c *Client) EnableSubdomainStripping(ctx context.Context, domainName string) error {
	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "enableSubDomainStripping"}, nil)
}

func (c *Client) DisableSubdomainStripping(ctx context.Context, domainName string) error {
	return c.doJSON(ctx, http.MethodPut, c.orgPath("domains", domainName), map[string]any{"mode": "disableSubDomainStripping"}, nil)
}

func (c *Client) doJSON(ctx context.Context, method string, endpoint string, payload any, out any) error {
	request, err := c.newRequest(ctx, method, endpoint, payload)
	if err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send zoho mail request: %w", err)
	}
	defer response.Body.Close()

	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read zoho mail response: %w", err)
	}

	if err := emptyBodyError(response.StatusCode, response.Status, rawBody); err != nil {
		return err
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return fmt.Errorf("decode zoho mail response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest || envelope.Status.Code >= http.StatusBadRequest {
		return apiErrorFromEnvelope(response.StatusCode, envelope.Status, envelope.Data)
	}

	if out == nil || len(bytes.TrimSpace(envelope.Data)) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return nil
	}

	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode zoho mail response data: %w", err)
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, method string, endpoint string, payload any) (*http.Request, error) {
	body, hasPayload, err := marshalBody(payload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create zoho mail request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Zoho-oauthtoken "+c.accessToken)
	if hasPayload {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func marshalBody(payload any) (io.Reader, bool, error) {
	if payload == nil {
		return nil, false, nil
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf("marshal zoho mail request: %w", err)
	}

	return bytes.NewReader(rawPayload), true, nil
}

func emptyBodyError(statusCode int, status string, rawBody []byte) error {
	if len(bytes.TrimSpace(rawBody)) != 0 {
		return nil
	}

	if statusCode >= http.StatusBadRequest {
		return &APIError{
			Description: status,
			StatusCode:  statusCode,
		}
	}

	return nil
}

func apiErrorFromEnvelope(statusCode int, status apiStatus, rawData json.RawMessage) error {
	if status.Code != 0 {
		statusCode = status.Code
	}

	return &APIError{
		Description: status.Description,
		Details:     apiErrorDetails(rawData),
		Message:     status.Message,
		StatusCode:  statusCode,
		ZohoCode:    status.Code,
	}
}

func apiErrorDetails(rawData json.RawMessage) string {
	if len(bytes.TrimSpace(rawData)) == 0 || bytes.Equal(bytes.TrimSpace(rawData), []byte("null")) {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(rawData, &payload); err != nil {
		return ""
	}

	parts := make([]string, 0, 3)
	for _, key := range []string{"moreInfo", "error", "errorData"} {
		value, ok := payload[key]
		if !ok {
			continue
		}

		text := strings.TrimSpace(stringValue(value))
		if text == "" {
			continue
		}

		parts = append(parts, text)
	}

	return strings.Join(parts, "; ")
}

func isAlreadyVerifiedError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	text := strings.ToUpper(strings.Join([]string{apiErr.Description, apiErr.Message, apiErr.Details}, " "))
	return strings.Contains(text, "ALREADY_VERIFIED")
}

func isAlreadyEnabledError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	text := strings.ToUpper(strings.Join([]string{apiErr.Description, apiErr.Message, apiErr.Details}, " "))
	return strings.Contains(text, "ALREADY ENABLED") || strings.Contains(text, "ALREADY_ENABLED")
}

func isAlreadyDisabledError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	text := strings.ToUpper(strings.Join([]string{apiErr.Description, apiErr.Message, apiErr.Details}, " "))
	return strings.Contains(text, "NOT ENABLED") || strings.Contains(text, "ALREADY DISABLED") || strings.Contains(text, "ALREADY_DISABLED")
}

func isDisableMailHostingRequiredError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	text := strings.ToUpper(strings.Join([]string{apiErr.Description, apiErr.Message, apiErr.Details}, " "))
	return strings.Contains(text, "RMV_HOSTING_BEFORE_DLT_DOMAIN")
}

func (c *Client) orgPath(parts ...string) string {
	prefixed := append([]string{"organization", c.organizationID}, parts...)
	return apiPath(prefixed...)
}

func apiPath(parts ...string) string {
	escaped := make([]string, 0, len(parts)+1)
	escaped = append(escaped, "/api")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		escaped = append(escaped, url.PathEscape(trimmed))
	}

	return path.Join(escaped...)
}

func convertMailbox(raw mailboxResponse) *Mailbox {
	result := &Mailbox{
		AccountID:      stringValue(raw.AccountID),
		Country:        strings.TrimSpace(raw.Country),
		DisplayName:    strings.TrimSpace(raw.DisplayName),
		FirstName:      strings.TrimSpace(raw.FirstName),
		Language:       strings.TrimSpace(raw.Language),
		LastName:       strings.TrimSpace(raw.LastName),
		MailboxAddress: strings.TrimSpace(raw.MailBoxAddress),
		MailboxStatus:  strings.TrimSpace(raw.MailBoxStatus),
		Role:           strings.TrimSpace(raw.Role),
		TimeZone:       strings.TrimSpace(raw.TimeZone),
		ZUID:           stringValue(raw.ZUID),
	}

	result.EmailAddresses, result.MailboxAddress = convertEmailAddresses(raw.EmailAddress, result.MailboxAddress)
	result.MailForwards = convertMailForwards(raw.MailForward)

	return result
}

func convertEmailAddresses(emails []emailAddress, mailboxAddress string) ([]string, string) {
	addresses := make([]string, 0, len(emails))

	for _, email := range emails {
		address := strings.TrimSpace(email.MailID)
		if address == "" {
			continue
		}

		addresses = append(addresses, address)
		if email.IsPrimary && mailboxAddress == "" {
			mailboxAddress = address
		}
	}

	return addresses, mailboxAddress
}

func convertMailForwards(raw json.RawMessage) []MailForward {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}

	var forwards []mailForward
	if err := json.Unmarshal(raw, &forwards); err != nil {
		return nil
	}

	result := make([]MailForward, 0, len(forwards))
	for _, forward := range forwards {
		address := strings.TrimSpace(forward.MailForward)
		if address == "" {
			continue
		}

		result = append(result, MailForward{
			DeleteCopy: forward.DeleteCopy,
			Email:      address,
			Status:     statusString(forward.Status),
		})
	}

	return result
}

func convertDomain(raw rawDomain) Domain {
	cnameCode := stringValue(raw.CNAMEVerificationCode)

	result := Domain{
		CNAMEVerificationCode: cnameCode,
		CatchAllAddress:       strings.TrimSpace(raw.CatchAllAddress),
		DomainID:              strings.TrimSpace(raw.DomainID),
		DomainName:            strings.TrimSpace(raw.DomainName),
		HTMLVerificationCode:  stringValue(raw.HTMLVerificationCode),
		IsDomainAlias:         raw.IsDomainAlias,
		IsPrimary:             raw.IsPrimary,
		MailHostingEnabled:    raw.MailHostingEnabled,
		MXStatus:              statusString(raw.MXStatus),
		SPFStatus:             statusString(raw.SPFStatus),
		SubDomainStripping:    boolValue(raw.SubDomainStripping),
		TXTVerificationValue:  domainTXTVerificationValue(cnameCode),
		VerificationStatus:    statusString(raw.VerificationStatus),
	}

	for _, item := range raw.DKIMDetailList {
		result.DKIMDetails = append(result.DKIMDetails, convertDKIM(item))
	}

	return result
}

func domainTXTVerificationValue(cnameCode string) string {
	if strings.TrimSpace(cnameCode) == "" {
		return ""
	}

	// Zoho Mail TXT verification uses the zb code returned by the domain API.
	return "zoho-verification=" + cnameCode + ".zmverify.zoho.com"
}

func convertDKIM(raw rawDKIM) DKIMDetail {
	return DKIMDetail{
		DKIMID:    strings.TrimSpace(raw.DKIMID),
		IsDefault: raw.IsDefault,
		PublicKey: strings.TrimSpace(raw.PublicKey),
		Selector:  strings.TrimSpace(raw.Selector),
		Status:    statusString(raw.DKIMStatus),
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "true"
		}

		return "false"
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func statusString(value any) string {
	result := stringValue(value)
	return result
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes":
			return true
		default:
			return false
		}
	case float64:
		return typed != 0
	default:
		return false
	}
}
