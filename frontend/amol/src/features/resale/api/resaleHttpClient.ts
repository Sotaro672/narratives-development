// frontend/amol/src/features/resale/api/resaleHttpClient.ts

import {
  requestJson,
  type ApiQueryParams,
} from "../../../lib/http";

export type ApiDataResponse<T> = {
  data: T;
};

type ResaleRequestInit = Omit<RequestInit, "body"> & {
  query?: ApiQueryParams;
  json?: unknown;
};

/**
 * 認証が必要な再販APIを実行する。
 */
export async function fetchResaleWithAuth<T>(
  path: string,
  init?: ResaleRequestInit,
): Promise<T> {
  const { query, json, ...requestInit } = init ?? {};

  return requestJson<T>(path, {
    ...requestInit,
    auth: "required",
    ...(query !== undefined ? { query } : {}),
    ...(json !== undefined ? { json } : {}),
    messages: {
      requestErrorMessage: "APIリクエストに失敗しました。",
    },
  });
}

/**
 * 認証不要の公開再販APIを実行する。
 */
export async function fetchPublicResale<T>(
  path: string,
  init?: ResaleRequestInit,
): Promise<T> {
  const { query, json, ...requestInit } = init ?? {};

  return requestJson<T>(path, {
    ...requestInit,
    auth: "none",
    ...(query !== undefined ? { query } : {}),
    ...(json !== undefined ? { json } : {}),
    messages: {
      requestErrorMessage: "APIリクエストに失敗しました。",
    },
  });
}