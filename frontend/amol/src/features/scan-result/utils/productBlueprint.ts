// frontend/amol/src/features/scan-result/utils/productBlueprint.ts

import { getNumber, getRecord, getString } from "./guards";

export type InfoRow = {
  label: string;
  value: string;
};

function toDisplayValue(value: string | number | null): string {
  if (value === null) {
    return "";
  }

  if (typeof value === "number") {
    return String(value);
  }

  return value.trim();
}

function getPatchOrCategoryString(
  productBlueprintPatch: Record<string, unknown>,
  categoryFields: Record<string, unknown> | null,
  key: string,
): string {
  return getString(categoryFields, key) || getString(productBlueprintPatch, key);
}

function getPatchOrCategoryNumber(
  productBlueprintPatch: Record<string, unknown>,
  categoryFields: Record<string, unknown> | null,
  key: string,
): number | null {
  return getNumber(categoryFields, key) ?? getNumber(productBlueprintPatch, key);
}

export function createProductBlueprintRows(
  productBlueprintPatch: Record<string, unknown> | null,
): InfoRow[] {
  if (!productBlueprintPatch) {
    return [];
  }

  const productIdTag = getRecord(productBlueprintPatch, "productIdTag");
  const categoryFields = getRecord(productBlueprintPatch, "categoryFields");

  const rows: InfoRow[] = [
    {
      label: "種別",
      value: getPatchOrCategoryString(
        productBlueprintPatch,
        categoryFields,
        "itemType",
      ),
    },
    {
      label: "フィット",
      value: getPatchOrCategoryString(
        productBlueprintPatch,
        categoryFields,
        "fit",
      ),
    },
    {
      label: "素材",
      value: getPatchOrCategoryString(
        productBlueprintPatch,
        categoryFields,
        "material",
      ),
    },
    {
      label: "重量",
      value: toDisplayValue(
        getPatchOrCategoryNumber(
          productBlueprintPatch,
          categoryFields,
          "weight",
        ),
      ),
    },
    {
      label: "商品IDタグ",
      value: getString(productIdTag, "Type") || getString(productIdTag, "type"),
    },
  ];

  return rows.filter((row) => row.value);
}