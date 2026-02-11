// backend/internal/application/usecase/shipping_address_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	shipaddrdom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressRepo defines the minimal persistence port needed by ShippingAddressUsecase.
//
// ✅ 注意:
// - GetByID は「存在しない」を shipaddrdom.ErrNotFound で返す前提
// - Create は「既に存在する」を shipaddrdom.ErrConflict で返す前提
// - Update は「存在しない」を shipaddrdom.ErrNotFound で返す前提
// - Upsert(Save) は廃止し、起票は Create のみで行う
// - ListByUserID は userId に紐づく住所一覧を返す（1ユーザー複数住所）
type ShippingAddressRepo interface {
	GetByID(ctx context.Context, id string) (*shipaddrdom.ShippingAddress, error)

	// Exists is optional-ish, but some callers may want it.
	// 実装が無い場合は GetByID で代替してください（この usecase 内では使いません）。
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
}

func NewShippingAddressUsecase(repo ShippingAddressRepo) *ShippingAddressUsecase {
	return &ShippingAddressUsecase{repo: repo}
}

func (u *ShippingAddressUsecase) ensureRepo() error {
	if u == nil || u.repo == nil {
		return errors.New("shippingAddress repo not configured")
	}
	return nil
}

func trimID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", shipaddrdom.ErrInvalidID
	}
	return id, nil
}

func trimUID(uid string) (string, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		// userId の不正なので ErrInvalidUserID を返すのが自然
		return "", shipaddrdom.ErrInvalidUserID
	}
	return uid, nil
}

func newDocID() string {
	// ランダム docId（UUID）を採番
	return uuid.NewString()
}

// --------------------
// Queries
// --------------------

func (u *ShippingAddressUsecase) GetByID(ctx context.Context, id string) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}
	tid, err := trimID(id)
	if err != nil {
		return nil, err
	}
	return u.repo.GetByID(ctx, tid)
}

func (u *ShippingAddressUsecase) Exists(ctx context.Context, id string) (bool, error) {
	if err := u.ensureRepo(); err != nil {
		return false, err
	}
	tid, err := trimID(id)
	if err != nil {
		return false, err
	}
	return u.repo.Exists(ctx, tid)
}

// ListByUserID: userId に紐づく住所一覧を返す（docID は UUID のまま）。
func (u *ShippingAddressUsecase) ListByUserID(ctx context.Context, uid string) ([]shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}
	tuid, err := trimUID(uid)
	if err != nil {
		return nil, err
	}
	return u.repo.ListByUserID(ctx, tuid)
}

// --------------------
// Commands
// --------------------

// Create: 起票は create のみ。
// ✅ 重要: Create 時は「id なし」を許容するコンストラクタで必ず作り直す
// - docId は usecase 側で採番
// - uid は userId として確定させる
func (u *ShippingAddressUsecase) Create(
	ctx context.Context,
	uid string,
	v shipaddrdom.ShippingAddress,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	tuid, err := trimUID(uid)
	if err != nil {
		return nil, err
	}

	// ✅ handler 側で v.ID="" のまま渡されてもここで「Create用」に作り直す
	//    （ドメインの validate が ID 必須でも、Create 用コンストラクタなら通せる）
	ent, err := shipaddrdom.NewForCreateWithNow(
		tuid,
		v.ZipCode,
		v.State,
		v.City,
		v.Street,
		v.Street2,
		v.Country,
		v.CreatedAt, // handler が now を入れている想定。ゼロならドメイン側で弾く設計でもOK
	)
	if err != nil {
		return nil, err
	}

	// docId は usecase 側で採番（外部入力に依存しない）
	ent.ID = newDocID()

	// userId は uid で確定（念のため再代入）
	ent.UserID = tuid

	return u.repo.Create(ctx, ent)
}

// Update: 既存前提で更新する（無ければ ErrNotFound）。docId は採番し直さない。
// ✅ 重要: 本人の住所かチェックする（他人のIDを更新させない）
func (u *ShippingAddressUsecase) Update(
	ctx context.Context,
	id string,
	uid string,
	v shipaddrdom.ShippingAddress,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	tid, err := trimID(id)
	if err != nil {
		return nil, err
	}
	tuid, err := trimUID(uid)
	if err != nil {
		return nil, err
	}

	// ✅ まず既存取得（存在しなければ ErrNotFound）
	current, err := u.repo.GetByID(ctx, tid)
	if err != nil {
		return nil, err
	}

	// ✅ 本人チェック（違う場合は not found 扱い）
	if strings.TrimSpace(current.UserID) != tuid {
		return nil, shipaddrdom.ErrNotFound
	}

	// 対象 docId を固定（再採番しない）
	v.ID = tid

	// uid を userId に格納（本人のアドレスだけ更新させる）
	v.UserID = tuid

	return u.repo.Update(ctx, v)
}

// Delete: 既存のシグネチャ維持（handler 側が uid を渡していないため）
// ⚠️ 推奨: 可能なら handler/usecase を改修し DeleteByUser を使う（本人チェック）
//
// 現状は docId が分かれば消せてしまう可能性があるため、
// handler を修正できるなら DeleteByUser(ctx, id, uid) を使ってください。
func (u *ShippingAddressUsecase) Delete(ctx context.Context, id string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}
	tid, err := trimID(id)
	if err != nil {
		return err
	}
	return u.repo.Delete(ctx, tid)
}

// DeleteByUser: 本人チェック付き delete（推奨）
func (u *ShippingAddressUsecase) DeleteByUser(ctx context.Context, id string, uid string) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}

	tid, err := trimID(id)
	if err != nil {
		return err
	}
	tuid, err := trimUID(uid)
	if err != nil {
		return err
	}

	current, err := u.repo.GetByID(ctx, tid)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.UserID) != tuid {
		return shipaddrdom.ErrNotFound
	}

	return u.repo.Delete(ctx, tid)
}
