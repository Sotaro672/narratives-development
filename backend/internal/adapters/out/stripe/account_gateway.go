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

	usecase "narratives/internal/application/usecase"
)

const (
	stripeAccountsV2BaseURL = "https://api.stripe.com/v2/core"

	// Stripe Accounts v2 の安定版APIバージョン。
	stripeAccountsAPIVersion = "2026-07-29.dahlia"

	defaultAccountCountry = "JP"
)

// ========================================
// AccountGateway
// ========================================

// AccountGateway は Stripe Connect の Connected Account を扱います。
//
// - Connected Account 作成
// - Connected Account 取得
// - Onboarding Account Link 作成
//
// Firestoreへの保存や Company / Brand の所有権検証は
// application/usecase 側の責務とします。
type AccountGateway struct {
	secretKey  string
	httpClient *http.Client
}

var _ usecase.StripeAccountGateway = (*AccountGateway)(nil)

// NewAccountGateway creates a Stripe Connected Account gateway.
func NewAccountGateway(
	secretKey string,
) *AccountGateway {
	return &AccountGateway{
		secretKey: strings.TrimSpace(
			secretKey,
		),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ========================================
// Create Account
// ========================================

// CreateAccount creates a Stripe Connected Account.
//
// AMOLでは Marketplace の受取先として利用するため、
// recipient configuration と stripe_transfers capability を要求します。
func (g *AccountGateway) CreateAccount(
	ctx context.Context,
	in usecase.CreateStripeAccountInput,
) (*usecase.StripeAccountResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	companyID := strings.TrimSpace(
		in.CompanyID,
	)
	brandID := strings.TrimSpace(
		in.BrandID,
	)
	displayName := strings.TrimSpace(
		in.DisplayName,
	)
	contactEmail := strings.TrimSpace(
		in.ContactEmail,
	)
	country := strings.ToUpper(
		strings.TrimSpace(
			in.Country,
		),
	)

	if companyID == "" {
		return nil, errors.New(
			"stripe account: companyId is empty",
		)
	}

	if brandID == "" {
		return nil, errors.New(
			"stripe account: brandId is empty",
		)
	}

	if displayName == "" {
		return nil, errors.New(
			"stripe account: displayName is empty",
		)
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
		Metadata: map[string]string{
			"companyId": companyID,
			"brandId":   brandID,
		},
	}

	var out stripeAccountResponse

	if err := g.doJSON(
		ctx,
		http.MethodPost,
		"/accounts",
		nil,
		reqBody,
		strings.TrimSpace(
			in.IdempotencyKey,
		),
		&out,
	); err != nil {
		return nil, err
	}

	return accountResultFromResponse(
		out,
	)
}

// ========================================
// Get Account
// ========================================

// GetAccount retrieves a Stripe Connected Account.
func (g *AccountGateway) GetAccount(
	ctx context.Context,
	stripeAccountID string,
) (*usecase.StripeAccountResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeAccountID = strings.TrimSpace(
		stripeAccountID,
	)

	if stripeAccountID == "" ||
		!strings.HasPrefix(
			stripeAccountID,
			"acct_",
		) {
		return nil, errors.New(
			"stripe account: invalid stripeAccountId",
		)
	}

	query := url.Values{}
	query.Add(
		"include[0]",
		"configuration.recipient",
	)
	query.Add(
		"include[1]",
		"identity",
	)
	query.Add(
		"include[2]",
		"requirements",
	)

	var out stripeAccountResponse

	if err := g.doJSON(
		ctx,
		http.MethodGet,
		"/accounts/"+url.PathEscape(
			stripeAccountID,
		),
		query,
		nil,
		"",
		&out,
	); err != nil {
		return nil, err
	}

	return accountResultFromResponse(
		out,
	)
}

// ========================================
// Account Link
// ========================================

// CreateOnboardingLink creates a single-use Stripe hosted onboarding URL.
func (g *AccountGateway) CreateOnboardingLink(
	ctx context.Context,
	in usecase.CreateStripeAccountLinkInput,
) (*usecase.StripeAccountLinkResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeAccountID := strings.TrimSpace(
		in.StripeAccountID,
	)
	returnURL := strings.TrimSpace(
		in.ReturnURL,
	)
	refreshURL := strings.TrimSpace(
		in.RefreshURL,
	)

	if stripeAccountID == "" ||
		!strings.HasPrefix(
			stripeAccountID,
			"acct_",
		) {
		return nil, errors.New(
			"stripe account: invalid stripeAccountId",
		)
	}

	if !isValidHTTPURL(
		returnURL,
	) {
		return nil, errors.New(
			"stripe account: invalid returnUrl",
		)
	}

	if !isValidHTTPURL(
		refreshURL,
	) {
		return nil, errors.New(
			"stripe account: invalid refreshUrl",
		)
	}

	reqBody := createAccountLinkRequest{
		Account: stripeAccountID,
		UseCase: accountLinkUseCaseRequest{
			Type: "account_onboarding",
			AccountOnboarding: accountOnboardingRequest{
				Configurations: []string{
					"recipient",
				},
				ReturnURL:  returnURL,
				RefreshURL: refreshURL,
			},
		},
	}

	var out stripeAccountLinkResponse

	if err := g.doJSON(
		ctx,
		http.MethodPost,
		"/account_links",
		nil,
		reqBody,
		strings.TrimSpace(
			in.IdempotencyKey,
		),
		&out,
	); err != nil {
		return nil, err
	}

	if strings.TrimSpace(
		out.Account,
	) == "" {
		return nil, errors.New(
			"stripe account: account link account is empty",
		)
	}

	if strings.TrimSpace(
		out.URL,
	) == "" {
		return nil, errors.New(
			"stripe account: account link url is empty",
		)
	}

	return &usecase.StripeAccountLinkResult{
		AccountID: strings.TrimSpace(
			out.Account,
		),
		URL: strings.TrimSpace(
			out.URL,
		),
		ExpiresAt: out.ExpiresAt,
	}, nil
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
	Type              string                   `json:"type"`
	AccountOnboarding accountOnboardingRequest `json:"account_onboarding"`
}

type accountOnboardingRequest struct {
	Configurations []string `json:"configurations"`
	ReturnURL      string   `json:"return_url"`
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
					StripeTransfers struct {
						Status string `json:"status"`
					} `json:"stripe_transfers"`
				} `json:"stripe_balance"`
			} `json:"capabilities"`
		} `json:"recipient"`
	} `json:"configuration"`
}

type stripeAccountLinkResponse struct {
	Object    string    `json:"object"`
	Account   string    `json:"account"`
	Created   time.Time `json:"created"`
	ExpiresAt time.Time `json:"expires_at"`
	Livemode  bool      `json:"livemode"`
	URL       string    `json:"url"`
}

// ========================================
// Result mapping
// ========================================

func accountResultFromResponse(
	out stripeAccountResponse,
) (*usecase.StripeAccountResult, error) {
	id := strings.TrimSpace(
		out.ID,
	)

	if id == "" ||
		!strings.HasPrefix(
			id,
			"acct_",
		) {
		return nil, errors.New(
			"stripe account: stripe account id is empty or invalid",
		)
	}

	closed := false
	if out.Closed != nil {
		closed = *out.Closed
	}

	return &usecase.StripeAccountResult{
		ID: id,
		DisplayName: strings.TrimSpace(
			out.DisplayName,
		),
		ContactEmail: strings.TrimSpace(
			out.ContactEmail,
		),
		Country: strings.ToUpper(
			strings.TrimSpace(
				out.Identity.Country,
			),
		),
		Dashboard: strings.TrimSpace(
			out.Dashboard,
		),
		Livemode: out.Livemode,
		Closed:   closed,
		RecipientTransferStatus: strings.TrimSpace(
			out.Configuration.
				Recipient.
				Capabilities.
				StripeBalance.
				StripeTransfers.
				Status,
		),
		CreatedAt: out.Created,
	}, nil
}

// ========================================
// HTTP
// ========================================

func (g *AccountGateway) validateReady() error {
	if g == nil {
		return errors.New(
			"stripe account gateway is nil",
		)
	}

	secretKey := strings.TrimSpace(
		g.secretKey,
	)

	if secretKey == "" {
		return errors.New(
			"stripe account gateway secret key is empty",
		)
	}

	if !strings.HasPrefix(
		secretKey,
		"sk_",
	) {
		return errors.New(
			"stripe account gateway secret key is invalid",
		)
	}

	if g.httpClient == nil {
		return errors.New(
			"stripe account gateway http client is nil",
		)
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
	if err := g.validateReady(); err != nil {
		return err
	}

	requestURL :=
		stripeAccountsV2BaseURL +
			path

	if len(query) > 0 {
		requestURL +=
			"?" +
				query.Encode()
	}

	var reader io.Reader

	if body != nil {
		payload, err := json.Marshal(
			body,
		)
		if err != nil {
			return err
		}

		reader = bytes.NewReader(
			payload,
		)
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
		"Bearer "+strings.TrimSpace(
			g.secretKey,
		),
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

	if key := strings.TrimSpace(
		idempotencyKey,
	); key != "" {
		req.Header.Set(
			"Idempotency-Key",
			key,
		)
	}

	res, err := g.httpClient.Do(
		req,
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(
		res.Body,
	)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 ||
		res.StatusCode >= 300 {
		return stripeAccountHTTPError(
			res.StatusCode,
			responseBody,
		)
	}

	if dst == nil ||
		len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(
		responseBody,
		dst,
	); err != nil {
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

	if err := json.Unmarshal(
		body,
		&out,
	); err == nil {
		code := strings.TrimSpace(
			out.Error.Code,
		)

		message := strings.TrimSpace(
			out.Error.Message,
		)

		switch {
		case code != "" &&
			message != "":
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

	raw := strings.TrimSpace(
		string(body),
	)

	if raw == "" {
		raw = http.StatusText(
			statusCode,
		)
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

func isValidHTTPURL(
	value string,
) bool {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return false
	}

	parsed, err := url.ParseRequestURI(
		value,
	)
	if err != nil {
		return false
	}

	if parsed.Scheme != "https" &&
		parsed.Scheme != "http" {
		return false
	}

	return parsed.Host != ""
}
