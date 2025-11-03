package usecase

import (
    "context"
    "errors"
    "strings"
    "time"

    accdom "narratives/internal/domain/account"
)

// AccountUsecase provides application-level operations for Account.
// It wraps the domain repository and offers simple pass-through methods.
// Only a minimal subset (GetByID) is required; create/update/delete are
// invoked when the underlying repository supports them.
type AccountUsecase struct {
    repo AccountRepo

    now func() time.Time
}

// AccountRepo is the minimal repository contract needed by this use case.
// Your domain repository should satisfy this interface.
type AccountRepo interface {
    GetByID(ctx context.Context, id string) (accdom.Account, error)
}

// Optional sub-interfaces. If the underlying repo implements them,
// corresponding use case methods will work; otherwise they return an error.

type accountCreator interface {
    Create(ctx context.Context, a accdom.Account) (accdom.Account, error)
}

type accountUpdater interface {
    Update(ctx context.Context, a accdom.Account) (accdom.Account, error)
}

type accountDeleter interface {
    Delete(ctx context.Context, id string) error
}

// Constructor
func NewAccountUsecase(repo AccountRepo) *AccountUsecase {
    return &AccountUsecase{
        repo: repo,
        now:  time.Now,
    }
}

// Override time provider (for testing)
func (u *AccountUsecase) WithNow(now func() time.Time) *AccountUsecase {
    u.now = now
    return u
}

// ============ Queries ============

// GetByID returns an account by ID.
func (u *AccountUsecase) GetByID(ctx context.Context, id string) (accdom.Account, error) {
    id = strings.TrimSpace(id)
    if id == "" {
        return accdom.Account{}, errors.New("account: invalid id")
    }
    return u.repo.GetByID(ctx, id)
}

// ============ Commands ============

// CreateAccount creates a new account using the repository if supported.
// The caller is responsible for constructing a valid accdom.Account entity.
func (u *AccountUsecase) CreateAccount(ctx context.Context, a accdom.Account) (accdom.Account, error) {
    if c, ok := u.repo.(accountCreator); ok {
        return c.Create(ctx, a)
    }
    return accdom.Account{}, errors.New("account: create not supported by repository")
}

// UpdateAccount updates an account using the repository if supported.
// The caller should provide a mutated accdom.Account reflecting desired changes.
func (u *AccountUsecase) UpdateAccount(ctx context.Context, a accdom.Account) (accdom.Account, error) {
    if up, ok := u.repo.(accountUpdater); ok {
        return up.Update(ctx, a)
    }
    return accdom.Account{}, errors.New("account: update not supported by repository")
}

// DeleteAccount deletes an account by ID if the repository supports deletion.
func (u *AccountUsecase) DeleteAccount(ctx context.Context, id string) error {
    id = strings.TrimSpace(id)
    if id == "" {
        return errors.New("account: invalid id")
    }
    if d, ok := u.repo.(accountDeleter); ok {
        return d.Delete(ctx, id)
    }
    return errors.New("account: delete not supported by repository")
}