// backend/internal/domain/company/service.go
package company

import (
	"context"
	"strings"
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
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return "", ErrNotFound
	}

	c, err := s.repo.GetByID(ctx, trimmedID)
	if err != nil {
		return "", err
	}

	name := strings.TrimSpace(c.Name)
	if name == "" {
		// 名称未設定の場合は NotFound 等、上位で同じ扱いにできるように揃える
		return "", ErrNotFound
	}
	return name, nil
}

// TryGetCompanyName is a convenience wrapper that does not treat empty name as error.
// It returns (name, ok, err), where ok indicates existence.
//
// Example use:
//
//	if name, ok, err := svc.TryGetCompanyName(ctx, id); err != nil { ... }
func (s *Service) TryGetCompanyName(ctx context.Context, id string) (string, bool, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return "", false, ErrNotFound
	}

	c, err := s.repo.GetByID(ctx, trimmedID)
	if err != nil {
		if err == ErrNotFound {
			return "", false, nil
		}
		return "", false, err
	}

	return strings.TrimSpace(c.Name), true, nil
}
