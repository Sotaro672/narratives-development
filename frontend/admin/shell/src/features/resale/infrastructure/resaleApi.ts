// frontend/admin/shell/src/features/resale/infrastructure/resaleApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { AvatarResale } from "../../../shared/type/avatar";

const BACKEND_BASE_URL =
  import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }

  return BACKEND_BASE_URL;
}

async function requireOk(response: Response): Promise<void> {
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
    `Failed to load resale. status=${response.status}${detail}`,
  );
}

export async function getResaleDetail(
  avatarId: string,
  resaleId: string,
): Promise<AvatarResale> {
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

  await requireOk(response);

  return (await response.json()) as AvatarResale;
}