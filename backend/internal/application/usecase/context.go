// internal/application/usecase/context.go
package usecase

import (
	"context"
	"strings"
)

// usecase 層で使う context key
type ctxKey string

const (
	ctxKeyCompanyID ctxKey = "companyId"
	ctxKeyMemberID  ctxKey = "memberId"
)

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

// ============================================================
// memberId 用コンテキストヘルパー
// ============================================================

// ミドルウェアなどから memberId を注入する
func WithMemberID(ctx context.Context, memberID string) context.Context {
	mid := strings.TrimSpace(memberID)
	if mid == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyMemberID, mid)
}

// usecase から memberId を取得する
func MemberIDFromContext(ctx context.Context) string {
	v := ctx.Value(ctxKeyMemberID)
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
