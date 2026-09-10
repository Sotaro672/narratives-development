// frontend/admin/shell/src/features/mint/infrastructure/mintApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";

import type { Mint, MintDetail, MintListResponse } from "../../../shared/type/mint";

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
    const body = (await response.json()) as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`Failed to load mint data. status=${response.status}${detail}`);
}

export async function listMints(): Promise<Mint[]> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${backendBaseUrl}/admin/mints`, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response);

  const body = (await response.json()) as MintListResponse;
  return Array.isArray(body.items) ? body.items : [];
}

export async function getMintDetail(mintId: string): Promise<MintDetail> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/mints/${encodeURIComponent(mintId)}`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(response);

  return (await response.json()) as MintDetail;
}