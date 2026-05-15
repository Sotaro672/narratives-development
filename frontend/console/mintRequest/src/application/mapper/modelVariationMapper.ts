// frontend/console/mintRequest/src/application/mapper/modelVariationMapper.ts

import type { ModelVariationForMintDTO } from "../../infrastructure/dto/mintRequestLocal.dto";
import type { MintModelMetaEntry } from "../../presentation/hook/useInspectionResultCard";

export function getModelVariationModelNumber(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  const modelNumber = (variation as any)?.modelNumber ?? null;

  return typeof modelNumber === "string" && modelNumber.trim()
    ? modelNumber.trim()
    : null;
}

export function getModelVariationSize(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  const size = (variation as any)?.size ?? null;

  return typeof size === "string" && size.trim() ? size.trim() : null;
}

export function getModelVariationColorName(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  const colorName =
    (variation as any)?.color?.name ?? (variation as any)?.colorName ?? null;

  return typeof colorName === "string" && colorName.trim()
    ? colorName.trim()
    : null;
}

export function getModelVariationRgb(
  variation: ModelVariationForMintDTO | null | undefined,
): number | null {
  if (!variation) return null;

  const rgb = (variation as any)?.color?.rgb ?? (variation as any)?.rgb ?? null;

  return typeof rgb === "number" && Number.isFinite(rgb) ? rgb : null;
}

export function toMintModelMetaEntry(
  variation: ModelVariationForMintDTO | null | undefined,
): MintModelMetaEntry | null {
  if (!variation) return null;

  return {
    modelNumber: getModelVariationModelNumber(variation),
    size: getModelVariationSize(variation),
    colorName: getModelVariationColorName(variation),
    rgb: getModelVariationRgb(variation),
  };
}