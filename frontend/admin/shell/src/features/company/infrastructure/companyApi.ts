// frontend/admin/shell/src/features/company/infrastructure/companyApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type {
  Company,
  CompanyListResponse,
} from "../../../shared/type/company";

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

  throw new Error(
    `${message} status=${response.status}${detail}`,
  );
}

export async function listCompanies(): Promise<Company[]> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/companies`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(
    response,
    "Failed to load companies.",
  );

  const body =
    (await response.json()) as CompanyListResponse;

  return Array.isArray(body.items)
    ? body.items
    : [];
}