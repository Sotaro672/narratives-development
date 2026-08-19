// frontend/console/shell/src/features/productBlueprint/presentation/hooks/shared/useProductBlueprintCategoryOptions.ts

import * as React from "react";
import { listProductBlueprintCategoryPaths } from "../../../application/productBlueprintCategoryService";
import type { ProductBlueprintCategoryPath } from "../../../domain/productBlueprintCategory";

export type UseProductBlueprintCategoryOptionsResult = {
  productBlueprintCategoryOptions: ProductBlueprintCategoryPath[];
  productBlueprintCategoryLoading: boolean;
  productBlueprintCategoryError: Error | null;
};

export function useProductBlueprintCategoryOptions(): UseProductBlueprintCategoryOptionsResult {
  const [productBlueprintCategoryOptions, setProductBlueprintCategoryOptions] =
    React.useState<ProductBlueprintCategoryPath[]>([]);
  const [productBlueprintCategoryLoading, setProductBlueprintCategoryLoading] = React.useState(false);
  const [productBlueprintCategoryError, setProductBlueprintCategoryError] =
    React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    const loadProductBlueprintCategoryOptions = async () => {
      try {
        setProductBlueprintCategoryLoading(true);
        setProductBlueprintCategoryError(null);

        const options = await listProductBlueprintCategoryPaths();

        if (!cancelled) {
          setProductBlueprintCategoryOptions(options);
        }
      } catch (error: unknown) {
        if (!cancelled) {
          const normalizedError = error instanceof Error ? error : new Error(String(error));
          setProductBlueprintCategoryError(normalizedError);
          setProductBlueprintCategoryOptions([]);
        }
      } finally {
        if (!cancelled) {
          setProductBlueprintCategoryLoading(false);
        }
      }
    };

    void loadProductBlueprintCategoryOptions();

    return () => {
      cancelled = true;
    };
  }, []);

  return {
    productBlueprintCategoryOptions,
    productBlueprintCategoryLoading,
    productBlueprintCategoryError,
  };
}