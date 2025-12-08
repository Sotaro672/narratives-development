// backend/internal/domain/brand/repository_port.go
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

	// ErrInvalidID は entity.go 側で定義済みのものを利用する

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

// ========================================
// Solana Brand Wallet 関連 (Domain メタ情報)
// ========================================

// SolanaBrandWallet は「ブランド専用ウォレット」のメタ情報を表します。
//   - BrandID: Firestore 上の brand ドキュメント ID
//   - Address: Solana ウォレットの公開鍵 (base58)
//   - SecretName: Secret Manager 上の秘密鍵保管先パス
//     例: projects/<project>/secrets/brand-wallet-<brandID>/versions/1
type SolanaBrandWallet struct {
	BrandID    string
	Address    string
	SecretName string
}

// MintAuthorityKey は「マスターウォレット（ミント権限）」のメタ情報を表します。
// ここでは domain 層なので、暗号ライブラリの実体ではなく
// 「どの鍵かを特定するための情報」だけを持たせています。
type MintAuthorityKey struct {
	Address    string // マスターウォレットの公開鍵 (base58)
	SecretName string // Secret Manager 上の秘密鍵パス
}

// SolanaBrandWalletService は brand 用 Solana ウォレットの開設・管理・権限委譲を扱うドメインサービスのポートです。
// 実装は infra/solana などのインフラレイヤーに置きます。
type SolanaBrandWalletService interface {
	// Brand 作成時に呼び出され、Brand 専用ウォレットを新規作成し、
	// Firestore に保存するためのメタ情報を返します。
	OpenBrandWallet(ctx context.Context, b Brand) (SolanaBrandWallet, error)

	// ブランドウォレットの「凍結」など、将来的な運用用フック。
	FreezeBrandWallet(ctx context.Context, wallet SolanaBrandWallet) error

	// マスターウォレットからブランドウォレットへのトークン運用権限委譲用フック。
	// 具体的な on-chain 実装はインフラ側で行う想定。
	DelegateTokenOperation(
		ctx context.Context,
		brandWallet SolanaBrandWallet,
		master MintAuthorityKey,
	) error
}
