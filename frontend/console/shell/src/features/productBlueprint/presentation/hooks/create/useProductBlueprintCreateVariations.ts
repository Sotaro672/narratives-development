// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateVariations.ts

import type {
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

import {
  useProductBlueprintVariations,
  type UseProductBlueprintVariationsResult,
} from "../shared/useProductBlueprintVariations";

export type UseProductBlueprintCreateVariationsResult = Omit<
  UseProductBlueprintVariationsResult,
  | "categoryCode"
  | "getCode"
  | "setFromUiState"
>;

export function useProductBlueprintCreateVariations(
  productBlueprintCategory:
    | ProductBlueprintCategorySnapshot
    | null,
): UseProductBlueprintCreateVariationsResult {
  return useProductBlueprintVariations({
    productBlueprintCategory,
  });
}