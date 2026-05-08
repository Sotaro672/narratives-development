// frontend/console/inventory/src/application/listCreate/priceCard.types.ts

/**
 * inventory/application 側で PriceCard 系の型を正として定義する
 * - list/presentation はこの型を参照する
 *
 * backend response を正とする。
 *
 * PriceRow:
 * - modelId を識別子として使う
 * - id / modelID / model_id などの名揺れは持たない
 *
 * NOTE:
 * - React の key は displayOrder ではなく modelId を使う
 * - displayOrder は重複/未設定があり得る
 * - 並び順は displayOrder 昇順のみ
 * - 未設定は null を保持し、UI 側で末尾扱いにする
 */

import type * as React from "react";

export type PriceCardMode = "view" | "edit";

export type PriceRow = {
  /**
   * 識別子。
   *
   * backend response:
   * {
   *   "modelId": "..."
   * }
   */
  modelId: string;

  /**
   * 並び順。
   *
   * backend response:
   * {
   *   "displayOrder": 1
   * }
   *
   * 未設定は null のまま保持する。
   */
  displayOrder?: number | null;

  size: string;
  color: string;

  /**
   * RGB。
   *
   * backend response では number が基本。
   * 既存 UI 互換として "#RRGGBB" string も許容する。
   */
  rgb?: number | string | null;

  stock: number;

  /**
   * 価格。
   * 未設定の場合は null。
   */
  price?: number | null;
};

export type PriceCardProps = {
  title?: string;
  rows: PriceRow[];
  className?: string;

  mode?: PriceCardMode;

  /**
   * edit 時に価格を更新するコールバック。
   *
   * IMPORTANT:
   * hook 内で displayOrder で並べ替えても、
   * index は「元の rows 配列の index」を返す。
   */
  onChangePrice?: (index: number, price: number | null, row: PriceRow) => void;

  currencySymbol?: string;
};

export type PriceRowVM = {
  /**
   * React key 用の識別子。
   */
  modelId: string;

  /**
   * 並び順。
   * 未設定は null。
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