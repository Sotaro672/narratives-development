// backend/internal/application/usecase/productBlueprint_usecase.go
package usecase

import (
	"context"
	"time"

	productbpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

// ProductBlueprintUsecase is the command application service for productBlueprint.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo

	// ProductBlueprint起票時にproductBlueprintReview側も
	// 初期化するためのPort。
	//
	// NewProductBlueprintUsecaseを唯一の初期化経路とし、
	// 外部からWith形式では差し込まない。
	reviewInit ProductBlueprintReviewInitializer
}

// NewProductBlueprintUsecaseはProductBlueprintUsecaseの
// 唯一の初期化経路です。
//
// reviewInitがnilの場合は、既存互換のため
// ProductBlueprintReviewの初期化を省略します。
func NewProductBlueprintUsecase(
	repo ProductBlueprintRepo,
	reviewInit ProductBlueprintReviewInitializer,
) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo:       repo,
		reviewInit: reviewInit,
	}
}

// ------------------------------------------------------------
// Ports
// ------------------------------------------------------------

// ProductBlueprintRepoはProductBlueprintUsecaseが必要とする
// command側の最小永続化Portです。
//
// 一覧・詳細などの画面構築用read modelは
// application/query/console側へ分離します。
//
// ModelRefsの同期はModelUsecase側でmodels collectionを正として
// 実行します。
type ProductBlueprintRepo interface {
	// Read for command-side existence and company-boundary checks.
	GetByID(
		ctx context.Context,
		id string,
	) (productbpdom.ProductBlueprint, error)

	// MarkPrintedはprintedをtrueへ更新します。
	MarkPrinted(
		ctx context.Context,
		id string,
	) (productbpdom.ProductBlueprint, error)

	// Write
	Create(
		ctx context.Context,
		in productbpdom.CreateInput,
	) (productbpdom.ProductBlueprint, error)

	Update(
		ctx context.Context,
		id string,
		patch productbpdom.Patch,
	) (productbpdom.ProductBlueprint, error)

	// Delete physically removes a ProductBlueprint.
	Delete(
		ctx context.Context,
		id string,
	) error
}

// ProductBlueprintReviewInitializerはProductBlueprint起票時に、
// Review側の商品単位初期化documentを作成するためのPortです。
type ProductBlueprintReviewInitializer interface {
	InitForProductBlueprint(
		ctx context.Context,
		productBlueprintID string,
		companyID string,
		createdAt time.Time,
		createdBy *string,
	) error
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

// Create creates a ProductBlueprint.
//
// Usecaseの公開APIはProductBlueprintを受け取り、
// RepositoryへはCreateInputを渡します。
func (
	u *ProductBlueprintUsecase,
) Create(
	ctx context.Context,
	value productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	if u == nil || u.repo == nil {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInternal
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInvalidCompanyID
	}

	// NewIDはcrypto/randの生成に失敗した場合に
	// errorを返すため、必ず処理する。
	id, err := productbpdom.NewID()
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	now := time.Now().UTC()
	createdAt := now

	input := productbpdom.CreateInput{
		ID: id,

		ProductName: value.ProductName,
		Description: value.Description,

		BrandID: value.BrandID,

		// productBlueprintCategoriesの正データから生成済みの
		// denormalized snapshot。
		ProductBlueprintCategory: value.ProductBlueprintCategory,

		// カテゴリ依存項目はProductBlueprint直下ではなく
		// CategoryFieldsへ集約する。
		CategoryFields: cloneCategoryFields(
			value.CategoryFields,
		),

		ProductIdTag: value.ProductIdTag,
		AssigneeID:   value.AssigneeID,

		// companyIdは認証Contextを正とし、
		// request由来の値は使用しない。
		CompanyID: companyID,

		CreatedBy: value.CreatedBy,

		// Domain validationでCreatedAtが必須のため、
		// Usecaseで必ず設定する。
		CreatedAt: &createdAt,

		// ModelRefsはModelUsecase側でmodels collectionを正として
		// 同期するため、起票時点では空にする。
		ModelRefs: nil,
	}

	created, err := u.repo.Create(
		ctx,
		input,
	)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	// ------------------------------------------------------------
	// ProductBlueprintReviewの初期化
	// ------------------------------------------------------------

	if u.reviewInit != nil {
		if err := u.reviewInit.InitForProductBlueprint(
			ctx,
			created.ID,
			created.CompanyID,
			created.CreatedAt,
			created.CreatedBy,
		); err != nil {
			return productbpdom.ProductBlueprint{}, err
		}
	}

	return created, nil
}

// MarkPrintedはprintedをtrueへ更新します。
//
// Handlerから
// /product-blueprints/{id}/mark-printed
// などで呼び出される想定です。
func (
	u *ProductBlueprintUsecase,
) MarkPrinted(
	ctx context.Context,
	id string,
) (productbpdom.ProductBlueprint, error) {
	if u == nil || u.repo == nil {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInternal
	}

	if id == "" {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInvalidID
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInvalidCompanyID
	}

	current, err := u.repo.GetByID(
		ctx,
		id,
	)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if current.CompanyID == "" ||
		current.CompanyID != companyID {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrForbidden
	}

	updated, err := u.repo.MarkPrinted(
		ctx,
		id,
	)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	return updated, nil
}

// Update updates a ProductBlueprint using Patch.
//
//   - companyId境界はUsecaseで確認する。
//   - Update APIではModelRefsを受け取らない。
//   - ModelRefsの同期はModelUsecaseを正とする。
//   - UpdatedByはHTTP bodyから直接受け取らず、
//     Usecaseが保持する更新者情報をRepositoryへ渡す。
func (
	u *ProductBlueprintUsecase,
) Update(
	ctx context.Context,
	value productbpdom.ProductBlueprint,
) (productbpdom.ProductBlueprint, error) {
	if u == nil || u.repo == nil {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInternal
	}

	id := value.ID
	if id == "" {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInvalidID
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrInvalidCompanyID
	}

	// 既存Entityを取得し、company境界とprinted状態を確認する。
	current, err := u.repo.GetByID(
		ctx,
		id,
	)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	if current.CompanyID == "" ||
		current.CompanyID != companyID {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrForbidden
	}

	if current.Printed {
		return productbpdom.ProductBlueprint{},
			productbpdom.ErrForbidden
	}

	productName := value.ProductName
	description := value.Description
	brandID := value.BrandID
	category := value.ProductBlueprintCategory
	productIDTag := value.ProductIdTag
	assigneeID := value.AssigneeID

	var categoryFieldsPointer *productbpdom.CategoryFields

	// nilは「更新しない」、空mapは「空へ更新する」を表す。
	if value.CategoryFields != nil {
		categoryFields :=
			cloneCategoryFields(
				value.CategoryFields,
			)

		categoryFieldsPointer =
			&categoryFields
	}

	updatedBy := value.UpdatedBy

	// 呼出側で新しい更新者が設定されていない場合は、
	// 現在保存されているUpdatedByを維持する。
	if updatedBy == nil {
		updatedBy = current.UpdatedBy
	}

	patch := productbpdom.Patch{
		ProductName: &productName,
		Description: &description,

		BrandID: &brandID,

		// productBlueprintCategoriesの正データから生成済みの
		// denormalized snapshot。
		ProductBlueprintCategory: &category,

		CategoryFields: categoryFieldsPointer,

		ProductIdTag: &productIDTag,

		AssigneeID: &assigneeID,

		// HTTP request bodyからは受け取らず、
		// UsecaseからRepositoryへ渡す内部値。
		UpdatedBy: updatedBy,

		// CompanyIDは通常更新しない。
		CompanyID: nil,

		// ModelRefsはModelUsecase側で同期するため、
		// 通常のProductBlueprint更新では渡さない。
		ModelRefs: nil,
	}

	updated, err := u.repo.Update(
		ctx,
		id,
		patch,
	)
	if err != nil {
		return productbpdom.ProductBlueprint{}, err
	}

	return updated, nil
}

// Delete physically deletes a ProductBlueprint.
func (
	u *ProductBlueprintUsecase,
) Delete(
	ctx context.Context,
	id string,
) error {
	if u == nil || u.repo == nil {
		return productbpdom.ErrInternal
	}

	if id == "" {
		return productbpdom.ErrInvalidID
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return productbpdom.ErrInvalidCompanyID
	}

	current, err := u.repo.GetByID(
		ctx,
		id,
	)
	if err != nil {
		return err
	}

	if current.CompanyID == "" ||
		current.CompanyID != companyID {
		return productbpdom.ErrForbidden
	}

	if current.Printed {
		return productbpdom.ErrForbidden
	}

	return u.repo.Delete(
		ctx,
		id,
	)
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func cloneCategoryFields(
	input productbpdom.CategoryFields,
) productbpdom.CategoryFields {
	if input == nil {
		return nil
	}

	output := make(
		productbpdom.CategoryFields,
		len(input),
	)

	for key, value := range input {
		if key == "" {
			continue
		}

		output[key] = cloneCategoryFieldValue(
			value,
		)
	}

	return output
}

func cloneCategoryFieldValue(
	value any,
) any {
	switch typedValue := value.(type) {
	case map[string]any:
		output := make(
			map[string]any,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			output[key] =
				cloneCategoryFieldValue(
					nestedValue,
				)
		}

		return output

	case []any:
		output := make(
			[]any,
			len(typedValue),
		)

		for index, nestedValue := range typedValue {
			output[index] =
				cloneCategoryFieldValue(
					nestedValue,
				)
		}

		return output

	case []string:
		return append(
			[]string(nil),
			typedValue...,
		)

	case []int:
		return append(
			[]int(nil),
			typedValue...,
		)

	case []float64:
		return append(
			[]float64(nil),
			typedValue...,
		)

	case map[string]string:
		output := make(
			map[string]string,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			output[key] = nestedValue
		}

		return output

	case map[string]int:
		output := make(
			map[string]int,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			output[key] = nestedValue
		}

		return output

	case map[string]float64:
		output := make(
			map[string]float64,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			output[key] = nestedValue
		}

		return output

	case map[string]bool:
		output := make(
			map[string]bool,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			output[key] = nestedValue
		}

		return output

	default:
		return value
	}
}
