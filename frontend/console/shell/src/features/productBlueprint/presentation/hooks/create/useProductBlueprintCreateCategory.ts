// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateCategory.ts
import * as React from "react";
import type { ProductBlueprintCategorySnapshot } from "../../../domain/productBlueprintCategory";
import { useProductBlueprintCategoryOptions } from "../shared/useProductBlueprintCategoryOptions";

function getCategoryLabel(category: ProductBlueprintCategorySnapshot | null): string {
  if (!category) {
    return "";
  }
  return category.nameJa || category.nameEn || category.code || category.id || "";
}

export type UseProductBlueprintCreateCategoryResult = {
  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryLabel: string;
  productBlueprintCategoryOptions: ProductBlueprintCategorySnapshot[];
  productBlueprintCategoryLoading: boolean;
  productBlueprintCategoryError: Error | null;
  onChangeProductBlueprintCategory: (category: ProductBlueprintCategorySnapshot | null) => void;
};

export function useProductBlueprintCreateCategory(): UseProductBlueprintCreateCategoryResult {
  const [productBlueprintCategory, setProductBlueprintCategory] =
    React.useState<ProductBlueprintCategorySnapshot | null>(null);

  const {
    productBlueprintCategoryOptions,
    productBlueprintCategoryLoading,
    productBlueprintCategoryError,
  } = useProductBlueprintCategoryOptions();

  const productBlueprintCategoryId = React.useMemo(
    () => productBlueprintCategory?.id ?? "",
    [productBlueprintCategory],
  );

  const productBlueprintCategoryLabel = React.useMemo(
    () => getCategoryLabel(productBlueprintCategory),
    [productBlueprintCategory],
  );

  return {
    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    productBlueprintCategoryOptions,
    productBlueprintCategoryLoading,
    productBlueprintCategoryError,
    onChangeProductBlueprintCategory: setProductBlueprintCategory,
  };
}