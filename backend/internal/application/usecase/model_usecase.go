// backend\internal\application\usecase\model_usecase.go
package usecase

import (
	"context"
	"strings"

	modeldom "narratives/internal/domain/model"
)

// ModelRepo defines the minimal persistence port needed by ModelUsecase.
type ModelRepo interface {
	GetByID(ctx context.Context, id string) (modeldom.Model, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, m modeldom.Model) (modeldom.Model, error)
	Save(ctx context.Context, m modeldom.Model) (modeldom.Model, error)
	Delete(ctx context.Context, id string) error
}

// ModelUsecase orchestrates model operations.
type ModelUsecase struct {
	repo ModelRepo
}

func NewModelUsecase(repo ModelRepo) *ModelUsecase {
	return &ModelUsecase{repo: repo}
}

// Queries

func (u *ModelUsecase) GetByID(ctx context.Context, id string) (modeldom.Model, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *ModelUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *ModelUsecase) Create(ctx context.Context, m modeldom.Model) (modeldom.Model, error) {
	return u.repo.Create(ctx, m)
}

func (u *ModelUsecase) Save(ctx context.Context, m modeldom.Model) (modeldom.Model, error) {
	return u.repo.Save(ctx, m)
}

func (u *ModelUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
