// frontend/console/tokenBlueprint/src/infrastructure/http/client.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

export type HttpMethod = "GET" | "POST" | "PUT" | "DELETE";

function getActorIdOrEmpty(): string {
  try {
    return auth.currentUser?.uid?.trim?.() ?? "";
  } catch {
    return "";
  }
}

function withActorHeader(headers: Record<string, string>): Record<string, string> {
  const actorId = getActorIdOrEmpty();
  if (actorId) headers["X-Actor-Id"] = actorId;
  return headers;
}

export function apiUrl(path: string): string {
  const p = String(path || "").trim();
  if (!p.startsWith("/")) return `${API_BASE}/${p}`;
  return `${API_BASE}${p}`;
}

export async function apiGet(path: string, extraHeaders?: Record<string, string>): Promise<Response> {
  const base = await getAuthHeadersOrThrow();
  const headers = withActorHeader({ ...(base as any), ...(extraHeaders ?? {}) });

  return fetch(apiUrl(path), {
    method: "GET",
    headers,
  });
}

export async function apiDelete(path: string, extraHeaders?: Record<string, string>): Promise<Response> {
  const base = await getAuthHeadersOrThrow();
  const headers = withActorHeader({ ...(base as any), ...(extraHeaders ?? {}) });

  return fetch(apiUrl(path), {
    method: "DELETE",
    headers,
  });
}

export async function apiPostJson(
  path: string,
  body: unknown,
  extraHeaders?: Record<string, string>,
): Promise<Response> {
  const base = await getAuthJsonHeadersOrThrow();
  const headers = withActorHeader({ ...(base as any), ...(extraHeaders ?? {}) });

  return fetch(apiUrl(path), {
    method: "POST",
    headers,
    body: JSON.stringify(body ?? {}),
  });
}

export async function apiPutJson(
  path: string,
  body: unknown,
  extraHeaders?: Record<string, string>,
): Promise<Response> {
  const base = await getAuthJsonHeadersOrThrow();
  const headers = withActorHeader({ ...(base as any), ...(extraHeaders ?? {}) });

  return fetch(apiUrl(path), {
    method: "PUT",
    headers,
    body: JSON.stringify(body ?? {}),
  });
}
