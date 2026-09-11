// frontend/admin/shell/src/features/productBlueprint/infrastructure/productBlueprintApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";

import type { ContractProductBlueprintDetailResponse } from "../model/productBlueprintDetail";
import type {
  ContractProductBlueprintReviewResponse,
  ContractProductBlueprintReviewStatus,
} from "../model/productBlueprintReview";

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

export async function getContractProductBlueprintDetail(
  companyId: string,
  productBlueprintId: string,
): Promise<ContractProductBlueprintDetailResponse> {
  const normalizedCompanyId = requireID(companyId, "companyId");
  const normalizedProductBlueprintId = requireID(
    productBlueprintId,
    "productBlueprintId",
  );

  return getAdminJSON<ContractProductBlueprintDetailResponse>(
    `/admin/companies/${encodeURIComponent(normalizedCompanyId)}/product-blueprints/${encodeURIComponent(normalizedProductBlueprintId)}`,
    "Failed to load contract product blueprint detail.",
  );
}

export async function getContractProductBlueprintReviews(
  companyId: string,
  productBlueprintId: string,
  status: ContractProductBlueprintReviewStatus = "PUBLISHED",
  page = 1,
  perPage = 20,
): Promise<ContractProductBlueprintReviewResponse> {
  const normalizedCompanyId = requireID(companyId, "companyId");
  const normalizedProductBlueprintId = requireID(
    productBlueprintId,
    "productBlueprintId",
  );

  const query = new URLSearchParams({
    status,
    page: String(page),
    perPage: String(perPage),
  });

  return getAdminJSON<ContractProductBlueprintReviewResponse>(
    `/admin/companies/${encodeURIComponent(normalizedCompanyId)}/product-blueprints/${encodeURIComponent(normalizedProductBlueprintId)}/reviews?${query.toString()}`,
    "Failed to load contract product blueprint reviews.",
  );
}