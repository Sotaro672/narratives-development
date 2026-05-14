// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreateCategory.ts

import * as React from "react";

import {
  listProductBlueprintCategoriesApi,
} from "../../infrastructure/api/productBlueprintCategoryApi";

import type {
  ProductBlueprintCategory,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

function toProductBlueprintCategorySnapshot(
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

function getCategoryLabel(
  category: ProductBlueprintCategorySnapshot | null,
): string {
  if (!category) {
    return "";
  }

  return (
    category.nameJa ||
    category.nameEn ||
    category.code ||
    category.id ||
    ""
  );
}

export type UseProductBlueprintCreateCategoryResult = {
  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryLabel: string;
  productBlueprintCategoryOptions: ProductBlueprintCategorySnapshot[];
  productBlueprintCategoryLoading: boolean;
  productBlueprintCategoryError: Error | null;
  onChangeProductBlueprintCategory: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;
};

export function useProductBlueprintCreateCategory(): UseProductBlueprintCreateCategoryResult {
  const [productBlueprintCategory, setProductBlueprintCategory] =
    React.useState<ProductBlueprintCategorySnapshot | null>(null);

  const [productBlueprintCategoryOptions, setProductBlueprintCategoryOptions] =
    React.useState<ProductBlueprintCategorySnapshot[]>([]);

  const [productBlueprintCategoryLoading, setProductBlueprintCategoryLoading] =
    React.useState(false);

  const [productBlueprintCategoryError, setProductBlueprintCategoryError] =
    React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadCategories() {
      setProductBlueprintCategoryLoading(true);
      setProductBlueprintCategoryError(null);

      try {
        const categories = await listProductBlueprintCategoriesApi();
        const snapshots = categories.map(toProductBlueprintCategorySnapshot);

        if (!cancelled) {
          setProductBlueprintCategoryOptions(snapshots);
        }
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));

        if (!cancelled) {
          setProductBlueprintCategoryError(err);
        }
      } finally {
        if (!cancelled) {
          setProductBlueprintCategoryLoading(false);
        }
      }
    }

    void loadCategories();

    return () => {
      cancelled = true;
    };
  }, []);

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