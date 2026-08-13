// frontend/console/shell/src/features/production/infrastructure/api/productionDetailApi.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import type { ProductBlueprintCategorySnapshot } from "../../../productBlueprint/domain/productBlueprintCategory";
import type { ProductionQuantityRow } from "../../application/productionQuantityRow";

export type ProductionDetailResponse = {
  id: string;
  productBlueprintId: string;
  productName: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  brandId: string;
  brandName: string;
  assigneeId: string;
  assigneeName: string;
  models: ProductionQuantityRow[];
  totalQuantity: number;
  printed: boolean;
  printedAt?: string | null;
  printedBy?: string | null;
  printedByName?: string;
  createdBy?: string | null;
  createdByName?: string;
  createdAt?: string | null;
  updatedBy?: string | null;
  updatedByName?: string;
  updatedAt?: string | null;
};

export async function fetchProductionDetail(
  productionId: string,
): Promise<ProductionDetailResponse> {
  const id = productionId.trim();

  if (!id) {
    throw new Error("fetchProductionDetail: productionId が空です");
  }

  const url = `${API_BASE}/productions/${encodeURIComponent(id)}`;

  return fetchJSON(url, {
    method: "GET",
    auth: "required",
  });
}