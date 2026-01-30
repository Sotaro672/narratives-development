// frontend/console/production/src/application/detail/types.ts

import type {
  // ✅ domain を正にする（ProductionStatus をここから取る）
  ProductionStatus,
  // ✅ quantity の最小表現は domain を正にする
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

  // Status
  status: ProductionStatus;

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
 * backend dto.ProductionModelRowDTO と整合
 */
export type ModelVariationSummary = {
  modelId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  displayOrder?: number;
};

/**
 * ✅ domain の ModelQuantity（modelId, quantity）を正として拡張する
 * - modelId が正キー
 * - quantity は domain と同一
 * - 表示用のメタ情報だけを追加
 */
export type ProductionQuantityRow = ModelQuantity & {
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  displayOrder?: number;
};

export type { ProductionStatus };
