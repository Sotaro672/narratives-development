// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
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

// ProductBlueprint の取得に必要な最小限のインターフェース
type ProductBlueprintNameRepository interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// Member → Firebase UID から氏名取得できればよい
//
// IMPORTANT:
// - NameResolver の member 名解決は uid 専用
// - member docId では検索しない
// - TokenBlueprint.createdBy / updatedBy は Firebase UID を保持する前提
type MemberNameRepository interface {
	GetByFirebaseUID(ctx context.Context, uid string) (memberdom.Member, error)
}

// Model → modelId から ModelVariation を 1 件取得できればよい
type ModelNumberRepository interface {
	GetModelVariationByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error)
}

// TokenBlueprint → name / symbol を取得できればよい
// GetNameByID のみ要求する（NameResolver用途の最小ポート）
type TokenBlueprintNameRepository interface {
	GetNameByID(ctx context.Context, id string) (string, error)
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

	id := brandID
	if id == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return b.Name
}

// ResolveBrandCompanyID は brandId から companyId を解決する。
// tokenBlueprint が companyId を持っていないケースに対応する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveBrandCompanyID(ctx context.Context, brandID string) string {
	if r == nil || r.brandRepo == nil {
		return ""
	}

	id := brandID
	if id == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return b.CompanyID
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

	id := companyID
	if id == "" {
		return ""
	}

	c, err := r.companyRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return c.Name
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

	id := productBlueprintID
	if id == "" {
		return ""
	}

	pb, err := r.productBlueprintRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return pb.ProductName
}

// ------------------------------------------------------------
// Member 関連
// ------------------------------------------------------------

func formatMemberDisplayName(m memberdom.Member) string {
	family := m.LastName // 姓
	given := m.FirstName // 名

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

// ResolveMemberName は Firebase UID から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - uid 専用
// - member docId fallback はしない
// - createdBy / updatedBy は Firebase UID を保持する前提
func (r *NameResolver) ResolveMemberName(ctx context.Context, uid string) string {
	if r == nil || r.memberRepo == nil {
		return ""
	}

	if uid == "" {
		return ""
	}

	m, err := r.memberRepo.GetByFirebaseUID(ctx, uid)
	if err != nil {
		return ""
	}

	return formatMemberDisplayName(m)
}

// ---- uid 派生フィールド向けヘルパ ----

func (r *NameResolver) resolveMemberNameFromPtr(ctx context.Context, uid *string) string {
	if uid == nil {
		return ""
	}

	return r.ResolveMemberName(ctx, *uid)
}

// ResolveAssigneeName は assigneeId(uid) から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - assigneeId は Firebase UID を保存する
// - member docId fallback はしない
// - 見つからない場合は空文字を返し、呼び出し側で fallback する
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

// ResolveModelNumber は modelId から modelNumber を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveModelNumber(ctx context.Context, variationID string) string {
	if r == nil || r.modelNumberRepo == nil {
		return ""
	}

	id := variationID
	if id == "" {
		return ""
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, id)
	if err != nil || mv == nil {
		return ""
	}

	if apparelMV, ok := mv.(modeldom.ApparelModelVariation); ok {
		return apparelMV.ModelNumber
	}

	if alcoholMV, ok := mv.(modeldom.AlcoholModelVariation); ok {
		return alcoholMV.ModelNumber
	}

	return ""
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

	id := tokenBlueprintID
	if id == "" {
		return ""
	}

	name, err := r.tokenBlueprintRepo.GetNameByID(ctx, id)
	if err != nil {
		return ""
	}

	return name
}
