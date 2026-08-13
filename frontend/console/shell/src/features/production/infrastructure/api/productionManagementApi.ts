// frontend/console/shell/src/features/production/infrastructure/query/productionQuery.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import type { ProductionListItem } from "../../../../shared/types/production";

export async function listProductionsHTTP(): Promise<ProductionListItem[]> {
  return fetchJSON<ProductionListItem[]>(`${API_BASE}/productions`, {
    method: "GET",
    auth: "required",
  });
}