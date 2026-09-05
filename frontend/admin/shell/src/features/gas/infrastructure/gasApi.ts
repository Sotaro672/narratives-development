// frontend/admin/shell/src/features/gas/infrastructure/gasApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { GasBalance } from "../../../shared/type/gas";

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }
  return BACKEND_BASE_URL;
}

async function requireOk(response: Response): Promise<void> {
  if (response.ok) return;

  let detail = "";

  try {
    const body = await response.json() as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`Failed to load gas balance. status=${response.status}${detail}`);
}

export async function getGasBalance(): Promise<GasBalance> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${backendBaseUrl}/admin/gas`, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response);

  return response.json() as Promise<GasBalance>;
}