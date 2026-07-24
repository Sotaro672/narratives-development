// internal/application/usecase/context.go
package usecase

import "context"

// usecase 層で使う context key
type ctxKey string

const (
	ctxKeyCompanyID ctxKey = "companyId"
	ctxKeyMemberID  ctxKey = "memberId"
)

func withStringValue(
	ctx context.Context,
	key ctxKey,
	value string,
) context.Context {
	if value == "" {
		return ctx
	}

	return context.WithValue(ctx, key, value)
}

func stringValueFromContext(
	ctx context.Context,
	key ctxKey,
) string {
	if ctx == nil {
		return ""
	}

	v, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}

	return v
}

// ミドルウェアなど外側から companyId を注入するためのヘルパー
func WithCompanyID(
	ctx context.Context,
	companyID string,
) context.Context {
	return withStringValue(
		ctx,
		ctxKeyCompanyID,
		companyID,
	)
}

// usecase 内部で companyId を取り出すためのヘルパー
func CompanyIDFromContext(
	ctx context.Context,
) string {
	return stringValueFromContext(
		ctx,
		ctxKeyCompanyID,
	)
}

// ミドルウェアなどから memberId を注入する
func WithMemberID(
	ctx context.Context,
	memberID string,
) context.Context {
	return withStringValue(
		ctx,
		ctxKeyMemberID,
		memberID,
	)
}

// usecase から memberId を取得する
func MemberIDFromContext(
	ctx context.Context,
) string {
	return stringValueFromContext(
		ctx,
		ctxKeyMemberID,
	)
}
