// backend\internal\adapters\out\firestore\common\sqlutil.go
package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
)

// RowScanner は *sql.Row, *sql.Rows の両方に共通の Scan() メソッドを持つ抽象型です。
type RowScanner interface {
	Scan(dest ...any) error
}

// IsUniqueViolation は PostgreSQL 一意制約違反（duplicate key）を検知します。
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return true
	}
	return false
}

// ToLowerString は string ベース型(~string)を安全に小文字化します。
func ToLowerString[T ~string](v T) string {
	return strings.ToLower(string(v))
}

// ToUpperString は string ベース型(~string)を安全に大文字化します。
func ToUpperString[T ~string](v T) string {
	return strings.ToUpper(string(v))
}

// NullableOrEmpty は空文字をそのまま格納する（将来、空文字を NULL にしたい場合はここで変更）
func NullableOrEmpty(s string) any {
	return s
}

// --- ここから共通ユーティリリティ追加 ---

// Runner は *sql.DB と *sql.Tx の共通インターフェースです。
type Runner interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// TxKey は context に *sql.Tx を格納するためのキーです。
type TxKey struct{}

// CtxWithTx は ctx に tx を格納して返します。
func CtxWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, TxKey{}, tx)
}

// TxFromCtx は ctx から *sql.Tx を取り出します（無ければ nil）。
func TxFromCtx(ctx context.Context) *sql.Tx {
	if v := ctx.Value(TxKey{}); v != nil {
		if tx, ok := v.(*sql.Tx); ok {
			return tx
		}
	}
	return nil
}

// GetRunner は ctx に Tx があればそれを、無ければ *sql.DB を返します。
func GetRunner(ctx context.Context, db *sql.DB) Runner {
	if tx := TxFromCtx(ctx); tx != nil {
		return tx
	}
	return db
}

// DB は *sql.DB / *sql.Tx の共通インターフェース
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// QueryCount は単純な COUNT(*) を実行して返します。
func QueryCount(ctx context.Context, db DB, query string, args ...any) (int, error) {
	var total int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// NormalizePage はページ番号/件数を正規化し、limit/offset を返します。
func NormalizePage(number, perPage, defaultPerPage, maxPerPage int) (page int, limit int, offset int) {
	page = number
	if page <= 0 {
		page = 1
	}
	limit = perPage
	if limit <= 0 {
		limit = defaultPerPage
	}
	if maxPerPage > 0 && limit > maxPerPage {
		limit = maxPerPage
	}
	offset = (page - 1) * limit
	return
}

// ComputeTotalPages は合計件数と1ページあたり件数から総ページ数を計算します。
func ComputeTotalPages(total, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}

// BuildOrderBy はドメインのソート指定を安全な SQL の ORDER BY に変換します。
// allowed はドメイン列名->SQL列名のホワイトリスト。fallback はデフォルト（例: "created_at DESC"）
func BuildOrderBy(column string, allowed map[string]string, order string, fallback string) string {
	if column == "" {
		if fallback == "" {
			return ""
		}
		return "ORDER BY " + fallback
	}
	colKey := strings.ToLower(column)
	sqlCol, ok := allowed[colKey]
	if !ok || sqlCol == "" {
		if fallback == "" {
			return ""
		}
		return "ORDER BY " + fallback
	}
	dir := strings.ToUpper(order)
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return fmt.Sprintf("ORDER BY %s %s", sqlCol, dir)
}

// AppendCond は WHERE 句の配列と引数配列に条件を追加します。
// exprFmt は $n プレースホルダの位置を自動で len(args)+1 に置き換える前提で "%d" を含めます。
func AppendCond(where *[]string, args *[]any, exprFmt string, val any) {
	*where = append(*where, fmt.Sprintf(exprFmt, len(*args)+1))
	*args = append(*args, val)
}

// FromNullString は sql.NullString を *string に変換します（無効なら nil）。
func FromNullString(ns sql.NullString) *string {
	if ns.Valid {
		v := ns.String
		return &v
	}
	return nil
}

// FromNullTime は sql.NullTime を *time.Time に変換します（無効なら nil）。
func FromNullTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		v := nt.Time
		return &v
	}
	return nil
}

// ToNullString は *string を sql.NullString に変換します（nil/空白は無効）。
func ToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{Valid: false}
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// Ptr は値からポインタを生成する汎用ヘルパー。
func Ptr[T any](v T) *T { return &v }

// ErrCode は grpc.Code からエラーコードを返します。
func ErrCode(grpcCode codes.Code) error {
	return fmt.Errorf("grpc code: %d", grpcCode)
}

// ToDBText は *string を DB に渡せる値(nil/trim)へ変換します。
func ToDBText(p *string) any {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return s
}

// ToDBInt converts *int to a nullable DB parameter.
func ToDBInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// ToDBInt64 converts *int64 to a nullable DB parameter.
func ToDBInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// ToDBTime は *time.Time を DB に渡せる値(nil/UTC)へ変換します。
func ToDBTime(p *time.Time) any {
	if p == nil {
		return nil
	}
	return p.UTC()
}

// NullableTrim returns nil for nil/blank pointers, otherwise the trimmed string.
// Useful for INSERT/UPDATE args to produce SQL NULLs.
func NullableTrim(p *string) any {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return v
}

// JoinErrors aggregates multiple errors into one error.
// If errs is empty it returns nil; if it has one element it returns that element.
func JoinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	var b strings.Builder
	b.WriteString("multiple errors: ")
	for i, e := range errs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Error())
	}
	return errors.New(b.String())
}

// TrimPtr returns a trimmed *string, or nil if the pointer is nil or empty.
// This is useful for Firestore or SQL repositories.
func TrimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// ========================
// 共通: スライス/文字列ユーティリティ
// ========================

// ContainsString は、v(Trim済) が slice 内(各要素 Trim済)に含まれているかを判定します。
func ContainsString(slice []string, v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	for _, s := range slice {
		if strings.TrimSpace(s) == v {
			return true
		}
	}
	return false
}

// IntersectsStrings は、2つのスライスに共通要素(Trim済/空文字除外)があるかどうかを返します。
func IntersectsStrings(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		if s := strings.TrimSpace(v); s != "" {
			set[s] = struct{}{}
		}
	}
	for _, v := range b {
		if s := strings.TrimSpace(v); s != "" {
			if _, ok := set[s]; ok {
				return true
			}
		}
	}
	return false
}

// HasAllStrings は、need の各要素(Trim済/空文字除外)がすべて have に含まれているかを判定します。
func HasAllStrings(have, need []string) bool {
	if len(need) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(have))
	for _, v := range have {
		if s := strings.TrimSpace(v); s != "" {
			set[s] = struct{}{}
		}
	}
	for _, v := range need {
		if s := strings.TrimSpace(v); s != "" {
			if _, ok := set[s]; !ok {
				return false
			}
		}
	}
	return true
}

// ========================
// Firestore 等で使う time ポインタ正規化
// ========================

// NormalizeTimePtr は nil/Zero の *time.Time を nil にし、
// 非 Zero の場合は UTC に変換した新しい *time.Time を返します。
func NormalizeTimePtr(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	utc := p.UTC()
	return &utc
}
