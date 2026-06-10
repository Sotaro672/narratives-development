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
// - backend は upload URL / signed URL を発行しない
// - url は Firebase Storage の getDownloadURL() で取得した downloadURL
// - objectPath は Firebase Storage 上の実体を差し替え・削除するための正規キー
// - name / contentType / size は表示・監査・差し替え判断用に保持する
type ContentFile struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        ContentFileType   `json:"type"`
	ContentType string            `json:"contentType,omitempty"`
	URL         string            `json:"url"`
	ObjectPath  string            `json:"objectPath"`
	Visibility  ContentVisibility `json:"visibility"`
	Size        int64             `json:"size"`

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
	if f.URL == "" {
		return ErrInvalidContentFile
	}
	if f.ObjectPath == "" {
		return ErrInvalidContentFile
	}
	if !IsValidVisibility(f.Visibility) {
		return ErrInvalidContentVisibility
	}
	if f.Size < 0 {
		return ErrInvalidContentFile
	}
	if f.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if f.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if f.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if f.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}
	return nil
}

// ValidateContentFiles validates embedded content files as a domain rule.
//
// This belongs to the domain layer because:
// - ContentFile validation is not Firestore-specific.
// - Duplicate content file IDs are an aggregate consistency rule.
// - Repository implementations should only persist already-valid domain data.
// - objectPath is required so Firebase Storage assets can be replaced/deleted later.
func ValidateContentFiles(files []ContentFile) error {
	seen := make(map[string]struct{}, len(files))
	objectPaths := make(map[string]struct{}, len(files))

	for i, f := range files {
		if f.ID == "" {
			return fmt.Errorf("%w: contentFiles[%d].id", ErrInvalidContentFile, i)
		}

		if _, ok := seen[f.ID]; ok {
			return WrapConflict(nil, fmt.Sprintf("contentFiles[%d].id duplicated: %s", i, f.ID))
		}
		seen[f.ID] = struct{}{}

		if f.ObjectPath == "" {
			return fmt.Errorf("%w: contentFiles[%d].objectPath", ErrInvalidContentFile, i)
		}

		if _, ok := objectPaths[f.ObjectPath]; ok {
			return WrapConflict(nil, fmt.Sprintf("contentFiles[%d].objectPath duplicated: %s", i, f.ObjectPath))
		}
		objectPaths[f.ObjectPath] = struct{}{}

		if err := f.Validate(); err != nil {
			return err
		}
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
// - backend は upload URL / signed URL を発行しない
// - iconUrl には Firebase Storage の downloadURL を保存する
// - iconObjectPath には Firebase Storage 上の icon objectPath を保存する
// - contentFiles[].url には Firebase Storage の downloadURL を保存する
// - contentFiles[].objectPath には Firebase Storage 上の objectPath を保存する
// - objectPath は差し替え・削除・cleanup のための正規キーとして扱う
//
// create 時:
// - minted は常に false
// - metadataUri は作成直後は空でもよい
// - icon は未登録状態を許容する
// - assignee / createdBy / updatedBy は member id を保持する
type TokenBlueprint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`

	// Firebase Storage metadata for tokenBlueprint icon.
	IconURL         string `json:"iconUrl,omitempty"`
	IconObjectPath  string `json:"iconObjectPath,omitempty"`
	IconFileName    string `json:"iconFileName,omitempty"`
	IconContentType string `json:"iconContentType,omitempty"`
	IconSize        int64  `json:"iconSize,omitempty"`

	ContentFiles []ContentFile `json:"contentFiles"`
	AssigneeID   string        `json:"assigneeId"`
	Minted       bool          `json:"minted"`
	CreatedAt    time.Time     `json:"createdAt"`
	CreatedBy    string        `json:"createdBy"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	UpdatedBy    string        `json:"updatedBy"`

	MetadataURI string `json:"metadataUri,omitempty"`
}

// Errors
var (
	ErrNilTokenBlueprint = errors.New("tokenBlueprint: nil receiver")

	ErrInvalidID         = errors.New("tokenBlueprint: invalid id")
	ErrInvalidName       = errors.New("tokenBlueprint: invalid name")
	ErrInvalidSymbol     = errors.New("tokenBlueprint: invalid symbol")
	ErrInvalidBrandID    = errors.New("tokenBlueprint: invalid brandId")
	ErrInvalidCompanyID  = errors.New("tokenBlueprint: invalid companyId")
	ErrInvalidAssigneeID = errors.New("tokenBlueprint: invalid assigneeId")

	ErrInvalidCreatedAt = errors.New("tokenBlueprint: invalid createdAt")
	ErrInvalidCreatedBy = errors.New("tokenBlueprint: invalid createdBy")
	ErrInvalidUpdatedAt = errors.New("tokenBlueprint: invalid updatedAt")
	ErrInvalidUpdatedBy = errors.New("tokenBlueprint: invalid updatedBy")

	ErrInvalidIconURL         = errors.New("tokenBlueprint: invalid iconUrl")
	ErrInvalidIconObjectPath  = errors.New("tokenBlueprint: invalid iconObjectPath")
	ErrInvalidIconFileName    = errors.New("tokenBlueprint: invalid iconFileName")
	ErrInvalidIconContentType = errors.New("tokenBlueprint: invalid iconContentType")
	ErrInvalidIconSize        = errors.New("tokenBlueprint: invalid iconSize")

	ErrInvalidContentFiles      = errors.New("tokenBlueprint: invalid contentFiles")
	ErrInvalidContentFile       = errors.New("tokenBlueprint: invalid contentFile")
	ErrInvalidContentType       = errors.New("tokenBlueprint: invalid contentFile.type")
	ErrInvalidContentVisibility = errors.New("tokenBlueprint: invalid contentFile.visibility")

	ErrAlreadyMinted = errors.New("tokenBlueprint: already minted; core fields are not allowed")
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

	if err := t.validateIcon(); err != nil {
		return err
	}

	if err := ValidateContentFiles(t.ContentFiles); err != nil {
		return err
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if t.CreatedBy == "" {
		return ErrInvalidCreatedBy
	}
	if t.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if t.UpdatedBy == "" {
		return ErrInvalidUpdatedBy
	}

	return nil
}

func (t TokenBlueprint) validateIcon() error {
	hasAnyIconField :=
		t.IconURL != "" ||
			t.IconObjectPath != "" ||
			t.IconFileName != "" ||
			t.IconContentType != "" ||
			t.IconSize != 0

	// Icon is optional. Empty icon fields are allowed for draft/new records.
	if !hasAnyIconField {
		return nil
	}

	if t.IconURL == "" {
		return ErrInvalidIconURL
	}
	if t.IconObjectPath == "" {
		return ErrInvalidIconObjectPath
	}
	if t.IconFileName == "" {
		return ErrInvalidIconFileName
	}
	if t.IconSize < 0 {
		return ErrInvalidIconSize
	}

	return nil
}

// ============================================================
// Constructors
// ============================================================

func New(
	id, name, symbol, brandID, companyID, description string,
	contentFiles []ContentFile,
	assigneeID string,
	createdAt time.Time,
	createdBy string,
	updatedAt time.Time,
) (TokenBlueprint, error) {
	tb := TokenBlueprint{
		ID:          id,
		Name:        name,
		Symbol:      symbol,
		BrandID:     brandID,
		CompanyID:   companyID,
		Description: description,

		IconURL:         "",
		IconObjectPath:  "",
		IconFileName:    "",
		IconContentType: "",
		IconSize:        0,

		ContentFiles: contentFiles,
		AssigneeID:   assigneeID,
		Minted:       false,

		CreatedAt: createdAt.UTC(),
		CreatedBy: createdBy,
		UpdatedAt: updatedAt.UTC(),
		UpdatedBy: createdBy,

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
		return ErrNilTokenBlueprint
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
		return ErrNilTokenBlueprint
	}
	t.Description = desc
	return nil
}

func (t *TokenBlueprint) UpdateAssigneeID(id string) error {
	if t == nil {
		return ErrNilTokenBlueprint
	}
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

func (t *TokenBlueprint) SetMinted(status bool) error {
	if t == nil {
		return ErrNilTokenBlueprint
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
		return ErrNilTokenBlueprint
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
		return ErrNilTokenBlueprint
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
		return ErrNilTokenBlueprint
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

func (t *TokenBlueprint) SetMetadataURI(uri string) error {
	if t == nil {
		return ErrNilTokenBlueprint
	}
	t.MetadataURI = uri
	return nil
}

// SetIconURL is kept for compatibility.
// New code should prefer SetIcon so objectPath is stored together with the downloadURL.
func (t *TokenBlueprint) SetIconURL(url string) error {
	if t == nil {
		return ErrNilTokenBlueprint
	}
	t.IconURL = url
	return nil
}

func (t *TokenBlueprint) SetIcon(
	url string,
	objectPath string,
	fileName string,
	contentType string,
	size int64,
) error {
	if t == nil {
		return ErrNilTokenBlueprint
	}
	if url == "" {
		return ErrInvalidIconURL
	}
	if objectPath == "" {
		return ErrInvalidIconObjectPath
	}
	if fileName == "" {
		return ErrInvalidIconFileName
	}
	if size < 0 {
		return ErrInvalidIconSize
	}

	t.IconURL = url
	t.IconObjectPath = objectPath
	t.IconFileName = fileName
	t.IconContentType = contentType
	t.IconSize = size

	return nil
}

func (t *TokenBlueprint) ClearIcon() error {
	if t == nil {
		return ErrNilTokenBlueprint
	}

	t.IconURL = ""
	t.IconObjectPath = ""
	t.IconFileName = ""
	t.IconContentType = ""
	t.IconSize = 0

	return nil
}

// ============================================================
// ContentFiles operations (embedded)
// ============================================================

func (t *TokenBlueprint) AddContentFile(f ContentFile) error {
	if t == nil {
		return ErrNilTokenBlueprint
	}

	files := append([]ContentFile{}, t.ContentFiles...)
	files = append(files, f)

	if err := ValidateContentFiles(files); err != nil {
		return err
	}

	t.ContentFiles = files
	return nil
}

func (t *TokenBlueprint) ReplaceContentFiles(files []ContentFile) error {
	if t == nil {
		return ErrNilTokenBlueprint
	}
	if err := ValidateContentFiles(files); err != nil {
		return err
	}
	t.ContentFiles = files
	return nil
}

func (t *TokenBlueprint) RemoveContentFile(contentID string) (*ContentFile, error) {
	if t == nil {
		return nil, ErrNilTokenBlueprint
	}
	if contentID == "" {
		return nil, ErrInvalidContentFile
	}

	next := make([]ContentFile, 0, len(t.ContentFiles))
	var removed *ContentFile

	for i := range t.ContentFiles {
		file := t.ContentFiles[i]
		if file.ID == contentID {
			copyFile := file
			removed = &copyFile
			continue
		}
		next = append(next, file)
	}

	if removed == nil {
		return nil, WrapNotFound(nil, "content file not found")
	}

	if err := ValidateContentFiles(next); err != nil {
		return nil, err
	}

	t.ContentFiles = next
	return removed, nil
}

func (t *TokenBlueprint) ReplaceContentFile(contentID string, nextFile ContentFile) (*ContentFile, error) {
	if t == nil {
		return nil, ErrNilTokenBlueprint
	}
	if contentID == "" {
		return nil, ErrInvalidContentFile
	}

	files := append([]ContentFile{}, t.ContentFiles...)
	var replaced *ContentFile

	for i := range files {
		if files[i].ID == contentID {
			copyFile := files[i]
			replaced = &copyFile
			files[i] = nextFile
			break
		}
	}

	if replaced == nil {
		return nil, WrapNotFound(nil, "content file not found")
	}

	if err := ValidateContentFiles(files); err != nil {
		return nil, err
	}

	t.ContentFiles = files
	return replaced, nil
}

func (t *TokenBlueprint) SetContentVisibility(contentID string, v ContentVisibility, actorID string, now time.Time) error {
	if t == nil {
		return ErrNilTokenBlueprint
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
