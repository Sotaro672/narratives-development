// backend/internal/application/usecase/account_usecase.go
package usecase

import (
	"context"
	"errors"

	accdom "narratives/internal/domain/account"
)

// AccountUsecase provides application-level operations for Account.
// Currently, only account lookup by ID is required.
type AccountUsecase struct {
	repo AccountRepo
}

// AccountRepo is the minimal repository contract needed by this use case.
type AccountRepo interface {
	GetByID(
		ctx context.Context,
		id string,
	) (accdom.Account, error)
}

// NewAccountUsecase creates an AccountUsecase.
func NewAccountUsecase(
	repo AccountRepo,
) *AccountUsecase {
	return &AccountUsecase{
		repo: repo,
	}
}

// GetByID returns an account by ID.
func (u *AccountUsecase) GetByID(
	ctx context.Context,
	id string,
) (accdom.Account, error) {
	if id == "" {
		return accdom.Account{}, errors.New("account: invalid id")
	}

	return u.repo.GetByID(ctx, id)
}
