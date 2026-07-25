// frontend/console/productBlueprint/src/domain/entity/productBlueprintCategory.ts

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
 */
export type CategoryFieldPrimitiveValue = string | number | boolean | null;

export type CategoryFieldValue =
  | CategoryFieldPrimitiveValue
  | CategoryFieldPrimitiveValue[]
  | Record<string, CategoryFieldPrimitiveValue>;

export type CategoryFieldValues = Record<string, CategoryFieldValue>;

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
  category: ProductBlueprintCategorySnapshot | null | undefined,
): string[] {
  const errors: string[] = [];

  if (!category) {
    return ["productBlueprintCategory is required"];
  }

  if (!category.id?.trim()) {
    errors.push("productBlueprintCategory.id is required");
  }

  if (!category.code?.trim()) {
    errors.push("productBlueprintCategory.code is required");
  }

  if (!category.nameJa?.trim()) {
    errors.push("productBlueprintCategory.nameJa is required");
  }

  if (!isValidProductBlueprintCategoryKind(category.kind)) {
    errors.push("productBlueprintCategory.kind is invalid");
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
  category: ProductBlueprintCategory | ProductBlueprintCategorySnapshot,
): string {
  return category.nameJa || category.nameEn || category.code;
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