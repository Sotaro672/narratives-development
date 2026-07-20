// backend/internal/adapters/out/firestore/shippingAddress_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	shipaddrdom "narratives/internal/domain/shippingAddress"
)

const shippingAddressCollection = "shippingAddresses"

// RepositoryPortの実装漏れをcompile時に検出します。
var _ shipaddrdom.RepositoryPort = (*ShippingAddressRepositoryFS)(nil)

// ShippingAddressRepositoryFSはFirestoreを使用した
// ShippingAddress Repository実装です.
type ShippingAddressRepositoryFS struct {
	Client *firestore.Client
}

// shippingAddressDocumentはFirestore documentの保存schemaです.
//
// document IDはこの構造体には保存しません。
// document IDはShippingAddress.IDとしてDocumentSnapshotから復元します。
type shippingAddressDocument struct {
	UserID  string `firestore:"userId"`
	ZipCode string `firestore:"zipCode"`
	State   string `firestore:"state"`
	City    string `firestore:"city"`
	Street  string `firestore:"street"`
	Street2 string `firestore:"street2"`
	Country string `firestore:"country"`

	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}

func NewShippingAddressRepositoryFS(
	client *firestore.Client,
) *ShippingAddressRepositoryFS {
	return &ShippingAddressRepositoryFS{
		Client: client,
	}
}

func (r *ShippingAddressRepositoryFS) ensureClient() error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}

	return nil
}

func (r *ShippingAddressRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection(shippingAddressCollection)
}

func validateShippingAddressRepositoryID(
	id string,
) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", shipaddrdom.ErrInvalidID
	}

	if _, err := uuid.Parse(id); err != nil {
		return "", shipaddrdom.ErrInvalidID
	}

	return id, nil
}

func validateShippingAddressRepositoryUserID(
	userID string,
) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", shipaddrdom.ErrInvalidUserID
	}

	if len([]rune(userID)) > shipaddrdom.MaxUserIDLength {
		return "", shipaddrdom.ErrInvalidUserID
	}

	return userID, nil
}

func shippingAddressNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// --------------------
// Read
// --------------------

// GetByIDはdocument IDだけでShippingAddressを取得します。
//
// このメソッドは所有者を確認しません。
// Mallの/me配下ではGetByUserを使用します。
func (r *ShippingAddressRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (*shipaddrdom.ShippingAddress, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err := validateShippingAddressRepositoryID(id)
	if err != nil {
		return nil, err
	}

	snapshot, err := r.col().Doc(validID).Get(ctx)
	if shippingAddressNotFound(err) {
		return nil, shipaddrdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	entity, err := docToShippingAddress(snapshot)
	if err != nil {
		return nil, err
	}

	return &entity, nil
}

// GetByUserはdocument IDと所有者UIDの両方を確認して取得します。
//
// 対象が存在しない場合と、対象が指定ユーザーの所有物でない場合は、
// いずれもErrNotFoundを返します。
func (r *ShippingAddressRepositoryFS) GetByUser(
	ctx context.Context,
	id string,
	userID string,
) (*shipaddrdom.ShippingAddress, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err := validateShippingAddressRepositoryID(id)
	if err != nil {
		return nil, err
	}

	validUserID, err :=
		validateShippingAddressRepositoryUserID(userID)
	if err != nil {
		return nil, err
	}

	snapshot, err := r.col().Doc(validID).Get(ctx)
	if shippingAddressNotFound(err) {
		return nil, shipaddrdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	entity, err := docToShippingAddress(snapshot)
	if err != nil {
		return nil, err
	}

	if entity.UserID != validUserID {
		return nil, shipaddrdom.ErrNotFound
	}

	return &entity, nil
}

// Existsはdocument IDに対応するdocumentが存在するか返します。
//
// 空IDまたは不正なUUIDはfalseとErrInvalidIDを返します。
func (r *ShippingAddressRepositoryFS) Exists(
	ctx context.Context,
	id string,
) (bool, error) {
	if err := r.ensureClient(); err != nil {
		return false, err
	}

	validID, err := validateShippingAddressRepositoryID(id)
	if err != nil {
		return false, err
	}

	_, err = r.col().Doc(validID).Get(ctx)
	if shippingAddressNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// ListByUserIDは指定ユーザーが所有する配送先住所を、
// updatedAtの降順で返します。
//
// 対象が0件の場合は空sliceを返します。
func (r *ShippingAddressRepositoryFS) ListByUserID(
	ctx context.Context,
	userID string,
) ([]shipaddrdom.ShippingAddress, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validUserID, err :=
		validateShippingAddressRepositoryUserID(userID)
	if err != nil {
		return nil, err
	}

	query := r.col().
		Where("userId", "==", validUserID).
		OrderBy("updatedAt", firestore.Desc)

	snapshots, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	result := make(
		[]shipaddrdom.ShippingAddress,
		0,
		len(snapshots),
	)

	for _, snapshot := range snapshots {
		entity, err := docToShippingAddress(snapshot)
		if err != nil {
			return nil, err
		}

		// Query条件と復元結果が一致することを確認します。
		if entity.UserID != validUserID {
			return nil, shipaddrdom.ErrInvalidUserID
		}

		result = append(result, entity)
	}

	return result, nil
}

// --------------------
// Write
// --------------------

// Createは新しいshippingAddresses/{id}を作成します。
//
// 保存前にDomain constructorを使用してEntityを再検証します。
// 同一IDが存在する場合はErrConflictを返し、上書きしません。
func (r *ShippingAddressRepositoryFS) Create(
	ctx context.Context,
	value shipaddrdom.ShippingAddress,
) (*shipaddrdom.ShippingAddress, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err :=
		validateShippingAddressRepositoryID(value.ID)
	if err != nil {
		return nil, err
	}

	// Repositoryへ直接渡されたEntityもDomain規則で再検証します。
	validated, err := shipaddrdom.New(
		validID,
		value.UserID,
		value.ZipCode,
		value.State,
		value.City,
		value.Street,
		value.Street2,
		value.Country,
		value.CreatedAt,
		value.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(validated.ID)

	_, err = ref.Create(
		ctx,
		shippingAddressToDocData(validated),
	)
	if status.Code(err) == codes.AlreadyExists {
		return nil, shipaddrdom.ErrConflict
	}
	if err != nil {
		return nil, err
	}

	return &validated, nil
}

// Updateは既存のshippingAddresses/{id}を更新します。
//
// Firestore transaction内で次の処理を行います。
//
//  1. 対象documentを取得する
//  2. 既存のUserIDとCreatedAtを保持する
//  3. 更新後EntityをDomain constructorで検証する
//  4. transaction.Updateで変更可能fieldだけを更新する
//
// Setは使用しないため、対象が途中で削除された場合でも
// documentを再作成しません。
func (r *ShippingAddressRepositoryFS) Update(
	ctx context.Context,
	value shipaddrdom.ShippingAddress,
) (*shipaddrdom.ShippingAddress, error) {
	if err := r.ensureClient(); err != nil {
		return nil, err
	}

	validID, err :=
		validateShippingAddressRepositoryID(value.ID)
	if err != nil {
		return nil, err
	}

	ref := r.col().Doc(validID)

	var updated shipaddrdom.ShippingAddress

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			snapshot, err := tx.Get(ref)
			if shippingAddressNotFound(err) {
				return shipaddrdom.ErrNotFound
			}
			if err != nil {
				return err
			}

			current, err := docToShippingAddress(snapshot)
			if err != nil {
				return err
			}

			// UserIDは更新不可です。
			if strings.TrimSpace(value.UserID) != current.UserID {
				return shipaddrdom.ErrInvalidUserID
			}

			// CreatedAtは更新不可です。
			if !value.CreatedAt.Equal(current.CreatedAt) {
				return shipaddrdom.ErrInvalidCreatedAt
			}

			// ID、UserIDおよびCreatedAtは既存値を使用します。
			next, err := shipaddrdom.New(
				current.ID,
				current.UserID,
				value.ZipCode,
				value.State,
				value.City,
				value.Street,
				value.Street2,
				value.Country,
				current.CreatedAt,
				value.UpdatedAt,
			)
			if err != nil {
				return err
			}

			updates := []firestore.Update{
				{
					Path:  "zipCode",
					Value: next.ZipCode,
				},
				{
					Path:  "state",
					Value: next.State,
				},
				{
					Path:  "city",
					Value: next.City,
				},
				{
					Path:  "street",
					Value: next.Street,
				},
				{
					Path:  "street2",
					Value: next.Street2,
				},
				{
					Path:  "country",
					Value: next.Country,
				},
				{
					Path:  "updatedAt",
					Value: next.UpdatedAt,
				},
			}

			if err := tx.Update(ref, updates); err != nil {
				if shippingAddressNotFound(err) {
					return shipaddrdom.ErrNotFound
				}

				return err
			}

			updated = next

			return nil
		},
	)
	if shippingAddressNotFound(err) {
		return nil, shipaddrdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// Deleteはdocument IDでShippingAddressを削除します。
//
// transaction内で存在確認と削除を行い、対象が存在しない場合は
// ErrNotFoundを返します。
func (r *ShippingAddressRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if err := r.ensureClient(); err != nil {
		return err
	}

	validID, err := validateShippingAddressRepositoryID(id)
	if err != nil {
		return err
	}

	ref := r.col().Doc(validID)

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			_, err := tx.Get(ref)
			if shippingAddressNotFound(err) {
				return shipaddrdom.ErrNotFound
			}
			if err != nil {
				return err
			}

			if err := tx.Delete(ref); err != nil {
				if shippingAddressNotFound(err) {
					return shipaddrdom.ErrNotFound
				}

				return err
			}

			return nil
		},
	)
	if shippingAddressNotFound(err) {
		return shipaddrdom.ErrNotFound
	}

	return err
}

// --------------------
// Mapping
// --------------------

// docToShippingAddressはFirestore documentをDomain Entityへ変換します.
//
// DataToでFirestore fieldの型を検証した後、Domain constructorを使用して
// Entity全体の不変条件を検証します。
func docToShippingAddress(
	document *firestore.DocumentSnapshot,
) (shipaddrdom.ShippingAddress, error) {
	if document == nil || document.Ref == nil {
		return shipaddrdom.ShippingAddress{},
			shipaddrdom.ErrNotFound
	}

	var data shippingAddressDocument
	if err := document.DataTo(&data); err != nil {
		return shipaddrdom.ShippingAddress{}, err
	}

	entity, err := shipaddrdom.New(
		document.Ref.ID,
		data.UserID,
		data.ZipCode,
		data.State,
		data.City,
		data.Street,
		data.Street2,
		data.Country,
		data.CreatedAt,
		data.UpdatedAt,
	)
	if err != nil {
		return shipaddrdom.ShippingAddress{}, err
	}

	return entity, nil
}

// shippingAddressToDocDataはDomain EntityをFirestore保存schemaへ変換します.
//
// 呼び出し前にDomain constructorによる検証が完了していることを前提とします。
// IDはdocument IDとして使用するためfieldには保存しません。
func shippingAddressToDocData(
	value shipaddrdom.ShippingAddress,
) shippingAddressDocument {
	return shippingAddressDocument{
		UserID:    value.UserID,
		ZipCode:   value.ZipCode,
		State:     value.State,
		City:      value.City,
		Street:    value.Street,
		Street2:   value.Street2,
		Country:   value.Country,
		CreatedAt: value.CreatedAt.UTC(),
		UpdatedAt: value.UpdatedAt.UTC(),
	}
}
