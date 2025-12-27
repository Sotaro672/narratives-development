// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"
	"reflect"
	"strings"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ------------------------------------------------------------
// Repository interfaces (最小限の読み取り専用ポート)
// ------------------------------------------------------------

// Brand 名の取得に必要な最小限のインターフェース
type BrandNameRepository interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// Company 名の取得に必要な最小限のインターフェース
type CompanyNameRepository interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

// ProductBlueprint → productName だけ取得できればよい
type ProductBlueprintNameRepository interface {
	GetProductNameByID(ctx context.Context, id string) (string, error)
}

// Member → 氏名取得だけできればよい
type MemberNameRepository interface {
	GetByID(ctx context.Context, id string) (memberdom.Member, error)
}

// Model → modelId から ModelVariation を 1 件取得できればよい
// ★ ModelNameRepository → ModelNumberRepository にリネーム
type ModelNumberRepository interface {
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

// TokenBlueprint → name / symbol を取得できればよい
type TokenBlueprintNameRepository interface {
	GetByID(ctx context.Context, id string) (tbdom.TokenBlueprint, error)
}

// ------------------------------------------------------------
// NameResolver 本体
// ------------------------------------------------------------

type NameResolver struct {
	brandRepo            BrandNameRepository
	companyRepo          CompanyNameRepository
	productBlueprintRepo ProductBlueprintNameRepository
	memberRepo           MemberNameRepository
	modelNumberRepo      ModelNumberRepository
	tokenBlueprintRepo   TokenBlueprintNameRepository
}

// NewNameResolver は各種 Name/Number 用リポジトリをまとめて受け取り、
// 画面向けの「名前解決ヘルパ」を生成する。
func NewNameResolver(
	brandRepo BrandNameRepository,
	companyRepo CompanyNameRepository,
	productBlueprintRepo ProductBlueprintNameRepository,
	memberRepo MemberNameRepository,
	modelNumberRepo ModelNumberRepository,
	tokenBlueprintRepo TokenBlueprintNameRepository,
) *NameResolver {
	return &NameResolver{
		brandRepo:            brandRepo,
		companyRepo:          companyRepo,
		productBlueprintRepo: productBlueprintRepo,
		memberRepo:           memberRepo,
		modelNumberRepo:      modelNumberRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
	}
}

// ------------------------------------------------------------
// Brand 関連
// ------------------------------------------------------------

// ResolveBrandName は brandId からブランド名（Name）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveBrandName(ctx context.Context, brandID string) string {
	if r == nil || r.brandRepo == nil {
		return ""
	}
	id := strings.TrimSpace(brandID)
	if id == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(b.Name)
}

// ✅ NEW: ResolveBrandCompanyID は brandId から companyId を解決する。
// tokenBlueprint が companyId を持っていないケースに対応する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveBrandCompanyID(ctx context.Context, brandID string) string {
	if r == nil || r.brandRepo == nil {
		return ""
	}
	id := strings.TrimSpace(brandID)
	if id == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	// Brand ドメインのフィールド名揺れを reflection で吸収（CompanyID / CompanyId / companyId）
	rv := reflect.ValueOf(b)
	if rv.IsValid() && rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return ""
	}

	for _, n := range []string{"CompanyID", "CompanyId", "companyId"} {
		f := rv.FieldByName(n)
		if !f.IsValid() {
			continue
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		if f.Kind() == reflect.String {
			s := strings.TrimSpace(f.String())
			if s != "" {
				return s
			}
		}
	}

	return ""
}

// ------------------------------------------------------------
// Company 関連
// ------------------------------------------------------------

// ResolveCompanyName は companyId から会社名（Name）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveCompanyName(ctx context.Context, companyID string) string {
	if r == nil || r.companyRepo == nil {
		return ""
	}
	id := strings.TrimSpace(companyID)
	if id == "" {
		return ""
	}

	c, err := r.companyRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(c.Name)
}

// ------------------------------------------------------------
// ProductBlueprint 関連
// ------------------------------------------------------------

// ResolveProductName は productBlueprintId から productName を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveProductName(ctx context.Context, productBlueprintID string) string {
	if r == nil || r.productBlueprintRepo == nil {
		return ""
	}
	id := strings.TrimSpace(productBlueprintID)
	if id == "" {
		return ""
	}

	name, err := r.productBlueprintRepo.GetProductNameByID(ctx, id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

// ------------------------------------------------------------
// Member 関連
// ------------------------------------------------------------

// ResolveMemberName は memberId から表示用の名前（例: "姓 名"）を解決する。
// Member ドメインの定義に合わせて LastName / FirstName を利用する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveMemberName(ctx context.Context, memberID string) string {
	if r == nil || r.memberRepo == nil {
		return ""
	}
	id := strings.TrimSpace(memberID)
	if id == "" {
		return ""
	}

	m, err := r.memberRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	family := strings.TrimSpace(m.LastName) // 姓
	given := strings.TrimSpace(m.FirstName) // 名

	switch {
	case family == "" && given == "":
		return ""
	case family == "":
		return given
	case given == "":
		return family
	default:
		return family + " " + given
	}
}

// ---- memberId 派生フィールド向けヘルパ ----

func (r *NameResolver) resolveMemberNameFromPtr(ctx context.Context, memberID *string) string {
	if memberID == nil {
		return ""
	}
	return r.ResolveMemberName(ctx, *memberID)
}

func (r *NameResolver) ResolveAssigneeName(ctx context.Context, assigneeID string) string {
	return r.ResolveMemberName(ctx, assigneeID)
}

func (r *NameResolver) ResolveCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, createdBy)
}

func (r *NameResolver) ResolveUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, updatedBy)
}

func (r *NameResolver) ResolveRequestedByName(ctx context.Context, requestedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, requestedBy)
}

func (r *NameResolver) ResolveInspectedByName(ctx context.Context, inspectedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, inspectedBy)
}

func (r *NameResolver) ResolvePrintedByName(ctx context.Context, printedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, printedBy)
}

// ------------------------------------------------------------
// ModelVariation (modelId → modelNumber) 関連
// ------------------------------------------------------------

// ResolveModelNumber は modelVariationId から modelNumber を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveModelNumber(ctx context.Context, variationID string) string {
	if r == nil || r.modelNumberRepo == nil {
		return ""
	}
	id := strings.TrimSpace(variationID)
	if id == "" {
		return ""
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, id)
	if err != nil || mv == nil {
		return ""
	}

	return strings.TrimSpace(mv.ModelNumber)
}

// ------------------------------------------------------------
// TokenBlueprint 関連
// ------------------------------------------------------------

// ResolveTokenName は tokenBlueprintId からトークン名（例: name or symbol）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveTokenName(ctx context.Context, tokenBlueprintID string) string {
	if r == nil || r.tokenBlueprintRepo == nil {
		return ""
	}
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return ""
	}

	tb, err := r.tokenBlueprintRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	name := strings.TrimSpace(tb.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(tb.Symbol)
}
