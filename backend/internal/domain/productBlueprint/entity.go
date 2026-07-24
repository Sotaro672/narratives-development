// backend/internal/domain/productBlueprint/entity.go
package productBlueprint

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"

	"narratives/internal/domain/common"
)

// ======================================
// Common errors
// ======================================

var (
	ErrNotFound = errors.New(
		"productBlueprint: not found",
	)

	ErrConflict = errors.New(
		"productBlueprint: conflict",
	)

	ErrInvalid = errors.New(
		"productBlueprint: invalid",
	)

	ErrUnauthorized = errors.New(
		"productBlueprint: unauthorized",
	)

	ErrForbidden = errors.New(
		"productBlueprint: forbidden",
	)

	ErrInternal = errors.New(
		"productBlueprint: internal",
	)
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

func WrapInvalid(
	err error,
	message string,
) error {
	if err == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrInvalid,
			message,
		)
	}

	return fmt.Errorf(
		"%w: %s: %w",
		ErrInvalid,
		message,
		err,
	)
}

func WrapConflict(
	err error,
	message string,
) error {
	if err == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrConflict,
			message,
		)
	}

	return fmt.Errorf(
		"%w: %s: %w",
		ErrConflict,
		message,
		err,
	)
}

func WrapNotFound(
	err error,
	message string,
) error {
	if err == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrNotFound,
			message,
		)
	}

	return fmt.Errorf(
		"%w: %s: %w",
		ErrNotFound,
		message,
		err,
	)
}

// ======================================
// ID
// ======================================

// NewIDは16byteの暗号学的乱数をhex文字列へ変換し、
// 32文字のProductBlueprint IDを生成します。
//
// crypto/randによる生成に失敗した場合は、衝突の可能性がある
// fallback IDを生成せず、errorを返します。
func NewID() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "",
			fmt.Errorf(
				"%w: generate product blueprint id: %w",
				ErrInternal,
				err,
			)
	}

	return hex.EncodeToString(randomBytes), nil
}

// ======================================
// ProductIDTagType
// ======================================

// ProductIDTagTypeは、商品へ付与する識別タグの種類です。
//
// stringのaliasにはせず、ProductIDTagTypeと通常のstringを
// 明確に区別します。
type ProductIDTagType string

const (
	TagQR ProductIDTagType = "qr"

	TagNFC ProductIDTagType = "nfc"
)

func IsValidTagType(
	value ProductIDTagType,
) bool {
	switch value {
	case TagQR, TagNFC:
		return true

	default:
		return false
	}
}

// ======================================
// Value objects
// ======================================

type ProductIDTag struct {
	Type ProductIDTagType
}

func (tag ProductIDTag) Validate() error {
	return tag.validate()
}

func (tag ProductIDTag) validate() error {
	if !IsValidTagType(tag.Type) {
		return ErrInvalidTagType
	}

	return nil
}

// ProductBlueprintCategorySnapshotは、ProductBlueprint側へ
// denormalize保存するカテゴリ表示用情報です。
//
// 正のカテゴリ定義はproductBlueprintCategory domainと
// productBlueprintCategories collection側で管理します。
type ProductBlueprintCategorySnapshot struct {
	ID     string
	Code   string
	NameJa string
	NameEn string
	Kind   common.ProductCategoryKind
	Path   []string
}

func (
	snapshot ProductBlueprintCategorySnapshot,
) Validate() error {
	return snapshot.validate()
}

func (
	snapshot ProductBlueprintCategorySnapshot,
) validate() error {
	if snapshot.ID == "" {
		return ErrInvalidCategoryID
	}

	if snapshot.Code == "" {
		return ErrInvalidCategoryCode
	}

	if snapshot.NameJa == "" {
		return ErrInvalidCategoryNameJa
	}

	if !common.IsValidProductCategoryKind(
		snapshot.Kind,
	) {
		return ErrInvalidCategoryKind
	}

	for _, pathElement := range snapshot.Path {
		if pathElement == "" {
			return ErrInvalidCategoryCode
		}
	}

	return nil
}

// ======================================
// CategoryFields
// ======================================

// CategoryFieldsは、カテゴリ固有のProductBlueprint入力値を保持します。
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
// model domainのAlcoholModelVariation.Volumeだけを正とします。
//
// color、size、measurementsなどのModel variation固有値も
// CategoryFieldsへ保存しません。
//
// brandId、productName、productIdTagType、descriptionなどの
// ProductBlueprint共通fieldもCategoryFieldsへ保存しません。
type CategoryFields map[string]any

// CategoryFieldsValidatorは、productBlueprintCategory側で管理する
// 正規schemaを使ってCategoryFieldsを検証するための関数型Portです。
//
// ProductBlueprint domain内へカテゴリごとのfield定義を複製せず、
// application/usecase側から正規schemaに基づくvalidatorを渡します。
type CategoryFieldsValidator func(
	category ProductBlueprintCategorySnapshot,
	fields CategoryFields,
) error

func validateCategoryFields(
	category ProductBlueprintCategorySnapshot,
	fields CategoryFields,
	validator CategoryFieldsValidator,
) error {
	if err := validateCategoryFieldsStructure(
		fields,
	); err != nil {
		return err
	}

	if validator == nil {
		return ErrCategoryFieldsValidatorNotConfigured
	}

	if err := validator(
		category,
		cloneCategoryFields(fields),
	); err != nil {
		return WrapInvalid(
			err,
			"categoryFields do not match category schema",
		)
	}

	return nil
}

func validateCategoryFieldsStructure(
	fields CategoryFields,
) error {
	for key := range fields {
		if key == "" {
			return WrapInvalid(
				ErrInvalidCategoryFields,
				"categoryFields key is empty",
			)
		}
	}

	return nil
}

func cloneCategoryFields(
	fields CategoryFields,
) CategoryFields {
	if fields == nil {
		return nil
	}

	cloned := make(
		CategoryFields,
		len(fields),
	)

	for key, value := range fields {
		cloned[key] = cloneCategoryFieldValue(value)
	}

	return cloned
}

func cloneCategoryFieldValue(
	value any,
) any {
	switch typedValue := value.(type) {
	case map[string]any:
		cloned := make(
			map[string]any,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			cloned[key] =
				cloneCategoryFieldValue(
					nestedValue,
				)
		}

		return cloned

	case []any:
		cloned := make(
			[]any,
			len(typedValue),
		)

		for index, nestedValue := range typedValue {
			cloned[index] =
				cloneCategoryFieldValue(
					nestedValue,
				)
		}

		return cloned

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
		cloned := make(
			map[string]string,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			cloned[key] = nestedValue
		}

		return cloned

	case map[string]int:
		cloned := make(
			map[string]int,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			cloned[key] = nestedValue
		}

		return cloned

	case map[string]float64:
		cloned := make(
			map[string]float64,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			cloned[key] = nestedValue
		}

		return cloned

	case map[string]bool:
		cloned := make(
			map[string]bool,
			len(typedValue),
		)

		for key, nestedValue := range typedValue {
			cloned[key] = nestedValue
		}

		return cloned

	default:
		return value
	}
}

// ======================================
// Model references
// ======================================

// ModelRefはProductBlueprint配下に紐づくModelの参照です.
//
// ModelID:
//   - models collectionのdocument ID
//
// DisplayOrder:
//   - ProductBlueprint内での表示順
//   - 1から始まる連番
type ModelRef struct {
	ModelID      string
	DisplayOrder int
}

// normalizeModelRefsはModel IDを正規化し、
// 入力順を維持したままDisplayOrderを1..Nで採番します。
//
// 空IDと重複IDは除外します。
func normalizeModelRefs(
	modelIDs []string,
) []ModelRef {
	seen := make(
		map[string]struct{},
		len(modelIDs),
	)

	refs := make(
		[]ModelRef,
		0,
		len(modelIDs),
	)

	for _, modelID := range modelIDs {
		if modelID == "" {
			continue
		}

		if _, exists := seen[modelID]; exists {
			continue
		}

		seen[modelID] = struct{}{}

		refs = append(
			refs,
			ModelRef{
				ModelID: modelID,
				DisplayOrder: len(
					refs,
				) + 1,
			},
		)
	}

	return refs
}

// normalizeExistingModelRefsは既存ModelRefをDisplayOrder順に並べ、
// 空IDと重複IDを除外したうえで1..Nへ採番し直します。
func normalizeExistingModelRefs(
	refs []ModelRef,
) []ModelRef {
	copied := append(
		[]ModelRef(nil),
		refs...,
	)

	sort.SliceStable(
		copied,
		func(i, j int) bool {
			return copied[i].DisplayOrder <
				copied[j].DisplayOrder
		},
	)

	modelIDs := make(
		[]string,
		0,
		len(copied),
	)

	for _, ref := range copied {
		modelIDs = append(
			modelIDs,
			ref.ModelID,
		)
	}

	return normalizeModelRefs(modelIDs)
}

// mergeAndRenumberModelRefsは既存参照と追加Model IDをマージし、
// 既存順序を維持しながら追加分を末尾へ配置します。
//
// 空IDと重複IDは除外し、DisplayOrderを1..Nへ採番し直します。
func mergeAndRenumberModelRefs(
	existing []ModelRef,
	appendIDs []string,
) []ModelRef {
	normalizedExisting :=
		normalizeExistingModelRefs(existing)

	seen := make(
		map[string]struct{},
		len(normalizedExisting)+len(appendIDs),
	)

	modelIDs := make(
		[]string,
		0,
		len(normalizedExisting)+len(appendIDs),
	)

	for _, ref := range normalizedExisting {
		seen[ref.ModelID] = struct{}{}

		modelIDs = append(
			modelIDs,
			ref.ModelID,
		)
	}

	for _, modelID := range appendIDs {
		if modelID == "" {
			continue
		}

		if _, exists := seen[modelID]; exists {
			continue
		}

		seen[modelID] = struct{}{}

		modelIDs = append(
			modelIDs,
			modelID,
		)
	}

	return normalizeModelRefs(modelIDs)
}

func validateModelRefs(
	refs []ModelRef,
) error {
	seen := make(
		map[string]struct{},
		len(refs),
	)

	for index, ref := range refs {
		if ref.ModelID == "" {
			return WrapInvalid(
				nil,
				"modelRefs.modelId is empty",
			)
		}

		if _, exists := seen[ref.ModelID]; exists {
			return WrapInvalid(
				nil,
				"modelRefs.modelId is duplicated",
			)
		}

		seen[ref.ModelID] = struct{}{}

		expectedDisplayOrder := index + 1

		if ref.DisplayOrder != expectedDisplayOrder {
			return WrapInvalid(
				nil,
				fmt.Sprintf(
					"modelRefs.displayOrder must be %d",
					expectedDisplayOrder,
				),
			)
		}
	}

	return nil
}

// ======================================
// Entity
// ======================================

type ProductBlueprint struct {
	ID string

	ProductName string

	// Descriptionは空文字を許容します。
	Description string

	CompanyID string
	BrandID   string

	ProductBlueprintCategory ProductBlueprintCategorySnapshot

	// CategoryFieldsはカテゴリ固有のProductBlueprint入力値です。
	//
	// color、size、measurements、alcohol volumeなどの
	// Model variation固有値は保存しません。
	CategoryFields CategoryFields

	ProductIdTag ProductIDTag
	AssigneeID   string

	// ModelRefsはModel IDと1..NのDisplayOrderを保持します。
	ModelRefs []ModelRef

	// Printed:
	//   - false: 未印刷
	//   - true: 印刷済み
	//
	// 印刷後はProductBlueprintと配下Modelの変更を禁止します。
	Printed bool

	CreatedBy *string
	CreatedAt time.Time
	UpdatedBy *string
	UpdatedAt time.Time
}

// ======================================
// Errors
// ======================================

var (
	ErrInvalidID = errors.New(
		"productBlueprint: invalid id",
	)

	ErrInvalidProduct = errors.New(
		"productBlueprint: invalid productName",
	)

	ErrInvalidBrand = errors.New(
		"productBlueprint: invalid brandId",
	)

	ErrInvalidTagType = errors.New(
		"productBlueprint: invalid productIdTag.type",
	)

	ErrInvalidCreatedAt = errors.New(
		"productBlueprint: invalid createdAt",
	)

	ErrInvalidAssignee = errors.New(
		"productBlueprint: invalid assigneeId",
	)

	ErrInvalidCompanyID = errors.New(
		"productBlueprint: invalid companyId",
	)

	ErrInvalidCategoryID = errors.New(
		"productBlueprint: invalid productBlueprintCategory.id",
	)

	ErrInvalidCategoryCode = errors.New(
		"productBlueprint: invalid productBlueprintCategory.code",
	)

	ErrInvalidCategoryNameJa = errors.New(
		"productBlueprint: invalid productBlueprintCategory.nameJa",
	)

	ErrInvalidCategoryKind = errors.New(
		"productBlueprint: invalid productBlueprintCategory.kind",
	)

	ErrInvalidCategoryFields = errors.New(
		"productBlueprint: invalid categoryFields",
	)

	ErrCategoryFieldsValidatorNotConfigured = errors.New(
		"productBlueprint: categoryFields validator is not configured",
	)
)

// ======================================
// Constructor
// ======================================

func New(
	id string,
	productName string,
	description string,
	brandID string,
	category ProductBlueprintCategorySnapshot,
	categoryFields CategoryFields,
	productIDTag ProductIDTag,
	assigneeID string,
	createdBy *string,
	createdAt time.Time,
	companyID string,
	categoryFieldsValidator CategoryFieldsValidator,
) (ProductBlueprint, error) {
	productBlueprint := ProductBlueprint{
		ID:                       id,
		ProductName:              productName,
		Description:              description,
		BrandID:                  brandID,
		ProductBlueprintCategory: category,
		CategoryFields: cloneCategoryFields(
			categoryFields,
		),
		ProductIdTag: productIDTag,
		AssigneeID:   assigneeID,
		CompanyID:    companyID,

		ModelRefs: nil,
		Printed:   false,

		CreatedBy: createdBy,
		CreatedAt: createdAt.UTC(),
		UpdatedBy: createdBy,
		UpdatedAt: createdAt.UTC(),
	}

	if err := productBlueprint.validate(); err != nil {
		return ProductBlueprint{}, err
	}

	if err := validateCategoryFields(
		productBlueprint.ProductBlueprintCategory,
		productBlueprint.CategoryFields,
		categoryFieldsValidator,
	); err != nil {
		return ProductBlueprint{}, err
	}

	return productBlueprint, nil
}

// ======================================
// Modification policy
// ======================================

// CanModifyはProductBlueprintと配下Modelを変更可能か返します。
//
// ProductBlueprintが印刷済みの場合はfalseです。
// Model側のCreate、Update、Delete、Replaceでも、
// 同じProductBlueprint境界を使って変更を拒否します。
func (
	productBlueprint ProductBlueprint,
) CanModify() bool {
	return productBlueprint.canModify()
}

// canModifyは印刷後の変更禁止規則を表します。
func (
	productBlueprint ProductBlueprint,
) canModify() bool {
	return !productBlueprint.Printed
}

// MarkPrintedはProductBlueprintを印刷済みに変更します。
//
// 印刷確定前にEntity全体とCategoryFields schemaを再検証します。
func (
	productBlueprint *ProductBlueprint,
) MarkPrinted(
	now time.Time,
	updatedBy *string,
	categoryFieldsValidator CategoryFieldsValidator,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if productBlueprint.Printed {
		return nil
	}

	if err := productBlueprint.validate(); err != nil {
		return err
	}

	if err := validateCategoryFields(
		productBlueprint.ProductBlueprintCategory,
		productBlueprint.CategoryFields,
		categoryFieldsValidator,
	); err != nil {
		return err
	}

	productBlueprint.Printed = true
	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// ======================================
// Update methods
// ======================================

func (
	productBlueprint *ProductBlueprint,
) UpdateProductName(
	productName string,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if productName == "" {
		return ErrInvalidProduct
	}

	productBlueprint.ProductName = productName
	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// UpdateDescriptionは空文字を有効値として扱います。
// 空文字は説明文を削除する操作を表します。
func (
	productBlueprint *ProductBlueprint,
) UpdateDescription(
	description string,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	productBlueprint.Description = description
	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

func (
	productBlueprint *ProductBlueprint,
) UpdateBrand(
	brandID string,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if brandID == "" {
		return ErrInvalidBrand
	}

	productBlueprint.BrandID = brandID
	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// UpdateCategoryはカテゴリを変更し、現在のCategoryFieldsが
// 新しいカテゴリの正規schemaに適合することを検証します。
//
// カテゴリとCategoryFieldsを同時に変更する場合は、
// UpdateCategoryAndFieldsを使用します。
func (
	productBlueprint *ProductBlueprint,
) UpdateCategory(
	category ProductBlueprintCategorySnapshot,
	categoryFieldsValidator CategoryFieldsValidator,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if err := category.validate(); err != nil {
		return err
	}

	if err := validateCategoryFields(
		category,
		productBlueprint.CategoryFields,
		categoryFieldsValidator,
	); err != nil {
		return err
	}

	productBlueprint.ProductBlueprintCategory =
		category

	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// UpdateCategoryFieldsは現在のカテゴリの正規schemaに基づいて
// CategoryFieldsを検証してから置換します。
func (
	productBlueprint *ProductBlueprint,
) UpdateCategoryFields(
	fields CategoryFields,
	categoryFieldsValidator CategoryFieldsValidator,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if err := validateCategoryFields(
		productBlueprint.ProductBlueprintCategory,
		fields,
		categoryFieldsValidator,
	); err != nil {
		return err
	}

	productBlueprint.CategoryFields =
		cloneCategoryFields(fields)

	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// UpdateCategoryAndFieldsはカテゴリとCategoryFieldsを
// 同一のdomain操作として更新します。
//
// 新しいカテゴリの正規schemaに対して新しいCategoryFieldsを検証し、
// 検証に成功した場合だけ両方を更新します。
func (
	productBlueprint *ProductBlueprint,
) UpdateCategoryAndFields(
	category ProductBlueprintCategorySnapshot,
	fields CategoryFields,
	categoryFieldsValidator CategoryFieldsValidator,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if err := category.validate(); err != nil {
		return err
	}

	if err := validateCategoryFields(
		category,
		fields,
		categoryFieldsValidator,
	); err != nil {
		return err
	}

	productBlueprint.ProductBlueprintCategory =
		category

	productBlueprint.CategoryFields =
		cloneCategoryFields(fields)

	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

func (
	productBlueprint *ProductBlueprint,
) UpdateAssignee(
	assigneeID string,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if assigneeID == "" {
		return ErrInvalidAssignee
	}

	productBlueprint.AssigneeID = assigneeID
	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

func (
	productBlueprint *ProductBlueprint,
) UpdateTag(
	tag ProductIDTag,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	if err := tag.validate(); err != nil {
		return err
	}

	productBlueprint.ProductIdTag = tag
	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// UpdateModelIDsはModel IDを受け取り、入力順を維持したまま
// DisplayOrderを1..Nで採番してModelRefsを置換します。
func (
	productBlueprint *ProductBlueprint,
) UpdateModelIDs(
	modelIDs []string,
	now time.Time,
	updatedBy *string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	productBlueprint.ModelRefs =
		normalizeModelRefs(modelIDs)

	productBlueprint.touch(
		now,
		updatedBy,
	)

	return nil
}

// AppendModelIDsNoTouchは、起票後にModel IDを追記するための操作です。
//
// UpdatedAtとUpdatedByは変更しません。
// 印刷後の追記は禁止します。
func (
	productBlueprint *ProductBlueprint,
) AppendModelIDsNoTouch(
	modelIDs []string,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	productBlueprint.ModelRefs =
		mergeAndRenumberModelRefs(
			productBlueprint.ModelRefs,
			modelIDs,
		)

	return nil
}

// ReplaceModelRefsWithoutTouchはmodels collectionを正として
// ModelRefs全体を同期するための操作です。
//
// UpdatedAtとUpdatedByは変更しません。
// 印刷後の同期は禁止します。
func (
	productBlueprint *ProductBlueprint,
) ReplaceModelRefsWithoutTouch(
	refs []ModelRef,
) error {
	if productBlueprint == nil {
		return ErrInvalid
	}

	if !productBlueprint.canModify() {
		return ErrForbidden
	}

	productBlueprint.ModelRefs =
		normalizeExistingModelRefs(refs)

	return nil
}

// ======================================
// Validation
// ======================================

// ValidateはEntityの共通不変条件を検証します。
//
// CategoryFieldsのカテゴリ別schema検証は、正規schemaを保持する
// productBlueprintCategory側のvalidatorが必要なため、
// ValidateCategoryFieldsまたは各更新メソッドで実行します。
func (
	productBlueprint ProductBlueprint,
) Validate() error {
	return productBlueprint.validate()
}

// ValidateCategoryFieldsは現在のカテゴリに対して
// CategoryFieldsのschema検証を実行します。
func (
	productBlueprint ProductBlueprint,
) ValidateCategoryFields(
	categoryFieldsValidator CategoryFieldsValidator,
) error {
	if err := productBlueprint.validate(); err != nil {
		return err
	}

	return validateCategoryFields(
		productBlueprint.ProductBlueprintCategory,
		productBlueprint.CategoryFields,
		categoryFieldsValidator,
	)
}

func (
	productBlueprint ProductBlueprint,
) validate() error {
	if productBlueprint.ID == "" {
		return ErrInvalidID
	}

	if productBlueprint.ProductName == "" {
		return ErrInvalidProduct
	}

	// Descriptionは空文字を許容するため検証しません。

	if productBlueprint.BrandID == "" {
		return ErrInvalidBrand
	}

	if err :=
		productBlueprint.
			ProductBlueprintCategory.
			validate(); err != nil {
		return err
	}

	if productBlueprint.CompanyID == "" {
		return ErrInvalidCompanyID
	}

	if err :=
		productBlueprint.
			ProductIdTag.
			validate(); err != nil {
		return err
	}

	if productBlueprint.AssigneeID == "" {
		return ErrInvalidAssignee
	}

	if productBlueprint.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if err := validateCategoryFieldsStructure(
		productBlueprint.CategoryFields,
	); err != nil {
		return err
	}

	if err := validateModelRefs(
		productBlueprint.ModelRefs,
	); err != nil {
		return err
	}

	return nil
}

// ======================================
// Helpers
// ======================================

func (
	productBlueprint *ProductBlueprint,
) touch(
	now time.Time,
	updatedBy *string,
) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	productBlueprint.UpdatedAt =
		now.UTC()

	productBlueprint.UpdatedBy =
		updatedBy
}
