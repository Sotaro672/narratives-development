// backend/internal/domain/productBlueprint/repository_port.go
package productBlueprint

import (
	"context"
	"time"
)

// ========================================
// Create/Update inputs
// ========================================

type CreateInput struct {
	// Create時にUsecaseで生成して渡す。
	ID string `json:"id"`

	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandID   string `json:"brandId"`
	CompanyID string `json:"companyId"`

	// productBlueprintCategoriesの正データから
	// Usecaseで生成して渡すdenormalized snapshot。
	ProductBlueprintCategory ProductBlueprintCategorySnapshot `json:"productBlueprintCategory"`

	// CategoryFieldsはカテゴリ別のProductBlueprint入力値を保持する。
	//
	// 例:
	//   - alcohol.sake:
	//     vintage、region、material、alcoholContent
	//   - apparel.tops:
	//     weight、fit、material
	//   - cosmetics.skincare:
	//     material、volume
	//
	// Alcoholの容量はCategoryFieldsへ保存せず、
	// Model variationのVolumeだけを正とする。
	//
	// color、size、measurementsなどのModel variation固有値も
	// CategoryFieldsへ保存しない。
	//
	// brandId、productName、productIdTagType、descriptionなどの
	// ProductBlueprint共通fieldもここには入れない。
	CategoryFields CategoryFields `json:"categoryFields,omitempty"`

	ProductIdTag ProductIDTag `json:"productIdTag"`
	AssigneeID   string       `json:"assigneeId"`

	// ModelRefsはmodelIdとdisplayOrderを保持する。
	//
	// create時点では空でもよい。
	// ModelRefsはModelUsecase側でmodels collectionを正として同期する。
	// 永続化層ではmodelRefsとして保存する。
	ModelRefs []ModelRef `json:"modelRefs,omitempty"`

	CreatedBy *string    `json:"createdBy,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

type Patch struct {
	ProductName *string `json:"productName,omitempty"`
	Description *string `json:"description,omitempty"`

	// ProductBlueprintに保存するBrand ID。
	BrandID *string `json:"brandId,omitempty"`

	// BrandNameはread-modelで使用する表示用項目。
	// ProductBlueprint Repositoryでは永続化しない。
	BrandName *string `json:"brandName,omitempty"`

	// CompanyIDはProductBlueprintに保存するCompany ID。
	//
	// マルチテナント境界に関わるため、通常の更新処理では
	// 原則として変更しない。変更を許可する場合はUsecase側で
	// 認可と整合性を確認してから設定する。
	CompanyID *string `json:"companyId,omitempty"`

	// CompanyNameはread-modelで使用する表示用項目。
	// ProductBlueprint Repositoryでは永続化しない。
	CompanyName *string `json:"companyName,omitempty"`

	// productBlueprintCategoriesの正データから
	// Usecaseで生成して渡すdenormalized snapshot。
	ProductBlueprintCategory *ProductBlueprintCategorySnapshot `json:"productBlueprintCategory,omitempty"`

	// CategoryFieldsはカテゴリ別のProductBlueprint入力値を保持する。
	//
	// 例:
	//   - alcohol.sake:
	//     vintage、region、material、alcoholContent
	//   - apparel.tops:
	//     weight、fit、material
	//   - cosmetics.skincare:
	//     material、volume
	//
	// Alcoholの容量はCategoryFieldsへ保存せず、
	// Model variationのVolumeだけを正とする。
	//
	// nilの場合は更新しない。
	// 空mapの場合はCategoryFieldsを空に更新する。
	CategoryFields *CategoryFields `json:"categoryFields,omitempty"`

	ProductIdTag *ProductIDTag `json:"productIdTag,omitempty"`
	AssigneeID   *string       `json:"assigneeId,omitempty"`

	// ModelRefsはmodelIdとdisplayOrderを保持する。
	//
	// 通常のProductBlueprint更新APIでは原則変更しない。
	// ModelRefsの同期はModelUsecaseと
	// ReplaceModelRefsWithoutTouchを正とする。
	//
	// 既存のread/write互換のためPatchに残すが、
	// command Usecase側では基本的に設定しない。
	ModelRefs *[]ModelRef `json:"modelRefs,omitempty"`

	// UpdatedByは認証ContextからUsecaseが設定する内部項目。
	//
	// HTTP request bodyから任意の更新者IDを受け取らないため、
	// JSONのencode/decode対象にはしない。
	UpdatedBy *string `json:"-"`
}

// ========================================
// Query contracts
// ========================================

type Filter struct {
	// CompanyIDはマルチテナント境界として必須。
	CompanyID string

	SearchTerm  string
	BrandIDs    []string
	AssigneeIDs []string

	// カテゴリ検索ではProductBlueprintに保存した
	// denormalized fieldを検索対象にする。
	ProductBlueprintCategoryIDs []string

	ProductBlueprintCategoryCodes []string

	ProductBlueprintCategoryKinds []string

	TagTypes []ProductIDTagType
}

type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items []ProductBlueprint

	TotalCount int
	TotalPages int

	Page    int
	PerPage int
}

// ========================================
// Repository Port
// ========================================

type Repository interface {
	// Read

	// GetByIDは通常利用可能なactive状態のProductBlueprintだけを返す。
	//
	// 論理削除済みProductBlueprintはErrNotFoundとして扱う。
	// 復旧処理ではGetByIDIncludingDeletedを使用する。
	GetByID(
		ctx context.Context,
		id string,
	) (ProductBlueprint, error)

	// GetByIDIncludingDeletedはactiveとdeletedの両方を取得する。
	//
	// 論理削除、復旧、物理削除判定などのライフサイクル処理専用とし、
	// 通常の一覧・詳細・Catalog処理では使用しない。
	GetByIDIncludingDeleted(
		ctx context.Context,
		id string,
	) (ProductBlueprint, error)

	// ListByCompanyIDはcompanyId単位でactive状態の
	// ProductBlueprint一覧を取得する正規Port。
	//
	// 論理削除済みProductBlueprintは返さない。
	// ID一覧が必要な場合も、戻り値から呼出側でIDを抽出する。
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]ProductBlueprint, error)

	// ListDeletedByCompanyIDはcompanyId単位で論理削除済みの
	// ProductBlueprint一覧を取得する。
	//
	// 復旧可能期間を過ぎ、物理削除バッチの実行を待っている
	// ProductBlueprintも含めて返してよい。
	//
	// 復旧可能かどうかはProductBlueprint.CanRestoreで判定する。
	ListDeletedByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]ProductBlueprint, error)

	// ListIDsByBrandIDはbrandIdに紐づくactive状態の
	// ProductBlueprint ID一覧を取得する。
	//
	// 論理削除済みProductBlueprintは返さない。
	ListIDsByBrandID(
		ctx context.Context,
		brandID string,
	) ([]string, error)

	// GetIDByModelIDはmodelIdから、そのModelが属する
	// active状態のProductBlueprint IDとModelRefsを取得する。
	//
	// ProductBlueprintまたはModelが論理削除済みの場合は返さない。
	//
	// 戻り値:
	//   - productBlueprintID:
	//     Modelが紐づくProductBlueprint ID
	//   - modelRefs:
	//     対象ProductBlueprintのModelRefs
	//
	// ProductBlueprint IDだけが必要なcallerは第1戻り値を使う。
	// DisplayOrderが必要なcallerは第2戻り値から対象Modelを探す。
	GetIDByModelID(
		ctx context.Context,
		modelID string,
	) (
		string,
		[]ModelRef,
		error,
	)

	// Write

	Create(
		ctx context.Context,
		in CreateInput,
	) (ProductBlueprint, error)

	// Updateはactiveかつprinted=falseのProductBlueprintだけを更新する。
	//
	// 論理削除済みまたは印刷済みの場合は更新を拒否する。
	Update(
		ctx context.Context,
		id string,
		patch Patch,
	) (ProductBlueprint, error)

	// SoftDeleteはProductBlueprintと配下Modelを論理削除する。
	//
	// Repository実装では、同一Transactionまたは同一WriteBatch内で
	// 次の条件を再確認する。
	//   - ProductBlueprint.companyId == companyID
	//   - ProductBlueprint.printed == false
	//   - ProductBlueprint.status == active
	//   - 配下Modelが対象ProductBlueprintに属している
	//
	// ProductBlueprintと配下Modelには同一のdeletedAtとpurgeAtを設定する。
	//
	// 同じProductBlueprintがすでに論理削除済みの場合は、
	// 最初のdeletedAtとpurgeAtを維持して冪等に成功してよい。
	SoftDelete(
		ctx context.Context,
		id string,
		companyID string,
		deletedBy *string,
		deletedAt time.Time,
	) (ProductBlueprint, error)

	// RestoreはProductBlueprintと配下Modelを同時に復旧する。
	//
	// Repository実装では、同一Transactionまたは同一WriteBatch内で
	// 次の条件を再確認する。
	//   - ProductBlueprint.companyId == companyID
	//   - ProductBlueprint.printed == false
	//   - ProductBlueprint.status == deleted
	//   - restoredAt < ProductBlueprint.purgeAt
	//   - 配下Modelが物理削除されず残っている
	//
	// ProductBlueprintまたは配下Modelの一部だけを復旧してはならない。
	Restore(
		ctx context.Context,
		id string,
		companyID string,
		restoredBy *string,
		restoredAt time.Time,
	) (ProductBlueprint, error)

	// ReplaceModelRefsWithoutTouchはProductBlueprintの
	// ModelRefsを置換する。
	//
	// ModelUsecaseがmodels collectionを正として同期するために使う。
	//
	// Repository実装側では次を保証する。
	//   - ProductBlueprint.status == active
	//   - ProductBlueprint.printed == false
	//   - DisplayOrder順に正規化する
	//   - 空IDと重複IDを除外する
	//   - DisplayOrderを1..Nへ再採番する
	//   - refsが空の場合は空配列へ置換する
	//   - UpdatedAtとUpdatedByは更新しない
	ReplaceModelRefsWithoutTouch(
		ctx context.Context,
		id string,
		refs []ModelRef,
	) (ProductBlueprint, error)

	// MarkPrintedはPrintedをfalseからtrueへ遷移させる。
	//
	// Repository実装ではTransaction内で次を確認する。
	//   - ProductBlueprint.status == active
	//   - ProductBlueprint.printed == false
	//
	// 論理削除済みの場合は印刷済みへ変更できない。
	// すでにPrinted=trueの場合は成功として扱う冪等な操作とする。
	MarkPrinted(
		ctx context.Context,
		id string,
	) (ProductBlueprint, error)
}

// ========================================
// Purge Repository Port
// ========================================

// PurgeRepositoryは復旧期限を経過したProductBlueprintと
// 配下Modelを物理削除するバッチ専用Portです。
//
// 通常のHTTP HandlerやProductBlueprintUsecaseからは参照せず、
// ProductBlueprintPurgeUsecaseだけが利用します。
type PurgeRepository interface {
	// ListPurgeCandidatesは物理削除対象を取得する。
	//
	// 対象条件:
	//   - status == deleted
	//   - purgeAt <= now
	//
	// limitは1回のバッチで処理する最大件数です。
	ListPurgeCandidates(
		ctx context.Context,
		now time.Time,
		limit int,
	) ([]ProductBlueprint, error)

	// PurgeWithModelsはProductBlueprintと配下Modelを物理削除する。
	//
	// 物理削除直前に、Transaction内で次を再確認する。
	//   - ProductBlueprint.status == deleted
	//   - ProductBlueprint.printed == false
	//   - ProductBlueprint.purgeAt <= now
	//   - ProductBlueprint.purgeAt == expectedPurgeAt
	//
	// expectedPurgeAtとの一致確認により、候補取得後に復旧された
	// ProductBlueprintを誤って物理削除することを防ぐ。
	//
	// 削除順は配下Modelおよび関連Documentを先にし、
	// ProductBlueprint本体を最後にする。
	PurgeWithModels(
		ctx context.Context,
		productBlueprintID string,
		expectedPurgeAt time.Time,
		now time.Time,
	) error
}
