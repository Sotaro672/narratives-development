// frontend/console/shell/src/features/production/infrastructure/api/productionDetailApi.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import type { ProductionDetail } from "../../../../shared/types/production";

export async function fetchProductionDetail(productionId: string): Promise<ProductionDetail> {
  const id = productionId.trim();
  if (!id) {
    throw new Error("fetchProductionDetail: productionId が空です");
  }

  const url = `${API_BASE}/productions/${encodeURIComponent(id)}`;
  const response = await fetchJSON<ProductionDetail>(url, {
    method: "GET",
    auth: "required",
  });

  return {
    ...response,
    assigneeName: response.assigneeName ?? "",
    printedByName: response.printedByName ?? "",
    createdByName: response.createdByName ?? "",
    updatedByName: response.updatedByName ?? "",
  };
}