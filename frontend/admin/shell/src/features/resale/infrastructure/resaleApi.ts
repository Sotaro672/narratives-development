// frontend/admin/shell/src/features/resale/infrastructure/resaleApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { Resale } from "../../../shared/type/resale";
import type { ResaleReviewResponse } from "../../../shared/type/resaleReview";

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

export async function getResaleDetail(
  avatarId: string,
  resaleId: string,
): Promise<Resale> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/avatars/${encodeURIComponent(avatarId)}/resales/${encodeURIComponent(resaleId)}`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(response, "Failed to load resale.");

  return (await response.json()) as Resale;
}

export async function getResaleReviews(
  avatarId: string,
  resaleId: string,
  page = 1,
  perPage = 20,
): Promise<ResaleReviewResponse> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const query = new URLSearchParams({
    page: String(page),
    perPage: String(perPage),
  });

  const response = await fetch(
    `${backendBaseUrl}/admin/avatars/${encodeURIComponent(avatarId)}/resales/${encodeURIComponent(resaleId)}/reviews?${query.toString()}`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(response, "Failed to load resale reviews.");

  return (await response.json()) as ResaleReviewResponse;
}