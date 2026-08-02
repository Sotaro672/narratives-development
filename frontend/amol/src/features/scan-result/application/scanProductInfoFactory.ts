// frontend/amol/src/features/scan-result/application/scanProductInfoFactory.ts

import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";
import type { PreviewState } from "../../shared/types/scanResult";
import {
  createScanAlcoholInfo,
  type ScanAlcoholInfo,
} from "./scanAlcoholInfoFactory";

export type ScanProductInfoViewModel = {
  productId: string;
  productBlueprintId: string;
  productName: string;
  brandName: string;
  companyName: string;

  modelKind: string;
  modelNumber: string;
  modelLabel: string;

  size: string;
  color: string;
  rgb: number | null;
  measurements: Record<string, number>;

  alcoholInfo: ScanAlcoholInfo | null;
};

function getString(value: unknown): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

function getNumberOrNull(
  value: unknown,
): number | null {
  return isFiniteNumber(value)
    ? value
    : null;
}

function getMeasurements(
  value: unknown,
): Record<string, number> {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    return {};
  }

  return Object.entries(value).reduce<
    Record<string, number>
  >(
    (acc, [key, rawValue]) => {
      if (isFiniteNumber(rawValue)) {
        acc[key] = rawValue;
      }

      return acc;
    },
    {},
  );
}

function getProductName(
  raw: Record<string, unknown>,
): string {
  const patch = raw.productBlueprintPatch;

  if (
    isRecord(patch) &&
    !Array.isArray(patch)
  ) {
    const productName = getString(
      patch.productName,
    );

    if (productName) {
      return productName;
    }
  }

  return getString(raw.productName);
}

export function createScanProductInfoViewModel(
  previewState: PreviewState | null,
): ScanProductInfoViewModel | null {
  const raw = previewState?.raw;

  if (
    !isRecord(raw) ||
    Array.isArray(raw)
  ) {
    return null;
  }

  const rawPatch =
    raw.productBlueprintPatch;

  const patch =
    isRecord(rawPatch) &&
    !Array.isArray(rawPatch)
      ? rawPatch
      : {};

  const categoryFields =
    patch.categoryFields;

  const alcoholInfo =
    createScanAlcoholInfo({
      categoryFields,
      volumeValue: raw.volumeValue,
      volumeUnit: raw.volumeUnit,
    });

  return {
    productId: getString(
      raw.productId,
    ),
    productBlueprintId: getString(
      raw.productBlueprintId,
    ),
    productName: getProductName(raw),
    brandName: getString(
      raw.brandName,
    ),
    companyName: getString(
      raw.companyName,
    ),

    modelKind: getString(
      raw.modelKind,
    ),
    modelNumber: getString(
      raw.modelNumber,
    ),
    modelLabel: getString(
      raw.modelLabel,
    ),

    size: getString(raw.size),
    color: getString(raw.color),
    rgb: getNumberOrNull(raw.rgb),
    measurements: getMeasurements(
      raw.measurements,
    ),

    alcoholInfo,
  };
}