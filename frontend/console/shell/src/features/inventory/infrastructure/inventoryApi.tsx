// frontend/console/shell/src/features/inventory/infrastructure/inventoryApi.tsx

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";

import type {
  InventoryDetailDTO,
} from "../../../shared/types/inventory";

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

async function patchJsonOrThrow(
  path: string,
  body: unknown,
): Promise<any> {
  const authHeaders =
    await getAuthHeaders();

  const headers =
    new Headers(
      authHeaders,
    );

  headers.set(
    "Content-Type",
    "application/json",
  );

  const res = await fetch(`${API_BASE}${path}`, {
    method: "PATCH",
    headers,
    body: JSON.stringify(
      body,
    ),
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

/**
 * PATCH /inventory/{inventoryId}/shipping-address
 */
export async function updateInventoryShippingAddressRaw(
  inventoryId: string,
  shippingAddressId: string,
): Promise<InventoryDetailDTO> {
  const id =
    s(inventoryId);

  const addressId =
    s(shippingAddressId);

  if (!id) {
    throw new Error(
      "inventoryId is empty",
    );
  }

  if (!addressId) {
    throw new Error(
      "shippingAddressId is empty",
    );
  }

  return patchJsonOrThrow(
    `/inventory/${encodeURIComponent(id)}/shipping-address`,
    {
      shippingAddressId:
        addressId,
    },
  );
}