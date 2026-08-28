// backend/internal/adapters/in/http/mall/handler/payoutAccount_handler.go

package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	payoutdom "narratives/internal/domain/payoutAccount"
)

type PayoutAccountHandler struct {
	uc *usecase.PayoutAccountUsecase
}

func NewPayoutAccountHandler(
	uc *usecase.PayoutAccountUsecase,
) http.Handler {
	return &PayoutAccountHandler{
		uc: uc,
	}
}

type payoutAccountBankDTO struct {
	BankName string `json:"bankName,omitempty"`
	Last4    string `json:"last4,omitempty"`
}

type payoutAccountDTO struct {
	StripeAccountID  string                `json:"stripeAccountId"`
	DetailsSubmitted bool                  `json:"detailsSubmitted"`
	PayoutsEnabled   bool                  `json:"payoutsEnabled"`
	BankAccount      *payoutAccountBankDTO `json:"bankAccount,omitempty"`
}

type payoutAccountResponse struct {
	Data *payoutAccountDTO `json:"data"`
}

type payoutAccountLinkBody struct {
	ReturnURL  string `json:"returnUrl"`
	RefreshURL string `json:"refreshUrl"`
}

type payoutAccountLinkDTO struct {
	URL string `json:"url"`
}

type payoutAccountLinkResponse struct {
	Data payoutAccountLinkDTO `json:"data"`
}

func (h *PayoutAccountHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
	}

	switch {
	case r.Method == http.MethodGet &&
		path == "/me/payout-account":
		h.getMe(w, r)
		return

	case r.Method == http.MethodPost &&
		path == "/me/payout-account/account-link":
		h.createAccountLink(w, r)
		return

	default:
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}
}

func (h *PayoutAccountHandler) getMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.uc == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "payout_account_usecase_not_initialized",
			},
		)
		return
	}

	userID, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(userID) == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	account, err := h.uc.GetByUserID(
		r.Context(),
		userID,
	)
	if err != nil {
		if errors.Is(err, payoutdom.ErrNotFound) {
			writeJSON(
				w,
				http.StatusOK,
				payoutAccountResponse{
					Data: nil,
				},
			)
			return
		}

		writePayoutAccountError(w, err)
		return
	}

	if account == nil {
		writeJSON(
			w,
			http.StatusOK,
			payoutAccountResponse{
				Data: nil,
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		payoutAccountResponse{
			Data: payoutAccountToDTO(*account),
		},
	)
}

func (h *PayoutAccountHandler) createAccountLink(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.uc == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "payout_account_usecase_not_initialized",
			},
		)
		return
	}

	userID, email, ok :=
		middleware.CurrentUserUIDAndEmail(r)

	if !ok || strings.TrimSpace(userID) == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	var body payoutAccountLinkBody

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_json",
			},
		)
		return
	}

	body.ReturnURL = strings.TrimSpace(body.ReturnURL)
	body.RefreshURL = strings.TrimSpace(body.RefreshURL)

	if body.ReturnURL == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "returnUrl is required",
			},
		)
		return
	}

	if body.RefreshURL == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "refreshUrl is required",
			},
		)
		return
	}

	displayName := ""

	if fullName, exists :=
		middleware.CurrentUserFullName(r); exists {
		displayName = strings.TrimSpace(fullName)
	}

	result, err := h.uc.CreateAccountLink(
		r.Context(),
		usecase.CreatePayoutAccountLinkInput{
			UserID:       strings.TrimSpace(userID),
			DisplayName:  displayName,
			ContactEmail: strings.TrimSpace(email),
			ReturnURL:    body.ReturnURL,
			RefreshURL:   body.RefreshURL,
		},
	)
	if err != nil {
		writePayoutAccountError(w, err)
		return
	}

	url := strings.TrimSpace(result.OnboardingURL)
	if url == "" {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "stripe_account_link_url_empty",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		payoutAccountLinkResponse{
			Data: payoutAccountLinkDTO{
				URL: url,
			},
		},
	)
}

func payoutAccountToDTO(
	account payoutdom.PayoutAccount,
) *payoutAccountDTO {
	dto := &payoutAccountDTO{
		StripeAccountID: strings.TrimSpace(
			account.StripeAccountID,
		),
		DetailsSubmitted: account.DetailsSubmitted,
		PayoutsEnabled:   account.PayoutsEnabled,
		BankAccount:      nil,
	}

	bankName := strings.TrimSpace(account.BankName)
	bankLast4 := strings.TrimSpace(account.BankLast4)

	if bankName != "" || bankLast4 != "" {
		dto.BankAccount = &payoutAccountBankDTO{
			BankName: bankName,
			Last4:    bankLast4,
		}
	}

	return dto
}

func writePayoutAccountError(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "unknown",
			},
		)
		return
	}

	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(
		err,
		payoutdom.ErrInvalidUserID,
	),
		errors.Is(
			err,
			payoutdom.ErrInvalidStripeAccountID,
		),
		errors.Is(
			err,
			payoutdom.ErrInvalidBankName,
		),
		errors.Is(
			err,
			payoutdom.ErrInvalidBankLast4,
		),
		errors.Is(
			err,
			payoutdom.ErrInvalidCreatedAt,
		),
		errors.Is(
			err,
			payoutdom.ErrInvalidUpdatedAt,
		),
		errors.Is(
			err,
			usecase.ErrPayoutAccountInvalidReturnURL,
		),
		errors.Is(
			err,
			usecase.ErrPayoutAccountInvalidRefreshURL,
		):
		statusCode = http.StatusBadRequest

	case errors.Is(
		err,
		payoutdom.ErrNotFound,
	):
		statusCode = http.StatusNotFound

	case errors.Is(
		err,
		payoutdom.ErrConflict,
	):
		statusCode = http.StatusConflict

	case errors.Is(
		err,
		usecase.ErrPayoutAccountRepositoryMissing,
	),
		errors.Is(
			err,
			usecase.ErrPayoutAccountStripeGatewayMissing,
		),
		errors.Is(
			err,
			usecase.ErrPayoutAccountAllowedReturnOriginMissing,
		):
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(
		w,
		statusCode,
		map[string]string{
			"error": err.Error(),
		},
	)
}
