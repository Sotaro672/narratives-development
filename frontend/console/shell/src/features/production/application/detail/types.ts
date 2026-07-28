// frontend/console/shell/src/features/production/application/detail/types.ts

import type { ModelQuantity } from "../../../../shared/types/production";

/**
 * Production 詳細（backend ProductionDetailDTO と整合）
 * - createdAt / updatedAt / printedAt は Date として保持する
 * - 未ロード時は null を許容する
 */
export type ProductionDetail = {
  id: string;
  productBlueprintId: string;

  // Brand（NameResolver 済み）
  brandId: string;
  brandName: string;

  // Assignee（NameResolver 済み）
  assigneeId: string;
  assigneeName: string;

  // Printed
  // true: 印刷済み
  // false: 印刷前
  printed: boolean;

  // Model breakdown
  models: ProductionQuantityRow[];
  totalQuantity: number;

  // timestamps
  printedAt: Date | null;

  createdById?: string | null;
  createdByName: string;
  createdAt: Date | null;

  updatedById?: string | null;
  updatedByName: string;
  updatedAt: Date | null;
};

/**
 * ModelVariation の表示用概要。
 *
 * apparel / alcohol の両方を扱う。
 * Production側では modelId を正キーとして扱う。
 */
export type ModelVariationSummary = {
  modelId: string;
  productBlueprintId?: string;

  kind?: "apparel" | "alcohol" | string;

  modelNumber: string;

  // apparel
  size?: string;
  color?: string;
  rgb?: number | string | null;

  // alcohol
  volumeValue?: number;
  volumeUnit?: string;

  /**
   * 元レスポンスの volume を保持する。
   * buildProductionQuantityRowVMs が meta.volume を参照する場合にも対応する。
   */
  volume?: {
    value: number;
    unit: string;
  };

  displayOrder?: number;
};

/**
 * shared/types/production.ts の
 * ModelQuantity（modelId, quantity）を正として拡張する。
 *
 * - modelId が正キー
 * - quantity は共有型と同一
 * - 表示に必要なメタ情報のみ追加する
 */
export type ProductionQuantityRow = ModelQuantity & {
  kind?: "apparel" | "alcohol" | string;

  modelNumber: string;

  // apparel
  size?: string;
  color?: string;
  rgb?: number | string | null;

  // alcohol
  volumeValue?: number;
  volumeUnit?: string;

  /**
   * 共通表示用。
   *
   * apparel: "M / Green"
   * alcohol: "720ml"
   */
  variationLabel?: string;

  displayOrder?: number;
};