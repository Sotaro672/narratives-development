// frontend/console/shell/src/features/production/application/detail/types.ts

import type { ModelQuantity } from "../../../../shared/types/production";
import type { ProductBlueprintCategorySnapshot } from "../../../productBlueprint/domain/productBlueprintCategory";

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

  createdById?: string | null;
  createdByName: string;
  createdAt: Date | null;

  updatedById?: string | null;
  updatedByName: string;
  updatedAt: Date | null;
};

export type ProductionQuantityRow = ModelQuantity & {
  kind?: "apparel" | "alcohol" | string;
  modelNumber: string;

  size?: string;
  color?: string;
  rgb?: number | string | null;

  volumeValue?: number;
  volumeUnit?: string;

  variationLabel?: string;
  displayOrder?: number;
};