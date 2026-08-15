// frontend/amol/src/features/inquiry/api/inquiryHttpClient.ts

import {
  requestJson,
  type ApiQueryParams,
} from "../../../lib/http";

export const INQUIRY_BASE_PATH = "/mall/me/inquiries";

export type ApiDataResponse<T> = {
  data: T;
};

export type ApiItemsResponse<T> = {
  items: T[];
};

export type ApiPagedItemsResponse<T> = {
  items: T[];
  page: number;
  perPage: number;
};

export type ApiUnreadCountResponse = {
  unreadCount: number;
};

export type InquiryRequestInit = Omit<RequestInit, "body"> & {
  json?: unknown;
  query?: ApiQueryParams;
};

export function buildInquiryPath(inquiryId: string): string {
  return `${INQUIRY_BASE_PATH}/${encodeURIComponent(inquiryId)}`;
}

/**
 * 認証が必要な問い合わせAPIを実行します。
 *
 * URL生成、Firebase認証、ヘッダー設定、
 * JSON解析、HTTPエラー処理は共通HTTPクライアントへ委譲します。
 *
 * request body は JSON.stringify せず、json としてそのまま渡します。
 */
export async function fetchInquiryWithAuth<T>(
  path: string,
  init?: InquiryRequestInit,
): Promise<T> {
  const { json, query, ...requestInit } = init ?? {};

  return requestJson<T>(path, {
    ...requestInit,
    auth: "required",
    query,
    ...(json !== undefined ? { json } : {}),
    messages: {
      requestErrorMessage: "APIリクエストに失敗しました。",
    },
  });
}