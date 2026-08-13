// frontend/console/shell/src/features/inventory/infrastructure/api/listCreateApi.tsx

import { API_BASE } from "../../../../shared/http/apiBase";

import { getAuthHeadersOrThrow } from "../../../../shared/http/authHeaders";

import type {
  ListCreateDTO,
} from "../http/listCreateRepositoryHTTP.types";

// ---------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------

async function requestJsonOrThrow<T>(
  path: string,
): Promise<T> {
  const headers =
    await getAuthHeadersOrThrow();

  const res =
    await fetch(
      `${API_BASE}${path}`,
      {
        method: "GET",
        headers,
      },
    );

  if (!res.ok) {
    const text =
      await res
        .text()
        .catch(() => "");

    throw new Error(
      `request failed: ${res.status} ${res.statusText} ${text}`,
    );
  }

  return await res.json() as T;
}

function s(
  value: unknown,
): string {
  return String(
    value ?? "",
  ).trim();
}

// ---------------------------------------------------------
// ListCreate API
// ---------------------------------------------------------

/**
 * GET
 * - /inventory/list-create/:inventoryId
 *
 * pbId/tbId ルートは廃止。
 * Backend BFF の ListCreateDTO をそのまま正とする。
 */
export async function getListCreateRaw(
  input: {
    inventoryId?: string;
  },
): Promise<ListCreateDTO> {
  const inventoryId =
    s(
      input.inventoryId,
    );

  if (!inventoryId) {
    throw new Error(
      "missing inventoryId",
    );
  }

  const path =
    `/inventory/list-create/${encodeURIComponent(
      inventoryId,
    )}`;

  return await requestJsonOrThrow<ListCreateDTO>(
    path,
  );
}