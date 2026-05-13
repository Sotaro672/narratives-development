// frontend/productBlueprint/src/domain/entity/productBlueprintCategory.ts

/**
 * backend/internal/domain/common.ProductCategoryKind に対応。
 */
export type ProductBlueprintCategoryKind =
  | "apparel"
  | "food"
  | "alcohol"
  | "cosmetics"
  | "goods"
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
 */
export interface ProductBlueprintCategorySnapshot {
  id: string;
  code: string;
  nameJa: string;
  nameEn: string;
  kind: ProductBlueprintCategoryKind;
  path: string[];
}

export type CategoryFieldValue = string | number | boolean | null;

export type CategoryFieldValues = Record<string, CategoryFieldValue>;

export function isValidProductBlueprintCategoryKind(
  value: string,
): value is ProductBlueprintCategoryKind {
  return (
    value === "apparel" ||
    value === "food" ||
    value === "alcohol" ||
    value === "cosmetics" ||
    value === "goods" ||
    value === "healthcare" ||
    value === "other"
  );
}

export function validateProductBlueprintCategorySnapshot(
  category: ProductBlueprintCategorySnapshot,
): string[] {
  const errors: string[] = [];

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
  if (!Array.isArray(category.path) || category.path.length === 0) {
    errors.push("productBlueprintCategory.path is required");
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
    kind: category.kind,
    path: [...category.path],
  };
}