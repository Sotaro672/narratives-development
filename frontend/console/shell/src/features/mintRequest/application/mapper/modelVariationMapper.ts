// frontend/console/shell/src/features/mintRequest/application/mapper/modelVariationMapper.ts

import type {
  MintModelMetaEntryDTO,
  ModelVariationForMintDTO,
  ModelVolumeUnitForMint,
} from "../../infrastructure/dto/mintRequestLocal.dto";

/**
 * BackendのmodelVariationDTOからモデル番号を取得する。
 *
 * modelNumberはapparelとalcoholで共通の必須フィールド。
 */
export function getModelVariationModelNumber(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): string | null {
  return variation?.modelNumber ?? null;
}

/**
 * apparel variationのサイズを取得する。
 */
export function getModelVariationSize(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): string | null {
  if (
    !variation ||
    variation.kind !== "apparel"
  ) {
    return null;
  }

  return variation.size;
}

/**
 * apparel variationのカラー名を取得する。
 */
export function getModelVariationColorName(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): string | null {
  if (
    !variation ||
    variation.kind !== "apparel"
  ) {
    return null;
  }

  return variation.color.name;
}

/**
 * apparel variationのRGB値を取得する。
 */
export function getModelVariationRgb(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): number | null {
  if (
    !variation ||
    variation.kind !== "apparel"
  ) {
    return null;
  }

  return variation.color.rgb;
}

/**
 * alcohol variationの容量を取得する。
 *
 * Backendではvolume.valueはnumber。
 */
export function getModelVariationVolume(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): number | null {
  if (
    !variation ||
    variation.kind !== "alcohol"
  ) {
    return null;
  }

  return variation.volume.value;
}

/**
 * alcohol variationの容量単位を取得する。
 *
 * Backendで有効な値は"ml"または"L"。
 */
export function getModelVariationVolumeUnit(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): ModelVolumeUnitForMint | null {
  if (
    !variation ||
    variation.kind !== "alcohol"
  ) {
    return null;
  }

  return variation.volume.unit;
}

/**
 * GET /models/{id}のレスポンスを、
 * ミント申請詳細画面用のモデル情報へ変換する。
 */
export function toMintModelMetaEntry(
  variation:
    | ModelVariationForMintDTO
    | null
    | undefined,
): MintModelMetaEntryDTO | null {
  if (!variation) {
    return null;
  }

  if (variation.kind === "apparel") {
    return {
      modelId: variation.id,
      modelNumber: variation.modelNumber,
      size: variation.size,
      colorName: variation.color.name,
      rgb: variation.color.rgb,
    };
  }

  if (variation.kind === "alcohol") {
    return {
      modelId: variation.id,
      modelNumber: variation.modelNumber,
      volume: variation.volume.value,
      volumeUnit: variation.volume.unit,
    };
  }

  return null;
}