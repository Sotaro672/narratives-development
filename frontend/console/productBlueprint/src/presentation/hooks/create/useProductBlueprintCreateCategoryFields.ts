// frontend/console/productBlueprint/src/presentation/hooks/create/useProductBlueprintCreateCategoryFields.ts

import * as React from "react";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../domain/entity/productBlueprintCategory";

import { getProductBlueprintCategoryFieldKeys } from "../../../domain/entity/categoryFieldRegistry";

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

/**
 * model domain 管轄の field。
 * ProductBlueprint.categoryFields には保存しない。
 */
const MODEL_OWNED_CATEGORY_FIELD_KEYS = new Set<string>(["volume"]);

function isModelOwnedCategoryFieldKey(key: string): boolean {
  return MODEL_OWNED_CATEGORY_FIELD_KEYS.has(key);
}

function normalizeNumberValue(value: number): number {
  if (Number.isNaN(value)) {
    return 0;
  }

  return value < 0 ? 0 : value;
}

function normalizeStringArrayValue(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(
    (item): item is string => typeof item === "string" && item.trim() !== "",
  );
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
    if (isModelOwnedCategoryFieldKey(key)) {
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

  const onChangeQualityAssurance = React.useCallback((value: string[]) => {
    const next = normalizeStringArrayValue(value);

    setQualityAssurance(next);

    setCategoryFields((prev) => ({
      ...prev,
      washTags: next,
    }));
  }, []);

  const onChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      if (isModelOwnedCategoryFieldKey(key)) {
        setCategoryFields((prev) => {
          const next = { ...prev };
          delete next[key];
          return next;
        });
        return;
      }

      setCategoryFields((prev) => ({
        ...prev,
        [key]: value,
      }));

      if (key === "fit" && typeof value === "string") {
        setFit(value as Fit);
        return;
      }

      if (key === "material") {
        setMaterial(typeof value === "string" ? value : "");
        return;
      }

      if (key === "weight") {
        setWeight(typeof value === "number" ? normalizeNumberValue(value) : 0);
        return;
      }

      if (key === "washTags" || key === "qualityAssurance") {
        setQualityAssurance(normalizeStringArrayValue(value));
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
    onChangeQualityAssurance,
    onChangeCategoryField,
    resetCategoryFields,
  };
}