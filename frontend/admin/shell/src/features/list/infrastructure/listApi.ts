// frontend/admin/shell/src/features/list/infrastructure/listApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { ContractListDetailResponse } from "../model/listDetail";

const BACKEND_BASE_URL =
  import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

function requireID(value: string, name: string): string {
  const normalized = value.trim();

  if (!normalized) {
    throw new Error(`${name} is required.`);
  }

  return normalized;
}

async function requireOk(
  response: Response,
  message: string,
): Promise<void> {
  if (response.ok) return;

  let detail = "";

  try {
    const body = (await response.json()) as { error?: string };
    detail = body.error ? ` error=${body.error}` : "";
  } catch {
    // Response body may not be JSON.
  }

  throw new Error(`${message} status=${response.status}${detail}`);
}

async function getAdminJSON<T>(
  path: string,
  errorMessage: string,
): Promise<T> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${backendBaseUrl}${path}`, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response, errorMessage);

  return (await response.json()) as T;
}

export async function getContractListDetail(
  companyId: string,
  listId: string,
): Promise<ContractListDetailResponse> {
  const normalizedCompanyId = requireID(companyId, "companyId");
  const normalizedListId = requireID(listId, "listId");

  return getAdminJSON<ContractListDetailResponse>(
    `/admin/companies/${encodeURIComponent(normalizedCompanyId)}/lists/${encodeURIComponent(normalizedListId)}`,
    "Failed to load contract list detail.",
  );
}