// backend/internal/domain/tokenBlueprint/entity.go
package tokenBlueprint

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
)

// ============================================================
// Types
// ============================================================

// ContentFileType mirrors TS: 'image' | 'video' | 'pdf' | 'document'
type ContentFileType string

const (
	ContentImage    ContentFileType = "image"
	ContentVideo    ContentFileType = "video"
	ContentPDF      ContentFileType = "pdf"
	ContentDocument ContentFileType = "document"
)

// ContentVisibility controls how the content is delivered.
//
// - private: member + ATA==1 のユーザーにのみ、バックエンドが GET 署名URLを発行して配布
// - public: allUsers が GCS 公開バケットの URL で閲覧可能（UIで切替）
type ContentVisibility string

const (
	VisibilityPrivate ContentVisibility = "private"
	VisibilityPublic  ContentVisibility = "public"
)

// 汎用エラー（リポジトリ/サービス共通）
var (
	ErrNotFound = errors.New("tokenBlueprint: not found")
	ErrConflict = errors.New("tokenBlueprint: conflict")
	ErrInvalid  = errors.New("tokenBlueprint: invalid")
)

// 判定ヘルパー
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool  { return errors.Is(err, ErrInvalid) }

// ラップヘルパー（原因を保持）
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

// ContentFile is embedded in TokenBlueprint (single Firestore document).
//
// NOTE:
// - URL を永続化しない（Signed URL は短寿命のため）
// - 永続化するのは objectPath（docId配下の規約）と visibility
type ContentFile struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        ContentFileType   `json:"type"`
	ContentType string            `json:"contentType,omitempty"` // MIME
	Size        int64             `json:"size"`
	ObjectPath  string            `json:"objectPath"` // e.g. "{tokenBlueprintId}/contents/{contentId}"
	Visibility  ContentVisibility `json:"visibility"` // private|public
	CreatedAt   time.Time         `json:"createdAt"`  // 監査（任意だが推奨）
	CreatedBy   string            `json:"createdBy"`  // 監査（任意だが推奨）
	UpdatedAt   time.Time         `json:"updatedAt"`  // 監査（任意）
	UpdatedBy   string            `json:"updatedBy"`  // 監査（任意）
}

func (f ContentFile) Validate() error {
	if strings.TrimSpace(f.ID) == "" {
		return ErrInvalidContentFile
	}
	if strings.TrimSpace(f.Name) == "" {
		return ErrInvalidContentFile
	}
	if !IsValidContentType(f.Type) {
		return ErrInvalidContentType
	}
	if strings.TrimSpace(f.ObjectPath) == "" {
		return ErrInvalidContentFile
	}
	if !IsValidVisibility(f.Visibility) {
		return ErrInvalidContentVisibility
	}
	if f.Size < 0 {
		return fmt.Errorf("%w: size", ErrInvalidContentFile)
	}
	// contentType は空を許容（不明な場合）
	// createdAt/createdBy は移行や既存データ互換のため validate では必須にしない
	return nil
}

// ============================================================
// TokenBlueprint
// ============================================================

// TokenBlueprint is the only persisted aggregate (Firestore).
//
// Design decisions for current implementation policy:
// - iconId/iconUrl は保持しない（icon は "{docId}/icon" 規約、表示URLは metadata 解決結果に含める）
// - content は tokenBlueprint に埋め込み（他テーブル参照しない）
type TokenBlueprint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"` // 空OK（既存データ互換）

	ContentFiles []ContentFile `json:"contentFiles"` // embedded
	AssigneeID   string        `json:"assigneeId"`
	Minted       bool          `json:"minted"` // false | true

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`

	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy *string    `json:"deletedBy,omitempty"`

	// metadataUri: backend resolver URL (Irys/Arweave discontinued).
	// e.g. "https://api.example.com/v1/token-blueprints/{id}/metadata"
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

	// minted=true の場合のコアフィールド/削除制約用
	// - name / symbol / brandId の変更は禁止（コア定義）
	// - Delete 系（DeletedBy/DeletedAt）は禁止
	ErrAlreadyMinted = errors.New("tokenBlueprint: already minted; core fields or deletion are not allowed")
)

var symbolRe = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// ============================================================
// Validation
// ============================================================

func (t TokenBlueprint) validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(t.Name) == "" {
		return ErrInvalidName
	}
	if !symbolRe.MatchString(strings.TrimSpace(t.Symbol)) {
		return ErrInvalidSymbol
	}
	if strings.TrimSpace(t.BrandID) == "" {
		return ErrInvalidBrandID
	}
	if strings.TrimSpace(t.CompanyID) == "" {
		return ErrInvalidCompanyID
	}
	if strings.TrimSpace(t.AssigneeID) == "" {
		return ErrInvalidAssigneeID
	}

	// Description は空OK（既存データ互換 + UI上の任意入力を想定）
	t.Description = strings.TrimSpace(t.Description)

	for _, f := range t.ContentFiles {
		if err := f.Validate(); err != nil {
			return err
		}
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(t.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	// MetadataURI は resolver URL だが、移行・作成直後の差分を考慮して validate では必須にしない
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
		ID:           strings.TrimSpace(id),
		Name:         strings.TrimSpace(name),
		Symbol:       strings.TrimSpace(symbol),
		BrandID:      strings.TrimSpace(brandID),
		CompanyID:    strings.TrimSpace(companyID),
		Description:  strings.TrimSpace(description),
		ContentFiles: dedupContentFiles(contentFiles),
		AssigneeID:   strings.TrimSpace(assigneeID),
		Minted:       false, // create 時は常に false
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    strings.TrimSpace(createdBy),
		UpdatedAt:    updatedAt.UTC(),
		UpdatedBy:    "",
		MetadataURI:  "",
	}

	if err := tb.validate(); err != nil {
		return TokenBlueprint{}, err
	}
	return tb, nil
}

func NewFromStrings(
	id, name, symbol, brandID, companyID, description string,
	assigneeID string,
	createdAt string,
	createdBy string,
	updatedAt string,
) (TokenBlueprint, error) {

	ca, err := parseTime(createdAt)
	if err != nil {
		return TokenBlueprint{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ua, err := parseTime(updatedAt)
	if err != nil {
		return TokenBlueprint{}, fmt.Errorf("invalid updatedAt: %v", err)
	}

	return New(id, name, symbol, brandID, companyID, description, nil, assigneeID, ca, createdBy, ua)
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

// Description は空OK
func (t *TokenBlueprint) UpdateDescription(desc string) error {
	if t == nil {
		return nil
	}
	t.Description = strings.TrimSpace(desc)
	return nil
}

// minted=true でも assigneeId は変更可能（要件より）
func (t *TokenBlueprint) UpdateAssignee(id string) error {
	if t == nil {
		return nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

// SetMinted updates minted flag (false / true)
// - false → true: 許可
// - true → true: 許可（冪等）
// - true → false: 禁止（ErrAlreadyMinted）
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

// minted=true では brandId 変更禁止
func (t *TokenBlueprint) SetBrand(b branddom.Brand) error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	id := strings.TrimSpace(b.ID)
	if id == "" {
		return ErrInvalidBrandID
	}
	t.BrandID = id
	return nil
}

func (t TokenBlueprint) ValidateBrandLink() error {
	if strings.TrimSpace(t.BrandID) == "" {
		return ErrInvalidBrandID
	}
	return nil
}

// minted=true でも assigneeId の設定は許可
func (t *TokenBlueprint) SetAssignee(m memberdom.Member) error {
	if t == nil {
		return nil
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

func (t TokenBlueprint) ValidateAssigneeLink() error {
	if strings.TrimSpace(t.AssigneeID) == "" {
		return ErrInvalidAssigneeID
	}
	return nil
}

func (t *TokenBlueprint) SetCreatedBy(m memberdom.Member) error {
	if t == nil {
		return nil
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return ErrInvalidCreatedBy
	}
	t.CreatedBy = id
	return nil
}

func (t TokenBlueprint) ValidateCreatedByLink() error {
	if strings.TrimSpace(t.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	return nil
}

func (t *TokenBlueprint) SetUpdatedBy(m memberdom.Member) error {
	if t == nil {
		return nil
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return ErrInvalidUpdatedBy
	}
	t.UpdatedBy = id
	return nil
}

func (t TokenBlueprint) ValidateUpdatedByLink() error {
	if strings.TrimSpace(t.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	return nil
}

// minted=true の場合は削除マーク禁止（＝ Delete 不可）
func (t *TokenBlueprint) SetDeletedBy(m memberdom.Member) error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	id := strings.TrimSpace(m.ID)
	if id == "" {
		return ErrInvalidDeletedBy
	}
	t.DeletedBy = &id
	return nil
}

// minted=true の場合は削除状態変更も禁止
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
	if strings.TrimSpace(*t.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	return nil
}

// SetMetadataURI sets backend resolver URL (not Arweave/Irys).
func (t *TokenBlueprint) SetMetadataURI(uri string) error {
	if t == nil {
		return nil
	}
	t.MetadataURI = strings.TrimSpace(uri)
	return nil
}

// ============================================================
// ContentFiles operations (embedded)
// ============================================================

// AddContentFile adds new content (default visibility is private if empty).
// minted=true でも contents の変更は許可（要件より）
func (t *TokenBlueprint) AddContentFile(f ContentFile) error {
	if t == nil {
		return nil
	}
	if strings.TrimSpace(string(f.Visibility)) == "" {
		f.Visibility = VisibilityPrivate
	}
	if err := f.Validate(); err != nil {
		return err
	}

	// duplicate check by ID
	for _, existing := range t.ContentFiles {
		if strings.TrimSpace(existing.ID) == strings.TrimSpace(f.ID) {
			return WrapConflict(nil, "content file id already exists")
		}
	}

	t.ContentFiles = append(t.ContentFiles, f)
	t.ContentFiles = dedupContentFiles(t.ContentFiles)
	return nil
}

// ReplaceContentFiles replaces all content files (used for admin operations).
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

// SetContentVisibility updates visibility for a specific contentId.
func (t *TokenBlueprint) SetContentVisibility(contentID string, v ContentVisibility, actorID string, now time.Time) error {
	if t == nil {
		return nil
	}
	contentID = strings.TrimSpace(contentID)
	if contentID == "" {
		return ErrInvalidContentFile
	}
	if !IsValidVisibility(v) {
		return ErrInvalidContentVisibility
	}

	for i := range t.ContentFiles {
		if strings.TrimSpace(t.ContentFiles[i].ID) == contentID {
			t.ContentFiles[i].Visibility = v
			if !now.IsZero() {
				t.ContentFiles[i].UpdatedAt = now.UTC()
			}
			if strings.TrimSpace(actorID) != "" {
				t.ContentFiles[i].UpdatedBy = strings.TrimSpace(actorID)
			}
			return nil
		}
	}

	return WrapNotFound(nil, "content file not found")
}

// ============================================================
// Helpers
// ============================================================

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

func dedupContentFiles(xs []ContentFile) []ContentFile {
	seen := make(map[string]struct{}, len(xs))
	out := make([]ContentFile, 0, len(xs))

	for _, x := range xs {
		id := strings.TrimSpace(x.ID)
		if id == "" {
			// invalid は validate で弾く。ここでは落とさず維持しておく（デバッグ性）
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
