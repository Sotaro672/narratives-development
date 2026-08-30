// backend/internal/adapters/in/http/mall/handler/payoutAccount_handler.go
package mallHandler

import (
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
	BankCode          string                    `json:"bankCode"`
	BankName          string                    `json:"bankName"`
	BranchCode        string                    `json:"branchCode"`
	BranchName        string                    `json:"branchName"`
	AccountType       payoutdom.BankAccountType `json:"accountType"`
	Last4             string                    `json:"last4"`
	AccountHolderName string                    `json:"accountHolderName"`
}

type payoutAccountDTO struct {
	Status      payoutdom.Status      `json:"status"`
	PayoutReady bool                  `json:"payoutReady"`
	BankAccount *payoutAccountBankDTO `json:"bankAccount,omitempty"`
}

type payoutAccountResponse struct {
	Data *payoutAccountDTO `json:"data"`
}

type payoutAccountSessionDTO struct {
	ClientSecret string `json:"clientSecret"`
}

type payoutAccountSessionResponse struct {
	Data *payoutAccountSessionDTO `json:"data"`
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
		path == "/me/payout-account/session":
		h.createSession(w, r)
		return

	case path == "/me/payout-account":
		w.Header().Set("Allow", "GET, OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method_not_allowed",
		})
		return

	case path == "/me/payout-account/session":
		w.Header().Set("Allow", "POST, OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method_not_allowed",
		})
		return

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})
		return
	}
}

func (h *PayoutAccountHandler) getMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !h.requireUsecase(w) {
		return
	}

	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	account, err := h.uc.GetByUserID(
		r.Context(),
		userID,
	)
	if err != nil {
		if errors.Is(err, payoutdom.ErrNotFound) {
			w.Header().Set("Cache-Control", "no-store")
			writeJSON(w, http.StatusOK, payoutAccountResponse{
				Data: nil,
			})
			return
		}

		writePayoutAccountError(w, err)
		return
	}

	if account == nil {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, payoutAccountResponse{
			Data: nil,
		})
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, payoutAccountResponse{
		Data: payoutAccountToDTO(*account),
	})
}

func (h *PayoutAccountHandler) createSession(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !h.requireUsecase(w) {
		return
	}

	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	session, err := h.uc.CreateAccountSession(
		r.Context(),
		userID,
	)
	if err != nil {
		writePayoutAccountError(w, err)
		return
	}

	if session == nil ||
		strings.TrimSpace(session.ClientSecret) == "" {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "payout_account_session_result_empty",
		})
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, payoutAccountSessionResponse{
		Data: &payoutAccountSessionDTO{
			ClientSecret: session.ClientSecret,
		},
	})
}

func (h *PayoutAccountHandler) requireUsecase(
	w http.ResponseWriter,
) bool {
	if h != nil && h.uc != nil {
		return true
	}

	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error": "payout_account_usecase_not_initialized",
	})

	return false
}

func (h *PayoutAccountHandler) requireUserID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	userID, ok := middleware.CurrentUserUID(r)
	userID = strings.TrimSpace(userID)

	if ok && userID != "" {
		return userID, true
	}

	writeJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "unauthorized",
	})

	return "", false
}

func payoutAccountToDTO(
	account payoutdom.PayoutAccount,
) *payoutAccountDTO {
	var bankAccount *payoutAccountBankDTO

	if hasPayoutAccountBankSnapshot(account) {
		bankAccount = &payoutAccountBankDTO{
			BankCode:          account.BankCode,
			BankName:          account.BankName,
			BranchCode:        account.BranchCode,
			BranchName:        account.BranchName,
			AccountType:       account.AccountType,
			Last4:             account.BankLast4,
			AccountHolderName: account.AccountHolderName,
		}
	}

	return &payoutAccountDTO{
		Status:      account.Status,
		PayoutReady: account.PayoutReady,
		BankAccount: bankAccount,
	}
}

func hasPayoutAccountBankSnapshot(
	account payoutdom.PayoutAccount,
) bool {
	return strings.TrimSpace(account.BankCode) != "" ||
		strings.TrimSpace(account.BankName) != "" ||
		strings.TrimSpace(account.BranchCode) != "" ||
		strings.TrimSpace(account.BranchName) != "" ||
		account.AccountType != "" ||
		strings.TrimSpace(account.BankLast4) != "" ||
		strings.TrimSpace(account.AccountHolderName) != ""
}

func isInvalidPayoutAccountError(err error) bool {
	return errors.Is(err, payoutdom.ErrInvalidUserID) ||
		errors.Is(err, payoutdom.ErrInvalidProvider) ||
		errors.Is(err, payoutdom.ErrInvalidProviderAccountID) ||
		errors.Is(err, payoutdom.ErrInvalidStatus) ||
		errors.Is(err, payoutdom.ErrInvalidPayoutReady) ||
		errors.Is(err, payoutdom.ErrInvalidBankCode) ||
		errors.Is(err, payoutdom.ErrInvalidBankName) ||
		errors.Is(err, payoutdom.ErrInvalidBranchCode) ||
		errors.Is(err, payoutdom.ErrInvalidBranchName) ||
		errors.Is(err, payoutdom.ErrInvalidBankAccountType) ||
		errors.Is(err, payoutdom.ErrInvalidBankLast4) ||
		errors.Is(err, payoutdom.ErrInvalidAccountHolderName) ||
		errors.Is(err, payoutdom.ErrInvalidCreatedAt) ||
		errors.Is(err, payoutdom.ErrInvalidUpdatedAt)
}

func writePayoutAccountError(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "unknown",
		})
		return
	}

	statusCode := http.StatusInternalServerError

	switch {
	case isInvalidPayoutAccountError(err):
		statusCode = http.StatusBadRequest

	case errors.Is(err, payoutdom.ErrNotFound):
		statusCode = http.StatusNotFound

	case errors.Is(err, payoutdom.ErrConflict),
		errors.Is(err, usecase.ErrPayoutAccountProviderMismatch):
		statusCode = http.StatusConflict

	case errors.Is(err, usecase.ErrPayoutAccountDirectRegistrationDisabled):
		statusCode = http.StatusMethodNotAllowed

	case errors.Is(err, usecase.ErrPayoutAccountStripeResultEmpty),
		errors.Is(err, usecase.ErrPayoutAccountSessionResultEmpty),
		errors.Is(err, usecase.ErrPayoutAccountStripeAccountMismatch):
		statusCode = http.StatusBadGateway

	case errors.Is(err, usecase.ErrPayoutAccountRepositoryMissing),
		errors.Is(err, usecase.ErrPayoutAccountStripeGatewayMissing),
		errors.Is(err, usecase.ErrPayoutAccountAvatarRepositoryMissing),
		errors.Is(err, usecase.ErrPayoutAccountAuthUserReaderMissing):
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, map[string]string{
		"error": err.Error(),
	})
}
