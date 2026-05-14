// frontend/console/production/src/application/detail/types.ts

import type {
  // quantity の最小表現は domain を正にする
  ModelQuantity,
} from "../../../../production/src/domain/entity/production";

/**
 * Production 詳細（backend ProductionDetailDTO と整合）
 * - createdAt/updatedAt/printedAt は Date として保持する（未ロード時は null を許容）
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
  // true: 印刷済
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
 * model variation summary
 *
 * apparel / alcohol の両方を扱う。
 * production 側では modelId を正キーとして扱う。
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
   * 元 response の volume を保持したい箇所向け。
   * buildProductionQuantityRowVMs 側が meta.volume を読む場合にも対応する。
   */
  volume?: {
    value: number;
    unit: string;
  };

  displayOrder?: number;
};

/**
 * domain の ModelQuantity（modelId, quantity）を正として拡張する
 * - modelId が正キー
 * - quantity は domain と同一
 * - 表示用のメタ情報だけを追加
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