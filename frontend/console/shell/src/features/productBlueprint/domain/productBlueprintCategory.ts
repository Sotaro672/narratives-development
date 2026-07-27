// frontend/console/shell/src/features/productBlueprint/domain/productBlueprintCategory.ts

/**
 * backend/internal/domain/common.ProductCategoryKind に対応。
 */
export type ProductBlueprintCategoryKind =
  | "apparel"
  | "alcohol"
  | "cosmetics"
  | "healthcare"
  | "other";

/**
 * productBlueprintCategory の属性フラグ。
 * backend/internal/domain/productBlueprintCategory.CategoryAttributes に対応。
 */
export interface ProductBlueprintCategoryAttributes {
  requiresExpirationDate: boolean;
  requiresLotNumber: boolean;
  requiresIngredients: boolean;
  requiresAlcoholNotice: boolean;
  requiresCosmeticNotice: boolean;
  requiresStorageMethod: boolean;
}

/**
 * Firestore の productBlueprintCategories に保存されるカテゴリマスタ。
 */
export interface ProductBlueprintCategory {
  id: string;
  code: string;
  nameJa: string;
  nameEn: string;
  parentId?: string | null;
  path: string[];
  kind: ProductBlueprintCategoryKind;
  displayOrder: number;
  attributes: ProductBlueprintCategoryAttributes;
  createdAt?: string | null;
  updatedAt?: string | null;
}

/**
 * ProductBlueprint 側に denormalize 保存されるカテゴリ snapshot。
 *
 * NOTE:
 * - parentId は category 選択 UI の親子階層判定で使う。
 * - displayOrder は category 選択 UI の並び順制御で使う。
 */
export interface ProductBlueprintCategorySnapshot {
  id: string;
  code: string;
  nameJa: string;
  nameEn: string;
  parentId?: string | null;
  kind: ProductBlueprintCategoryKind;
  path: string[];
  displayOrder?: number;
}

/**
 * backend/internal/domain/productBlueprintCategory.InputFieldScope に対応。
 */
export type CategoryInputFieldScope = "productBlueprint" | "model";

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
 * washTags を保存用に正規化する。
 *
 * - 文字列以外を除外
 * - 空文字を除外
 * - 前後の空白を除去
 * - 重複を除外
 */
export function normalizeWashTags(
  value: unknown,
): WashTags {
  if (!Array.isArray(value)) {
    return [];
  }

  const normalized = value
    .filter(
      (item): item is string =>
        typeof item === "string",
    )
    .map((item) => item.trim())
    .filter((item) => item !== "");

  return [...new Set(normalized)];
}

/**
 * apparel の必須 categoryFields を検証する。
 */
export function validateApparelCategoryFields(
  fields: CategoryFieldValues | null | undefined,
): string[] {
  const errors: string[] = [];

  if (!fields) {
    return ["categoryFields is required for apparel"];
  }

  if (!isValidWashTags(fields.washTags)) {
    errors.push(
      "categoryFields.washTags must contain at least one value",
    );
  }

  return errors;
}

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

export function validateProductBlueprintCategorySnapshot(
  category:
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): string[] {
  const errors: string[] = [];

  if (!category) {
    return ["productBlueprintCategory is required"];
  }

  if (!category.id?.trim()) {
    errors.push(
      "productBlueprintCategory.id is required",
    );
  }

  if (!category.code?.trim()) {
    errors.push(
      "productBlueprintCategory.code is required",
    );
  }

  if (!category.nameJa?.trim()) {
    errors.push(
      "productBlueprintCategory.nameJa is required",
    );
  }

  if (
    !isValidProductBlueprintCategoryKind(
      category.kind,
    )
  ) {
    errors.push(
      "productBlueprintCategory.kind is invalid",
    );
  }

  return errors;
}

export function toProductBlueprintCategorySnapshot(
  category: ProductBlueprintCategory,
): ProductBlueprintCategorySnapshot {
  return {
    id: category.id,
    code: category.code,
    nameJa: category.nameJa,
    nameEn: category.nameEn,
    parentId: category.parentId ?? null,
    kind: category.kind,
    path: [...category.path],
    displayOrder: category.displayOrder,
  };
}

export function getProductBlueprintCategoryDisplayName(
  category:
    | ProductBlueprintCategory
    | ProductBlueprintCategorySnapshot,
): string {
  return (
    category.nameJa ||
    category.nameEn ||
    category.code
  );
}

export function isApparelProductBlueprintCategory(
  category:
    | ProductBlueprintCategory
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): boolean {
  return category?.kind === "apparel";
}

export function isAlcoholProductBlueprintCategory(
  category:
    | ProductBlueprintCategory
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): boolean {
  return category?.kind === "alcohol";
}

export function isCosmeticsProductBlueprintCategory(
  category:
    | ProductBlueprintCategory
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): boolean {
  return category?.kind === "cosmetics";
}

export function isHealthcareProductBlueprintCategory(
  category:
    | ProductBlueprintCategory
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): boolean {
  return category?.kind === "healthcare";
}

export function isOtherProductBlueprintCategory(
  category:
    | ProductBlueprintCategory
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): boolean {
  return category?.kind === "other";
}