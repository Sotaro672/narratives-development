// backend/internal/domain/brand/service.go
package brand

import (
	"context"
	"strings"
)

// Service は brand 領域のユースケース的な便宜関数を提供します。
type Service struct {
	repo Repository
}

// NewService は brand.Service を生成します。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

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
