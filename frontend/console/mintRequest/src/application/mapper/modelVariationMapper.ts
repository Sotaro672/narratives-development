import type {
  MintModelMetaEntryDTO,
  ModelVariationForMintDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function pick(obj: unknown, ...keys: string[]): unknown {
  if (!isRecord(obj)) return null;

  for (const key of keys) {
    const value = obj[key];
    if (value !== undefined && value !== null) {
      return value;
    }
  }

  return null;
}

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

function getVolumeObject(
  variation: ModelVariationForMintDTO | null | undefined,
): unknown {
  if (!variation) return null;

  return pick(variation, "volume", "Volume");
}

function getVolumeValueFromObject(value: unknown): string | number | null {
  if (!isRecord(value)) return null;

  return toVolume(pick(value, "value", "Value"));
}

function getVolumeUnitFromObject(value: unknown): string | null {
  if (!isRecord(value)) return null;

  return toText(pick(value, "unit", "Unit"));
}

export function getModelVariationModelNumber(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  return toText(pick(variation, "modelNumber", "ModelNumber"));
}

export function getModelVariationSize(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  return toText(pick(variation, "size", "Size"));
}

export function getModelVariationColorName(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  const color = pick(variation, "color", "Color");

  return (
    toText(pick(color, "name", "Name")) ??
    toText(pick(variation, "colorName", "ColorName"))
  );
}

export function getModelVariationRgb(
  variation: ModelVariationForMintDTO | null | undefined,
): number | null {
  if (!variation) return null;

  const color = pick(variation, "color", "Color");

  return (
    toNumberOrNull(pick(color, "rgb", "RGB")) ??
    toNumberOrNull(pick(variation, "rgb", "RGB"))
  );
}

export function getModelVariationVolume(
  variation: ModelVariationForMintDTO | null | undefined,
): string | number | null {
  if (!variation) return null;

  const volume = getVolumeObject(variation);

  return (
    getVolumeValueFromObject(volume) ??
    toVolume(pick(variation, "volumeValue", "VolumeValue")) ??
    toVolume(volume)
  );
}

export function getModelVariationVolumeUnit(
  variation: ModelVariationForMintDTO | null | undefined,
): string | null {
  if (!variation) return null;

  const volume = getVolumeObject(variation);

  return (
    getVolumeUnitFromObject(volume) ??
    toText(pick(variation, "volumeUnit", "VolumeUnit", "unit", "Unit"))
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