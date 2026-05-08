// backend/internal/domain/member/repository_port.go
package member

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Filter defines list conditions for members.
// NOTE: Field names/semantics align with entity.Member (camelCase in JSON/Firestore).
type Filter struct {
	// Firebase Auth UID.
	UID string

	// Free-text query for name/kana/email, etc.
	SearchQuery string

	// Brand filters
	BrandIDs []string

	// Company scope & status
	CompanyID string // owning company to scope results
	Status    string // "", "active", "inactive"

	// Ranges
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// Permission names (AND)
	Permissions []string
}

// Common aliases
type Page = common.Page
type PageResult = common.PageResult[Member]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Member]
type SaveOptions = common.SaveOptions

// ============================================================
// DTOs for application/handler layers
// entity に id を持たせず、docId は外側のDTOで扱う
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
type Repository interface {
	// Common CRUD/List
	common.RepositoryCRUD[Member, MemberPatch]
	common.RepositoryList[Member, Filter]

	// Additional requirements
	ListByCursor(ctx context.Context, filter Filter, cpage CursorPage) (CursorPageResult, error)
	GetByID(ctx context.Context, id string) (Member, error)
	GetByEmail(ctx context.Context, email string) (Member, error)

	// GetByFirebaseUID returns a member whose uid field matches the Firebase Auth UID.
	// Firestore document ID and Firebase Auth UID are intentionally separated.
	GetByFirebaseUID(ctx context.Context, firebaseUID string) (Member, error)

	// GetRecordByFirebaseUID returns a member record whose uid field matches the Firebase Auth UID.
	// Use this when the caller also needs the Firestore document ID.
	GetRecordByFirebaseUID(ctx context.Context, firebaseUID string) (Record, error)

	Exists(ctx context.Context, id string) (bool, error)

	// Save persists a member with the repository's default document ID behavior.
	Save(ctx context.Context, m Member, opts *SaveOptions) (Member, error)

	// SaveByDocID persists a member using the specified document ID explicitly.
	// This is used when updating an existing member document, including invitation flow.
	SaveByDocID(ctx context.Context, docID string, m Member, opts *SaveOptions) (Member, error)

	// ========================================================
	// docId を application / handler 層に返すための DTO 用 API
	// ========================================================

	// CreateWithDocID creates a member and returns the created Firestore docId.
	CreateWithDocID(ctx context.Context, m Member) (Record, error)

	// GetByDocID returns a member record with its docId.
	GetByDocID(ctx context.Context, docID string) (Record, error)

	// ListWithDocID returns page results including each member's docId.
	ListWithDocID(ctx context.Context, f Filter, s common.Sort, p Page) (RecordPageResult, error)
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
// currentMember の companyId を使って Member を一覧取得するヘルパー
// ============================================================

// ListMembersByCompanyID は、与えられた companyID でスコープされた
// Member 一覧（カーソルページ）を取得するドメインヘルパーです。
// auth ミドルウェアで context に積まれた companyId を取得したあと、
// ハンドラー側で呼び出すことを想定しています。
func ListMembersByCompanyID(
	ctx context.Context,
	repo Repository,
	companyID string,
	cpage CursorPage,
) (CursorPageResult, error) {
	cid := companyID
	if cid == "" {
		return CursorPageResult{}, errors.New("member: companyID is empty")
	}

	filter := Filter{
		CompanyID: cid,
	}

	return repo.ListByCursor(ctx, filter, cpage)
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

// GetNameLastFirstByID は memberID から Member を取得し、
// 「lastName firstName」の順で整形した表示名を返します。
// - lastName / firstName の両方が存在: "last first"
// - 片方のみ存在: その値のみ
// - どちらも空: ""
func (s *Service) GetNameLastFirstByID(ctx context.Context, memberID string) (string, error) {
	if memberID == "" {
		return "", errors.New("member: memberID is empty")
	}

	m, err := s.repo.GetByID(ctx, memberID)
	if err != nil {
		return "", err
	}

	return FormatLastFirst(m.LastName, m.FirstName), nil
}

// GetNameLastFirstByDocID は docId から Member を取得し、
// 「lastName firstName」の順で整形した表示名を返します。
func (s *Service) GetNameLastFirstByDocID(ctx context.Context, docID string) (string, error) {
	if docID == "" {
		return "", errors.New("member: docID is empty")
	}

	rec, err := s.repo.GetByDocID(ctx, docID)
	if err != nil {
		return "", err
	}

	return FormatLastFirst(rec.Member.LastName, rec.Member.FirstName), nil
}

// FormatLastFirst は「姓→名」の順で半角スペース区切りの表示名を返します。
// 空要素は除外され、両方空の場合は空文字を返します。
func FormatLastFirst(lastName, firstName string) string {
	ln := lastName
	fn := firstName

	switch {
	case ln != "" && fn != "":
		return ln + " " + fn
	case ln != "":
		return ln
	case fn != "":
		return fn
	default:
		return ""
	}
}
