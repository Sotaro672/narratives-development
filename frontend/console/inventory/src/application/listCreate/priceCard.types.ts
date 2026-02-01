// frontend/console/inventory/src/application/listCreate/priceCard.types.ts

/**
 * ✅ inventory/application 側で PriceCard 系の型を正として定義する
 * - list/presentation はこの型を参照する（依存方向を正す）
 *
 * NOTE:
 * - React の key は displayOrder ではなく modelId を使う（displayOrder は重複/未設定があり得る）
 * - 並び順は displayOrder 昇順「のみ」。未設定は null を保持し、UI 側で末尾扱いにする
 */

import type * as React from "react";

export type PriceCardMode = "view" | "edit";

export type PriceRow = {
  /**
   * ✅ 識別子（modelId）
   * - DTO の modelId をここに詰める
   */
  id: string;

  /**
   * ✅ 並び順（modelRefs.displayOrder）
   * - 未設定は null のまま保持
   */
  displayOrder?: number | null;

  size: string;
  color: string;

  /**
   * RGB
   * - int(0xRRGGBB) で来ることもあるので、表示時に hex 化して dot に反映する
   * - "#RRGGBB" の string が来ても許容（hook 側の rgbIntToHex が string も受ける前提）
   */
  rgb?: number | string | null;

  stock: number;
  price?: number | null;
};

export type PriceCardProps = {
  title?: string;
  rows: PriceRow[];
  className?: string;

  mode?: PriceCardMode;

  /**
   * edit 時に価格を更新するコールバック
   * - IMPORTANT: hook 内で displayOrder で並べ替えても、index は「元の rows 配列の index」を返す
   */
  onChangePrice?: (index: number, price: number | null, row: PriceRow) => void;

  currencySymbol?: string;
};

export type PriceRowVM = {
  /**
   * ✅ React key 用の識別子
   * - modelId（= PriceRow.id）を正とする
   */
  modelId: string;

  /**
   * ✅ 並び順（未設定は null）
   */
  displayOrder: number | null;

  size: string;
  color: string;
  stock: number;

  bgColor: string;
  rgbTitle: string;

  priceInputValue: string;
  priceDisplayText: string;

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
