// backend/internal/domain/brand/service.go
package brand

import (
	"context"
	"errors"
	"strings"
)

// ========================================
// Ports
// ========================================

// AssignedMemberReader は、ある brandID が assignedBrands に含まれている
// Member の ID 一覧を取得するためのポートインターフェースです。
// 実装は member ドメイン / Firestore アダプタ側に置きます。
type AssignedMemberReader interface {
	// brandID を assignedBrands に含む Member の ID 一覧を返す。
	ListMemberIDsByAssignedBrand(ctx context.Context, brandID string) ([]string, error)
}

// ========================================
// Service
// ========================================

// Service は brand 領域のユースケース的な便宜関数を提供します。
type Service struct {
	repo                 Repository
	assignedMemberReader AssignedMemberReader
}

// NewService は brand.Service を生成します。
// ※ 既存コードとの互換性維持用（assignedMemberReader なし）
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// NewServiceWithAssignedMember は、assignedBrands を使ったメンバー取得も行いたい場合に使うコンストラクタです。
func NewServiceWithAssignedMember(repo Repository, am AssignedMemberReader) *Service {
	return &Service{
		repo:                 repo,
		assignedMemberReader: am,
	}
}

// ========================================
// Errors
// ========================================

var (
	// assignedMemberReader が注入されていない状態で
	// ListAssignedMemberIDs を呼び出した場合に返すエラー。
	ErrAssignedMemberReaderNotConfigured = errors.New("brand: assignedMemberReader not configured")
)

// ========================================
// Existing: Brand 名取得
// ========================================

// GetNameByID は brandID から Brand を取得し、Name を返します。
// - brandID が空文字: ErrInvalidID を返す
// - repo.GetByID でエラー: そのまま返却（ErrNotFound など）
// - 正常: Brand.Name を trim した文字列を返却
func (s *Service) GetNameByID(ctx context.Context, brandID string) (string, error) {
	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		return "", ErrInvalidID
	}

	b, err := s.repo.GetByID(ctx, brandID)
	if err != nil {
		// ErrNotFound / その他のドメインエラーをそのまま返す
		return "", err
	}

	return formatBrandName(b.Name), nil
}

// formatBrandName は Brand 名の整形用ヘルパーです。
// 現状は trim するだけですが、将来 prefix/suffix などを付けたい場合に備えて分離。
func formatBrandName(name string) string {
	return strings.TrimSpace(name)
}

// ========================================
// New: assignedBrands から Member ID 一覧を取得
// ========================================

// ListAssignedMemberIDs は、指定した brandID を assignedBrands に含む
// Member の ID 一覧を返します。
//
// - brandID が空文字: ErrInvalidID を返す
// - assignedMemberReader が nil: ErrAssignedMemberReaderNotConfigured を返す
// - それ以外のエラー: assignedMemberReader からのエラーをそのまま返却
// - 正常: 空文字を除外し、trim & 重複排除した memberID 一覧を返却
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

	// 正規化（trim & 空文字除外 & 重複排除）
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
