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
// entityにIDを持たせず、docIDは外側のDTOで扱う
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
//   - docIDによる単体取得はGetByIDを使う。
//   - Firebase UIDによる単体取得はGetByUIDを使う。
//   - entity.MemberにはdocIDを持たせないため、GetByIDとGetByUIDはRecordを返す。
//   - Existsは廃止する。存在確認はGetByIDまたはGetByUIDのErrNotFoundで判定する。
//   - Saveは廃止し、Updateに置き換える。
//   - Deleteは物理削除として扱う。DeletedAtまたはDeletedByによる論理削除は扱わない。
//   - List系はListByCompanyIDに統一する。
//   - emailなどでの検索が必要な場合はListByCompanyIDとFilterを使う。
type Repository interface {
	// Create creates a member and returns the created Firestore document ID.
	Create(ctx context.Context, m Member) (Record, error)

	// GetByID returns a member record by Firestore document ID.
	GetByID(ctx context.Context, id string) (Record, error)

	// GetByUID returns a member record by Firebase Auth UID.
	GetByUID(ctx context.Context, uid string) (Record, error)

	// Update updates an existing member by Firestore document ID.
	// This replaces Save and SaveByDocID.
	Update(ctx context.Context, id string, patch MemberPatch) (Record, error)

	// Delete physically deletes a member by Firestore document ID.
	// It does not perform logical deletion with DeletedAt or DeletedBy.
	Delete(ctx context.Context, id string) error

	// ListByCompanyID returns members scoped by companyID.
	// This is the only list method in this repository port.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
		filter Filter,
		page Page,
	) (RecordPageResult, error)
}

// ============================================================
// Service: member領域のユースケース的な便宜関数
// ============================================================

// Serviceはmember領域のユースケース的な便宜関数を提供します。
type Service struct {
	repo Repository
}

// NewServiceはmember.Serviceを生成します。
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetNameLastFirstByUIDはFirebase Auth UIDからMemberを取得し、
// 「lastName firstName」の順で整形した表示名を返します。
// - lastNameとfirstNameの両方が存在: "last first"
// - 片方のみ存在: その値のみ
// - どちらも空: ""
func (s *Service) GetNameLastFirstByUID(
	ctx context.Context,
	uid string,
) (string, error) {
	if uid == "" {
		return "", errors.New("member: uid is empty")
	}

	rec, err := s.repo.GetByUID(ctx, uid)
	if err != nil {
		return "", err
	}

	return FormatLastFirst(
		rec.Member.LastName,
		rec.Member.FirstName,
	), nil
}

// FormatLastFirstは「姓→名」の順で半角スペース区切りの表示名を返します。
// 空要素は除外され、両方空の場合は空文字を返します。
func FormatLastFirst(lastName string, firstName string) string {
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
