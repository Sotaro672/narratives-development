// frontend/console/list/src/presentation/hook/usePriceCard.tsx
import * as React from "react";

import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

// ✅ 型は inventory/application を正とする（依存方向を正す）
import type {
  PriceCardProps,
  PriceRowVM,
  UsePriceCardResult,
} from "../../../../inventory/src/application/listCreate/priceCard.types";

// ----------------------------------------------------------
// Helpers
// ----------------------------------------------------------
function s(v: unknown): string {
  return String(v ?? "").trim();
}

/**
 * ✅ 数値入力の正規化
 * - 全角数字 → 半角
 * - カンマ除去（1,000 → 1000）
 * - 空白除去
 */
function normalizeNumericString(raw: string): string {
  const t = s(raw);
  if (!t) return "";

  // 全角 ０-９ → 半角 0-9
  const half = t.replace(/[０-９]/g, (ch) => {
    const code = ch.charCodeAt(0) - 0xfee0;
    return String.fromCharCode(code);
  });

  // カンマ/スペース除去
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

  // ✅ hook に渡ってきた “取得済みの全データ” を確認
  React.useEffect(() => {
    console.groupCollapsed("[usePriceCard] props snapshot");
    console.log("title:", title);
    console.log("mode:", mode);
    console.log("currencySymbol:", currencySymbol);
    console.log("rows (raw):", rows);
    console.table(
      (rows ?? []).map((r, i) => ({
        i,
        id: (r as any)?.id,
        displayOrder: (r as any)?.displayOrder,
        size: (r as any)?.size,
        color: (r as any)?.color,
        stock: (r as any)?.stock,
        rgb: (r as any)?.rgb,
        price: (r as any)?.price,
      })),
    );
    console.log("hasOnChangePrice:", typeof onChangePrice === "function");
    console.groupEnd();
  }, [title, mode, currencySymbol, rows, onChangePrice]);

  const rowsVM = React.useMemo<PriceRowVM[]>(() => {
    // ✅ 並び順は displayOrder の昇順「のみ」
    // - 未設定(null/undefined) は末尾へ
    // - 同値は何もしない（0 を返す）
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
        return 0; // ✅ 同値は何もしない（displayOrder のみに従う）
      });

    // ✅ 並び替え結果ログ
    console.groupCollapsed("[usePriceCard] rows sort result");
    console.log("sorted (row + originalIdx):", sorted);
    console.table(
      sorted.map(({ row, originalIdx }, sortedIdx) => ({
        sortedIdx,
        originalIdx,
        id: (row as any)?.id,
        displayOrder: (row as any)?.displayOrder,
        size: (row as any)?.size,
        color: (row as any)?.color,
        stock: (row as any)?.stock,
        rgb: (row as any)?.rgb,
        price: (row as any)?.price,
      })),
    );
    console.groupEnd();

    return sorted.map(({ row, originalIdx }) => {
      const modelId = s((row as any)?.id);

      const rgbHex = rgbIntToHex((row as any)?.rgb) ?? null;

      const bgColor =
        typeof (row as any)?.rgb === "string" && String((row as any)?.rgb).trim().startsWith("#")
          ? String((row as any)?.rgb).trim()
          : rgbHex ?? "#ffffff";

      const rgbTitle = rgbHex ?? (typeof (row as any)?.rgb === "string" ? (row as any)?.rgb : "");

      const priceInputValue =
        (row as any)?.price === null || (row as any)?.price === undefined
          ? ""
          : String((row as any)?.price);

      const priceDisplayText =
        (row as any)?.price === null || (row as any)?.price === undefined
          ? "-"
          : `${currencySymbol ?? ""}${(row as any)?.price}`;

      const onChangePriceInput = (e: React.ChangeEvent<HTMLInputElement>) => {
        const raw = e.target.value;
        const next = parsePriceInput(raw);

        // IMPORTANT: 元の rows 配列の index を返す
        onChangePrice?.(originalIdx, next, row);

        console.debug("[usePriceCard] onChangePriceInput", {
          originalIdx,
          modelId,
          displayOrder: (row as any)?.displayOrder ?? null,
          raw,
          parsed: next,
          row,
        });
      };

      // ✅ VM の displayOrder は null を保持
      const displayOrder =
        (row as any)?.displayOrder === null || (row as any)?.displayOrder === undefined
          ? null
          : (row as any)?.displayOrder;

      return {
        modelId,
        displayOrder,
        size: (row as any)?.size,
        color: (row as any)?.color,
        stock: Number((row as any)?.stock ?? 0),
        bgColor,
        rgbTitle,
        priceInputValue,
        priceDisplayText,
        onChangePriceInput,
      } as PriceRowVM;
    });
  }, [rows, onChangePrice, currencySymbol]);

  // ✅ VM ログ
  React.useEffect(() => {
    console.groupCollapsed("[usePriceCard] rowsVM snapshot");
    console.log("rowsVM:", rowsVM);
    console.table(
      (rowsVM ?? []).map((r, i) => ({
        i,
        modelId: r.modelId,
        displayOrder: r.displayOrder,
        size: r.size,
        color: r.color,
        stock: r.stock,
        bgColor: r.bgColor,
        rgbTitle: r.rgbTitle,
        priceInputValue: r.priceInputValue,
        priceDisplayText: r.priceDisplayText,
      })),
    );
    console.groupEnd();
  }, [rowsVM]);

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
