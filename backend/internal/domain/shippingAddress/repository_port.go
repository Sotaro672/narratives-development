// backend/internal/domain/shippingAddress/repository_port.go
package shippingAddress

import (
	"context"
	"errors"
)

// RepositoryPortはShippingAddressの永続化契約です。
//
// ShippingAddress Entityをデータ構造と不変条件の正本とします。
//
// Identity:
//
//   - document ID = ShippingAddress.ID
//   - ShippingAddress.IDはUUID
//   - ShippingAddress.UserIDは所有者の認証UID
//   - ShippingAddress.IDとShippingAddress.UserIDは異なる値
//   - 1ユーザーは複数の配送先住所を所有可能
//
// Ownership:
//
// Mallの/me配下など、ユーザー本人のデータを操作する処理では、
// GetByIDではなくGetByUserを使用します。
//
// Repository実装は、対象が存在しない場合と、対象が指定ユーザーの
// 所有物でない場合の両方でErrNotFoundを返します。
// これにより、他ユーザーのデータの存在を外部へ公開しません。
type RepositoryPort interface {
	// GetByIDはdocument IDだけでShippingAddressを取得します。
	//
	// このメソッドは所有者を検証しません。
	// 管理処理や、所有者検証が不要であることが明確な内部処理に限定して
	// 使用してください。
	//
	// idが空、またはUUIDとして不正な場合はErrInvalidIDを返します。
	// 対象が存在しない場合はErrNotFoundを返します。
	GetByID(
		ctx context.Context,
		id string,
	) (*ShippingAddress, error)

	// GetByUserはdocument IDと所有者UIDの両方を条件として取得します。
	//
	// 次のいずれかに該当する場合はErrNotFoundを返します。
	//
	//   - 対象documentが存在しない
	//   - 対象documentのUserIDがuserIDと一致しない
	//
	// idが空、またはUUIDとして不正な場合はErrInvalidIDを返します。
	// userIDが空の場合はErrInvalidUserIDを返します。
	//
	// Mallの/me配下の単件取得、更新、削除では、このメソッドを使用します。
	GetByUser(
		ctx context.Context,
		id string,
		userID string,
	) (*ShippingAddress, error)

	// Existsはdocument IDに対応するShippingAddressが存在するか返します。
	//
	// idが空、またはUUIDとして不正な場合は、falseとErrInvalidIDを返します。
	//
	// このメソッドは所有者を検証しないため、外部向けAPIで存在確認結果を
	// そのまま公開してはいけません。
	Exists(
		ctx context.Context,
		id string,
	) (bool, error)

	// ListByUserIDは指定ユーザーが所有するShippingAddress一覧を返します。
	//
	// userIDが空の場合はErrInvalidUserIDを返します。
	// 対象が0件の場合は、ErrNotFoundではなく空のsliceを返します。
	//
	// 並び順はupdatedAtの降順とします。
	ListByUserID(
		ctx context.Context,
		userID string,
	) ([]ShippingAddress, error)

	// Createは新しいShippingAddress documentを作成します。
	//
	// 呼び出し時点で、ShippingAddressはDomain規則を満たし、
	// IDがUUIDとして採番済みでなければなりません。
	//
	// Repository実装もDomain constructorを使用してEntityを再検証し、
	// 不正なEntityを永続化してはいけません。
	//
	// 同じIDのdocumentが既に存在する場合はErrConflictを返します。
	// Createは既存documentを上書きしてはいけません。
	Create(
		ctx context.Context,
		a ShippingAddress,
	) (*ShippingAddress, error)

	// Updateは既存のShippingAddressを更新します。
	//
	// Updateはupsertではありません。
	// 対象が存在しない場合はErrNotFoundを返します。
	//
	// Repository実装はFirestore transaction、document update、
	// update-time preconditionなどを使用し、存在確認後に対象が削除された場合でも
	// documentを再作成してはいけません。
	//
	// 次の値は変更してはいけません。
	//
	//   - ID
	//   - UserID
	//   - CreatedAt
	//
	// Repository実装は永続化前にDomain constructorを使用してEntityを検証します。
	Update(
		ctx context.Context,
		a ShippingAddress,
	) (*ShippingAddress, error)

	// Deleteはdocument IDでShippingAddressを削除します。
	//
	// このメソッドは所有者を検証しません。
	// Mallの/me配下では、事前にGetByUserを使用して所有者を確認します。
	//
	// 対象が存在しない場合はErrNotFoundを返します。
	// idが空、またはUUIDとして不正な場合はErrInvalidIDを返します。
	Delete(
		ctx context.Context,
		id string,
	) error
}

// Repository共通エラーです。
//
// Adapter層は必要に応じてこれらをwrapできます。
// 呼び出し側は文字列比較や直接比較ではなく、errors.Isを使用して判定します。
var (
	ErrNotFound = errors.New("shippingAddress: not found")
	ErrConflict = errors.New("shippingAddress: conflict")
)
