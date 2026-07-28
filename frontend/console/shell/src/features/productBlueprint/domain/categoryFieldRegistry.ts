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
 * カテゴリコードに対応する
 * ProductBlueprint.categoryFieldsのkey一覧を返す。
 */
export function getProductBlueprintCategoryFieldKeys(
  categoryCode: string,
): ProductBlueprintCategoryFieldKey[] {
  if (
    isAlcoholCategoryCode(
      categoryCode,
    )
  ) {
    return getAlcoholCategoryFieldKeys(
      categoryCode,
    );
  }

  if (
    isApparelCategoryCode(
      categoryCode,
    )
  ) {
    return getApparelCategoryFieldKeys(
      categoryCode,
    );
  }

  if (
    isCosmeticsCategoryCode(
      categoryCode,
    )
  ) {
    return getCosmeticsCategoryFieldKeys(
      categoryCode,
    );
  }

  if (
    isHealthcareCategoryCode(
      categoryCode,
    )
  ) {
    return getHealthcareCategoryFieldKeys(
      categoryCode,
    );
  }

  if (
    isOtherCategoryCode(
      categoryCode,
    )
  ) {
    return getOtherCategoryFieldKeys(
      categoryCode,
    );
  }

  return [];
}