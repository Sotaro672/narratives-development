// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateCategory.ts

import * as React from "react";

import { listProductBlueprintCategoriesApi } from "../../../infrastructure/api/productBlueprintApi";

import {
  toProductBlueprintCategorySnapshot,
  type ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

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
  const [
    productBlueprintCategory,
    setProductBlueprintCategory,
  ] =
    React.useState<ProductBlueprintCategorySnapshot | null>(
      null,
    );

  const [
    productBlueprintCategoryOptions,
    setProductBlueprintCategoryOptions,
  ] = React.useState<
    ProductBlueprintCategorySnapshot[]
  >([]);

  const [
    productBlueprintCategoryLoading,
    setProductBlueprintCategoryLoading,
  ] = React.useState(false);

  const [
    productBlueprintCategoryError,
    setProductBlueprintCategoryError,
  ] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadCategories() {
      setProductBlueprintCategoryLoading(true);
      setProductBlueprintCategoryError(null);

      try {
        const categories =
          await listProductBlueprintCategoriesApi();

        const snapshots = categories.map(
          toProductBlueprintCategorySnapshot,
        );

        if (!cancelled) {
          setProductBlueprintCategoryOptions(
            snapshots,
          );
        }
      } catch (error) {
        const err =
          error instanceof Error
            ? error
            : new Error(String(error));

        if (!cancelled) {
          setProductBlueprintCategoryError(err);
        }
      } finally {
        if (!cancelled) {
          setProductBlueprintCategoryLoading(
            false,
          );
        }
      }
    }

    void loadCategories();

    return () => {
      cancelled = true;
    };
  }, []);

  const productBlueprintCategoryId =
    React.useMemo(
      () =>
        productBlueprintCategory?.id ?? "",
      [productBlueprintCategory],
    );

  const productBlueprintCategoryLabel =
    React.useMemo(
      () =>
        getCategoryLabel(
          productBlueprintCategory,
        ),
      [productBlueprintCategory],
    );

  return {
    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    productBlueprintCategoryOptions,
    productBlueprintCategoryLoading,
    productBlueprintCategoryError,
    onChangeProductBlueprintCategory:
      setProductBlueprintCategory,
  };
}