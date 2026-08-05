// frontend/amol/src/features/resale/api/resaleHttpClient.ts

import {
  requestJson,
  type ApiQueryParams,
} from "../../../lib/http";

export type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};

type ResaleRequestInit =
  RequestInit & {
    query?: ApiQueryParams;
  };

/**
 * 既存のRequestInit.bodyを、
 * 共通HTTPクライアントのjsonオプションへ変換します。
 *
 * resale APIではJSON本文のみを扱います。
 */
function parseJsonRequestBody(
  body: BodyInit | null | undefined,
): unknown {
  if (
    body === undefined ||
    body === null
  ) {
    return undefined;
  }

  if (typeof body !== "string") {
    throw new Error(
      "再販APIのリクエスト本文はJSON文字列で指定してください。",
    );
  }

  const normalizedBody =
    body.trim();

  if (!normalizedBody) {
    return undefined;
  }

  try {
    return JSON.parse(
      normalizedBody,
    ) as unknown;
  } catch {
    throw new Error(
      "再販APIのリクエスト本文が不正なJSONです。",
    );
  }
}

/**
 * 認証が必要な再販APIを実行します。
 *
 * URL生成、Firebase認証、ヘッダー設定、
 * JSON解析、HTTPエラー処理は共通HTTPクライアントへ委譲します。
 */
export async function fetchResaleWithAuth<T>(
  path: string,
  init?: ResaleRequestInit,
): Promise<T> {
  const {
    body,
    query,
    ...requestInit
  } = init ?? {};

  const json =
    parseJsonRequestBody(body);

  return requestJson<T>(
    path,
    {
      ...requestInit,

      auth:
        "required",

      ...(query !== undefined
        ? {
            query,
          }
        : {}),

      ...(json !== undefined
        ? {
            json,
          }
        : {}),

      messages: {
        requestErrorMessage:
          "APIリクエストに失敗しました。",
      },
    },
  );
}

/**
 * 認証不要の公開再販APIを実行します。
 *
 * URL生成、ヘッダー設定、JSON解析、
 * HTTPエラー処理は共通HTTPクライアントへ委譲します。
 */
export async function fetchPublicResale<T>(
  path: string,
  init?: ResaleRequestInit,
): Promise<T> {
  const {
    body,
    query,
    ...requestInit
  } = init ?? {};

  const json =
    parseJsonRequestBody(body);

  return requestJson<T>(
    path,
    {
      ...requestInit,

      auth:
        "none",

      ...(query !== undefined
        ? {
            query,
          }
        : {}),

      ...(json !== undefined
        ? {
            json,
          }
        : {}),

      messages: {
        requestErrorMessage:
          "APIリクエストに失敗しました。",
      },
    },
  );
}