// backend/internal/adapters/in/http/mall/handler/identity_verification_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	identitydom "narratives/internal/domain/identity_verification"
)

// ============================================================
// Handler
// ============================================================

// IdentityVerificationHandler は Mall ユーザーの本人確認状態を扱います。
//
// Endpoints:
//
//	GET  /mall/me/identity-verification
//	POST /mall/me/identity-verification/mock
//
// userId は request body や query parameter から受け取りません。
// UserAuthMiddleware が検証した Firebase UID のみを使用します。
type IdentityVerificationHandler struct {
	uc *usecase.IdentityVerificationUsecase
}

func NewIdentityVerificationHandler(
	uc *usecase.IdentityVerificationUsecase,
) http.Handler {
	return &IdentityVerificationHandler{
		uc: uc,
	}
}

// ============================================================
// Routing
// ============================================================

func (h *IdentityVerificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// /mall/me/... と /me/... の両方を同じ Handler で扱えるようにする。
	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
	}

	switch {
	case path == "/me/identity-verification" &&
		r.Method == http.MethodGet:
		h.getStatus(w, r)
		return

	case path == "/me/identity-verification/mock" &&
		r.Method == http.MethodPost:
		h.verifyWithMockMyNumberCard(w, r)
		return

	case path == "/me/identity-verification":
		w.Header().Set(
			"Allow",
			"GET, OPTIONS",
		)
		writeJSON(
			w,
			http.StatusMethodNotAllowed,
			map[string]string{
				"error": "method_not_allowed",
			},
		)
		return

	case path == "/me/identity-verification/mock":
		w.Header().Set(
			"Allow",
			"POST, OPTIONS",
		)
		writeJSON(
			w,
			http.StatusMethodNotAllowed,
			map[string]string{
				"error": "method_not_allowed",
			},
		)
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

// ============================================================
// GET /mall/me/identity-verification
// ============================================================

// getStatus は現在のユーザーの本人確認状態を返します。
//
// 本人確認 document がまだ存在しない場合も Usecase が
// status=unverified として返すため、404 にはなりません.
//
// Response example:
//
//	{
//	  "status": "unverified"
//	}
//
// verified:
//
//	{
//	  "status": "verified",
//	  "method": "my_number_card",
//	  "provider": "mock",
//	  "verifiedAt": "..."
//	}
func (h *IdentityVerificationHandler) getStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !h.requireUsecase(w) {
		return
	}

	userID, ok := requireIdentityVerificationUserID(
		w,
		r,
	)
	if !ok {
		return
	}

	result, err := h.uc.GetStatus(
		r.Context(),
		userID,
	)
	if err != nil {
		writeIdentityVerificationError(
			w,
			err,
		)
		return
	}

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

// ============================================================
// POST /mall/me/identity-verification/mock
// ============================================================

// verifyWithMockMyNumberCard は開発段階のモック KYC を実行します。
//
// request body から本人情報は受け取りません。
// 個人番号、カード画像、暗証番号なども受け取りません。
//
// 同じユーザーが複数回 POST しても、Usecase 側で冪等に処理します。
func (h *IdentityVerificationHandler) verifyWithMockMyNumberCard(
	w http.ResponseWriter,
	r *http.Request,
) {
	if !h.requireUsecase(w) {
		return
	}

	userID, ok := requireIdentityVerificationUserID(
		w,
		r,
	)
	if !ok {
		return
	}

	_, err := h.uc.VerifyWithMockMyNumberCard(
		r.Context(),
		userID,
	)
	if err != nil {
		writeIdentityVerificationError(
			w,
			err,
		)
		return
	}

	// Domain Entity 自体を返すと将来内部フィールドが増えた際に
	// 意図せず公開する可能性があるため、公開用 status を改めて取得する。
	result, err := h.uc.GetStatus(
		r.Context(),
		userID,
	)
	if err != nil {
		writeIdentityVerificationError(
			w,
			err,
		)
		return
	}

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

// ============================================================
// Auth
// ============================================================

func requireIdentityVerificationUserID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	userID, ok := middleware.CurrentUserUID(r)
	userID = strings.TrimSpace(userID)

	if !ok || userID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return "", false
	}

	return userID, true
}

// ============================================================
// Dependencies
// ============================================================

func (h *IdentityVerificationHandler) requireUsecase(
	w http.ResponseWriter,
) bool {
	if h != nil && h.uc != nil {
		return true
	}

	writeJSON(
		w,
		http.StatusServiceUnavailable,
		map[string]string{
			"error": "identity_verification_usecase_not_initialized",
		},
	)

	return false
}

// ============================================================
// Error mapping
// ============================================================

func writeIdentityVerificationError(
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
	errorCode := "internal_error"

	switch {
	case errors.Is(
		err,
		identitydom.ErrInvalidUserID,
	):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_user_id"

	case errors.Is(
		err,
		identitydom.ErrNotFound,
	):
		statusCode = http.StatusNotFound
		errorCode = "identity_verification_not_found"

	case errors.Is(
		err,
		identitydom.ErrInvalidState,
	):
		statusCode = http.StatusConflict
		errorCode = "invalid_identity_verification_state"

	case errors.Is(
		err,
		usecase.ErrIdentityVerificationNotConfigured,
	):
		statusCode = http.StatusServiceUnavailable
		errorCode = "identity_verification_usecase_not_initialized"

	case errors.Is(
		err,
		identitydom.ErrInvalidStatus,
	),
		errors.Is(
			err,
			identitydom.ErrInvalidMethod,
		),
		errors.Is(
			err,
			identitydom.ErrInvalidProvider,
		),
		errors.Is(
			err,
			identitydom.ErrInvalidProviderVerificationID,
		),
		errors.Is(
			err,
			identitydom.ErrInvalidVerifiedAt,
		),
		errors.Is(
			err,
			identitydom.ErrInvalidCreatedAt,
		),
		errors.Is(
			err,
			identitydom.ErrInvalidUpdatedAt,
		):
		statusCode = http.StatusInternalServerError
		errorCode = "invalid_identity_verification_data"
	}

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	writeJSON(
		w,
		statusCode,
		map[string]string{
			"error": errorCode,
		},
	)
}
