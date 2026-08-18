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
//   - ShippingAddress.UserIDは登録・所有する認証ユーザーのUID
//   - ShippingAddress.CompanyIDは所属するcompanyのdocument ID
//   - ShippingAddress.ID、UserID、CompanyIDはそれぞれ異なる識別子
//   - 1ユーザーは複数のShippingAddressを所有可能
//   - 1companyは複数のShippingAddressを所有可能
//
// Ownership:
//
// Mallの/me配下など、ユーザー本人のデータを操作する処理ではGetByUserを使用します。
// Consoleなどcompany単位でデータを操作する処理ではGetByCompanyを使用します。
//
// Repository実装は、対象が存在しない場合と、指定されたuserまたはcompanyに所属しない場合の両方でErrNotFoundを返します。
// これにより、他ユーザー・他companyのShippingAddressの存在を外部へ公開しません。
type RepositoryPort interface {
	// GetByIDはdocument IDだけでShippingAddressを取得します。
	//
	// このメソッドはUserID、CompanyIDによる所有権を検証しません。
	// 所有権検証が不要であることが明確な内部処理に限定して使用してください。
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

	// GetByCompanyはdocument IDとCompanyIDの両方を条件として取得します。
	//
	// 次のいずれかに該当する場合はErrNotFoundを返します。
	//
	//   - 対象documentが存在しない
	//   - 対象documentのCompanyIDがcompanyIDと一致しない
	//
	// idが空、またはUUIDとして不正な場合はErrInvalidIDを返します。
	// companyIDが空の場合はErrInvalidCompanyIDを返します。
	//
	// Consoleなどcompany単位でShippingAddressを取得、更新、削除する場合は、このメソッドを使用します。
	GetByCompany(
		ctx context.Context,
		id string,
		companyID string,
	) (*ShippingAddress, error)

	// Existsはdocument IDに対応するShippingAddressが存在するか返します。
	//
	// idが空、またはUUIDとして不正な場合はfalseとErrInvalidIDを返します。
	//
	// このメソッドはUserID、CompanyIDによる所有権を検証しないため、外部向けAPIで存在確認結果をそのまま公開してはいけません。
	Exists(
		ctx context.Context,
		id string,
	) (bool, error)

	// ListByUserIDは指定ユーザーが所有するShippingAddress一覧を返します。
	//
	// userIDが空の場合はErrInvalidUserIDを返します。
	// 対象が0件の場合はErrNotFoundではなく空のsliceを返します。
	//
	// 並び順はupdatedAtの降順とします。
	ListByUserID(
		ctx context.Context,
		userID string,
	) ([]ShippingAddress, error)

	// ListByCompanyIDは指定companyに所属するShippingAddress一覧を返します。
	//
	// companyIDが空の場合はErrInvalidCompanyIDを返します。
	// 対象が0件の場合はErrNotFoundではなく空のsliceを返します。
	//
	// 並び順はupdatedAtの降順とします。
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]ShippingAddress, error)

	// Createは新しいShippingAddress documentを作成します。
	//
	// 呼び出し時点でShippingAddressはDomain規則を満たし、IDがUUIDとして採番済みでなければなりません。
	// UserIDおよびCompanyIDも必須です。
	//
	// Repository実装もDomain constructorを使用してEntityを再検証し、不正なEntityを永続化してはいけません。
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
	// Repository実装はFirestore transaction、document update、update-time preconditionなどを使用し、
	// 存在確認後に対象が削除された場合でもdocumentを再作成してはいけません。
	//
	// 次の値は変更してはいけません。
	//
	//   - ID
	//   - UserID
	//   - CompanyID
	//   - CreatedAt
	//
	// Repository実装は永続化前にDomain constructorを使用してEntityを検証します。
	Update(
		ctx context.Context,
		a ShippingAddress,
	) (*ShippingAddress, error)

	// Deleteはdocument IDでShippingAddressを削除します。
	//
	// このメソッドはUserID、CompanyIDによる所有権を検証しません。
	// Mallの/me配下では事前にGetByUserを使用して所有者を確認します。
	// Consoleでは事前にGetByCompanyを使用してcompany所属を確認します。
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
