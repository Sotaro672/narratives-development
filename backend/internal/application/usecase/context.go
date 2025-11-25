// internal/application/usecase/context.go
package usecase

import (
	"context"
	"strings"
)

// usecase 層で使う context key
type ctxKey string

const ctxKeyCompanyID ctxKey = "companyId"

// ミドルウェアなど外側から companyId を注入するためのヘルパー
func WithCompanyID(ctx context.Context, companyID string) context.Context {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyCompanyID, cid)
}

// usecase 内部で companyId を取り出すためのヘルパー
func CompanyIDFromContext(ctx context.Context) string {
	v := ctx.Value(ctxKeyCompanyID)
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// 既存コードとの互換用（MemberUsecase, BrandUsecase から呼ばせる）
func companyIDFromContext(ctx context.Context) string {
	return CompanyIDFromContext(ctx)
}
