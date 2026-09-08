// frontend/console/shell/src/features/notification/infrastructure/newsApi.ts

import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";
import type { PageParams } from "../../../shared/types/common/common";
import type {
  NewsPage,
  NewsReadResponse,
  NewsUnreadCountResponse,
} from "../../../shared/types/news";

export type ListNewsParams = PageParams;

type ErrorResponse = {
  error?: string;
  message?: string;
  detail?: string;
};

async function readJsonResponse<T>(
  response: Response,
  url: string,
): Promise<T> {
  const contentType = response.headers.get("content-type") ?? "";
  const text = await response.text().catch(() => "");

  if (!response.ok) {
    let message = "";

    if (contentType.includes("application/json") && text) {
      try {
        const errorResponse = JSON.parse(text) as ErrorResponse;
        message =
          errorResponse.error ??
          errorResponse.message ??
          errorResponse.detail ??
          "";
      } catch {
        message = text;
      }
    } else {
      message = text;
    }

    throw new Error(
      message ||
        `システム通知APIの呼び出しに失敗しました（${response.status} ${response.statusText}）`,
    );
  }

  if (!text) {
    throw new Error(
      `システム通知APIのレスポンスが空です。URL=${url}`,
    );
  }

  if (!contentType.includes("application/json")) {
    throw new Error(
      `システム通知APIからJSON以外のレスポンスが返されました。URL=${url}`,
    );
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(
      `システム通知APIのJSONレスポンスを解析できませんでした。URL=${url}`,
    );
  }
}

async function requestJson<T>(
  url: string,
  init: RequestInit,
): Promise<T> {
  const authHeaders = await getAuthHeaders();

  const response = await fetch(url, {
    ...init,
    headers: {
      ...authHeaders,
      Accept: "application/json",
      ...init.headers,
    },
    credentials: "include",
  });

  return readJsonResponse<T>(response, url);
}

function buildListQuery(params?: ListNewsParams): string {
  const searchParams = new URLSearchParams();

  if (params?.page !== undefined) {
    searchParams.set("page", String(params.page));
  }

  if (params?.perPage !== undefined) {
    searchParams.set("perPage", String(params.perPage));
  }

  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

/**
 * GET /news
 *
 * ログイン中のConsoleメンバー向けシステム通知を取得する。
 * 既読状態はBackendで認証済みmemberIdを基に解決する。
 */
export async function listNewsApi(
  params?: ListNewsParams,
): Promise<NewsPage> {
  const query = buildListQuery(params);
  const url = `${API_BASE}/news${query}`;

  return requestJson<NewsPage>(url, {
    method: "GET",
  });
}

/**
 * GET /news/unread-count
 *
 * ログイン中のConsoleメンバーの未読システム通知件数を取得する。
 */
export async function getNewsUnreadCountApi(): Promise<NewsUnreadCountResponse> {
  const url = `${API_BASE}/news/unread-count`;

  return requestJson<NewsUnreadCountResponse>(url, {
    method: "GET",
  });
}

/**
 * POST /news/{newsId}/read
 *
 * 指定したシステム通知をログイン中のConsoleメンバーについて既読にする。
 */
export async function markNewsReadApi(
  newsId: string,
): Promise<NewsReadResponse> {
  const normalizedNewsId = newsId.trim();

  if (!normalizedNewsId) {
    throw new Error("newsId is required");
  }

  const encodedNewsId = encodeURIComponent(normalizedNewsId);
  const url = `${API_BASE}/news/${encodedNewsId}/read`;

  return requestJson<NewsReadResponse>(url, {
    method: "POST",
  });
}

export const newsApi = {
  list: listNewsApi,
  unreadCount: getNewsUnreadCountApi,
  markRead: markNewsReadApi,
};