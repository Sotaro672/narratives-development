// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
	tcdom "narratives/internal/domain/tokenContents"
	tidom "narratives/internal/domain/tokenIcon"
)

// ArweaveUploader は メタデータ JSON を Arweave にアップロードし、URI を返す責務を持つ。
type ArweaveUploader interface {
	// data: JSON バイト列
	// 戻り値: Arweave 上の URI (例: https://arweave.net/xxxx)
	UploadMetadata(ctx context.Context, data []byte) (string, error)
}

// TokenMetadataBuilder は TokenBlueprint から NFT メタデータ JSON を生成する責務を持ちます。
type TokenMetadataBuilder struct{}

// NewTokenMetadataBuilder はビルダーのコンストラクタです。
func NewTokenMetadataBuilder() *TokenMetadataBuilder {
	return &TokenMetadataBuilder{}
}

// BuildFromBlueprint は TokenBlueprint から Arweave 用メタデータ JSON を生成します。
func (b *TokenMetadataBuilder) BuildFromBlueprint(pb tbdom.TokenBlueprint) ([]byte, error) {
	name := strings.TrimSpace(pb.Name)
	symbol := strings.TrimSpace(pb.Symbol)

	if name == "" || symbol == "" {
		return nil, fmt.Errorf("token blueprint name or symbol is empty")
	}

	metadata := map[string]interface{}{
		"name":   name,
		"symbol": symbol,
	}

	// description フィールドが存在する場合だけ追加
	if desc := strings.TrimSpace(pb.Description); desc != "" {
		metadata["description"] = desc
	}

	// アイコンや画像 URL などを追加したくなったらここに追記するイメージ:
	// if pb.IconURL != "" {
	//     metadata["image"] = pb.IconURL
	// }

	return json.Marshal(metadata)
}

// TokenBlueprintUsecase coordinates TokenBlueprint, TokenContents, and TokenIcon domains.
type TokenBlueprintUsecase struct {
	tbRepo tbdom.RepositoryPort
	tcRepo tcdom.RepositoryPort
	tiRepo tidom.RepositoryPort

	memberSvc *memdom.Service

	// ★ Arweave 連携用
	arweave ArweaveUploader

	// ★ TokenBlueprint → NFT メタデータ JSON 生成用
	metadataBuilder *TokenMetadataBuilder
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	tcRepo tcdom.RepositoryPort,
	tiRepo tidom.RepositoryPort,
	memberSvc *memdom.Service,
	arweave ArweaveUploader,
	metadataBuilder *TokenMetadataBuilder,
) *TokenBlueprintUsecase {

	return &TokenBlueprintUsecase{
		tbRepo:          tbRepo,
		tcRepo:          tcRepo,
		tiRepo:          tiRepo,
		memberSvc:       memberSvc,
		arweave:         arweave,
		metadataBuilder: metadataBuilder,
	}
}

// Upload DTOs

type IconUpload struct {
	FileName    string
	ContentType string
	Reader      io.Reader
}

type ContentUpload struct {
	Name        string
	Type        tcdom.ContentType
	FileName    string
	ContentType string
	Reader      io.Reader
}

// Create

type CreateBlueprintRequest struct {
	Name        string
	Symbol      string
	BrandID     string
	CompanyID   string // テナント
	Description string

	AssigneeID string
	CreatedBy  string // 作成者（memberId）
	ActorID    string // 操作者（更新者・監査用）

	Icon     *IconUpload
	Contents []ContentUpload
}

func (u *TokenBlueprintUsecase) CreateWithUploads(ctx context.Context, in CreateBlueprintRequest) (*tbdom.TokenBlueprint, error) {

	var iconIDPtr *string
	if in.Icon != nil {
		iconURL, size, err := u.tiRepo.UploadIcon(ctx, in.Icon.FileName, in.Icon.ContentType, in.Icon.Reader)
		if err != nil {
			return nil, fmt.Errorf("upload icon: %w", err)
		}

		icon, err := u.tiRepo.Create(ctx, tidom.CreateTokenIconInput{
			URL:      strings.TrimSpace(iconURL),
			FileName: strings.TrimSpace(in.Icon.FileName),
			Size:     size,
		})
		if err != nil {
			return nil, fmt.Errorf("create token icon: %w", err)
		}
		iconID := strings.TrimSpace(icon.ID)
		if iconID != "" {
			iconIDPtr = &iconID
		}
	}

	contentIDs := make([]string, 0, len(in.Contents))
	for _, c := range in.Contents {
		url, size, err := u.tcRepo.UploadContent(ctx, c.FileName, c.ContentType, c.Reader)
		if err != nil {
			return nil, fmt.Errorf("upload content(%s): %w", c.FileName, err)
		}

		tc, err := u.tcRepo.Create(ctx, tcdom.CreateTokenContentInput{
			Name: strings.TrimSpace(c.Name),
			Type: c.Type,
			URL:  strings.TrimSpace(url),
			Size: size,
		})
		if err != nil {
			return nil, fmt.Errorf("create token content(%s): %w", c.Name, err)
		}
		if id := strings.TrimSpace(tc.ID); id != "" {
			contentIDs = append(contentIDs, id)
		}
	}
	contentIDs = dedupStrings(contentIDs)

	// ★ minted は create 時は必ず false（domain/Repository 側で固定される）
	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:         strings.TrimSpace(in.Name),
		Symbol:       strings.TrimSpace(in.Symbol),
		BrandID:      strings.TrimSpace(in.BrandID),
		CompanyID:    strings.TrimSpace(in.CompanyID),
		Description:  strings.TrimSpace(in.Description),
		IconID:       iconIDPtr,
		ContentFiles: contentIDs,
		AssigneeID:   strings.TrimSpace(in.AssigneeID),

		// ★ 作成時は UpdatedAt / UpdatedBy は入力しない（nil / 空文字）
		CreatedAt: nil,
		CreatedBy: strings.TrimSpace(in.CreatedBy),
		UpdatedAt: nil,
		UpdatedBy: "",
	})
	if err != nil {
		return nil, err
	}

	// ─────────────────────────────────────────────
	// ここから Arweave 連携（自動 Publish）
	// ─────────────────────────────────────────────
	if u.arweave == nil || u.metadataBuilder == nil {
		return tb, nil
	}

	// 1) メタデータ JSON を生成
	metaJSON, err := u.metadataBuilder.BuildFromBlueprint(*tb)
	if err != nil {
		return nil, fmt.Errorf("build metadata: %w", err)
	}

	// 2) Arweave にアップロード
	uri, err := u.arweave.UploadMetadata(ctx, metaJSON)
	if err != nil {
		return nil, fmt.Errorf("upload metadata to arweave: %w", err)
	}

	// 3) metadataUri を更新
	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
	})
	if err != nil {
		return nil, fmt.Errorf("update token blueprint metadataUri: %w", err)
	}

	return updated, nil
}

// Read

func (u *TokenBlueprintUsecase) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	tid := strings.TrimSpace(id)
	return u.tbRepo.GetByID(ctx, tid)
}

// ★ createdBy を氏名に解決して返す補助メソッド
//   - 戻り値: (TokenBlueprint, createdByName, error)
//   - memberSvc が未設定 or 解決失敗時は createdByName は空文字
func (u *TokenBlueprintUsecase) GetByIDWithCreatorName(
	ctx context.Context,
	id string,
) (*tbdom.TokenBlueprint, string, error) {
	tid := strings.TrimSpace(id)

	tb, err := u.tbRepo.GetByID(ctx, tid)
	if err != nil {
		return nil, "", err
	}

	if u.memberSvc == nil {
		return tb, "", nil
	}

	memberID := strings.TrimSpace(tb.CreatedBy)
	if memberID == "" {
		return tb, "", nil
	}

	name, err := u.memberSvc.GetNameLastFirstByID(ctx, memberID)
	if err != nil {
		// 氏名解決に失敗してもエラーにはせず、TokenBlueprint 自体は返す
		return tb, "", nil
	}

	return tb, name, nil
}

// companyID で tenant-scoped 一覧取得
func (u *TokenBlueprintUsecase) ListByCompanyID(ctx context.Context, companyID string, page tbdom.Page) (tbdom.PageResult, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return tbdom.PageResult{}, fmt.Errorf("companyId is empty")
	}

	return u.tbRepo.ListByCompanyID(ctx, cid, page)
}

// ★ brandId 単位での一覧取得（domain のヘルパーを利用）
func (u *TokenBlueprintUsecase) ListByBrandID(ctx context.Context, brandID string, page tbdom.Page) (tbdom.PageResult, error) {
	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return tbdom.PageResult{}, fmt.Errorf("brandId is empty")
	}
	return tbdom.ListByBrandID(ctx, u.tbRepo, bid, page)
}

// ★ minted = false のみの一覧取得
func (u *TokenBlueprintUsecase) ListMintedNotYet(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	return tbdom.ListMintedNotYet(ctx, u.tbRepo, page)
}

// ★ minted = true のみの一覧取得
func (u *TokenBlueprintUsecase) ListMintedCompleted(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	return tbdom.ListMintedCompleted(ctx, u.tbRepo, page)
}

// ==== ★ ID → Name をまとめて解決する便利関数 ====
func (u *TokenBlueprintUsecase) ResolveNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {

	result := make(map[string]string, len(ids))

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		name, err := u.tbRepo.GetNameByID(ctx, id)
		if err != nil {
			// NotFound → 空文字
			result[id] = ""
			continue
		}

		result[id] = strings.TrimSpace(name)
	}

	return result, nil
}

// Update

type UpdateBlueprintRequest struct {
	ID           string
	Name         *string
	Symbol       *string
	BrandID      *string
	Description  *string
	AssigneeID   *string
	IconID       *string   // "" を渡すと NULL にする
	ContentFiles *[]string // 全置換
	ActorID      string
}

func (u *TokenBlueprintUsecase) Update(ctx context.Context, in UpdateBlueprintRequest) (*tbdom.TokenBlueprint, error) {

	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(in.ID), tbdom.UpdateTokenBlueprintInput{
		Name:         trimPtr(in.Name),
		Symbol:       trimPtr(in.Symbol),
		BrandID:      trimPtr(in.BrandID),
		Description:  trimPtr(in.Description),
		IconID:       normalizeEmptyToNil(in.IconID),
		ContentFiles: normalizeSlicePtr(in.ContentFiles),
		AssigneeID:   trimPtr(in.AssigneeID),

		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(in.ActorID)),
		DeletedAt: nil,
		DeletedBy: nil,
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

// Convenient helpers

func (u *TokenBlueprintUsecase) ReplaceIconWithUpload(ctx context.Context, blueprintID string, icon IconUpload, actorID string) (*tbdom.TokenBlueprint, error) {

	url, size, err := u.tiRepo.UploadIcon(ctx, icon.FileName, icon.ContentType, icon.Reader)
	if err != nil {
		return nil, fmt.Errorf("upload icon: %w", err)
	}
	ti, err := u.tiRepo.Create(ctx, tidom.CreateTokenIconInput{
		URL:      strings.TrimSpace(url),
		FileName: strings.TrimSpace(icon.FileName),
		Size:     size,
	})
	if err != nil {
		return nil, fmt.Errorf("create token icon: %w", err)
	}
	iconID := strings.TrimSpace(ti.ID)
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		IconID:    &iconID,
		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		return nil, err
	}
	return tb, nil
}

func (u *TokenBlueprintUsecase) AddContentsWithUploads(ctx context.Context, blueprintID string, uploads []ContentUpload, actorID string) (*tbdom.TokenBlueprint, error) {

	if len(uploads) == 0 {
		return u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	}

	ids := make([]string, 0, len(uploads))
	for _, up := range uploads {
		url, size, err := u.tcRepo.UploadContent(ctx, up.FileName, up.ContentType, up.Reader)
		if err != nil {
			return nil, fmt.Errorf("upload content(%s): %w", up.FileName, err)
		}
		tc, err := u.tcRepo.Create(ctx, tcdom.CreateTokenContentInput{
			Name: strings.TrimSpace(up.Name),
			Type: up.Type,
			URL:  strings.TrimSpace(url),
			Size: size,
		})
		if err != nil {
			return nil, fmt.Errorf("create token content(%s): %w", up.Name, err)
		}
		if id := strings.TrimSpace(tc.ID); id != "" {
			ids = append(ids, id)
		}
	}

	current, err := u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	if err != nil {
		return nil, err
	}

	merged := append([]string{}, current.ContentFiles...)
	merged = append(merged, ids...)
	merged = dedupStrings(merged)

	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &merged,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		return nil, err
	}

	return tb, nil
}

func (u *TokenBlueprintUsecase) ClearIcon(ctx context.Context, blueprintID string, actorID string) (*tbdom.TokenBlueprint, error) {
	empty := ""
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		IconID:    &empty,
		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		return nil, err
	}
	return tb, nil
}

func (u *TokenBlueprintUsecase) ReplaceContentIDs(ctx context.Context, blueprintID string, contentIDs []string, actorID string) (*tbdom.TokenBlueprint, error) {

	clean := dedupStrings(contentIDs)
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &clean,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		return nil, err
	}
	return tb, nil
}

// Delete

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	tid := strings.TrimSpace(id)
	err := u.tbRepo.Delete(ctx, tid)
	if err != nil {
		return err
	}
	return nil
}

// ============================================================
// Additional API: TokenBlueprint minted 更新（移譲版）
// ============================================================
//
// MarkTokenBlueprintMinted は、指定された tokenBlueprintId の minted を
// false（notYet） → true（minted） に更新する usecase です。
func (u *TokenBlueprintUsecase) MarkTokenBlueprintMinted(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {

	if u == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase is nil")
	}
	if u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	// 現状を取得
	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// すでに minted=true なら冪等的にそのまま返す
	if tb.Minted {
		return tb, nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

	// RepositoryPort.Update 経由で Firestore に minted / updatedAt / updatedBy を反映
	updated, err := u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		Description:  nil,
		IconID:       nil,
		ContentFiles: nil,
		AssigneeID:   nil,

		Minted: &minted,

		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
		DeletedAt: nil,
		DeletedBy: nil,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}
