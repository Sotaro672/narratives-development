// frontend/admin/shell/src/features/avatar/infrastructure/avatarApi.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";

import type {
  Avatar,
  AvatarListResponse,
  AvatarResale,
  AvatarResaleListResponse,
} from "../../../shared/type/avatar";

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
    `Failed to load avatars. status=${response.status}${detail}`,
  );
}

export async function listAvatars(): Promise<Avatar[]> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${backendBaseUrl}/admin/avatars`, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  await requireOk(response);

  const body = (await response.json()) as AvatarListResponse;
  return Array.isArray(body.items) ? body.items : [];
}

export async function listAvatarResales(
  avatarId: string,
): Promise<AvatarResale[]> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(
    `${backendBaseUrl}/admin/avatars/${encodeURIComponent(avatarId)}/resales`,
    {
      method: "GET",
      headers: {
        ...authHeaders,
        Accept: "application/json",
      },
    },
  );

  await requireOk(response);

  const body = (await response.json()) as AvatarResaleListResponse;
  return Array.isArray(body.items) ? body.items : [];
}