// backend/internal/domain/tokenBlueprint/entity.go
package tokenBlueprint

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	branddom "narratives/internal/domain/brand"
)

// ============================================================
// Types
// ============================================================

type ContentFileType string

const (
	ContentImage    ContentFileType = "image"
	ContentVideo    ContentFileType = "video"
	ContentPDF      ContentFileType = "pdf"
	ContentDocument ContentFileType = "document"
)

type ContentVisibility string

const (
	VisibilityPrivate ContentVisibility = "private"
	VisibilityPublic  ContentVisibility = "public"
)

var (
	ErrNotFound = errors.New("tokenBlueprint: not found")
	ErrConflict = errors.New("tokenBlueprint: conflict")
	ErrInvalid  = errors.New("tokenBlueprint: invalid")
)

func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool  { return errors.Is(err, ErrInvalid) }

func WrapInvalid(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrInvalid, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrInvalid, msg, err)
}

func WrapConflict(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrConflict, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrConflict, msg, err)
}

func WrapNotFound(err error, msg string) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	}
	return fmt.Errorf("%w: %s: %v", ErrNotFound, msg, err)
}

// ============================================================
// ContentFile
// ============================================================

func IsValidContentType(t ContentFileType) bool {
	switch t {
	case ContentImage, ContentVideo, ContentPDF, ContentDocument:
		return true
	default:
		return false
	}
}

func IsValidVisibility(v ContentVisibility) bool {
	switch v {
	case VisibilityPrivate, VisibilityPublic:
		return true
	default:
		return false
	}
}

// ContentFile is embedded in TokenBlueprint.
//
// Firebase Storage 移行後:
// - frontend が Firebase Storage へ直接 upload する
// - backend は upload URL を発行しない
// - 永続化するのは Firebase Storage の objectPath と downloadURL
// - URL は Firebase Storage の getDownloadURL() で取得した値を保存する
type ContentFile struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        ContentFileType   `json:"type"`
	ContentType string            `json:"contentType,omitempty"`
	Size        int64             `json:"size"`
	ObjectPath  string            `json:"objectPath"`
	URL         string            `json:"url,omitempty"`
	Visibility  ContentVisibility `json:"visibility"`

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

func (f ContentFile) Validate() error {
	if f.ID == "" {
		return ErrInvalidContentFile
	}
	if f.Name == "" {
		return ErrInvalidContentFile
	}
	if !IsValidContentType(f.Type) {
		return ErrInvalidContentType
	}
	if f.ObjectPath == "" {
		return ErrInvalidContentFile
	}
	if !IsValidVisibility(f.Visibility) {
		return ErrInvalidContentVisibility
	}
	if f.Size < 0 {
		return fmt.Errorf("%w: size", ErrInvalidContentFile)
	}
	return nil
}

// ============================================================
// TokenBlueprint
// ============================================================

// TokenBlueprint is the only persisted aggregate.
//
// Firebase Storage 移行後:
// - tokenBlueprintIcon / tokenBlueprintContents は frontend から Firebase Storage へ直接 upload する
// - backend は GCS signed URL endpoint を持たない
// - iconUrl には Firebase Storage の downloadURL を保存する
// - contentFiles[].url にも Firebase Storage の downloadURL を保存する
// - objectPath は Firebase Storage 上の参照パスとして保持する
//
// create 時:
// - metadataUri は作成しない（空のまま）
// - assignee / createdBy / updatedBy / deletedBy は member id を保持する
type TokenBlueprint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`

	// Firebase Storage downloadURL for tokenBlueprint icon.
	IconURL string `json:"iconUrl,omitempty"`

	// Firebase Storage object paths.
	// TokenIconObjectPath:
	//   旧GCS用途ではなく、Firebase Storage 上の icon objectPath として扱う。
	// TokenContentsObjectPath:
	//   旧GCS用途ではなく、Firebase Storage 上の contents root/reference path として扱う。
	TokenIconObjectPath     string `json:"tokenIconObjectPath"`
	TokenContentsObjectPath string `json:"tokenContentsObjectPath"`

	ContentFiles []ContentFile `json:"contentFiles"`
	AssigneeID   string        `json:"assigneeId"`
	Minted       bool          `json:"minted"`
	CreatedAt    time.Time     `json:"createdAt"`
	CreatedBy    string        `json:"createdBy"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	UpdatedBy    string        `json:"updatedBy"`

	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`

	MetadataURI string `json:"metadataUri,omitempty"`
}

// Errors
var (
	ErrInvalidID         = errors.New("tokenBlueprint: invalid id")
	ErrInvalidName       = errors.New("tokenBlueprint: invalid name")
	ErrInvalidSymbol     = errors.New("tokenBlueprint: invalid symbol")
	ErrInvalidBrandID    = errors.New("tokenBlueprint: invalid brandId")
	ErrInvalidCompanyID  = errors.New("tokenBlueprint: invalid companyId")
	ErrInvalidAssigneeID = errors.New("tokenBlueprint: invalid assigneeId")

	ErrInvalidCreatedAt = errors.New("tokenBlueprint: invalid createdAt")
	ErrInvalidCreatedBy = errors.New("tokenBlueprint: invalid createdBy")
	ErrInvalidUpdatedBy = errors.New("tokenBlueprint: invalid updatedBy")
	ErrInvalidDeletedBy = errors.New("tokenBlueprint: invalid deletedBy")

	ErrInvalidContentFiles      = errors.New("tokenBlueprint: invalid contentFiles")
	ErrInvalidContentFile       = errors.New("tokenBlueprint: invalid contentFile")
	ErrInvalidContentType       = errors.New("tokenBlueprint: invalid contentFile.type")
	ErrInvalidContentVisibility = errors.New("tokenBlueprint: invalid contentFile.visibility")

	ErrInvalidTokenIconObjectPath     = errors.New("tokenBlueprint: invalid tokenIconObjectPath")
	ErrInvalidTokenContentsObjectPath = errors.New("tokenBlueprint: invalid tokenContentsObjectPath")

	ErrAlreadyMinted = errors.New("tokenBlueprint: already minted; core fields or deletion are not allowed")
)

var symbolRe = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// ============================================================
// Validation
// ============================================================

func (t TokenBlueprint) validate() error {
	if t.ID == "" {
		return ErrInvalidID
	}
	if t.Name == "" {
		return ErrInvalidName
	}
	if !symbolRe.MatchString(t.Symbol) {
		return ErrInvalidSymbol
	}
	if t.BrandID == "" {
		return ErrInvalidBrandID
	}
	if t.CompanyID == "" {
		return ErrInvalidCompanyID
	}
	if t.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}

	for _, f := range t.ContentFiles {
		if err := f.Validate(); err != nil {
			return err
		}
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if t.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}

	// IconURL / tokenIconObjectPath / tokenContentsObjectPath / MetadataURI は、
	// 作成直後・既存データ移行・画像未登録状態を考慮し必須にしない。
	return nil
}

// ============================================================
// Constructors
// ============================================================

func New(
	id, name, symbol, brandID, companyID, description string,
	contentFiles []ContentFile,
	assigneeID string,
	tokenIconObjectPath string,
	tokenContentsObjectPath string,
	createdAt time.Time,
	createdBy string,
	updatedAt time.Time,
) (TokenBlueprint, error) {
	tid := id

	iconPath := tokenIconObjectPath
	contentsPath := tokenContentsObjectPath

	if iconPath == "" {
		iconPath = DefaultTokenIconObjectPath(tid)
	}
	if contentsPath == "" {
		contentsPath = DefaultTokenContentsObjectPath(tid)
	}

	tb := TokenBlueprint{
		ID:           tid,
		Name:         name,
		Symbol:       symbol,
		BrandID:      brandID,
		CompanyID:    companyID,
		Description:  description,
		IconURL:      "",
		ContentFiles: dedupContentFiles(contentFiles),
		AssigneeID:   assigneeID,
		Minted:       false,

		TokenIconObjectPath:     iconPath,
		TokenContentsObjectPath: contentsPath,

		CreatedAt: createdAt.UTC(),
		CreatedBy: createdBy,
		UpdatedAt: updatedAt.UTC(),
		UpdatedBy: "",

		MetadataURI: "",
	}

	if err := tb.validate(); err != nil {
		return TokenBlueprint{}, err
	}
	return tb, nil
}

// ============================================================
// Core mutability constraints (minted)
// ============================================================

func (t *TokenBlueprint) ensureMutableCoreOrDeletable() error {
	if t == nil {
		return nil
	}
	if t.Minted {
		return ErrAlreadyMinted
	}
	return nil
}

// ============================================================
// Mutators
// ============================================================

func (t *TokenBlueprint) UpdateDescription(desc string) error {
	if t == nil {
		return nil
	}
	t.Description = desc
	return nil
}

func (t *TokenBlueprint) UpdateAssigneeID(id string) error {
	if t == nil {
		return nil
	}
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

func (t *TokenBlueprint) SetMinted(status bool) error {
	if t == nil {
		return nil
	}
	if t.Minted && !status {
		return ErrAlreadyMinted
	}
	t.Minted = status
	return nil
}

func (t *TokenBlueprint) SetBrand(b branddom.Brand) error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	id := b.ID
	if id == "" {
		return ErrInvalidBrandID
	}
	t.BrandID = id
	return nil
}

func (t TokenBlueprint) ValidateBrandLink() error {
	if t.BrandID == "" {
		return ErrInvalidBrandID
	}
	return nil
}

func (t *TokenBlueprint) SetAssigneeID(id string) error {
	if t == nil {
		return nil
	}
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

func (t TokenBlueprint) ValidateAssigneeLink() error {
	if t.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}
	return nil
}

func (t *TokenBlueprint) SetCreatedBy(createdBy string) error {
	if t == nil {
		return nil
	}
	if createdBy == "" {
		return ErrInvalidCreatedBy
	}
	t.CreatedBy = createdBy
	return nil
}

func (t TokenBlueprint) ValidateCreatedByLink() error {
	if t.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}
	return nil
}

func (t *TokenBlueprint) SetUpdatedBy(updatedBy string) error {
	if t == nil {
		return nil
	}
	if updatedBy == "" {
		return ErrInvalidUpdatedBy
	}
	t.UpdatedBy = updatedBy
	return nil
}

func (t TokenBlueprint) ValidateUpdatedByLink() error {
	if t.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}
	return nil
}

func (t *TokenBlueprint) SetDeletedBy(deletedBy string) error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	if deletedBy == "" {
		return ErrInvalidDeletedBy
	}
	t.DeletedBy = &deletedBy
	return nil
}

func (t *TokenBlueprint) ClearDeletedBy() error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	t.DeletedBy = nil
	return nil
}

func (t TokenBlueprint) ValidateDeletedByLink() error {
	if t.DeletedBy == nil {
		return nil
	}
	if *t.DeletedBy == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

func (t *TokenBlueprint) SetMetadataURI(uri string) error {
	if t == nil {
		return nil
	}
	t.MetadataURI = uri
	return nil
}

func (t *TokenBlueprint) SetIconURL(url string) error {
	if t == nil {
		return nil
	}
	t.IconURL = url
	return nil
}

func (t *TokenBlueprint) SetTokenIconObjectPath(path string) error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	if path == "" {
		return ErrInvalidTokenIconObjectPath
	}
	t.TokenIconObjectPath = path
	return nil
}

func (t *TokenBlueprint) SetTokenContentsObjectPath(path string) error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	if path == "" {
		return ErrInvalidTokenContentsObjectPath
	}
	t.TokenContentsObjectPath = path
	return nil
}

// ============================================================
// ContentFiles operations (embedded)
// ============================================================

func (t *TokenBlueprint) AddContentFile(f ContentFile) error {
	if t == nil {
		return nil
	}
	if string(f.Visibility) == "" {
		f.Visibility = VisibilityPrivate
	}
	if err := f.Validate(); err != nil {
		return err
	}

	for _, existing := range t.ContentFiles {
		if existing.ID == f.ID {
			return WrapConflict(nil, "content file id already exists")
		}
	}

	t.ContentFiles = append(t.ContentFiles, f)
	t.ContentFiles = dedupContentFiles(t.ContentFiles)
	return nil
}

func (t *TokenBlueprint) ReplaceContentFiles(files []ContentFile) error {
	if t == nil {
		return nil
	}
	clean := dedupContentFiles(files)
	for _, f := range clean {
		if err := f.Validate(); err != nil {
			return err
		}
	}
	t.ContentFiles = clean
	return nil
}

func (t *TokenBlueprint) SetContentVisibility(contentID string, v ContentVisibility, actorID string, now time.Time) error {
	if t == nil {
		return nil
	}
	if contentID == "" {
		return ErrInvalidContentFile
	}
	if !IsValidVisibility(v) {
		return ErrInvalidContentVisibility
	}

	for i := range t.ContentFiles {
		if t.ContentFiles[i].ID == contentID {
			t.ContentFiles[i].Visibility = v
			if !now.IsZero() {
				t.ContentFiles[i].UpdatedAt = now.UTC()
			}
			if actorID != "" {
				t.ContentFiles[i].UpdatedBy = actorID
			}
			return nil
		}
	}

	return WrapNotFound(nil, "content file not found")
}

// ============================================================
// Helpers
// ============================================================

func DefaultTokenIconObjectPath(tokenBlueprintID string) string {
	if tokenBlueprintID == "" {
		return ""
	}
	return fmt.Sprintf("%s/icon", tokenBlueprintID)
}

func DefaultTokenContentsObjectPath(tokenBlueprintID string) string {
	if tokenBlueprintID == "" {
		return ""
	}
	return fmt.Sprintf("%s/.keep", tokenBlueprintID)
}

func dedupContentFiles(xs []ContentFile) []ContentFile {
	seen := make(map[string]struct{}, len(xs))
	out := make([]ContentFile, 0, len(xs))

	for _, x := range xs {
		id := x.ID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, x)
	}

	return out
}
