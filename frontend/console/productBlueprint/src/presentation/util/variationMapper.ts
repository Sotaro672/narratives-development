// frontend/console/productBlueprint/src/presentation/util/variationMapper.ts

import type { ItemType } from "../../domain/entity/catalog";
import type {
  ProductBlueprintSizeRow as SizeRow,
  ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";
import {
  rgbIntToHex,
  coerceRgbInt,
} from "../../../../shell/src/shared/util/color";

type AnyVar = any;

function s(v: unknown): string {
  return String(v ?? "").trim();
}

function pickId(v: AnyVar): string {
  return s(typeof v?.id === "string" ? v.id : "");
}

function pickSizeLabel(v: AnyVar): string {
  return s(typeof v?.size === "string" ? v.size : "");
}

function pickColorName(v: AnyVar): string {
  if (typeof v?.color?.name === "string") return s(v.color.name);
  return "";
}

function pickModelNumber(v: AnyVar): string {
  return s(typeof v?.modelNumber === "string" ? v.modelNumber : "");
}

function pickMeasurements(
  v: AnyVar,
): Record<string, number | null> | undefined {
  const ms = v?.measurements;
  if (!ms || typeof ms !== "object") return undefined;
  return ms as Record<string, number | null>;
}

function pickRgbInt(v: AnyVar): number | undefined {
  return coerceRgbInt(v?.color?.rgb);
}

function isBottomsLike(itemType: ItemType): boolean {
  const normalized = String(itemType ?? "").trim().toLowerCase();
  return normalized === "bottoms" || normalized === "ボトムス";
}

/** variations から color 名のユニーク配列を抽出 */
export function extractColors(varsAny: unknown[]): string[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const set = new Set<string>();

  for (const v of list) {
    const name = pickColorName(v);
    if (name) set.add(name);
  }

  return Array.from(set);
}

/** variations から sizeLabel のユニーク配列を抽出 */
export function extractSizes(varsAny: unknown[]): string[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const set = new Set<string>();

  for (const v of list) {
    const size = pickSizeLabel(v);
    if (size) set.add(size);
  }

  return Array.from(set);
}

/**
 * 同一サイズの variations 群から measurements をマージする
 * - 先に入っている値を優先
 * - 未設定項目だけ後続 variation から補完
 */
function mergeMeasurementsBySize(
  vars: AnyVar[],
): Record<string, number | undefined> {
  const merged: Record<string, number | undefined> = {};

  for (const v of vars) {
    const ms = pickMeasurements(v);
    if (!ms) continue;

    for (const [key, value] of Object.entries(ms)) {
      if (typeof value !== "number" || Number.isNaN(value)) continue;
      if (merged[key] !== undefined) continue;
      merged[key] = value;
    }
  }

  return merged;
}

/**
 * variations から SizeRow[] を構築
 * - backend の id をそのまま使う
 * - 同じサイズの複数色 variation から measurements をマージする
 */
export function buildSizeRows(
  varsAny: unknown[],
  itemType: ItemType,
): SizeRow[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const sizeLabels = extractSizes(list);

  return sizeLabels.map((label) => {
    const sameSizeVars = list.filter((v) => pickSizeLabel(v) === label);
    const representative = sameSizeVars[0];
    const merged = mergeMeasurementsBySize(sameSizeVars);

    const base: SizeRow = {
      id: pickId(representative),
      sizeLabel: label,
    };

    if (isBottomsLike(itemType)) {
      base.waist = merged["ウエスト"];
      base.hip = merged["ヒップ"];
      base.rise = merged["股上"];
      base.inseam = merged["股下"];
      base.thigh = merged["わたり幅"];
      base.hemWidth = merged["裾幅"];
    } else {
      base.length = merged["着丈"];
      base.width = merged["身幅"];
      base.chest = merged["胸囲"];
      base.shoulder = merged["肩幅"];
      base.sleeveLength = merged["袖丈"];
    }

    return base;
  });
}

/** variations から ModelNumberRow[]（size/color/code）を構築 */
export function buildModelNumberRows(varsAny: unknown[]): ModelNumberRow[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];

  return list.map((v) => {
    const size = pickSizeLabel(v);
    const color = pickColorName(v);
    const code = pickModelNumber(v);

    return { size, color, code };
  });
}

/**
 * variations から colorRgbMap を構築
 */
export function buildColorRgbMap(
  varsAny: unknown[],
): Record<string, string> {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const rgbMap: Record<string, string> = {};

  for (const v of list) {
    const name = pickColorName(v);
    if (!name) continue;

    const rgbInt = pickRgbInt(v);
    const hex = rgbIntToHex(rgbInt);
    if (!hex) continue;

    rgbMap[name] = hex.toLowerCase();
  }

  return rgbMap;
}

/**
 * variations をまとめて UI state に使える形へ変換
 */
export function mapVariationsToUiState(args: {
  varsAny: unknown[];
  itemType: ItemType;
}): {
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];
  colorRgbMap: Record<string, string>;
} {
  const { varsAny, itemType } = args;

  const colors = extractColors(varsAny);
  const sizes = buildSizeRows(varsAny, itemType);
  const modelNumbers = buildModelNumberRows(varsAny);
  const colorRgbMap = buildColorRgbMap(varsAny);

  return { colors, sizes, modelNumbers, colorRgbMap };
}