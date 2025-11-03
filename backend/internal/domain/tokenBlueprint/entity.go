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

// Validation (moved from entity.go)

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
	if strings.TrimSpace(t.Description) == "" {
		return ErrInvalidDescription
	}
	if t.AssigneeID == "" {
		return ErrInvalidAssigneeID
	}

	// IconID は任意。与えられた場合は空文字でないことのみ検証
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
	Size int64 // bytes
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

// TokenBlueprint mirrors web-app/src/shared/types/tokenBlueprint.ts
type TokenBlueprint struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Symbol       string     `json:"symbol"`
	BrandID      string     `json:"brandId"`
	Description  string     `json:"description"`
	IconID       *string    `json:"iconId,omitempty"`
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"` // TS側のキー名に合わせる
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
	ErrInvalidDescription = errors.New("tokenBlueprint: invalid description")
	ErrInvalidAssigneeID  = errors.New("tokenBlueprint: invalid assigneeId")
	// 置換: IconURL -> IconID
	ErrInvalidIconID       = errors.New("tokenBlueprint: invalid iconId")
	ErrInvalidCreatedAt    = errors.New("tokenBlueprint: invalid createdAt")
	ErrInvalidCreatedBy    = errors.New("tokenBlueprint: invalid createdBy")
	// 追加: UpdatedBy/DeletedBy のリンク検証用
	ErrInvalidUpdatedBy    = errors.New("tokenBlueprint: invalid updatedBy")
	ErrInvalidDeletedBy    = errors.New("tokenBlueprint: invalid deletedBy")

	// 追加: ContentFiles/ContentFile 用のエラー
	ErrInvalidContentFiles = errors.New("tokenBlueprint: invalid contentFiles")
	ErrInvalidContentFile  = errors.New("tokenBlueprint: invalid contentFile")
	ErrInvalidContentType  = errors.New("tokenBlueprint: invalid contentFile.type")
)

var symbolRe = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

// Constructors

// New creates a TokenBlueprint with time.Time arguments.
func New(
	id, name, symbol, brandID, description string,
	iconID *string, // URL ではなく ID を参照
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

// NewFromStrings matches TS (createdAt/updatedAt: string, iconId: string|null).
func NewFromStrings(
	id, name, symbol, brandID, description string,
	iconID string, // "" で null 扱い
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
	return New(id, name, symbol, brandID, description, iconPtr, contentFiles, assigneeID, ca, createdBy, ua)
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

// SetIconID sets or clears icon id (empty clears).
func (t *TokenBlueprint) SetIconID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		t.IconID = nil
		return nil
	}
	t.IconID = &id
	return nil
}

// ClearIconID clears the icon reference.
func (t *TokenBlueprint) ClearIconID() {
	t.IconID = nil
}

// 互換: 既存呼び出しが残っている場合のためのラッパー（将来削除）
func (t *TokenBlueprint) SetIconURL(u string) error { return t.SetIconID(u) }
func (t *TokenBlueprint) ClearIconURL()             { t.ClearIconID() }

// SetBrand は与えられた Brand の主キー（Brand.ID）を BrandID に設定します。
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

// ValidateBrandLink は BrandID が空でないことを検証します。
func (t TokenBlueprint) ValidateBrandLink() error {
	if strings.TrimSpace(t.BrandID) == "" {
		return ErrInvalidBrandID
	}
	return nil
}

// SetIcon は与えられた TokenIcon の主キー（TokenIcon.ID）を IconID に設定します。
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

// ClearIcon はアイコンの参照を解除します（NULL にします）。
func (t *TokenBlueprint) ClearIcon() {
	t.IconID = nil
}

// ValidateIconLink は IconID の妥当性（存在する場合は空でないこと）を検証します。
// 実在性の確認は上位レイヤー（リポジトリ/ユースケース）で行ってください。
func (t TokenBlueprint) ValidateIconLink() error {
	if t.IconID == nil {
		return nil
	}
	if strings.TrimSpace(*t.IconID) == "" {
		return ErrInvalidIconID
	}
	return nil
}

// SetAssignee は与えられた Member の主キー（Member.ID）を AssigneeID に設定します。
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

// ValidateAssigneeLink は AssigneeID が空でないことを検証します。
// 実在性チェックは上位レイヤーで行ってください。
func (t TokenBlueprint) ValidateAssigneeLink() error {
	if strings.TrimSpace(t.AssigneeID) == "" {
		return ErrInvalidAssigneeID
	}
	return nil
}

// SetCreatedBy は与えられた Member の主キー（Member.ID）を CreatedBy に設定します。
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

// ValidateCreatedByLink は CreatedBy が空でないことを検証します。
// 実在性チェックは上位レイヤーで行ってください。
func (t TokenBlueprint) ValidateCreatedByLink() error {
	if strings.TrimSpace(t.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	return nil
}

// SetUpdatedBy は与えられた Member の主キー（Member.ID）を UpdatedBy に設定します。
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

// ValidateUpdatedByLink は UpdatedBy が空でないことを検証します。
func (t TokenBlueprint) ValidateUpdatedByLink() error {
	if strings.TrimSpace(t.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	return nil
}

// SetDeletedBy は与えられた Member の主キー（Member.ID）を DeletedBy に設定します。
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

// ClearDeletedBy は DeletedBy を解除します（NULL にします）。
func (t *TokenBlueprint) ClearDeletedBy() {
	t.DeletedBy = nil
}

// ValidateDeletedByLink は DeletedBy が存在する場合、空でないことを検証します。
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

// TokenBlueprintsTableDDL defines the SQL for the token_blueprints table migration.
const TokenBlueprintsTableDDL = `
-- Migration: Initialize TokenBlueprint domain
-- Mirrors backend/internal/domain/tokenBlueprint/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS token_blueprints (
  id               TEXT        PRIMARY KEY,
  name             TEXT        NOT NULL,
  symbol           TEXT        NOT NULL,
  brand_id         TEXT        NOT NULL,
  description      TEXT        NOT NULL,
  icon_id          TEXT,       -- icon の ID 参照（任意）
  content_files    TEXT[]      NOT NULL DEFAULT '{}',
  assignee_id      TEXT        NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by       TEXT        NOT NULL,
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by       TEXT        NOT NULL,
  deleted_at       TIMESTAMPTZ,
  deleted_by       TEXT,

  -- Non-empty checks
  CONSTRAINT chk_tb_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(name)) > 0
    AND char_length(trim(symbol)) > 0
    AND char_length(trim(brand_id)) > 0
    AND char_length(trim(description)) > 0
    AND char_length(trim(assignee_id)) > 0
    AND char_length(trim(created_by)) > 0
    AND char_length(trim(updated_by)) > 0
  ),

  -- Symbol format (matches ^[A-Z0-9]{1,10}$)
  CONSTRAINT chk_tb_symbol_format CHECK (symbol ~ '^[A-Z0-9]{1,10}$'),

  -- content_files: no empty items
  CONSTRAINT chk_tb_content_files_no_empty CHECK (
    NOT EXISTS (SELECT 1 FROM unnest(content_files) t(x) WHERE x = '')
  ),

  -- Time order
  CONSTRAINT chk_tb_time_order CHECK (
    updated_at >= created_at
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Optional FKs (add only if referenced tables exist)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'brands'
  ) THEN
    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_brand
        FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'members'
  ) THEN
    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_assignee
        FOREIGN KEY (assignee_id) REFERENCES members(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;

    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_created_by
        FOREIGN KEY (created_by) REFERENCES members(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;

    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_updated_by
        FOREIGN KEY (updated_by) REFERENCES members(id) ON DELETE RESTRICT;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;

    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES members(id) ON DELETE SET NULL;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'token_icons'
  ) THEN
    BEGIN
      ALTER TABLE token_blueprints
        ADD CONSTRAINT fk_tb_icon
        FOREIGN KEY (icon_id) REFERENCES token_icons(id) ON DELETE SET NULL;
    EXCEPTION WHEN duplicate_object THEN
      NULL;
    END;
  END IF;
END$$;

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_tb_brand_id    ON token_blueprints(brand_id);
CREATE INDEX IF NOT EXISTS idx_tb_symbol      ON token_blueprints(symbol);
CREATE INDEX IF NOT EXISTS idx_tb_created_at  ON token_blueprints(created_at);
CREATE INDEX IF NOT EXISTS idx_tb_updated_at  ON token_blueprints(updated_at);
CREATE INDEX IF NOT EXISTS idx_tb_icon_id     ON token_blueprints(icon_id);
CREATE INDEX IF NOT EXISTS idx_tb_updated_by  ON token_blueprints(updated_by);
CREATE INDEX IF NOT EXISTS idx_tb_deleted_by  ON token_blueprints(deleted_by);

COMMIT;
`
