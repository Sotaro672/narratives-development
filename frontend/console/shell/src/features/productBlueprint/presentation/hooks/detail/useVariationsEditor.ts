// frontend/console/shell/src/features/productBlueprint/presentation/hooks/detail/useVariationsEditor.ts

import type {
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

import {
  useProductBlueprintVariations,
  type ProductBlueprintVariationsState,
  type UseProductBlueprintVariationsResult,
} from "../shared/useProductBlueprintVariations";

/**
 * ModelVariation一覧から生成されたUI state。
 *
 * 作成画面・詳細画面で共通のVariation state型を使用する。
 */
export type VariationsUiState =
  ProductBlueprintVariationsState;

/**
 * 詳細画面で使用するVariation editorの返却型。
 *
 * 以下は共有Hook内部でのみ使用するため、
 * 詳細画面の公開インターフェースから除外する。
 *
 * - categoryCode
 * - isApparelCategory
 * - isAlcoholCategory
 * - measurementOptions
 * - resetVariations
 */
export type UseVariationsEditorResult = Omit<
  UseProductBlueprintVariationsResult,
  | "categoryCode"
  | "isApparelCategory"
  | "isAlcoholCategory"
  | "measurementOptions"
  | "resetVariations"
>;

/**
 * 商品設計詳細画面のVariation編集Hook。
 *
 * stateおよび操作処理は
 * useProductBlueprintVariationsへ集約する。
 */
export function useVariationsEditor(
  productBlueprintCategory:
    | ProductBlueprintCategorySnapshot
    | null,
  initial?: Partial<VariationsUiState>,
): UseVariationsEditorResult {
  return useProductBlueprintVariations({
    productBlueprintCategory,
    initialState: initial,
  });
}