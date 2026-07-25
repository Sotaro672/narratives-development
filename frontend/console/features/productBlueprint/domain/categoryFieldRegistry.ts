// frontend/console/productBlueprint/src/domain/entity/categoryFieldRegistry.ts

import {
  getAlcoholCategoryFieldKeys,
  isAlcoholCategoryCode,
  type AlcoholCategoryFieldKey,
} from "./alcohol";
import {
  getApparelCategoryFieldKeys,
  getApparelModelFieldKeys,
  isApparelCategoryCode,
  type ApparelCategoryFieldKey,
  type ApparelModelFieldKey,
} from "./apparel";
import {
  getCosmeticsCategoryFieldKeys,
  isCosmeticsCategoryCode,
  type CosmeticsCategoryFieldKey,
} from "./cosmetics";
import {
  getHealthcareCategoryFieldKeys,
  isHealthcareCategoryCode,
  type HealthcareCategoryFieldKey,
} from "./healthcare";
import {
  getOtherCategoryFieldKeys,
  isOtherCategoryCode,
  type OtherCategoryFieldKey,
} from "./other";

/**
 * productBlueprint.CategoryFields に保存するカテゴリ別入力値。
 *
 * 注意:
 * - brandId / productName / productIdTagType / description は共通 field。
 * - color / size / measurements は model variation 側。
 * - 上記は categoryFields には含めない。
 */
export type CategoryFieldPrimitiveValue = string | number | boolean | null;

export type CategoryFieldValue =
  | CategoryFieldPrimitiveValue
  | CategoryFieldPrimitiveValue[]
  | Record<string, CategoryFieldPrimitiveValue>;

export type CategoryFieldValues = Record<string, CategoryFieldValue>;

export type ProductBlueprintCategoryFieldKey =
  | AlcoholCategoryFieldKey
  | ApparelCategoryFieldKey
  | CosmeticsCategoryFieldKey
  | HealthcareCategoryFieldKey
  | OtherCategoryFieldKey;

export type ModelCategoryFieldKey = ApparelModelFieldKey;

export const COMMON_PRODUCT_BLUEPRINT_FIELD_KEYS = [
  "brandId",
  "productName",
  "productIdTagType",
  "description",
] as const;

export type CommonProductBlueprintFieldKey =
  (typeof COMMON_PRODUCT_BLUEPRINT_FIELD_KEYS)[number];

export function isCommonProductBlueprintFieldKey(
  key: string,
): key is CommonProductBlueprintFieldKey {
  return COMMON_PRODUCT_BLUEPRINT_FIELD_KEYS.some(
    (fieldKey) => fieldKey === key,
  );
}

export function getProductBlueprintCategoryFieldKeys(
  categoryCode: string,
): ProductBlueprintCategoryFieldKey[] {
  if (isAlcoholCategoryCode(categoryCode)) {
    return getAlcoholCategoryFieldKeys(categoryCode);
  }

  if (isApparelCategoryCode(categoryCode)) {
    return getApparelCategoryFieldKeys(categoryCode);
  }

  if (isCosmeticsCategoryCode(categoryCode)) {
    return getCosmeticsCategoryFieldKeys(categoryCode);
  }

  if (isHealthcareCategoryCode(categoryCode)) {
    return getHealthcareCategoryFieldKeys(categoryCode);
  }

  if (isOtherCategoryCode(categoryCode)) {
    return getOtherCategoryFieldKeys(categoryCode);
  }

  return [];
}

export function getModelCategoryFieldKeys(
  categoryCode: string,
): ModelCategoryFieldKey[] {
  if (isApparelCategoryCode(categoryCode)) {
    return getApparelModelFieldKeys(categoryCode);
  }

  return [];
}

export function hasModelCategoryFields(categoryCode: string): boolean {
  return getModelCategoryFieldKeys(categoryCode).length > 0;
}

export function hasModelMeasurements(categoryCode: string): boolean {
  return getModelCategoryFieldKeys(categoryCode).includes("measurements");
}

function normalizeCategoryFieldValue(
  value: unknown,
): CategoryFieldValue | undefined {
  if (value === undefined) {
    return undefined;
  }

  if (value === "") {
    return null;
  }

  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean" ||
    value === null
  ) {
    return value;
  }

  if (Array.isArray(value)) {
    const normalized = value.filter(
      (item): item is CategoryFieldPrimitiveValue =>
        typeof item === "string" ||
        typeof item === "number" ||
        typeof item === "boolean" ||
        item === null,
    );

    return normalized;
  }

  if (typeof value === "object" && value !== null) {
    const out: Record<string, CategoryFieldPrimitiveValue> = {};

    for (const [objectKey, objectValue] of Object.entries(value)) {
      if (
        typeof objectValue === "string" ||
        typeof objectValue === "number" ||
        typeof objectValue === "boolean" ||
        objectValue === null
      ) {
        out[objectKey] = objectValue;
      }
    }

    return out;
  }

  return undefined;
}

export function pickCategoryFields(
  categoryCode: string,
  values: Record<string, unknown>,
): CategoryFieldValues {
  const keys = getProductBlueprintCategoryFieldKeys(categoryCode);
  const out: CategoryFieldValues = {};

  for (const key of keys) {
    const normalizedValue = normalizeCategoryFieldValue(values[key]);

    if (normalizedValue === undefined) {
      continue;
    }

    out[key] = normalizedValue;
  }

  return out;
}

export function omitCommonAndModelFields(
  categoryCode: string,
  values: Record<string, unknown>,
): CategoryFieldValues {
  const productBlueprintKeys =
    getProductBlueprintCategoryFieldKeys(categoryCode);
  const modelKeys = getModelCategoryFieldKeys(categoryCode);

  const allowedKeys = new Set<string>(productBlueprintKeys);
  const excludedKeys = new Set<string>([
    ...COMMON_PRODUCT_BLUEPRINT_FIELD_KEYS,
    ...modelKeys,
  ]);

  const out: CategoryFieldValues = {};

  for (const [key, value] of Object.entries(values)) {
    if (excludedKeys.has(key)) {
      continue;
    }

    if (!allowedKeys.has(key)) {
      continue;
    }

    const normalizedValue = normalizeCategoryFieldValue(value);

    if (normalizedValue === undefined) {
      continue;
    }

    out[key] = normalizedValue;
  }

  return out;
}