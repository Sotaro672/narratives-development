// frontend/console/inventory/src/application/listCreate/listCreate.types.ts

import type * as React from "react";
import type { RefObject } from "react";

/**
 * Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
 */
export type ImageInputRef = RefObject<HTMLInputElement | null>;

/**
 * List create route params
 *
 * UI ルートは inventoryId（= inventoryKey: "pb__tb"）のみを正とする。
 * productBlueprintId / tokenBlueprintId は互換用途では扱わない。
 */
export type ListCreateRouteParams = {
  inventoryId?: string;
};

export type ResolvedListCreateParams = {
  inventoryId: string;
  raw: ListCreateRouteParams;
};

/**
 * POST /lists の priceRows
 *
 * - modelId を識別子として使う
 * - 未入力 price は undefined のまま素通りさせる
 * - 明示的な未設定は null
 * - 入力済み価格は number
 */
export type CreateListPriceRow = {
  modelId: string;
  price?: number | null;
};

export type PriceCardMode = "view" | "edit";

/**
 * PriceCard 用 row
 *
 * backend response を正とする。
 *
 * - modelId を識別子として使う
 * - id / modelID / model_id などの名揺れは持たない
 * - React key は displayOrder ではなく modelId を使う
 * - displayOrder は重複/未設定があり得る
 * - 並び順は displayOrder 昇順のみ
 * - 未設定は null を保持し、UI 側で末尾扱いにする
 */
export type PriceRow = {
  modelId: string;

  /**
   * 並び順。
   * 未設定は null のまま保持する。
   */
  displayOrder?: number | null;

  size: string;
  color: string;

  /**
   * RGB。
   * backend response では number が基本。
   * 既存 UI 互換として "#RRGGBB" string も許容する。
   */
  rgb?: number | string | null;

  stock: number;

  /**
   * 価格。
   * 未入力は undefined、明示的な未設定は null。
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