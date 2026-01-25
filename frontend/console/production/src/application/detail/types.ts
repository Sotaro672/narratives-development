//frontend\console\production\src\application\detail\types.ts
import type {
  Production,
  ProductionStatus,
} from "../../../../shell/src/shared/types/production";

/**
 * 詳細表示用型（Production）
 * - createdAt/updatedAt/printedAt は Date として保持する
 */
export type ProductionDetail = Omit<
  Production,
  "createdAt" | "updatedAt" | "printedAt"
> & {
  totalQuantity: number;
  assigneeName?: string;
  productName?: string;
  brandName?: string;

  printedAt: Date | null;
  createdAt: Date | null;
  updatedAt: Date | null;
};

export type ModelVariationSummary = {
  id: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
};

export type ProductionQuantityRow = {
  id: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  quantity: number;
};

export type { ProductionStatus };
