// backend/internal/domain/productBlueprint/service.go
package productBlueprint

import (
	"context"
	"strings"
)

// ========================================
// Reader Repository (Service 用の小さいポート)
// ========================================

// ReaderRepository は Service が利用する最小限の読み取り専用ポートです。
// 既存の Repository インターフェースはより多くのメソッドを持ちますが、
// Service 側では「ID から ProductBlueprint を 1 件取ってくる」だけに依存させることで
// インターフェースを小さく保ちます。
type ReaderRepository interface {
	GetByID(ctx context.Context, id string) (ProductBlueprint, error)
}

// Service は productBlueprint 領域の便宜関数を提供します。
// 内部では ReaderRepository のみを利用し、書き込み系の責務は持ちません。
type Service struct {
	repo ReaderRepository
}

// NewService は productBlueprint.Service を生成します。
// 引数には ReaderRepository（GetByID だけを持つ小さいポート）を受け取ります。
func NewService(repo ReaderRepository) *Service {
	return &Service{repo: repo}
}

// GetProductNameByID は productBlueprintID から ProductBlueprint を取得し、
// ProductName を返します。
// - id が空の場合: ErrInvalidID
// - repo.GetByID が ErrNotFound などを返した場合: そのまま返却
// - ProductName が空の場合: 空文字をそのまま返却
func (s *Service) GetProductNameByID(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", ErrInvalidID
	}

	pb, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// ErrNotFound などドメインエラーはそのまま上位へ
		return "", err
	}

	return strings.TrimSpace(pb.ProductName), nil
}

// GetBrandIDByID は productBlueprintID から ProductBlueprint を取得し、
// BrandID を返します。
// - id が空の場合: ErrInvalidID
// - repo.GetByID が ErrNotFound などを返した場合: そのまま返却
// - BrandID が空の場合: 空文字をそのまま返却
func (s *Service) GetBrandIDByID(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", ErrInvalidID
	}

	pb, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// ErrNotFound などドメインエラーはそのまま上位へ
		return "", err
	}

	return strings.TrimSpace(pb.BrandID), nil
}
