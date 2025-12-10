// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"
	"strings"

	branddom "narratives/internal/domain/brand"
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
	productBlueprintRepo ProductBlueprintNameRepository
	memberRepo           MemberNameRepository
	modelNumberRepo      ModelNumberRepository
	tokenBlueprintRepo   TokenBlueprintNameRepository
}

// NewNameResolver は各種 Name/Number 用リポジトリをまとめて受け取り、
// 画面向けの「名前解決ヘルパ」を生成する。
func NewNameResolver(
	brandRepo BrandNameRepository,
	productBlueprintRepo ProductBlueprintNameRepository,
	memberRepo MemberNameRepository,
	modelNumberRepo ModelNumberRepository,
	tokenBlueprintRepo TokenBlueprintNameRepository,
) *NameResolver {
	return &NameResolver{
		brandRepo:            brandRepo,
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

	// Brand ドメインの Name フィールドを想定
	return strings.TrimSpace(b.Name)
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

	// ★ Member ドメイン構造体に合わせて LastName / FirstName を使用
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

// 内部共通: *string 型の memberId から氏名を解決
func (r *NameResolver) resolveMemberNameFromPtr(ctx context.Context, memberID *string) string {
	if memberID == nil {
		return ""
	}
	return r.ResolveMemberName(ctx, *memberID)
}

// ResolveAssigneeName は assigneeId → 氏名を解決する。
func (r *NameResolver) ResolveAssigneeName(ctx context.Context, assigneeID string) string {
	return r.ResolveMemberName(ctx, assigneeID)
}

// ResolveCreatedByName は createdBy (memberId) → 氏名を解決する。
func (r *NameResolver) ResolveCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, createdBy)
}

// ResolveUpdatedByName は updatedBy (memberId) → 氏名を解決する。
func (r *NameResolver) ResolveUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, updatedBy)
}

// ResolveRequestedByName は requestedBy (memberId) → 氏名を解決する。
func (r *NameResolver) ResolveRequestedByName(ctx context.Context, requestedBy *string) string {
	return r.resolveMemberNameFromPtr(ctx, requestedBy)
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

	// TokenBlueprint ドメインの Name を優先し、なければ Symbol を表示
	name := strings.TrimSpace(tb.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(tb.Symbol)
}
