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
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"`
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
		ContentFiles: dedupTrim(contentFiles),
		AssigneeID:   strings.TrimSpace(assigneeID),
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

	return New(id, name, symbol, brandID, companyID, description, iconPtr, contentFiles, assigneeID, ca, createdBy, ua)
}

// Mutators

func (t *TokenBlueprint) UpdateDescription(desc string) error {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ErrInvalidDescription
	}
	t.Description = desc
	return nil
}

func (t *TokenBlueprint) UpdateAssignee(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalidAssigneeID
	}
	t.AssigneeID = id
	return nil
}

// SetIconID sets or clears icon id
func (t *TokenBlueprint) SetIconID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		t.IconID = nil
		return nil
	}
	t.IconID = &id
	return nil
}

func (t *TokenBlueprint) ClearIconID() {
	t.IconID = nil
}

// 互換用
func (t *TokenBlueprint) SetIconURL(u string) error { return t.SetIconID(u) }
func (t *TokenBlueprint) ClearIconURL()             { t.ClearIconID() }

func (t *TokenBlueprint) SetBrand(b branddom.Brand) error {
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

func (t *TokenBlueprint) SetDeletedBy(m memberdom.Member) error {
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

func (t *TokenBlueprint) ClearDeletedBy() {
	t.DeletedBy = nil
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
