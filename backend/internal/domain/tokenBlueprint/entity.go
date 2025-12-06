package tokenBlueprint

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	branddom "narratives/internal/domain/brand"
	memberdom "narratives/internal/domain/member"
	tokenicondom "narratives/internal/domain/tokenIcon"
)

// ContentFileType mirrors TS: 'image' | 'video' | 'pdf' | 'document'
type ContentFileType string

const (
	ContentImage    ContentFileType = "image"
	ContentVideo    ContentFileType = "video"
	ContentPDF      ContentFileType = "pdf"
	ContentDocument ContentFileType = "document"
)

// minted: "notYet" | "minted"
type MintStatus string

const (
	MintStatusNotYet MintStatus = "notYet"
	MintStatusMinted MintStatus = "minted"
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

// Validation

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
	// companyId 必須
	if strings.TrimSpace(t.CompanyID) == "" {
		return ErrInvalidCompanyID
	}
	if strings.TrimSpace(t.Description) == "" {
		return ErrInvalidDescription
	}
	if t.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}

	// IconID は任意
	if t.IconID != nil && strings.TrimSpace(*t.IconID) == "" {
		return ErrInvalidIconID
	}

	for _, id := range t.ContentFiles {
		if strings.TrimSpace(id) == "" {
			return ErrInvalidContentFiles
		}
	}

	if !IsValidMintStatus(t.Minted) {
		return ErrInvalidMintStatus
	}

	if t.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(t.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}

	return nil
}

func IsValidContentType(t ContentFileType) bool {
	switch t {
	case ContentImage, ContentVideo, ContentPDF, ContentDocument:
		return true
	default:
		return false
	}
}

func IsValidMintStatus(s MintStatus) bool {
	switch s {
	case MintStatusNotYet, MintStatusMinted:
		return true
	default:
		return false
	}
}

// ContentFile mirrors shared/types/tokenBlueprint.ts
type ContentFile struct {
	ID   string
	Name string
	Type ContentFileType
	URL  string
	Size int64
}

func (f ContentFile) Validate() error {
	if strings.TrimSpace(f.ID) == "" || strings.TrimSpace(f.Name) == "" {
		return ErrInvalidContentFile
	}
	if !IsValidContentType(f.Type) {
		return ErrInvalidContentType
	}
	if f.Size < 0 {
		return fmt.Errorf("%w: size", ErrInvalidContentFile)
	}
	return nil
}

// TokenBlueprint mirrors TS-type
type TokenBlueprint struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Symbol       string     `json:"symbol"`
	BrandID      string     `json:"brandId"`
	CompanyID    string     `json:"companyId"`
	Description  string     `json:"description"`
	IconID       *string    `json:"iconId,omitempty"`
	IconURL      string     `json:"iconUrl,omitempty"` // ★ 追加: 表示用のアイコン URL
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"`
	Minted       MintStatus `json:"minted"` // "notYet" | "minted"
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	UpdatedBy    string     `json:"updatedBy"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	DeletedBy    *string    `json:"deletedBy,omitempty"`
}

// Errors
var (
	ErrInvalidID          = errors.New("tokenBlueprint: invalid id")
	ErrInvalidName        = errors.New("tokenBlueprint: invalid name")
	ErrInvalidSymbol      = errors.New("tokenBlueprint: invalid symbol")
	ErrInvalidBrandID     = errors.New("tokenBlueprint: invalid brandId")
	ErrInvalidCompanyID   = errors.New("tokenBlueprint: invalid companyId")
	ErrInvalidDescription = errors.New("tokenBlueprint: invalid description")
	ErrInvalidAssigneeID  = errors.New("tokenBlueprint: invalid assigneeId")

	ErrInvalidIconID    = errors.New("tokenBlueprint: invalid iconId")
	ErrInvalidCreatedAt = errors.New("tokenBlueprint: invalid createdAt")
	ErrInvalidCreatedBy = errors.New("tokenBlueprint: invalid createdBy")
	ErrInvalidUpdatedBy = errors.New("tokenBlueprint: invalid updatedBy")
	ErrInvalidDeletedBy = errors.New("tokenBlueprint: invalid deletedBy")

	ErrInvalidContentFiles = errors.New("tokenBlueprint: invalid contentFiles")
	ErrInvalidContentFile  = errors.New("tokenBlueprint: invalid contentFile")
	ErrInvalidContentType  = errors.New("tokenBlueprint: invalid contentFile.type")

	ErrInvalidMintStatus = errors.New("tokenBlueprint: invalid minted")

	// ★ minted=MintStatusMinted の場合のコアフィールド/削除制約用
	//   - name / symbol / brandId の変更は禁止（コア定義）
	//   - Delete 系（DeletedBy/DeletedAt）は禁止
	ErrAlreadyMinted = errors.New("tokenBlueprint: already minted; core fields or deletion are not allowed")
)

var symbolRe = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// Constructors

func New(
	id, name, symbol, brandID, companyID, description string,
	iconID *string,
	contentFiles []string,
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
		IconID:       normalizePtr(iconID),
		IconURL:      "", // 初期値は空。必要に応じて別レイヤーで補完。
		ContentFiles: dedupTrim(contentFiles),
		AssigneeID:   strings.TrimSpace(assigneeID),
		Minted:       MintStatusNotYet, // create 時は常に notYet
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    strings.TrimSpace(createdBy),
		UpdatedAt:    updatedAt.UTC(),
	}

	if err := tb.validate(); err != nil {
		return TokenBlueprint{}, err
	}

	return tb, nil
}

func NewFromStrings(
	id, name, symbol, brandID, companyID, description string,
	iconID string,
	contentFiles []string,
	assigneeID string,
	createdAt string,
	createdBy string,
	updatedAt string,
) (TokenBlueprint, error) {
	var iconPtr *string
	if strings.TrimSpace(iconID) != "" {
		icon := strings.TrimSpace(iconID)
		iconPtr = &icon
	}

	ca, err := parseTime(createdAt)
	if err != nil {
		return TokenBlueprint{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ua, err := parseTime(updatedAt)
	if err != nil {
		return TokenBlueprint{}, fmt.Errorf("invalid updatedAt: %v", err)
	}

	// minted は create 時は常に notYet
	return New(id, name, symbol, brandID, companyID, description, iconPtr, contentFiles, assigneeID, ca, createdBy, ua)
}

// ===============================
// 内部ヘルパー: minted=MintStatusMinted の場合の制約
//
//	※ここでは「削除系」や「brandId変更」のような
//	  コアな変更にのみ使う想定（icon/assignee/contents は対象外）
//
// ===============================
func (t *TokenBlueprint) ensureMutableCoreOrDeletable() error {
	if t == nil {
		return nil
	}
	if t.Minted == MintStatusMinted {
		return ErrAlreadyMinted
	}
	return nil
}

// Mutators

// ★ minted=MintStatusMinted でも Description は変更可能（要件に含まれていないため）
//
//	→ ここでは ensureMutableCoreOrDeletable を呼ばない
func (t *TokenBlueprint) UpdateDescription(desc string) error {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ErrInvalidDescription
	}
	t.Description = desc
	return nil
}

// ★ minted=MintStatusMinted でも assigneeId は変更可能（要件より）
func (t *TokenBlueprint) UpdateAssignee(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

// SetMinted updates minted status ("notYet" / "minted")
// - notYet → minted: 許可
// - minted → minted: 許可（冪等）
// - minted → notYet: 禁止（ErrAlreadyMinted）
func (t *TokenBlueprint) SetMinted(status MintStatus) error {
	if t == nil {
		return nil
	}

	status = MintStatus(strings.TrimSpace(string(status)))
	if status == "" {
		status = MintStatusNotYet
	}
	if !IsValidMintStatus(status) {
		return ErrInvalidMintStatus
	}

	// すでに minted 済みの場合、別の状態へ戻すことは禁止
	if t.Minted == MintStatusMinted && status != MintStatusMinted {
		return ErrAlreadyMinted
	}

	t.Minted = status
	return nil
}

// SetIconID sets or clears icon id
// ★ minted=MintStatusMinted でも icon の変更は許可（要件より）
func (t *TokenBlueprint) SetIconID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		t.IconID = nil
		return nil
	}
	t.IconID = &id
	return nil
}

// ★ minted=MintStatusMinted でも icon のクリアは許可
func (t *TokenBlueprint) ClearIconID() error {
	t.IconID = nil
	return nil
}

// アイコン URL 用のヘルパ（表示専用）
// TokenIcon リポジトリ等で URL を解決したあとにセットする想定。
func (t *TokenBlueprint) SetIconURL(u string) error {
	if t == nil {
		return nil
	}
	t.IconURL = strings.TrimSpace(u)
	return nil
}

func (t *TokenBlueprint) ClearIconURL() error {
	if t == nil {
		return nil
	}
	t.IconURL = ""
	return nil
}

// ★ minted=MintStatusMinted では brandId 変更禁止
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

// ★ minted=MintStatusMinted でも icon の設定は許可（SetIconID と同様の方針）
func (t *TokenBlueprint) SetIcon(icon tokenicondom.TokenIcon) error {
	if t == nil {
		return nil
	}
	id := strings.TrimSpace(icon.ID)
	if id == "" {
		return ErrInvalidIconID
	}
	t.IconID = &id
	return nil
}

func (t TokenBlueprint) ValidateIconLink() error {
	if t.IconID == nil {
		return nil
	}
	if strings.TrimSpace(*t.IconID) == "" {
		return ErrInvalidIconID
	}
	return nil
}

// ★ minted=MintStatusMinted でも assigneeId の設定は許可
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

// createdBy/updatedBy はコア定義とは少し性質が異なるが、
// 一般には更新されない想定なので minted でも特に制約しない。
// （必要であれば ensureMutableCoreOrDeletable を噛ませる）
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

// ★ minted=MintStatusMinted の場合は削除マーク禁止（＝ Delete 不可）
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

// ★ minted=MintStatusMinted の場合は削除状態変更も禁止
func (t *TokenBlueprint) ClearDeletedBy() error {
	if err := t.ensureMutableCoreOrDeletable(); err != nil {
		return err
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

// Helpers

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

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func dedupTrim(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))

	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}

	return out
}
