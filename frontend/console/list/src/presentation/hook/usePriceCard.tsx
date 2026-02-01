// frontend/console/list/src/presentation/hook/usePriceCard.tsx
import * as React from "react";

import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

// ----------------------------------------------------------
// Types
// ----------------------------------------------------------
export type PriceRow = {
  /** ✅ optional: id (= modelId) を持てるようにする（hook 連携とデバッグのため） */
  id?: string;

  /**
   * ✅ modelRefs.displayOrder に対応（並び順はこれの昇順のみ）
   * - backend の productBlueprintPatch.ModelRefs.DisplayOrder を詰めて渡す想定
   */
  displayOrder?: number | null;

  /** サイズ (例: "S" | "M" | "L") */
  size: string;

  /** カラー表示名 (例: "ホワイト") */
  color: string;

  /**
   * RGB
   * - int(0xRRGGBB) で来ることもあるので、表示時に hex 化して dot に反映する
   * - "#RRGGBB" の string が来ても許容
   */
  rgb?: number | string | null;

  /** 在庫数 */
  stock: number;

  /** ✅ 価格（円など。UIは数値入力） */
  price?: number | null;
};

export type PriceCardMode = "view" | "edit";

export type PriceCardProps = {
  title?: string;
  rows: PriceRow[];
  className?: string;

  /** view / edit */
  mode?: PriceCardMode;

  /**
   * edit 時に価格を更新するコールバック
   * - 親が rows を state 管理している前提
   *
   * IMPORTANT:
   * - hook 内で displayOrder で並べ替えても、index は「元の rows 配列の index」を返す
   */
  onChangePrice?: (index: number, price: number | null, row: PriceRow) => void;

  /** 表示用（例: "¥" / "$"）。未指定なら空 */
  currencySymbol?: string;
};

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
// ViewModel
// ----------------------------------------------------------
export type PriceRowVM = {
  // ✅ key ではなく displayOrder を使う
  displayOrder: number;

  size: string;
  color: string;
  stock: number;

  // color dot
  bgColor: string;
  rgbTitle: string;

  // price
  priceInputValue: string; // Input value
  priceDisplayText: string; // view text

  // handler
  onChangePriceInput: (e: React.ChangeEvent<HTMLInputElement>) => void;
};

export type UsePriceCardResult = {
  title: string;
  mode: PriceCardMode;
  isEdit: boolean;
  showModeBadge: boolean;

  currencySymbol: string;

  rowsVM: PriceRowVM[];
  isEmpty: boolean;
};

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
    // ✅ modelRefs.displayOrder の昇順「のみ」に従って並べる（同値は元順＝安定）
    const sorted = rows
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
        return a.originalIdx - b.originalIdx; // 安定
      });

    // ✅ 並び替え結果も確認できるようにログ
    console.groupCollapsed("[usePriceCard] rows sort result");
    console.log("sorted (row + originalIdx):", sorted);
    console.table(
      sorted.map(({ row, originalIdx }, sortedIdx) => ({
        sortedIdx,
        originalIdx,
        id: (row as any)?.id,
        displayOrder: (row as any)?.displayOrder,
        size: row.size,
        color: row.color,
        stock: row.stock,
        rgb: row.rgb,
        price: row.price,
      })),
    );
    console.groupEnd();

    return sorted.map(({ row, originalIdx }, _sortedIdx) => {
      const rgbHex = rgbIntToHex(row.rgb) ?? null;

      const bgColor =
        typeof row.rgb === "string" && row.rgb.trim().startsWith("#")
          ? row.rgb.trim()
          : rgbHex ?? "#ffffff";

      const rgbTitle = rgbHex ?? (typeof row.rgb === "string" ? row.rgb : "");

      const priceInputValue =
        row.price === null || row.price === undefined ? "" : String(row.price);

      const priceDisplayText =
        row.price === null || row.price === undefined
          ? "-"
          : `${currencySymbol ?? ""}${row.price}`;

      const onChangePriceInput = (e: React.ChangeEvent<HTMLInputElement>) => {
        const raw = e.target.value;
        const next = parsePriceInput(raw);
        // IMPORTANT: 元の rows 配列の index を返す
        onChangePrice?.(originalIdx, next, row);

        console.debug("[usePriceCard] onChangePriceInput", {
          originalIdx,
          id: (row as any)?.id,
          displayOrder: (row as any)?.displayOrder,
          raw,
          parsed: next,
          row,
        });
      };

      // ✅ PriceRowVM の識別子は displayOrder を採用（未設定は 0）
      const displayOrder =
        row.displayOrder === null || row.displayOrder === undefined
          ? 0
          : row.displayOrder;

      return {
        displayOrder,
        size: row.size,
        color: row.color,
        stock: row.stock,
        bgColor,
        rgbTitle,
        priceInputValue,
        priceDisplayText,
        onChangePriceInput,
      };
    });
  }, [rows, onChangePrice, currencySymbol]);

  // ✅ VM も確認できるようにログ（必要ならコメントアウト）
  React.useEffect(() => {
    console.groupCollapsed("[usePriceCard] rowsVM snapshot");
    console.log("rowsVM:", rowsVM);
    console.table(
      (rowsVM ?? []).map((r, i) => ({
        i,
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
    isEmpty: rows.length === 0,
  };
}
