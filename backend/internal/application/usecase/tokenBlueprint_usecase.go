// backend/internal/application/usecase/tokenBlueprint_usecase.go
package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"

	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Config
// ============================================================

// token icon 用の GCS バケット名（デフォルト）
// ※ 環境変数 TOKEN_ICON_BUCKET があればそれを優先
const defaultTokenIconBucket = "narratives-development_token_icon"

// 署名に使うサービスアカウントメール（必須）
// 例: narratives-backend-sa@narratives-development-26c2d.iam.gserviceaccount.com
// ※ Cloud Run では env で明示推奨
const envTokenIconSignerEmail = "TOKEN_ICON_SIGNER_EMAIL"

// 署名付きURLの有効期限（PUT）
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
	return ""
}

// gcsObjectPublicURL returns public HTTPS URL for an object.
func gcsObjectPublicURL(bucket, object string) string {
	bucket = strings.TrimSpace(bucket)
	object = strings.TrimLeft(strings.TrimSpace(object), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object)
}

// tokenIconObjectPath is a stable object path under "{tokenBlueprintId}/".
// 画像を後から差し替えても URL を固定化するため、ファイル名は常に "icon" に寄せる
func tokenIconObjectPath(tokenBlueprintID string) string {
	id := strings.Trim(strings.TrimSpace(tokenBlueprintID), "/")
	return id + "/icon"
}

// ============================================================
// Usecase
// ============================================================

// TokenBlueprintUsecase coordinates TokenBlueprint domain only.
// Current policy:
// - Firestore persists only tokenBlueprint aggregate (embedded contents).
// - icon is stored in GCS at "{docId}/icon" and is NOT persisted in tokenBlueprint.
type TokenBlueprintUsecase struct {
	tbRepo tbdom.RepositoryPort

	memberSvc *memdom.Service
}

func NewTokenBlueprintUsecase(
	tbRepo tbdom.RepositoryPort,
	memberSvc *memdom.Service,
) *TokenBlueprintUsecase {
	return &TokenBlueprintUsecase{
		tbRepo:    tbRepo,
		memberSvc: memberSvc,
	}
}

// ============================================================
// Signed URL (Front PUT -> GCS) : token_icon
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
// Required:
// - env TOKEN_ICON_SIGNER_EMAIL set
// - Cloud Run runtime SA has iam.serviceAccounts.signBlob
//
// Note:
// - SignedURL includes ContentType; frontend PUT must match.
func (u *TokenBlueprintUsecase) IssueTokenIconUploadURL(
	ctx context.Context,
	tokenBlueprintID string,
	_ string, // fileName: not persisted; kept only to match handler signature
	contentType string,
) (*TokenIconUploadURL, error) {

	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	// ensure blueprint exists
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

	objectPath := tokenIconObjectPath(id)

	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = "application/octet-stream"
	}

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
}

func (u *TokenBlueprintUsecase) Create(ctx context.Context, in CreateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/repo is nil")
	}

	tb, err := u.tbRepo.Create(ctx, tbdom.CreateTokenBlueprintInput{
		Name:        strings.TrimSpace(in.Name),
		Symbol:      strings.TrimSpace(in.Symbol),
		BrandID:     strings.TrimSpace(in.BrandID),
		CompanyID:   strings.TrimSpace(in.CompanyID),
		Description: strings.TrimSpace(in.Description),

		// entity.go 正: embedded contents
		ContentFiles: nil,

		AssigneeID: strings.TrimSpace(in.AssigneeID),

		CreatedAt: nil,
		CreatedBy: strings.TrimSpace(in.CreatedBy),
		UpdatedAt: nil,
		UpdatedBy: "",
		// metadataUri は resolver URL を後でセット（or repo 側のデフォルトでも可）
		MetadataURI: "",
	})
	if err != nil {
		return nil, err
	}

	// metadataUri の解決URL方針（推奨）
	// repo 側が既に metadataUri を組み立てるならここは不要だが、
	// entity.go 正として "backend resolver URL" を持つので、
	// 現状 repo が空で保存するならここで埋める。
	if strings.TrimSpace(tb.MetadataURI) == "" {
		uri := buildMetadataResolverURL(tb.ID)
		updated, uerr := u.tbRepo.Update(ctx, tb.ID, tbdom.UpdateTokenBlueprintInput{
			MetadataURI: &uri,
			UpdatedAt:   nil,
			UpdatedBy:   ptr(strings.TrimSpace(in.ActorID)),
			DeletedAt:   nil,
			DeletedBy:   nil,
		})
		if uerr == nil && updated != nil {
			return updated, nil
		}
		// Update に失敗しても create 自体は成功しているため、作成結果を返す（運用で検知）
	}

	return tb, nil
}

// buildMetadataResolverURL builds resolver URL for metadataUri.
// NOTE: base URL は環境差分が大きいので env で明示する。
func buildMetadataResolverURL(tokenBlueprintID string) string {
	base := strings.TrimSpace(os.Getenv("TOKEN_METADATA_BASE_URL"))
	if base == "" {
		// 例: "https://api.example.com" のように設定する想定。
		// 未設定の場合は空のままにする（validate では必須にしない方針）。
		return ""
	}
	base = strings.TrimSuffix(base, "/")
	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("%s/v1/token-blueprints/%s/metadata", base, id)
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
	ID          string
	Name        *string
	Symbol      *string
	BrandID     *string
	Description *string
	AssigneeID  *string

	// entity.go 正: embedded
	ContentFiles *[]tbdom.ContentFile // 全置換

	ActorID string
}

func (u *TokenBlueprintUsecase) Update(ctx context.Context, in UpdateBlueprintRequest) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/repo is nil")
	}

	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(in.ID), tbdom.UpdateTokenBlueprintInput{
		Name:        trimPtr(in.Name),
		Symbol:      trimPtr(in.Symbol),
		BrandID:     trimPtr(in.BrandID),
		Description: trimPtr(in.Description),
		AssigneeID:  trimPtr(in.AssigneeID),

		ContentFiles: normalizeContentFilesPtr(in.ContentFiles),

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
// Convenience helpers (contents)
// ============================================================

// ReplaceContentFiles replaces all embedded contents.
func (u *TokenBlueprintUsecase) ReplaceContentFiles(ctx context.Context, blueprintID string, files []tbdom.ContentFile, actorID string) (*tbdom.TokenBlueprint, error) {
	clean := dedupAndValidateContentFiles(files)
	if clean == nil {
		empty := make([]tbdom.ContentFile, 0)
		clean = empty
	}

	tb, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &clean,
		UpdatedAt:    nil,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}
	return tb, nil
}

// SetContentVisibility updates visibility for a specific contentId (delegates to domain method).
func (u *TokenBlueprintUsecase) SetContentVisibility(ctx context.Context, blueprintID string, contentID string, v tbdom.ContentVisibility, actorID string) (*tbdom.TokenBlueprint, error) {
	if u == nil || u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint usecase/repo is nil")
	}

	tb, err := u.tbRepo.GetByID(ctx, strings.TrimSpace(blueprintID))
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	now := time.Now().UTC()
	if err := tb.SetContentVisibility(contentID, v, actorID, now); err != nil {
		return nil, err
	}

	files := tb.ContentFiles
	updated, err := u.tbRepo.Update(ctx, strings.TrimSpace(blueprintID), tbdom.UpdateTokenBlueprintInput{
		ContentFiles: &files,
		UpdatedAt:    &now,
		UpdatedBy:    ptr(strings.TrimSpace(actorID)),
		DeletedAt:    nil,
		DeletedBy:    nil,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
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
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	if tb.Minted {
		return tb, nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

	updated, err := u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		// entity.go 正: iconId は存在しない
		ContentFiles: nil,
		AssigneeID:   nil,
		Description:  nil,

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

// ============================================================
// internal helpers
// ============================================================
func normalizeContentFilesPtr(p *[]tbdom.ContentFile) *[]tbdom.ContentFile {
	if p == nil {
		return nil
	}
	clean := dedupAndValidateContentFiles(*p)
	return &clean
}

func dedupAndValidateContentFiles(files []tbdom.ContentFile) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	seen := make(map[string]struct{}, len(files))
	out := make([]tbdom.ContentFile, 0, len(files))

	for _, f := range files {
		// default visibility
		if strings.TrimSpace(string(f.Visibility)) == "" {
			f.Visibility = tbdom.VisibilityPrivate
		}
		if err := f.Validate(); err != nil {
			// validate は domain の責務。ここで落とす。
			// 例外を握りつぶすと Firestore に壊れたデータが残る。
			panic(err)
		}

		id := strings.TrimSpace(f.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, f)
	}

	return out
}
