// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	log.Printf("[token-blueprint] NewTokenBlueprintUsecase init (arweave=%v, metadataBuilder=%v)", arweave != nil, metadataBuilder != nil)

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
	log.Printf("[token-blueprint] CreateWithUploads start name=%s symbol=%s brandId=%s companyId=%s contents=%d",
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Symbol),
		strings.TrimSpace(in.BrandID),
		strings.TrimSpace(in.CompanyID),
		len(in.Contents),
	)

	var iconIDPtr *string
	if in.Icon != nil {
		log.Printf("[token-blueprint] uploading icon fileName=%s contentType=%s", in.Icon.FileName, in.Icon.ContentType)
		iconURL, size, err := u.tiRepo.UploadIcon(ctx, in.Icon.FileName, in.Icon.ContentType, in.Icon.Reader)
		if err != nil {
			log.Printf("[token-blueprint] upload icon FAILED fileName=%s err=%v", in.Icon.FileName, err)
			return nil, fmt.Errorf("upload icon: %w", err)
		}
		log.Printf("[token-blueprint] upload icon OK url=%s size=%d", iconURL, size)

		icon, err := u.tiRepo.Create(ctx, tidom.CreateTokenIconInput{
			URL:      strings.TrimSpace(iconURL),
			FileName: strings.TrimSpace(in.Icon.FileName),
			Size:     size,
		})
		if err != nil {
			log.Printf("[token-blueprint] create token icon FAILED fileName=%s err=%v", in.Icon.FileName, err)
			return nil, fmt.Errorf("create token icon: %w", err)
		}
		iconID := strings.TrimSpace(icon.ID)
		log.Printf("[token-blueprint] create token icon OK id=%s", iconID)
		if iconID != "" {
			iconIDPtr = &iconID
		}
	}

	contentIDs := make([]string, 0, len(in.Contents))
	for _, c := range in.Contents {
		log.Printf("[token-blueprint] uploading content name=%s fileName=%s type=%s", c.Name, c.FileName, c.Type)
		url, size, err := u.tcRepo.UploadContent(ctx, c.FileName, c.ContentType, c.Reader)
		if err != nil {
			log.Printf("[token-blueprint] upload content FAILED fileName=%s err=%v", c.FileName, err)
			return nil, fmt.Errorf("upload content(%s): %w", c.FileName, err)
		}
		log.Printf("[token-blueprint] upload content OK fileName=%s url=%s size=%d", c.FileName, url, size)

		tc, err := u.tcRepo.Create(ctx, tcdom.CreateTokenContentInput{
			Name: strings.TrimSpace(c.Name),
			Type: c.Type,
			URL:  strings.TrimSpace(url),
			Size: size,
		})
		if err != nil {
			log.Printf("[token-blueprint] create token content FAILED name=%s err=%v", c.Name, err)
			return nil, fmt.Errorf("create token content(%s): %w", c.Name, err)
		}
		if id := strings.TrimSpace(tc.ID); id != "" {
			log.Printf("[token-blueprint] create token content OK id=%s name=%s", id, c.Name)
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
		log.Printf("[token-blueprint] CreateWithUploads FAILED name=%s symbol=%s err=%v", in.Name, in.Symbol, err)
		return nil, err
	}

	log.Printf("[token-blueprint] CreateWithUploads OK id=%s name=%s symbol=%s", tb.ID, tb.Name, tb.Symbol)

	// ─────────────────────────────────────────────
	// ここから Arweave 連携（自動 Publish）
	// ─────────────────────────────────────────────
	if u.arweave == nil || u.metadataBuilder == nil {
		log.Printf("[token-blueprint] Arweave publish SKIP id=%s (arweave=%v, metadataBuilder=%v)",
			tb.ID, u.arweave != nil, u.metadataBuilder != nil)
		return tb, nil
	}

	log.Printf("[token-blueprint] Arweave publish START id=%s", tb.ID)

	// 1) メタデータ JSON を生成
	metaJSON, err := u.metadataBuilder.BuildFromBlueprint(*tb)
	if err != nil {
		log.Printf("[token-blueprint] Arweave build metadata FAILED id=%s err=%v", tb.ID, err)
		return nil, fmt.Errorf("build metadata: %w", err)
	}
	log.Printf("[token-blueprint] Arweave build metadata OK id=%s size=%d", tb.ID, len(metaJSON))

	// 2) Arweave にアップロード
	log.Printf("[token-blueprint] Arweave upload START id=%s", tb.ID)
	uri, err := u.arweave.UploadMetadata(ctx, metaJSON)
	if err != nil {
		log.Printf("[token-blueprint] Arweave upload FAILED id=%s err=%v", tb.ID, err)
		return nil, fmt.Errorf("upload metadata to arweave: %w", err)
	}
	log.Printf("[token-blueprint] Arweave upload OK id=%s uri=%s", tb.ID, uri)

	// 3) metadataUri を更新
	log.Printf("[token-blueprint] Arweave update metadataUri START id=%s", tb.ID)
	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
	})
	if err != nil {
		log.Printf("[token-blueprint] Arweave update metadataUri FAILED id=%s err=%v", tb.ID, err)
		return nil, fmt.Errorf("update token blueprint metadataUri: %w", err)
	}
	log.Printf("[token-blueprint] Arweave publish OK id=%s uri=%s", updated.ID, uri)

	return updated, nil
}

// Read

func (u *TokenBlueprintUsecase) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	tid := strings.TrimSpace(id)
	log.Printf("[token-blueprint] GetByID id=%s", tid)
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
	log.Printf("[token-blueprint] GetByIDWithCreatorName id=%s", tid)

	tb, err := u.tbRepo.GetByID(ctx, tid)
	if err != nil {
		log.Printf("[token-blueprint] GetByIDWithCreatorName FAILED id=%s err=%v", tid, err)
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
		log.Printf("[token-blueprint] resolve creator name FAILED memberId=%s err=%v", memberID, err)
		return tb, "", nil
	}

	log.Printf("[token-blueprint] resolve creator name OK memberId=%s name=%s", memberID, name)
	return tb, name, nil
}

// companyID で tenant-scoped 一覧取得
func (u *TokenBlueprintUsecase) ListByCompanyID(ctx context.Context, companyID string, page tbdom.Page) (tbdom.PageResult, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return tbdom.PageResult{}, fmt.Errorf("companyId is empty")
	}

	log.Printf("[token-blueprint] ListByCompanyID companyId=%s page=%d perPage=%d", cid, page.Number, page.PerPage)
	return u.tbRepo.ListByCompanyID(ctx, cid, page)
}

// ★ brandId 単位での一覧取得（domain のヘルパーを利用）
func (u *TokenBlueprintUsecase) ListByBrandID(ctx context.Context, brandID string, page tbdom.Page) (tbdom.PageResult, error) {
	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return tbdom.PageResult{}, fmt.Errorf("brandId is empty")
	}
	log.Printf("[token-blueprint] ListByBrandID brandId=%s page=%d perPage=%d", bid, page.Number, page.PerPage)
	return tbdom.ListByBrandID(ctx, u.tbRepo, bid, page)
}

// ★ minted = false のみの一覧取得
func (u *TokenBlueprintUsecase) ListMintedNotYet(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	log.Printf("[token-blueprint] ListMintedNotYet page=%d perPage=%d", page.Number, page.PerPage)
	return tbdom.ListMintedNotYet(ctx, u.tbRepo, page)
}

// ★ minted = true のみの一覧取得
func (u *TokenBlueprintUsecase) ListMintedCompleted(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	log.Printf("[token-blueprint] ListMintedCompleted page=%d perPage=%d", page.Number, page.PerPage)
	return tbdom.ListMintedCompleted(ctx, u.tbRepo, page)
}

// ==== ★ ID → Name をまとめて解決する便利関数 ====
func (u *TokenBlueprintUsecase) ResolveNames(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {

	log.Printf("[token-blueprint] ResolveNames ids=%v", ids)

	result := make(map[string]string, len(ids))

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		name, err := u.tbRepo.GetNameByID(ctx, id)
		if err != nil {
			// NotFound → 空文字
			log.Printf("[token-blueprint] ResolveNames GetNameByID FAILED id=%s err=%v", id, err)
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
	log.Printf("[token-blueprint] Update id=%s actorId=%s", strings.TrimSpace(in.ID), strings.TrimSpace(in.ActorID))

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
		log.Printf("[token-blueprint] Update FAILED id=%s err=%v", in.ID, err)
		return nil, err
	}

	log.Printf("[token-blueprint] Update OK id=%s", tb.ID)
	return tb, nil
}

// Convenient helpers

func (u *TokenBlueprintUsecase) ReplaceIconWithUpload(ctx context.Context, blueprintID string, icon IconUpload, actorID string) (*tbdom.TokenBlueprint, error) {
	log.Printf("[token-blueprint] ReplaceIconWithUpload blueprintId=%s fileName=%s", blueprintID, icon.FileName)

	url, size, err := u.tiRepo.UploadIcon(ctx, icon.FileName, icon.ContentType, icon.Reader)
	if err != nil {
		log.Printf("[token-blueprint] ReplaceIconWithUpload upload icon FAILED id=%s err=%v", blueprintID, err)
		return nil, fmt.Errorf("upload icon: %w", err)
	}
	ti, err := u.tiRepo.Create(ctx, tidom.CreateTokenIconInput{
		URL:      strings.TrimSpace(url),
		FileName: strings.TrimSpace(icon.FileName),
		Size:     size,
	})
	if err != nil {
		log.Printf("[token-blueprint] ReplaceIconWithUpload create icon FAILED id=%s err=%v", blueprintID, err)
		return nil, fmt.Errorf("create token icon: %w", err)
	}
	iconID := strings.TrimSpace(ti.ID)
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		IconID:    &iconID,
		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		log.Printf("[token-blueprint] ReplaceIconWithUpload update FAILED id=%s err=%v", blueprintID, err)
		return nil, err
	}
	log.Printf("[token-blueprint] ReplaceIconWithUpload OK blueprintId=%s iconId=%s", blueprintID, iconID)
	return tb, nil
}

func (u *TokenBlueprintUsecase) AddContentsWithUploads(ctx context.Context, blueprintID string, uploads []ContentUpload, actorID string) (*tbdom.TokenBlueprint, error) {
	log.Printf("[token-blueprint] AddContentsWithUploads blueprintId=%s uploads=%d", blueprintID, len(uploads))

	if len(uploads) == 0 {
		return u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	}

	ids := make([]string, 0, len(uploads))
	for _, up := range uploads {
		log.Printf("[token-blueprint] AddContentsWithUploads uploading name=%s fileName=%s type=%s", up.Name, up.FileName, up.Type)
		url, size, err := u.tcRepo.UploadContent(ctx, up.FileName, up.ContentType, up.Reader)
		if err != nil {
			log.Printf("[token-blueprint] AddContentsWithUploads upload FAILED fileName=%s err=%v", up.FileName, err)
			return nil, fmt.Errorf("upload content(%s): %w", up.FileName, err)
		}
		tc, err := u.tcRepo.Create(ctx, tcdom.CreateTokenContentInput{
			Name: strings.TrimSpace(up.Name),
			Type: up.Type,
			URL:  strings.TrimSpace(url),
			Size: size,
		})
		if err != nil {
			log.Printf("[token-blueprint] AddContentsWithUploads create FAILED name=%s err=%v", up.Name, err)
			return nil, fmt.Errorf("create token content(%s): %w", up.Name, err)
		}
		if id := strings.TrimSpace(tc.ID); id != "" {
			log.Printf("[token-blueprint] AddContentsWithUploads create OK id=%s name=%s", id, up.Name)
			ids = append(ids, id)
		}
	}

	current, err := u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	if err != nil {
		log.Printf("[token-blueprint] AddContentsWithUploads get current FAILED id=%s err=%v", blueprintID, err)
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
		log.Printf("[token-blueprint] AddContentsWithUploads update FAILED id=%s err=%v", blueprintID, err)
		return nil, err
	}

	log.Printf("[token-blueprint] AddContentsWithUploads OK blueprintId=%s added=%d total=%d", blueprintID, len(ids), len(tb.ContentFiles))
	return tb, nil
}

func (u *TokenBlueprintUsecase) ClearIcon(ctx context.Context, blueprintID string, actorID string) (*tbdom.TokenBlueprint, error) {
	log.Printf("[token-blueprint] ClearIcon blueprintId=%s", blueprintID)
	empty := ""
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		IconID:    &empty,
		UpdatedAt: nil,
		UpdatedBy: ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		log.Printf("[token-blueprint] ClearIcon FAILED id=%s err=%v", blueprintID, err)
		return nil, err
	}
	log.Printf("[token-blueprint] ClearIcon OK blueprintId=%s", blueprintID)
	return tb, nil
}

func (u *TokenBlueprintUsecase) ReplaceContentIDs(ctx context.Context, blueprintID string, contentIDs []string, actorID string) (*tbdom.TokenBlueprint, error) {
	log.Printf("[token-blueprint] ReplaceContentIDs blueprintId=%s newIds=%v", blueprintID, contentIDs)

	clean := dedupStrings(contentIDs)
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &clean,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
	})
	if err != nil {
		log.Printf("[token-blueprint] ReplaceContentIDs FAILED id=%s err=%v", blueprintID, err)
		return nil, err
	}
	log.Printf("[token-blueprint] ReplaceContentIDs OK blueprintId=%s total=%d", blueprintID, len(tb.ContentFiles))
	return tb, nil
}

// Delete

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	tid := strings.TrimSpace(id)
	log.Printf("[token-blueprint] Delete id=%s", tid)
	err := u.tbRepo.Delete(ctx, tid)
	if err != nil {
		log.Printf("[token-blueprint] Delete FAILED id=%s err=%v", tid, err)
		return err
	}
	log.Printf("[token-blueprint] Delete OK id=%s", tid)
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

	log.Printf("[token-blueprint] MarkTokenBlueprintMinted start id=%s actorId=%s", id, actorID)

	// 現状を取得
	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[token-blueprint] MarkTokenBlueprintMinted get FAILED id=%s err=%v", id, err)
		return nil, err
	}

	// すでに minted=true なら冪等的にそのまま返す
	if tb.Minted {
		log.Printf("[token-blueprint] MarkTokenBlueprintMinted noop id=%s already minted", id)
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
		log.Printf("[token-blueprint] MarkTokenBlueprintMinted update FAILED id=%s err=%v", id, err)
		return nil, err
	}

	log.Printf("[token-blueprint] MarkTokenBlueprintMinted OK id=%s", id)
	return updated, nil
}
