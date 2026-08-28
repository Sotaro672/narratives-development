// backend/internal/adapters/out/stripe/account_gateway.go
package stripe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
)

const (
	stripeAccountsV2BaseURL = "https://api.stripe.com/v2/core"
	stripeAccountsV1BaseURL = "https://api.stripe.com/v1"

	// Stripe Accounts v2 の安定版APIバージョン。
	stripeAccountsAPIVersion = "2026-07-29.dahlia"

	defaultAccountCountry = "JP"
)

// ========================================
// AccountGateway
// ========================================

// AccountGateway は Stripe Connect の Connected Account を扱います。
//
// Company Account と Mall User の PayoutAccount の双方から共用します。
// Firestoreへの保存や所有権検証は application/usecase 側の責務です。
type AccountGateway struct {
	secretKey  string
	httpClient *http.Client
}

var _ applicationport.StripeAccountGateway = (*AccountGateway)(nil)
var _ applicationport.StripePayoutAccountGateway = (*AccountGateway)(nil)

// NewAccountGateway creates a Stripe Connected Account gateway.
func NewAccountGateway(secretKey string) *AccountGateway {
	return &AccountGateway{
		secretKey: strings.TrimSpace(secretKey),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ========================================
// Company Account
// ========================================

// CreateAccount creates a Company-owned Stripe Connected Account.
func (g *AccountGateway) CreateAccount(
	ctx context.Context,
	in applicationport.CreateStripeAccountInput,
) (*applicationport.StripeAccountResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	accountID := strings.TrimSpace(in.AccountID)
	companyID := strings.TrimSpace(in.CompanyID)
	displayName := strings.TrimSpace(in.DisplayName)
	contactEmail := strings.TrimSpace(in.ContactEmail)
	country := strings.ToUpper(strings.TrimSpace(in.Country))

	if accountID == "" {
		return nil, errors.New("stripe account: accountId is empty")
	}
	if companyID == "" {
		return nil, errors.New("stripe account: companyId is empty")
	}
	if displayName == "" {
		return nil, errors.New("stripe account: displayName is empty")
	}
	if country == "" {
		country = defaultAccountCountry
	}

	out, err := g.createRecipientAccount(
		ctx,
		displayName,
		contactEmail,
		country,
		map[string]string{
			"accountId": accountID,
			"companyId": companyID,
			"ownerType": "company_account",
		},
		strings.TrimSpace(in.IdempotencyKey),
	)
	if err != nil {
		return nil, err
	}

	return accountResultFromResponse(*out)
}

// GetAccount retrieves a Company Stripe Connected Account.
func (g *AccountGateway) GetAccount(
	ctx context.Context,
	stripeAccountID string,
) (*applicationport.StripeAccountResult, error) {
	out, err := g.getAccount(ctx, stripeAccountID)
	if err != nil {
		return nil, err
	}

	return accountResultFromResponse(*out)
}

// CreateOnboardingLink creates a single-use Stripe hosted onboarding URL for
// the existing Company Account flow.
func (g *AccountGateway) CreateOnboardingLink(
	ctx context.Context,
	in applicationport.CreateStripeAccountLinkInput,
) (*applicationport.StripeAccountLinkResult, error) {
	out, err := g.createAccountLink(
		ctx,
		strings.TrimSpace(in.StripeAccountID),
		"account_onboarding",
		strings.TrimSpace(in.ReturnURL),
		strings.TrimSpace(in.RefreshURL),
	)
	if err != nil {
		return nil, err
	}

	return &applicationport.StripeAccountLinkResult{
		AccountID: strings.TrimSpace(out.Account),
		URL:       strings.TrimSpace(out.URL),
		ExpiresAt: out.ExpiresAt,
	}, nil
}

// ========================================
// Mall User Payout Account
// ========================================

// CreatePayoutAccount creates the Stripe Connected Account used as the Mall
// user's resale payout destination.
//
// One User -> one Connected Account is enforced by the application layer and
// by the stable Idempotency-Key supplied to this method.
func (g *AccountGateway) CreatePayoutAccount(
	ctx context.Context,
	in applicationport.CreateStripePayoutAccountInput,
) (*applicationport.StripePayoutAccountResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	userID := strings.TrimSpace(in.UserID)
	displayName := strings.TrimSpace(in.DisplayName)
	contactEmail := strings.TrimSpace(in.ContactEmail)
	country := strings.ToUpper(strings.TrimSpace(in.Country))

	if userID == "" {
		return nil, errors.New("stripe payout account: userId is empty")
	}
	if displayName == "" {
		return nil, errors.New("stripe payout account: displayName is empty")
	}
	if country == "" {
		country = defaultAccountCountry
	}

	out, err := g.createRecipientAccount(
		ctx,
		displayName,
		contactEmail,
		country,
		map[string]string{
			"userId":    userID,
			"ownerType": "mall_user",
		},
		strings.TrimSpace(in.IdempotencyKey),
	)
	if err != nil {
		return nil, err
	}

	return payoutAccountResultFromResponse(*out)
}

// GetPayoutAccount retrieves the current Stripe state of a Mall user's
// Connected Account.
func (g *AccountGateway) GetPayoutAccount(
	ctx context.Context,
	stripeAccountID string,
) (*applicationport.StripePayoutAccountResult, error) {
	out, err := g.getAccount(ctx, stripeAccountID)
	if err != nil {
		return nil, err
	}

	return payoutAccountResultFromResponse(*out)
}

// CreatePayoutAccountLink creates a single-use hosted flow.
//
// Supported use cases:
//   - account_onboarding: first registration or outstanding onboarding
//   - account_update: previously onboarded account maintenance
func (g *AccountGateway) CreatePayoutAccountLink(
	ctx context.Context,
	in applicationport.CreateStripePayoutAccountLinkInput,
) (*applicationport.StripePayoutAccountLinkResult, error) {
	useCase := strings.TrimSpace(string(in.UseCase))

	switch useCase {
	case string(applicationport.StripePayoutAccountLinkUseCaseOnboarding),
		string(applicationport.StripePayoutAccountLinkUseCaseUpdate):
	default:
		return nil, errors.New("stripe payout account: invalid account link use case")
	}

	out, err := g.createAccountLink(
		ctx,
		strings.TrimSpace(in.StripeAccountID),
		useCase,
		strings.TrimSpace(in.ReturnURL),
		strings.TrimSpace(in.RefreshURL),
	)
	if err != nil {
		return nil, err
	}

	return &applicationport.StripePayoutAccountLinkResult{
		AccountID: strings.TrimSpace(out.Account),
		URL:       strings.TrimSpace(out.URL),
		ExpiresAt: out.ExpiresAt,
	}, nil
}

// GetPayoutBankAccount returns display-only information about the most recent
// bank account attached to the Connected Account.
//
// Full account numbers and routing numbers are intentionally discarded.
func (g *AccountGateway) GetPayoutBankAccount(
	ctx context.Context,
	stripeAccountID string,
) (*applicationport.StripePayoutBankAccountResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeAccountID = strings.TrimSpace(stripeAccountID)
	if !isValidStripeAccountID(stripeAccountID) {
		return nil, errors.New("stripe payout account: invalid stripeAccountId")
	}

	query := url.Values{}
	query.Set("object", "bank_account")
	query.Set("limit", "1")

	var out stripeExternalBankAccountListResponse
	if err := g.doJSONWithBaseURL(
		ctx,
		stripeAccountsV1BaseURL,
		http.MethodGet,
		"/accounts/"+url.PathEscape(stripeAccountID)+"/external_accounts",
		query,
		nil,
		"",
		&out,
	); err != nil {
		return nil, err
	}

	if len(out.Data) == 0 {
		return nil, nil
	}

	bank := out.Data[0]
	bankName := strings.TrimSpace(bank.BankName)
	last4 := strings.TrimSpace(bank.Last4)

	if last4 != "" && !isValidLast4(last4) {
		return nil, errors.New("stripe payout account: invalid bank account last4")
	}

	return &applicationport.StripePayoutBankAccountResult{
		BankName: bankName,
		Last4:    last4,
	}, nil
}

// ========================================
// Shared Account operations
// ========================================

func (g *AccountGateway) createRecipientAccount(
	ctx context.Context,
	displayName string,
	contactEmail string,
	country string,
	metadata map[string]string,
	idempotencyKey string,
) (*stripeAccountResponse, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	displayName = strings.TrimSpace(displayName)
	contactEmail = strings.TrimSpace(contactEmail)
	country = strings.ToUpper(strings.TrimSpace(country))

	if displayName == "" {
		return nil, errors.New("stripe account: displayName is empty")
	}
	if country == "" {
		country = defaultAccountCountry
	}

	reqBody := createAccountRequest{
		DisplayName:  displayName,
		ContactEmail: contactEmail,
		Dashboard:    "express",
		Identity: accountIdentityRequest{
			Country: country,
		},
		Defaults: accountDefaultsRequest{
			Responsibilities: accountResponsibilitiesRequest{
				FeesCollector:   "application",
				LossesCollector: "application",
			},
		},
		Configuration: accountConfigurationRequest{
			Recipient: accountRecipientConfigurationRequest{
				Capabilities: accountRecipientCapabilitiesRequest{
					StripeBalance: accountStripeBalanceCapabilityRequest{
						StripeTransfers: accountStripeTransfersCapabilityRequest{
							Requested: true,
						},
					},
				},
			},
		},
		Include: []string{
			"configuration.recipient",
			"identity",
			"requirements",
		},
		Metadata: metadata,
	}

	var out stripeAccountResponse

	if err := g.doJSON(
		ctx,
		http.MethodPost,
		"/accounts",
		nil,
		reqBody,
		strings.TrimSpace(idempotencyKey),
		&out,
	); err != nil {
		return nil, err
	}

	if !isValidStripeAccountID(out.ID) {
		return nil, errors.New("stripe account: stripe account id is empty or invalid")
	}

	return &out, nil
}

func (g *AccountGateway) getAccount(
	ctx context.Context,
	stripeAccountID string,
) (*stripeAccountResponse, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeAccountID = strings.TrimSpace(stripeAccountID)
	if !isValidStripeAccountID(stripeAccountID) {
		return nil, errors.New("stripe account: invalid stripeAccountId")
	}

	query := url.Values{}
	query.Add("include[0]", "configuration.recipient")
	query.Add("include[1]", "identity")
	query.Add("include[2]", "requirements")

	var out stripeAccountResponse

	if err := g.doJSON(
		ctx,
		http.MethodGet,
		"/accounts/"+url.PathEscape(stripeAccountID),
		query,
		nil,
		"",
		&out,
	); err != nil {
		return nil, err
	}

	if strings.TrimSpace(out.ID) != stripeAccountID {
		return nil, errors.New("stripe account: retrieved account id mismatch")
	}

	return &out, nil
}

func (g *AccountGateway) createAccountLink(
	ctx context.Context,
	stripeAccountID string,
	useCase string,
	returnURL string,
	refreshURL string,
) (*stripeAccountLinkResponse, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeAccountID = strings.TrimSpace(stripeAccountID)
	useCase = strings.TrimSpace(useCase)
	returnURL = strings.TrimSpace(returnURL)
	refreshURL = strings.TrimSpace(refreshURL)

	if !isValidStripeAccountID(stripeAccountID) {
		return nil, errors.New("stripe account: invalid stripeAccountId")
	}
	if !isValidHTTPURL(returnURL) {
		return nil, errors.New("stripe account: invalid returnUrl")
	}
	if !isValidHTTPURL(refreshURL) {
		return nil, errors.New("stripe account: invalid refreshUrl")
	}

	reqBody := createAccountLinkRequest{
		Account: stripeAccountID,
	}

	switch useCase {
	case "account_onboarding":
		reqBody.UseCase = accountLinkUseCaseRequest{
			Type: "account_onboarding",
			AccountOnboarding: &accountOnboardingRequest{
				Configurations: []string{"recipient"},
				ReturnURL:      returnURL,
				RefreshURL:     refreshURL,
			},
		}

	case "account_update":
		reqBody.UseCase = accountLinkUseCaseRequest{
			Type: "account_update",
			AccountUpdate: &accountUpdateRequest{
				Configurations: []string{"recipient"},
				ReturnURL:      returnURL,
				RefreshURL:     refreshURL,
			},
		}

	default:
		return nil, errors.New("stripe account: invalid account link use case")
	}

	var out stripeAccountLinkResponse

	if err := g.doJSON(
		ctx,
		http.MethodPost,
		"/account_links",
		nil,
		reqBody,
		"",
		&out,
	); err != nil {
		return nil, err
	}

	out.Account = strings.TrimSpace(out.Account)
	out.URL = strings.TrimSpace(out.URL)

	if out.Account == "" || out.Account != stripeAccountID {
		return nil, errors.New("stripe account: account link account is empty or mismatched")
	}
	if out.URL == "" {
		return nil, errors.New("stripe account: account link url is empty")
	}

	return &out, nil
}

// ========================================
// Request DTO
// ========================================

type createAccountRequest struct {
	DisplayName   string                      `json:"display_name"`
	ContactEmail  string                      `json:"contact_email,omitempty"`
	Dashboard     string                      `json:"dashboard"`
	Identity      accountIdentityRequest      `json:"identity"`
	Defaults      accountDefaultsRequest      `json:"defaults"`
	Configuration accountConfigurationRequest `json:"configuration"`
	Include       []string                    `json:"include"`
	Metadata      map[string]string           `json:"metadata,omitempty"`
}

type accountIdentityRequest struct {
	Country string `json:"country"`
}

type accountDefaultsRequest struct {
	Responsibilities accountResponsibilitiesRequest `json:"responsibilities"`
}

type accountResponsibilitiesRequest struct {
	FeesCollector   string `json:"fees_collector"`
	LossesCollector string `json:"losses_collector"`
}

type accountConfigurationRequest struct {
	Recipient accountRecipientConfigurationRequest `json:"recipient"`
}

type accountRecipientConfigurationRequest struct {
	Capabilities accountRecipientCapabilitiesRequest `json:"capabilities"`
}

type accountRecipientCapabilitiesRequest struct {
	StripeBalance accountStripeBalanceCapabilityRequest `json:"stripe_balance"`
}

type accountStripeBalanceCapabilityRequest struct {
	StripeTransfers accountStripeTransfersCapabilityRequest `json:"stripe_transfers"`
}

type accountStripeTransfersCapabilityRequest struct {
	Requested bool `json:"requested"`
}

type createAccountLinkRequest struct {
	Account string                    `json:"account"`
	UseCase accountLinkUseCaseRequest `json:"use_case"`
}

type accountLinkUseCaseRequest struct {
	Type              string                    `json:"type"`
	AccountOnboarding *accountOnboardingRequest `json:"account_onboarding,omitempty"`
	AccountUpdate     *accountUpdateRequest     `json:"account_update,omitempty"`
}

type accountOnboardingRequest struct {
	Configurations []string `json:"configurations"`
	ReturnURL      string   `json:"return_url,omitempty"`
	RefreshURL     string   `json:"refresh_url"`
}

type accountUpdateRequest struct {
	Configurations []string `json:"configurations"`
	ReturnURL      string   `json:"return_url,omitempty"`
	RefreshURL     string   `json:"refresh_url"`
}

// ========================================
// Response DTO
// ========================================

type stripeAccountResponse struct {
	ID           string    `json:"id"`
	Object       string    `json:"object"`
	DisplayName  string    `json:"display_name"`
	ContactEmail string    `json:"contact_email"`
	Dashboard    string    `json:"dashboard"`
	Livemode     bool      `json:"livemode"`
	Closed       *bool     `json:"closed"`
	Created      time.Time `json:"created"`

	Identity struct {
		Country string `json:"country"`
	} `json:"identity"`

	Configuration struct {
		Recipient struct {
			Capabilities struct {
				StripeBalance struct {
					StripeTransfers stripeCapabilityResponse `json:"stripe_transfers"`
				} `json:"stripe_balance"`
			} `json:"capabilities"`
		} `json:"recipient"`
	} `json:"configuration"`

	Requirements stripeRequirementsResponse `json:"requirements"`
}

type stripeCapabilityResponse struct {
	Status        string                         `json:"status"`
	StatusDetails []stripeCapabilityStatusDetail `json:"status_details"`
}

type stripeCapabilityStatusDetail struct {
	Code       string `json:"code"`
	Resolution string `json:"resolution"`
}

type stripeRequirementsResponse struct {
	Entries []stripeRequirementEntry  `json:"entries"`
	Summary stripeRequirementsSummary `json:"summary"`
}

type stripeRequirementEntry struct {
	AwaitingActionFrom string `json:"awaiting_action_from"`

	MinimumDeadline struct {
		Status string `json:"status"`
	} `json:"minimum_deadline"`
}

type stripeRequirementsSummary struct {
	MinimumDeadline *struct {
		Status string `json:"status"`
	} `json:"minimum_deadline"`
}

type stripeAccountLinkResponse struct {
	Object    string    `json:"object"`
	Account   string    `json:"account"`
	Created   time.Time `json:"created"`
	ExpiresAt time.Time `json:"expires_at"`
	Livemode  bool      `json:"livemode"`
	URL       string    `json:"url"`
}

type stripeExternalBankAccountListResponse struct {
	Object  string                      `json:"object"`
	Data    []stripeExternalBankAccount `json:"data"`
	HasMore bool                        `json:"has_more"`
	URL     string                      `json:"url"`
}

type stripeExternalBankAccount struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	BankName string `json:"bank_name"`
	Last4    string `json:"last4"`
	Status   string `json:"status"`
	Account  string `json:"account"`
}

// ========================================
// Result mapping
// ========================================

func accountResultFromResponse(
	out stripeAccountResponse,
) (*applicationport.StripeAccountResult, error) {
	id := strings.TrimSpace(out.ID)
	if !isValidStripeAccountID(id) {
		return nil, errors.New("stripe account: stripe account id is empty or invalid")
	}

	closed := out.Closed != nil && *out.Closed

	return &applicationport.StripeAccountResult{
		ID:           id,
		DisplayName:  strings.TrimSpace(out.DisplayName),
		ContactEmail: strings.TrimSpace(out.ContactEmail),
		Country:      strings.ToUpper(strings.TrimSpace(out.Identity.Country)),
		Dashboard:    strings.TrimSpace(out.Dashboard),
		Livemode:     out.Livemode,
		Closed:       closed,
		RecipientTransferStatus: strings.TrimSpace(
			out.Configuration.Recipient.Capabilities.StripeBalance.StripeTransfers.Status,
		),
		CreatedAt: out.Created,
	}, nil
}

func payoutAccountResultFromResponse(
	out stripeAccountResponse,
) (*applicationport.StripePayoutAccountResult, error) {
	id := strings.TrimSpace(out.ID)
	if !isValidStripeAccountID(id) {
		return nil, errors.New("stripe payout account: stripe account id is empty or invalid")
	}

	closed := out.Closed != nil && *out.Closed
	transferStatus := strings.ToLower(strings.TrimSpace(
		out.Configuration.Recipient.Capabilities.StripeBalance.StripeTransfers.Status,
	))

	detailsSubmitted := !hasOutstandingUserRequirements(out.Requirements)
	payoutsEnabled := !closed && transferStatus == "active"

	if closed {
		detailsSubmitted = false
	}

	return &applicationport.StripePayoutAccountResult{
		ID:               id,
		DetailsSubmitted: detailsSubmitted,
		PayoutsEnabled:   payoutsEnabled,
	}, nil
}

func hasOutstandingUserRequirements(
	requirements stripeRequirementsResponse,
) bool {
	for _, entry := range requirements.Entries {
		if !strings.EqualFold(
			strings.TrimSpace(entry.AwaitingActionFrom),
			"user",
		) {
			continue
		}

		status := strings.ToLower(strings.TrimSpace(
			entry.MinimumDeadline.Status,
		))

		switch status {
		case "":
			// Stripe says the user must act but did not provide a deadline.
			// Treat this conservatively as incomplete.
			return true

		case "currently_due", "past_due":
			return true

		case "eventually_due":
			continue

		default:
			return true
		}
	}

	return false
}

// ========================================
// HTTP
// ========================================

func (g *AccountGateway) validateReady() error {
	if g == nil {
		return errors.New("stripe account gateway is nil")
	}

	secretKey := strings.TrimSpace(g.secretKey)
	if secretKey == "" {
		return errors.New("stripe account gateway secret key is empty")
	}
	if !strings.HasPrefix(secretKey, "sk_") {
		return errors.New("stripe account gateway secret key is invalid")
	}
	if g.httpClient == nil {
		return errors.New("stripe account gateway http client is nil")
	}

	return nil
}

func (g *AccountGateway) doJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	idempotencyKey string,
	dst any,
) error {
	return g.doJSONWithBaseURL(
		ctx,
		stripeAccountsV2BaseURL,
		method,
		path,
		query,
		body,
		idempotencyKey,
		dst,
	)
}

func (g *AccountGateway) doJSONWithBaseURL(
	ctx context.Context,
	baseURL string,
	method string,
	path string,
	query url.Values,
	body any,
	idempotencyKey string,
	dst any,
) error {
	if err := g.validateReady(); err != nil {
		return err
	}

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path = "/" + strings.TrimLeft(strings.TrimSpace(path), "/")

	if baseURL == "" {
		return errors.New("stripe account: base url is empty")
	}

	requestURL := baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		requestURL,
		reader,
	)
	if err != nil {
		return err
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+strings.TrimSpace(g.secretKey),
	)
	req.Header.Set(
		"Stripe-Version",
		stripeAccountsAPIVersion,
	)
	req.Header.Set(
		"Accept",
		"application/json",
	)

	if body != nil {
		req.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	if key := strings.TrimSpace(idempotencyKey); key != "" {
		req.Header.Set(
			"Idempotency-Key",
			key,
		)
	}

	res, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return stripeAccountHTTPError(
			res.StatusCode,
			responseBody,
		)
	}

	if dst == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, dst); err != nil {
		return err
	}

	return nil
}

// ========================================
// Error
// ========================================

type stripeAccountErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func stripeAccountHTTPError(
	statusCode int,
	body []byte,
) error {
	var out stripeAccountErrorResponse

	if err := json.Unmarshal(body, &out); err == nil {
		code := strings.TrimSpace(out.Error.Code)
		message := strings.TrimSpace(out.Error.Message)

		switch {
		case code != "" && message != "":
			return fmt.Errorf(
				"stripe account http %d: %s: %s",
				statusCode,
				code,
				message,
			)

		case message != "":
			return fmt.Errorf(
				"stripe account http %d: %s",
				statusCode,
				message,
			)

		case code != "":
			return fmt.Errorf(
				"stripe account http %d: %s",
				statusCode,
				code,
			)
		}
	}

	raw := strings.TrimSpace(string(body))
	if raw == "" {
		raw = http.StatusText(statusCode)
	}

	return fmt.Errorf(
		"stripe account http %d: %s",
		statusCode,
		raw,
	)
}

// ========================================
// Helper
// ========================================

func isValidStripeAccountID(
	value string,
) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		strings.HasPrefix(value, "acct_")
}

func isValidLast4(
	value string,
) bool {
	value = strings.TrimSpace(value)

	if len(value) != 4 {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func isValidHTTPURL(
	value string,
) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	if parsed.Scheme != "https" &&
		parsed.Scheme != "http" {
		return false
	}

	return parsed.Host != ""
}
