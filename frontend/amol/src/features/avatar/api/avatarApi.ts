// frontend/src/features/avatar/api/avatarApi.ts

import type {
  CreateAvatarPayload,
  CreateAvatarResponse,
  MyAvatarResponse,
  UpdateAvatarPayload,
  UpdateAvatarResponse,
} from "../types/avatarCreateTypes";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function joinPaths(basePath: string, path: string): string {
  if (!basePath || basePath === "/") {
    return path.startsWith("/") ? path : `/${path}`;
  }

  if (!path || path === "/") {
    return basePath;
  }

  if (basePath.endsWith("/") && path.startsWith("/")) {
    return basePath + path.slice(1);
  }

  if (!basePath.endsWith("/") && !path.startsWith("/")) {
    return `${basePath}/${path}`;
  }

  return basePath + path;
}

function buildApiUrl(baseUrl: string, path: string): string {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);

  if (!normalizedBaseUrl) {
    throw new Error("API base が未設定です。");
  }

  const url = new URL(normalizedBaseUrl);
  url.pathname = joinPaths(url.pathname, path);
  url.search = "";
  url.hash = "";

  return url.toString();
}

async function readApiError(response: Response): Promise<string> {
  const contentType = response.headers.get("content-type") || "";

  if (contentType.includes("application/json")) {
    const body = (await response.json().catch(() => null)) as
      | { error?: string; message?: string }
      | null;

    if (body?.error) return body.error;
    if (body?.message) return body.message;
  }

  const text = await response.text().catch(() => "");
  return text || `API request failed (${response.status})`;
}

type AuthedRequestParams = {
  backendUrl: string;
  idToken: string;
};

function unwrapData<T>(body: unknown): T {
  if (
    body &&
    typeof body === "object" &&
    "data" in body &&
    (body as { data?: unknown }).data
  ) {
    return (body as { data: T }).data;
  }

  return body as T;
}

export async function getMyAvatar({
  backendUrl,
  idToken,
}: AuthedRequestParams): Promise<MyAvatarResponse | null> {
  const response = await fetch(buildApiUrl(backendUrl, "/mall/me/avatars"), {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    throw new Error(await readApiError(response));
  }

  const body = await response.json();
  return unwrapData<MyAvatarResponse>(body);
}

export async function createAvatar({
  backendUrl,
  idToken,
  payload,
}: AuthedRequestParams & {
  payload: CreateAvatarPayload;
}): Promise<CreateAvatarResponse> {
  const response = await fetch(buildApiUrl(backendUrl, "/mall/avatars"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await readApiError(response));
  }

  const body = await response.json();
  return unwrapData<CreateAvatarResponse>(body);
}

export async function updateAvatar({
  backendUrl,
  idToken,
  avatarId,
  payload,
}: AuthedRequestParams & {
  avatarId: string;
  payload: UpdateAvatarPayload;
}): Promise<UpdateAvatarResponse> {
  void avatarId;

  const response = await fetch(buildApiUrl(backendUrl, "/mall/me/avatars"), {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await readApiError(response));
  }

  const body = await response.json().catch(() => ({ avatarId }));
  return unwrapData<UpdateAvatarResponse>(body);
}