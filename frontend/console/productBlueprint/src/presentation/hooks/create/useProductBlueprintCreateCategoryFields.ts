// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreateCategoryFields.ts

import * as React from "react";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/entity/productBlueprintCategory";

import {
  getProductBlueprintCategoryFieldKeys,
} from "../../../domain/entity/categoryFieldRegistry";

import type { Fit } from "../../../domain/entity/apparel";

export type UseProductBlueprintCreateCategoryFieldsResult = {
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];
  categoryFields: CategoryFieldValues;
  onChangeFit: (value: Fit) => void;
  onChangeMaterial: (value: string) => void;
  onChangeWeight: (value: number) => void;
  onChangeQualityAssurance: (value: string[]) => void;
  onChangeCategoryField: (key: string, value: CategoryFieldValue) => void;
  resetCategoryFields: () => void;
};

function normalizeNumberValue(value: number): number {
  if (Number.isNaN(value)) {
    return 0;
  }

  return value < 0 ? 0 : value;
}

function normalizeCategoryFieldsForCategory(
  category: ProductBlueprintCategorySnapshot | null,
  fields: CategoryFieldValues,
): CategoryFieldValues {
  const categoryCode = String(category?.code ?? "").trim();

  if (!categoryCode) {
    return {};
  }

  const allowedKeys = new Set<string>(
    getProductBlueprintCategoryFieldKeys(categoryCode),
  );

  const next: CategoryFieldValues = {};

  for (const [key, value] of Object.entries(fields)) {
    if (!allowedKeys.has(key)) {
      continue;
    }

    next[key] = value;
  }

  return next;
}

export function useProductBlueprintCreateCategoryFields(
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null,
): UseProductBlueprintCreateCategoryFieldsResult {
  const [fit, setFit] = React.useState<Fit>("" as Fit);
  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);
  const [categoryFields, setCategoryFields] =
    React.useState<CategoryFieldValues>({});

  React.useEffect(() => {
    setCategoryFields((prev) =>
      normalizeCategoryFieldsForCategory(productBlueprintCategory, prev),
    );
  }, [productBlueprintCategory]);

  const onChangeFit = React.useCallback((value: Fit) => {
    setFit(value);

    setCategoryFields((prev) => ({
      ...prev,
      fit: value,
    }));
  }, []);

  const onChangeMaterial = React.useCallback((value: string) => {
    setMaterial(value);

    setCategoryFields((prev) => ({
      ...prev,
      material: value.trim() === "" ? null : value,
    }));
  }, []);

  const onChangeWeight = React.useCallback((value: number) => {
    const next = normalizeNumberValue(value);
    setWeight(next);

    setCategoryFields((prev) => ({
      ...prev,
      weight: next,
    }));
  }, []);

  const onChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      setCategoryFields((prev) => ({
        ...prev,
        [key]: value,
      }));

      if (key === "fit" && typeof value === "string") {
        setFit(value as Fit);
      }

      if (key === "material" && typeof value === "string") {
        setMaterial(value);
      }

      if (key === "weight" && typeof value === "number") {
        setWeight(normalizeNumberValue(value));
      }
    },
    [],
  );

  const resetCategoryFields = React.useCallback(() => {
    setFit("" as Fit);
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
    onChangeQualityAssurance: setQualityAssurance,
    onChangeCategoryField,
    resetCategoryFields,
  };
}