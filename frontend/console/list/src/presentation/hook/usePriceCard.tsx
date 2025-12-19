// frontend/console/list/src/presentation/hook/usePriceCard.tsx
import * as React from "react";

// ----------------------------------------------------------
// Types
// ----------------------------------------------------------
export type PriceRow = {
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

  /** 合計行の表示（デフォルト true） */
  showTotal?: boolean;
};

// ----------------------------------------------------------
// Helpers
// ----------------------------------------------------------
function s(v: unknown): string {
  return String(v ?? "").trim();
}

// RGB(int) → HEX (#RRGGBB)
// - row.rgb は string | number | null どれでも来うる前提で安全に変換
function rgbIntToHex(rgb: number | string | null | undefined): string | null {
  if (rgb === null || rgb === undefined) return null;
  const n = typeof rgb === "string" ? Number(rgb) : rgb;
  if (!Number.isFinite(n)) return null;

  const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
  const hex = clamped.toString(16).padStart(6, "0");
  return `#${hex}`;
}

function parsePriceInput(v: string): number | null {
  const trimmed = s(v);
  if (!trimmed) return null;

  const num = Number(trimmed);
  if (!Number.isFinite(num)) return null;

  // number input なので基本は整数想定（step=1）。念のため floor。
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
  showTotal: boolean;

  rowsVM: PriceRowVM[];
  isEmpty: boolean;

  totalStock: number;
  totalPrice: number;
};

export function usePriceCard(props: PriceCardProps): UsePriceCardResult {
  const {
    title = "価格設定",
    rows,
    mode = "view",
    onChangePrice,
    currencySymbol = "¥",
    showTotal = true,
  } = props;

  const isEdit = mode === "edit";
  const showModeBadge = mode !== "view";

  const totalStock = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [rows],
  );

  const totalPrice = React.useMemo(
    () => rows.reduce((sum, r) => sum + (Number(r.price) || 0), 0),
    [rows],
  );

  const rowsVM = React.useMemo<PriceRowVM[]>(() => {
    return rows.map((row, idx) => {
      const rgbHex = rgbIntToHex(row.rgb);

      const bgColor =
        row.rgb &&
        typeof row.rgb === "string" &&
        row.rgb.trim().startsWith("#")
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
        const next = parsePriceInput(e.target.value);
        onChangePrice?.(idx, next, row);
      };

      return {
        key: `${String(row.size)}-${String(row.color)}-${idx}`,
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
    showTotal,

    rowsVM,
    isEmpty: rows.length === 0,

    totalStock,
    totalPrice,
  };
}
