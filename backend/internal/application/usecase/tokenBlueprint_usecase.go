// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"

	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
	tcdom "narratives/internal/domain/tokenContents"
	tidom "narratives/internal/domain/tokenIcon"
)

// ============================================================
// Config
// ============================================================

// ★ token icon 用の GCS バケット名（デフォルト）
// ※ 環境変数 TOKEN_ICON_BUCKET があればそれを優先
const defaultTokenIconBucket = "narratives-development_token_icon"

// ★ 署名に使うサービスアカウントメール（必須）
// 例: narratives-backend-sa@narratives-development-26c2d.iam.gserviceaccount.com
// ※ Cloud Run では自動で入らないことが多いので env で明示推奨
const envTokenIconSignerEmail = "TOKEN_ICON_SIGNER_EMAIL"

// ★ 署名付きURLの有効期限（PUT）
const tokenIconSignedURLTTL = 15 * time.Minute

func tokenIconBucketName() string {
	if v := strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET")); v != "" {
		return v
	}
	return defaultTokenIconBucket
}

// tokenIconSignerEmail returns signer service account email.
func tokenIconSignerEmail() string {
	if v := strings.TrimSpace(os.Getenv(envTokenIconSignerEmail)); v != "" {
		return v
	}
	// 互換用の別名も見ておく（任意）
	if v := strings.TrimSpace(os.Getenv("GCS_SIGNER_EMAIL")); v != "" {
		return v
	}
	return ""
}

// gcsObjectPublicURL returns public HTTPS URL for an object.
func gcsObjectPublicURL(bucket, object string) string {
	bucket = strings.TrimSpace(bucket)
	object = strings.TrimLeft(strings.TrimSpace(object), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object)
}

// tokenIconObjectPath is a stable object path under "{tokenBlueprintId}/".
// ★ 画像を後から差し替えても URL を固定化するため、ファイル名は常に "icon" に寄せる
func tokenIconObjectPath(tokenBlueprintID string) string {
	id := strings.Trim(strings.TrimSpace(tokenBlueprintID), "/")
	return id + "/icon"
}

// ============================================================
// Arweave / Metaplex metadata
// ============================================================

// ArweaveUploader は メタデータ JSON を Arweave にアップロードし、URI を返す責務を持つ。
type ArweaveUploader interface {
	UploadMetadata(ctx context.Context, data []byte) (string, error)
}

// TokenMetadataBuilder は TokenBlueprint から NFT メタデータ JSON を生成する責務を持ちます。
type TokenMetadataBuilder struct{}

// NewTokenMetadataBuilder はビルダーのコンストラクタです。
func NewTokenMetadataBuilder() *TokenMetadataBuilder { return &TokenMetadataBuilder{} }

// BuildFromBlueprint は TokenBlueprint から Arweave 用メタデータ JSON を生成します（互換用）。
func (b *TokenMetadataBuilder) BuildFromBlueprint(pb tbdom.TokenBlueprint) ([]byte, error) {
	return b.BuildFromBlueprintWithImage(pb, "")
}

// BuildFromBlueprintWithImage は TokenBlueprint から Arweave 用メタデータ JSON を生成します。
// ★ imageURL が非空なら Metaplex 形式の "image" に格納します
func (b *TokenMetadataBuilder) BuildFromBlueprintWithImage(pb tbdom.TokenBlueprint, imageURL string) ([]byte, error) {
	name := strings.TrimSpace(pb.Name)
	symbol := strings.TrimSpace(pb.Symbol)

	if name == "" || symbol == "" {
		return nil, fmt.Errorf("token blueprint name or symbol is empty")
	}

	metadata := map[string]interface{}{
		"name":   name,
		"symbol": symbol,
	}

	if desc := strings.TrimSpace(pb.Description); desc != "" {
		metadata["description"] = desc
	}

	if u := strings.TrimSpace(imageURL); u != "" {
		metadata["image"] = u
	}

	return json.Marshal(metadata)
}

// ============================================================
// Usecase
// ============================================================

// TokenBlueprintUsecase coordinates TokenBlueprint, TokenContents, and TokenIcon domains.
type TokenBlueprintUsecase struct {
	tbRepo tbdom.RepositoryPort
	tcRepo tcdom.RepositoryPort
	tiRepo tidom.RepositoryPort

	memberSvc *memdom.Service

	arweave         ArweaveUploader
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

// Upload DTOs（contents は従来通り）
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

// ============================================================
// Signed URL (Front PUT -> GCS)
// ============================================================

// TokenIconUploadURL is returned to front for direct PUT.
type TokenIconUploadURL struct {
	UploadURL  string     `json:"uploadUrl"`
	PublicURL  string     `json:"publicUrl"`
	ObjectPath string     `json:"objectPath"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// IssueTokenIconUploadURL issues V4 signed PUT URL for "{tokenBlueprintId}/icon".
//
// 必要条件:
// - env TOKEN_ICON_SIGNER_EMAIL に署名用SAメールを設定
// - Cloud Run 実行SAに iam.serviceAccounts.signBlob 権限（= IAMCredentials SignBlob が通る）
//
// 注意:
// - SignedURL に ContentType を含めるため、フロントの PUT の Content-Type は一致必須
func (u *TokenBlueprintUsecase) IssueTokenIconUploadURL(
	ctx context.Context,
	tokenBlueprintID string,
	fileName string, // 現状はログ・互換用（object名は固定 "icon"）
	contentType string, // 署名に含める。PUT時の Content-Type と一致必須
) (*TokenIconUploadURL, error) {

	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	// blueprint の存在確認（誤ったIDで勝手にアップロードURL発行しない）
	if _, err := u.tbRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	bucket := tokenIconBucketName()
	if bucket == "" {
		return nil, fmt.Errorf("token icon bucket is empty")
	}

	accessID := tokenIconSignerEmail()
	if accessID == "" {
		return nil, fmt.Errorf("missing %s env (signer service account email)", envTokenIconSignerEmail)
	}

	// object path は固定（後から差し替えても URL が変わらない）
	objectPath := tokenIconObjectPath(id)

	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = "application/octet-stream"
	}

	// IAM Credentials API による署名（秘密鍵不要）
	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create iamcredentials service: %w", err)
	}

	signBytes := func(b []byte) ([]byte, error) {
		name := "projects/-/serviceAccounts/" + accessID
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(b),
		}
		resp, err := iamSvc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(resp.SignedBlob)
	}

	expires := time.Now().UTC().Add(tokenIconSignedURLTTL)

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        expires,
		ContentType:    ct,
	})
	if err != nil {
		return nil, fmt.Errorf("sign gcs url: %w", err)
	}

	_ = fileName // 互換用（将来ログに使いたければここで使う）

	publicURL := gcsObjectPublicURL(bucket, objectPath)
	return &TokenIconUploadURL{
		UploadURL:  strings.TrimSpace(uploadURL),
		PublicURL:  strings.TrimSpace(publicURL),
		ObjectPath: strings.TrimSpace(objectPath),
		ExpiresAt:  ptr(expires),
	}, nil
}

// ============================================================
// Create
// ============================================================

type CreateBlueprintRequest struct {
	Name        string
	Symbol      string
	BrandID     string
	CompanyID   string
	Description string

	AssigneeID string
	CreatedBy  string
	ActorID    string

	// ★ 方針A: icon は create では受け取らない（署名付きURLでフロントがPUT）
	// 互換のため struct は残すが、CreateWithUploads では受け付けない
	Icon     *IconUpload
	Contents []ContentUpload
}

func (u *TokenBlueprintUsecase) CreateWithUploads(ctx context.Context, in CreateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil || u.tcRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/repo is nil")
	}

	// ★ 方針A: create での icon アップロードは廃止
	if in.Icon != nil {
		return nil, ErrNotSupported("CreateWithUploads with icon (use IssueTokenIconUploadURL + frontend PUT + Update(iconId))")
	}

	// 0) contents（tokenBlueprintId に依存しない）
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

	// 1) まず TokenBlueprint を作る（docId 確定）
	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:         strings.TrimSpace(in.Name),
		Symbol:       strings.TrimSpace(in.Symbol),
		BrandID:      strings.TrimSpace(in.BrandID),
		CompanyID:    strings.TrimSpace(in.CompanyID),
		Description:  strings.TrimSpace(in.Description),
		IconID:       nil, // ★ 画像は後から Update で objectPath をセット
		ContentFiles: contentIDs,
		AssigneeID:   strings.TrimSpace(in.AssigneeID),

		CreatedAt: nil,
		CreatedBy: strings.TrimSpace(in.CreatedBy),
		UpdatedAt: nil,
		UpdatedBy: "",
	})
	if err != nil {
		return nil, err
	}

	// 2) Arweave 連携（image には “将来アップロードされる予定のURL” を入れておく）
	//    ※ object が無い間は 404 だが、後で PUT すれば同URLで表示される
	if u.arweave == nil || u.metadataBuilder == nil {
		return tb, nil
	}

	bucket := tokenIconBucketName()
	imageURLForMetaplex := ""
	if bucket != "" && strings.TrimSpace(tb.ID) != "" {
		imageURLForMetaplex = gcsObjectPublicURL(bucket, tokenIconObjectPath(tb.ID))
	}

	metaJSON, err := u.metadataBuilder.BuildFromBlueprintWithImage(*tb, imageURLForMetaplex)
	if err != nil {
		return nil, fmt.Errorf("build metadata: %w", err)
	}

	uri, err := u.arweave.UploadMetadata(ctx, metaJSON)
	if err != nil {
		return nil, fmt.Errorf("upload metadata to arweave: %w", err)
	}

	updated, err := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
		MetadataURI: &uri,
	})
	if err != nil {
		return nil, fmt.Errorf("update token blueprint metadataUri: %w", err)
	}

	return updated, nil
}

// ============================================================
// Read
// ============================================================

func (u *TokenBlueprintUsecase) GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error) {
	tid := strings.TrimSpace(id)
	return u.tbRepo.GetByID(ctx, tid)
}

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
		return tb, "", nil
	}

	return tb, name, nil
}

func (u *TokenBlueprintUsecase) ListByCompanyID(ctx context.Context, companyID string, page tbdom.Page) (tbdom.PageResult, error) {
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return tbdom.PageResult{}, fmt.Errorf("companyId is empty")
	}
	return u.tbRepo.ListByCompanyID(ctx, cid, page)
}

func (u *TokenBlueprintUsecase) ListByBrandID(ctx context.Context, brandID string, page tbdom.Page) (tbdom.PageResult, error) {
	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return tbdom.PageResult{}, fmt.Errorf("brandId is empty")
	}
	return tbdom.ListByBrandID(ctx, u.tbRepo, bid, page)
}

func (u *TokenBlueprintUsecase) ListMintedNotYet(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	return tbdom.ListMintedNotYet(ctx, u.tbRepo, page)
}

func (u *TokenBlueprintUsecase) ListMintedCompleted(ctx context.Context, page tbdom.Page) (tbdom.PageResult, error) {
	return tbdom.ListMintedCompleted(ctx, u.tbRepo, page)
}

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
			result[id] = ""
			continue
		}

		result[id] = strings.TrimSpace(name)
	}

	return result, nil
}

// ============================================================
// Update
// ============================================================

type UpdateBlueprintRequest struct {
	ID           string
	Name         *string
	Symbol       *string
	BrandID      *string
	Description  *string
	AssigneeID   *string
	IconID       *string   // "" を渡すと NULL にする（現在の仕様維持）
	ContentFiles *[]string // 全置換
	ActorID      string
}

func normalizeIconIDForUpdate(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		empty := ""
		return &empty
	}
	return &v
}

func (u *TokenBlueprintUsecase) Update(ctx context.Context, in UpdateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(in.ID), tbdom.UpdateTokenBlueprintInput{
		Name:         trimPtr(in.Name),
		Symbol:       trimPtr(in.Symbol),
		BrandID:      trimPtr(in.BrandID),
		Description:  trimPtr(in.Description),
		IconID:       normalizeIconIDForUpdate(in.IconID),
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

// ============================================================
// Convenience helpers
// ============================================================

// 方針A: バックエンドでのアップロードは廃止（署名付きURLでフロントPUT）
func (u *TokenBlueprintUsecase) ReplaceIconWithUpload(ctx context.Context, blueprintID string, icon IconUpload, actorID string) (*tbdom.TokenBlueprint, error) {
	return nil, ErrNotSupported("ReplaceIconWithUpload (use signed URL PUT from frontend)")
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

// ============================================================
// Delete
// ============================================================

func (u *TokenBlueprintUsecase) Delete(ctx context.Context, id string) error {
	tid := strings.TrimSpace(id)
	return u.tbRepo.Delete(ctx, tid)
}

// ============================================================
// Additional API: TokenBlueprint minted 更新（移譲版）
// ============================================================

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

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if tb.Minted {
		return tb, nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

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
