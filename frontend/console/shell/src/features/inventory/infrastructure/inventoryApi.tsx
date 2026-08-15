// frontend/console/shell/src/features/inventory/infrastructure/inventoryApi.tsx

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";

// ---------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------

async function requestJsonOrThrow(path: string): Promise<any> {
  const headers = await getAuthHeaders();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");

    throw new Error(
      `request failed: ${res.status} ${res.statusText} ${text}`,
    );
  }

  return res.json();
}

function s(value: unknown): string {
  return String(value ?? "").trim();
}

// ---------------------------------------------------------
// Inventory APIs
// ---------------------------------------------------------

/**
 * GET /inventory
 */
export async function getInventoryListRaw(): Promise<any> {
  return requestJsonOrThrow("/inventory");
}

/**
 * GET /inventory/{inventoryId}
 */
export async function getInventoryDetailRaw(
  inventoryId: string,
): Promise<any> {
  const id = s(inventoryId);

  if (!id) {
    throw new Error("inventoryId is empty");
  }

  return requestJsonOrThrow(
    `/inventory/${encodeURIComponent(id)}`,
  );
}