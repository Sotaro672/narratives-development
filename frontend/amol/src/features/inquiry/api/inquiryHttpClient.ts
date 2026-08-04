// frontend/amol/src/features/inquiry/api/inquiryHttpClient.ts

import {
  requestJson,
  type ApiQueryParams,
} from "../../../lib/http";

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

export type InquiryRequestInit =
  Omit<
    RequestInit,
    "body"
  > & {
    body?: BodyInit | null;
    query?: ApiQueryParams;
  };

/**
 * 既存のRequestInit.bodyを、
 * 共通HTTPクライアントのjsonオプションへ変換します。
 *
 * inquiry APIではJSON本文のみを扱います。
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
      "問い合わせAPIのリクエスト本文はJSON文字列で指定してください。",
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
      "問い合わせAPIのリクエスト本文が不正なJSONです。",
    );
  }
}

export function buildInquiryPath(
  inquiryId: string,
): string {
  return `${INQUIRY_BASE_PATH}/${encodeURIComponent(
    inquiryId,
  )}`;
}

/**
 * 認証が必要な問い合わせAPIを実行します。
 *
 * URL生成、Firebase認証、ヘッダー設定、
 * JSON解析、HTTPエラー処理は共通HTTPクライアントへ委譲します。
 */
export async function fetchInquiryWithAuth<T>(
  path: string,
  init?: InquiryRequestInit,
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

      auth: "required",

      query,

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