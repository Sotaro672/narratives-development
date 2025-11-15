package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	memdom "narratives/internal/domain/member"
)

// ------------------------------
// Utility functions
// ------------------------------

// MethodNotAllowed writes 405 response.
func MethodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
}

// IsNotSupported checks error message for "not supported".
func IsNotSupported(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not supported")
}

// NormalizeStrPtr trims a *string; empty/blank becomes nil.
func NormalizeStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// ------------------------------
// BaseHandler (共通機能)
// ------------------------------

type BaseHandler struct {
	MemberRepo memdom.Repository
}

// CurrentCompanyID はログイン中ユーザーの companyId を返す。
// 認証されていない場合はエラーを返す。
func (h *BaseHandler) CurrentCompanyID(r *http.Request) (string, error) {
	me, ok := middleware.CurrentMember(r)
	if !ok {
		return "", errors.New("unauthorized")
	}
	return me.CompanyID, nil
}
