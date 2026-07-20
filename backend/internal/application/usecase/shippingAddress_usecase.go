// backend/internal/application/usecase/shipping_address_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	shipaddrdom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressRepoはDomain層のRepositoryPortをそのまま使用します。
// Application層では同じinterfaceを再定義しません。
type ShippingAddressRepo = shipaddrdom.RepositoryPort

// CreateShippingAddressInputは配送先住所の新規作成入力です。
//
// ID、UserID、CreatedAtおよびUpdatedAtは受け取りません。
// IDはUsecaseがUUIDを採番し、UserIDは認証UIDから設定し、
// 時刻はUsecaseのserver clockから設定します。
type CreateShippingAddressInput struct {
	ZipCode string
	State   string
	City    string
	Street  string
	Street2 string
	Country string
}

// UpdateShippingAddressInputは配送先住所の部分更新入力です。
//
// nilは変更なしを表します。
// Street2は任意項目であるため、空文字を指定すると明示的に消去できます。
// Countryへ空文字を指定した場合は、Domain規則によりJPへ正規化されます。
type UpdateShippingAddressInput struct {
	ZipCode *string
	State   *string
	City    *string
	Street  *string
	Street2 *string
	Country *string
}

// ShippingAddressUsecaseは配送先住所に関する処理を制御します。
type ShippingAddressUsecase struct {
	repo ShippingAddressRepo

	// テスト時に差し替え可能な依存です。
	newDocID func() string
	now      func() time.Time
}

// NewShippingAddressUsecaseはShippingAddressUsecaseを生成します。
//
// repoがnilの場合でもconstructor自体は失敗させず、各メソッドの
// ensureRepoでエラーにします。HTTP HandlerではUsecase自体がnilの場合に
// 503を返すguardを追加します。
func NewShippingAddressUsecase(
	repo ShippingAddressRepo,
) *ShippingAddressUsecase {
	return &ShippingAddressUsecase{
		repo:     repo,
		newDocID: uuid.NewString,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (u *ShippingAddressUsecase) ensureRepo() error {
	if u == nil {
		return errors.New("shippingAddress usecase is nil")
	}

	if u.repo == nil {
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

func validateShippingAddressID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", shipaddrdom.ErrInvalidID
	}

	if _, err := uuid.Parse(id); err != nil {
		return "", shipaddrdom.ErrInvalidID
	}

	return id, nil
}

func validateShippingAddressUID(uid string) (string, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", shipaddrdom.ErrInvalidUserID
	}

	if len([]rune(uid)) > shipaddrdom.MaxUserIDLength {
		return "", shipaddrdom.ErrInvalidUserID
	}

	return uid, nil
}

// --------------------
// Queries
// --------------------

// GetByIDはdocument IDだけでShippingAddressを取得します。
//
// このメソッドは所有者を確認しません。
// Mallの/me配下の単件取得ではGetByUserを使用してください。
func (u *ShippingAddressUsecase) GetByID(
	ctx context.Context,
	id string,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validID, err := validateShippingAddressID(id)
	if err != nil {
		return nil, err
	}

	return u.repo.GetByID(ctx, validID)
}

// GetByUserはdocument IDと認証UIDの両方を条件として取得します。
//
// 対象が存在しない場合と、対象が指定ユーザーの所有物でない場合は、
// いずれもErrNotFoundを返します。
//
// Mallの/me配下の単件取得では、このメソッドを使用します。
func (u *ShippingAddressUsecase) GetByUser(
	ctx context.Context,
	id string,
	uid string,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validID, err := validateShippingAddressID(id)
	if err != nil {
		return nil, err
	}

	validUID, err := validateShippingAddressUID(uid)
	if err != nil {
		return nil, err
	}

	return u.repo.GetByUser(ctx, validID, validUID)
}

// Existsはdocument IDに対応するShippingAddressが存在するか返します。
//
// このメソッドは所有者を確認しないため、存在確認結果を
// 外部向けAPIへ直接公開してはいけません。
func (u *ShippingAddressUsecase) Exists(
	ctx context.Context,
	id string,
) (bool, error) {
	if err := u.ensureRepo(); err != nil {
		return false, err
	}

	validID, err := validateShippingAddressID(id)
	if err != nil {
		return false, err
	}

	return u.repo.Exists(ctx, validID)
}

// ListByUserIDは指定ユーザーが所有する配送先住所一覧を返します。
func (u *ShippingAddressUsecase) ListByUserID(
	ctx context.Context,
	uid string,
) ([]shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validUID, err := validateShippingAddressUID(uid)
	if err != nil {
		return nil, err
	}

	return u.repo.ListByUserID(ctx, validUID)
}

// --------------------
// Commands
// --------------------

// Createは新しい配送先住所を作成します。
//
// IDはUsecaseがUUIDを採番します。
// UserIDは認証UIDから設定します。
// CreatedAtおよびUpdatedAtはserver clockから設定します。
// Countryの既定値はDomain constructorが決定します。
func (u *ShippingAddressUsecase) Create(
	ctx context.Context,
	uid string,
	in CreateShippingAddressInput,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validUID, err := validateShippingAddressUID(uid)
	if err != nil {
		return nil, err
	}

	documentID := u.newDocID()
	if _, err := validateShippingAddressID(documentID); err != nil {
		return nil, err
	}

	now := u.now().UTC()

	entity, err := shipaddrdom.NewWithNow(
		documentID,
		validUID,
		in.ZipCode,
		in.State,
		in.City,
		in.Street,
		in.Street2,
		in.Country,
		now,
	)
	if err != nil {
		return nil, err
	}

	return u.repo.Create(ctx, entity)
}

// Updateは指定ユーザーが所有する配送先住所を部分更新します。
//
// Handlerは既存Entityを取得・mergeせず、UpdateShippingAddressInputを
// そのままこのメソッドへ渡します。
//
// Usecaseが所有者確認付き取得、入力merge、Domain更新、永続化を
// 一度だけ実行します。
//
// ID、UserIDおよびCreatedAtは既存値を保持します。
// UpdatedAtはserver clockで更新します。
func (u *ShippingAddressUsecase) Update(
	ctx context.Context,
	id string,
	uid string,
	in UpdateShippingAddressInput,
) (*shipaddrdom.ShippingAddress, error) {
	if err := u.ensureRepo(); err != nil {
		return nil, err
	}

	validID, err := validateShippingAddressID(id)
	if err != nil {
		return nil, err
	}

	validUID, err := validateShippingAddressUID(uid)
	if err != nil {
		return nil, err
	}

	current, err := u.repo.GetByUser(
		ctx,
		validID,
		validUID,
	)
	if err != nil {
		return nil, err
	}

	if current == nil {
		return nil, shipaddrdom.ErrNotFound
	}

	zipCode := current.ZipCode
	state := current.State
	city := current.City
	street := current.Street
	street2 := current.Street2
	country := current.Country

	if in.ZipCode != nil {
		zipCode = *in.ZipCode
	}

	if in.State != nil {
		state = *in.State
	}

	if in.City != nil {
		city = *in.City
	}

	if in.Street != nil {
		street = *in.Street
	}

	if in.Street2 != nil {
		street2 = *in.Street2
	}

	if in.Country != nil {
		country = *in.Country
	}

	now := u.now().UTC()

	if err := current.UpdateFromForm(
		zipCode,
		state,
		city,
		street,
		street2,
		country,
		now,
	); err != nil {
		return nil, err
	}

	return u.repo.Update(ctx, *current)
}

// Deleteはdocument IDだけで配送先住所を削除します。
//
// このメソッドは所有者を確認しません。
// Mallの/me配下ではDeleteByUserを使用してください。
func (u *ShippingAddressUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}

	validID, err := validateShippingAddressID(id)
	if err != nil {
		return err
	}

	return u.repo.Delete(ctx, validID)
}

// DeleteByUserは所有者を確認して配送先住所を削除します。
//
// 対象が存在しない場合と、対象が指定ユーザーの所有物でない場合は、
// いずれもErrNotFoundを返します。
func (u *ShippingAddressUsecase) DeleteByUser(
	ctx context.Context,
	id string,
	uid string,
) error {
	if err := u.ensureRepo(); err != nil {
		return err
	}

	validID, err := validateShippingAddressID(id)
	if err != nil {
		return err
	}

	validUID, err := validateShippingAddressUID(uid)
	if err != nil {
		return err
	}

	current, err := u.repo.GetByUser(
		ctx,
		validID,
		validUID,
	)
	if err != nil {
		return err
	}

	if current == nil {
		return shipaddrdom.ErrNotFound
	}

	return u.repo.Delete(ctx, validID)
}
