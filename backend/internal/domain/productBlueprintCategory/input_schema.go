// backend/internal/domain/productBlueprintCategory/input_schema.go
package productBlueprintCategory

// InputFieldScope は、その入力項目がどの保存単位に属するかを表す。
// - productBlueprint: productBlueprint document に保存する項目
// - model: model variation 側に保存する項目
type InputFieldScope string

const (
	InputFieldScopeProductBlueprint InputFieldScope = "productBlueprint"
	InputFieldScopeModel            InputFieldScope = "model"
)

type InputFieldType string

const (
	InputFieldTypeText        InputFieldType = "text"
	InputFieldTypeTextarea    InputFieldType = "textarea"
	InputFieldTypeNumber      InputFieldType = "number"
	InputFieldTypeSelect      InputFieldType = "select"
	InputFieldTypeMultiSelect InputFieldType = "multiSelect"
	InputFieldTypeBoolean     InputFieldType = "boolean"
	InputFieldTypeDate        InputFieldType = "date"
)

// CategoryInputFieldDefinition は category ごとの入力項目定義。
// Key は frontend / backend / Firestore の categoryFields key または model variation key として使う。
type CategoryInputFieldDefinition struct {
	Scope    InputFieldScope `json:"scope"`
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Type     InputFieldType  `json:"type"`
	Required bool            `json:"required"`
	Unit     string          `json:"unit,omitempty"`
}

// CategoryInputSchema は category code ごとの入力 schema。
// ProductBlueprintFields は productBlueprint 側に保存する項目。
// ModelFields は model variation 側に保存する項目。
type CategoryInputSchema struct {
	CategoryCode           string                         `json:"categoryCode"`
	CategoryKind           string                         `json:"categoryKind"`
	CategoryNameJa         string                         `json:"categoryNameJa"`
	ProductBlueprintFields []CategoryInputFieldDefinition `json:"productBlueprintFields"`
	ModelFields            []CategoryInputFieldDefinition `json:"modelFields"`
}

// ------------------------------------------------------------
// Common productBlueprint fields
// ------------------------------------------------------------

var commonProductBlueprintFields = []CategoryInputFieldDefinition{
	{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "brandId",
		Label:    "ブランドID",
		Type:     InputFieldTypeText,
		Required: true,
	},
	{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "productName",
		Label:    "商品名",
		Type:     InputFieldTypeText,
		Required: true,
	},
	{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "productIdTagType",
		Label:    "商品IDタグ",
		Type:     InputFieldTypeText,
		Required: true,
	},
	{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "description",
		Label:    "説明",
		Type:     InputFieldTypeTextarea,
		Required: false,
	},
}

func withCommonProductBlueprintFields(extra ...CategoryInputFieldDefinition) []CategoryInputFieldDefinition {
	out := make([]CategoryInputFieldDefinition, 0, len(commonProductBlueprintFields)+len(extra))
	out = append(out, commonProductBlueprintFields...)
	out = append(out, extra...)
	return out
}

// ------------------------------------------------------------
// Reusable productBlueprint fields
// ------------------------------------------------------------
//
// NOTE:
// model variation 側の値は model domain を正とする。
// そのため、酒類の volume は productBlueprint field ではなく model field として扱う。

var (
	fieldWeight = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "weight",
		Label:    "重量",
		Type:     InputFieldTypeNumber,
		Required: false,
		Unit:     "g",
	}

	fieldVintage = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "vintage",
		Label:    "ヴィンテージ",
		Type:     InputFieldTypeNumber,
		Required: false,
	}

	fieldRegion = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "region",
		Label:    "地域・産地",
		Type:     InputFieldTypeText,
		Required: false,
	}

	fieldFit = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "fit",
		Label:    "フィット",
		Type:     InputFieldTypeText,
		Required: false,
	}

	fieldMaterial = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "material",
		Label:    "素材",
		Type:     InputFieldTypeText,
		Required: false,
	}

	fieldAlcoholContent = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeProductBlueprint,
		Key:      "alcoholContent",
		Label:    "アルコール度数",
		Type:     InputFieldTypeNumber,
		Required: false,
		Unit:     "%",
	}
)

// ------------------------------------------------------------
// Reusable model fields
// ------------------------------------------------------------
//
// NOTE:
// ここにある定義は model domain の値オブジェクト定義ではなく、
// category ごとの入力 schema metadata。
// 実際の保存構造・validation の正は model domain 側。
// - apparel: Color / Size / Measurements
// - alcohol: Volume

var (
	modelFieldColor = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeModel,
		Key:      "color",
		Label:    "カラー",
		Type:     InputFieldTypeText,
		Required: false,
	}

	modelFieldSize = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeModel,
		Key:      "size",
		Label:    "サイズ",
		Type:     InputFieldTypeText,
		Required: false,
	}

	modelFieldMeasurements = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeModel,
		Key:      "measurements",
		Label:    "採寸",
		Type:     InputFieldTypeTextarea,
		Required: false,
	}

	modelFieldVolume = CategoryInputFieldDefinition{
		Scope:    InputFieldScopeModel,
		Key:      "volume",
		Label:    "容量",
		Type:     InputFieldTypeNumber,
		Required: true,
		Unit:     "ml",
	}
)

var modelFieldsColorSize = []CategoryInputFieldDefinition{
	modelFieldColor,
	modelFieldSize,
}

var modelFieldsColorSizeMeasurements = []CategoryInputFieldDefinition{
	modelFieldColor,
	modelFieldSize,
	modelFieldMeasurements,
}

var modelFieldsVolume = []CategoryInputFieldDefinition{
	modelFieldVolume,
}

// ------------------------------------------------------------
// Category codes
// ------------------------------------------------------------

const (
	// alcohol
	CategoryCodeAlcoholBeer    = "alcohol.beer"
	CategoryCodeAlcoholSake    = "alcohol.sake"
	CategoryCodeAlcoholShochu  = "alcohol.shochu"
	CategoryCodeAlcoholSpirits = "alcohol.spirits"
	CategoryCodeAlcoholWhisky  = "alcohol.whisky"
	CategoryCodeAlcoholWine    = "alcohol.wine"

	// apparel
	CategoryCodeApparelAccessory = "apparel.accessory"
	CategoryCodeApparelBag       = "apparel.bag"
	CategoryCodeApparelBottoms   = "apparel.bottoms"
	CategoryCodeApparelDress     = "apparel.dress"
	CategoryCodeApparelOuterwear = "apparel.outerwear"
	CategoryCodeApparelShoes     = "apparel.shoes"
	CategoryCodeApparelTops      = "apparel.tops"

	// cosmetics
	CategoryCodeCosmeticsBodycare  = "cosmetics.bodycare"
	CategoryCodeCosmeticsFragrance = "cosmetics.fragrance"
	CategoryCodeCosmeticsHaircare  = "cosmetics.haircare"
	CategoryCodeCosmeticsMakeup    = "cosmetics.makeup"
	CategoryCodeCosmeticsSkincare  = "cosmetics.skincare"

	// healthcare
	CategoryCodeHealthcareMedicalDevice = "healthcare.medical_device"
	CategoryCodeHealthcareSupplement    = "healthcare.supplement"
	CategoryCodeHealthcareWellness      = "healthcare.wellness"

	// other
	CategoryCodeOtherGeneral = "other.general"
)

// ------------------------------------------------------------
// Category input schemas
// ------------------------------------------------------------

var categoryInputSchemaRegistry = map[string]CategoryInputSchema{
	// ------------------------------------------------------------
	// alcohol
	// productBlueprint:
	// brandId, productName, productIdTagType, description,
	// vintage, region, material, alcoholContent
	// model:
	// volume
	// ------------------------------------------------------------
	CategoryCodeAlcoholBeer:    alcoholSchema(CategoryCodeAlcoholBeer, "ビール"),
	CategoryCodeAlcoholSake:    alcoholSchema(CategoryCodeAlcoholSake, "日本酒"),
	CategoryCodeAlcoholShochu:  alcoholSchema(CategoryCodeAlcoholShochu, "焼酎"),
	CategoryCodeAlcoholSpirits: alcoholSchema(CategoryCodeAlcoholSpirits, "スピリッツ"),
	CategoryCodeAlcoholWhisky:  alcoholSchema(CategoryCodeAlcoholWhisky, "ウイスキー"),
	CategoryCodeAlcoholWine:    alcoholSchema(CategoryCodeAlcoholWine, "ワイン"),

	// ------------------------------------------------------------
	// apparel
	// ------------------------------------------------------------
	CategoryCodeApparelAccessory: {
		CategoryCode:   CategoryCodeApparelAccessory,
		CategoryKind:   "apparel",
		CategoryNameJa: "アクセサリー",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldMaterial,
		),
		ModelFields: nil,
	},
	CategoryCodeApparelBag: {
		CategoryCode:   CategoryCodeApparelBag,
		CategoryKind:   "apparel",
		CategoryNameJa: "バッグ",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldMaterial,
		),
		ModelFields: nil,
	},
	CategoryCodeApparelBottoms: {
		CategoryCode:   CategoryCodeApparelBottoms,
		CategoryKind:   "apparel",
		CategoryNameJa: "ボトムス",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldWeight,
			fieldFit,
			fieldMaterial,
		),
		ModelFields: modelFieldsColorSizeMeasurements,
	},
	CategoryCodeApparelDress: {
		CategoryCode:   CategoryCodeApparelDress,
		CategoryKind:   "apparel",
		CategoryNameJa: "ワンピース",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldWeight,
			fieldFit,
			fieldMaterial,
		),
		ModelFields: modelFieldsColorSizeMeasurements,
	},
	CategoryCodeApparelOuterwear: {
		CategoryCode:   CategoryCodeApparelOuterwear,
		CategoryKind:   "apparel",
		CategoryNameJa: "アウター",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldMaterial,
		),
		ModelFields: modelFieldsColorSize,
	},
	CategoryCodeApparelShoes: {
		CategoryCode:   CategoryCodeApparelShoes,
		CategoryKind:   "apparel",
		CategoryNameJa: "靴",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldMaterial,
		),
		ModelFields: modelFieldsColorSize,
	},
	CategoryCodeApparelTops: {
		CategoryCode:   CategoryCodeApparelTops,
		CategoryKind:   "apparel",
		CategoryNameJa: "トップス",
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldWeight,
			fieldFit,
			fieldMaterial,
		),
		ModelFields: modelFieldsColorSizeMeasurements,
	},

	// ------------------------------------------------------------
	// cosmetics
	// productBlueprint:
	// brandId, productName, productIdTagType, description,
	// material
	// model: none
	// ------------------------------------------------------------
	CategoryCodeCosmeticsBodycare:  cosmeticsSchema(CategoryCodeCosmeticsBodycare, "ボディケア"),
	CategoryCodeCosmeticsFragrance: cosmeticsSchema(CategoryCodeCosmeticsFragrance, "香水"),
	CategoryCodeCosmeticsHaircare:  cosmeticsSchema(CategoryCodeCosmeticsHaircare, "ヘアケア"),
	CategoryCodeCosmeticsMakeup:    cosmeticsSchema(CategoryCodeCosmeticsMakeup, "メイクアップ"),
	CategoryCodeCosmeticsSkincare:  cosmeticsSchema(CategoryCodeCosmeticsSkincare, "スキンケア"),

	// ------------------------------------------------------------
	// healthcare
	// productBlueprint:
	// brandId, productName, productIdTagType, description
	// model: none
	// ------------------------------------------------------------
	CategoryCodeHealthcareMedicalDevice: basicOnlySchema(CategoryCodeHealthcareMedicalDevice, "医療・衛生用品", "healthcare"),
	CategoryCodeHealthcareSupplement:    basicOnlySchema(CategoryCodeHealthcareSupplement, "サプリメント", "healthcare"),
	CategoryCodeHealthcareWellness:      basicOnlySchema(CategoryCodeHealthcareWellness, "ウェルネス用品", "healthcare"),

	// ------------------------------------------------------------
	// other
	// productBlueprint:
	// brandId, productName, productIdTagType, description
	// model: none
	// ------------------------------------------------------------
	CategoryCodeOtherGeneral: basicOnlySchema(CategoryCodeOtherGeneral, "その他一般", "other"),
}

func alcoholSchema(categoryCode string, nameJa string) CategoryInputSchema {
	return CategoryInputSchema{
		CategoryCode:   categoryCode,
		CategoryKind:   "alcohol",
		CategoryNameJa: nameJa,
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldVintage,
			fieldRegion,
			fieldMaterial,
			fieldAlcoholContent,
		),
		ModelFields: modelFieldsVolume,
	}
}

func cosmeticsSchema(categoryCode string, nameJa string) CategoryInputSchema {
	return CategoryInputSchema{
		CategoryCode:   categoryCode,
		CategoryKind:   "cosmetics",
		CategoryNameJa: nameJa,
		ProductBlueprintFields: withCommonProductBlueprintFields(
			fieldMaterial,
		),
		ModelFields: nil,
	}
}

func basicOnlySchema(categoryCode string, nameJa string, kind string) CategoryInputSchema {
	return CategoryInputSchema{
		CategoryCode:           categoryCode,
		CategoryKind:           kind,
		CategoryNameJa:         nameJa,
		ProductBlueprintFields: withCommonProductBlueprintFields(),
		ModelFields:            nil,
	}
}

// GetCategoryInputSchema returns the input schema for the category code.
func GetCategoryInputSchema(categoryCode string) (CategoryInputSchema, bool) {
	if categoryCode == "" {
		return CategoryInputSchema{}, false
	}

	schema, ok := categoryInputSchemaRegistry[categoryCode]
	return schema, ok
}

// MustGetCategoryInputSchema returns the input schema for the category code.
// It returns an empty schema when not found.
func MustGetCategoryInputSchema(categoryCode string) CategoryInputSchema {
	schema, _ := GetCategoryInputSchema(categoryCode)
	return schema
}

// ListCategoryInputSchemas returns all category input schemas.
func ListCategoryInputSchemas() []CategoryInputSchema {
	out := make([]CategoryInputSchema, 0, len(categoryInputSchemaRegistry))
	for _, schema := range categoryInputSchemaRegistry {
		out = append(out, schema)
	}
	return out
}

// HasModelFields returns true when this category requires model input.
func HasModelFields(categoryCode string) bool {
	schema, ok := GetCategoryInputSchema(categoryCode)
	if !ok {
		return false
	}

	return len(schema.ModelFields) > 0
}

// HasMeasurements returns true when this category requires model measurements.
func HasMeasurements(categoryCode string) bool {
	schema, ok := GetCategoryInputSchema(categoryCode)
	if !ok {
		return false
	}

	for _, field := range schema.ModelFields {
		if field.Key == "measurements" {
			return true
		}
	}

	return false
}
