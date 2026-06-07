// backend/internal/application/usecase/user_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	userdom "narratives/internal/domain/user"
)

// domain/user/repository_port.go を正として採用
// UserUsecase 側で独自 interface を再定義せず、repository port と完全一致させる。
type UserRepo = userdom.RepositoryPort

type UserUsecase struct {
	repo UserRepo
	now  func() time.Time
}

// NewUserUsecase is the only entry point for constructing UserUsecase.
//
// すべての依存はここに集約する。
// - repo: user repository
// - now: 時刻取得関数。nil の場合は time.Now を使用する。
func NewUserUsecase(repo UserRepo, now func() time.Time) *UserUsecase {
	if now == nil {
		now = time.Now
	}

	return &UserUsecase{
		repo: repo,
		now:  now,
	}
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

func (u *UserUsecase) GetByID(ctx context.Context, id string) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	return u.repo.GetByID(ctx, id)
}

// --------------------
// Commands
// --------------------

// Create: docId=uid で新規作成（既存なら ErrConflict）
// - createdAt/updatedAt はサーバが決める（入力は上書き）
// - deletedAt は caller 指定を尊重（nil/zero/非zero OK）
func (u *UserUsecase) Create(ctx context.Context, id string, in userdom.CreateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	now := u.now().UTC()
	in.CreatedAt = &now
	in.UpdatedAt = &now

	return u.repo.Create(ctx, id, in)
}

// Update: users/{id} を部分更新
// - updatedAt はサーバが決める（未指定なら上書き）
// - deletedAt は caller 指定を尊重（nil=変更なし / zero=not deleted / non-zero=deleted）
func (u *UserUsecase) Update(ctx context.Context, id string, in userdom.UpdateUserInput) (*userdom.User, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	if id == "" {
		return nil, userdom.ErrInvalidID
	}

	if in.UpdatedAt == nil || in.UpdatedAt.IsZero() {
		now := u.now().UTC()
		in.UpdatedAt = &now
	}

	return u.repo.Update(ctx, id, in)
}

// Delete: users/{id} を削除
func (u *UserUsecase) Delete(ctx context.Context, id string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}

	if id == "" {
		return userdom.ErrInvalidID
	}

	return u.repo.Delete(ctx, id)
}
