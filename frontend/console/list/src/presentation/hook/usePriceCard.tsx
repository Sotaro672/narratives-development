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
            : a.row.displayOrder;

        const bo =
          b.row.displayOrder === null || b.row.displayOrder === undefined
            ? Number.POSITIVE_INFINITY
            : b.row.displayOrder;

        if (ao !== bo) return ao - bo;
        return 0;
      });

    return sorted.map(({ row, originalIdx }) => {
      const modelId = s(row.modelId);

      const rgbHex = rgbIntToHex(row.rgb) ?? null;

      const bgColor =
        typeof row.rgb === "string" && String(row.rgb).trim().startsWith("#")
          ? String(row.rgb).trim()
          : rgbHex ?? "#ffffff";

      const rgbTitle =
        rgbHex ?? (typeof row.rgb === "string" ? String(row.rgb) : "");

      const priceInputValue =
        row.price === null || row.price === undefined ? "" : String(row.price);

      const priceDisplayText =
        row.price === null || row.price === undefined
          ? "-"
          : `${currencySymbol ?? ""}${row.price}`;

      const onChangePriceInput = (e: React.ChangeEvent<HTMLInputElement>) => {
        const raw = e.target.value;
        const next = parsePriceInput(raw);

        // 元の rows 配列の index を返す
        onChangePrice?.(originalIdx, next, row);
      };

      const displayOrder =
        row.displayOrder === null || row.displayOrder === undefined
          ? null
          : row.displayOrder;

      return {
        modelId,
        displayOrder,
        size: row.size,
        color: row.color,
        stock: Number(row.stock ?? 0),
        bgColor,
        rgbTitle,
        priceInputValue,
        priceDisplayText,
        onChangePriceInput,
      } as PriceRowVM;
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