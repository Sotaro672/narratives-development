// frontend/console/list/src/presentation/hook/usePriceCard.tsx
import * as React from "react";

import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

// ----------------------------------------------------------
// Types
// ----------------------------------------------------------
export type PriceRow = {
  /** ✅ optional: id (= modelId) を持てるようにする（hook 連携とデバッグのため） */
  id?: string;

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
   */
  onChangePrice?: (index: number, price: number | null, row: PriceRow) => void;

  /** 表示用（例: "¥" / "$"）。未指定なら空 */
  currencySymbol?: string;

  /** ✅ 互換のため残す（合計行は廃止。受け取っても無視） */
  showTotal?: boolean;
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
  key: string;
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

  const rowsVM = React.useMemo<PriceRowVM[]>(() => {
    return rows.map((row, idx) => {
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
        onChangePrice?.(idx, next, row);
      };

      const stableId = s((row as any)?.id);
      const keyBase =
        stableId || `${String(row.size)}-${String(row.color)}-${idx}`;

      return {
        key: keyBase,
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
