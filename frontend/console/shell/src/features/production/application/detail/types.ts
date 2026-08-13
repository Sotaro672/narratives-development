// frontend/console/shell/src/features/production/application/detail/types.ts

import type { ProductBlueprintCategorySnapshot } from "../../../productBlueprint/domain/productBlueprintCategory";
import type { ProductionQuantityRow } from "../productionQuantityRow";

export type ProductionDetail = {
  id: string;
  productBlueprintId: string;
  productName: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  brandId: string;
  brandName: string;
  assigneeId: string;
  assigneeName: string;
  printed: boolean;
  models: ProductionQuantityRow[];
  totalQuantity: number;
  printedAt: Date | null;
  createdBy?: string | null;
  createdByName: string;
  createdAt: Date | null;
  updatedBy?: string | null;
  updatedByName: string;
  updatedAt: Date | null;
};