// frontend/console/mintRequest/src/application/mapper/modelVariationMapper.ts

import type {
  MintModelMetaEntryDTO,
  ModelVariationForMintDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

function toText(value: unknown): string | null {
  if (typeof value !== "string") return null;

  const trimmed = value.trim();
  return trimmed ? trimmed : null;
}

function toNumberOrNull(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return null;

    const parsed = Number(trimmed);
    return Number.isFinite(parsed) ? parsed : null;
  }

  return null;
}

function toVolume(value: unknown): string | number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed ? trimmed : null;
  }

  return null;
}

export function getModelVariationModelNumber(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  return toText((variation as any)?.modelNumber);
}

export function getModelVariationSize(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  return toText((variation as any)?.size);
}

export function getModelVariationColorName(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  return (
    toText((variation as any)?.color?.name) ??
    toText((variation as any)?.colorName)
  );
}

export function getModelVariationRgb(
  variation: ModelVariationForMintDTO | null | undefined,
): number | null {
  if (!variation) return null;

  return (
    toNumberOrNull((variation as any)?.color?.rgb) ??
    toNumberOrNull((variation as any)?.rgb)
  );
}

export function getModelVariationVolume(
  variation: ModelVariationForMintDTO | null | undefined,
): string | number | null {
  if (!variation) return null;

  return toVolume((variation as any)?.volume);
}

export function getModelVariationVolumeUnit(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  return (
    toText((variation as any)?.volumeUnit) ??
    toText((variation as any)?.unit)
  );
}

export function toMintModelMetaEntry(
  variation: ModelVariationForMintDTO | null | undefined,
): MintModelMetaEntryDTO | null {
  if (!variation) return null;

  return {
    modelNumber: getModelVariationModelNumber(variation),
    size: getModelVariationSize(variation),
    colorName: getModelVariationColorName(variation),
    rgb: getModelVariationRgb(variation),
    volume: getModelVariationVolume(variation),
    volumeUnit: getModelVariationVolumeUnit(variation),
  };
}