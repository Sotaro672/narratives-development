// backend/internal/domain/company/service.go
package company

import (
	"context"
)

// Service provides read-only helpers for Company domain.
type Service struct {
	repo Repository
}

// NewService constructs a Service with the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetCompanyNameByID fetches a company by id and returns its normalized name.
//
// Returns:
//   - name (string): trimmed company name
//   - error:
//   - ErrNotFound … company not found OR the name is effectively empty
//   - repo-originated errors … passed through as-is
func (s *Service) GetCompanyNameByID(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", ErrNotFound
	}

	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	name := c.Name
	if name == "" {
		// 名称未設定の場合は NotFound 等、上位で同じ扱いにできるように揃える
		return "", ErrNotFound
	}

	return name, nil
}
