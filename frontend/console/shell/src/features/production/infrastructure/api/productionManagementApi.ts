// frontend/console/shell/src/features/production/infrastructure/query/productionQuery.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";

export type ProductionListModelResponse = {
  modelId: string;
  quantity: number;
};

export type ProductionListItemResponse = {
  id: string;
  productBlueprintId: string;
  productName: string;
  brandName: string;
  assigneeId: string;
  assigneeName: string;
  models: ProductionListModelResponse[];
  printed: boolean;
  printedAt: string | null;
  printedBy: string | null;
  printedByName: string;
  createdBy: string | null;
  createdByName: string;
  createdAt: string | null;
  updatedBy: string | null;
  updatedByName: string;
  updatedAt: string | null;
  totalQuantity: number;
};

export async function listProductionsHTTP(): Promise<ProductionListItemResponse[]> {
  return fetchJSON(`${API_BASE}/productions`, {
    method: "GET",
    auth: "required",
  });
}