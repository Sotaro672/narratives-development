// frontend/amol/src/features/scan-result/infrastructure/scanResultHttp.ts

import { getIdToken } from "firebase/auth";

import { auth } from "../../../lib/firebase";
import { isRecord } from "../../shared/utils/typeGuards";

export function jsonHeaders(): HeadersInit {
  return {
    Accept: "application/json",
  };
}

export function jsonPostHeaders(): HeadersInit {
  return {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
}

export function mergeHeaders(
  base: HeadersInit,
  extra?: HeadersInit,
): Headers {
  const headers = new Headers(base);

  if (extra) {
    new Headers(extra).forEach(
      (value, key) => {
        headers.set(key, value);
      },
    );
  }

  return headers;
}

export function getAuthorizationHeader(
  headers?: HeadersInit,
): string {
  const normalizedHeaders =
    new Headers(headers);

  return (
    normalizedHeaders.get(
      "Authorization",
    ) ||
    normalizedHeaders.get(
      "authorization",
    ) ||
    ""
  ).trim();
}

export async function readJsonObject(
  response: Response,
  label: string,
  url: string,
): Promise<Record<string, unknown>> {
  const text = await response.text();

  if (!response.ok) {
    const body =
      text.length > 300
        ? text.slice(0, 300)
        : text;

    throw new Error(
      `${label} failed: ${response.status} url=${url} body=${body}`,
    );
  }

  let decoded: unknown;

  try {
    decoded = text
      ? JSON.parse(text)
      : {};
  } catch {
    throw new Error(
      `${label} failed: invalid json url=${url}`,
    );
  }

  if (
    !isRecord(decoded) ||
    Array.isArray(decoded)
  ) {
    throw new Error(
      "invalid json shape (expected object)",
    );
  }

  return decoded;
}

export async function getAuthHeadersOrUndefined(): Promise<
  Record<string, string> | undefined
> {
  const user = auth.currentUser;

  if (!user) {
    return undefined;
  }

  try {
    const token =
      await getIdToken(user);

    const normalizedToken = String(
      token || "",
    ).trim();

    return normalizedToken
      ? {
          Authorization:
            `Bearer ${normalizedToken}`,
        }
      : undefined;
  } catch {
    return undefined;
  }
}