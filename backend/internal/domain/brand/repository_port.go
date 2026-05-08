// backend/internal/domain/brand/repository_port.go
package brand

import (
	"context"
	"errors"
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

// ========================================
// Filter (align to common.FilterCommon / common.TimeRange)
// ========================================

// Filter は一覧取得条件です。
// common.FilterCommon を内包して SearchQuery/Created/Updated の表現を統一します。
// Brand 固有条件はこの struct に追加します。
type Filter struct {
	common.FilterCommon

	CompanyID     *string
	CompanyIDs    []string
	ManagerID     *string
	ManagerIDs    []string
	IsActive      *bool
	WalletAddress *string

	// Brand 固有: deleted の期間やフラグは必要なら残す
	DeletedFrom *time.Time
	DeletedTo   *time.Time
	Deleted     *bool
}

// 共通型エイリアス（インフラ非依存）
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

// Sort も共通を利用
type Sort = common.Sort
type SortOrder = common.SortOrder

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// 代表エラー（契約）
var (
	ErrNotFound = errors.New("brand: not found")
	ErrConflict = errors.New("brand: conflict")

	// ErrInvalidID は entity.go 側で定義済みのものを利用する

	ErrAssignedMemberReaderNotConfigured = errors.New("brand: assignedMemberReader not configured")
)

// ========================================
// Brand summary value objects
// ========================================

type NameIcon struct {
	Name      string `json:"name"`
	BrandIcon string `json:"brandIcon"`
}

type NameIconBackground struct {
	Name                 string `json:"name"`
	BrandIcon            string `json:"brandIcon"`
	BrandBackgroundImage string `json:"brandBackgroundImage"`
}

type BrandProfile struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	URL                  string `json:"websiteUrl"`
	BrandIcon            string `json:"brandIcon"`
	BrandBackgroundImage string `json:"brandBackgroundImage"`
	CompanyID            string `json:"companyId"`
}

// ========================================
// Port (Repository)
// ========================================

type Repository interface {
	// common.RepositoryList のシグネチャに合わせて sort を利用
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Brand], error)

	// Count は一覧の総件数取得に利用
	Count(ctx context.Context, filter Filter) (int, error)

	ListByCursor(ctx context.Context, filter Filter, cpage CursorPage) (CursorPageResult[Brand], error)

	GetByID(ctx context.Context, id string) (Brand, error)
	Exists(ctx context.Context, id string) (bool, error)

	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error

	Save(ctx context.Context, b Brand, opts *SaveOptions) (Brand, error)
}

// ========================================
// Additional query ports
// ========================================

type BrandNameIconReader interface {
	GetNameIconByID(ctx context.Context, brandID string) (NameIcon, error)
}

type BrandNameIconBackgroundReader interface {
	GetNameIconBackgroundByID(ctx context.Context, brandID string) (NameIconBackground, error)
}

type BrandProfileReader interface {
	GetBrandProfileByID(ctx context.Context, brandID string) (BrandProfile, error)
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
	if brandID == "" {
		return "", ErrInvalidID
	}

	b, err := s.repo.GetByID(ctx, brandID)
	if err != nil {
		return "", err
	}

	return b.Name, nil
}

// ========================================
// brandId から brandName + brandIcon を取得
// ========================================

func (s *Service) GetNameIconByID(ctx context.Context, brandID string) (NameIcon, error) {
	if brandID == "" {
		return NameIcon{}, ErrInvalidID
	}

	b, err := s.repo.GetByID(ctx, brandID)
	if err != nil {
		return NameIcon{}, err
	}

	return NameIcon{
		Name:      b.Name,
		BrandIcon: b.BrandIcon,
	}, nil
}

// ========================================
// brandId から brandName + brandIcon + brandBackgroundImage を取得
// ========================================

func (s *Service) GetNameIconBackgroundByID(ctx context.Context, brandID string) (NameIconBackground, error) {
	if brandID == "" {
		return NameIconBackground{}, ErrInvalidID
	}

	b, err := s.repo.GetByID(ctx, brandID)
	if err != nil {
		return NameIconBackground{}, err
	}

	return NameIconBackground{
		Name:                 b.Name,
		BrandIcon:            b.BrandIcon,
		BrandBackgroundImage: b.BrandBackgroundImage,
	}, nil
}

// ========================================
// brandId から brandName + brandIcon + brandBackgroundImage + description + url + companyId を取得
// ========================================

func (s *Service) GetBrandProfileByID(ctx context.Context, brandID string) (BrandProfile, error) {
	if brandID == "" {
		return BrandProfile{}, ErrInvalidID
	}

	b, err := s.repo.GetByID(ctx, brandID)
	if err != nil {
		return BrandProfile{}, err
	}

	return BrandProfile{
		Name:                 b.Name,
		Description:          b.Description,
		URL:                  b.URL,
		BrandIcon:            b.BrandIcon,
		BrandBackgroundImage: b.BrandBackgroundImage,
		CompanyID:            b.CompanyID,
	}, nil
}

// ========================================
// currentMember と同じ companyId を持つ Brand 一覧取得
// ========================================

func (s *Service) ListByCompanyID(
	ctx context.Context,
	companyID string,
	page Page,
) (PageResult[Brand], error) {

	if companyID == "" {
		return PageResult[Brand]{}, ErrInvalidID
	}

	filter := Filter{
		FilterCommon: common.FilterCommon{},
		CompanyID:    &companyID,
	}

	var sort Sort
	return s.repo.List(ctx, filter, sort, page)
}

// CountByCompanyID は companyId でスコープした件数を返します。
func (s *Service) CountByCompanyID(ctx context.Context, companyID string) (int, error) {
	if companyID == "" {
		return 0, ErrInvalidID
	}
	filter := Filter{
		FilterCommon: common.FilterCommon{},
		CompanyID:    &companyID,
	}
	return s.repo.Count(ctx, filter)
}

// ========================================
// assignedBrands から Member ID 一覧を取得
// ========================================

func (s *Service) ListAssignedMemberIDs(ctx context.Context, brandID string) ([]string, error) {
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

// ========================================
// Solana Brand Wallet 関連 (Domain メタ情報)
// ========================================

type SolanaBrandWallet struct {
	BrandID    string
	Address    string
	SecretName string
}

type MintAuthorityKey struct {
	Address    string
	SecretName string
}

type SolanaBrandWalletService interface {
	OpenBrandWallet(ctx context.Context, b Brand) (SolanaBrandWallet, error)
	FreezeBrandWallet(ctx context.Context, wallet SolanaBrandWallet) error
	DelegateTokenOperation(
		ctx context.Context,
		brandWallet SolanaBrandWallet,
		master MintAuthorityKey,
	) error
}
