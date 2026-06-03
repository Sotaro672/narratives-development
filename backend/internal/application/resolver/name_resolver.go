// backend/internal/application/resolver/name_resolver.go
package resolver

import (
	"context"
	"strings"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
	userdom "narratives/internal/domain/user"
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
// - member repository port の Get 系は GetByUID に統一する。
// - GetByUID は Firebase Auth UID を受け取り、member.Record を返す。
// - member docId での名前解決はここでは扱わない。
// - ProductBlueprint.assigneeId / createdBy / updatedBy は Firebase Auth UID 前提。
type MemberNameRepository interface {
	GetByUID(ctx context.Context, uid string) (memberdom.Record, error)
}

// User 名の取得に必要な最小限のインターフェース。
//
// IMPORTANT:
// - user repository port の GetByID を使う。
// - GetNameByID は使わない。
type UserNameRepository interface {
	GetByID(ctx context.Context, id string) (*userdom.User, error)
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
	userRepo             UserNameRepository
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
	userRepo UserNameRepository,
	modelNumberRepo ModelNumberRepository,
	tokenBlueprintRepo TokenBlueprintNameRepository,
) *NameResolver {
	return &NameResolver{
		brandRepo:            brandRepo,
		companyRepo:          companyRepo,
		productBlueprintRepo: productBlueprintRepo,
		memberRepo:           memberRepo,
		userRepo:             userRepo,
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

	if brandID == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, brandID)
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

	if brandID == "" {
		return ""
	}

	b, err := r.brandRepo.GetByID(ctx, brandID)
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

	if companyID == "" {
		return ""
	}

	c, err := r.companyRepo.GetByID(ctx, companyID)
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

	if productBlueprintID == "" {
		return ""
	}

	pb, err := r.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return ""
	}

	return pb.ProductName
}

// ------------------------------------------------------------
// Member 関連
// ------------------------------------------------------------

func formatMemberDisplayName(rec memberdom.Record) string {
	return memberdom.FormatLastFirst(rec.Member.LastName, rec.Member.FirstName)
}

// ResolveMemberName は Firebase Auth UID から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - Firebase Auth UID 専用
// - member docId fallback はしない
func (r *NameResolver) ResolveMemberName(ctx context.Context, memberUID string) string {
	return r.ResolveMemberNameByUID(ctx, memberUID)
}

// ResolveMemberNameByUID は Firebase Auth UID から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - Firebase Auth UID 専用
// - member docId fallback はしない
// - ProductBlueprint.assigneeId / createdBy / updatedBy など、Firebase Auth UID を保存しているフィールド向け
func (r *NameResolver) ResolveMemberNameByUID(ctx context.Context, memberUID string) string {
	if r == nil || r.memberRepo == nil {
		return ""
	}

	if memberUID == "" {
		return ""
	}

	rec, err := r.memberRepo.GetByUID(ctx, memberUID)
	if err != nil {
		return ""
	}

	return formatMemberDisplayName(rec)
}

// ---- member UID 派生フィールド向けヘルパ ----

func (r *NameResolver) resolveMemberNameByUIDFromPtr(ctx context.Context, memberUID *string) string {
	if memberUID == nil {
		return ""
	}

	return r.ResolveMemberNameByUID(ctx, *memberUID)
}

// ResolveAssigneeName は assigneeId から member の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - assigneeId は Firebase Auth UID 前提
// - member docId fallback はしない
func (r *NameResolver) ResolveAssigneeName(ctx context.Context, assigneeUID string) string {
	return r.ResolveMemberNameByUID(ctx, assigneeUID)
}

func (r *NameResolver) ResolveCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, createdBy)
}

func (r *NameResolver) ResolveUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, updatedBy)
}

func (r *NameResolver) ResolveRequestedByName(ctx context.Context, requestedBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, requestedBy)
}

func (r *NameResolver) ResolveInspectedByName(ctx context.Context, inspectedBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, inspectedBy)
}

func (r *NameResolver) ResolvePrintedByName(ctx context.Context, printedBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, printedBy)
}

// ---- ProductBlueprint 専用: member UID 解決 ----

func (r *NameResolver) ResolveProductBlueprintAssigneeName(ctx context.Context, assigneeUID string) string {
	return r.ResolveMemberNameByUID(ctx, assigneeUID)
}

func (r *NameResolver) ResolveProductBlueprintCreatedByName(ctx context.Context, createdBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, createdBy)
}

func (r *NameResolver) ResolveProductBlueprintUpdatedByName(ctx context.Context, updatedBy *string) string {
	return r.resolveMemberNameByUIDFromPtr(ctx, updatedBy)
}

// ------------------------------------------------------------
// User 関連
// ------------------------------------------------------------

func formatUserDisplayName(u *userdom.User) string {
	if u == nil {
		return ""
	}

	lastName := ""
	firstName := ""

	if u.LastName != nil {
		lastName = strings.TrimSpace(*u.LastName)
	}
	if u.FirstName != nil {
		firstName = strings.TrimSpace(*u.FirstName)
	}

	switch {
	case lastName != "" && firstName != "":
		return lastName + " " + firstName
	case lastName != "":
		return lastName
	case firstName != "":
		return firstName
	default:
		return ""
	}
}

// ResolveUserName は userId から user の表示名（例: "姓 名"）を解決する。
//
// IMPORTANT:
// - user repository port の GetByID を使う
// - GetNameByID は使わない
func (r *NameResolver) ResolveUserName(ctx context.Context, userID string) string {
	if r == nil || r.userRepo == nil {
		return ""
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}

	u, err := r.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return ""
	}

	return formatUserDisplayName(u)
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

	if variationID == "" {
		return ""
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, variationID)
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

	if tokenBlueprintID == "" {
		return ""
	}

	tb, err := r.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil || tb == nil {
		return ""
	}

	return tb.Name
}
