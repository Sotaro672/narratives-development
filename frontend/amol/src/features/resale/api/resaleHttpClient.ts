// frontend/amol/src/features/resale/api/resaleHttpClient.ts

import {
  requestJson,
  type ApiQueryParams,
} from "../../../lib/http";

export type ApiDataResponse<T> = {
  data: T;
};

type ResaleRequestInit = RequestInit & {
  query?: ApiQueryParams;
};

/**
 * RequestInit.body の JSON 文字列を、
 * 共通HTTPクライアントの json オプションへ変換する。
 */
function parseJsonRequestBody(
  body: BodyInit | null | undefined,
): unknown {
  if (body === undefined || body === null) {
    return undefined;
  }

  if (typeof body !== "string") {
    throw new Error(
      "再販APIのリクエスト本文はJSON文字列で指定してください。",
    );
  }

  if (!body) {
    return undefined;
  }

  try {
    return JSON.parse(body) as unknown;
  } catch {
    throw new Error(
      "再販APIのリクエスト本文が不正なJSONです。",
    );
  }
}

/**
 * 認証が必要な再販APIを実行する。
 */
export async function fetchResaleWithAuth<T>(
  path: string,
  init?: ResaleRequestInit,
): Promise<T> {
  const { body, query, ...requestInit } = init ?? {};
  const json = parseJsonRequestBody(body);

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
  const { body, query, ...requestInit } = init ?? {};
  const json = parseJsonRequestBody(body);

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