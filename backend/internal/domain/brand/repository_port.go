package brand

import (
	"context"
	"errors"
	"strings"
	"time"

	common "narratives/internal/domain/common"
)

// ========================================
// Ports (RepositoryPort: query-friendly repository interface)
// ========================================

// RepositoryPort defines the data access interface for brand domain (query-friendly).
type RepositoryPort interface {
	List(ctx context.Context, filter Filter, page Page) (PageResult[Brand], error)
	ListByCursor(ctx context.Context, filter Filter, cpage CursorPage) (CursorPageResult[Brand], error)
	GetByID(ctx context.Context, id string) (Brand, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error
	Save(ctx context.Context, b Brand, opts *SaveOptions) (Brand, error)
}

// Filter / Page 構造体（一覧取得用）
type Filter struct {
	SearchQuery string

	CompanyID     *string
	CompanyIDs    []string
	ManagerID     *string
	ManagerIDs    []string
	IsActive      *bool
	WalletAddress *string

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time

	Deleted *bool
}

// 共通型エイリアス（インフラ非依存）
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

// 代表エラー（契約）
var (
	ErrNotFound = errors.New("brand: not found")
	ErrConflict = errors.New("brand: conflict")

	ErrAssignedMemberReaderNotConfigured = errors.New("brand: assignedMemberReader not configured")
)

// ========================================
// Port (Repository)
// ========================================

type Repository interface {
	List(ctx context.Context, filter Filter, page Page) (PageResult[Brand], error)
	ListByCursor(ctx context.Context, filter Filter, cpage CursorPage) (CursorPageResult[Brand], error)

	GetByID(ctx context.Context, id string) (Brand, error)
	Exists(ctx context.Context, id string) (bool, error)

	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error

	Save(ctx context.Context, b Brand, opts *SaveOptions) (Brand, error)
}

// ========================================
// AssignedMemberReader Port
// ========================================

type AssignedMemberReader interface {
	ListMemberIDsByAssignedBrand(ctx context.Context, brandID string) ([]string, error)
}

// ========================================
// Service
// ========================================

type Service struct {
	repo                 Repository
	assignedMemberReader AssignedMemberReader
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithAssignedMember(repo Repository, am AssignedMemberReader) *Service {
	return &Service{
		repo:                 repo,
		assignedMemberReader: am,
	}
}

// ========================================
// Brand 名取得
// ========================================

func (s *Service) GetNameByID(ctx context.Context, brandID string) (string, error) {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", ErrInvalidID
	}

	b, err := s.repo.GetByID(ctx, brandID)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(b.Name), nil
}

// ========================================
// currentMember と同じ companyId を持つ Brand 一覧取得
// ========================================

func (s *Service) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page Page,
) (PageResult[Brand], error) {

	cid := strings.TrimSpace(companyID)
	if cid == "" {
		return PageResult[Brand]{}, ErrInvalidID
	}

	filter := Filter{
		CompanyID: &cid,
	}

	return s.repo.List(ctx, filter, page)
}

// ========================================
// assignedBrands から Member ID 一覧を取得
// ========================================

func (s *Service) ListAssignedMemberIDs(ctx context.Context, brandID string) ([]string, error) {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return nil, ErrInvalidID
	}

	if s.assignedMemberReader == nil {
		return nil, ErrAssignedMemberReaderNotConfigured
	}

	rawIDs, err := s.assignedMemberReader.ListMemberIDsByAssignedBrand(ctx, brandID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(rawIDs))
	result := make([]string, 0, len(rawIDs))

	for _, id := range rawIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	return result, nil
}
