// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateCategory.ts

import * as React from "react";

import {
  APPAREL_CATEGORY_OPTIONS,
} from "../../../../../shared/types/apparel";

import {
  ALCOHOL_CATEGORY_OPTIONS,
} from "../../../domain/alcohol";

import {
  COSMETICS_CATEGORY_OPTIONS,
} from "../../../domain/cosmetics";

import {
  HEALTHCARE_CATEGORY_OPTIONS,
} from "../../../domain/healthcare";

import {
  OTHER_CATEGORY_OPTIONS,
} from "../../../domain/other";

import {
  toProductBlueprintCategoryPathKey,
  type ProductBlueprintCategoryPath,
} from "../../../domain/productBlueprintCategory";

import { useProductBlueprintCategoryOptions } from "../shared/useProductBlueprintCategoryOptions";

const CATEGORY_LABEL_BY_PATH_KEY: Readonly<
  Record<string, string>
> = Object.fromEntries(
  [
    ...APPAREL_CATEGORY_OPTIONS,
    ...ALCOHOL_CATEGORY_OPTIONS,
    ...COSMETICS_CATEGORY_OPTIONS,
    ...HEALTHCARE_CATEGORY_OPTIONS,
    ...OTHER_CATEGORY_OPTIONS,
  ].map(
    (option) => [
      option.value,
      option.label,
    ],
  ),
);

function getCategoryLabel(
  productBlueprintCategoryPath: ProductBlueprintCategoryPath | null,
): string {

  if (
    !productBlueprintCategoryPath ||
    productBlueprintCategoryPath.length === 0
  ) {

    return "";

  }

  const pathKey =
    toProductBlueprintCategoryPathKey(
      productBlueprintCategoryPath,
    );

  return (
    CATEGORY_LABEL_BY_PATH_KEY[pathKey] ??
    productBlueprintCategoryPath[
      productBlueprintCategoryPath.length - 1
    ] ??
    ""
  );

}

export type UseProductBlueprintCreateCategoryResult = {

  productBlueprintCategoryPath: ProductBlueprintCategoryPath | null;

  productBlueprintCategoryLabel: string;

  productBlueprintCategoryOptions: ProductBlueprintCategoryPath[];

  productBlueprintCategoryLoading: boolean;

  productBlueprintCategoryError: Error | null;

  onChangeProductBlueprintCategoryPath: (
    productBlueprintCategoryPath: ProductBlueprintCategoryPath | null,
  ) => void;

};

export function useProductBlueprintCreateCategory(): UseProductBlueprintCreateCategoryResult {

  const [
    productBlueprintCategoryPath,
    setProductBlueprintCategoryPath,
  ] =
    React.useState<ProductBlueprintCategoryPath | null>(null);

  const {

    productBlueprintCategoryOptions,

    productBlueprintCategoryLoading,

    productBlueprintCategoryError,

  } = useProductBlueprintCategoryOptions();

  const productBlueprintCategoryLabel = React.useMemo(

    () =>
      getCategoryLabel(
        productBlueprintCategoryPath,
      ),

    [productBlueprintCategoryPath],

  );

  const onChangeProductBlueprintCategoryPath =
    React.useCallback(
      (
        nextProductBlueprintCategoryPath:
          ProductBlueprintCategoryPath | null,
      ) => {

        setProductBlueprintCategoryPath(
          nextProductBlueprintCategoryPath
            ? [
                ...nextProductBlueprintCategoryPath,
              ]
            : null,
        );

      },
      [],
    );

  return {

    productBlueprintCategoryPath,

    productBlueprintCategoryLabel,

    productBlueprintCategoryOptions,

    productBlueprintCategoryLoading,

    productBlueprintCategoryError,

    onChangeProductBlueprintCategoryPath,

  };

}