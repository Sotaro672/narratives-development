// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateCategoryFields.ts

import * as React from "react";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

import { getProductBlueprintCategoryFieldKeys } from "../../../domain/categoryFieldRegistry";

import {
  FIT_OPTIONS,
  type Fit,
} from "../../../domain/apparel";

export type FitInputValue = Fit | "";

export type UseProductBlueprintCreateCategoryFieldsResult = {
  fit: FitInputValue;
  material: string;
  weight: number;
  qualityAssurance: string[];
  categoryFields: CategoryFieldValues;
  onChangeFit: (value: Fit) => void;
  onChangeMaterial: (value: string) => void;
  onChangeWeight: (value: number) => void;
  onChangeQualityAssurance: (
    value: string[],
  ) => void;
  onChangeCategoryField: (
    key: string,
    value: CategoryFieldValue,
  ) => void;
  resetCategoryFields: () => void;
};

const FIT_VALUE_SET: ReadonlySet<string> =
  new Set(
    FIT_OPTIONS.map(
      (option) => option.value,
    ),
  );

/**
 * Fitとして有効な値か判定する。
 */
function isFit(value: unknown): value is Fit {
  return (
    typeof value === "string" &&
    FIT_VALUE_SET.has(value)
  );
}

/**
 * model domain管轄のfieldか判定する。
 *
 * alcoholのvolumeはmodel variation側へ保存する。
 * cosmeticsのvolumeはProductBlueprint.categoryFieldsへ保存する。
 */
function isModelOwnedCategoryFieldKey(
  categoryCode: string,
  key: string,
): boolean {
  return (
    categoryCode.startsWith("alcohol.") &&
    key === "volume"
  );
}

function normalizeNumberValue(
  value: number,
): number {
  if (Number.isNaN(value)) {
    return 0;
  }

  return value < 0 ? 0 : value;
}

function normalizeStringArrayValue(
  value: unknown,
): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(
    (item): item is string =>
      typeof item === "string" &&
      item.trim() !== "",
  );
}

function normalizeCategoryFieldsForCategory(
  category:
    | ProductBlueprintCategorySnapshot
    | null,
  fields: CategoryFieldValues,
): CategoryFieldValues {
  const categoryCode = String(
    category?.code ?? "",
  ).trim();

  if (!categoryCode) {
    return {};
  }

  const allowedKeys = new Set<string>(
    getProductBlueprintCategoryFieldKeys(
      categoryCode,
    ),
  );

  const next: CategoryFieldValues = {};

  for (const [key, value] of Object.entries(
    fields,
  )) {
    if (
      isModelOwnedCategoryFieldKey(
        categoryCode,
        key,
      )
    ) {
      continue;
    }

    if (!allowedKeys.has(key)) {
      continue;
    }

    next[key] = value;
  }

  return next;
}

export function useProductBlueprintCreateCategoryFields(
  productBlueprintCategory:
    | ProductBlueprintCategorySnapshot
    | null,
): UseProductBlueprintCreateCategoryFieldsResult {
  const categoryCode = React.useMemo(
    () =>
      String(
        productBlueprintCategory?.code ?? "",
      ).trim(),
    [productBlueprintCategory?.code],
  );

  const [fit, setFit] =
    React.useState<FitInputValue>("");

  const [material, setMaterial] =
    React.useState("");

  const [weight, setWeight] =
    React.useState<number>(0);

  const [
    qualityAssurance,
    setQualityAssurance,
  ] = React.useState<string[]>([]);

  const [
    categoryFields,
    setCategoryFields,
  ] = React.useState<CategoryFieldValues>(
    {},
  );

  React.useEffect(() => {
    setCategoryFields((previous) =>
      normalizeCategoryFieldsForCategory(
        productBlueprintCategory,
        previous,
      ),
    );
  }, [productBlueprintCategory]);

  const onChangeFit = React.useCallback(
    (value: Fit) => {
      setFit(value);

      setCategoryFields((previous) => ({
        ...previous,
        fit: value,
      }));
    },
    [],
  );

  const onChangeMaterial =
    React.useCallback((value: string) => {
      setMaterial(value);

      setCategoryFields((previous) => ({
        ...previous,
        material:
          value.trim() === ""
            ? null
            : value,
      }));
    }, []);

  const onChangeWeight =
    React.useCallback((value: number) => {
      const next =
        normalizeNumberValue(value);

      setWeight(next);

      setCategoryFields((previous) => ({
        ...previous,
        weight: next,
      }));
    }, []);

  const onChangeQualityAssurance =
    React.useCallback(
      (value: string[]) => {
        const next =
          normalizeStringArrayValue(
            value,
          );

        setQualityAssurance(next);

        setCategoryFields(
          (previous) => ({
            ...previous,
            washTags: next,
          }),
        );
      },
      [],
    );

  const onChangeCategoryField =
    React.useCallback(
      (
        key: string,
        value: CategoryFieldValue,
      ) => {
        if (
          isModelOwnedCategoryFieldKey(
            categoryCode,
            key,
          )
        ) {
          setCategoryFields(
            (previous) => {
              const next = {
                ...previous,
              };

              delete next[key];

              return next;
            },
          );

          return;
        }

        if (key === "fit") {
          const nextFit = isFit(value)
            ? value
            : "";

          setFit(nextFit);

          setCategoryFields(
            (previous) => {
              const next = {
                ...previous,
              };

              if (nextFit) {
                next.fit = nextFit;
              } else {
                delete next.fit;
              }

              return next;
            },
          );

          return;
        }

        if (key === "material") {
          const nextMaterial =
            typeof value === "string"
              ? value
              : "";

          setMaterial(nextMaterial);

          setCategoryFields(
            (previous) => ({
              ...previous,
              material:
                nextMaterial.trim() === ""
                  ? null
                  : nextMaterial,
            }),
          );

          return;
        }

        if (key === "weight") {
          const nextWeight =
            typeof value === "number"
              ? normalizeNumberValue(
                  value,
                )
              : 0;

          setWeight(nextWeight);

          setCategoryFields(
            (previous) => ({
              ...previous,
              weight: nextWeight,
            }),
          );

          return;
        }

        if (
          key === "washTags" ||
          key === "qualityAssurance"
        ) {
          const nextWashTags =
            normalizeStringArrayValue(
              value,
            );

          setQualityAssurance(
            nextWashTags,
          );

          setCategoryFields(
            (previous) => ({
              ...previous,
              washTags: nextWashTags,
            }),
          );

          return;
        }

        setCategoryFields(
          (previous) => ({
            ...previous,
            [key]: value,
          }),
        );
      },
      [categoryCode],
    );

  const resetCategoryFields =
    React.useCallback(() => {
      setFit("");
      setMaterial("");
      setWeight(0);
      setQualityAssurance([]);
      setCategoryFields({});
    }, []);

  return {
    fit,
    material,
    weight,
    qualityAssurance,
    categoryFields,
    onChangeFit,
    onChangeMaterial,
    onChangeWeight,
    onChangeQualityAssurance,
    onChangeCategoryField,
    resetCategoryFields,
  };
}