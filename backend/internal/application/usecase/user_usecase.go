// backend/internal/application/usecase/user_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	userdom "narratives/internal/domain/user"
)

// UserRepo defines the minimal persistence port needed by UserUsecase.
type UserRepo interface {
	GetByID(ctx context.Context, id string) (userdom.User, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, v userdom.User) (userdom.User, error)
	Save(ctx context.Context, v userdom.User) (userdom.User, error)
	Delete(ctx context.Context, id string) error
}

// UserUsecase orchestrates user operations.
type UserUsecase struct {
	repo UserRepo
	now  func() time.Time
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{
		repo: repo,
		now:  time.Now,
	}
}

func (u *UserUsecase) WithNow(now func() time.Time) *UserUsecase {
	if now != nil {
		u.now = now
	}
	return u
}

func (u *UserUsecase) ensureRepo() error {
	if u == nil || u.repo == nil {
		return errors.New("user repo not configured")
	}
	return nil
}

// --------------------
// Queries
// --------------------

func (u *UserUsecase) GetByID(ctx context.Context, id string) (userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return userdom.User{}, err
	}
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *UserUsecase) Exists(ctx context.Context, id string) (bool, error) {
	if err := u.ensureRepo(); err != nil {
		return false, err
	}
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// --------------------
// Commands
// --------------------

func (u *UserUsecase) Create(ctx context.Context, v userdom.User) (userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return userdom.User{}, err
	}
	normalizeUser(&v)
	return u.repo.Create(ctx, v)
}

func (u *UserUsecase) Save(ctx context.Context, v userdom.User) (userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return userdom.User{}, err
	}
	normalizeUser(&v)
	return u.repo.Save(ctx, v)
}

func (u *UserUsecase) Delete(ctx context.Context, id string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// Patch は handler 側で「部分フィールドだけ入った userdom.User を作って Save」しないための補助です。
// - 現在値を GetByID で読み、入力で指定されたものだけ上書きし、UpdatedAt を更新して Save します。
//
// ※ userdom.UpdateUserInput が domain 側に存在する前提（あなたの mall/user_handler.go が使用している型）
func (u *UserUsecase) Patch(ctx context.Context, id string, in userdom.UpdateUserInput) (userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return userdom.User{}, err
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return userdom.User{}, userdom.ErrInvalidID
	}

	cur, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return userdom.User{}, err
	}

	// merge（nil は維持）
	if in.FirstName != nil {
		cur.FirstName = trimPtr(in.FirstName)
	}
	if in.FirstNameKana != nil {
		cur.FirstNameKana = trimPtr(in.FirstNameKana)
	}
	if in.LastNameKana != nil {
		cur.LastNameKana = trimPtr(in.LastNameKana)
	}
	if in.LastName != nil {
		cur.LastName = trimPtr(in.LastName)
	}

	// deletedAt
	if in.DeletedAt != nil && !in.DeletedAt.IsZero() {
		t := in.DeletedAt.UTC()
		cur.DeletedAt = t
	}

	// touch updatedAt
	cur.UpdatedAt = u.now().UTC()

	normalizeUser(&cur)
	return u.repo.Save(ctx, cur)
}

// --------------------
// helpers
// --------------------

func normalizeUser(v *userdom.User) {
	if v == nil {
		return
	}
	v.ID = strings.TrimSpace(v.ID)

	// ポインタ文字列は trim（nil はそのまま）
	v.FirstName = trimPtr(v.FirstName)
	v.FirstNameKana = trimPtr(v.FirstNameKana)
	v.LastNameKana = trimPtr(v.LastNameKana)
	v.LastName = trimPtr(v.LastName)
}
