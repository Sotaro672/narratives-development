// backend/internal/application/usecase/shipping_address_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	shipaddrdom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressRepo defines the minimal persistence port needed by ShippingAddressUsecase.
//
// 注意:
// - GetByID は「存在しない」を shipaddrdom.ErrNotFound で返す前提
// - Create は「既に存在する」を shipaddrdom.ErrConflict で返す前提
// - Update は「存在しない」を shipaddrdom.ErrNotFound で返す前提
// - Upsert(Save) は廃止し、起票は Create のみで行う
// - ListByUserID は userId に紐づく住所一覧を返す（1ユーザー複数住所）
// - docId = ShippingAddress.ID = UUID
// - UserID = owner uid
type ShippingAddressRepo interface {
	GetByID(ctx context.Context, id string) (*shipaddrdom.ShippingAddress, error)

	// Exists is optional-ish, but some callers may want it.
	// 実装が無い場合は GetByID で代替してください。
	Exists(ctx context.Context, id string) (bool, error)

	// Create creates a new document.
	Create(ctx context.Context, v shipaddrdom.ShippingAddress) (*shipaddrdom.ShippingAddress, error)

	// Update updates an existing document (no upsert).
	Update(ctx context.Context, v shipaddrdom.ShippingAddress) (*shipaddrdom.ShippingAddress, error)

	// ListByUserID lists documents by userId (1 user -> many addresses).
	ListByUserID(ctx context.Context, userID string) ([]shipaddrdom.ShippingAddress, error)

	Delete(ctx context.Context, id string) error
}

// ShippingAddressUsecase orchestrates shippingAddress operations.
type ShippingAddressUsecase struct {
	repo ShippingAddressRepo

	// NewShippingAddressUsecase で集中管理する可変依存。
	// テストでは同一 package 内から差し替え可能。
	newDocID func() string
	now      func() time.Time
}

func NewShippingAddressUsecase(repo ShippingAddressRepo) *ShippingAddressUsecase {
	return &ShippingAddressUsecase{
		repo:     repo,
		newDocID: uuid.NewString,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (u *ShippingAddressUsecase) ensureRepo() error {
	if u == nil || u.repo == nil {
		return errors.New("shippingAddress repo not configured")
	}
	if u.newDocID == nil {
		return errors.New("shippingAddress newDocID not configured")
	}
	if u.now == nil {
		return errors.New("shippingAddress now not configured")
	}
	return nil
}

func validateID(id string) (string, error) {
	if id == "" {
		return "", shipaddrdom.ErrInvalidID
	}
	return id, nil
}

func validateUID(uid string) (string, error) {
	if uid == "" {
		return "", shipaddrdom.ErrInvalidUserID
	}
	return uid, nil
}

// --------------------
// Queries
// --------------------

func (u *ShippingAddressUsecase) GetByID(ctx context.Context, id string) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	tid, err := validateID(id)
	if err != nil {
		return nil, err
	}

	return u.repo.GetByID(ctx, tid)
}

func (u *ShippingAddressUsecase) Exists(ctx context.Context, id string) (bool, error) {
	if err := u.ensureRepo(); err != nil {
		return false, err
	}

	tid, err := validateID(id)
	if err != nil {
		return false, err
	}

	return u.repo.Exists(ctx, tid)
}

// ListByUserID returns shipping addresses owned by the given user.
// docId は UUID のまま、userId で絞り込む。
func (u *ShippingAddressUsecase) ListByUserID(ctx context.Context, uid string) ([]shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	tuid, err := validateUID(uid)
	if err != nil {
		return nil, err
	}

	return u.repo.ListByUserID(ctx, tuid)
}

// --------------------
// Commands
// --------------------

// Create creates a new shipping address.
// - docId は usecase 側で UUID 採番する
// - userId は auth context 由来の uid で確定する
// - handler から渡された ID / UserID / CreatedAt / UpdatedAt には依存しない
func (u *ShippingAddressUsecase) Create(
	ctx context.Context,
	uid string,
	v shipaddrdom.ShippingAddress,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	tuid, err := validateUID(uid)
	if err != nil {
		return nil, err
	}

	ent, err := shipaddrdom.NewForCreateWithNow(
		tuid,
		v.ZipCode,
		v.State,
		v.City,
		v.Street,
		v.Street2,
		v.Country,
		u.now(),
	)
	if err != nil {
		return nil, err
	}

	ent.ID = u.newDocID()
	ent.UserID = tuid

	return u.repo.Create(ctx, ent)
}

// Update updates an existing shipping address.
// - 対象 docId は更新時に再採番しない
// - 更新前に本人の住所かチェックする
// - CreatedAt / ID / UserID は既存値を保持する
// - UpdatedAt は usecase 側の now() で更新する
func (u *ShippingAddressUsecase) Update(
	ctx context.Context,
	id string,
	uid string,
	v shipaddrdom.ShippingAddress,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	tid, err := validateID(id)
	if err != nil {
		return nil, err
	}

	tuid, err := validateUID(uid)
	if err != nil {
		return nil, err
	}

	current, err := u.repo.GetByID(ctx, tid)
	if err != nil {
		return nil, err
	}

	if current.UserID != tuid {
		return nil, shipaddrdom.ErrNotFound
	}

	if err := current.UpdateFromForm(
		v.ZipCode,
		v.State,
		v.City,
		v.Street,
		v.Street2,
		v.Country,
		u.now(),
	); err != nil {
		return nil, err
	}

	return u.repo.Update(ctx, *current)
}

// Delete deletes a shipping address by docId.
// 注意:
// - この method は本人チェックをしない
// - /mall/me/... 系では DeleteByUser を使うこと
func (u *ShippingAddressUsecase) Delete(ctx context.Context, id string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}

	tid, err := validateID(id)
	if err != nil {
		return err
	}

	return u.repo.Delete(ctx, tid)
}

// DeleteByUser deletes a shipping address after ownership check.
// /mall/me/... 系ではこちらを使う。
func (u *ShippingAddressUsecase) DeleteByUser(ctx context.Context, id string, uid string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}

	tid, err := validateID(id)
	if err != nil {
		return err
	}

	tuid, err := validateUID(uid)
	if err != nil {
		return err
	}

	current, err := u.repo.GetByID(ctx, tid)
	if err != nil {
		return err
	}

	if current.UserID != tuid {
		return shipaddrdom.ErrNotFound
	}

	return u.repo.Delete(ctx, tid)
}
