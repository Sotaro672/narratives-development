// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"
	"log"
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
	if r == nil {
		log.Printf("[name_resolver] ResolveBrandName: resolver is nil")
		return ""
	}
	if r.brandRepo == nil {
		log.Printf("[name_resolver] ResolveBrandName: brandRepo is nil")
		return ""
	}

	id := strings.TrimSpace(brandID)
	if id == "" {
		log.Printf("[name_resolver] ResolveBrandName: brandID is empty")
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[name_resolver] ResolveBrandName: GetByID error brandId=%q err=%v", id, err)
		return ""
	}

	name := strings.TrimSpace(b.Name)
	log.Printf("[name_resolver] ResolveBrandName: ok brandId=%q name=%q", id, name)
	return name
}

// ------------------------------------------------------------
// Company 関連
// ------------------------------------------------------------

// ResolveCompanyName は companyId から会社名（Name）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveCompanyName(ctx context.Context, companyID string) string {
	if r == nil {
		log.Printf("[name_resolver] ResolveCompanyName: resolver is nil")
		return ""
	}
	if r.companyRepo == nil {
		log.Printf("[name_resolver] ResolveCompanyName: companyRepo is nil")
		return ""
	}

	id := strings.TrimSpace(companyID)
	if id == "" {
		log.Printf("[name_resolver] ResolveCompanyName: companyID is empty")
		return ""
	}

	c, err := r.companyRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[name_resolver] ResolveCompanyName: GetByID error companyId=%q err=%v", id, err)
		return ""
	}

	name := strings.TrimSpace(c.Name)
	log.Printf("[name_resolver] ResolveCompanyName: ok companyId=%q name=%q", id, name)
	return name
}

// ------------------------------------------------------------
// ProductBlueprint 関連
// ------------------------------------------------------------

// ResolveProductName は productBlueprintId から productName を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveProductName(ctx context.Context, productBlueprintID string) string {
	if r == nil {
		log.Printf("[name_resolver] ResolveProductName: resolver is nil")
		return ""
	}
	if r.productBlueprintRepo == nil {
		log.Printf("[name_resolver] ResolveProductName: productBlueprintRepo is nil")
		return ""
	}

	id := strings.TrimSpace(productBlueprintID)
	if id == "" {
		log.Printf("[name_resolver] ResolveProductName: productBlueprintID is empty")
		return ""
	}

	name, err := r.productBlueprintRepo.GetProductNameByID(ctx, id)
	if err != nil {
		log.Printf("[name_resolver] ResolveProductName: GetProductNameByID error productBlueprintId=%q err=%v", id, err)
		return ""
	}

	out := strings.TrimSpace(name)
	log.Printf("[name_resolver] ResolveProductName: ok productBlueprintId=%q name=%q", id, out)
	return out
}

// ------------------------------------------------------------
// Member 関連
// ------------------------------------------------------------

// ResolveMemberName は memberId から表示用の名前（例: "姓 名"）を解決する。
// Member ドメインの定義に合わせて LastName / FirstName を利用する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveMemberName(ctx context.Context, memberID string) string {
	if r == nil {
		log.Printf("[name_resolver] ResolveMemberName: resolver is nil")
		return ""
	}
	if r.memberRepo == nil {
		log.Printf("[name_resolver] ResolveMemberName: memberRepo is nil")
		return ""
	}

	id := strings.TrimSpace(memberID)
	if id == "" {
		log.Printf("[name_resolver] ResolveMemberName: memberID is empty")
		return ""
	}

	m, err := r.memberRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[name_resolver] ResolveMemberName: GetByID error memberId=%q err=%v", id, err)
		return ""
	}

	family := strings.TrimSpace(m.LastName) // 姓
	given := strings.TrimSpace(m.FirstName) // 名

	var out string
	switch {
	case family == "" && given == "":
		out = ""
	case family == "":
		out = given
	case given == "":
		out = family
	default:
		out = family + " " + given
	}

	log.Printf("[name_resolver] ResolveMemberName: ok memberId=%q name=%q", id, out)
	return out
}

// ---- memberId 派生フィールド向けヘルパ ----

func (r *NameResolver) resolveMemberNameFromPtr(ctx context.Context, memberID *string) string {
	if memberID == nil {
		log.Printf("[name_resolver] resolveMemberNameFromPtr: memberID is nil")
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
	if r == nil {
		log.Printf("[name_resolver] ResolveModelNumber: resolver is nil")
		return ""
	}
	if r.modelNumberRepo == nil {
		log.Printf("[name_resolver] ResolveModelNumber: modelNumberRepo is nil")
		return ""
	}

	id := strings.TrimSpace(variationID)
	if id == "" {
		log.Printf("[name_resolver] ResolveModelNumber: variationID is empty")
		return ""
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, id)
	if err != nil {
		log.Printf("[name_resolver] ResolveModelNumber: GetModelVariationByID error variationId=%q err=%v", id, err)
		return ""
	}
	if mv == nil {
		log.Printf("[name_resolver] ResolveModelNumber: modelVariation is nil variationId=%q", id)
		return ""
	}

	out := strings.TrimSpace(mv.ModelNumber)
	log.Printf("[name_resolver] ResolveModelNumber: ok variationId=%q modelNumber=%q", id, out)
	return out
}

// ※ 注意:
// - ModelResolved 型 / ResolveModelResolved メソッドは model_resolver.go 側に定義済みのため、
//   このファイルでは重複定義しない（DuplicateDecl / DuplicateMethod 回避）。

// ------------------------------------------------------------
// TokenBlueprint 関連
// ------------------------------------------------------------

// ResolveTokenName は tokenBlueprintId からトークン名（例: name or symbol）を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveTokenName(ctx context.Context, tokenBlueprintID string) string {
	if r == nil {
		log.Printf("[name_resolver] ResolveTokenName: resolver is nil")
		return ""
	}
	if r.tokenBlueprintRepo == nil {
		log.Printf("[name_resolver] ResolveTokenName: tokenBlueprintRepo is nil")
		return ""
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		log.Printf("[name_resolver] ResolveTokenName: tokenBlueprintID is empty")
		return ""
	}

	tb, err := r.tokenBlueprintRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[name_resolver] ResolveTokenName: GetByID error tokenBlueprintId=%q err=%v", id, err)
		return ""
	}

	name := strings.TrimSpace(tb.Name)
	symbol := strings.TrimSpace(tb.Symbol)

	if name != "" {
		log.Printf("[name_resolver] ResolveTokenName: ok tokenBlueprintId=%q name=%q", id, name)
		return name
	}

	log.Printf("[name_resolver] ResolveTokenName: ok tokenBlueprintId=%q (name empty) symbol=%q", id, symbol)
	return symbol
}
