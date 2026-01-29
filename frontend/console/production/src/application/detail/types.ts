// frontend\console\production\src\application\detail\types.ts
import type {
  ProductionStatus,
} from "../../../../shell/src/shared/types/production";

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

export type ProductionQuantityRow = {
  modelId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  displayOrder?: number;
  quantity: number;
};

export type { ProductionStatus };
