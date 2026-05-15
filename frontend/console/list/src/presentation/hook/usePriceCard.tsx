// frontend/console/list/src/presentation/hook/usePriceCard.tsx

import * as React from "react";

import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

// 型は inventory/application を正とする
import type {
  PriceCardProps,
  PriceRowVM,
  UsePriceCardResult,
} from "../../../../inventory/src/application/listCreate/listCreate.types";

// ----------------------------------------------------------
// Helpers
// ----------------------------------------------------------
function s(v: unknown): string {
  return String(v ?? "").trim();
}

/**
 * 数値入力の正規化
 * - 全角数字 → 半角
 * - カンマ除去（1,000 → 1000）
 * - 空白除去
 */
function normalizeNumericString(raw: string): string {
  const t = s(raw);
  if (!t) return "";

  const half = t.replace(/[０-９]/g, (ch) => {
    const code = ch.charCodeAt(0) - 0xfee0;
    return String.fromCharCode(code);
  });

  return half.replace(/[, \t]/g, "");
}

function parsePriceInput(v: string): number | null {
  const normalized = normalizeNumericString(v);
  if (!normalized) return null;

  const num = Number(normalized);
  if (!Number.isFinite(num)) return null;

  const int = Math.floor(num);
  return int < 0 ? 0 : int;
}

function toDisplayOrder(v: unknown): number | null {
  if (v === null || v === undefined) return null;

  const n = Number(v);
  return Number.isFinite(n) ? n : null;
}

function toStock(v: unknown): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function getBgColor(rgb: unknown): string {
  const rgbHex = rgbIntToHex(rgb as number | string | null | undefined) ?? null;

  if (typeof rgb === "string" && rgb.trim().startsWith("#")) {
    return rgb.trim();
  }

  return rgbHex ?? "#ffffff";
}

function getRgbTitle(rgb: unknown): string {
  const rgbHex = rgbIntToHex(rgb as number | string | null | undefined) ?? null;

  if (rgbHex) {
    return rgbHex;
  }

  if (typeof rgb === "string") {
    return rgb;
  }

  return "";
}

function getPriceInputValue(price: unknown): string {
  if (price === null || price === undefined) return "";
  return String(price);
}

function getPriceDisplayText(args: {
  price: unknown;
  currencySymbol: string;
}): string {
  const { price, currencySymbol } = args;

  if (price === null || price === undefined) {
    return "-";
  }

  return `${currencySymbol ?? ""}${price}`;
}

function getVolumeValue(row: { volumeValue?: number | null }): number | null {
  const value = row.volumeValue;

  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  return null;
}

function getVolumeUnit(row: { volumeUnit?: string | null }): string | null {
  const unit = s(row.volumeUnit);
  return unit || null;
}

// ----------------------------------------------------------
// Hook
// ----------------------------------------------------------
export function usePriceCard(props: PriceCardProps): UsePriceCardResult {
  const {
    title = "価格設定",
    rows,
    mode = "view",
    onChangePrice,
    currencySymbol = "¥",
  } = props;

  const isEdit = mode === "edit";
  const showModeBadge = mode !== "view";

  const rowsVM = React.useMemo<PriceRowVM[]>(() => {
    const sorted = (rows ?? [])
      .map((row, originalIdx) => ({ row, originalIdx }))
      .sort((a, b) => {
        const ao =
          a.row.displayOrder === null || a.row.displayOrder === undefined
            ? Number.POSITIVE_INFINITY
            : Number(a.row.displayOrder);

        const bo =
          b.row.displayOrder === null || b.row.displayOrder === undefined
            ? Number.POSITIVE_INFINITY
            : Number(b.row.displayOrder);

        if (ao !== bo) return ao - bo;

        // displayOrder が同じ場合は、元の rows 配列の順序を維持する
        return a.originalIdx - b.originalIdx;
      });

    return sorted.map(({ row, originalIdx }) => {
      const modelId = s(row.modelId);

      const priceInputValue = getPriceInputValue(row.price);

      const priceDisplayText = getPriceDisplayText({
        price: row.price,
        currencySymbol,
      });

      const onChangePriceInput = (e: React.ChangeEvent<HTMLInputElement>) => {
        const raw = e.target.value;
        const next = parsePriceInput(raw);

        // 元の rows 配列の index を返す
        onChangePrice?.(originalIdx, next, row);
      };

      return {
        modelId,

        kind: row.kind ?? null,

        displayOrder: toDisplayOrder(row.displayOrder),

        // apparel category 用
        size: row.size ?? null,
        color: row.color ?? null,

        // alcohol category 用
        volumeValue: getVolumeValue(row),
        volumeUnit: getVolumeUnit(row),

        stock: toStock(row.stock),

        bgColor: getBgColor(row.rgb),
        rgbTitle: getRgbTitle(row.rgb),

        priceInputValue,
        priceDisplayText,

        onChangePriceInput,
      };
    });
  }, [rows, onChangePrice, currencySymbol]);

  return {
    title,
    mode,
    isEdit,
    showModeBadge,
    currencySymbol,
    rowsVM,
    isEmpty: (rows ?? []).length === 0,
  };
}