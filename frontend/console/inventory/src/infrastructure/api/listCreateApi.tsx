// frontend/console/inventory/src/infrastructure/api/listCreateApi.tsx

// ✅ Shared console API base (修正案A)
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";

// ✅ Shared auth headers (shell authService を委譲)
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
// ListCreate API (raw JSON)
// ---------------------------------------------------------

/**
 * GET
 * - /inventory/list-create/:inventoryId
 *
 * ✅ pbId/tbId ルートは廃止
 */
export async function getListCreateRaw(input: {
  inventoryId?: string;
}): Promise<any> {
  const inventoryId = s(input.inventoryId);

  if (!inventoryId) {
    throw new Error("missing inventoryId");
  }

  const path = `/inventory/list-create/${encodeURIComponent(inventoryId)}`;
  return await requestJsonOrThrow(path);
}
