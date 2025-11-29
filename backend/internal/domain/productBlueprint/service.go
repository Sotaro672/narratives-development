// backend/internal/domain/productBlueprint/service.go
package productBlueprint

import (
	"context"
	"strings"
)

// Service は productBlueprint 領域の便宜関数を提供します。
// Repository は同一パッケージ内（repository.go など）で
//
//	type Repository interface {
//	    GetByID(ctx context.Context, id string) (ProductBlueprint, error)
//	    // ...（他のメソッドがあれば）
//	}
//
// のように定義されている前提です。
type Service struct {
	repo Repository
}

// NewService は productBlueprint.Service を生成します。
func NewService(repo Repository) *Service {
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
