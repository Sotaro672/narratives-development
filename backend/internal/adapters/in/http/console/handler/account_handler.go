// backend/internal/adapters/in/http/console/handler/account_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	uc "narratives/internal/application/usecase"
	accountdom "narratives/internal/domain/account"
)

// AccountHandler は /accounts 関連のエンドポイントを担当します。
type AccountHandler struct {
	uc *uc.AccountUsecase
}

// NewAccountHandler はHTTPハンドラを初期化します。
func NewAccountHandler(
	accountUC *uc.AccountUsecase,
) http.Handler {
	return &AccountHandler{
		uc: accountUC,
	}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *AccountHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	path := strings.TrimSuffix(
		r.URL.Path,
		"/",
	)

	switch {
	case r.Method == http.MethodGet &&
		path == "/accounts":
		h.list(w, r)

	case r.Method == http.MethodPost &&
		path == "/accounts":
		h.create(w, r)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			path,
			"/accounts/brand/",
		):
		brandID := strings.TrimPrefix(
			path,
			"/accounts/brand/",
		)
		h.getByBrandID(
			w,
			r,
			brandID,
		)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(
			path,
			"/accounts/",
		):
		id := strings.TrimPrefix(
			path,
			"/accounts/",
		)
		h.get(
			w,
			r,
			id,
		)

	case r.Method == http.MethodPatch &&
		strings.HasPrefix(
			path,
			"/accounts/",
		):
		id := strings.TrimPrefix(
			path,
			"/accounts/",
		)
		h.update(
			w,
			r,
			id,
		)

	default:
		w.WriteHeader(
			http.StatusNotFound,
		)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_found",
			},
		)
	}
}

// ========================================
// Request
// ========================================

type createAccountRequest struct {
	BrandID         string                   `json:"brandId"`
	StripeAccountID string                   `json:"stripeAccountId"`
	MemberID        string                   `json:"memberId"`
	BankName        string                   `json:"bankName"`
	BranchName      string                   `json:"branchName"`
	AccountNumber   int                      `json:"accountNumber"`
	AccountType     accountdom.AccountType   `json:"accountType"`
	Currency        string                   `json:"currency"`
	Status          accountdom.AccountStatus `json:"status"`
}

type updateAccountRequest struct {
	BrandID         *string                   `json:"brandId"`
	StripeAccountID *string                   `json:"stripeAccountId"`
	MemberID        *string                   `json:"memberId"`
	BankName        *string                   `json:"bankName"`
	BranchName      *string                   `json:"branchName"`
	AccountNumber   *int                      `json:"accountNumber"`
	AccountType     *accountdom.AccountType   `json:"accountType"`
	Currency        *string                   `json:"currency"`
	Status          *accountdom.AccountStatus `json:"status"`
}

// ========================================
// GET /accounts
// ========================================

func (h *AccountHandler) list(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h.uc == nil {
		writeAccountErr(
			w,
			errors.New(
				"account: usecase is nil",
			),
		)
		return
	}

	accounts, err := h.uc.ListByCompanyID(
		r.Context(),
	)
	if err != nil {
		writeAccountErr(
			w,
			err,
		)
		return
	}

	if accounts == nil {
		accounts = []accountdom.Account{}
	}

	w.WriteHeader(
		http.StatusOK,
	)
	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"items": accounts,
		},
	)
}

// ========================================
// GET /accounts/{id}
// ========================================

func (h *AccountHandler) get(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if h.uc == nil {
		writeAccountErr(
			w,
			errors.New(
				"account: usecase is nil",
			),
		)
		return
	}

	id = strings.TrimSpace(id)
	if id == "" ||
		strings.Contains(
			id,
			"/",
		) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidID,
		)
		return
	}

	account, err := h.uc.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		writeAccountErr(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusOK,
	)
	_ = json.NewEncoder(w).Encode(
		account,
	)
}

// ========================================
// GET /accounts/brand/{brandId}
// ========================================

func (h *AccountHandler) getByBrandID(
	w http.ResponseWriter,
	r *http.Request,
	brandID string,
) {
	if h.uc == nil {
		writeAccountErr(
			w,
			errors.New(
				"account: usecase is nil",
			),
		)
		return
	}

	brandID = strings.TrimSpace(
		brandID,
	)
	if brandID == "" ||
		strings.Contains(
			brandID,
			"/",
		) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidBrandID,
		)
		return
	}

	account, err := h.uc.GetByBrandID(
		r.Context(),
		brandID,
	)
	if err != nil {
		writeAccountErr(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusOK,
	)
	_ = json.NewEncoder(w).Encode(
		account,
	)
}

// ========================================
// POST /accounts
// ========================================

func (h *AccountHandler) create(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h.uc == nil {
		writeAccountErr(
			w,
			errors.New(
				"account: usecase is nil",
			),
		)
		return
	}

	var req createAccountRequest
	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&req,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_request",
		)
		return
	}

	companyID := uc.CompanyIDFromContext(
		r.Context(),
	)
	if companyID == "" {
		writeAccountErr(
			w,
			accountdom.ErrInvalidCompanyID,
		)
		return
	}

	req.BrandID = strings.TrimSpace(
		req.BrandID,
	)
	if req.BrandID == "" {
		writeAccountErr(
			w,
			accountdom.ErrInvalidBrandID,
		)
		return
	}

	req.StripeAccountID = strings.TrimSpace(
		req.StripeAccountID,
	)
	if req.StripeAccountID == "" ||
		!strings.HasPrefix(
			req.StripeAccountID,
			"acct_",
		) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidStripeAccountID,
		)
		return
	}

	req.MemberID = strings.TrimSpace(
		req.MemberID,
	)
	if req.MemberID == "" {
		req.MemberID = uc.MemberIDFromContext(
			r.Context(),
		)
	}
	if req.MemberID == "" {
		writeAccountErr(
			w,
			accountdom.ErrInvalidMemberID,
		)
		return
	}

	if req.Currency == "" {
		req.Currency =
			accountdom.DefaultCurrency
	}

	if req.Status == "" {
		req.Status =
			accountdom.StatusInactive
	}

	now := time.Now().UTC()

	accountID :=
		accountdom.AccountIDPrefix +
			uuid.NewString()

	account, err := accountdom.NewWithNow(
		accountID,
		companyID,
		req.BrandID,
		req.StripeAccountID,
		req.MemberID,
		req.BankName,
		req.BranchName,
		req.AccountNumber,
		req.AccountType,
		req.Currency,
		req.Status,
		now,
	)
	if err != nil {
		writeAccountErr(
			w,
			err,
		)
		return
	}

	memberID := uc.MemberIDFromContext(
		r.Context(),
	)
	if memberID != "" {
		account.CreatedBy =
			&memberID
		account.UpdatedBy =
			&memberID
	}

	created, err := h.uc.Create(
		r.Context(),
		account,
	)
	if err != nil {
		writeAccountErr(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusCreated,
	)
	_ = json.NewEncoder(w).Encode(
		created,
	)
}

// ========================================
// PATCH /accounts/{id}
// ========================================

func (h *AccountHandler) update(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	if h.uc == nil {
		writeAccountErr(
			w,
			errors.New(
				"account: usecase is nil",
			),
		)
		return
	}

	id = strings.TrimSpace(id)
	if id == "" ||
		strings.Contains(
			id,
			"/",
		) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidID,
		)
		return
	}

	var req updateAccountRequest
	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&req,
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_request",
		)
		return
	}

	if req.BrandID != nil {
		value := strings.TrimSpace(
			*req.BrandID,
		)
		if value == "" {
			writeAccountErr(
				w,
				accountdom.ErrInvalidBrandID,
			)
			return
		}

		req.BrandID = &value
	}

	if req.StripeAccountID != nil {
		value := strings.TrimSpace(
			*req.StripeAccountID,
		)
		if value == "" ||
			!strings.HasPrefix(
				value,
				"acct_",
			) {
			writeAccountErr(
				w,
				accountdom.ErrInvalidStripeAccountID,
			)
			return
		}

		req.StripeAccountID = &value
	}

	if req.MemberID != nil {
		value := strings.TrimSpace(
			*req.MemberID,
		)
		if value == "" {
			writeAccountErr(
				w,
				accountdom.ErrInvalidMemberID,
			)
			return
		}

		req.MemberID = &value
	}

	if req.AccountNumber != nil &&
		(*req.AccountNumber <
			accountdom.MinAccountNumber ||
			*req.AccountNumber >
				accountdom.MaxAccountNumber) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidAccountNumber,
		)
		return
	}

	if req.AccountType != nil &&
		*req.AccountType != "" &&
		!accountdom.IsValidAccountType(
			*req.AccountType,
		) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidAccountType,
		)
		return
	}

	if req.Currency != nil {
		value := strings.TrimSpace(
			*req.Currency,
		)
		if value == "" {
			writeAccountErr(
				w,
				accountdom.ErrInvalidCurrency,
			)
			return
		}

		req.Currency = &value
	}

	if req.Status != nil &&
		!accountdom.IsValidStatus(
			*req.Status,
		) {
		writeAccountErr(
			w,
			accountdom.ErrInvalidStatus,
		)
		return
	}

	patch := accountdom.AccountPatch{
		BrandID:         req.BrandID,
		StripeAccountID: req.StripeAccountID,
		MemberID:        req.MemberID,
		BankName:        req.BankName,
		BranchName:      req.BranchName,
		AccountNumber:   req.AccountNumber,
		AccountType:     req.AccountType,
		Currency:        req.Currency,
		Status:          req.Status,
	}

	memberID := uc.MemberIDFromContext(
		r.Context(),
	)
	if memberID != "" {
		patch.UpdatedBy =
			&memberID
	}

	updated, err := h.uc.Update(
		r.Context(),
		id,
		patch,
	)
	if err != nil {
		writeAccountErr(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusOK,
	)
	_ = json.NewEncoder(w).Encode(
		updated,
	)
}

// ========================================
// Error
// ========================================

func writeAccountErr(
	w http.ResponseWriter,
	err error,
) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(
		err,
		accountdom.ErrInvalidID,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidCompanyID,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidBrandID,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidStripeAccountID,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidMemberID,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidBankName,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidBranchName,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidAccountNumber,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidAccountType,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidCurrency,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidStatus,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidCreatedAt,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrInvalidUpdatedAt,
	):
		code = http.StatusBadRequest

	case errors.Is(
		err,
		accountdom.ErrNotFound,
	):
		code = http.StatusNotFound

	case errors.Is(
		err,
		accountdom.ErrConflict,
	):
		code = http.StatusConflict
	}

	writeJSONError(
		w,
		code,
		err.Error(),
	)
}

func writeJSONError(
	w http.ResponseWriter,
	code int,
	message string,
) {
	w.WriteHeader(
		code,
	)
	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": message,
		},
	)
}
