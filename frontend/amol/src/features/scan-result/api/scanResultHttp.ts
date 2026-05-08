// frontend/amol/src/features/scan-result/api/scanResultHttp.ts
import { getIdToken } from "firebase/auth";

import { auth } from "../../../lib/firebase";
import { isRecord } from "../utils/format";

export function resolveApiBase(): string {
  return normalizeBaseUrl(String(import.meta.env.VITE_API_BASE_URL || ""));
}

export function normalizeBaseUrl(value: string): string {
  let v = value.trim();
  while (v.endsWith("/")) {
    v = v.slice(0, -1);
  }
  return v;
}

export function jsonHeaders(): HeadersInit {
  return { Accept: "application/json" };
}

export function jsonPostHeaders(): HeadersInit {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
}

export function mergeHeaders(base: HeadersInit, extra?: HeadersInit): Headers {
  const headers = new Headers(base);

  if (extra) {
    new Headers(extra).forEach((value, key) => headers.set(key, value));
  }

  return headers;
}

export function getAuthorizationHeader(headers?: HeadersInit): string {
  const h = new Headers(headers);
  return (h.get("Authorization") || h.get("authorization") || "").trim();
}

export async function readJsonObject(
  response: Response,
  label: string,
  url: string
): Promise<Record<string, unknown>> {
  const text = await response.text();

  if (!response.ok) {
    const body = text.length > 300 ? text.slice(0, 300) : text;
    throw new Error(`${label} failed: ${response.status} url=${url} body=${body}`);
  }

  let decoded: unknown;

  try {
    decoded = text ? JSON.parse(text) : {};
  } catch {
    throw new Error(`${label} failed: invalid json url=${url}`);
  }

  if (!isRecord(decoded)) {
    throw new Error("invalid json shape (expected object)");
  }

  return decoded;
}

export async function getAuthHeadersOrUndefined(): Promise<
  Record<string, string> | undefined
> {
  const user = auth.currentUser;
  if (!user) return undefined;

  try {
    const token = await getIdToken(user);
    const t = String(token || "").trim();
    return t ? { Authorization: `Bearer ${t}` } : undefined;
  } catch {
    return undefined;
  }
}