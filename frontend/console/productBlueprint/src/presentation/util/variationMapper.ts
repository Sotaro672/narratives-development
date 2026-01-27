// frontend/console/productBlueprint/src/presentation/util/variationMapper.ts

import type { ItemType } from "../../domain/entity/catalog";
import type { SizeRow, ModelNumberRow } from "../../infrastructure/api/productBlueprintApi";
import { rgbIntToHex, coerceRgbInt } from "../../../../shell/src/shared/util/color";

/**
 * NOTE:
 * - backend 由来の variations は snake/camel/Pascal が混在しうるため、best-effort で吸収する。
 * - この util は「純粋変換」に徹し、setState など副作用は持たない。
 */

type AnyVar = any;

function s(v: unknown): string {
  return String(v ?? "").trim();
}

function pickSizeLabel(v: AnyVar): string {
  return s(typeof v?.size === "string" ? v.size : v?.Size);
}

function pickColorName(v: AnyVar): string {
  if (typeof v?.color?.name === "string") return s(v.color.name);
  if (typeof v?.Color?.Name === "string") return s(v.Color.Name);
  if (typeof v?.color?.Name === "string") return s(v.color.Name); // 念のため
  return "";
}

function pickModelNumber(v: AnyVar): string {
  return s(typeof v?.modelNumber === "string" ? v.modelNumber : v?.ModelNumber);
}

function pickMeasurements(v: AnyVar): Record<string, number | null> | undefined {
  const ms = v?.measurements ?? v?.Measurements;
  if (!ms || typeof ms !== "object") return undefined;
  return ms as Record<string, number | null>;
}

function pickRgbInt(v: AnyVar): number | undefined {
  // number / string / null が混ざりうる前提で吸収
  const raw =
    typeof v?.color?.rgb !== "undefined"
      ? v.color.rgb
      : typeof v?.color?.RGB !== "undefined"
        ? v.color.RGB
        : typeof v?.Color?.RGB !== "undefined"
          ? v.Color.RGB
          : undefined;

  return coerceRgbInt(raw);
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
 * variations から SizeRow[] を構築
 * - 既存実装に合わせ、id は "1".."N" の連番（表示用）で生成
 * - measurements は itemType に応じて SizeRow の各フィールドへマッピング
 */
export function buildSizeRows(varsAny: unknown[], itemType: ItemType): SizeRow[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const sizes = extractSizes(list);

  return sizes.map((label, index) => {
    const base: any = {
      id: String(index + 1),
      sizeLabel: label,
    };

    // その sizeLabel に該当する variation を1つ拾って measurements を参照
    const found = list.find((v) => pickSizeLabel(v) === label);
    const ms = found ? pickMeasurements(found) : undefined;

    if (ms) {
      if (itemType === "ボトムス") {
        base.waist = ms["ウエスト"] ?? undefined;
        base.hip = ms["ヒップ"] ?? undefined;
        base.rise = ms["股上"] ?? undefined;
        base.inseam = ms["股下"] ?? undefined;

        const thighVal = ms["わたり幅"] ?? undefined;
        if (thighVal != null) base.thigh = thighVal;

        base.hemWidth = ms["裾幅"] ?? undefined;
      } else {
        const lenVal = ms["着丈"] ?? undefined;
        if (lenVal != null) base.length = lenVal;

        const chestVal = ms["胸囲"] ?? undefined;
        if (chestVal != null) base.chest = chestVal;

        const widthVal = ms["身幅"] ?? undefined;
        if (widthVal != null) base.width = widthVal;

        const shoulderVal = ms["肩幅"] ?? undefined;
        if (shoulderVal != null) base.shoulder = shoulderVal;

        const sleeveVal = ms["袖丈"] ?? undefined;
        if (sleeveVal != null) base.sleeveLength = sleeveVal;
      }
    }

    return base as SizeRow;
  });
}

/** variations から ModelNumberRow[]（size/color/code）を構築 */
export function buildModelNumberRows(varsAny: unknown[]): ModelNumberRow[] {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];

  return list.map((v) => {
    const size = pickSizeLabel(v);
    const color = pickColorName(v);
    const code = pickModelNumber(v);

    return { size, color, code } as ModelNumberRow;
  });
}

/**
 * variations から colorRgbMap を構築
 * - rgb int を "#rrggbb" に正規化して格納
 * - 不正値はスキップ
 */
export function buildColorRgbMap(varsAny: unknown[]): Record<string, string> {
  const list = Array.isArray(varsAny) ? (varsAny as AnyVar[]) : [];
  const rgbMap: Record<string, string> = {};

  for (const v of list) {
    const name = pickColorName(v);
    if (!name) continue;

    const rgbInt = pickRgbInt(v);
    const hex = rgbIntToHex(rgbInt);
    if (!hex) continue;

    // 既存と合わせて lower-case を採用（UI入力と差分が出にくい）
    rgbMap[name] = hex.toLowerCase();
  }

  return rgbMap;
}

/**
 * variations をまとめて UI state に使える形へ変換
 * - hook 側の try/catch を短くするための convenience
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
