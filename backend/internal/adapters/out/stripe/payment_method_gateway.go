// backend/internal/adapters/out/stripe/payment_method_gateway.go
package stripe

import (
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
	pm "narratives/internal/domain/paymentMethod"
)

const (
	stripeAPIBaseURL                  = "https://api.stripe.com/v1"
	developmentTestPaymentMethodToken = "tok_visa"
)

type PaymentMethodCustomerStore interface {
	GetStripeCustomerIDByUser(ctx context.Context, userID string) (string, error)
	SaveStripeCustomerIDByUser(ctx context.Context, userID string, stripeCustomerID string) error
}

type PaymentMethodGateway struct {
	secretKey     string
	customerStore PaymentMethodCustomerStore
	httpClient    *http.Client
}

var _ applicationport.StripePaymentMethodGateway = (*PaymentMethodGateway)(nil)
var _ applicationport.StripePaymentIntentGateway = (*PaymentMethodGateway)(nil)

func NewPaymentMethodGateway(
	secretKey string,
	customerStore PaymentMethodCustomerStore,
) *PaymentMethodGateway {
	return &PaymentMethodGateway{
		secretKey:     strings.TrimSpace(secretKey),
		customerStore: customerStore,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (g *PaymentMethodGateway) GetOrCreateCustomer(
	ctx context.Context,
	userID string,
	cardholderName string,
) (string, error) {
	if err := g.validateReady(); err != nil {
		return "", err
	}

	userID = strings.TrimSpace(userID)
	cardholderName = strings.TrimSpace(cardholderName)

	if userID == "" {
		return "", pm.ErrInvalidUserID
	}

	existing, err := g.customerStore.GetStripeCustomerIDByUser(ctx, userID)
	if err == nil && strings.TrimSpace(existing) != "" {
		return strings.TrimSpace(existing), nil
	}
	if err != nil && !errors.Is(err, pm.ErrNotFound) {
		return "", err
	}

	form := url.Values{}
	if cardholderName != "" {
		form.Set("name", cardholderName)
	}
	form.Set("metadata[userId]", userID)

	var out stripeCustomerResponse
	if err := g.postForm(ctx, "/customers", form, &out); err != nil {
		return "", err
	}

	stripeCustomerID := strings.TrimSpace(out.ID)
	if stripeCustomerID == "" {
		return "", errors.New("stripe customer id is empty")
	}

	if err := g.customerStore.SaveStripeCustomerIDByUser(
		ctx,
		userID,
		stripeCustomerID,
	); err != nil {
		return "", err
	}

	return stripeCustomerID, nil
}

func (g *PaymentMethodGateway) CreateSetupIntent(
	ctx context.Context,
	stripeCustomerID string,
	cardholderName string,
) (string, error) {
	if err := g.validateReady(); err != nil {
		return "", err
	}

	stripeCustomerID = strings.TrimSpace(stripeCustomerID)
	cardholderName = strings.TrimSpace(cardholderName)

	if stripeCustomerID == "" {
		return "", pm.ErrInvalidStripeCustomerID
	}

	form := url.Values{}
	form.Set("customer", stripeCustomerID)
	form.Add("payment_method_types[]", "card")
	form.Set("usage", "off_session")

	if cardholderName != "" {
		form.Set("metadata[cardholderName]", cardholderName)
	}

	var out stripeSetupIntentResponse
	if err := g.postForm(ctx, "/setup_intents", form, &out); err != nil {
		return "", err
	}

	clientSecret := strings.TrimSpace(out.ClientSecret)
	if clientSecret == "" {
		return "", errors.New("stripe setup intent client_secret is empty")
	}

	return clientSecret, nil
}

func (g *PaymentMethodGateway) CreateDevelopmentTestPaymentMethod(
	ctx context.Context,
	userID string,
	cardholderName string,
) (*applicationport.DevelopmentTestPaymentMethodResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(
		strings.TrimSpace(g.secretKey),
		"sk_test_",
	) {
		return nil, errors.New(
			"stripe development test payment method requires sk_test_ secret key",
		)
	}

	userID = strings.TrimSpace(userID)
	cardholderName = strings.TrimSpace(cardholderName)

	if userID == "" {
		return nil, pm.ErrInvalidUserID
	}

	if cardholderName == "" {
		return nil, pm.ErrInvalidCardholderName
	}

	stripeCustomerID, err := g.GetOrCreateCustomer(
		ctx,
		userID,
		cardholderName,
	)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("type", "card")
	form.Set("card[token]", developmentTestPaymentMethodToken)
	form.Set("billing_details[name]", cardholderName)
	form.Set("metadata[userId]", userID)

	var created stripePaymentMethodResponse
	if err := g.postForm(
		ctx,
		"/payment_methods",
		form,
		&created,
	); err != nil {
		return nil, err
	}

	stripePaymentMethodID := strings.TrimSpace(created.ID)
	if stripePaymentMethodID == "" {
		return nil, errors.New(
			"stripe development test payment method id is empty",
		)
	}

	attachForm := url.Values{}
	attachForm.Set("customer", stripeCustomerID)

	var attached stripePaymentMethodResponse
	if err := g.postForm(
		ctx,
		"/payment_methods/"+url.PathEscape(stripePaymentMethodID)+"/attach",
		attachForm,
		&attached,
	); err != nil {
		return nil, err
	}

	if strings.TrimSpace(attached.ID) != "" {
		created = attached
	}

	brand := strings.TrimSpace(created.Card.Brand)
	last4 := strings.TrimSpace(created.Card.Last4)

	if brand == "" {
		return nil, pm.ErrInvalidBrand
	}

	if last4 == "" {
		return nil, pm.ErrInvalidLast4
	}

	if created.Card.ExpMonth <= 0 {
		return nil, pm.ErrInvalidExpMonth
	}

	if created.Card.ExpYear <= 0 {
		return nil, pm.ErrInvalidExpYear
	}

	return &applicationport.DevelopmentTestPaymentMethodResult{
		StripeCustomerID:      stripeCustomerID,
		StripePaymentMethodID: stripePaymentMethodID,
		Brand:                 brand,
		Last4:                 last4,
		ExpMonth:              created.Card.ExpMonth,
		ExpYear:               created.Card.ExpYear,
	}, nil
}

// CreateAndConfirmPaymentIntent creates and confirms a Stripe PaymentIntent.
//
// Mallからの通常決済ではoff_session=false、
// Consoleの発送時決済ではoff_session=trueとして実行する。
// frontend / Console はsecret keyを持たず、backendがStripe secret keyでAPIを呼ぶ。
//
// Separate Charges and Transfers方式を使用するため、
// PaymentIntentにはOrder単位のtransfer_groupを付与する。
// 後続のStripe Transferも同じtransfer_groupを使用する。
func (g *PaymentMethodGateway) CreateAndConfirmPaymentIntent(
	ctx context.Context,
	in applicationport.CreateAndConfirmPaymentIntentInput,
) (*applicationport.CreateAndConfirmPaymentIntentResult, error) {
	if err := g.validateReady(); err != nil {
		return nil, err
	}

	stripeCustomerID := strings.TrimSpace(in.StripeCustomerID)
	stripePaymentMethodID := strings.TrimSpace(in.StripePaymentMethodID)
	currency := strings.TrimSpace(strings.ToLower(in.Currency))
	paymentMethodID := strings.TrimSpace(in.PaymentMethodID)
	transferGroup := strings.TrimSpace(in.TransferGroup)

	if stripeCustomerID == "" {
		return nil, pm.ErrInvalidStripeCustomerID
	}
	if stripePaymentMethodID == "" {
		return nil, pm.ErrInvalidStripePaymentMethod
	}
	if in.Amount <= 0 {
		return nil, errors.New("stripe payment intent amount is invalid")
	}
	if transferGroup == "" {
		return nil, errors.New("stripe payment intent transfer_group is empty")
	}
	if currency == "" {
		currency = "jpy"
	}

	form := url.Values{}
	form.Set("amount", fmt.Sprintf("%d", in.Amount))
	form.Set("currency", currency)
	form.Set("customer", stripeCustomerID)
	form.Set("payment_method", stripePaymentMethodID)
	form.Set("confirm", "true")
	form.Set("transfer_group", transferGroup)
	form.Add("payment_method_types[]", "card")

	if in.OffSession {
		form.Set("off_session", "true")
	} else {
		form.Set("off_session", "false")
	}

	if desc := strings.TrimSpace(in.Description); desc != "" {
		form.Set("description", desc)
	}

	if paymentMethodID != "" {
		form.Set("metadata[paymentMethodId]", paymentMethodID)
	}

	form.Set("metadata[transferGroup]", transferGroup)

	var out stripePaymentIntentResponse
	requestErr := g.postFormWithIdempotencyKey(
		ctx,
		"/payment_intents",
		form,
		strings.TrimSpace(in.IdempotencyKey),
		&out,
	)

	status := strings.TrimSpace(out.Status)
	clientSecret := strings.TrimSpace(out.ClientSecret)
	paymentIntentID := strings.TrimSpace(out.ID)
	stripeChargeID := strings.TrimSpace(out.LatestCharge)

	if requestErr != nil && paymentIntentID == "" {
		return nil, requestErr
	}

	result := &applicationport.CreateAndConfirmPaymentIntentResult{
		StripePaymentIntentID: paymentIntentID,
		StripeChargeID:        stripeChargeID,
		Status:                status,
		ClientSecret:          clientSecret,
		RequiresAction:        status == "requires_action" || status == "requires_source_action",
	}

	if out.LastPaymentError != nil {
		result.ErrorType = strings.TrimSpace(out.LastPaymentError.Type)
		result.ErrorCode = strings.TrimSpace(out.LastPaymentError.Code)
		result.ErrorMessage = strings.TrimSpace(out.LastPaymentError.Message)
	}

	if requestErr != nil {
		if result.ErrorMessage == "" {
			result.ErrorMessage = requestErr.Error()
		}

		return result, requestErr
	}

	return result, nil
}

func (g *PaymentMethodGateway) validateReady() error {
	if g == nil {
		return errors.New("stripe payment method gateway is nil")
	}

	secretKey := strings.TrimSpace(g.secretKey)
	if secretKey == "" {
		return errors.New("stripe secret key is empty")
	}
	if !strings.HasPrefix(secretKey, "sk_") {
		return errors.New("stripe secret key is invalid")
	}

	if g.customerStore == nil {
		return errors.New("stripe customer store is nil")
	}

	if g.httpClient == nil {
		return errors.New("stripe http client is nil")
	}

	return nil
}

func (g *PaymentMethodGateway) postForm(
	ctx context.Context,
	path string,
	form url.Values,
	dst any,
) error {
	return g.postFormWithIdempotencyKey(ctx, path, form, "", dst)
}

func (g *PaymentMethodGateway) postFormWithIdempotencyKey(
	ctx context.Context,
	path string,
	form url.Values,
	idempotencyKey string,
	dst any,
) error {
	if err := g.validateReady(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		stripeAPIBaseURL+path,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(g.secretKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	if key := strings.TrimSpace(idempotencyKey); key != "" {
		req.Header.Set("Idempotency-Key", key)
	}

	res, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var serr stripeErrorResponse

		if json.Unmarshal(body, &serr) == nil {
			if target, ok := dst.(*stripePaymentIntentResponse); ok &&
				target != nil {
				if serr.Error.PaymentIntent != nil {
					*target = *serr.Error.PaymentIntent
				}

				if target.LastPaymentError == nil &&
					(strings.TrimSpace(serr.Error.Type) != "" ||
						strings.TrimSpace(serr.Error.Code) != "" ||
						strings.TrimSpace(serr.Error.Message) != "") {
					target.LastPaymentError =
						&stripePaymentError{
							Type:    strings.TrimSpace(serr.Error.Type),
							Code:    strings.TrimSpace(serr.Error.Code),
							Message: strings.TrimSpace(serr.Error.Message),
						}
				}
			}

			if serr.Error.Message != "" {
				return fmt.Errorf(
					"stripe http %d: %s",
					res.StatusCode,
					serr.Error.Message,
				)
			}
		}

		return fmt.Errorf("stripe http %d: %s", res.StatusCode, string(body))
	}

	if dst == nil {
		return nil
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return err
	}

	return nil
}

type stripeCustomerResponse struct {
	ID string `json:"id"`
}

type stripeSetupIntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
}

type stripePaymentError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type stripePaymentIntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	Status       string `json:"status"`
	LatestCharge string `json:"latest_charge"`

	LastPaymentError *stripePaymentError `json:"last_payment_error"`
}

type stripePaymentMethodResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Card struct {
		Brand    string `json:"brand"`
		Last4    string `json:"last4"`
		ExpMonth int    `json:"exp_month"`
		ExpYear  int    `json:"exp_year"`
	} `json:"card"`
}

type stripeErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`

		PaymentIntent *stripePaymentIntentResponse `json:"payment_intent"`
	} `json:"error"`
}
