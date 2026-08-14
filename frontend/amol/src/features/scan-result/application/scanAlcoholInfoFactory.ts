// frontend/amol/src/features/scan-result/application/scanAlcoholInfoFactory.ts

import type {
  ProductBlueprintCategoryFields,
  ProductCategoryKind,
} from "../../shared/types/category";

export type ScanAlcoholInfo = {
  isAlcohol: boolean;
  vintage: string;
  region: string;
  material: string;
  alcoholContent: string;
  volumeLabel: string;
};

function toDisplayValue(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

export function buildScanVolumeLabel(input: {
  volumeValue?: number | null;
  volumeUnit?: string;
}): string {
  if (input.volumeValue == null) return "";
  return `${input.volumeValue}${input.volumeUnit ?? ""}`;
}

export function createScanAlcoholInfo(input: {
  categoryFields?: ProductBlueprintCategoryFields;
  volumeValue?: number | null;
  volumeUnit?: string;
  productBlueprintCategoryKind?: ProductCategoryKind;
}): ScanAlcoholInfo | null {
  if (input.productBlueprintCategoryKind !== "alcohol") {
    return null;
  }

  const fields = input.categoryFields ?? {};

  return {
    isAlcohol: true,
    vintage: toDisplayValue(fields.vintage),
    region: toDisplayValue(fields.region),
    material: toDisplayValue(fields.material),
    alcoholContent: toDisplayValue(fields.alcoholContent),
    volumeLabel: buildScanVolumeLabel({
      volumeValue: input.volumeValue,
      volumeUnit: input.volumeUnit,
    }),
  };
}