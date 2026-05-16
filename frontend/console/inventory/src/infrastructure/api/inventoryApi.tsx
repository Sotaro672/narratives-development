// frontend/console/inventory/src/infrastructure/api/inventoryApi.tsx

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

// ---------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------
async function requestJsonOrThrow(path: string): Promise<any> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`request failed: ${res.status} ${res.statusText} ${text}`);
  }

  return await res.json();
}

function s(v: unknown): string {
  return String(v ?? "").trim();
}

// ---------------------------------------------------------
// Inventory APIs (raw JSON)
// ---------------------------------------------------------

/** GET /inventory */
export async function getInventoryListRaw(): Promise<any> {
  return await requestJsonOrThrow(`/inventory`);
}

/** GET /product-blueprints/{id} */
export async function getProductBlueprintRaw(
  productBlueprintId: string,
): Promise<any> {
  const pbId = s(productBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");

  return await requestJsonOrThrow(
    `/product-blueprints/${encodeURIComponent(pbId)}`,
  );
}

/** GET /inventory */
export async function getPrintedProductBlueprintsRaw(): Promise<any> {
  return await requestJsonOrThrow(`/inventory`);
}

/** GET /inventory/{inventoryId} */
export async function getInventoryDetailRaw(inventoryId: string): Promise<any> {
  const id = s(inventoryId);
  if (!id) throw new Error("inventoryId is empty");

  return await requestJsonOrThrow(`/inventory/${encodeURIComponent(id)}`);
}