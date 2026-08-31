// backend/internal/adapters/in/http/mall/handler/payoutAccount_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	payoutdom "narratives/internal/domain/payoutAccount"
)

const maxPayoutAccountBodyBytes int64 = 64 << 10 // 64 KiB

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
	BankAccount *payoutAccountBankDTO `json:"bankAccount"`
}

type payoutAccountResponse struct {
	Data *payoutAccountDTO `json:"data"`
}

type payoutAccountRegistrationRequest struct {
	BankCode          string                    `json:"bankCode"`
	BankName          string                    `json:"bankName"`
	BranchCode        string                    `json:"branchCode"`
	BranchName        string                    `json:"branchName"`
	AccountType       payoutdom.BankAccountType `json:"accountType"`
	AccountNumber     string                    `json:"accountNumber"`
	AccountHolderName string                    `json:"accountHolderName"`
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

	case r.Method == http.MethodPut &&
		path == "/me/payout-account":
		h.putMe(w, r)
		return

	case path == "/me/payout-account":
		w.Header().Set("Allow", "GET, PUT, OPTIONS")
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

func (h *PayoutAccountHandler) putMe(
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

	var request payoutAccountRegistrationRequest
	if err := decodePayoutAccountJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid_json",
		})
		return
	}

	account, err := h.uc.Register(
		r.Context(),
		usecase.RegisterPayoutAccountInput{
			UserID:            userID,
			BankCode:          request.BankCode,
			BankName:          request.BankName,
			BranchCode:        request.BranchCode,
			BranchName:        request.BranchName,
			AccountType:       request.AccountType,
			AccountNumber:     request.AccountNumber,
			AccountHolderName: request.AccountHolderName,
		},
	)
	if err != nil {
		writePayoutAccountError(w, err)
		return
	}

	if account == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "payout_account_registration_result_empty",
		})
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, payoutAccountResponse{
		Data: payoutAccountToDTO(*account),
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

	if ok &&
		userID != "" &&
		!strings.ContainsAny(userID, " \t\r\n") {
		return userID, true
	}

	writeJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "unauthorized",
	})

	return "", false
}

func decodePayoutAccountJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPayoutAccountBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var trailing any
	err := decoder.Decode(&trailing)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New(
				"request body must contain exactly one JSON value",
			)
		}

		return err
	}

	return nil
}

func payoutAccountToDTO(
	account payoutdom.PayoutAccount,
) *payoutAccountDTO {
	return &payoutAccountDTO{
		BankAccount: &payoutAccountBankDTO{
			BankCode:          account.BankCode,
			BankName:          account.BankName,
			BranchCode:        account.BranchCode,
			BranchName:        account.BranchName,
			AccountType:       account.AccountType,
			Last4:             account.BankLast4,
			AccountHolderName: account.AccountHolderName,
		},
	}
}

func isInvalidPayoutAccountError(err error) bool {
	return errors.Is(err, payoutdom.ErrInvalidUserID) ||
		errors.Is(err, payoutdom.ErrInvalidBankCode) ||
		errors.Is(err, payoutdom.ErrInvalidBankName) ||
		errors.Is(err, payoutdom.ErrInvalidBranchCode) ||
		errors.Is(err, payoutdom.ErrInvalidBranchName) ||
		errors.Is(err, payoutdom.ErrInvalidBankAccountType) ||
		errors.Is(err, payoutdom.ErrInvalidAccountNumberCiphertext) ||
		errors.Is(err, payoutdom.ErrInvalidBankLast4) ||
		errors.Is(err, payoutdom.ErrInvalidAccountHolderName) ||
		errors.Is(err, payoutdom.ErrInvalidCreatedAt) ||
		errors.Is(err, payoutdom.ErrInvalidUpdatedAt) ||
		errors.Is(err, usecase.ErrPayoutAccountInvalidAccountNumber)
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
		errors.Is(err, usecase.ErrPayoutAccountOwnershipMismatch):
		statusCode = http.StatusConflict

	case errors.Is(err, usecase.ErrPayoutAccountEncryptionFailed):
		statusCode = http.StatusBadGateway

	case errors.Is(err, usecase.ErrPayoutAccountRepositoryMissing),
		errors.Is(err, usecase.ErrPayoutAccountCipherMissing),
		errors.Is(err, usecase.ErrPayoutAccountClockMissing):
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, map[string]string{
		"error": err.Error(),
	})
}
