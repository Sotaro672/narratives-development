// backend/internal/domain/tokenBlueprint/repository_port.go
package tokenBlueprint

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ===============================
// Create 用入力
// ===============================
//
// Firebase Storage 移行後の正:
// - backend は GCS signed URL / upload endpoint を持たない
// - iconId は存在しない
// - tokenBlueprintIcon は frontend が Firebase Storage へ直接 upload する
// - iconUrl は Firebase Storage の downloadURL を保存する
// - iconObjectPath は Firebase Storage 上の icon objectPath を保存する
// - iconFileName / iconContentType / iconSize は表示・差し替え・監査用に保存する
// - tokenBlueprintContents も frontend が Firebase Storage へ直接 upload する
// - contentFiles は []ContentFile embedded として保存する
// - contentFiles[].url には Firebase Storage の downloadURL を保存する
// - contentFiles[].objectPath には Firebase Storage 上の objectPath を保存する
// - contentFiles[].name / contentFiles[].size は表示・差し替え・監査用に保存する
// - minted は create 時は常に false
// - metadataUri は任意
type CreateTokenBlueprintInput struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`

	IconURL         string `json:"iconUrl,omitempty"`
	IconObjectPath  string `json:"iconObjectPath,omitempty"`
	IconFileName    string `json:"iconFileName,omitempty"`
	IconContentType string `json:"iconContentType,omitempty"`
	IconSize        int64  `json:"iconSize,omitempty"`

	ContentFiles []ContentFile `json:"contentFiles"`

	AssigneeID string `json:"assigneeId"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	CreatedBy string     `json:"createdBy"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string     `json:"updatedBy"`

	MetadataURI string `json:"metadataUri,omitempty"`
}

// ===============================
// Update 用入力
// ===============================
//
// Firebase Storage 移行後の正:
// - backend は GCS signed URL / upload endpoint を持たない
// - iconUrl は Firebase Storage downloadURL
// - iconObjectPath は Firebase Storage objectPath
// - iconFileName / iconContentType / iconSize は表示・差し替え・監査用
// - contentFiles は []ContentFile の全置換
// - contentFiles[].url は Firebase Storage downloadURL
// - contentFiles[].objectPath は Firebase Storage objectPath
// - contentFiles[].name / contentFiles[].size は表示・差し替え・監査用
// - minted は bool
// - metadataUri は任意
type UpdateTokenBlueprintInput struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`

	IconURL         *string `json:"iconUrl,omitempty"`
	IconObjectPath  *string `json:"iconObjectPath,omitempty"`
	IconFileName    *string `json:"iconFileName,omitempty"`
	IconContentType *string `json:"iconContentType,omitempty"`
	IconSize        *int64  `json:"iconSize,omitempty"`

	ContentFiles *[]ContentFile `json:"contentFiles,omitempty"`

	AssigneeID *string `json:"assigneeId,omitempty"`
	Minted     *bool   `json:"minted,omitempty"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy *string    `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`

	MetadataURI *string `json:"metadataUri,omitempty"`
}

// ===============================
// Patch（表示用）
// ===============================
//
// Firebase Storage 移行後の正:
// - Patch は read-model 用の最小情報
// - iconUrl は Firebase Storage downloadURL
// - iconObjectPath は Firebase Storage objectPath
// - iconFileName / iconContentType / iconSize は表示・差し替え用に返す
// - metadataUri を含める
type Patch struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description"`
	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`

	IconURL         string `json:"iconUrl,omitempty"`
	IconObjectPath  string `json:"iconObjectPath,omitempty"`
	IconFileName    string `json:"iconFileName,omitempty"`
	IconContentType string `json:"iconContentType,omitempty"`
	IconSize        int64  `json:"iconSize,omitempty"`
}

// NewPatchFromTokenBlueprint builds a display Patch from TokenBlueprint.
// BrandName は caller 側の NameResolver / BrandRepo で補完する。
func NewPatchFromTokenBlueprint(tb *TokenBlueprint) Patch {
	if tb == nil {
		return Patch{}
	}

	return Patch{
		ID:          tb.ID,
		TokenName:   tb.Name,
		Symbol:      tb.Symbol,
		BrandID:     tb.BrandID,
		BrandName:   "",
		CompanyID:   tb.CompanyID,
		Description: tb.Description,
		Minted:      tb.Minted,
		MetadataURI: tb.MetadataURI,

		IconURL:         tb.IconURL,
		IconObjectPath:  tb.IconObjectPath,
		IconFileName:    tb.IconFileName,
		IconContentType: tb.IconContentType,
		IconSize:        tb.IconSize,
	}
}

// ===============================
// RepositoryPort（リポジトリ境界）
// ===============================
type RepositoryPort interface {
	// 単体取得
	//
	// Patch 表示用の最小情報や ID → Name 解決が必要な場合も、
	// 個別の read-model 専用メソッドを使わず GetByID の結果から組み立てる。
	GetByID(ctx context.Context, id string) (*TokenBlueprint, error)

	// companyId で限定した一覧
	ListByCompanyID(ctx context.Context, companyID string, page common.Page) (common.PageResult[TokenBlueprint], error)

	// brandId で限定した一覧
	ListByBrandID(ctx context.Context, brandID string, page common.Page) (common.PageResult[TokenBlueprint], error)

	// 作成・更新・削除
	Create(ctx context.Context, in CreateTokenBlueprintInput) (*TokenBlueprint, error)
	Update(ctx context.Context, id string, in UpdateTokenBlueprintInput) (*TokenBlueprint, error)
	Delete(ctx context.Context, id string) error

	// 一意性チェック
	IsSymbolUnique(ctx context.Context, symbol string, excludeID string) (bool, error)
	IsNameUnique(ctx context.Context, name string, excludeID string) (bool, error)
}

// ===============================
// CreateOperation Repository
// ===============================

var (
	ErrCreateOperationNotFound = errors.New(
		"tokenBlueprint create operation: not found",
	)

	ErrCreateOperationConflict = errors.New(
		"tokenBlueprint create operation: conflict",
	)

	ErrCreateOperationIdempotencyConflict = errors.New(
		"tokenBlueprint create operation: idempotency key conflict",
	)
)

// CreateOperationRepository persists TokenBlueprint create operations.
//
// TokenBlueprint 本体とは別の永続化境界として扱う。
//
// Idempotency policy:
//   - IdempotencyKey は一意でなければならない。
//   - 未使用の IdempotencyKey に対する Create は新しい Operation を作成する。
//   - 同一要求で同じ IdempotencyKey が再送された場合は既存 Operation を返してよい。
//   - 同じ IdempotencyKey が異なる要求に使用された場合は
//     ErrCreateOperationIdempotencyConflict を返す。
//
// Concurrency policy:
// - Update は楽観ロックを使用する。
// - expectedVersion は読み込み時点の Version と一致しなければならない。
// - Update 成功時は Version をインクリメントする。
// - Version が一致しない場合は ErrCreateOperationConflict を返す。
//
// Upload policy:
// - frontend が Firebase Storage へ直接 upload する。
// - icon / contents の upload 完了結果は CreateOperation に逐次記録する。
// - waiting_upload の間は frontend が処理主体となる。
// - queued 以降は Cloud Tasks / backend が処理主体となる。
type CreateOperationRepository interface {
	// Create は新しい waiting_upload Operation を永続化する。
	Create(
		ctx context.Context,
		operation CreateOperation,
	) (CreateOperation, error)

	// GetByID は Operation ID から CreateOperation を取得する。
	GetByID(
		ctx context.Context,
		operationID string,
	) (CreateOperation, error)

	// GetByIdempotencyKey は IdempotencyKey に対応する
	// CreateOperation を取得する。
	GetByIdempotencyKey(
		ctx context.Context,
		idempotencyKey string,
	) (CreateOperation, error)

	// Update は楽観ロックを使用して現在の Operation 状態を永続化する。
	//
	// expectedVersion は更新前に読み込んだ Version を指定する。
	// 更新成功時、返却される Operation の Version は
	// インクリメントされていなければならない。
	Update(
		ctx context.Context,
		operation CreateOperation,
		expectedVersion int64,
	) (CreateOperation, error)
}

// ===============================
// Helper Functions
// ===============================

// brandId ごとに一覧取得
func ListByBrandID(
	ctx context.Context,
	repo RepositoryPort,
	brandID string,
	page common.Page,
) (common.PageResult[TokenBlueprint], error) {
	return repo.ListByBrandID(ctx, brandID, page)
}
