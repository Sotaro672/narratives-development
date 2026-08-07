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

	// GetByIDはProductBlueprintをIDで取得する。
	//
	// 対象Documentが存在しない場合はErrNotFoundを返す。
	GetByID(
		ctx context.Context,
		id string,
	) (ProductBlueprint, error)

	// ListByCompanyIDはcompanyId単位で
	// ProductBlueprint一覧を取得する正規Port。
	//
	// ID一覧が必要な場合も、戻り値から呼出側でIDを抽出する。
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]ProductBlueprint, error)

	// ListIDsByBrandIDはbrandIdに紐づく
	// ProductBlueprint ID一覧を取得する。
	ListIDsByBrandID(
		ctx context.Context,
		brandID string,
	) ([]string, error)

	// GetIDByModelIDはmodelIdから、そのModelが属する
	// ProductBlueprint IDとModelRefsを取得する。
	//
	// 対象ModelまたはProductBlueprintが存在しない場合は
	// ErrNotFoundを返す。
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

	// Updateはprinted=falseのProductBlueprintだけを更新する。
	//
	// 印刷済みの場合は更新を拒否する。
	Update(
		ctx context.Context,
		id string,
		patch Patch,
	) (ProductBlueprint, error)

	// DeleteはProductBlueprintと配下Modelを物理削除する。
	//
	// Repository実装では、同一Transaction内で
	// 次の条件を再確認する。
	//   - ProductBlueprint.companyId == companyID
	//   - ProductBlueprint.printed == false
	//
	// 配下Modelはmodels collectionの
	// productBlueprintId == idを正として取得し、
	// ProductBlueprintに紐づくModelをすべて物理削除する。
	//
	// 削除順は配下Modelを先にし、
	// ProductBlueprint本体を最後にする。
	//
	// printed=trueのProductBlueprintは物理削除してはならない。
	Delete(
		ctx context.Context,
		id string,
		companyID string,
	) error

	// ReplaceModelRefsWithoutTouchはProductBlueprintの
	// ModelRefsを置換する。
	//
	// ModelUsecaseがmodels collectionを正として同期するために使う。
	//
	// Repository実装側では次を保証する。
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
	// Repository実装ではTransaction内で
	// ProductBlueprint.printedを確認する。
	//
	// すでにPrinted=trueの場合は成功として扱う冪等な操作とする。
	MarkPrinted(
		ctx context.Context,
		id string,
	) (ProductBlueprint, error)
}
