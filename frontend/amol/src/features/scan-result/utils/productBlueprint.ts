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

export function createProductBlueprintRows(
  productBlueprintPatch: Record<string, unknown> | null
): InfoRow[] {
  if (!productBlueprintPatch) {
    return [];
  }

  const productIdTag = getRecord(productBlueprintPatch, "productIdTag");

  const rows: InfoRow[] = [
    {
      label: "種別",
      value: getString(productBlueprintPatch, "itemType"),
    },
    {
      label: "フィット",
      value: getString(productBlueprintPatch, "fit"),
    },
    {
      label: "素材",
      value: getString(productBlueprintPatch, "material"),
    },
    {
      label: "重量",
      value: toDisplayValue(getNumber(productBlueprintPatch, "weight")),
    },
    {
      label: "商品IDタグ",
      value: getString(productIdTag, "Type"),
    },
  ];

  return rows.filter((row) => row.value);
}