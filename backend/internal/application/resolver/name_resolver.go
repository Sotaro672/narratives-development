// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
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

// ProductBlueprint の取得に必要な最小限のインターフェース
type ProductBlueprintNameRepository interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// Member 名の取得に必要な最小限のインターフェース。
//
// IMPORTANT:
// - member repository port の Get 系は GetByID に統一する。
// - GetByID は member docId を受け取り、member.Record を返す。
// - Firebase UID での名前解決はここでは扱わない。
// - ProductBlueprint.assigneeId / createdBy / updatedBy は member docId 前提。
type MemberNameRepository interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
}

// Model → modelId から ModelVariation を 1 件取得できればよい
type ModelNumberRepository interface {
	GetModelVariationByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error)
}

// TokenBlueprint の取得に必要な最小限のインターフェース
//
// tokenBlueprint.RepositoryPort と同じ GetByID に統一する。
// name 解決が必要な場合も、GetByID の結果から Name を参照する。
type TokenBlueprintNameRepository interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
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
//
// NOTE:
// ProductBlueprint 自身の list/detail では ProductBlueprint.ProductName をそのまま使う。
// このメソッドは他ドメインが productBlueprintId から productName を解決したい場合向け。
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
	family := m.LastName
	given := m.FirstName

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

// ResolveMemberName は member docId から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - member docId 専用
// - Firebase UID fallback はしない
// - member repository port の Get 系は GetByID のみに統一する
func (r *NameResolver) ResolveMemberName(ctx context.Context, memberID string) string {
	return r.ResolveMemberNameByID(ctx, memberID)
}

// ResolveMemberNameByID は member docId から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - member docId 専用
// - Firebase UID fallback はしない
// - ProductBlueprint.assigneeId / createdBy / updatedBy など、member docId を保存しているフィールド向け
func (r *NameResolver) ResolveMemberNameByID(ctx context.Context, memberID string) string {
	if r == nil || r.memberRepo == nil {
		return ""
	}

	id := memberID
	if id == "" {
		return ""
	}

	rec, err := r.memberRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}

	return formatMemberDisplayName(rec.Member)
}

// ---- member docId 派生フィールド向けヘルパ ----

func (r *NameResolver) resolveMemberNameByIDFromPtr(ctx context.Context, memberID *string) string {
	if memberID == nil {
		return ""
	}

	return r.ResolveMemberNameByID(ctx, *memberID)
}

// ResolveAssigneeName は assigneeId から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - member docId 専用
// - Firebase UID fallback はしない
func (r *NameResolver) ResolveAssigneeName(ctx context.Context, assigneeID string) string {
	return r.ResolveMemberNameByID(ctx, assigneeID)
}

func (r *NameResolver) ResolveCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, createdBy)
}

func (r *NameResolver) ResolveUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, updatedBy)
}

func (r *NameResolver) ResolveRequestedByName(ctx context.Context, requestedBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, requestedBy)
}

func (r *NameResolver) ResolveInspectedByName(ctx context.Context, inspectedBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, inspectedBy)
}

func (r *NameResolver) ResolvePrintedByName(ctx context.Context, printedBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, printedBy)
}

// ---- ProductBlueprint 専用: member docId 解決 ----

func (r *NameResolver) ResolveProductBlueprintAssigneeName(ctx context.Context, assigneeID string) string {
	return r.ResolveMemberNameByID(ctx, assigneeID)
}

func (r *NameResolver) ResolveProductBlueprintCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, createdBy)
}

func (r *NameResolver) ResolveProductBlueprintUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameByIDFromPtr(ctx, updatedBy)
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

// ResolveTokenName は tokenBlueprintId からトークン名を解決する。
// 取得できなかった場合は空文字列を返す。
func (r *NameResolver) ResolveTokenName(ctx context.Context, tokenBlueprintID string) string {
	if r == nil || r.tokenBlueprintRepo == nil {
		return ""
	}

	id := tokenBlueprintID
	if id == "" {
		return ""
	}

	tb, err := r.tokenBlueprintRepo.GetByID(ctx, id)
	if err != nil || tb == nil {
		return ""
	}

	return tb.Name
}
