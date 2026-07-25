// frontend/console/productBlueprint/src/presentation/util/variationMapper.ts

import type {
  ApparelModelNumberRow as ModelNumberRow,
  ApparelSizeRow as SizeRow,
} from "../../domain/entity/apparel";

import type {
  AlcoholModelNumber,
  Volume,
  VolumeRow,
} from "../../../../model/src/application/modelCreateService";

import {
  rgbIntToHex,
  coerceRgbInt,
} from "../../../../shell/src/shared/util/color";

type AnyVar = any;

export type VariationsUiState = {
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];
  colorRgbMap: Record<string, string>;

  /**
   * alcohol model variation 用。
   * volume は ProductBlueprint.categoryFields ではなく model domain 側で扱う。
   */
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
};

function s(value: unknown): string {
  return String(value ?? "").trim();
}

function pickId(value: AnyVar): string {
  return s(typeof value?.id === "string" ? value.id : "");
}

function pickKind(value: AnyVar): string {
  return s(typeof value?.kind === "string" ? value.kind : "apparel");
}

function pickSizeLabel(value: AnyVar): string {
  return s(typeof value?.size === "string" ? value.size : "");
}

function pickColorName(value: AnyVar): string {
  if (typeof value?.color?.name === "string") {
    return s(value.color.name);
  }

  return "";
}

function pickModelNumber(value: AnyVar): string {
  return s(typeof value?.modelNumber === "string" ? value.modelNumber : "");
}

function pickMeasurements(
  value: AnyVar,
): Record<string, number | null> | undefined {
  const measurements = value?.measurements;

  if (!measurements || typeof measurements !== "object") {
    return undefined;
  }

  return measurements as Record<string, number | null>;
}

function pickRgbInt(value: AnyVar): number | undefined {
  return coerceRgbInt(value?.color?.rgb);
}

function pickVolume(value: AnyVar): Volume | null {
  const rawVolume = value?.volume;

  if (!rawVolume || typeof rawVolume !== "object") {
    return null;
  }

  const rawValue = rawVolume.value;
  const rawUnit = rawVolume.unit;

  if (typeof rawValue !== "number" || !Number.isFinite(rawValue)) {
    return null;
  }

  const unit = s(rawUnit) || "ml";

  if (rawValue <= 0) {
    return null;
  }

  return {
    value: rawValue,
    unit,
  };
}

function toVolumeLabel(volume: Volume): string {
  const value =
    typeof volume.value === "number" && Number.isFinite(volume.value)
      ? volume.value
      : 0;

  const unit = s(volume.unit) || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function isApparelVariation(value: AnyVar): boolean {
  return pickKind(value) === "apparel";
}

function isAlcoholVariation(value: AnyVar): boolean {
  return pickKind(value) === "alcohol";
}

function isBottomsLikeCategory(categoryCode: string): boolean {
  const normalized = s(categoryCode).toLowerCase();
  return normalized === "apparel.bottoms";
}

function isDressLikeCategory(categoryCode: string): boolean {
  const normalized = s(categoryCode).toLowerCase();
  return normalized === "apparel.dress";
}

/**
 * variations から color 名のユニーク配列を抽出
 */
export function extractColors(varsAny: unknown[]): string[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const set = new Set<string>();

  for (const variation of list) {
    if (!isApparelVariation(variation)) {
      continue;
    }

    const name = pickColorName(variation);

    if (name) {
      set.add(name);
    }
  }

  return Array.from(set);
}

/**
 * variations から sizeLabel のユニーク配列を抽出
 */
export function extractSizes(varsAny: unknown[]): string[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const set = new Set<string>();

  for (const variation of list) {
    if (!isApparelVariation(variation)) {
      continue;
    }

    const size = pickSizeLabel(variation);

    if (size) {
      set.add(size);
    }
  }

  return Array.from(set);
}

/**
 * variations から volume label のユニーク配列を抽出
 */
export function extractVolumeLabels(varsAny: unknown[]): string[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const set = new Set<string>();

  for (const variation of list) {
    if (!isAlcoholVariation(variation)) {
      continue;
    }

    const volume = pickVolume(variation);

    if (!volume) {
      continue;
    }

    const label = toVolumeLabel(volume);

    if (label) {
      set.add(label);
    }
  }

  return Array.from(set);
}

/**
 * 同一サイズの variations 群から measurements をマージする
 * - 先に入っている値を優先
 * - 未設定項目だけ後続 variation から補完
 */
function mergeMeasurementsBySize(
  variations: AnyVar[],
): Record<string, number | undefined> {
  const merged: Record<string, number | undefined> = {};

  for (const variation of variations) {
    const measurements = pickMeasurements(variation);

    if (!measurements) {
      continue;
    }

    for (const [key, value] of Object.entries(measurements)) {
      if (typeof value !== "number" || Number.isNaN(value)) {
        continue;
      }

      if (merged[key] !== undefined) {
        continue;
      }

      merged[key] = value;
    }
  }

  return merged;
}

/**
 * variations から SizeRow[] を構築
 * - backend の id を代表 variation から使う
 * - 同じサイズの複数色 variation から measurements をマージする
 */
export function buildSizeRows(
  varsAny: unknown[],
  categoryCode: string,
): SizeRow[] {
  const list = Array.isArray(varsAny)
    ? (varsAny as AnyVar[]).filter(isApparelVariation)
    : [];

  const sizeLabels = extractSizes(list);

  return sizeLabels.map((label) => {
    const sameSizeVars = list.filter(
      (variation) => pickSizeLabel(variation) === label,
    );

    const representative = sameSizeVars[0];
    const merged = mergeMeasurementsBySize(sameSizeVars);

    const base: SizeRow = {
      id: pickId(representative) || `size-${label}`,
      sizeLabel: label,
    };

    if (isBottomsLikeCategory(categoryCode)) {
      base.waist = merged["ウエスト"];
      base.hip = merged["ヒップ"];
      base.rise = merged["股上"];
      base.inseam = merged["股下"];
      base.thigh = merged["わたり幅"];
      base.hemWidth = merged["裾幅"];
      return base;
    }

    if (isDressLikeCategory(categoryCode)) {
      base.length = merged["着丈"];
      base.width = merged["身幅"];
      base.chest = merged["胸囲"];
      base.shoulder = merged["肩幅"];
      base.sleeveLength = merged["袖丈"];
      base.waist = merged["ウエスト"];
      base.hip = merged["ヒップ"];
      return base;
    }

    base.length = merged["着丈"];
    base.width = merged["身幅"];
    base.chest = merged["胸囲"];
    base.shoulder = merged["肩幅"];
    base.sleeveLength = merged["袖丈"];

    return base;
  });
}

/**
 * apparel variations から ModelNumberRow[]（size/color/code）を構築
 */
export function buildModelNumberRows(varsAny: unknown[]): ModelNumberRow[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];

  return list
    .filter(isApparelVariation)
    .map((variation) => {
      const size = pickSizeLabel(variation);
      const color = pickColorName(variation);
      const code = pickModelNumber(variation);

      return {
        size,
        color,
        code,
      };
    })
    .filter((row) => row.size || row.color || row.code);
}

/**
 * alcohol variations から VolumeRow[] を構築
 */
export function buildVolumeRows(varsAny: unknown[]): VolumeRow[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const seen = new Set<string>();
  const rows: VolumeRow[] = [];

  for (const variation of list) {
    if (!isAlcoholVariation(variation)) {
      continue;
    }

    const volume = pickVolume(variation);

    if (!volume) {
      continue;
    }

    const label = toVolumeLabel(volume);

    if (!label || seen.has(label)) {
      continue;
    }

    seen.add(label);

    rows.push({
      id: pickId(variation) || `volume-${label}`,
      volumeValue: volume.value,
      volumeUnit: volume.unit,
    });
  }

  return rows;
}

/**
 * alcohol variations から AlcoholModelNumber[] を構築
 */
export function buildAlcoholModelNumberRows(
  varsAny: unknown[],
): AlcoholModelNumber[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];

  return list
    .filter(isAlcoholVariation)
    .map((variation) => {
      const volume = pickVolume(variation);
      const code = pickModelNumber(variation);

      if (!volume) {
        return null;
      }

      const volumeLabel = toVolumeLabel(volume);

      if (!volumeLabel && !code) {
        return null;
      }

      return {
        kind: "alcohol" as const,
        volume,
        volumeLabel,
        code,
      };
    })
    .filter((row): row is AlcoholModelNumber => row !== null);
}

/**
 * apparel variations から colorRgbMap を構築
 */
export function buildColorRgbMap(varsAny: unknown[]): Record<string, string> {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const rgbMap: Record<string, string> = {};

  for (const variation of list) {
    if (!isApparelVariation(variation)) {
      continue;
    }

    const name = pickColorName(variation);

    if (!name) {
      continue;
    }

    const rgbInt = pickRgbInt(variation);
    const hex = rgbIntToHex(rgbInt);

    if (!hex) {
      continue;
    }

    rgbMap[name] = hex.toLowerCase();
  }

  return rgbMap;
}

/**
 * variations をまとめて UI state に使える形へ変換
 */
export function mapVariationsToUiState(args: {
  varsAny: unknown[];
  categoryCode: string;
}): VariationsUiState {
  const { varsAny, categoryCode } = args;

  const colors = extractColors(varsAny);
  const sizes = buildSizeRows(varsAny, categoryCode);
  const modelNumbers = buildModelNumberRows(varsAny);
  const colorRgbMap = buildColorRgbMap(varsAny);

  const volumes = buildVolumeRows(varsAny);
  const alcoholModelNumbers = buildAlcoholModelNumberRows(varsAny);

  return {
    colors,
    sizes,
    modelNumbers,
    colorRgbMap,
    volumes,
    alcoholModelNumbers,
  };
}