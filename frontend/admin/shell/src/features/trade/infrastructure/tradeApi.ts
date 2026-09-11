// frontend/admin/shell/src/features/trade/infrastructure/tradeApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { ResaleTradeListResponse } from "../../../shared/type/trade";

const BACKEND_BASE_URL =
  import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

async function requireOk(
  response: Response,
  message: string,
): Promise<void> {
  if (response.ok) {
    return;
  }

  let detail = "";

  try {
    const body = (await response.json()) as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`${message} status=${response.status}${detail}`);
}

export async function getResaleTrades(
  resaleId: string,
): Promise<ResaleTradeListResponse> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/resales/${encodeURIComponent(resaleId)}/trades`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(response, "Failed to load resale trades.");

  return (await response.json()) as ResaleTradeListResponse;
}