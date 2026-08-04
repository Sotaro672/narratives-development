// frontend/amol/src/features/resale/api/resaleHttpClient.ts

import {
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";

import {
  getFirebaseIdToken,
} from "../../../lib/authToken";

type ApiErrorResponse = {
  error?: string;
};

function buildApiUrl(
  path: string,
): string {
  const baseUrl = getApiBaseUrl();

  return baseUrl
    ? `${baseUrl}${path}`
    : path;
}

async function readApiJson<T>(
  response: Response,
): Promise<T> {
  return (
    await response
      .json()
      .catch(() => ({}))
  ) as T;
}

export async function fetchResaleWithAuth<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const token =
    await getFirebaseIdToken();

  const headers =
    new Headers(init?.headers);

  headers.set(
    "Authorization",
    `Bearer ${token}`,
  );

  if (
    init?.body &&
    !headers.has("Content-Type")
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  const response = await fetch(
    buildApiUrl(path),
    {
      ...init,
      headers,
    },
  );

  const json =
    await readApiJson<
      T & ApiErrorResponse
    >(response);

  if (!response.ok) {
    throw new Error(
      json.error ||
        "APIリクエストに失敗しました。",
    );
  }

  return json;
}

export async function fetchPublicResale<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(
    buildApiUrl(path),
    init,
  );

  const json =
    await readApiJson<
      T & ApiErrorResponse
    >(response);

  if (!response.ok) {
    throw new Error(
      json.error ||
        "APIリクエストに失敗しました。",
    );
  }

  return json;
}

export type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};