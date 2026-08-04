// frontend/amol/src/features/inquiry/api/inquiryHttpClient.ts

import {
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";

import {
  getFirebaseIdToken,
} from "../../../lib/authToken";

export const INQUIRY_BASE_PATH =
  "/mall/me/inquiries";

export type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};

export type ApiItemsResponse<T> = {
  items?: T[];
  page?: number;
  perPage?: number;
  total?: number;
  totalCount?: number;
  error?: string;
};

export type ApiUnreadCountResponse = {
  count?: number;
  unreadCount?: number;
  error?: string;
};

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

export function buildInquiryPath(
  inquiryId: string,
): string {
  return `${INQUIRY_BASE_PATH}/${encodeURIComponent(
    inquiryId,
  )}`;
}

export async function fetchInquiryWithAuth<T>(
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