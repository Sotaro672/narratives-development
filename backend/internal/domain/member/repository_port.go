// backend/internal/domain/member/repository_port.go
package member

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Filter defines list conditions for members.
// NOTE:
//   - List scope is always companyID through ListByCompanyID.
//   - CompanyID is not included here to avoid duplicated scope sources.
type Filter struct {
	// Firebase Auth UID.
	UID string

	// Free-text query for name/kana/email, etc.
	SearchQuery string

	// Brand filters.
	BrandIDs []string

	// Status filter.
	// "", "active", "inactive"
	Status string

	// Ranges.
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// Permission names.
	Permissions []string
}

// Common aliases.
type Page = common.Page

// ============================================================
// DTOs for application/handler layers
// entity に id を持たせず、docId は外側の DTO で扱う
// ============================================================

type Record struct {
	DocID  string
	Member Member
}

type RecordPageResult struct {
	Items      []Record
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// Repository is the persistence port for the Member aggregate.
//
// 方針:
//   - Get 系は GetByID に統一する。
//   - entity.Member には docId を持たせないため、GetByID は Record を返す。
//   - Exists は廃止する。存在確認は GetByID の ErrNotFound で判定する。
//   - Save は廃止し、Update に置き換える。
//   - List 系は ListByCompanyID に統一する。
//   - Firebase UID / email などでの検索が必要な場合は ListByCompanyID + Filter を使う。
type Repository interface {
	// Create creates a member and returns the created Firestore docId.
	Create(ctx context.Context, m Member) (Record, error)

	// GetByID returns a member record by Firestore document ID.
	// This is the only Get method in this repository port.
	GetByID(ctx context.Context, id string) (Record, error)

	// Update updates an existing member by Firestore document ID.
	// This replaces Save / SaveByDocID.
	Update(ctx context.Context, id string, patch MemberPatch) (Record, error)

	// Delete deletes a member by Firestore document ID.
	Delete(ctx context.Context, id string) error

	// ListByCompanyID returns members scoped by companyID.
	// This is the only List method in this repository port.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
		filter Filter,
		page Page,
	) (RecordPageResult, error)
}

var (
	// 招待トークンが見つからない場合
	ErrInvitationTokenNotFound = errors.New("invitation: token not found")
	// 必要なら衝突エラーなども今後追加可能
	// ErrInvitationTokenConflict = errors.New("invitation: token conflict")
)

// ============================================================
// InvitationToken 用 Repository ポート（ヘキサゴナルの out ポート）
// ============================================================

type InvitationTokenRepository interface {
	// トークン文字列から InvitationInfo を取得
	// 見つからない場合は ErrInvitationTokenNotFound を返す
	ResolveInvitationInfoByToken(ctx context.Context, token string) (InvitationInfo, error)

	// InvitationInfo をもとに新しい招待トークンを作成し、token 文字列を返す
	CreateInvitationToken(ctx context.Context, info InvitationInfo) (string, error)

	// 招待トークンを消費済みにする
	ConsumeInvitationToken(ctx context.Context, token string) error
}

// ============================================================
// Service: member 領域のユースケース的な便宜関数
// ============================================================

// Service は member 領域のユースケース的な便宜関数を提供します。
type Service struct {
	repo Repository
}

// NewService は member.Service を生成します。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetNameLastFirstByID は memberID/docID から Member を取得し、
// 「lastName firstName」の順で整形した表示名を返します。
// - lastName / firstName の両方が存在: "last first"
// - 片方のみ存在: その値のみ
// - どちらも空: ""
func (s *Service) GetNameLastFirstByID(ctx context.Context, memberID string) (string, error) {
	if memberID == "" {
		return "", errors.New("member: memberID is empty")
	}

	rec, err := s.repo.GetByID(ctx, memberID)
	if err != nil {
		return "", err
	}

	return FormatLastFirst(rec.Member.LastName, rec.Member.FirstName), nil
}

// FormatLastFirst は「姓→名」の順で半角スペース区切りの表示名を返します。
// 空要素は除外され、両方空の場合は空文字を返します。
func FormatLastFirst(lastName, firstName string) string {
	switch {
	case lastName != "" && firstName != "":
		return lastName + " " + firstName
	case lastName != "":
		return lastName
	case firstName != "":
		return firstName
	default:
		return ""
	}
}
