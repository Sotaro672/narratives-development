// frontend/console/shell/src/features/productBlueprint/domain/productBlueprintCategory.ts

/**
 * category input schema の categoryKind に使用する値。
 *
 * ProductBlueprintCategory master の永続化項目ではない。
 */
export type ProductBlueprintCategoryKind =
  | "apparel"
  | "alcohol"
  | "cosmetics"
  | "healthcare"
  | "other";

/**
 * ProductBlueprintCategory の path。
 *
 * 例:
 * ["apparel", "tops"]
 * ["alcohol", "sake"]
 */
export type ProductBlueprintCategoryPath =
  string[];

/**
 * Firestore の productBlueprintCategories に保存されるカテゴリマスタ。
 *
 * category master の正は productBlueprintCategoryPath のみとする。
 */
export interface ProductBlueprintCategory {
  productBlueprintCategoryPath: ProductBlueprintCategoryPath;
}

/**
 * backend/internal/domain/productBlueprintCategory.InputFieldScope に対応。
 */
export type CategoryInputFieldScope =
  | "productBlueprint"
  | "model";

/**
 * backend/internal/domain/productBlueprintCategory.InputFieldType に対応。
 */
export type CategoryInputFieldType =
  | "text"
  | "textarea"
  | "number"
  | "select"
  | "multiSelect"
  | "boolean"
  | "date";

/**
 * backend/internal/domain/productBlueprintCategory.CategoryInputFieldDefinition に対応。
 */
export interface CategoryInputFieldDefinition {
  scope: CategoryInputFieldScope;
  key: string;
  label: string;
  type: CategoryInputFieldType;
  required: boolean;
  unit?: string;
}

/**
 * backend/internal/domain/productBlueprintCategory.CategoryInputSchema に対応。
 *
 * categoryCode / categoryKind / categoryNameJa は
 * category master の永続化項目ではなく、
 * frontend の入力 schema を構築するための metadata として扱う。
 */
export interface CategoryInputSchema {
  categoryCode: string;
  categoryKind: ProductBlueprintCategoryKind;
  categoryNameJa: string;
  productBlueprintFields: CategoryInputFieldDefinition[];
  modelFields: CategoryInputFieldDefinition[];
}

/**
 * productBlueprint.CategoryFields に保存するカテゴリ別入力値。
 *
 * 注意:
 * - brandId / productName / productIdTagType / description は ProductBlueprint の共通 field。
 * - これらは categoryFields には入れない。
 * - color / size / measurements は model variation 側。
 * - これらも categoryFields には入れない。
 * - washTags は apparel の ProductBlueprint 側へ保存する必須 field。
 */
export type CategoryFieldPrimitiveValue =
  | string
  | number
  | boolean
  | null;

export type CategoryFieldArrayValue =
  CategoryFieldPrimitiveValue[];

export type CategoryFieldObjectValue = Record<
  string,
  CategoryFieldPrimitiveValue
>;

export type CategoryFieldValue =
  | CategoryFieldPrimitiveValue
  | CategoryFieldArrayValue
  | CategoryFieldObjectValue;

export type CategoryFieldValues = Record<
  string,
  CategoryFieldValue
>;

/**
 * apparel の洗濯表示に使用する正式な categoryFields key。
 */
export const WASH_TAGS_CATEGORY_FIELD_KEY =
  "washTags" as const;

export type WashTagsCategoryFieldKey =
  typeof WASH_TAGS_CATEGORY_FIELD_KEY;

/**
 * apparel の洗濯表示。
 *
 * ProductBlueprint.categoryFields.washTags に保存する。
 * apparel では最低1件の選択を必須とする。
 */
export type WashTags = string[];

/**
 * apparel の ProductBlueprint.categoryFields。
 *
 * washTags は必須。
 * weight / fit / material はカテゴリによって任意または非対象。
 */
export type ApparelCategoryFieldValues =
  CategoryFieldValues & {
    washTags: WashTags;
  };

/**
 * washTags の値が有効かを判定する。
 *
 * 有効条件:
 * - 配列である
 * - 1件以上存在する
 * - 全項目が空文字ではない文字列である
 */
export function isValidWashTags(
  value: unknown,
): value is WashTags {
  return (
    Array.isArray(value) &&
    value.length > 0 &&
    value.every(
      (item): item is string =>
        typeof item === "string" &&
        item.trim() !== "",
    )
  );
}

/**
 * ProductBlueprintCategoryKindとして有効な値か判定する。
 *
 * category master の validation ではなく、
 * category input schema metadata の validation に使用する。
 */
export function isValidProductBlueprintCategoryKind(
  value: string | null | undefined,
): value is ProductBlueprintCategoryKind {
  return (
    value === "apparel" ||
    value === "alcohol" ||
    value === "cosmetics" ||
    value === "healthcare" ||
    value === "other"
  );
}

/**
 * productBlueprintCategoryPath を
 * category schema registry 用の key に変換する。
 *
 * 例:
 * ["apparel", "tops"]
 * ↓
 * "apparel.tops"
 */
export function toProductBlueprintCategoryPathKey(
  productBlueprintCategoryPath: ProductBlueprintCategoryPath,
): string {
  return productBlueprintCategoryPath.join(".");
}

/**
 * productBlueprintCategoryPath の root を返す。
 *
 * 例:
 * ["alcohol", "sake"]
 * ↓
 * "alcohol"
 */
export function getProductBlueprintCategoryRoot(
  productBlueprintCategoryPath: ProductBlueprintCategoryPath,
): string | null {
  if (productBlueprintCategoryPath.length === 0) {
    return null;
  }

  return productBlueprintCategoryPath[0] ?? null;
}