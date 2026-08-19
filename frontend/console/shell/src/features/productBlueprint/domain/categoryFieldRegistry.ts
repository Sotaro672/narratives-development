// frontend/console/shell/src/features/productBlueprint/domain/categoryFieldRegistry.ts

import {
  getAlcoholCategoryFieldKeys,
  isAlcoholCategoryCode,
  type AlcoholCategoryFieldKey,
} from "./alcohol";

import {
  getApparelCategoryFieldKeys,
  isApparelCategoryCode,
  type ApparelCategoryFieldKey,
} from "../../../shared/types/apparel";

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

import {
  toProductBlueprintCategoryPathKey,
  type ProductBlueprintCategoryPath,
} from "./productBlueprintCategory";

/**
 * ProductBlueprint.categoryFieldsに保存するカテゴリ別入力値。
 *
 * 注意:
 * - brandId / productName / productIdTagType / descriptionは共通field。
 * - color / size / measurementsはmodel variation側。
 * - 上記はcategoryFieldsには含めない。
 */
export type CategoryFieldPrimitiveValue =
  | string
  | number
  | boolean
  | null;

export type CategoryFieldValue =
  | CategoryFieldPrimitiveValue
  | CategoryFieldPrimitiveValue[]
  | Record<
      string,
      CategoryFieldPrimitiveValue
    >;

export type CategoryFieldValues = Record<
  string,
  CategoryFieldValue
>;

export type ProductBlueprintCategoryFieldKey =
  | AlcoholCategoryFieldKey
  | ApparelCategoryFieldKey
  | CosmeticsCategoryFieldKey
  | HealthcareCategoryFieldKey
  | OtherCategoryFieldKey;

/**
 * productBlueprintCategoryPathに対応する
 * ProductBlueprint.categoryFieldsのkey一覧を返す。
 */
export function getProductBlueprintCategoryFieldKeys(
  productBlueprintCategoryPath: ProductBlueprintCategoryPath,
): ProductBlueprintCategoryFieldKey[] {
  const pathKey =
    toProductBlueprintCategoryPathKey(
      productBlueprintCategoryPath,
    );

  if (
    isAlcoholCategoryCode(
      pathKey,
    )
  ) {
    return getAlcoholCategoryFieldKeys(
      pathKey,
    );
  }

  if (
    isApparelCategoryCode(
      pathKey,
    )
  ) {
    return getApparelCategoryFieldKeys(
      pathKey,
    );
  }

  if (
    isCosmeticsCategoryCode(
      pathKey,
    )
  ) {
    return getCosmeticsCategoryFieldKeys(
      pathKey,
    );
  }

  if (
    isHealthcareCategoryCode(
      pathKey,
    )
  ) {
    return getHealthcareCategoryFieldKeys(
      pathKey,
    );
  }

  if (
    isOtherCategoryCode(
      pathKey,
    )
  ) {
    return getOtherCategoryFieldKeys(
      pathKey,
    );
  }

  return [];
}